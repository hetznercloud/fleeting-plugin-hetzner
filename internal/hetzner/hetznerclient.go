package hetzner

import (
	"context"
	"fmt"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"strconv"
)

// Inspired by
// https://github.com/JonasProgrammer/docker-machine-driver-hetzner/blob/master/driver/driver.go
// (MIT/Expat-licensed)

type Client interface {
	GetServersInInstanceGroup(ctx context.Context, name string) ([]*hcloud.Server, error)

	CreateServer(ctx context.Context, name string, instanceGroupName string, sshPublicKey string, enablePublicIPv4 bool, enablePublicIPv6 bool, networks []int64, cloudInitUserData string) (hcloud.ServerCreateResult, error)

	DeleteServer(ctx context.Context, id string) error
	DeleteSSHKey(ctx context.Context, id int64) error
	GetNetwork(ctx context.Context, networkName string) (*hcloud.Network, error)
	GetServer(ctx context.Context, id string) (*hcloud.Server, error)
	GetSSHKeysInInstanceGroup(ctx context.Context, name string) ([]*hcloud.SSHKey, error)
	GetSSHKeyByName(ctx context.Context, name string) (*hcloud.SSHKey, error)
}

var _ Client = (*client)(nil)

type client struct {
	Config  Config
	Name    string
	Version string
}

type Config struct {
	// The Hetzner Cloud API token to use when connecting to the Hetzner Cloud API.
	AccessToken string

	// The Hetzner Cloud "Location" to use. See https://docs.hetzner.com/cloud/general/locations/
	// for a list of the locations.
	Location string

	// The Hetzner Cloud "Server Type" to use. See https://docs.hetzner.com/cloud/servers/overview/
	// for the list of available types.
	ServerType string

	// The name of the OS image to use. Run `hcloud image list --type system` to see the list of
	// available images.
	Image string
}

func New(cfg Config, name string, version string) (Client, error) {
	// Note: cfg is presumed to have been validated by the caller before this gets called, but we
	// validate all the fields anyway just for the sake of it. The error messages here deliberately
	// do not include the environment variables, in case they ever get changed in the calling side
	// etc. This is just a "Java programmer writing Go". ;)
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("cfg.AccessToken must be set")
	}

	if cfg.Location == "" {
		return nil, fmt.Errorf("cfg.Location must be set")
	}

	if cfg.ServerType == "" {
		return nil, fmt.Errorf("cfg.ServerType must be set")
	}

	if cfg.Image == "" {
		return nil, fmt.Errorf("cfg.Image must be set")
	}

	return &client{
		Config:  cfg,
		Name:    name,
		Version: version,
	}, nil
}

func (c *client) CreateServer(ctx context.Context, name string, instanceGroupName string, sshPublicKey string, enablePublicIPv4 bool, enablePublicIPv6 bool, networks []int64, cloudInitUserData string) (hcloud.ServerCreateResult, error) {

	hetznerClient := c.getHetznerClient()

	sshKeyCreateOpts := hcloud.SSHKeyCreateOpts{
		// Give the SSH key the same name as the instance, for simplicity
		Name: name,

		PublicKey: sshPublicKey,
		Labels: map[string]string{
			"instance-group": instanceGroupName,

			// If everything goes completely bonkers and the plugin doesn't clean things up as
			// expected, this label can be used to manually find and delete servers and SSH keys
			// created by the plugin (but not deleted on shutdown as expected).
			//
			// We *could* go ahead and perform such cleanup on plugin startup, but is it completely
			// safe? Could there be cases where multiple instances of the plugin is being used
			// simultaneously, for example...? (gitlab.com is probably the most obvious example of
			// when aggressive parallelism can be expected)
			"created-by": c.Name,
		},
	}

	sshKey, _, err := hetznerClient.SSHKey.Create(ctx, sshKeyCreateOpts)

	if err != nil {
		return hcloud.ServerCreateResult{}, fmt.Errorf("error creating SSH key for server %v: %w", name, err)
	}

	var hetznerNetworks []*hcloud.Network

	for _, network := range networks {
		hetznerNetworks = append(hetznerNetworks, &hcloud.Network{ID: network})
	}

	serverCreateOpts := hcloud.ServerCreateOpts{
		Name: name,

		ServerType: &hcloud.ServerType{
			Name: c.Config.ServerType,
		},

		Image: &hcloud.Image{
			Name: c.Config.Image,
		},

		Labels: map[string]string{
			"instance-group": instanceGroupName,
			"created-by":     c.Name,
		},

		Networks: hetznerNetworks,

		PublicNet: &hcloud.ServerCreatePublicNet{
			EnableIPv4: enablePublicIPv4,
			EnableIPv6: enablePublicIPv6,
		},

		SSHKeys: []*hcloud.SSHKey{sshKey},

		UserData: cloudInitUserData,
	}

	serverCreateResult, _, err := hetznerClient.Server.Create(ctx, serverCreateOpts)

	return serverCreateResult, err
}

func (c *client) DeleteServer(ctx context.Context, id string) error {
	serverId, err := strconv.ParseInt(id, 10, 64)

	if err != nil {
		// Should never happen, since we use int IDs internally, but... The `fleeting` interface
		// unfortunately forces us to use a `string` ID, so the conversion needs to happen
		// somewhere; either here inside the Hetzner client or in the calling code.
		return err
	}

	server := hcloud.Server{
		ID: serverId,
	}

	_, _, err = c.getHetznerClient().Server.DeleteWithResult(ctx, &server)

	return err
}

func (c *client) DeleteSSHKey(ctx context.Context, id int64) error {
	sshKey := hcloud.SSHKey{
		ID: id,
	}

	_, err := c.getHetznerClient().SSHKey.Delete(ctx, &sshKey)

	return err
}

func (c *client) GetNetwork(ctx context.Context, networkName string) (*hcloud.Network, error) {
	network, _, err := c.getHetznerClient().Network.GetByName(ctx, networkName)

	return network, err
}

func (c *client) GetServer(ctx context.Context, id string) (*hcloud.Server, error) {
	server, _, err := c.getHetznerClient().Server.Get(ctx, id)

	return server, err
}

func (c *client) GetServersInInstanceGroup(ctx context.Context, name string) ([]*hcloud.Server, error) {
	if name == "" {
		return nil, fmt.Errorf("instance group name was unexpectedly empty")
	}

	serverListOpts := hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{LabelSelector: "instance-group=" + name},
	}

	servers, _, err := c.getHetznerClient().Server.List(ctx, serverListOpts)

	return servers, err
}

func (c *client) GetSSHKeyByName(ctx context.Context, name string) (*hcloud.SSHKey, error) {
	sshKey, _, err := c.getHetznerClient().SSHKey.GetByName(ctx, name)

	return sshKey, err
}

func (c *client) GetSSHKeysInInstanceGroup(ctx context.Context, name string) ([]*hcloud.SSHKey, error) {
	sshKeyListOpts := hcloud.SSHKeyListOpts{
		ListOpts: hcloud.ListOpts{LabelSelector: "instance-group=" + name},
	}

	sshKeys, _, err := c.getHetznerClient().SSHKey.List(ctx, sshKeyListOpts)

	return sshKeys, err
}

func (c *client) getHetznerClient() *hcloud.Client {
	return hcloud.NewClient(hcloud.WithToken(c.Config.AccessToken), hcloud.WithApplication("fleeting-plugin-hetzner", c.Version))
}
