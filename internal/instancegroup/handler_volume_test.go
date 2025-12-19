package instancegroup

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

func TestVolumeHandlerCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig
		config.VolumeSize = 10

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "POST", Path: "/volumes",
				Want: func(t *testing.T, r *http.Request) {
					require.Equal(t, "/volumes", r.RequestURI)

					var payload schema.VolumeCreateRequest
					mustUnmarshal(t, r.Body, &payload)
					require.Equal(t, "fleeting-a", payload.Name)
					require.Equal(t, 10, payload.Size)
					require.Equal(t, &map[string]string{"instance-group": "fleeting"}, payload.Labels)
				},
				Status: 201,
				JSON: schema.VolumeCreateResponse{
					Volume: schema.Volume{ID: 1, Name: "fleeting-a"},
					Action: &schema.Action{ID: 101, Status: "running"},
				},
			},
		})

		instance := NewInstance("fleeting-a")
		{
			handler := &BaseHandler{}
			require.NoError(t, handler.Create(ctx, group, instance))
		}

		handler := &VolumeHandler{}

		require.NoError(t, handler.PreIncrease(ctx, group))
		require.NoError(t, handler.Create(ctx, group, instance))

		assert.NotNil(t, instance.waitFn)

		assert.Equal(t, instance.Name, handler.volumes[instance.Name].Name)

		assert.Len(t, instance.opts.Volumes, 1)
		assert.Equal(t, int64(1), instance.opts.Volumes[0].ID)
	})

	t.Run("disabled", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig
		config.VolumeSize = 0

		group := setupInstanceGroup(t, config, []mockutil.Request{})

		instance := NewInstance("fleeting-a")

		handler := &VolumeHandler{}

		require.NoError(t, handler.PreIncrease(ctx, group))
		require.NoError(t, handler.Create(ctx, group, instance))

		assert.Nil(t, instance.waitFn)
	})
}

func TestVolumeHandlerCleanup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "GET", Path: "/volumes?label_selector=instance-group%3Dfleeting&page=1&per_page=50",
				Status: 200,
				JSON: schema.VolumeListResponse{
					Volumes: []schema.Volume{
						{ID: 1, Name: "fleeting-a"},
						{ID: 2, Name: "fleeting-b"},
					},
				},
			},
			{
				Method: "DELETE", Path: "/volumes/1",
				Status: 204,
			},
		})

		instance := &Instance{Name: "fleeting-a", ID: 1}

		handler := &VolumeHandler{}

		require.NoError(t, handler.PreDecrease(ctx, group))
		require.NoError(t, handler.Cleanup(ctx, group, instance))
		assert.Nil(t, instance.waitFn)
	})

	t.Run("passthrough", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig

		group := setupInstanceGroup(t, config, []mockutil.Request{
			{
				Method: "GET", Path: "/volumes?label_selector=instance-group%3Dfleeting&page=1&per_page=50",
				Status: 200,
				JSON: schema.VolumeListResponse{
					Volumes: []schema.Volume{
						{ID: 2, Name: "fleeting-b"},
					},
				},
			},
		})

		instance := &Instance{Name: "fleeting-a", ID: 1}

		handler := &VolumeHandler{}

		require.NoError(t, handler.PreDecrease(ctx, group))
		require.NoError(t, handler.Cleanup(ctx, group, instance))
	})
}
