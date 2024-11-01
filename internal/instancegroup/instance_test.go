package instancegroup

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

func TestCreateInstance(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(mockutil.Handler(t,
			[]mockutil.Request{
				{
					Method: "POST", Path: "/servers",
					Status: 201,
					JSON: schema.ServerCreateResponse{
						Server:      schema.Server{ID: 1},
						Action:      schema.Action{ID: 10, Status: "running"},
						NextActions: []schema.Action{{ID: 20, Status: "running"}},
					},
				},
				{
					Method: "GET", Path: "/actions?id=10&id=20&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 10, Status: "success"},
							{ID: 20, Status: "success"},
						},
					},
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
					JSON: schema.ServerCreateResponse{
						Server:      schema.Server{ID: 1},
						Action:      schema.Action{ID: 10, Status: "running"},
						NextActions: []schema.Action{{ID: 20, Status: "running"}},
					},
				},
				{
					Method: "GET", Path: "/actions?id=10&id=20&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 10, Status: "error", Error: &schema.ActionError{Code: "failure", Message: "Something failed"}},
							{ID: 20, Status: "success"},
						},
					},
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
				JSON: schema.ServerDeleteResponse{
					Action: schema.Action{ID: 10, Status: "running"},
				},
			},
			{
				Method: "GET", Path: "/actions?id=10&page=1&sort=status&sort=id",
				Status: 200,
				JSON: schema.ActionListResponse{
					Actions: []schema.Action{
						{ID: 10, Status: "success"},
					},
				},
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
