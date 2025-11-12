package instancegroup

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/hashicorp/go-hclog"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/randutil"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/ippool"
)

type InstanceGroup interface {
	Init(ctx context.Context) error

	Increase(ctx context.Context, delta int) ([]string, error)
	Decrease(ctx context.Context, iids []string) ([]string, error)

	List(ctx context.Context) ([]*Instance, error)
	Get(ctx context.Context, iid string) (*Instance, error)

	Sanity(ctx context.Context, init bool) error
}

var _ InstanceGroup = (*instanceGroup)(nil)

func New(client *hcloud.Client, log hclog.Logger, name string, config Config) InstanceGroup {
	return &instanceGroup{
		name:   name,
		config: config,
		log:    log,
		client: client,
	}
}

type instanceGroup struct {
	name   string
	config Config

	// TODO: Replace with slog once https://github.com/hashicorp/go-hclog/pull/144 is
	// merged.
	log    hclog.Logger
	client *hcloud.Client
	ipPool *ippool.IPPool

	location                *hcloud.Location
	serverTypes             []*hcloud.ServerType
	serverTypesArchitecture hcloud.Architecture
	image                   *hcloud.Image
	privateNetworks         []*hcloud.Network
	sshKeys                 []*hcloud.SSHKey
	labels                  map[string]string

	randomNameFn func() string
}

func (g *instanceGroup) Init(ctx context.Context) (err error) {
	if g.randomNameFn == nil {
		g.randomNameFn = func() string {
			return g.name + "-" + randutil.GenerateID()
		}
	}

	// Location
	g.location, _, err = g.client.Location.Get(ctx, g.config.Location)
	if err != nil {
		return fmt.Errorf("could not get location: %w", err)
	}
	if g.location == nil {
		return fmt.Errorf("location not found: %s", g.config.Location)
	}

	// Server Types
	for _, serverTypeID := range g.config.ServerTypes {
		serverType, _, err := g.client.ServerType.Get(ctx, serverTypeID)
		if err != nil {
			return fmt.Errorf("could not get server type: %w", err)
		}
		if serverType == nil {
			return fmt.Errorf("server type not found: %s", serverTypeID)
		}

		if g.serverTypesArchitecture == "" {
			g.serverTypesArchitecture = serverType.Architecture
		} else if g.serverTypesArchitecture != serverType.Architecture {
			return fmt.Errorf("unexpected server type architecture found: %s (%s)", serverType.Architecture, serverTypeID)
		}

		g.serverTypes = append(g.serverTypes, serverType)
	}

	// Image
	g.image, _, err = g.client.Image.GetForArchitecture(ctx, g.config.Image, g.serverTypesArchitecture)
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

	g.labels = make(map[string]string, len(g.config.Labels)+1)
	if g.config.Labels != nil {
		maps.Copy(g.labels, g.config.Labels)
	}
	g.labels["instance-group"] = g.name

	if g.config.PublicIPPoolEnabled {
		g.ipPool = ippool.New(g.config.Location, g.config.PublicIPPoolSelector)
	}

	// Run sanity checks before starting.
	return g.Sanity(ctx, true)
}

func (g *instanceGroup) Increase(ctx context.Context, delta int) ([]string, error) {
	handlers := []CreateHandler{
		&BaseHandler{},   // Configure the instance server create options from the instance group config.
		&IPPoolHandler{}, // Configure the IPs in the instance server create options.
		&VolumeHandler{}, // Create and configure a volume in the instance server create options.
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
		instances = append(instances, NewInstance(g.randomNameFn()))
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
		&VolumeHandler{}, // Delete the volume of the instance.
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

func (g *instanceGroup) Sanity(ctx context.Context, init bool) error {
	handlers := []SanityHandler{}

	// Only run volume handler when configured by the user or during init to clean left
	// overs from a previous config.
	if g.config.VolumeSize > 0 || init {
		handlers = append(handlers, &VolumeHandler{}) // Delete dangling volumes.
	}

	// Run all sanity handlers
	for _, h := range handlers {
		if err := h.Sanity(ctx, g); err != nil {
			g.log.With("handler", reflect.TypeOf(h).String()).Error(err.Error())
		}
	}

	return nil
}
