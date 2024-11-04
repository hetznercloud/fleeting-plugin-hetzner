package instancegroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
)

func TestBaseHandlerCreate(t *testing.T) {
	ctx := context.Background()
	config := DefaultTestConfig

	group := setupInstanceGroup(t, config, []mockutil.Request{})

	instance := NewInstance("fleeting-a")

	handler := &BaseHandler{}
	err := handler.Create(ctx, group, instance)
	require.NoError(t, err)

	assert.Equal(t, "fleeting-a", instance.Name)
	assert.Equal(t, int64(0), instance.ID)

	assert.NotNil(t, instance.opts)
	assert.Equal(t, "fleeting-a", instance.opts.Name)
}
