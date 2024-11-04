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

func TestInstanceFromServer(t *testing.T) {
	instance := InstanceFromServer(&hcloud.Server{ID: 1, Name: "fleeting-a", Status: hcloud.ServerStatusRunning})
	require.Equal(t, int64(1), instance.ID)
	require.Equal(t, "fleeting-a", instance.Name)
	require.NotNil(t, instance.Server)
	require.Equal(t, hcloud.ServerStatusRunning, instance.Server.Status)
}

func TestInstanceFromIID(t *testing.T) {
	testCases := []struct {
		name     string
		iid      string
		instance *Instance
	}{
		{
			name:     "success",
			iid:      "fleeting-a:1",
			instance: &Instance{Name: "fleeting-a", ID: 1},
		},
		{
			name:     "fail no separator",
			iid:      "fleeting-a-1",
			instance: nil,
		},
		{
			name:     "fail to many separator",
			iid:      "fleeting:a:1",
			instance: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			instance, err := InstanceFromIID(testCase.iid)
			if testCase.instance == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.instance, instance)
		})
	}
}

func TestInstanceIID(t *testing.T) {
	testCases := []struct {
		name     string
		instance *Instance
		iid      string
	}{
		{
			name:     "success",
			instance: &Instance{Name: "fleeting-a", ID: 1},
			iid:      "fleeting-a:1",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.iid, testCase.instance.IID())
		})
	}
}

func TestCreateInstance(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(mockutil.Handler(t,
			[]mockutil.Request{
				{
					Method: "POST", Path: "/servers",
					Status: 201,
					JSON: schema.ServerCreateResponse{
						Server:      schema.Server{ID: 1, Name: "fleeting-a"},
						Action:      schema.Action{ID: 101, Status: "running"},
						NextActions: []schema.Action{{ID: 102, Status: "running"}},
					},
				},
				{
					Method: "GET", Path: "/actions?id=101&id=102&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 101, Status: "success"},
							{ID: 102, Status: "success"},
						},
					},
				},
			},
		))

		ctx := context.Background()
		client := testutils.MakeTestClient(server.URL)

		instance, err := CreateInstance(ctx, client, hcloud.ServerCreateOpts{
			Name:       "fleeting-a",
			ServerType: &hcloud.ServerType{Name: "cpx11"},
			Image:      &hcloud.Image{Name: "debian-12"},
		})
		require.NoError(t, err)
		require.NotNil(t, instance.waitFn)

		err = instance.wait()
		require.NoError(t, err)
		require.Nil(t, instance.waitFn)
	})

	t.Run("failure", func(t *testing.T) {
		server := httptest.NewServer(mockutil.Handler(t,
			[]mockutil.Request{
				{
					Method: "POST", Path: "/servers",
					Status: 201,
					JSON: schema.ServerCreateResponse{
						Server:      schema.Server{ID: 1, Name: "fleeting-a"},
						Action:      schema.Action{ID: 101, Status: "running"},
						NextActions: []schema.Action{{ID: 102, Status: "running"}},
					},
				},
				{
					Method: "GET", Path: "/actions?id=101&id=102&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 101, Status: "error", Error: &schema.ActionError{Code: "failure", Message: "Something failed"}},
							{ID: 102, Status: "success"},
						},
					},
				},
			},
		))

		ctx := context.Background()
		client := testutils.MakeTestClient(server.URL)

		instance, err := CreateInstance(ctx, client, hcloud.ServerCreateOpts{
			Name:       "fleeting-a",
			ServerType: &hcloud.ServerType{Name: "cpx11"},
			Image:      &hcloud.Image{Name: "debian-12"},
		})
		require.NoError(t, err)
		require.NotNil(t, instance.waitFn)

		err = instance.wait()
		require.Error(t, err)
		require.Nil(t, instance.waitFn)
	})
}

func TestInstanceDelete(t *testing.T) {
	server := httptest.NewServer(mockutil.Handler(t,
		[]mockutil.Request{
			{
				Method: "DELETE", Path: "/servers/1",
				Status: 201,
				JSON: schema.ServerDeleteResponse{
					Action: schema.Action{ID: 103, Status: "running"},
				},
			},
			{
				Method: "GET", Path: "/actions?id=103&page=1&sort=status&sort=id",
				Status: 200,
				JSON: schema.ActionListResponse{
					Actions: []schema.Action{
						{ID: 103, Status: "success"},
					},
				},
			},
		},
	))

	ctx := context.Background()
	client := testutils.MakeTestClient(server.URL)

	instance := &Instance{Name: "fleeting-a", ID: 1}

	err := instance.Delete(ctx, client)
	require.NoError(t, err)
	require.NotNil(t, instance.waitFn)

	err = instance.wait()
	require.NoError(t, err)
	require.Nil(t, instance.waitFn)
}
