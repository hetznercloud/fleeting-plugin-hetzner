package instancegroup

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/testutils"
)

var (
	DefaultTestConfig = Config{
		Location:           "hel1",
		ServerType:         "cpx11",
		Image:              "debian-12",
		PublicIPv4Disabled: true,
	}
)

func makeTestClient(endpoint string) *hcloud.Client {
	return hcloud.NewClient(
		hcloud.WithEndpoint(endpoint),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: func(_ int) time.Duration { return 0 }, MaxRetries: 3}),
		hcloud.WithPollBackoffFunc(func(_ int) time.Duration { return 0 }),
	)
}

func TestInit(t *testing.T) {
	server := httptest.NewServer(mockutil.Handler(t,
		[]mockutil.Request{
			testutils.GetLocationHel1Request,
			testutils.GetServerTypeCPX11Request,
			testutils.GetImageDebian12Request,
		},
	))

	client := makeTestClient(server.URL)

	group := New(client, "dummy", DefaultTestConfig)

	err := group.Init(context.Background())
	require.NoError(t, err)
}

func TestIncrease(t *testing.T) {
	server := httptest.NewServer(mockutil.Handler(t,
		[]mockutil.Request{
			testutils.GetLocationHel1Request,
			testutils.GetServerTypeCPX11Request,
			testutils.GetImageDebian12Request,
			{
				Method: "POST", Path: "/servers",
				Status: 201,
				JSONRaw: `{
					"server": { "id": 1 },
					"action": { "id": 10, "status": "running" },
					"next_actions": [
						{ "id": 20, "status": "running" }
					]
				}`,
			},
			{
				Method: "POST", Path: "/servers",
				Status: 201,
				JSONRaw: `{
					"server": { "id": 2 },
					"action": { "id": 30, "status": "running" },
					"next_actions": [
						{ "id": 40, "status": "running" }
					]
				}`,
			},
			{
				Method: "GET", Path: "/actions?id=10&id=20&page=1&sort=status&sort=id",
				Status: 200,
				JSONRaw: `{
					"actions": [
						{ "id": 10, "status": "success" },
						{ "id": 20, "status": "success" }
					]
				}`,
			},
			{
				Method: "GET", Path: "/actions?id=30&id=40&page=1&sort=status&sort=id",
				Status: 200,
				JSONRaw: `{
					"actions": [
						{ "id": 30, "status": "success" },
						{ "id": 40, "status": "success" }
					]
				}`,
			},
		},
	))

	client := makeTestClient(server.URL)

	group := New(client, "dummy", DefaultTestConfig)
	err := group.Init(context.Background())
	require.NoError(t, err)

	created, err := group.Increase(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, created)
}

func TestDecrease(t *testing.T) {
	server := httptest.NewServer(mockutil.Handler(t,
		[]mockutil.Request{
			testutils.GetLocationHel1Request,
			testutils.GetServerTypeCPX11Request,
			testutils.GetImageDebian12Request,
			{
				Method: "DELETE", Path: "/servers/1",
				Status: 200,
				JSONRaw: `{
					"action": { "id": 10, "status": "running" }
				}`,
			},
			{
				Method: "DELETE", Path: "/servers/2",
				Status: 200,
				JSONRaw: `{
					"action": { "id": 20, "status": "running" }
				}`,
			},
			{
				Method: "GET", Path: "/actions?id=10&page=1&sort=status&sort=id",
				Status: 200,
				JSONRaw: `{
					"actions": [
						{ "id": 10, "status": "success" }
					]
				}`,
			},
			{
				Method: "GET", Path: "/actions?id=20&page=1&sort=status&sort=id",
				Status: 200,
				JSONRaw: `{
					"actions": [
						{ "id": 20, "status": "success" }
					]
				}`,
			},
		},
	))

	client := makeTestClient(server.URL)

	group := New(client, "dummy", DefaultTestConfig)
	err := group.Init(context.Background())
	require.NoError(t, err)

	deleted, err := group.Decrease(context.Background(), []int64{1, 2})
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, deleted)
}
