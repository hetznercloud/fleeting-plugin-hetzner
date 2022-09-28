package aws

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2instanceconnect"
	"github.com/hashicorp/go-hclog"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

type InstanceGroup struct {
	CredentialsProfile string `json:"credentials_profile"`
	CredentialsFile    string `json:"credentials_file"`
	Name               string `json:"name"`
	Region             string `json:"region"`

	log        hclog.Logger
	autoscaler *autoscaling.AutoScaling
	ec2        *ec2.EC2
	ec2connect *ec2instanceconnect.EC2InstanceConnect
	size       int

	settings provider.Settings
}

func (g *InstanceGroup) Init(ctx context.Context, log hclog.Logger, settings provider.Settings) (provider.ProviderInfo, error) {
	g.log = log.With("region", g.Region, "name", g.Name)
	g.settings = settings

	options := session.Options{
		Profile: g.CredentialsProfile,
	}
	if g.CredentialsFile != "" {
		options.SharedConfigFiles = append(options.SharedConfigFiles, g.CredentialsFile)
	}

	sess, err := session.NewSessionWithOptions(options)
	if err != nil {
		return provider.ProviderInfo{}, fmt.Errorf("creating aws session: %w", err)
	}

	cfg := aws.NewConfig().WithRegion(g.Region)

	g.autoscaler = autoscaling.New(sess, cfg)
	g.ec2 = ec2.New(sess, cfg)
	g.ec2connect = ec2instanceconnect.New(sess, cfg)

	return provider.ProviderInfo{
		ID:      path.Join("aws", g.Region, g.Name),
		MaxSize: 1000,
	}, nil
}

func (g *InstanceGroup) Update(ctx context.Context, update func(id string, state provider.State)) error {
	g.size = 0

	return g.autoscaler.DescribeAutoScalingInstancesPages(&autoscaling.DescribeAutoScalingInstancesInput{}, func(output *autoscaling.DescribeAutoScalingInstancesOutput, b bool) bool {
		for _, instance := range output.AutoScalingInstances {
			if *instance.AutoScalingGroupName == g.Name {
				state := provider.StateCreating

				// Terminating, Terminating:*, Terminated
				if strings.HasPrefix(*instance.LifecycleState, "Terminat") {
					state = provider.StateDeleting
				} else {
					g.size += 1
				}

				if *instance.LifecycleState == "InService" {
					state = provider.StateRunning
				}

				update(*instance.InstanceId, state)
			}
		}

		return !b
	})
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (int, error) {
	_, err := g.autoscaler.SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
		AutoScalingGroupName: aws.String(g.Name),
		DesiredCapacity:      aws.Int64(int64(g.size + delta)),
		HonorCooldown:        aws.Bool(false),
	})
	if err != nil {
		return 0, fmt.Errorf("increase instances: %w", err)
	}

	return delta, nil
}

func (g *InstanceGroup) Decrease(ctx context.Context, instances []string) ([]string, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	for {
		instance := instances[0]
		instances = instances[1:]

		_, err := g.autoscaler.TerminateInstanceInAutoScalingGroup(&autoscaling.TerminateInstanceInAutoScalingGroupInput{
			InstanceId:                     aws.String(instance),
			ShouldDecrementDesiredCapacity: aws.Bool(true),
		})
		if err != nil {
			return instances, err
		}

		if len(instances) == 0 {
			break
		}
	}

	return nil, nil
}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, id string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	output, err := g.ec2.DescribeInstancesWithContext(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{id}),
	})
	if err != nil {
		return info, fmt.Errorf("fetching instance: %w", err)
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return info, fmt.Errorf("fetching instance: not found")
	}

	instance := output.Reservations[0].Instances[0]

	if info.OS == "" {
		switch strings.ToLower(aws.StringValue(instance.Platform)) {
		case "windows":
			info.OS = "windows"
		case "macos":
			info.OS = "darwin"
		default:
			info.OS = "linux"
			switch info.Arch {
			case "arm64_mac", "x86_64_mac":
				info.OS = "darwin"
			}
		}
	}

	if info.Arch == "" {
		info.Arch = strings.ToLower(aws.StringValue(instance.Architecture))
		switch {
		case strings.HasPrefix(info.Arch, "x86_64"): // x86_64, x86_64_mac
			info.Arch = "amd64"
		case strings.HasPrefix(info.Arch, "arm64"): // arm64, arm64_mac
			info.Arch = "arm64"
		}
	}

	if info.Username == "" {
		info.Username = "ec2-user"
	}

	info.InternalAddr = aws.StringValue(instance.PrivateIpAddress)
	info.ExternalAddr = aws.StringValue(instance.PublicIpAddress)

	if info.UseStaticCredentials {
		return info, nil
	}

	if info.Protocol == "" {
		info.Protocol = provider.ProtocolSSH
		if info.OS == "windows" {
			info.Protocol = provider.ProtocolWinRM
		}
	}

	switch info.Protocol {
	case provider.ProtocolSSH:
		err = g.ssh(ctx, &info, instance)

	case provider.ProtocolWinRM:
		err = g.winrm(ctx, &info, instance)
	}

	return info, err
}
