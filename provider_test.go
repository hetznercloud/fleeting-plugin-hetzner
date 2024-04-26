package hetzner

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"gitlab.com/hiboxsystems/fleeting-plugin-hetzner/internal/hetzner"
	"gitlab.com/hiboxsystems/fleeting-plugin-hetzner/internal/hetzner/fake"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func setupFakeClient(t *testing.T, setup func(client *fake.Client)) *InstanceGroup {
	t.Helper()

	oldClient := newClient
	t.Cleanup(func() {
		newClient = oldClient
	})

	// Create a fake client which overrides all the Hetzner API calls and returns dummy data
	newClient = func(_ hetzner.Config, _ string, _ string) (hetzner.Client, error) {
		client, err := fake.New()

		if err != nil {
			panic(fmt.Errorf("creating fake Hetzner client failed: %w", err))
		}

		if setup != nil {
			setup(client)
		}

		return client, nil
	}

	return &InstanceGroup{
		AccessToken: "dummy-token",
		Location:    "dummy-location",
		ServerType:  "cx11",
		Image:       "ubuntu-22.04",
		Name:        "test-group",
	}
}

func TestIncrease(t *testing.T) {
	group := setupFakeClient(t, nil)

	ctx := context.Background()

	var count int
	_, err := group.Init(ctx, hclog.NewNullLogger(), provider.Settings{})
	require.NoError(t, err)
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {
		count++
	}))
	require.Equal(t, 0, group.size)
	require.Equal(t, 0, count)

	num, err := group.Increase(ctx, 2)
	require.Equal(t, 2, num)
	require.NoError(t, err)
	count = 0
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {
		require.Equal(t, provider.StateRunning, state)
		count++
	}))
	require.Equal(t, 2, group.size)
	require.Equal(t, 2, count)
}

func TestDecrease(t *testing.T) {
	group := setupFakeClient(t, func(client *fake.Client) {
		client.Servers = append(
			client.Servers,
			&hcloud.Server{
				ID:     646457,
				Name:   "pre-existing-1",
				Status: hcloud.ServerStatusRunning,
			},
			&hcloud.Server{
				ID:     382443,
				Name:   "pre-existing-2",
				Status: hcloud.ServerStatusRunning,
			})
	})

	ctx := context.Background()

	var count int
	_, err := group.Init(ctx, hclog.NewNullLogger(), provider.Settings{})
	require.NoError(t, err)
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {
		require.Equal(t, provider.StateRunning, state)
		count++
	}))
	require.Equal(t, 2, group.size)
	require.Equal(t, 2, count)

	removed, err := group.Decrease(ctx, []string{"646457"})
	require.Contains(t, removed, "646457")
	require.NoError(t, err)
	count = 0
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {
		count++
	}))
	require.Equal(t, 1, group.size)
}

func TestConnectInfo(t *testing.T) {
	group := setupFakeClient(t, func(client *fake.Client) {
		client.Servers = append(
			client.Servers,
			&hcloud.Server{
				ID:   218452,
				Name: "pre-existing-1",

				Image: &hcloud.Image{
					OSFlavor: "ubuntu",
				},

				Status: hcloud.ServerStatusRunning,

				ServerType: &hcloud.ServerType{
					Name: "cx11",
				},
			})

		// Add private keys for the servers above, so that ConnectInfo will be able to retrieve
		// them.
		sshPrivateKeys["pre-existing-1"] = []byte("dummy-private-key-1")
	})

	ctx := context.Background()

	_, err := group.Init(ctx, hclog.NewNullLogger(), provider.Settings{})
	require.NoError(t, err)
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {}))

	encodedKey := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(fake.Key()),
		},
	)

	tests := []struct {
		config provider.ConnectorConfig
		assert func(t *testing.T, info provider.ConnectInfo, err error)
	}{
		{
			config: provider.ConnectorConfig{
				OS: "linux",
			},
			assert: func(t *testing.T, info provider.ConnectInfo, err error) {
				require.NoError(t, err)
				require.Equal(t, info.Protocol, provider.ProtocolSSH)

				require.Equal(t, info.Key, []byte("dummy-private-key-1"))
			},
		},
		{
			config: provider.ConnectorConfig{
				Protocol: provider.ProtocolSSH,
				Key:      encodedKey,
			},
			assert: func(t *testing.T, info provider.ConnectInfo, err error) {
				require.ErrorContains(t, err, "plugin does not support providing an SSH key in advance")
			},
		},
		{
			config: provider.ConnectorConfig{
				Protocol: provider.ProtocolWinRM,
			},
			assert: func(t *testing.T, info provider.ConnectInfo, err error) {
				require.ErrorContains(t, err, "plugin does not support the WinRM protocol")
			},
		},
		{
			config: provider.ConnectorConfig{
				Protocol: provider.ProtocolWinRM,
				Key:      []byte("invalid key"),
			},
			assert: func(t *testing.T, info provider.ConnectInfo, err error) {
				require.ErrorContains(t, err, "plugin does not support the WinRM protocol")
			},
		},
		{
			config: provider.ConnectorConfig{
				Protocol: provider.ProtocolWinRM,
				Key:      encodedKey,
			},
			assert: func(t *testing.T, info provider.ConnectInfo, err error) {
				require.ErrorContains(t, err, "plugin does not support the WinRM protocol")
			},
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			group.settings.ConnectorConfig = tc.config

			info, err := group.ConnectInfo(ctx, "218452")
			tc.assert(t, info, err)
		})
	}
}
