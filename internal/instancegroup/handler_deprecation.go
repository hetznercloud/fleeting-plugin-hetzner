package instancegroup

import (
	"context"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/deprecationutil"
)

// DeprecationHandler checks for deprecations in resources used by the instance group.
type DeprecationHandler struct{}

var _ SanityHandler = (*DeprecationHandler)(nil)

func (h *DeprecationHandler) Sanity(_ context.Context, group *instanceGroup) error {
	// Check server type deprecation
	for _, serverType := range group.serverTypes {
		if message, isUnavailable := deprecationutil.ServerTypeMessage(serverType, group.location.Name); message != "" {
			message = strings.ReplaceAll(strings.ToLower(message), "\"", "")
			if isUnavailable {
				group.log.Error(message, "server_type", serverType.Name)
			} else {
				group.log.Warn(message, "server_type", serverType.Name)
			}
		}
	}

	// Check image deprecation
	if message, isUnavailable := deprecationutil.ImageMessage(group.image); message != "" {
		message = strings.ReplaceAll(strings.ToLower(message), "\"", "")
		if isUnavailable {
			group.log.Error(message, "image", group.image.Name)
		} else {
			group.log.Warn(message, "image", group.image.Name)
		}
	}

	return nil
}
