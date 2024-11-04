package instancegroup

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/testutils"
)

func makeTestInstanceGroup(t *testing.T, config Config, requests []mockutil.Request) *instanceGroup {
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
