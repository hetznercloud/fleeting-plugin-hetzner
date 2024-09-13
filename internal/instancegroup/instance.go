package instancegroup

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/actionutil"
)

type Instance struct {
	ID int64

	waitFunc func() error
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
	i.waitFunc = func() error {
		if err := client.Action.WaitFor(ctx, actionutil.AppendNext(result.Action, result.NextActions)...); err != nil {
			return fmt.Errorf("could not create instance: %w", err)
		}

		return nil
	}

	return i, nil
}

func (i *Instance) Wait() error {
	if i.waitFunc != nil {
		defer func() {
			i.waitFunc = nil
		}()

		return i.waitFunc()
	}

	return nil
}

func (i *Instance) Delete(ctx context.Context, client *hcloud.Client) error {
	result, _, err := client.Server.DeleteWithResult(ctx, &hcloud.Server{ID: i.ID})
	if err != nil {
		return fmt.Errorf("could not request instance deletion: %w", err)
	}

	i.waitFunc = func() error {
		if err := client.Action.WaitFor(ctx, result.Action); err != nil {
			return fmt.Errorf("could not delete instance: %w", err)
		}

		return nil
	}

	return nil
}
