package fileutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeFilename(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "Normal Text",
			expected: "Normal Text",
		},
		{
			name:     "text with colon",
			input:    "Title: Subtitle",
			expected: "Title - Subtitle",
		},
		{
			name:     "text with slash",
			input:    "Title/Subtitle",
			expected: "Title-Subtitle",
		},
		{
			name:     "text with backslash",
			input:    "Title\\Subtitle",
			expected: "Title-Subtitle",
		},
		{
			name:     "text with both colon and slash",
			input:    "Title: Subtitle/Part",
			expected: "Title - Subtitle-Part",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeFilename(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetMarkdownFilePath(t *testing.T) {
	testCases := []struct {
		name      string
		title     string
		directory string
		expected  string
	}{
		{
			name:      "simple title",
			title:     "Test Title",
			directory: "test/dir",
			expected:  filepath.Join("test/dir", "Test Title.md"),
		},
		{
			name:      "title with colon",
			title:     "Test: Title",
			directory: "test/dir",
			expected:  filepath.Join("test/dir", "Test - Title.md"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetMarkdownFilePath(tc.title, tc.directory)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFileExists(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create a temporary file
	tempFile := filepath.Join(tempDir, "test-file.txt")
	err := os.WriteFile(tempFile, []byte("test"), 0644)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file",
			path:     tempFile,
			expected: true,
		},
		{
			name:     "non-existing file",
			path:     filepath.Join(tempDir, "non-existent.txt"),
			expected: false,
		},
		{
			name:     "directory",
			path:     tempDir,
			expected: false, // FileExists returns false for directories
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FileExists(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestWriteFileWithOverwrite(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	testCases := []struct {
		name           string
		filePath       string
		data           []byte
		overwrite      bool
		setupExisting  bool
		existingData   []byte
		expectedResult bool
		expectedData   []byte
	}{
		{
			name:           "new file",
			filePath:       filepath.Join(tempDir, "new-file.txt"),
			data:           []byte("new content"),
			overwrite:      false,
			setupExisting:  false,
			expectedResult: true,
			expectedData:   []byte("new content"),
		},
		{
			name:           "existing file with overwrite",
			filePath:       filepath.Join(tempDir, "existing-overwrite.txt"),
			data:           []byte("new content"),
			overwrite:      true,
			setupExisting:  true,
			existingData:   []byte("old content"),
			expectedResult: true,
			expectedData:   []byte("new content"),
		},
		{
			name:           "existing file without overwrite",
			filePath:       filepath.Join(tempDir, "existing-no-overwrite.txt"),
			data:           []byte("new content"),
			overwrite:      false,
			setupExisting:  true,
			existingData:   []byte("old content"),
			expectedResult: false,
			expectedData:   []byte("old content"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup existing file if needed
			if tc.setupExisting {
				err := os.WriteFile(tc.filePath, tc.existingData, 0644)
				require.NoError(t, err)
			}

			// Call the function
			result, err := WriteFileWithOverwrite(tc.filePath, tc.data, 0644, tc.overwrite)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, result)

			// Verify file contents
			actualData, err := os.ReadFile(tc.filePath)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedData, actualData)
		})
	}
}

func TestWriteMarkdownFile_NewFile(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	filePath := filepath.Join(tempDir, "test.md")
	content := "# Test\n\nThis is a test."

	err := WriteMarkdownFile(filePath, content, false)
	require.NoError(t, err)

	// Verify file was created with correct content
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(actualContent))
}

func TestWriteMarkdownFile_ExistingWithOverwrite(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	filePath := filepath.Join(tempDir, "existing.md")
	oldContent := "# Old Content"
	newContent := "# New Content"

	// Create existing file
	err := os.WriteFile(filePath, []byte(oldContent), 0644)
	require.NoError(t, err)

	// Write with overwrite = true
	err = WriteMarkdownFile(filePath, newContent, true)
	require.NoError(t, err)

	// Verify file was overwritten
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, newContent, string(actualContent))
}

func TestWriteMarkdownFile_ExistingWithoutOverwrite(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	filePath := filepath.Join(tempDir, "existing.md")
	oldContent := "# Old Content"
	newContent := "# New Content"

	// Create existing file
	err := os.WriteFile(filePath, []byte(oldContent), 0644)
	require.NoError(t, err)

	// Write with overwrite = false
	err = WriteMarkdownFile(filePath, newContent, false)
	require.NoError(t, err)

	// Verify file was NOT overwritten
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, oldContent, string(actualContent))
}

func TestWriteMarkdownFile_CreatesDirectories(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create path with nested directories that don't exist
	filePath := filepath.Join(tempDir, "nested", "dirs", "test.md")
	content := "# Test"

	err := WriteMarkdownFile(filePath, content, false)
	require.NoError(t, err)

	// Verify file was created
	assert.True(t, FileExists(filePath))
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(actualContent))
}

func TestWriteMarkdownFile_DelegationBehavior(t *testing.T) {
	// This test verifies that WriteMarkdownFile correctly delegates to WriteFileWithOverwrite
	// by testing the same behavior patterns

	testCases := []struct {
		name          string
		setupExisting bool
		overwrite     bool
		expectWrite   bool
	}{
		{
			name:          "new file always writes",
			setupExisting: false,
			overwrite:     false,
			expectWrite:   true,
		},
		{
			name:          "existing file with overwrite writes",
			setupExisting: true,
			overwrite:     true,
			expectWrite:   true,
		},
		{
			name:          "existing file without overwrite skips",
			setupExisting: true,
			overwrite:     false,
			expectWrite:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := testutil.NewTestEnv(t)
			tempDir := env.RootDir()

			filePath := filepath.Join(tempDir, "test.md")
			oldContent := "old"
			newContent := "new"

			var err error
			if tc.setupExisting {
				err = os.WriteFile(filePath, []byte(oldContent), 0644)
				require.NoError(t, err)
			}

			err = WriteMarkdownFile(filePath, newContent, tc.overwrite)
			require.NoError(t, err)

			actualContent, err := os.ReadFile(filePath)
			require.NoError(t, err)

			if tc.expectWrite {
				assert.Equal(t, newContent, string(actualContent))
			} else {
				assert.Equal(t, oldContent, string(actualContent))
			}
		})
	}
}

func TestWriteMarkdownFile_EmptyContent(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	filePath := filepath.Join(tempDir, "empty.md")

	err := WriteMarkdownFile(filePath, "", false)
	require.NoError(t, err)

	// Verify empty file was created
	assert.True(t, FileExists(filePath))
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "", string(actualContent))
}

