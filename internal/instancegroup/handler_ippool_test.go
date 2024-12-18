package instancegroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

func TestIPPoolHandlerCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig
		config.PublicIPv4Disabled = false
		config.PublicIPPoolEnabled = true
		config.PublicIPPoolSelector = "fleeting"

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "GET", Path: "/primary_ips?label_selector=fleeting&page=1",
				Status: 200,
				JSON: schema.PrimaryIPListResult{
					PrimaryIPs: []schema.PrimaryIP{
						{
							ID:           1,
							Name:         "fleeting-a-ipv6",
							IP:           "2a01:4f9:c010:cfde::/64",
							Type:         "ipv6",
							AssigneeID:   nil,
							AssigneeType: "server",
							Datacenter:   schema.Datacenter{ID: 3, Name: "hel1-dc2", Location: schema.Location{ID: 3, Name: "hel1"}},
						},
						{
							ID:           2,
							Name:         "fleeting-a-ipv4",
							IP:           "201.55.32.12",
							Type:         "ipv4",
							AssigneeID:   nil,
							AssigneeType: "server",
							Datacenter:   schema.Datacenter{ID: 3, Name: "hel1-dc2", Location: schema.Location{ID: 3, Name: "hel1"}},
						},
						{
							ID:           3,
							Name:         "fleeting-b-ipv6",
							IP:           "2a01:4f9:c010:cfdf::/64",
							Type:         "ipv6",
							AssigneeID:   nil,
							AssigneeType: "server",
							Datacenter:   schema.Datacenter{ID: 3, Name: "hel1-dc2", Location: schema.Location{ID: 3, Name: "hel1"}},
						},

						{
							ID:           4,
							Name:         "fleeting-b-ipv4",
							IP:           "201.23.56.76",
							Type:         "ipv4",
							AssigneeID:   nil,
							AssigneeType: "server",
							Datacenter:   schema.Datacenter{ID: 3, Name: "hel1-dc2", Location: schema.Location{ID: 3, Name: "hel1"}},
						},
					},
				},
			},
		})

		instance := NewInstance("fleeting-a")
		{
			handler := &BaseHandler{}
			require.NoError(t, handler.Create(ctx, group, instance))
		}

		handler := &IPPoolHandler{}

		require.NoError(t, handler.PreIncrease(ctx, group))
		require.NoError(t, handler.Create(ctx, group, instance))

		assert.NotNil(t, instance.opts.PublicNet.IPv6)
		assert.NotNil(t, instance.opts.PublicNet.IPv4)
		assert.Equal(t, int64(1), instance.opts.PublicNet.IPv6.ID)
		assert.Equal(t, int64(2), instance.opts.PublicNet.IPv4.ID)
	})

	t.Run("disabled", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{})

		instance := NewInstance("fleeting-a")
		{
			handler := &BaseHandler{}
			require.NoError(t, handler.Create(ctx, group, instance))
		}

		handler := &IPPoolHandler{}
		require.NoError(t, handler.PreIncrease(ctx, group))
		require.NoError(t, handler.Create(ctx, group, instance))
	})
}
