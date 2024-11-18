package hetzner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/netip"
	"path"
	"time"

	"github.com/hashicorp/go-hclog"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sshutil"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/instancegroup"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

type InstanceGroup struct {
	Name string `json:"name"`

	Token    string `json:"token"`
	Endpoint string `json:"endpoint"`

	Location     string `json:"location"`
	ServerType   string `json:"server_type"`
	Image        string `json:"image"`
	UserData     string `json:"user_data"`
	UserDataFile string `json:"user_data_file"`

	VolumeSize int `json:"volume_size"`

	PublicIPv4Disabled   bool   `json:"public_ipv4_disabled"`
	PublicIPv6Disabled   bool   `json:"public_ipv6_disabled"`
	PublicIPPoolEnabled  bool   `json:"public_ip_pool_enabled"`
	PublicIPPoolSelector string `json:"public_ip_pool_selector"`

	PrivateNetworks []string `json:"private_networks"`

	sshKey *hcloud.SSHKey
	labels map[string]string

	log      hclog.Logger
	settings provider.Settings

	size int

	client *hcloud.Client
	group  instancegroup.InstanceGroup
}

func (g *InstanceGroup) Init(ctx context.Context, log hclog.Logger, settings provider.Settings) (info provider.ProviderInfo, err error) {
	g.settings = settings
	g.log = log.With("location", g.Location, "name", g.Name)

	if err = g.validate(); err != nil {
		return
	}

	if err = g.populate(); err != nil {
		return
	}

	// Create client
	clientOptions := []hcloud.ClientOption{
		hcloud.WithApplication(Version.Name, Version.String()),
		hcloud.WithToken(g.Token),
		hcloud.WithHTTPClient(&http.Client{
			Timeout: 15 * time.Second,
		}),
		hcloud.WithPollOpts(hcloud.PollOpts{
			BackoffFunc: hcloud.ExponentialBackoffWithOpts(hcloud.ExponentialBackoffOpts{
				Base:       time.Second,
				Multiplier: 2.0,
				Cap:        5 * time.Second,
			}),
		}),
	}
	if g.Endpoint != "" {
		clientOptions = append(clientOptions, hcloud.WithEndpoint(g.Endpoint))
	}
	g.client = hcloud.NewClient(clientOptions...)

	// Prepare credentials
	if !g.settings.UseStaticCredentials {
		g.log.Info("generating ssh key")
		sshPrivateKey, sshPublicKey, err := sshutil.GenerateKeyPair()
		if err != nil {
			return info, err
		}

		g.settings.Key = sshPrivateKey

		g.sshKey, err = g.UploadSSHPublicKey(ctx, sshPublicKey)
		if err != nil {
			return info, err
		}
	} else if len(g.settings.Key) > 0 {
		g.log.Info("using static ssh key")
		sshPublicKey, err := sshutil.GeneratePublicKey(g.settings.Key)
		if err != nil {
			return info, err
		}

		g.sshKey, err = g.UploadSSHPublicKey(ctx, sshPublicKey)
		if err != nil {
			return info, err
		}
	}

	// Create instance group
	groupConfig := instancegroup.Config{
		Location:             g.Location,
		ServerType:           g.ServerType,
		Image:                g.Image,
		UserData:             g.UserData,
		PublicIPv4Disabled:   g.PublicIPv4Disabled,
		PublicIPv6Disabled:   g.PublicIPv6Disabled,
		PublicIPPoolEnabled:  g.PublicIPPoolEnabled,
		PublicIPPoolSelector: g.PublicIPPoolSelector,
		PrivateNetworks:      g.PrivateNetworks,
		Labels:               g.labels,
		VolumeSize:           g.VolumeSize,
	}

	if g.sshKey != nil {
		groupConfig.SSHKeys = []string{g.sshKey.Name}
	}

	g.group = instancegroup.New(g.client, g.log, g.Name, groupConfig)

	if err = g.group.Init(ctx); err != nil {
		return
	}

	return provider.ProviderInfo{
		ID:        path.Join("hetzner", g.Location, g.ServerType, g.Name),
		MaxSize:   math.MaxInt,
		Version:   Version.String(),
		BuildInfo: Version.BuildInfo(),
	}, nil
}

func (g *InstanceGroup) Update(ctx context.Context, update func(string, provider.State)) error {
	instances, err := g.group.List(ctx)
	if err != nil {
		return err
	}

	g.size = len(instances)

	for _, instance := range instances {
		id := instance.IID()

		var state provider.State

		switch instance.Server.Status {
		case hcloud.ServerStatusStopping, hcloud.ServerStatusDeleting:
			state = provider.StateDeleting

		// Server creation always go through `initializing` and `off`. Since we never
		// shutdown servers, we can assume that "off" is still in the creation phase.
		case hcloud.ServerStatusOff:
			state = provider.StateCreating

		case hcloud.ServerStatusInitializing, hcloud.ServerStatusStarting:
			state = provider.StateCreating

		case hcloud.ServerStatusRunning:
			state = provider.StateRunning

		case hcloud.ServerStatusMigrating, hcloud.ServerStatusRebuilding, hcloud.ServerStatusUnknown:
			g.log.Debug("unhandled instance status", "id", id, "status", instance.Server.Status)

		default:
			g.log.Error("unexpected instance status", "id", id, "status", instance.Server.Status)
		}

		update(id, state)
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (int, error) {
	created, err := g.group.Increase(ctx, delta)

	g.size += len(created)

	if sanityErr := g.group.Sanity(ctx); sanityErr != nil {
		g.log.Error("sanity check failed", "error", sanityErr)
	}

	return len(created), err
}

func (g *InstanceGroup) Decrease(ctx context.Context, iids []string) ([]string, error) {
	if len(iids) == 0 {
		return nil, nil
	}

	deleted, err := g.group.Decrease(ctx, iids)

	g.size -= len(deleted)

	if sanityErr := g.group.Sanity(ctx); sanityErr != nil {
		g.log.Error("sanity check failed", "error", sanityErr)
	}

	return deleted, err
}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, iid string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	instance, err := g.group.Get(ctx, iid)
	if err != nil {
		return info, fmt.Errorf("could not get instance: %w", err)
	}

	info.ID = iid
	info.OS = instance.Server.Image.OSFlavor

	switch instance.Server.ServerType.Architecture {
	case hcloud.ArchitectureX86:
		info.Arch = "amd64"
	case hcloud.ArchitectureARM:
		info.Arch = "arm64"
	default:
		g.log.Warn("unsupported architecture", "architecture", instance.Server.ServerType.Architecture)
	}

	switch {
	case !instance.Server.PublicNet.IPv4.IsUnspecified():
		info.ExternalAddr = instance.Server.PublicNet.IPv4.IP.String()
	case !instance.Server.PublicNet.IPv6.IsUnspecified():
		network, ok := netip.AddrFromSlice(instance.Server.PublicNet.IPv6.IP)
		if ok {
			info.ExternalAddr = network.Next().String()
		} else {
			return info, fmt.Errorf("could not parse server public ipv6: %s", instance.Server.PublicNet.IPv6.IP.String())
		}
	}

	if len(instance.Server.PrivateNet) > 0 {
		info.InternalAddr = instance.Server.PrivateNet[0].IP.String()
	}

	return info, err
}

func (g *InstanceGroup) Shutdown(ctx context.Context) error {
	errs := make([]error, 0)

	if g.sshKey != nil {
		g.log.Debug("deleting ssh key", "id", fmt.Sprint(g.sshKey.ID))
		_, err := g.client.SSHKey.Delete(ctx, g.sshKey)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
