package instancegroup

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/actionutil"
)

// VolumeHandler creates a volume and updates the instance server create options with
// the created volume.
type VolumeHandler struct {
	volumes map[string]*hcloud.Volume
}

var _ PreIncreaseHandler = (*VolumeHandler)(nil)
var _ PreDecreaseHandler = (*VolumeHandler)(nil)
var _ CreateHandler = (*VolumeHandler)(nil)
var _ CleanupHandler = (*VolumeHandler)(nil)

func (h *VolumeHandler) PreIncrease(_ context.Context, _ *instanceGroup) error {
	h.volumes = make(map[string]*hcloud.Volume)

	return nil
}

func (h *VolumeHandler) Create(ctx context.Context, group *instanceGroup, instance *Instance) error {
	if group.config.VolumeSize == 0 {
		return nil
	}

	// Create a volume
	result, _, err := group.client.Volume.Create(ctx, hcloud.VolumeCreateOpts{
		Name:     instance.Name,
		Size:     group.config.VolumeSize,
		Location: group.location,
		Labels:   group.labels,
	})
	if err != nil {
		return fmt.Errorf("could not request volume creation: %w", err)
	}

	// Add volume to server creation opts
	instance.opts.Volumes = append(instance.opts.Volumes, result.Volume)

	// Save volume for potential cleanup
	h.volumes[instance.Name] = result.Volume

	instance.waitFn = func() error {
		// Wait for the volume to be created
		if err := group.client.Action.WaitFor(ctx, actionutil.AppendNext(result.Action, result.NextActions)...); err != nil {
			return fmt.Errorf("could not create volume: %w", err)
		}

		return nil
	}

	return nil
}

func (h *VolumeHandler) PreDecrease(ctx context.Context, group *instanceGroup) error {
	h.volumes = make(map[string]*hcloud.Volume)

	volumes, err := group.client.Volume.AllWithOpts(ctx,
		hcloud.VolumeListOpts{
			ListOpts: hcloud.ListOpts{
				LabelSelector: fmt.Sprintf("instance-group=%s", group.name),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("could not list volumes: %w", err)
	}

	for _, volume := range volumes {
		h.volumes[volume.Name] = volume
	}

	return nil
}

func (h *VolumeHandler) Cleanup(ctx context.Context, group *instanceGroup, instance *Instance) error {
	volume, ok := h.volumes[instance.Name]
	if !ok {
		return nil
	}

	_, err := group.client.Volume.Delete(ctx, volume)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			group.log.Warn("tried to delete a volume that do not exist: %s", instance.Name)
			return nil
		}
		return fmt.Errorf("could not request volume deletion: %w", err)
	}

	return nil
}

func (h *VolumeHandler) Sanity(ctx context.Context, group *instanceGroup) error {
	volumes, err := group.client.Volume.AllWithOpts(ctx,
		hcloud.VolumeListOpts{
			ListOpts: hcloud.ListOpts{
				LabelSelector: fmt.Sprintf("instance-group=%s", group.name),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("could not list volumes: %w", err)
	}

	for _, volume := range volumes {
		if volume.Server != nil {
			continue
		}

		group.log.Warn("deleting dangling volume", "name", volume.Name)
		_, err := group.client.Volume.Delete(ctx, volume)
		if err != nil {
			return fmt.Errorf("could not request volume deletion: %w", err)
		}
	}

	return nil
}