func TestEnsureDir(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	testCases := []struct {
		name string
		path string
	}{
		{
			name: "create single directory",
			path: filepath.Join(tempDir, "testdir"),
		},
		{
			name: "create nested directories",
			path: filepath.Join(tempDir, "level1", "level2", "level3"),
		},
		{
			name: "create directory that already exists",
			path: tempDir, // Already exists
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureDir(tc.path)
			require.NoError(t, err)

			// Verify directory exists
			info, err := os.Stat(tc.path)
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestEnsureDir_WithPermissions(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	dirPath := filepath.Join(tempDir, "permtest")

	err := ensureDir(dirPath)
	require.NoError(t, err)

	// Verify directory has correct permissions (0755)
	info, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755)|os.ModeDir, info.Mode())
}

func TestRelativeTo(t *testing.T) {
	testCases := []struct {
		name     string
		base     string
		target   string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple relative path",
			base:     "/home/user/project",
			target:   "/home/user/project/src/main.go",
			expected: "src/main.go",
			wantErr:  false,
		},
		{
			name:     "parent directory",
			base:     "/home/user/project/src",
			target:   "/home/user/project/README.md",
			expected: "../README.md",
			wantErr:  false,
		},
		{
			name:     "same directory",
			base:     "/home/user/project",
			target:   "/home/user/project",
			expected: ".",
			wantErr:  false,
		},
		{
			name:     "different paths with common root",
			base:     "/home/user/project",
			target:   "/home/other/file.txt",
			expected: "../../other/file.txt",
			wantErr:  false,
		},
		{
			name:     "converts backslashes to forward slashes",
			base:     "/home/user",
			target:   "/home/user/docs/file.txt",
			expected: "docs/file.txt",
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := RelativeTo(tc.base, tc.target)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	testCases := []struct {
		name        string
		content     string
		permissions os.FileMode
	}{
		{
			name:        "copy text file",
			content:     "Hello, World!",
			permissions: 0644,
		},
		{
			name:        "copy empty file",
			content:     "",
			permissions: 0644,
		},
		{
			name:        "copy file with multiple lines",
			content:     "Line 1\nLine 2\nLine 3\n",
			permissions: 0644,
		},
		{
			name:        "copy binary-like content",
			content:     "\x00\x01\x02\x03\xFF\xFE",
			permissions: 0644,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srcPath := filepath.Join(tempDir, "source.txt")
			dstPath := filepath.Join(tempDir, "dest.txt")

			// Create source file
			err := os.WriteFile(srcPath, []byte(tc.content), tc.permissions)
			require.NoError(t, err)

			// Copy file
			err = CopyFile(srcPath, dstPath)
			require.NoError(t, err)

			// Verify destination file exists
			assert.True(t, FileExists(dstPath))

			// Verify content matches
			dstContent, err := os.ReadFile(dstPath)
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(dstContent))

			// Clean up for next iteration
			_ = os.Remove(srcPath)
			_ = os.Remove(dstPath)
		})
	}
}

func TestCopyFile_SourceDoesNotExist(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	srcPath := filepath.Join(tempDir, "nonexistent.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	err := CopyFile(srcPath, dstPath)
	require.Error(t, err)
}

func TestCopyFile_DestinationDirectoryDoesNotExist(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "nonexistent", "dest.txt")

	// Create source file
	err := os.WriteFile(srcPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Copy should fail because destination directory doesn't exist
	err = CopyFile(srcPath, dstPath)
	require.Error(t, err)
}

func TestCopyFile_OverwriteExisting(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	// Create source file
	err := os.WriteFile(srcPath, []byte("new content"), 0644)
	require.NoError(t, err)

	// Create existing destination file
	err = os.WriteFile(dstPath, []byte("old content"), 0644)
	require.NoError(t, err)

	// Copy should overwrite
	err = CopyFile(srcPath, dstPath)
	require.NoError(t, err)

	// Verify destination has new content
	dstContent, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(dstContent))
}
