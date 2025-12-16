package automation_test

import (
	"context"
	"errors"
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

// MockCDPRunner is a test implementation of CDPRunner
type MockCDPRunner struct {
	RunFunc func(ctx context.Context, actions ...chromedp.Action) error
}

func (m *MockCDPRunner) NewExecAllocator(ctx context.Context, opts ...chromedp.ExecAllocatorOption) (context.Context, context.CancelFunc) {
	return ctx, func() {}
}

func (m *MockCDPRunner) NewContext(parent context.Context, opts ...chromedp.ContextOption) (context.Context, context.CancelFunc) {
	return parent, func() {}
}

func (m *MockCDPRunner) Run(ctx context.Context, actions ...chromedp.Action) error {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, actions...)
	}
	return nil
}

func TestConfigureDownloadDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	t.Run("successfully configures download directory", func(t *testing.T) {
		t.Parallel()
		runner := &MockCDPRunner{
			RunFunc: func(ctx context.Context, actions ...chromedp.Action) error {
				assert.Len(t, actions, 1) // Expecting one action
				return nil
			},
		}
		err := automation.ConfigureDownloadDirectory(ctx, runner, tempDir)
		require.NoError(t, err)
	})

	t.Run("returns error if chromedp.Run fails", func(t *testing.T) {
		t.Parallel()
		runner := &MockCDPRunner{
			RunFunc: func(ctx context.Context, actions ...chromedp.Action) error {
				return assert.AnError
			},
		}
		err := automation.ConfigureDownloadDirectory(ctx, runner, tempDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to configure download directory")
	})
}

func TestWaitForSelector(t *testing.T) {
	t.Skip("Skipping WaitForSelector tests - requires more complex chromedp mocking")
	// TODO: Implement proper mocking for chromedp.Evaluate that can set boolean values
}

// TODO: Test BuildExecAllocatorOptions

func TestPollWithTimeout(t *testing.T) {
	t.Parallel()

	t.Run("returns immediately when condition is met", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		callCount := 0
		checkFunc := func() (string, bool, error) {
			callCount++
			return "result", true, nil
		}

		result, err := automation.PollWithTimeout(ctx, 100*time.Millisecond, 1*time.Second, "test", checkFunc)
		require.NoError(t, err)
		assert.Equal(t, "result", result)
		assert.Equal(t, 1, callCount)
	})

	t.Run("polls until condition is met", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		callCount := 0
		checkFunc := func() (string, bool, error) {
			callCount++
			if callCount >= 3 {
				return "success", true, nil
			}
			return "", false, nil
		}

		result, err := automation.PollWithTimeout(ctx, 10*time.Millisecond, 1*time.Second, "test", checkFunc)
		require.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 3, callCount)
	})

	t.Run("returns error when timeout exceeded", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		checkFunc := func() (string, bool, error) {
			return "", false, nil
		}

		_, err := automation.PollWithTimeout(ctx, 10*time.Millisecond, 50*time.Millisecond, "test operation", checkFunc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for test operation")
	})

	t.Run("returns error when context canceled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		checkFunc := func() (string, bool, error) {
			cancel() // Cancel on first call
			return "", false, nil
		}

		_, err := automation.PollWithTimeout(ctx, 10*time.Millisecond, 1*time.Second, "test", checkFunc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
	})

	t.Run("propagates check function errors", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		expectedErr := errors.New("check failed")
		checkFunc := func() (string, bool, error) {
			return "", false, expectedErr
		}

		_, err := automation.PollWithTimeout(ctx, 10*time.Millisecond, 100*time.Millisecond, "test", checkFunc)
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestWaitForURLChange(t *testing.T) {
	t.Parallel()

	t.Run("detects URL change immediately", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		callCount := 0
		getURL := func() (string, error) {
			callCount++
			return "https://example.com/dashboard", nil
		}

		err := automation.WaitForURLChange(ctx, &MockCDPRunner{}, getURL, []string{"/login", "/signin"}, 1*time.Second)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("polls until URL changes", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		callCount := 0
		getURL := func() (string, error) {
			callCount++
			if callCount < 3 {
				return "https://example.com/login", nil
			}
			return "https://example.com/home", nil
		}

		err := automation.WaitForURLChange(ctx, &MockCDPRunner{}, getURL, []string{"/login"}, 5*time.Second)
		require.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("returns error on timeout", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		getURL := func() (string, error) {
			return "https://example.com/login", nil
		}

		err := automation.WaitForURLChange(ctx, &MockCDPRunner{}, getURL, []string{"/login"}, 50*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("propagates getURL errors", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		expectedErr := errors.New("navigation error")
		getURL := func() (string, error) {
			return "", expectedErr
		}

		err := automation.WaitForURLChange(ctx, &MockCDPRunner{}, getURL, []string{"/login"}, 1*time.Second)
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}
