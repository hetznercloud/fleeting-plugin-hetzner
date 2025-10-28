package hetzner

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/integration"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/randutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sshutil"
)

func TestProvisioning(t *testing.T) {
	if os.Getenv("HCLOUD_TOKEN") == "" {
		t.Skip("mandatory environment variable HCLOUD_TOKEN not set")
	}

	ctx := context.Background()

	opts := []hcloud.ClientOption{
		hcloud.WithApplication(Version.Name, Version.String()),
		hcloud.WithToken(os.Getenv("HCLOUD_TOKEN")),
	}

	if endpoint := os.Getenv("HCLOUD_ENDPOINT"); endpoint != "" {
		opts = append(opts, hcloud.WithEndpoint(endpoint))
	}

	client := hcloud.NewClient(opts...)

	pluginBinary := integration.BuildPluginBinary(t, "cmd/fleeting-plugin-hetzner", "fleeting-plugin-hetzner")

	t.Run("generated credentials", func(t *testing.T) {
		t.Parallel()

		name := "fleeting-" + randutil.GenerateID()

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: name,

					Token:    os.Getenv("HCLOUD_TOKEN"),
					Endpoint: os.Getenv("HCLOUD_ENDPOINT"),

					Location:    "hel1",
					ServerTypes: []string{"cx23", "cpx22"},
					Image:       "debian-12",

					Labels: map[string]string{"key": "value"},
				},
				ConnectorConfig: provider.ConnectorConfig{
					Timeout: 10 * time.Minute,
				},
				MaxInstances:    3,
				UseExternalAddr: true,
			},
		)

		ensureNoServers(t, ctx, client, name)
		ensureNoVolumes(t, ctx, client, name)
	})

	t.Run("static credentials", func(t *testing.T) {
		t.Parallel()

		name := "fleeting-" + randutil.GenerateID()

		sshPrivateKey, _, err := sshutil.GenerateKeyPair()
		require.NoError(t, err)

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: name,

					Token:    os.Getenv("HCLOUD_TOKEN"),
					Endpoint: os.Getenv("HCLOUD_ENDPOINT"),

					Location:    "hel1",
					ServerTypes: []string{"cx23", "cpx22"},
					Image:       "debian-12",
				},
				ConnectorConfig: provider.ConnectorConfig{
					Timeout: 10 * time.Minute,

					UseStaticCredentials: true,
					Username:             "root",
					Key:                  sshPrivateKey,
				},
				MaxInstances:    3,
				UseExternalAddr: true,
			},
		)

		ensureNoServers(t, ctx, client, name)
		ensureNoVolumes(t, ctx, client, name)
	})

	t.Run("public ip pool", func(t *testing.T) {
		t.Parallel()

		name := "fleeting-" + randutil.GenerateID()

		sshPrivateKey, _, err := sshutil.GenerateKeyPair()
		require.NoError(t, err)

		primaryIPs := make([]*hcloud.PrimaryIP, 0, 3*2)
		actions := make([]*hcloud.Action, 0, 3*2)
		for ipTypeName, ipType := range map[string]hcloud.PrimaryIPType{
			"ipv4": hcloud.PrimaryIPTypeIPv4,
			"ipv6": hcloud.PrimaryIPTypeIPv6,
		} {
			for i := range 3 {
				result, _, err := client.PrimaryIP.Create(ctx, hcloud.PrimaryIPCreateOpts{
					Name:         fmt.Sprintf("%s-primary-%s-%d", name, ipTypeName, i),
					Type:         ipType,
					AssigneeType: "server",
					Datacenter:   "hel1-dc2",
					Labels:       map[string]string{"pool": name},
				})
				require.NoError(t, err)
				primaryIPs = append(primaryIPs, result.PrimaryIP)
				actions = append(actions, result.Action)
			}
		}
		require.NoError(t, client.Action.WaitFor(ctx, actions...))
		defer func() {
			for _, primaryIP := range primaryIPs {
				client.PrimaryIP.Delete(ctx, primaryIP)
			}
		}()

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: name,

					Token:    os.Getenv("HCLOUD_TOKEN"),
					Endpoint: os.Getenv("HCLOUD_ENDPOINT"),

					Location:    "hel1",
					ServerTypes: []string{"cpx22"},
					Image:       "debian-12",

					PublicIPPoolEnabled:  true,
					PublicIPPoolSelector: fmt.Sprintf("pool=%s", name),
				},
				ConnectorConfig: provider.ConnectorConfig{
					Timeout: 10 * time.Minute,

					UseStaticCredentials: true,
					Username:             "root",
					Key:                  sshPrivateKey,
				},
				MaxInstances:    3,
				UseExternalAddr: true,
			},
		)

		ensureNoServers(t, ctx, client, name)
		ensureNoVolumes(t, ctx, client, name)
	})

	t.Run("ipv6 only", func(t *testing.T) {
		t.Parallel()

		name := "fleeting-" + randutil.GenerateID()

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: name,

					Token:    os.Getenv("HCLOUD_TOKEN"),
					Endpoint: os.Getenv("HCLOUD_ENDPOINT"),

					Location:    "hel1",
					ServerTypes: []string{"cpx22"},
					Image:       "debian-12",

					PublicIPv4Disabled: true,
				},
				ConnectorConfig: provider.ConnectorConfig{
					Timeout: 10 * time.Minute,
				},
				MaxInstances:    3,
				UseExternalAddr: true,
			},
		)

		ensureNoServers(t, ctx, client, name)
		ensureNoVolumes(t, ctx, client, name)
	})

	t.Run("volume", func(t *testing.T) {
		t.Parallel()

		name := "fleeting-" + randutil.GenerateID()

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: name,

					Token:    os.Getenv("HCLOUD_TOKEN"),
					Endpoint: os.Getenv("HCLOUD_ENDPOINT"),

					Location:    "hel1",
					ServerTypes: []string{"cpx22"},
					Image:       "debian-12",
					VolumeSize:  10,
				},
				ConnectorConfig: provider.ConnectorConfig{
					Timeout: 10 * time.Minute,
				},
				MaxInstances:    3,
				UseExternalAddr: true,
			},
		)

		ensureNoServers(t, ctx, client, name)
		ensureNoVolumes(t, ctx, client, name)
	})
}

// Ensure all servers were cleaned.
func ensureNoServers(t *testing.T, ctx context.Context, client *hcloud.Client, name string) { // nolint: revive
	t.Helper()

	result, err := client.Server.AllWithOpts(ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: "instance-group=" + name,
		},
	})
	require.NoError(t, err)
	require.Empty(t, result)
}

// Ensure all volumes were cleaned.
func ensureNoVolumes(t *testing.T, ctx context.Context, client *hcloud.Client, name string) { // nolint: revive
	t.Helper()

	result, err := client.Volume.AllWithOpts(ctx, hcloud.VolumeListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: "instance-group=" + name,
		},
	})
	require.NoError(t, err)
	require.Empty(t, result)
}
