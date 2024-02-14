package hetzner

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"path"
	"strconv"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"gitlab.com/hiboxsystems/fleeting-plugin-hetzner/internal/hetzner"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

var newClient = hetzner.New

type InstanceGroup struct {
	Profile         string `json:"profile"`
	ConfigFile      string `json:"config_file"`
	CredentialsFile string `json:"credentials_file"`

	// Because of limitations in the Hetzner API, instance groups do not formally exist in the
	// Hetzner API. The Name here is mapped to a label which is set on all machines created in this
	// "instance group".
	Name string `json:"name"`

	log    hclog.Logger
	client hetzner.Client
	size   int

	settings provider.Settings
}

func (g *InstanceGroup) Init(ctx context.Context, log hclog.Logger, settings provider.Settings) (provider.ProviderInfo, error) {
	cfg := hetzner.Config{
		// TODO: allow overriding this via env variable
		Location: "hel1",
	}

	g.client = newClient(cfg, Version.String())
	g.log = log.With("location", cfg.Location, "name", g.Name)
	g.settings = settings

	if _, err := g.getServersInGroup(ctx); err != nil {
		return provider.ProviderInfo{}, err
	}

	return provider.ProviderInfo{
		ID:        path.Join("hetzner", cfg.Location, g.Name),
		MaxSize:   1000,
		Version:   Version.String(),
		BuildInfo: Version.BuildInfo(),
	}, nil
}

func (g *InstanceGroup) Update(ctx context.Context, update func(id string, state provider.State)) error {
	instances, err := g.getServersInGroup(ctx)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		state := provider.StateCreating

		switch instance.Status {
		case hcloud.ServerStatusOff, hcloud.ServerStatusStopping, hcloud.ServerStatusDeleting:
			state = provider.StateDeleting

		case hcloud.ServerStatusInitializing, hcloud.ServerStatusRunning, hcloud.ServerStatusStarting:
			state = provider.StateRunning

		// TODO: how about these? What should we map them to in the Fleeting world view?
		// hcloud.ServerStatusMigrating
		// hcloud.ServerStatusRebuilding
		// hcloud.ServerStatusUnknown
		default:
			return fmt.Errorf("unexpected instance status encountered: %v", instance.Status)
		}

		update(strconv.Itoa(instance.ID), state)
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (int, error) {
	for i := 0; i < delta; i++ {
		_, err := g.client.CreateServer(ctx)

		if err != nil {
			return i + 1, fmt.Errorf("error creating server: %w", err)
		}
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
		err := g.client.DeleteServer(ctx, instance)

		if err != nil {
			return succeeded, err
		}

		g.size--

		succeeded = append(succeeded, instance)
	}

	return succeeded, nil
}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, id string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	if info.Protocol == provider.ProtocolWinRM {
		return info, fmt.Errorf("plugin does not support the WinRM protocol")
	}

	if info.Key != nil {
		// TODO: This is probably something we would *want* to support, but we currently don't.
		return info, fmt.Errorf("plugin does not support providing an SSH key in advance")
	}

	server, err := g.client.GetServer(ctx, id)

	if err != nil {
		return info, fmt.Errorf("error getting server: %w", err)
	}

	if server == nil {
		return info, fmt.Errorf("fetching instance %v: not found", id)
	}

	info.OS = server.Image.OSFlavor

	// TODO: get this from server-type API. Here we can continue tomorrow.
	// info.Arch - Hetzner provides a "server type" API (https://docs.hetzner.cloud/#server-types), but regretfully the architecture field contains "

	//instance := output.Reservations[0].Instances[0]
	//
	//if info.OS == "" {
	//	switch {
	//	case instance.Architecture == types.ArchitectureValuesX8664Mac ||
	//		instance.Architecture == types.ArchitectureValuesArm64Mac:
	//		info.OS = "darwin"
	//	case strings.EqualFold(string(instance.Platform), string(types.PlatformValuesWindows)):
	//		info.OS = "windows"
	//	default:
	//		info.OS = "linux"
	//	}
	//}
	//
	//if info.Arch == "" {
	//	switch instance.Architecture {
	//	case types.ArchitectureValuesI386:
	//		info.Arch = "386"
	//	case types.ArchitectureValuesX8664, types.ArchitectureValuesX8664Mac:
	//		info.Arch = "amd64"
	//	case types.ArchitectureValuesArm64, types.ArchitectureValuesArm64Mac:
	//		info.Arch = "arm64"
	//	}
	//}
	//
	//if info.Username == "" {
	//	info.Username = "ec2-user"
	//	if info.OS == "windows" {
	//		info.Username = "Administrator"
	//	}
	//}
	//
	//info.InternalAddr = aws.ToString(instance.PrivateIpAddress)
	//info.ExternalAddr = aws.ToString(instance.PublicIpAddress)
	//
	//if info.UseStaticCredentials {
	//	return info, nil
	//}
	//

	if info.Protocol == "" {
		info.Protocol = provider.ProtocolSSH
	}

	return info, err
}

func (g *InstanceGroup) Shutdown(ctx context.Context) error {
	return nil
}

func (g *InstanceGroup) getServersInGroup(ctx context.Context) ([]*hcloud.Server, error) {
	servers, err := g.client.GetServersInGroup(ctx, g.Name)

	if err != nil {
		return nil, fmt.Errorf("GetServersInGroup: %w", err)
	}

	// Workaround for tests which may have added/removed servers without calling our Increase() or
	// Decrease() methods.
	g.size = len(servers)

	return servers, nil

	// TODO: Remove when we are done

	//desc, err := g.client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
	//	AutoScalingGroupNames: []string{g.Name},
	//})
	//if err != nil {
	//	return nil, fmt.Errorf("describing autoscaling groups: %w", err)
	//}
	//if len(desc.AutoScalingGroups) != 1 {
	//	return nil, fmt.Errorf("unexpected number of autoscaling groups returned: %v", len(desc.AutoScalingGroups))
	//}
	//
	//// detect out-of-sync capacity changes
	//group := desc.AutoScalingGroups[0]
	//capacity := group.DesiredCapacity
	//var size int
	//if capacity != nil {
	//	size = int(*capacity)
	//}
	//
	//if initial {
	//	if !aws.ToBool(group.NewInstancesProtectedFromScaleIn) {
	//		g.log.Error("new instances are not protected from scale in and should be")
	//	}
	//}
	//
	//if !initial && size != g.size {
	//	g.log.Error("out-of-sync capacity", "expected", g.size, "actual", size)
	//}
	//g.size = size
	//
	//return group.Instances, nil
}
