package instancegroup

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/testutils"
)

func TestCreateInstance(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(mockutil.Handler(t,
			[]mockutil.Request{
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
					Method: "GET", Path: "/actions?id=10&id=20&page=1&sort=status&sort=id",
					Status: 200,
					JSONRaw: `{
						"actions": [
							{ "id": 10, "status": "success" },
							{ "id": 20, "status": "success" }
						]
					}`,
				},
			},
		))

		ctx := context.Background()
		client := testutils.MakeTestClient(server.URL)

		instance, err := CreateInstance(ctx, client, hcloud.ServerCreateOpts{
			Name:       "test",
			ServerType: &hcloud.ServerType{Name: "cpx11"},
			Image:      &hcloud.Image{Name: "debian-12"},
		})
		require.NoError(t, err)
		require.NotNil(t, instance.waitFunc)

		err = instance.Wait()
		require.NoError(t, err)
		require.Nil(t, instance.waitFunc)
	})

	t.Run("failure", func(t *testing.T) {
		server := httptest.NewServer(mockutil.Handler(t,
			[]mockutil.Request{
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
					Method: "GET", Path: "/actions?id=10&id=20&page=1&sort=status&sort=id",
					Status: 200,
					JSONRaw: `{
						"actions": [
							{ "id": 10, "status": "error", "error": { "code": "failure", "message": "Something failed" }},
							{ "id": 20, "status": "success" }
						]
					}`,
				},
			},
		))

		ctx := context.Background()
		client := testutils.MakeTestClient(server.URL)

		instance, err := CreateInstance(ctx, client, hcloud.ServerCreateOpts{
			Name:       "test",
			ServerType: &hcloud.ServerType{Name: "cpx11"},
			Image:      &hcloud.Image{Name: "debian-12"},
		})
		require.NoError(t, err)
		require.NotNil(t, instance.waitFunc)

		err = instance.Wait()
		require.Error(t, err)
		require.Nil(t, instance.waitFunc)
	})
}

func TestInstanceDelete(t *testing.T) {
	server := httptest.NewServer(mockutil.Handler(t,
		[]mockutil.Request{
			{
				Method: "DELETE", Path: "/servers/1",
				Status: 201,
				JSONRaw: `{
					"action": { "id": 10, "status": "running" }
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
		},
	))

	ctx := context.Background()
	client := testutils.MakeTestClient(server.URL)

	instance := NewInstance(1)

	err := instance.Delete(ctx, client)
	require.NoError(t, err)
	require.NotNil(t, instance.waitFunc)

	err = instance.Wait()
	require.NoError(t, err)
	require.Nil(t, instance.waitFunc)
}
