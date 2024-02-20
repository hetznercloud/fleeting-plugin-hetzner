package fake

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
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

func New() (*Client, error) {
	return &Client{}, nil
}

func (c *Client) CreateServer(_ context.Context, name string, _ string, _ string) (hcloud.ServerCreateResult, error) {
	c.Servers = append(c.Servers, &hcloud.Server{
		Status: hcloud.ServerStatusRunning,
		Name:   name,
	})

	return hcloud.ServerCreateResult{}, nil
}

func (c *Client) DeleteServer(_ context.Context, id string) error {
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

func (c *Client) DeleteSSHKey(context.Context, int) error {
	// no-op
	return nil
}

func (c *Client) GetServer(_ context.Context, id string) (*hcloud.Server, error) {
	for _, server := range c.Servers {
		if strconv.Itoa(server.ID) == id {
			return server, nil
		}
	}

	return nil, fmt.Errorf("server not found: %v", id)
}

func (c *Client) GetServersInInstanceGroup(_ context.Context, _ string) ([]*hcloud.Server, error) {
	// TODO: could implement some form of filtering here instead of just returning all the data
	return c.Servers, nil
}

func (c *Client) GetSSHKeyByName(_ context.Context, name string) (*hcloud.SSHKey, error) {
	return &hcloud.SSHKey{
		Name:      name,
		PublicKey: "a-dummy-public-ssh-key",
	}, nil
}

func (c *Client) GetSSHKeysInInstanceGroup(ctx context.Context, name string) ([]*hcloud.SSHKey, error) {
	//TODO implement me
	panic("implement me")
}
