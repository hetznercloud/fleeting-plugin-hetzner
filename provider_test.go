package hetzner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"go.uber.org/mock/gomock"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sshutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/instancegroup"
	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/testutils"
)

func sshKeyFixture(t *testing.T) ([]byte, schema.SSHKey) {
	t.Helper()

	privateKey, publicKey, err := sshutil.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	fingerprint, err := sshutil.GetPublicKeyFingerprint(publicKey)
	if err != nil {
		t.Fatal(err)
	}

	return privateKey, schema.SSHKey{ID: 1, Name: "fleeting", Fingerprint: fingerprint, PublicKey: string(publicKey)}
}

func TestInit(t *testing.T) {
	sshPrivateKey, sshKey := sshKeyFixture(t)

	testCases := []struct {
		name     string
		requests []mockutil.Request
		run      func(t *testing.T, group *InstanceGroup, ctx context.Context, log hclog.Logger, settings provider.Settings)
	}{
		{name: "generated ssh key upload",
			requests: []mockutil.Request{
				{Method: "GET",
					Want: func(t *testing.T, r *http.Request) {
						require.True(t, strings.HasPrefix(r.RequestURI, "/ssh_keys?fingerprint="))
					},
					Status: 200,
					JSON:   schema.SSHKeyListResponse{SSHKeys: []schema.SSHKey{}},
				},
				{Method: "GET", Path: "/ssh_keys?name=fleeting",
					Status: 200,
					JSON:   schema.SSHKeyListResponse{SSHKeys: []schema.SSHKey{}},
				},
				{Method: "POST", Path: "/ssh_keys",
					Status: 201,
					JSON:   schema.SSHKeyCreateResponse{SSHKey: sshKey},
				},
				testutils.GetLocationHel1Request,
				testutils.GetServerTypeCPX11Request,
				testutils.GetImageDebian12Request,
				{Method: "GET", Path: "/ssh_keys?name=fleeting",
					Status: 200,
					JSON: schema.SSHKeyListResponse{
						SSHKeys: []schema.SSHKey{sshKey},
					},
				},
			},
			run: func(t *testing.T, group *InstanceGroup, ctx context.Context, log hclog.Logger, settings provider.Settings) {
				info, err := group.Init(ctx, log, settings)
				require.NoError(t, err)
				require.Equal(t, "hetzner/hel1/cpx11/fleeting", info.ID)
			},
		},
		{name: "static ssh key upload",
			requests: []mockutil.Request{
				{Method: "GET", Path: "/ssh_keys?fingerprint=" + url.QueryEscape(sshKey.Fingerprint),
					Status: 200,
					JSON: schema.SSHKeyListResponse{
						SSHKeys: []schema.SSHKey{},
					},
				},
				{Method: "GET", Path: "/ssh_keys?name=fleeting",
					Status: 200,
					JSON: schema.SSHKeyListResponse{
						SSHKeys: []schema.SSHKey{},
					},
				},
				{Method: "POST", Path: "/ssh_keys",
					Status: 201,
					JSON:   schema.SSHKeyCreateResponse{SSHKey: sshKey},
				},
				testutils.GetLocationHel1Request,
				testutils.GetServerTypeCPX11Request,
				testutils.GetImageDebian12Request,
				{Method: "GET", Path: "/ssh_keys?name=fleeting",
					Status: 200,
					JSON: schema.SSHKeyListResponse{
						SSHKeys: []schema.SSHKey{sshKey},
					},
				},
			},
			run: func(t *testing.T, group *InstanceGroup, ctx context.Context, log hclog.Logger, settings provider.Settings) {
				settings.UseStaticCredentials = true
				settings.Key = sshPrivateKey

				info, err := group.Init(ctx, log, settings)
				require.NoError(t, err)
				require.Equal(t, "hetzner/hel1/cpx11/fleeting", info.ID)
			},
		},
		{name: "static ssh key existing",
			requests: []mockutil.Request{
				{Method: "GET", Path: "/ssh_keys?fingerprint=" + url.QueryEscape(sshKey.Fingerprint),
					Status: 200,
					JSON: schema.SSHKeyListResponse{
						SSHKeys: []schema.SSHKey{sshKey},
					},
				},
				testutils.GetLocationHel1Request,
				testutils.GetServerTypeCPX11Request,
				testutils.GetImageDebian12Request,
				{Method: "GET", Path: "/ssh_keys?name=fleeting",
					Status: 200,
					JSON: schema.SSHKeyListResponse{
						SSHKeys: []schema.SSHKey{sshKey},
					},
				},
			},
			run: func(t *testing.T, group *InstanceGroup, ctx context.Context, log hclog.Logger, settings provider.Settings) {
				settings.UseStaticCredentials = true
				settings.Key = sshPrivateKey

				info, err := group.Init(ctx, log, settings)
				require.NoError(t, err)
				require.Equal(t, "hetzner/hel1/cpx11/fleeting", info.ID)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(mockutil.Handler(t, testCase.requests))

			group := &InstanceGroup{
				Name:       "fleeting",
				Token:      "dummy",
				Endpoint:   server.URL,
				Location:   "hel1",
				ServerType: "cpx11",
				Image:      "debian-12",

				client: hcloud.NewClient(),
			}
			ctx := context.Background()
			log := hclog.NewNullLogger()
			settings := provider.Settings{}

			testCase.run(t, group, ctx, log, settings)
		})
	}
}

