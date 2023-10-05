package aws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/fleeting/fleeting/integration"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func TestProvisioning(t *testing.T) {
	integrationTestTemplateID := os.Getenv("AWS_LAUNCH_TEMPLATE_ID")
	if integrationTestTemplateID == "" {
		t.Skip("no integration test launch template id defined")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	client := autoscaling.NewFromConfig(cfg)
	name := uniqueASGName()

	_, err = client.CreateAutoScalingGroup(ctx, &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:             aws.String(name),
		MinSize:                          aws.Int32(0),
		MaxSize:                          aws.Int32(3),
		DesiredCapacity:                  aws.Int32(0),
		NewInstancesProtectedFromScaleIn: aws.Bool(true),
		LaunchTemplate: &types.LaunchTemplateSpecification{
			LaunchTemplateId: aws.String(integrationTestTemplateID),
		},
	})
	require.NoError(t, err)

	defer func() {
		_, err := client.DeleteAutoScalingGroup(ctx, &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(name),
			ForceDelete:          aws.Bool(true),
		})
		require.NoError(t, err)
	}()

	integration.TestProvisioning(t,
		integration.BuildPluginBinary(t, "cmd/fleeting-plugin-aws", "fleeting-plugin-aws"),
		integration.Config{
			PluginConfig: InstanceGroup{
				Name: name,
			},
			ConnectorConfig: provider.ConnectorConfig{
				Username: os.Getenv("AWS_FLEETING_SSH_USERNAME"),
				Timeout:  10 * time.Minute,
			},
			MaxInstances:    3,
			UseExternalAddr: true,
		},
	)
}

func uniqueASGName() string {
	var buf [8]byte
	io.ReadFull(rand.Reader, buf[:])

	return "asg-fleeting-integration-" + hex.EncodeToString(buf[:])
}
