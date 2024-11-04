package instancegroup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

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

func TestInit(t *testing.T) {
	server := httptest.NewServer(mockutil.Handler(t,
		[]mockutil.Request{
			testutils.GetLocationHel1Request,
			testutils.GetServerTypeCPX11Request,
			testutils.GetImageDebian12Request,
		},
	))

	client := testutils.MakeTestClient(server.URL)

	config := DefaultTestConfig

	group := &instanceGroup{name: "fleeting", config: config, client: client}
	err := group.Init(context.Background())
	require.NoError(t, err)
}

func TestIncrease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(mockutil.Handler(t,
			[]mockutil.Request{
				testutils.GetLocationHel1Request,
				testutils.GetServerTypeCPX11Request,
				testutils.GetImageDebian12Request,
				{
					Method: "POST", Path: "/servers",
					Want: func(t *testing.T, r *http.Request) {
						require.Equal(t, "/servers", r.RequestURI)

						var payload schema.ServerCreateRequest
						mustUnmarshal(t, r.Body, &payload)
						require.Equal(t, "3", payload.Location)
						require.Equal(t, float64(114690387), payload.Image.(float64))
						require.Equal(t, float64(1), payload.ServerType.(float64))
						require.Equal(t, false, payload.PublicNet.EnableIPv4)
						require.Equal(t, true, payload.PublicNet.EnableIPv6)
					},
					Status: 201,
					JSON: schema.ServerCreateResponse{
						Server:      schema.Server{ID: 1, Name: "fleeting-a"},
						Action:      schema.Action{ID: 101, Status: "running"},
						NextActions: []schema.Action{{ID: 102, Status: "running"}},
					},
				},
				{
					Method: "POST", Path: "/servers",
					Want: func(t *testing.T, r *http.Request) {
						require.Equal(t, "/servers", r.RequestURI)

						var payload schema.ServerCreateRequest
						mustUnmarshal(t, r.Body, &payload)
						require.Equal(t, "3", payload.Location)
						require.Equal(t, float64(114690387), payload.Image.(float64))
						require.Equal(t, float64(1), payload.ServerType.(float64))
						require.Equal(t, false, payload.PublicNet.EnableIPv4)
						require.Equal(t, true, payload.PublicNet.EnableIPv6)
					},
					Status: 201,
					JSON: schema.ServerCreateResponse{
						Server:      schema.Server{ID: 2, Name: "fleeting-b"},
						Action:      schema.Action{ID: 201, Status: "running"},
						NextActions: []schema.Action{{ID: 202, Status: "running"}},
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
				{
					Method: "GET", Path: "/actions?id=201&id=202&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 201, Status: "success"},
							{ID: 202, Status: "success"},
						},
					},
				},
			},
		))

		client := testutils.MakeTestClient(server.URL)

		config := DefaultTestConfig

		group := &instanceGroup{name: "fleeting", config: config, client: client}
		err := group.Init(context.Background())
		require.NoError(t, err)

		created, err := group.Increase(context.Background(), 2)
		require.NoError(t, err)
		require.Equal(t, []string{"fleeting-a:1", "fleeting-b:2"}, created)
	})

	t.Run("failure", func(t *testing.T) {
		server := httptest.NewServer(mockutil.Handler(t,
			[]mockutil.Request{
				testutils.GetLocationHel1Request,
				testutils.GetServerTypeCPX11Request,
				testutils.GetImageDebian12Request,
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
					Method: "POST", Path: "/servers",
					Status: 201,
					JSON: schema.ServerCreateResponse{
						Server:      schema.Server{ID: 2, Name: "fleeting-b"},
						Action:      schema.Action{ID: 201, Status: "running"},
						NextActions: []schema.Action{{ID: 202, Status: "running"}},
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
				{
					Method: "GET", Path: "/actions?id=201&id=202&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 201, Status: "success"},
							{ID: 202, Status: "success"},
						},
					},
				},
				{
					Method: "DELETE", Path: "/servers/1",
					Status: 200,
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

		client := testutils.MakeTestClient(server.URL)

		config := DefaultTestConfig

		group := &instanceGroup{name: "fleeting", config: config, client: client}
		err := group.Init(context.Background())
		require.NoError(t, err)

		created, err := group.Increase(context.Background(), 2)
		require.Error(t, err)
		require.Equal(t, []string{"fleeting-b:2"}, created)
	})
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
				JSON: schema.ServerDeleteResponse{
					Action: schema.Action{ID: 103, Status: "running"},
				},
			},
			{
				Method: "DELETE", Path: "/servers/2",
				Status: 200,
				JSON: schema.ServerDeleteResponse{
					Action: schema.Action{ID: 203, Status: "running"},
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
			{
				Method: "GET", Path: "/actions?id=203&page=1&sort=status&sort=id",
				Status: 200,
				JSON: schema.ActionListResponse{
					Actions: []schema.Action{
						{ID: 203, Status: "success"},
					},
				},
			},
		},
	))

	client := testutils.MakeTestClient(server.URL)

	config := DefaultTestConfig

	group := &instanceGroup{name: "fleeting", config: config, client: client}
	err := group.Init(context.Background())
	require.NoError(t, err)

	deleted, err := group.Decrease(context.Background(), []string{"fleeting-a:1", "fleeting-b:2"})
	require.NoError(t, err)
	require.Equal(t, []string{"fleeting-a:1", "fleeting-b:2"}, deleted)
}
