package instancegroup

import (
	"context"
	"fmt"
)

// IPPoolHandler updates the instance server create options with IPs from a pool of existing IPs.
type IPPoolHandler struct{}

var _ PreIncreaseHandler = (*IPPoolHandler)(nil)
var _ CreateHandler = (*IPPoolHandler)(nil)

func (h *IPPoolHandler) PreIncrease(ctx context.Context, group *instanceGroup) error {
	if !group.config.PublicIPPoolEnabled {
		return nil
	}

	err := group.ipPool.Refresh(ctx, group.client)
	if err != nil {
		return err
	}

	return nil
}

func (h *IPPoolHandler) Create(_ context.Context, group *instanceGroup, instance *Instance) error {
	if !group.config.PublicIPPoolEnabled {
		return nil
	}

	if !group.config.PublicIPv4Disabled {
		ipv4, err := group.ipPool.NextIPv4()
		if err != nil {
			return fmt.Errorf("could not get ipv4 from pool: %w", err)
		}

		instance.opts.PublicNet.IPv4 = ipv4
	}

	if !group.config.PublicIPv6Disabled {
		ipv6, err := group.ipPool.NextIPv6()
		if err != nil {
			return fmt.Errorf("could not get ipv6 from pool: %w", err)
		}

		instance.opts.PublicNet.IPv6 = ipv6
	}

	return nil
}
