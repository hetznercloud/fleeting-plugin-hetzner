package hetzner

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/integration"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"

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
}
