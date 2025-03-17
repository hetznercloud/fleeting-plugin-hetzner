package instancegroup

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/actionutil"
)

// ServerHandler creates a server from the instance server create options.
type ServerHandler struct{}

var _ CreateHandler = (*ServerHandler)(nil)
var _ CleanupHandler = (*ServerHandler)(nil)

func (h *ServerHandler) Create(ctx context.Context, group *instanceGroup, instance *Instance) error {
	instance.opts.Name = instance.Name
	instance.opts.Labels = group.labels
	instance.opts.Location = group.location
	instance.opts.Image = group.image
	instance.opts.SSHKeys = group.sshKeys
	instance.opts.UserData = group.config.UserData
	instance.opts.PublicNet.EnableIPv4 = !group.config.PublicIPv4Disabled
	instance.opts.PublicNet.EnableIPv6 = !group.config.PublicIPv6Disabled
	instance.opts.Networks = group.privateNetworks

	var result hcloud.ServerCreateResult
	var err error

	for _, serverType := range group.serverTypes {
		instance.opts.ServerType = serverType

		result, _, err = group.client.Server.Create(ctx, *instance.opts)
		if err != nil && hcloud.IsError(err, hcloud.ErrorCodeResourceUnavailable) {
			group.log.Warn("resource unavailable", "server_type", serverType.Name, "err", err)
			continue
		}
		break
	}
	if err != nil {
		return fmt.Errorf("could not request instance creation: %w", err)
	}

	*instance = *InstanceFromServer(result.Server)

	instance.waitFn = func() error {
		if err := group.client.Action.WaitFor(ctx, actionutil.AppendNext(result.Action, result.NextActions)...); err != nil {
			return fmt.Errorf("could not create instance: %w", err)
		}

		return nil
	}

	return nil
}

func (h *ServerHandler) Cleanup(ctx context.Context, group *instanceGroup, instance *Instance) error {
	if instance.ID == 0 {
		return nil
	}

	result, _, err := group.client.Server.DeleteWithResult(ctx, &hcloud.Server{ID: instance.ID})
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			group.log.Warn("tried to delete a server that do not exist", "name", instance.Name, "id", instance.ID)
			return nil
		}
		return fmt.Errorf("could not request instance deletion: %w", err)
	}

	instance.waitFn = func() error {
		if err := group.client.Action.WaitFor(ctx, result.Action); err != nil {
			return fmt.Errorf("could not delete instance: %w", err)
		}
		return nil
	}

	return nil
}
