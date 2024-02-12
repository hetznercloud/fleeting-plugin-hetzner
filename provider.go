package aws

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	asgtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hashicorp/go-hclog"

	"gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws/internal/awsclient"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

var newClient = awsclient.New

type InstanceGroup struct {
	Profile         string `json:"profile"`
	ConfigFile      string `json:"config_file"`
	CredentialsFile string `json:"credentials_file"`
	Name            string `json:"name"`

	log    hclog.Logger
	client awsclient.Client
	size   int

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

	g.client = newClient(cfg)
	g.log = log.With("region", cfg.Region, "name", g.Name)
	g.settings = settings

	if _, err := g.getInstances(ctx, true); err != nil {
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
	instances, err := g.getInstances(ctx, false)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		state := provider.StateCreating

		switch instance.LifecycleState {
		case asgtypes.LifecycleStateTerminated, asgtypes.LifecycleStateTerminating, asgtypes.LifecycleStateTerminatingProceed, asgtypes.LifecycleStateTerminatingWait:
			state = provider.StateDeleting

		case asgtypes.LifecycleStateInService:
			state = provider.StateRunning
		}

		update(*instance.InstanceId, state)
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (int, error) {
	_, err := g.client.SetDesiredCapacity(ctx, &autoscaling.SetDesiredCapacityInput{
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

	var succeeded []string
	for _, instance := range instances {
		_, err := g.client.TerminateInstanceInAutoScalingGroup(ctx, &autoscaling.TerminateInstanceInAutoScalingGroupInput{
			InstanceId:                     aws.String(instance),
			ShouldDecrementDesiredCapacity: aws.Bool(true),
		})
		if err != nil {
			return succeeded, err
		}
		g.size--
		succeeded = append(succeeded, instance)
	}

	return succeeded, nil
}

func (g *InstanceGroup) getInstances(ctx context.Context, initial bool) ([]asgtypes.Instance, error) {
	desc, err := g.client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{g.Name},
	})
	if err != nil {
		return nil, fmt.Errorf("describing autoscaling groups: %w", err)
	}
	if len(desc.AutoScalingGroups) != 1 {
		return nil, fmt.Errorf("unexpected number of autoscaling groups returned: %v", len(desc.AutoScalingGroups))
	}

	// detect out-of-sync capacity changes
	group := desc.AutoScalingGroups[0]
	capacity := group.DesiredCapacity
	var size int
	if capacity != nil {
		size = int(*capacity)
	}

	if initial {
		if !aws.ToBool(group.NewInstancesProtectedFromScaleIn) {
			g.log.Error("new instances are not protected from scale in and should be")
		}
	}

	if !initial && size != g.size {
		g.log.Error("out-of-sync capacity", "expected", g.size, "actual", size)
	}
	g.size = size

	return group.Instances, nil
}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, id string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	output, err := g.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
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
		err = fmt.Errorf("plugin does not support the WinRM protocol")
	}

	return info, err
}

func (g *InstanceGroup) Shutdown(ctx context.Context) error {
	return nil
}
