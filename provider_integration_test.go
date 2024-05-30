package hetzner

import (
	"os"
	"testing"
	"time"

	"gitlab.com/gitlab-org/fleeting/fleeting/integration"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/utils"
)

func TestProvisioning(t *testing.T) {
	if os.Getenv("HCLOUD_TOKEN") == "" {
		t.Skip("mandatory environment variable HCLOUD_TOKEN not set")
	}

	integration.TestProvisioning(t,
		integration.BuildPluginBinary(t, "cmd/fleeting-plugin-hetzner", "fleeting-plugin-hetzner"),
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
}
