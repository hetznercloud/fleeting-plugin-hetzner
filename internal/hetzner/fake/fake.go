package fake

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"gitlab.com/hiboxsystems/fleeting-plugin-hetzner/internal/hetzner"
	"strconv"
	"sync"
)

type Client struct {
	Servers []*hcloud.Server
}

var once sync.Once

var rsaPrivateKey *rsa.PrivateKey

func Key() *rsa.PrivateKey {
	once.Do(func() {
		var err error
		rsaPrivateKey, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			panic(err)
		}
	})

	return rsaPrivateKey
}

func New(cfg hetzner.Config) *Client {
	return &Client{}
}

func (c *Client) CreateServer(ctx context.Context) (hcloud.ServerCreateResult, error) {
	c.Servers = append(c.Servers, &hcloud.Server{
		Status: hcloud.ServerStatusRunning,
	})

	return hcloud.ServerCreateResult{}, nil
}

func (c *Client) DeleteServer(ctx context.Context, id string) error {
	// We currently don't have any error handling here. If no matching server could be found, the
	// c.Servers field is simply left intact as-is.
	for i, server := range c.Servers {
		if strconv.Itoa(server.ID) == id {
			c.Servers = append(c.Servers[:i], c.Servers[i+1:]...)
		}
	}

	// Can never fail, in line with comment above
	return nil
}

func (c *Client) GetServer(ctx context.Context, id string) (*hcloud.Server, error) {
	for _, server := range c.Servers {
		if strconv.Itoa(server.ID) == id {
			return server, nil
		}
	}

	return nil, fmt.Errorf("server not found: %v", id)
}

func (c *Client) GetServersInGroup(ctx context.Context, name string) ([]*hcloud.Server, error) {
	// TODO: could implement some form of filtering here instead of just returning all the data
	return c.Servers, nil
}