func TestIncrease(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context)
	}{
		{name: "success",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.size = 3

				mock.EXPECT().
					Increase(ctx, 2).
					Return([]int64{1, 2}, nil)

				count, err := group.Increase(ctx, 2)
				require.NoError(t, err)
				require.Equal(t, 2, count)
				require.Equal(t, 5, group.size)
			},
		},
		{name: "failure",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.size = 3

				mock.EXPECT().
					Increase(ctx, 2).
					Return([]int64{1}, fmt.Errorf("some error"))

				count, err := group.Increase(ctx, 2)
				require.Error(t, err)
				require.Equal(t, 1, count)
				require.Equal(t, 4, group.size)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:      hclog.NewNullLogger(),
				settings: provider.Settings{},
				group:    mock,
			}

			testCase.run(t, mock, group, context.Background())
		})
	}
}

func TestDecrease(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context)
	}{
		{name: "success",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.size = 2

				mock.EXPECT().
					Decrease(ctx, []int64{1, 2}).
					Return([]int64{1, 2}, nil)

				result, err := group.Decrease(ctx, []string{"1", "2"})
				require.NoError(t, err)
				require.Equal(t, []string{"1", "2"}, result)

				require.Equal(t, 0, group.size)
			},
		},
		{name: "failure",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.size = 2

				mock.EXPECT().
					Decrease(ctx, []int64{1, 2}).
					Return([]int64{1}, fmt.Errorf("some error"))

				result, err := group.Decrease(ctx, []string{"1", "2"})
				require.Error(t, err)
				require.Equal(t, []string{"1"}, result)

				require.Equal(t, 1, group.size)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:      hclog.NewNullLogger(),
				settings: provider.Settings{},
				group:    mock,
			}

			testCase.run(t, mock, group, context.Background())
		})
	}
}

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context)
	}{
		{name: "success",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				mock.EXPECT().
					List(ctx).
					Return([]*hcloud.Server{{ID: 1, Status: hcloud.ServerStatusRunning}}, nil)

				updateIDs := make([]string, 0)
				err := group.Update(ctx, func(id string, state provider.State) {
					updateIDs = append(updateIDs, id)
				})
				require.NoError(t, err)
				require.Equal(t, []string{"1"}, updateIDs)
				require.Equal(t, 1, group.size)
			},
		},
		{name: "failure",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				mock.EXPECT().
					List(ctx).
					Return(nil, fmt.Errorf("some error"))

				err := group.Update(ctx, func(id string, state provider.State) {
					require.Fail(t, "update should not have been called")
				})
				require.Error(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:      hclog.NewNullLogger(),
				settings: provider.Settings{},
				group:    mock,
			}

			testCase.run(t, mock, group, context.Background())
		})
	}
}

