package instancegroup

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustUnmarshal[T any](t *testing.T, src io.ReadCloser, dest T) {
	body, err := io.ReadAll(src)
	require.NoError(t, err)

	err = json.Unmarshal(body, dest)
	require.NoError(t, err)
}
