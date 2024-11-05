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
		VolumeSize:         10,
		PublicIPv4Disabled: true,
	}
)

func TestInit(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
		want   func(t *testing.T, group *instanceGroup)
	}{
		{
			name:   "success",
			config: DefaultTestConfig,
			want: func(t *testing.T, group *instanceGroup) {
				require.NotNil(t, group.location)
				require.NotNil(t, group.serverType)
				require.NotNil(t, group.image)
				require.Equal(t, map[string]string{"instance-group": "fleeting"}, group.labels)
			},
		},
		{
			name: "success extra labels",
			config: Config{
				Location:   "hel1",
				ServerType: "cpx11",
				Image:      "debian-12",
				Labels:     map[string]string{"foo": "bar"},
			},
			want: func(t *testing.T, group *instanceGroup) {
				require.NotNil(t, group.location)
				require.NotNil(t, group.serverType)
				require.NotNil(t, group.image)
				require.Equal(t, map[string]string{"instance-group": "fleeting", "foo": "bar"}, group.labels)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(mockutil.Handler(t,
				[]mockutil.Request{
					testutils.GetLocationHel1Request,
					testutils.GetServerTypeCPX11Request,
					testutils.GetImageDebian12Request,
				},
			))

			client := testutils.MakeTestClient(server.URL)

			group := &instanceGroup{name: "fleeting", config: testCase.config, client: client}
			err := group.Init(context.Background())
			require.NoError(t, err)

			testCase.want(t, group)
		})
	}
}

func TestIncrease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config,
			[]mockutil.Request{
				{
					Method: "POST", Path: "/volumes",
					Want: func(t *testing.T, r *http.Request) {
						require.Equal(t, "/volumes", r.RequestURI)

						var payload schema.VolumeCreateRequest
						mustUnmarshal(t, r.Body, &payload)
						require.Equal(t, "fleeting-a", payload.Name)
						require.Equal(t, 10, payload.Size)
						require.Equal(t, &map[string]string{"instance-group": "fleeting"}, payload.Labels)
					},
					Status: 201,
					JSON: schema.VolumeCreateResponse{
						Volume: schema.Volume{ID: 1, Name: "fleeting-a"},
						Action: &schema.Action{ID: 101, Status: "running"},
					},
				},
				{
					Method: "POST", Path: "/volumes",
					Status: 201,
					JSON: schema.VolumeCreateResponse{
						Volume: schema.Volume{ID: 2, Name: "fleeting-b"},
						Action: &schema.Action{ID: 201, Status: "running"},
					},
				},
				{
					Method: "GET", Path: "/actions?id=101&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 101, Status: "success"},
						},
					},
				},
				{
					Method: "GET", Path: "/actions?id=201&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 201, Status: "success"},
						},
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
						require.Equal(t, int64(1), payload.Volumes[0])
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
						require.Equal(t, int64(2), payload.Volumes[0])
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
			})

		created, err := group.Increase(ctx, 2)
		require.NoError(t, err)
		require.Equal(t, []string{"fleeting-a:1", "fleeting-b:2"}, created)
	})

	t.Run("failure", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config,
			[]mockutil.Request{
				{
					Method: "POST", Path: "/volumes",
					Status: 201,
					JSON: schema.VolumeCreateResponse{
						Volume: schema.Volume{ID: 1, Name: "fleeting-a"},
						Action: &schema.Action{ID: 101, Status: "running"},
					},
				},
				{
					Method: "POST", Path: "/volumes",
					Status: 201,
					JSON: schema.VolumeCreateResponse{
						Volume: schema.Volume{ID: 2, Name: "fleeting-b"},
						Action: &schema.Action{ID: 201, Status: "running"},
					},
				},
				{
					Method: "GET", Path: "/actions?id=101&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 101, Status: "success"},
						},
					},
				},
				{
					Method: "GET", Path: "/actions?id=201&page=1&sort=status&sort=id",
					Status: 200,
					JSON: schema.ActionListResponse{
						Actions: []schema.Action{
							{ID: 201, Status: "success"},
						},
					},
				},
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
				{
					Method: "DELETE", Path: "/volumes/1",
					Status: 204,
				},
			},
		)

		created, err := group.Increase(ctx, 2)
		require.Error(t, err)
		require.Equal(t, []string{"fleeting-b:2"}, created)
	})
}

func TestDecrease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config,
			[]mockutil.Request{
				{
					Method: "GET", Path: "/volumes?label_selector=instance-group%3Dfleeting&page=1",
					Status: 200,
					JSON: schema.VolumeListResponse{
						Volumes: []schema.Volume{
							{ID: 1, Name: "fleeting-a"},
							{ID: 2, Name: "fleeting-b"},
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
				{
					Method: "DELETE", Path: "/volumes/1",
					Status: 204,
				},
				{
					Method: "DELETE", Path: "/volumes/2",
					Status: 204,
				},
			},
		)

		deleted, err := group.Decrease(ctx, []string{"fleeting-a:1", "fleeting-b:2"})
		require.NoError(t, err)
		require.Equal(t, []string{"fleeting-a:1", "fleeting-b:2"}, deleted)
	})
}
