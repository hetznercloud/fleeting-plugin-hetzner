package instancegroup

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/testutils"
)

func mustUnmarshal[T any](t *testing.T, src io.ReadCloser, dest T) {
	body, err := io.ReadAll(src)
	require.NoError(t, err)

	err = json.Unmarshal(body, dest)
	require.NoError(t, err)
}

func setupInstanceGroup(t *testing.T, config Config, requests []mockutil.Request) *instanceGroup {
	t.Helper()

	requests = append(
		[]mockutil.Request{
			testutils.GetLocationHel1Request,
			testutils.GetServerTypeCPX11Request,
			testutils.GetImageDebian12Request,
		},
		requests...,
	)

	server := httptest.NewServer(mockutil.Handler(t, requests))
	client := testutils.MakeTestClient(server.URL)

	group := &instanceGroup{name: "fleeting", config: config, client: client}

	err := group.Init(context.Background())
	require.NoError(t, err)

	return group
}
