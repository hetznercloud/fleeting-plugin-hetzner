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

	// Give these env variables reasonable defaults, so the integration test can run with only the
	// token set in the environment.
	if region, ok := os.LookupEnv("FLEETING_PLUGIN_HETZNER_LOCATION"); ok {
		t.Cleanup(func() {
			os.Setenv("FLEETING_PLUGIN_HETZNER_LOCATION", region)
		})
	} else {
		t.Cleanup(func() {
			os.Unsetenv("FLEETING_PLUGIN_HETZNER_LOCATION")
		})
	}
	os.Setenv("FLEETING_PLUGIN_HETZNER_LOCATION", "hel1")

	if region, ok := os.LookupEnv("FLEETING_PLUGIN_HETZNER_SERVER_TYPE"); ok {
		t.Cleanup(func() {
			os.Setenv("FLEETING_PLUGIN_HETZNER_SERVER_TYPE", region)
		})
	} else {
		t.Cleanup(func() {
			os.Unsetenv("FLEETING_PLUGIN_HETZNER_SERVER_TYPE")
		})
	}
	os.Setenv("FLEETING_PLUGIN_HETZNER_SERVER_TYPE", "cx11")

	if region, ok := os.LookupEnv("FLEETING_PLUGIN_HETZNER_IMAGE"); ok {
		t.Cleanup(func() {
			os.Setenv("FLEETING_PLUGIN_HETZNER_IMAGE", region)
		})
	} else {
		t.Cleanup(func() {
			os.Unsetenv("FLEETING_PLUGIN_HETZNER_IMAGE")
		})
	}
	os.Setenv("FLEETING_PLUGIN_HETZNER_IMAGE", "ubuntu-22.04")

	name := uniqueASGName()

	integration.TestProvisioning(t,
		integration.BuildPluginBinary(t, "cmd/fleeting-plugin-hetzner", "fleeting-plugin-hetzner"),
		integration.Config{
			PluginConfig: InstanceGroup{
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
