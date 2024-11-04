package instancegroup

import (
	"context"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// BaseHandler configure the instance server create options with the instance group configuration.
type BaseHandler struct{}

var _ CreateHandler = (*BaseHandler)(nil)

func (h *BaseHandler) Create(_ context.Context, group *instanceGroup, instance *Instance) error {
	opts := hcloud.ServerCreateOpts{}
	opts.Name = instance.Name
	opts.Labels = group.labels
	opts.Location = group.location
	opts.ServerType = group.serverType
	opts.Image = group.image
	opts.PublicNet = &hcloud.ServerCreatePublicNet{
		EnableIPv4: !group.config.PublicIPv4Disabled,
		EnableIPv6: !group.config.PublicIPv6Disabled,
	}
	opts.Networks = group.privateNetworks
	opts.SSHKeys = group.sshKeys
	opts.UserData = group.config.UserData

	instance.opts = &opts

	return nil
}
