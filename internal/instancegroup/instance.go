package instancegroup

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/actionutil"
)

type Instance struct {
	ID int64

	// waitFn is used to postpone long background/remote tasks in between each handlers.
	//
	// This allows to trigger the creation of 3 servers in parallel, and only wait once
	// all "create server" action have been triggered. The execution order changes from
	// [create, wait 1m, create, wait 1m, create wait 1m] which could take ~ 3 minutes,
	// to [create, create, create, wait 1m].
	waitFn func() error
}

func NewInstance(id int64) *Instance {
	return &Instance{
		ID: id,
	}
}

func CreateInstance(ctx context.Context, client *hcloud.Client, opts hcloud.ServerCreateOpts) (*Instance, error) {
	result, _, err := client.Server.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("could not request instance creation: %w", err)
	}

	i := &Instance{}
	i.ID = result.Server.ID
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
