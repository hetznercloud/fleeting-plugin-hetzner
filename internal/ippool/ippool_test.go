package ippool

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/testutils"
)

func TestNextIP(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		ipPool := New("hel1", "instance-group=fleeting")
		ipv4, err := ipPool.NextIPv4()
		require.Equal(t, ErrNotInitialized, err)
		require.Nil(t, ipv4)

		ipv6, err := ipPool.NextIPv6()
		require.Equal(t, ErrNotInitialized, err)
		require.Nil(t, ipv6)
	})

	t.Run("empty", func(t *testing.T) {
		ipPool := New("hel1", "instance-group=fleeting")

		testServer := httptest.NewServer(mockutil.Handler(t, []mockutil.Request{
			{
				Method: "GET", Path: "/primary_ips?label_selector=instance-group%3Dfleeting&page=1",
				Status: 200,
				JSON:   schema.PrimaryIPListResponse{},
			},
		}))
		testClient := testutils.MakeTestClient(testServer.URL)

		ipPool.Refresh(context.Background(), testClient)

		ipv4, err := ipPool.NextIPv4()
		require.Equal(t, ErrEmpty, err)
		require.Nil(t, ipv4)

		ipv6, err := ipPool.NextIPv6()
		require.Equal(t, ErrEmpty, err)
		require.Nil(t, ipv6)
	})

	t.Run("happy", func(t *testing.T) {
		ipPool := New("hel1", "instance-group=fleeting")

		datacenterHel1 := schema.Datacenter{Location: schema.Location{Name: "hel1"}}
		datacenterFsn1 := schema.Datacenter{Location: schema.Location{Name: "fsn1"}}

		testServer := httptest.NewServer(mockutil.Handler(t, []mockutil.Request{
			{
				Method: "GET", Path: "/primary_ips?label_selector=instance-group%3Dfleeting&page=1",
				Status: 200,
				JSON: schema.PrimaryIPListResponse{
					PrimaryIPs: []schema.PrimaryIP{
						{ID: 41, IP: "1.1.1.1", Type: "ipv4", AssigneeID: hcloud.Ptr(int64(0)), Datacenter: datacenterHel1},
						{ID: 42, IP: "2.2.2.2", Type: "ipv4", AssigneeID: hcloud.Ptr(int64(1)), Datacenter: datacenterHel1},
						{ID: 43, IP: "3.3.3.3", Type: "ipv4", AssigneeID: hcloud.Ptr(int64(2)), Datacenter: datacenterFsn1},
						{ID: 44, IP: "4.4.4.4", Type: "ipv4", AssigneeID: hcloud.Ptr(int64(0)), Datacenter: datacenterFsn1},
						{ID: 61, IP: "2001:db8:c012:d011::/64", Type: "ipv6", AssigneeID: hcloud.Ptr(int64(0)), Datacenter: datacenterHel1},
						{ID: 62, IP: "2001:db8:c012:d022::/64", Type: "ipv6", AssigneeID: hcloud.Ptr(int64(3)), Datacenter: datacenterHel1},
						{ID: 63, IP: "2001:db8:c012:d033::/64", Type: "ipv6", AssigneeID: hcloud.Ptr(int64(4)), Datacenter: datacenterFsn1},
						{ID: 64, IP: "2001:db8:c012:d044::/64", Type: "ipv6", AssigneeID: hcloud.Ptr(int64(0)), Datacenter: datacenterFsn1},
					},
				},
			},
		}))
		testClient := testutils.MakeTestClient(testServer.URL)

		err := ipPool.Refresh(context.Background(), testClient)
		require.NoError(t, err)

		require.Equal(t, 1, ipPool.SizeIPv4())

		ipv4, err := ipPool.NextIPv4()
		require.NoError(t, err)
		require.NotNil(t, ipv4)
		require.Equal(t, int64(41), ipv4.ID)
		require.Equal(t, "1.1.1.1", ipv4.IP.String())

		require.Equal(t, 0, ipPool.SizeIPv4())

		ipv4, err = ipPool.NextIPv4()
		require.Equal(t, ErrEmpty, err)
		require.Nil(t, ipv4)

		require.Equal(t, 1, ipPool.SizeIPv6())
		ipv6, err := ipPool.NextIPv6()
		require.NoError(t, err)
		require.NotNil(t, ipv6)
		require.Equal(t, int64(61), ipv6.ID)
		require.Equal(t, "2001:db8:c012:d011::", ipv6.IP.String())

		require.Equal(t, 0, ipPool.SizeIPv6())

		ipv6, err = ipPool.NextIPv6()
		require.Equal(t, ErrEmpty, err)
		require.Nil(t, ipv6)
	})
}
