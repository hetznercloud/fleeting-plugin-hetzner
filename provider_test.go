package hetzner

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	asgtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"gitlab.com/hiboxsystems/fleeting-plugin-hetzner/internal/awsclient"
	"gitlab.com/hiboxsystems/fleeting-plugin-hetzner/internal/awsclient/fake"
)

func setupFakeClient(t *testing.T, setup func(client *fake.Client)) *InstanceGroup {
	t.Helper()

	oldClient := newClient
	t.Cleanup(func() {
		newClient = oldClient
	})

	if region, ok := os.LookupEnv("AWS_REGION"); ok {
		t.Cleanup(func() {
			os.Setenv("AWS_REGION", region)
		})
	} else {
		t.Cleanup(func() {
			os.Unsetenv("AWS_REGION")
		})
	}
	os.Setenv("AWS_REGION", "fake")

	newClient = func(cfg aws.Config) awsclient.Client {
		client := fake.New(cfg)
		client.Name = "test-group"
		if setup != nil {
			setup(client)
		}

		return client
	}

	return &InstanceGroup{
		Name: "test-group",
	}
}

func TestCapacitySync(t *testing.T) {
	setupFakeClient(t, func(client *fake.Client) {
		client.DesiredCapacity = 1
		client.Instances = append(client.Instances, fake.Instance{
			InstanceId: "pre-existing",
			State:      "Running",
		})
	})

	ctx := context.Background()

	group := &InstanceGroup{
		Name: "test-group",
	}

	var buf bytes.Buffer
	logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{Output: &buf})

	// initialize with 1 instance
	_, err := group.Init(ctx, logger, provider.Settings{})
	require.NoError(t, err)
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {}))

	// increase to 5
	num, err := group.Increase(ctx, 5)
	require.Equal(t, 5, num)
	require.NoError(t, err)
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {}))
	require.Equal(t, 6, group.client.(*fake.Client).DesiredCapacity)
	require.Equal(t, 6, group.size)

	// set ASG to have 10 instances manually and update to detect out-of-sync
	group.client.(*fake.Client).DesiredCapacity = 10
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {}))
	require.Contains(t, buf.String(), "[ERROR] out-of-sync capacity: name=test-group region=fake expected=6 actual=10")
	require.Equal(t, 10, group.client.(*fake.Client).DesiredCapacity)
	require.Equal(t, 10, group.size)
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
	require.Equal(t, 2, group.client.(*fake.Client).DesiredCapacity)
	require.Equal(t, 2, group.size)
	require.Equal(t, 2, count)
}

func TestDecrease(t *testing.T) {
	group := setupFakeClient(t, func(client *fake.Client) {
		client.DesiredCapacity = 2
		client.Instances = append(
			client.Instances,
			fake.Instance{
				InstanceId: "pre-existing-1",
				State:      asgtypes.LifecycleStateInService,
			},
			fake.Instance{
				InstanceId: "pre-existing-2",
				State:      asgtypes.LifecycleStateInService,
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

	removed, err := group.Decrease(ctx, []string{"pre-existing-1"})
	require.Contains(t, removed, "pre-existing-1")
	require.NoError(t, err)
	count = 0
	require.NoError(t, group.Update(ctx, func(id string, state provider.State) {
		count++
	}))
	require.Equal(t, 1, group.client.(*fake.Client).DesiredCapacity)
	require.Equal(t, 1, group.size)
}

func TestConnectInfo(t *testing.T) {
	group := setupFakeClient(t, func(client *fake.Client) {
		client.DesiredCapacity = 1
		client.Instances = append(client.Instances, fake.Instance{
			InstanceId: "pre-existing-1",
			State:      asgtypes.LifecycleStateInService,
		})
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
				require.NotEmpty(t, info.Key)
			},
		},
		{
			config: provider.ConnectorConfig{
				Protocol: provider.ProtocolSSH,
				Key:      []byte("invalid-key"),
			},
			assert: func(t *testing.T, info provider.ConnectInfo, err error) {
				require.ErrorContains(t, err, "reading private key: ssh: no key found")
			},
		},
		{
			config: provider.ConnectorConfig{
				Protocol: provider.ProtocolSSH,
				Key:      encodedKey,
			},
			assert: func(t *testing.T, info provider.ConnectInfo, err error) {
				require.NoError(t, err)
				require.Equal(t, info.Protocol, provider.ProtocolSSH)
				require.NotEmpty(t, info.Key)
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

			info, err := group.ConnectInfo(ctx, "pre-existing-1")
			tc.assert(t, info, err)
		})
	}
}
