package goodreads

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/lepinkainen/hermes/internal/automation"
	"github.com/stretchr/testify/require"
)

func TestPrepareDownloadDirCreatesTemp(t *testing.T) {
	dir, cleanup, err := automation.PrepareDownloadDir("", "hermes-goodreads-test-*")
	require.NoError(t, err)
	require.DirExists(t, dir)
	require.NotNil(t, cleanup)

	cleanup()

	_, statErr := os.Stat(dir)
	require.Error(t, statErr)
	require.True(t, os.IsNotExist(statErr))
}

func TestMoveDownloadedCSVToCustomDir(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "original.csv")
	require.NoError(t, os.WriteFile(source, []byte("data"), 0o644))

	targetDir := filepath.Join(tempDir, "target")
	targetPath, err := moveDownloadedCSV(source, targetDir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(targetDir, exportFileName), targetPath)
	require.FileExists(t, targetPath)

	_, err = os.Stat(source)
	require.True(t, os.IsNotExist(err))
}

func TestWaitForDownloadFindsExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, exportFileName)
	require.NoError(t, os.WriteFile(target, []byte("ok"), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	runner := &mockCDPRunner{}
	path, err := waitForDownload(ctx, runner, tempDir)
	require.NoError(t, err)
	require.Equal(t, target, path)
}

type mockCDPRunner struct{}

func (m *mockCDPRunner) NewExecAllocator(ctx context.Context, opts ...chromedp.ExecAllocatorOption) (context.Context, context.CancelFunc) {
	return context.Background(), func() {}
}

func (m *mockCDPRunner) NewContext(parent context.Context, opts ...chromedp.ContextOption) (context.Context, context.CancelFunc) {
	return context.Background(), func() {}
}

func (m *mockCDPRunner) Run(ctx context.Context, actions ...chromedp.Action) error {
	return nil
}

func TestFindDownloadedCSVSkipsPartialFiles(t *testing.T) {
	tempDir := t.TempDir()
	partial := filepath.Join(tempDir, exportFileName+".crdownload")
	require.NoError(t, os.WriteFile(partial, []byte("incomplete"), 0o644))

	_, found, err := findDownloadedCSV(tempDir)
	require.NoError(t, err)
	require.False(t, found)
}
