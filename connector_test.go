package hetzner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"go.uber.org/mock/gomock"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/instancegroup"
	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/testutils"
)

func TestUploadSSHPublicKey(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, ctx context.Context, group *InstanceGroup, server *mockutil.Server)
	}{
		{
			name: "existing fingerprint",
			run: func(t *testing.T, ctx context.Context, group *InstanceGroup, server *mockutil.Server) {
				_, sshKey := sshKeyFixture(t)

				server.Expect([]mockutil.Request{
					{
						Method: "GET",
						Want: func(t *testing.T, r *http.Request) {
							require.Equal(t, "/ssh_keys?fingerprint="+url.QueryEscape(sshKey.Fingerprint), r.RequestURI)
						},
						Status: 200,
						JSON:   schema.SSHKeyListResponse{SSHKeys: []schema.SSHKey{sshKey}},
					},
				})

				result, err := group.UploadSSHPublicKey(ctx, []byte(sshKey.PublicKey))
				require.NoError(t, err)

				require.Equal(t, int64(1), result.ID)
				require.Equal(t, "fleeting", result.Name)
			},
		},
		{
			name: "new",
			run: func(t *testing.T, ctx context.Context, group *InstanceGroup, server *mockutil.Server) {
				_, sshKey := sshKeyFixture(t)

				server.Expect([]mockutil.Request{
					{
						Method: "GET",
						Want: func(t *testing.T, r *http.Request) {
							require.Equal(t, "/ssh_keys?fingerprint="+url.QueryEscape(sshKey.Fingerprint), r.RequestURI)
						},
						Status: 200,
						JSON:   schema.SSHKeyListResponse{SSHKeys: []schema.SSHKey{}},
					},
					{
						Method: "GET", Path: "/ssh_keys?name=fleeting",
						Status: 200,
						JSON:   schema.SSHKeyListResponse{SSHKeys: []schema.SSHKey{}},
					},
					{
						Method: "POST", Path: "/ssh_keys",
						Want: func(t *testing.T, r *http.Request) {
							body, err := io.ReadAll(r.Body)
							require.NoError(t, err)

							publicKey, err := json.Marshal(sshKey.PublicKey)
							require.NoError(t, err)

							require.JSONEq(t, fmt.Sprintf(`{
								"name": "fleeting",
								"public_key": %s
							}`, publicKey), string(body))
						},
						Status: 201,
						JSON:   schema.SSHKeyCreateResponse{SSHKey: sshKey},
					},
				})

				result, err := group.UploadSSHPublicKey(ctx, []byte(sshKey.PublicKey))
				require.NoError(t, err)

				require.Equal(t, int64(1), result.ID)
				require.Equal(t, "fleeting", result.Name)
			},
		},
		{
			name: "new with existing name",
			run: func(t *testing.T, ctx context.Context, group *InstanceGroup, server *mockutil.Server) {
				_, sshKey := sshKeyFixture(t)

				server.Expect([]mockutil.Request{
					{
						Method: "GET",
						Want: func(t *testing.T, r *http.Request) {
							require.Equal(t, "/ssh_keys?fingerprint="+url.QueryEscape(sshKey.Fingerprint), r.RequestURI)
						},
						Status: 200,
						JSON:   schema.SSHKeyListResponse{SSHKeys: []schema.SSHKey{}},
					},
					{
						Method: "GET", Path: "/ssh_keys?name=fleeting",
						Status: 200,
						JSON:   schema.SSHKeyListResponse{SSHKeys: []schema.SSHKey{sshKey}},
					},
					{
						Method: "DELETE", Path: "/ssh_keys/1",
						Status: 204,
					},
					{
						Method: "POST", Path: "/ssh_keys",
						Want: func(t *testing.T, r *http.Request) {
							body, err := io.ReadAll(r.Body)
							require.NoError(t, err)

							publicKey, err := json.Marshal(sshKey.PublicKey)
							require.NoError(t, err)

							require.JSONEq(t, fmt.Sprintf(`{
								"name": "fleeting",
								"public_key": %s
							}`, publicKey), string(body))
						},
						Status: 201,
						JSON:   schema.SSHKeyCreateResponse{SSHKey: sshKey},
					},
				})

				result, err := group.UploadSSHPublicKey(ctx, []byte(sshKey.PublicKey))
				require.NoError(t, err)

				require.Equal(t, int64(1), result.ID)
				require.Equal(t, "fleeting", result.Name)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			mock := instancegroup.NewMockInstanceGroup(ctrl)

			server := mockutil.NewServer(t, nil)
			client := testutils.MakeTestClient(server.URL)

			group := &InstanceGroup{
				Name:     "fleeting",
				log:      hclog.New(hclog.DefaultOptions),
				settings: provider.Settings{},
				group:    mock,
				client:   client,
			}

			testCase.run(t, ctx, group, server)
		})
	}
}
