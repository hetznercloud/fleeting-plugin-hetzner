package instancegroup

import (
	"context"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// BaseHandler configure the instance server create options with the instance group configuration.
type BaseHandler struct{}

var _ CreateHandler = (*BaseHandler)(nil)

func (h *BaseHandler) Create(_ context.Context, _ *instanceGroup, instance *Instance) error {
	instance.opts = &hcloud.ServerCreateOpts{}
	instance.opts.PublicNet = &hcloud.ServerCreatePublicNet{}

	return nil
}
