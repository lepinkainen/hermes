package fileutil

import (
	"os"
	"path/filepath"
	"testing"

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
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test-file-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_ = tempFile.Close()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-dir-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file",
			path:     tempFile.Name(),
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
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-write-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

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
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-markdown-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	filePath := filepath.Join(tempDir, "test.md")
	content := "# Test\n\nThis is a test."

	err = WriteMarkdownFile(filePath, content, false)
	require.NoError(t, err)

	// Verify file was created with correct content
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(actualContent))
}

func TestWriteMarkdownFile_ExistingWithOverwrite(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-markdown-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	filePath := filepath.Join(tempDir, "existing.md")
	oldContent := "# Old Content"
	newContent := "# New Content"

	// Create existing file
	err = os.WriteFile(filePath, []byte(oldContent), 0644)
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
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-markdown-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	filePath := filepath.Join(tempDir, "existing.md")
	oldContent := "# Old Content"
	newContent := "# New Content"

	// Create existing file
	err = os.WriteFile(filePath, []byte(oldContent), 0644)
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
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-markdown-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create path with nested directories that don't exist
	filePath := filepath.Join(tempDir, "nested", "dirs", "test.md")
	content := "# Test"

	err = WriteMarkdownFile(filePath, content, false)
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
			tempDir, err := os.MkdirTemp("", "test-delegation-*")
			require.NoError(t, err)
			defer func() { _ = os.RemoveAll(tempDir) }()

			filePath := filepath.Join(tempDir, "test.md")
			oldContent := "old"
			newContent := "new"

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
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-markdown-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	filePath := filepath.Join(tempDir, "empty.md")

	err = WriteMarkdownFile(filePath, "", false)
	require.NoError(t, err)

	// Verify empty file was created
	assert.True(t, FileExists(filePath))
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "", string(actualContent))
}
