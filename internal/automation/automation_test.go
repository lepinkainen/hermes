package automation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/lepinkainen/hermes/internal/automation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareDownloadDir(t *testing.T) {
	t.Parallel()

	t.Run("creates temporary directory if path is empty", func(t *testing.T) {
		t.Parallel()
		dir, cleanup, err := automation.PrepareDownloadDir("", "test-prefix-*")
		require.NoError(t, err)
		assert.DirExists(t, dir)
		assert.NotNil(t, cleanup)
		assert.Contains(t, dir, "test-prefix-")

		cleanup()
		_, statErr := os.Stat(dir)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("creates specified directory if path is provided", func(t *testing.T) {
		t.Parallel()
		customDir := filepath.Join(t.TempDir(), "my-custom-dir")
		dir, cleanup, err := automation.PrepareDownloadDir(customDir, "test-prefix-*")
		require.NoError(t, err)
		assert.DirExists(t, dir)
		assert.Nil(t, cleanup)
		assert.Equal(t, customDir, dir)

		// Cleanup customDir manually as cleanup is nil
		require.NoError(t, os.RemoveAll(customDir))
	})

	t.Run("returns error if directory creation fails", func(t *testing.T) {
		t.Parallel()
		// Attempt to create a directory in a non-writable location (e.g., root for most OS)
		// This test might be OS-dependent, skip if running as root or on permissive systems.
		if os.Geteuid() == 0 {
			t.Skip("Skipping test as running with root privileges might allow writing to root dir")
		}
		
		dir, cleanup, err := automation.PrepareDownloadDir("/root/nonexistent-dir", "test-prefix-*")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create download directory")
		assert.Empty(t, dir)
		assert.Nil(t, cleanup)
	})
}

func TestCopyFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")

	testContent := "hello world"
	require.NoError(t, os.WriteFile(srcPath, []byte(testContent), 0644))

	err := automation.CopyFile(srcPath, dstPath)
	require.NoError(t, err)

	assert.FileExists(t, dstPath)
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Test case where source file does not exist
	err = automation.CopyFile(filepath.Join(tempDir, "nonexistent.txt"), dstPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}




func TestConfigureDownloadDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	t.Run("successfully configures download directory", func(t *testing.T) {
		t.Parallel()
		origRun := automation.Run
		defer func() { automation.Run = origRun }()
		automation.Run = func(ctx context.Context, actions ...chromedp.Action) error {
			assert.Len(t, actions, 1) // Expecting one action
			return nil
		}
		err := automation.ConfigureDownloadDirectory(ctx, tempDir)
		require.NoError(t, err)
	})

	t.Run("returns error if chromedp.Run fails", func(t *testing.T) {
		t.Parallel()
		origRun := automation.Run
		defer func() { automation.Run = origRun }()
		automation.Run = func(ctx context.Context, actions ...chromedp.Action) error {
			return assert.AnError
		}
		err := automation.ConfigureDownloadDirectory(ctx, tempDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to configure download directory")
	})
}

func TestWaitForSelector(t *testing.T) {
	t.Parallel()

	testTimeout := 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("finds XPath selector", func(t *testing.T) {
		t.Parallel()
		expectedSelector := "//div[@id='myDiv']"
		origRun := automation.Run
		defer func() { automation.Run = origRun }()
		automation.Run = func(ctx context.Context, actions ...chromedp.Action) error {
			assert.Len(t, actions, 2) // Expect Wait action and Evaluate action
			return nil // Simulate success
		}
		selector, err := automation.WaitForSelector(ctx, []string{expectedSelector}, "test selector", testTimeout)
		require.NoError(t, err)
		assert.Equal(t, expectedSelector, selector)
	})

	t.Run("finds CSS selector", func(t *testing.T) {
		t.Parallel()
		expectedSelector := "#myDiv"
		origRun := automation.Run
		defer func() { automation.Run = origRun }()
		automation.Run = func(ctx context.Context, actions ...chromedp.Action) error {
			assert.Len(t, actions, 2) // Expect Wait action and Evaluate action
			return nil // Simulate success
		}
		selector, err := automation.WaitForSelector(ctx, []string{expectedSelector}, "test selector", testTimeout)
		require.NoError(t, err)
		assert.Equal(t, expectedSelector, selector)
	})

	t.Run("returns error on timeout", func(t *testing.T) {
		t.Parallel()
		origRun := automation.Run
		defer func() { automation.Run = origRun }()
		automation.Run = func(ctx context.Context, actions ...chromedp.Action) error {
			return context.DeadlineExceeded // Simulate timeout
		}
		_, err := automation.WaitForSelector(ctx, []string{"#nonexistent"}, "test selector", testTimeout)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for test selector")
	})

	t.Run("returns error if chromedp.Run fails", func(t *testing.T) {
		t.Parallel()
		origRun := automation.Run
		defer func() { automation.Run = origRun }()
		automation.Run = func(ctx context.Context, actions ...chromedp.Action) error {
			return assert.AnError // Simulate generic error
		}
		_, err := automation.WaitForSelector(ctx, []string{"#someDiv"}, "test selector", testTimeout)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find test selector")
	})
}

// TODO: Test BuildExecAllocatorOptions
