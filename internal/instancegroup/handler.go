package instancegroup

import (
	"context"
)

type PreIncreaseHandler interface {
	// PreIncrease is run before an increase. Any error during this phase will stop the
	// increase.
	PreIncrease(ctx context.Context, group *instanceGroup) error
}

type PreDecreaseHandler interface {
	// PreDecrease is run before a decrease. Any error during this phase will stop the
	// decrease.
	PreDecrease(ctx context.Context, group *instanceGroup) error
}

type CreateHandler interface {
	// Create is run once per instance during an increase. Any error during this phase
	// will be stored and the instance will be marked as failed and will not be passed
	// to the next handler.
	Create(ctx context.Context, group *instanceGroup, instance *Instance) error
}

type CleanupHandler interface {
	// Cleanup is run once per instance during a decrease and potentially an increase.
	// Any error during this phase will be stored and the instance will be passed to the
	// next handler.
	Cleanup(ctx context.Context, group *instanceGroup, instance *Instance) error
}

type SanityHandler interface {
	// Sanity is run once per sanity check. Any error during this phase will only be
	// logged.
	Sanity(ctx context.Context, group *instanceGroup) error
}
