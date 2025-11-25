package instancegroup

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
)

func TestDeprecationHandlerSanity(t *testing.T) {
	t.Run("passthrough", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig
		group := setupInstanceGroup(t, config, []mockutil.Request{})

		handler := &DeprecationHandler{}

		require.NoError(t, handler.Sanity(ctx, group))
	})

	t.Run("deprecation", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultTestConfig
		group := setupInstanceGroup(t, config, []mockutil.Request{})

		serverTypeLocationIndex := slices.IndexFunc(
			group.serverTypes[0].Locations,
			func(o hcloud.ServerTypeLocation) bool { return o.Location.Name == group.location.Name },
		)
		assert.GreaterOrEqual(t, serverTypeLocationIndex, 0)

		group.serverTypes[0].Locations[serverTypeLocationIndex].Deprecation = &hcloud.DeprecationInfo{
			Announced:        time.Now().UTC().AddDate(0, -1, 0),
			UnavailableAfter: time.Now().UTC().AddDate(0, 2, 0),
		}

		group.image.Deprecated = time.Now().UTC().AddDate(0, -1, 0)

		handler := &DeprecationHandler{}

		require.NoError(t, handler.Sanity(ctx, group))
	})
}
