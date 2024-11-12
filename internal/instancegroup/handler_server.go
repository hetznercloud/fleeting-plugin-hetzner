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
	result, _, err := group.client.Server.Create(ctx, *instance.opts)
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
			group.log.Warn("tried to delete a server that do not exist: %s", instance.Name)
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
