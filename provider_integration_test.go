package hetzner

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"testing"
	"time"

	"gitlab.com/gitlab-org/fleeting/fleeting/integration"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func TestProvisioning(t *testing.T) {
	if os.Getenv("FLEETING_PLUGIN_HETZNER_TOKEN") == "" {
		t.Skip("mandatory environment variable FLEETING_PLUGIN_HETZNER_TOKEN not set")
	}

	name := uniqueASGName()

	integration.TestProvisioning(t,
		integration.BuildPluginBinary(t, "cmd/fleeting-plugin-hetzner", "fleeting-plugin-hetzner"),
		integration.Config{
			PluginConfig: InstanceGroup{
				AccessToken: os.Getenv("FLEETING_PLUGIN_HETZNER_TOKEN"),

				// Give these plugin config settings reasonable defaults, so the integration test
				// can run with only the token set in the environment.
				Location:   "hel1",
				ServerType: "cx11",
				Image:      "ubuntu-22.04",

				Name: name,
			},
			ConnectorConfig: provider.ConnectorConfig{
				Timeout: 10 * time.Minute,
			},
			MaxInstances:    3,
			UseExternalAddr: true,
		},
	)
}

func uniqueASGName() string {
	var buf [8]byte
	io.ReadFull(rand.Reader, buf[:])

	return "fleeting-integration-" + hex.EncodeToString(buf[:])
}
