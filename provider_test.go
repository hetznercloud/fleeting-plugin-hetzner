package aws

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws/internal/awsclient"
	"gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws/internal/awsclient/fake"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func TestCapacitySync(t *testing.T) {
	oldClient := newClient
	defer func() {
		newClient = oldClient
	}()

	if region, ok := os.LookupEnv("AWS_REGION"); ok {
		defer os.Setenv("AWS_REGION", region)
	} else {
		defer os.Unsetenv("AWS_REGION")
	}
	os.Setenv("AWS_REGION", "fake")

	newClient = func(cfg aws.Config) awsclient.Client {
		client := fake.New(cfg)
		client.Name = "test-group"
		client.DesiredCapacity = 1
		client.Instances = append(client.Instances, fake.Instance{
			InstanceId: "pre-existing",
			State:      "Running",
		})

		return client
	}

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
