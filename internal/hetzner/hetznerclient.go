package hetzner

import (
	"context"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"strconv"
)

// Inspired by
// https://github.com/JonasProgrammer/docker-machine-driver-hetzner/blob/master/driver/driver.go
// (MIT/Expat-licensed)

type Client interface {
	GetServersInGroup(ctx context.Context, name string) ([]*hcloud.Server, error)
	CreateServer(ctx context.Context) (hcloud.ServerCreateResult, error)
	DeleteServer(ctx context.Context, id string) error
	GetServer(ctx context.Context, id string) (*hcloud.Server, error)
}

var _ Client = (*client)(nil)

type client struct {
	AccessToken string
	Version     string
}

type Config struct {
	Location string
}

func New(cfg Config, version string) Client {
	return &client{
		AccessToken: "",
		Version:     version,
	}
}

func (c *client) CreateServer(ctx context.Context) (hcloud.ServerCreateResult, error) {
	serverCreateOpts := hcloud.ServerCreateOpts{}

	serverCreateResponse, _, err := c.getHetznerClient().Server.Create(ctx, serverCreateOpts)

	return serverCreateResponse, err
}

func (c *client) DeleteServer(ctx context.Context, id string) error {
	serverId, err := strconv.Atoi(id)

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

func (c *client) GetServer(ctx context.Context, id string) (*hcloud.Server, error) {
	server, _, err := c.getHetznerClient().Server.Get(ctx, id)

	return server, err
}

func (c *client) GetServersInGroup(ctx context.Context, name string) ([]*hcloud.Server, error) {
	serverListOpts := hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{LabelSelector: name},
	}

	servers, _, err := c.getHetznerClient().Server.List(ctx, serverListOpts)

	return servers, err
}

func (c *client) getHetznerClient() *hcloud.Client {
	return hcloud.NewClient(hcloud.WithToken(c.AccessToken), hcloud.WithApplication("fleeting-plugin-hetzner", c.Version))
}