func TestConnectInfo(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context)
	}{
		{name: "success",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.settings.UseStaticCredentials = true
				group.settings.Key = []byte("-----BEGIN OPENSSH PRIVATE KEY-----")

				mock.EXPECT().
					Get(ctx, gomock.Any()).
					Return(hcloud.ServerFromSchema(schema.Server{
						ID:     1,
						Name:   "existing-1",
						Status: "running",
						Image: &schema.Image{
							OSFlavor:  "debian",
							OSVersion: hcloud.Ptr("12"),
						},
						ServerType: schema.ServerType{
							Name:         "cpx11",
							Architecture: "x86",
						},
						PublicNet: schema.ServerPublicNet{
							IPv4: schema.ServerPublicNetIPv4{
								IP: "37.1.1.1",
							},
						},
						PrivateNet: []schema.ServerPrivateNet{
							{IP: "10.0.1.2"},
						},
					}), nil)

				result, err := group.ConnectInfo(ctx, "1")
				require.NoError(t, err)
				require.Equal(t, provider.ConnectInfo{
					ConnectorConfig: provider.ConnectorConfig{
						OS:                   "debian",
						Arch:                 "amd64",
						Protocol:             "ssh",
						UseStaticCredentials: true,
						Username:             "root",
						Key:                  []byte("-----BEGIN OPENSSH PRIVATE KEY-----"),
					},
					ID:           "1",
					ExternalAddr: "37.1.1.1",
					InternalAddr: "10.0.1.2",
				}, result)
			},
		},
		{name: "success ipv6",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				group.settings.UseStaticCredentials = true
				group.settings.Key = []byte("-----BEGIN OPENSSH PRIVATE KEY-----")

				mock.EXPECT().
					Get(ctx, gomock.Any()).
					Return(hcloud.ServerFromSchema(schema.Server{
						ID:     1,
						Name:   "existing-1",
						Status: "running",
						Image: &schema.Image{
							OSFlavor:  "debian",
							OSVersion: hcloud.Ptr("12"),
						},
						ServerType: schema.ServerType{
							Name:         "cpx11",
							Architecture: "x86",
						},
						PublicNet: schema.ServerPublicNet{
							IPv6: schema.ServerPublicNetIPv6{
								IP: "2a01:4f8:1c19:1403::/64",
							},
						},
					}), nil)

				result, err := group.ConnectInfo(ctx, "1")
				require.NoError(t, err)
				require.Equal(t, provider.ConnectInfo{
					ConnectorConfig: provider.ConnectorConfig{
						OS:                   "debian",
						Arch:                 "amd64",
						Protocol:             "ssh",
						UseStaticCredentials: true,
						Username:             "root",
						Key:                  []byte("-----BEGIN OPENSSH PRIVATE KEY-----"),
					},
					ID:           "1",
					ExternalAddr: "2a01:4f8:1c19:1403::1",
				}, result)
			},
		},
		{name: "failure",
			run: func(t *testing.T, mock *instancegroup.MockInstanceGroup, group *InstanceGroup, ctx context.Context) {
				mock.EXPECT().
					Get(ctx, gomock.Any()).
					Return(nil, fmt.Errorf("some error"))

				result, err := group.ConnectInfo(ctx, "1")
				require.Error(t, err)
				require.Equal(t, provider.ConnectInfo{
					ConnectorConfig: provider.ConnectorConfig{
						Protocol: "ssh",
						Username: "root",
					},
				}, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)
			group := &InstanceGroup{
				log:      hclog.NewNullLogger(),
				settings: provider.Settings{},
				group:    mock,
			}

			group.settings.Protocol = "ssh"
			group.settings.Username = "root"

			testCase.run(t, mock, group, context.Background())
		})
	}
}
