package instancegroup

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/actionutil"
)

type Instance struct {
	// Name of the instance, used for the underlying server and other attached resources.
	Name string
	// ID of the instance's underlying server.
	ID int64

	// Server is the instance's underlying server, and must never be partially populated.
	Server *hcloud.Server

	// waitFn is used to postpone long background/remote tasks in between each handlers.
	//
	// This allows to trigger the creation of 3 servers in parallel, and only wait once
	// all "create server" action have been triggered. The execution order changes from
	// [create, wait 1m, create, wait 1m, create wait 1m] which could take ~ 3 minutes,
	// to [create, create, create, wait 1m].
	waitFn func() error
}

func NewInstance(name string) *Instance {
	return &Instance{Name: name}
}

func InstanceFromServer(server *hcloud.Server) *Instance {
	return &Instance{Name: server.Name, ID: server.ID, Server: server}
}

func InstanceFromIID(value string) (*Instance, error) {
	parts := strings.Split(value, ":")

	// Handle iid and extract name and id
	if len(parts) == 2 {
		name := parts[0]

		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse instance id: %w", err)
		}

		return &Instance{Name: name, ID: id}, nil
	}

	return nil, fmt.Errorf("invalid instance id: %s", value)
}

// IID holds to data to identify the instance outside of the instance group.
func (i *Instance) IID() string {
	return fmt.Sprintf("%s:%d", i.Name, i.ID)
}

func CreateInstance(ctx context.Context, client *hcloud.Client, opts hcloud.ServerCreateOpts) (*Instance, error) {
	result, _, err := client.Server.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("could not request instance creation: %w", err)
	}

	i := InstanceFromServer(result.Server)

	i.waitFn = func() error {
		if err := client.Action.WaitFor(ctx, actionutil.AppendNext(result.Action, result.NextActions)...); err != nil {
			return fmt.Errorf("could not create instance: %w", err)
		}

		return nil
	}

	return i, nil
}

func (i *Instance) wait() error {
	if i.waitFn == nil {
		return nil
	}

	defer func() {
		i.waitFn = nil
	}()

	return i.waitFn()
}

func (i *Instance) Delete(ctx context.Context, client *hcloud.Client) error {
	result, _, err := client.Server.DeleteWithResult(ctx, &hcloud.Server{ID: i.ID})
	if err != nil {
		return fmt.Errorf("could not request instance deletion: %w", err)
	}

	i.waitFn = func() error {
		if err := client.Action.WaitFor(ctx, result.Action); err != nil {
			return fmt.Errorf("could not delete instance: %w", err)
		}

		return nil
	}

	return nil
}
