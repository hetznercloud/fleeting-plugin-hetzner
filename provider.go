package aws

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"github.com/hashicorp/go-hclog"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

type InstanceGroup struct {
	Profile         string `json:"profile"`
	ConfigFile      string `json:"config_file"`
	CredentialsFile string `json:"credentials_file"`
	Name            string `json:"name"`

	log        hclog.Logger
	autoscaler *autoscaling.Client
	ec2        *ec2.Client
	ec2connect *ec2instanceconnect.Client
	size       int

	settings provider.Settings
}

func (g *InstanceGroup) Init(ctx context.Context, log hclog.Logger, settings provider.Settings) (provider.ProviderInfo, error) {
	var opts []func(*config.LoadOptions) error

	if g.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(g.Profile))
	}
	if g.ConfigFile != "" {
		opts = append(opts, config.WithSharedConfigFiles([]string{g.ConfigFile}))
	}
	if g.CredentialsFile != "" {
		opts = append(opts, config.WithSharedCredentialsFiles([]string{g.CredentialsFile}))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return provider.ProviderInfo{}, fmt.Errorf("creating aws config: %w", err)
	}

	g.autoscaler = autoscaling.NewFromConfig(cfg)
	g.ec2 = ec2.NewFromConfig(cfg)
	g.ec2connect = ec2instanceconnect.NewFromConfig(cfg)

	g.log = log.With("region", cfg.Region, "name", g.Name)
	g.settings = settings

	if err := g.updateCapacity(ctx, true); err != nil {
		return provider.ProviderInfo{}, err
	}

	return provider.ProviderInfo{
		ID:        path.Join("aws", cfg.Region, g.Name),
		MaxSize:   1000,
		Version:   Version.String(),
		BuildInfo: Version.BuildInfo(),
	}, nil
}

func (g *InstanceGroup) Update(ctx context.Context, update func(id string, state provider.State)) error {
	if err := g.updateCapacity(ctx, false); err != nil {
		return err
	}

	var next *string
	for {
		output, err := g.autoscaler.DescribeAutoScalingInstances(ctx, &autoscaling.DescribeAutoScalingInstancesInput{
			MaxRecords: aws.Int32(50),
			NextToken:  next,
		})
		if err != nil {
			return err
		}

		for _, instance := range output.AutoScalingInstances {
			if *instance.AutoScalingGroupName == g.Name {
				state := provider.StateCreating

				// Terminating, Terminating:*, Terminated
				if strings.HasPrefix(*instance.LifecycleState, "Terminat") {
					state = provider.StateDeleting
				}

				if *instance.LifecycleState == "InService" {
					state = provider.StateRunning
				}

				update(*instance.InstanceId, state)
			}
		}

		if output.NextToken == nil {
			break
		}
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (int, error) {
	_, err := g.autoscaler.SetDesiredCapacity(ctx, &autoscaling.SetDesiredCapacityInput{
		AutoScalingGroupName: aws.String(g.Name),
		DesiredCapacity:      aws.Int32(int32(g.size + delta)),
		HonorCooldown:        aws.Bool(false),
	})
	if err != nil {
		return 0, fmt.Errorf("increase instances: %w", err)
	}

	g.size += delta

	return delta, nil
}

func (g *InstanceGroup) Decrease(ctx context.Context, instances []string) ([]string, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	for {
		instance := instances[0]
		instances = instances[1:]

		_, err := g.autoscaler.TerminateInstanceInAutoScalingGroup(ctx, &autoscaling.TerminateInstanceInAutoScalingGroupInput{
			InstanceId:                     aws.String(instance),
			ShouldDecrementDesiredCapacity: aws.Bool(true),
		})
		if err != nil {
			return instances, err
		}
		g.size--

		if len(instances) == 0 {
			break
		}
	}

	return nil, nil
}

func (g *InstanceGroup) updateCapacity(ctx context.Context, initial bool) error {
	desc, err := g.autoscaler.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{g.Name},
	})
	if err != nil {
		return fmt.Errorf("describing autoscaling groups: %w", err)
	}
	if len(desc.AutoScalingGroups) == 0 {
		return fmt.Errorf("autoscaler details not returned")
	}

	capacity := desc.AutoScalingGroups[0].DesiredCapacity

	var size int
	if capacity != nil {
		size = int(*capacity)
	}

	if !initial && size != g.size {
		g.log.Error("out-of-sync capacity", "expected", g.size, "actual", size)
	}
	g.size = size

	return nil
}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, id string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	output, err := g.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	})
	if err != nil {
		return info, fmt.Errorf("fetching instance: %w", err)
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return info, fmt.Errorf("fetching instance: not found")
	}

	instance := output.Reservations[0].Instances[0]

	if info.OS == "" {
		switch {
		case instance.Architecture == types.ArchitectureValuesX8664Mac ||
			instance.Architecture == types.ArchitectureValuesArm64Mac:
			info.OS = "darwin"
		case strings.EqualFold(string(instance.Platform), string(types.PlatformValuesWindows)):
			info.OS = "windows"
		default:
			info.OS = "linux"
		}
	}

	if info.Arch == "" {
		switch instance.Architecture {
		case types.ArchitectureValuesI386:
			info.Arch = "386"
		case types.ArchitectureValuesX8664, types.ArchitectureValuesX8664Mac:
			info.Arch = "amd64"
		case types.ArchitectureValuesArm64, types.ArchitectureValuesArm64Mac:
			info.Arch = "arm64"
		}
	}

	if info.Username == "" {
		info.Username = "ec2-user"
		if info.OS == "windows" {
			info.Username = "Administrator"
		}
	}

	info.InternalAddr = aws.ToString(instance.PrivateIpAddress)
	info.ExternalAddr = aws.ToString(instance.PublicIpAddress)

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
