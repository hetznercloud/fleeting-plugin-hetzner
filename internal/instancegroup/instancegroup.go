package instancegroup

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/ippool"
	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/utils"
)

type InstanceGroup interface {
	Init(ctx context.Context) error

	Increase(ctx context.Context, delta int) ([]string, error)
	Decrease(ctx context.Context, iids []string) ([]string, error)

	List(ctx context.Context) ([]*Instance, error)
	Get(ctx context.Context, iid string) (*Instance, error)
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
	ipPool *ippool.IPPool

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

	if g.config.PublicIPPoolEnabled {
		g.ipPool = ippool.New(g.config.Location, g.config.PublicIPPoolSelector)
	}

	return nil
}

func (g *instanceGroup) Increase(ctx context.Context, delta int) ([]string, error) {
	handlers := []CreateHandler{
		&BaseHandler{},   // Configure the instance server create options from the instance group config.
		&IPPoolHandler{}, // Configure the IPs in the instance server create options.
		&ServerHandler{}, // Create a server from the instance server create options.
	}

	// Run all pre increase handlers
	for _, handler := range handlers {
		h, ok := handler.(PreIncreaseHandler)
		if !ok {
			continue
		}

		if err := h.PreIncrease(ctx, g); err != nil {
			return nil, err
		}
	}

	errs := make([]error, 0)

	instances := make([]*Instance, 0, delta)
	failed := make([]*Instance, 0, delta)

	// Create a list of new instances
	for i := 0; i < delta; i++ {
		instances = append(instances, NewInstance(g.name+"-"+utils.GenerateRandomID()))
	}

	// Run all create handlers on each instance
	for _, handler := range handlers {
		{
			succeeded := make([]*Instance, 0, len(instances))
			for _, instance := range instances {
				if err := handler.Create(ctx, g, instance); err != nil {
					errs = append(errs, err)
					failed = append(failed, instance)
				} else {
					succeeded = append(succeeded, instance)
				}
			}
			instances = succeeded
		}

		// Wait for each instance background tasks to complete
		{
			succeeded := make([]*Instance, 0, len(instances))
			for _, instance := range instances {
				if err := instance.wait(); err != nil {
					errs = append(errs, err)
					failed = append(failed, instance)
				} else {
					succeeded = append(succeeded, instance)
				}
			}
			instances = succeeded
		}
	}

	// Cleanup failed instances
	if len(failed) > 0 {
		// During cleanup, the handlers must be run backwards
		slices.Reverse(handlers)

		// Run all cleanup handlers on each failed instance
		for _, handler := range handlers {
			h, ok := handler.(CleanupHandler)
			if !ok {
				continue
			}

			for _, instance := range failed {
				if err := h.Cleanup(ctx, g, instance); err != nil {
					errs = append(errs, err)
				}
			}

			// Wait for each instance background tasks to complete
			for _, instance := range failed {
				if err := instance.wait(); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	// Collect created instances IIDs
	created := make([]string, 0, len(instances))
	for _, instance := range instances {
		created = append(created, instance.IID())
	}

	return created, errors.Join(errs...)
}

func (g *instanceGroup) Decrease(ctx context.Context, iids []string) ([]string, error) {
	handlers := []CleanupHandler{
		&ServerHandler{}, // Delete the server of the instance.
	}

	// Run all pre decrease handlers
	for _, handler := range handlers {
		h, ok := handler.(PreDecreaseHandler)
		if !ok {
			continue
		}

		if err := h.PreDecrease(ctx, g); err != nil {
			return nil, err
		}
	}

	errs := make([]error, 0)

	instances := make([]*Instance, 0, len(iids))

	// Populate a list of instances from their IIDs
	for _, iid := range iids {
		instance, err := InstanceFromIID(iid)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		instances = append(instances, instance)
	}

	// Run all cleanup handlers on each instance
	for _, handler := range handlers {
		{
			succeeded := make([]*Instance, 0, len(instances))
			for _, instance := range instances {
				if err := handler.Cleanup(ctx, g, instance); err != nil {
					errs = append(errs, err)
				} else {
					succeeded = append(succeeded, instance)
				}
			}
			instances = succeeded
		}

		// Wait for each instance background tasks to complete
		{
			succeeded := make([]*Instance, 0, len(instances))
			for _, instance := range instances {
				if err := instance.wait(); err != nil {
					errs = append(errs, err)
				} else {
					succeeded = append(succeeded, instance)
				}
			}
			instances = succeeded
		}
	}

	// Collect deleted instances IIDs
	deleted := make([]string, 0, len(instances))
	for _, instance := range instances {
		deleted = append(deleted, instance.IID())
	}

	return deleted, errors.Join(errs...)
}

func (g *instanceGroup) List(ctx context.Context) ([]*Instance, error) {
	servers, err := g.client.Server.AllWithOpts(ctx,
		hcloud.ServerListOpts{
			ListOpts: hcloud.ListOpts{
				LabelSelector: fmt.Sprintf("instance-group=%s", g.name),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not list instances: %w", err)
	}

	instances := make([]*Instance, 0, len(servers))
	for _, server := range servers {
		instances = append(instances, InstanceFromServer(server))
	}

	return instances, nil
}

func (g *instanceGroup) Get(ctx context.Context, iid string) (*Instance, error) {
	instance, err := InstanceFromIID(iid)
	if err != nil {
		return nil, err
	}

	server, _, err := g.client.Server.GetByID(ctx, instance.ID)
	if err != nil {
		return nil, fmt.Errorf("could not get instance: %w", err)
	}

	return InstanceFromServer(server), nil
}
