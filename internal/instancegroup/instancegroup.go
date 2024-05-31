package instancegroup

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/utils"
)

type InstanceGroup interface {
	Init(ctx context.Context) error

	Increase(ctx context.Context, delta int) ([]int64, error)
	Decrease(ctx context.Context, ids []int64) ([]int64, error)

	List(ctx context.Context) ([]*hcloud.Server, error)
	Get(ctx context.Context, id int64) (*hcloud.Server, error)
}

var _ InstanceGroup = (*instanceGroup)(nil)

func New(client *hcloud.Client, name string, config Config) InstanceGroup {
	return &instanceGroup{
		name:   name,
		config: config,
		client: client,
	}
}

type instanceGroup struct {
	name   string
	config Config

	client *hcloud.Client

	location        *hcloud.Location
	serverType      *hcloud.ServerType
	image           *hcloud.Image
	privateNetworks []*hcloud.Network
	sshKeys         []*hcloud.SSHKey
	labels          map[string]string
}

func (g *instanceGroup) Init(ctx context.Context) (err error) {
	// Location
	g.location, _, err = g.client.Location.Get(ctx, g.config.Location)
	if err != nil {
		return fmt.Errorf("could not get location: %w", err)
	}
	if g.location == nil {
		return fmt.Errorf("location not found: %s", g.config.Location)
	}

	// Server Type
	g.serverType, _, err = g.client.ServerType.Get(ctx, g.config.ServerType)
	if err != nil {
		return fmt.Errorf("could not get server type: %w", err)
	}
	if g.serverType == nil {
		return fmt.Errorf("server type not found: %s", g.config.ServerType)
	}

	// Image
	g.image, _, err = g.client.Image.GetForArchitecture(ctx, g.config.Image, g.serverType.Architecture)
	if err != nil {
		return fmt.Errorf("could not get image: %w", err)
	}
	if g.image == nil {
		return fmt.Errorf("image not found: %s", g.config.Image)
	}

	// Private Networks
	g.privateNetworks = make([]*hcloud.Network, 0, len(g.config.PrivateNetworks))
	for _, networkID := range g.config.PrivateNetworks {
		network, _, err := g.client.Network.Get(ctx, networkID)
		if err != nil {
			return fmt.Errorf("could not get network: %w", err)
		}
		if network == nil {
			return fmt.Errorf("network not found: %s", networkID)
		}

		g.privateNetworks = append(g.privateNetworks, network)
	}

	// SSH Keys
	g.sshKeys = make([]*hcloud.SSHKey, 0, len(g.config.SSHKeys))
	for _, sshKeyID := range g.config.SSHKeys {
		sshKey, _, err := g.client.SSHKey.Get(ctx, sshKeyID)
		if err != nil {
			return fmt.Errorf("could not get ssh key: %w", err)
		}
		if sshKey == nil {
			return fmt.Errorf("ssh key not found: %s", sshKeyID)
		}

		g.sshKeys = append(g.sshKeys, sshKey)
	}

	if g.config.Labels != nil {
		g.labels = maps.Clone(g.config.Labels)
		maps.Copy(g.labels, map[string]string{"instance-group": g.name})
	}

	return nil
}

func (g *instanceGroup) Increase(ctx context.Context, delta int) ([]int64, error) {
	created := make([]int64, 0, delta)
	errs := make([]error, 0, delta)
	results := make([]resourceActions, 0, delta)

	for i := delta; i > 0; i-- {
		opts := hcloud.ServerCreateOpts{}

		opts.Name = g.name + "-" + utils.GenerateRandomID()
		opts.Labels = g.labels

		opts.Location = g.location
		opts.ServerType = g.serverType
		opts.Image = g.image
		opts.PublicNet = &hcloud.ServerCreatePublicNet{
			EnableIPv4: !g.config.PublicIPv4Disabled,
			EnableIPv6: !g.config.PublicIPv6Disabled,
		}
		opts.Networks = g.privateNetworks
		opts.SSHKeys = g.sshKeys

		opts.UserData = g.config.UserData

		result, _, err := g.client.Server.Create(ctx, opts)
		if err != nil {
			errs = append(errs, fmt.Errorf("could not request instance creation: %w", err))
			continue
		}

		results = append(results, newResourceActions(result.Server.ID, AppendNextActions(result.Action, result.NextActions)...))
	}

	for _, result := range results {
		if err := g.client.Action.WaitFor(ctx, result.Actions...); err != nil {
			errs = append(errs, fmt.Errorf("could not create instance: %w", err))
			continue
		}

		created = append(created, result.ID)
	}

	return created, errors.Join(errs...)
}

func (g *instanceGroup) Decrease(ctx context.Context, ids []int64) ([]int64, error) {
	deleted := make([]int64, 0, len(ids))
	errs := make([]error, 0, len(ids))
	results := make([]resourceActions, 0, len(ids))

	for _, id := range ids {
		result, _, err := g.client.Server.DeleteWithResult(ctx, &hcloud.Server{ID: id})
		if err != nil {
			errs = append(errs, fmt.Errorf("could not request instance deletion: %w", err))
			continue
		}

		results = append(results, newResourceActions(id, result.Action))
	}

	for _, result := range results {
		if err := g.client.Action.WaitFor(ctx, result.Actions...); err != nil {
			errs = append(errs, fmt.Errorf("could not delete instance: %w", err))
			continue
		}

		deleted = append(deleted, result.ID)
	}

	return deleted, errors.Join(errs...)
}

func (g *instanceGroup) List(ctx context.Context) ([]*hcloud.Server, error) {
	result, err := g.client.Server.AllWithOpts(ctx,
		hcloud.ServerListOpts{
			ListOpts: hcloud.ListOpts{
				LabelSelector: fmt.Sprintf("instance-group=%s", g.name),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not list instances: %w", err)
	}

	return result, nil
}

func (g *instanceGroup) Get(ctx context.Context, id int64) (*hcloud.Server, error) {
	result, _, err := g.client.Server.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get instance: %w", err)
	}
	return result, nil
}
