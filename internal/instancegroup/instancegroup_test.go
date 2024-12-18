package instancegroup

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
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

func TestNew(t *testing.T) {
	New(hcloud.NewClient(), hclog.Default(), "fleeting", DefaultTestConfig)
}

func TestInit(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
		run    func(t *testing.T, group *instanceGroup, server *mockutil.Server)
	}{
		{
			name: "success",
			config: Config{
				Location:        "hel1",
				ServerType:      "cpx11",
				Image:           "debian-12",
				VolumeSize:      10,
				PrivateNetworks: []string{"network"},
				SSHKeys:         []string{"ssh-key"},
				Labels:          map[string]string{"key": "value"},
			},
			run: func(t *testing.T, group *instanceGroup, server *mockutil.Server) {
				server.Expect([]mockutil.Request{
					testutils.GetLocationHel1Request,
					testutils.GetServerTypeCPX11Request,
					testutils.GetImageDebian12Request,
					{
						Method: "GET", Path: "/networks?name=network",
						Status: 200,
						JSON: schema.NetworkListResponse{
							Networks: []schema.Network{{ID: 1, Name: "network"}},
						},
					},
					{
						Method: "GET", Path: "/ssh_keys?name=ssh-key",
						Status: 200,
						JSON: schema.SSHKeyListResponse{
							SSHKeys: []schema.SSHKey{{ID: 1, Name: "ssh-key"}},
						},
					},
				})

				err := group.Init(context.Background())
				require.NoError(t, err)

				require.Equal(t, "hel1", group.location.Name)
				require.Equal(t, "cpx11", group.serverType.Name)
				require.Equal(t, "debian-12", group.image.Name)
				require.Equal(t, "network", group.privateNetworks[0].Name)
				require.Equal(t, "ssh-key", group.sshKeys[0].Name)
				require.Equal(t, map[string]string{"instance-group": "fleeting", "key": "value"}, group.labels)
			},
		},
		{
			name:   "invalid location",
			config: DefaultTestConfig,
			run: func(t *testing.T, group *instanceGroup, server *mockutil.Server) {
				server.Expect([]mockutil.Request{
					{
						Method: "GET", Path: "/locations?name=hel1",
						Status: 200,
						JSON: schema.LocationListResponse{
							Locations: []schema.Location{},
						},
					},
				})

				err := group.Init(context.Background())
				require.EqualError(t, err, "location not found: hel1")
			},
		},
		{
			name:   "invalid server type",
			config: DefaultTestConfig,
			run: func(t *testing.T, group *instanceGroup, server *mockutil.Server) {
				server.Expect([]mockutil.Request{
					testutils.GetLocationHel1Request,
					{
						Method: "GET", Path: "/server_types?name=cpx11",
						Status: 200,
						JSON: schema.ServerTypeListResponse{
							ServerTypes: []schema.ServerType{},
						},
					},
				})

				err := group.Init(context.Background())
				require.EqualError(t, err, "server type not found: cpx11")
			},
		},
		{
			name:   "invalid image",
			config: DefaultTestConfig,
			run: func(t *testing.T, group *instanceGroup, server *mockutil.Server) {
				server.Expect([]mockutil.Request{
					testutils.GetLocationHel1Request,
					testutils.GetServerTypeCPX11Request,
					{
						Method: "GET", Path: "/images?architecture=x86&include_deprecated=true&name=debian-12",
						Status: 200,
						JSON: schema.ImageListResponse{
							Images: []schema.Image{},
						},
					},
				})

				err := group.Init(context.Background())
				require.EqualError(t, err, "image not found: debian-12")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := mockutil.NewServer(t, nil)
			client := testutils.MakeTestClient(server.URL)

			log := hclog.New(hclog.DefaultOptions)

			group := &instanceGroup{name: "fleeting", config: testCase.config, log: log, client: client}

			testCase.run(t, group, server)
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
						require.Equal(t, int64(114690387), payload.Image.ID)
						require.Equal(t, int64(1), payload.ServerType.ID)
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
						require.Equal(t, int64(114690387), payload.Image.ID)
						require.Equal(t, int64(1), payload.ServerType.ID)
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

func TestList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config,
			[]mockutil.Request{
				{
					Method: "GET", Path: "/servers?label_selector=instance-group%3Dfleeting&page=1",
					Status: 200,
					JSON: schema.ServerListResponse{
						Servers: []schema.Server{
							{ID: 1, Name: "fleeting-a"},
							{ID: 2, Name: "fleeting-b"},
						},
					},
				},
			},
		)

		result, err := group.List(ctx)
		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, int64(1), result[0].ID)
		require.Equal(t, "fleeting-a", result[0].Name)
		require.Equal(t, int64(2), result[1].ID)
		require.Equal(t, "fleeting-b", result[1].Name)
	})
}

func TestGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config,
			[]mockutil.Request{
				{
					Method: "GET", Path: "/servers/1",
					Status: 200,
					JSON: schema.ServerGetResponse{
						Server: schema.Server{ID: 1, Name: "fleeting-a"},
					},
				},
			},
		)

		result, err := group.Get(ctx, "fleeting-a:1")
		require.NoError(t, err)
		require.Equal(t, int64(1), result.ID)
		require.Equal(t, "fleeting-a", result.Name)
	})
}

func TestSanity(t *testing.T) {
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
							{ID: 1, Name: "fleeting-a", Server: hcloud.Ptr[int64](1)},
							{ID: 2, Name: "fleeting-b"},
						},
					},
				},
				{
					Method: "DELETE", Path: "/volumes/2",
					Status: 204,
				},
			},
		)

		err := group.Sanity(ctx)
		require.NoError(t, err)
	})
}
