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
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sshutils"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/utils"
)

func TestProvisioning(t *testing.T) {
	if os.Getenv("HCLOUD_TOKEN") == "" {
		t.Skip("mandatory environment variable HCLOUD_TOKEN not set")
	}

	pluginBinary := integration.BuildPluginBinary(t, "cmd/fleeting-plugin-hetzner", "fleeting-plugin-hetzner")

	t.Run("generated credentials", func(t *testing.T) {
		t.Parallel()

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: "fleeting-" + utils.GenerateRandomID(),

					Token: os.Getenv("HCLOUD_TOKEN"),

					Location:   "hel1",
					ServerType: "cpx11",
					Image:      "ubuntu-24.04",
				},
				ConnectorConfig: provider.ConnectorConfig{
					Timeout: 10 * time.Minute,
				},
				MaxInstances:    3,
				UseExternalAddr: true,
			},
		)
	})

	t.Run("static credentials", func(t *testing.T) {
		t.Parallel()

		sshPrivateKey, _, err := sshutils.GenerateKeyPair()
		require.NoError(t, err)

		integration.TestProvisioning(t,
			pluginBinary,
			integration.Config{
				PluginConfig: InstanceGroup{
					Name: "fleeting-" + utils.GenerateRandomID(),

					Token: os.Getenv("HCLOUD_TOKEN"),

					Location:   "hel1",
					ServerType: "cpx11",
					Image:      "ubuntu-24.04",
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
	})

	t.Run("public ip pool", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		name := "fleeting-" + utils.GenerateRandomID()

		sshPrivateKey, _, err := sshutils.GenerateKeyPair()
		require.NoError(t, err)

		client := hcloud.NewClient(
			hcloud.WithApplication(Version.Name, Version.String()),
			hcloud.WithToken(os.Getenv("HCLOUD_TOKEN")),
		)

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

					Token: os.Getenv("HCLOUD_TOKEN"),

					Location:   "hel1",
					ServerType: "cpx11",
					Image:      "ubuntu-24.04",

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
	})
}
