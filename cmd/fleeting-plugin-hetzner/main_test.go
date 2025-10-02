package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting"
)

func buildBinary(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	binaryName := filepath.Join(t.TempDir(), filepath.Base(dir))
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", binaryName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	require.NoError(t, cmd.Run())

	return binaryName
}

func TestPluginMain(t *testing.T) {
	runner, err := fleeting.RunPlugin(buildBinary(t), nil)
	require.NoError(t, err)
	runner.Kill()
}
