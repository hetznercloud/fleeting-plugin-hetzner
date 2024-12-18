package instancegroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

func TestServerHandlerCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "POST", Path: "/servers",
				Status: 201,
				JSON: schema.ServerCreateResponse{
					Server:      schema.Server{ID: 1, Name: "fleeting-a"},
					Action:      schema.Action{ID: 101, Status: "running"},
					NextActions: []schema.Action{{ID: 102, Status: "running"}},
				},
			},
		})

		instance := NewInstance("fleeting-a")
		{
			handler := &BaseHandler{}
			require.NoError(t, handler.Create(ctx, group, instance))
		}

		handler := &ServerHandler{}

		require.NoError(t, handler.Create(ctx, group, instance))

		assert.NotNil(t, instance.ID)
		assert.NotNil(t, instance.waitFn)
	})
}

func TestServerHandlerCleanup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "DELETE", Path: "/servers/1",
				Status: 200,
				JSON: schema.ServerDeleteResponse{
					Action: schema.Action{ID: 101, Status: "running"},
				},
			},
		})

		instance := &Instance{Name: "fleeting-a", ID: 1}

		handler := &ServerHandler{}

		require.NoError(t, handler.Cleanup(ctx, group, instance))

		assert.Equal(t, "fleeting-a", instance.Name)
		assert.Equal(t, int64(1), instance.ID)
		assert.NotNil(t, instance.waitFn)
	})

	t.Run("success not found", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "DELETE", Path: "/servers/1",
				Status: 404,
				JSON: schema.ErrorResponse{
					Error: schema.Error{Code: "not_found"},
				},
			},
		})

		instance := &Instance{Name: "fleeting-a", ID: 1}

		handler := &ServerHandler{}

		require.NoError(t, handler.Cleanup(ctx, group, instance))

		assert.Equal(t, "fleeting-a", instance.Name)
		assert.Equal(t, int64(1), instance.ID)
		assert.Nil(t, instance.waitFn)
	})

	t.Run("passthrough", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{})

		instance := &Instance{Name: "fleeting-a"}

		handler := &ServerHandler{}

		require.NoError(t, handler.Cleanup(ctx, group, instance))

		assert.Equal(t, "fleeting-a", instance.Name)
		assert.Equal(t, int64(0), instance.ID)
		assert.Nil(t, instance.waitFn)
	})
}
