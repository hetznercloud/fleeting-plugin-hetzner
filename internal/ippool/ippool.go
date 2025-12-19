package ippool

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// IPPool defines a pool of both IPv4 and IPv6 Primary IPs, populated with unused
// Primary IPs from the Hetzner Cloud "Project". The Primary IPs can be filtered using a
// label selector (https://docs.hetzner.cloud/reference/cloud#label-selector).
type IPPool struct {
	location      string
	labelSelector string

	mu sync.Mutex

	ipv4 []*hcloud.PrimaryIP
	ipv6 []*hcloud.PrimaryIP
}

var (
	// ErrNotInitialized is returned when the IP pool has not been initialized.
	// Calling [IPPool.Refresh] will initialize the pool.
	ErrNotInitialized = fmt.Errorf("ip pool is not initialized")
	// ErrEmpty is returned when the queried IP pool is empty.
	ErrEmpty = fmt.Errorf("ip pool is empty")
)

// New creates a new IPPool.
func New(location string, labelSelector string) *IPPool {
	return &IPPool{
		location:      location,
		labelSelector: labelSelector,
	}
}

// Refresh initialize or refresh the pool of Primary IPs. This function must be called
// before starting to consume the pool.
func (o *IPPool) Refresh(ctx context.Context, client *hcloud.Client) error {
	ips, err := client.PrimaryIP.AllWithOpts(ctx, hcloud.PrimaryIPListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: o.labelSelector,
		},
	})
	if err != nil {
		return fmt.Errorf("could not refresh ip pool: %w", err)
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.ipv4 = make([]*hcloud.PrimaryIP, 0, len(ips))
	o.ipv6 = make([]*hcloud.PrimaryIP, 0, len(ips))

	for _, ip := range ips {
		if ip.Location.Name != o.location {
			continue
		}
		if ip.AssigneeID != 0 {
			continue
		}
		switch ip.Type {
		case hcloud.PrimaryIPTypeIPv4:
			o.ipv4 = append(o.ipv4, ip)
		case hcloud.PrimaryIPTypeIPv6:
			o.ipv6 = append(o.ipv6, ip)
		}
	}

	o.ipv4 = slices.Clip(o.ipv4)
	o.ipv6 = slices.Clip(o.ipv6)

	return nil
}

// SizeIPv4 returns the size of the IPv4 pool.
func (o *IPPool) SizeIPv4() int { return len(o.ipv4) }

// NextIPv4 returns and remove the first IPv4 from the IPv4 pool.
func (o *IPPool) NextIPv4() (*hcloud.PrimaryIP, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.ipv4 == nil {
		return nil, ErrNotInitialized
	}

	if len(o.ipv4) == 0 {
		return nil, ErrEmpty
	}

	var ip *hcloud.PrimaryIP
	ip, o.ipv4 = o.ipv4[0], o.ipv4[1:]

	return ip, nil
}

// SizeIPv6 returns the size of the IPv6 pool.
func (o *IPPool) SizeIPv6() int { return len(o.ipv6) }

// NextIPv6 returns and remove the first IPv6 from the IPv6 pool.
func (o *IPPool) NextIPv6() (*hcloud.PrimaryIP, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.ipv6 == nil {
		return nil, ErrNotInitialized
	}

	if len(o.ipv6) == 0 {
		return nil, ErrEmpty
	}

	var ip *hcloud.PrimaryIP
	ip, o.ipv6 = o.ipv6[0], o.ipv6[1:]

	return ip, nil
}
