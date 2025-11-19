package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GoldenHelper provides utilities for golden file testing.
// It handles comparing generated content with golden files and
// supports updating golden files when the UPDATE_GOLDEN environment
// variable is set to "true".
type GoldenHelper struct {
	t          *testing.T
	goldenDir  string
	updateMode bool
}

// NewGoldenHelper creates a new golden file helper.
// goldenDir is the base directory where golden files are stored.
func NewGoldenHelper(t *testing.T, goldenDir string) *GoldenHelper {
	t.Helper()

	return &GoldenHelper{
		t:          t,
		goldenDir:  goldenDir,
		updateMode: os.Getenv("UPDATE_GOLDEN") == "true",
	}
}

// GoldenPath returns the full path to a golden file.
func (g *GoldenHelper) GoldenPath(name string) string {
	return filepath.Join(g.goldenDir, name)
}

// IsUpdateMode returns true if golden files should be updated.
func (g *GoldenHelper) IsUpdateMode() bool {
	return g.updateMode
}

// AssertGolden compares the actual content with the golden file.
// If UPDATE_GOLDEN is set, it updates the golden file instead.
func (g *GoldenHelper) AssertGolden(name string, actual []byte) {
	g.t.Helper()

	goldenPath := g.GoldenPath(name)

	if g.updateMode {
		// Update the golden file
		err := os.MkdirAll(filepath.Dir(goldenPath), 0o755)
		require.NoError(g.t, err, "failed to create golden file directory")

		err = os.WriteFile(goldenPath, actual, 0o644)
		require.NoError(g.t, err, "failed to update golden file")

		g.t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read the golden file
	golden, err := os.ReadFile(goldenPath)
	require.NoError(g.t, err, "failed to read golden file %s", goldenPath)

	// Compare
	assert.Equal(g.t, string(golden), string(actual),
		"content does not match golden file %s", name)
}

// AssertGoldenString is a convenience method for string content.
func (g *GoldenHelper) AssertGoldenString(name, actual string) {
	g.t.Helper()
	g.AssertGolden(name, []byte(actual))
}

// AssertGoldenFile compares the content of an actual file with a golden file.
func (g *GoldenHelper) AssertGoldenFile(actualPath, goldenName string) {
	g.t.Helper()

	actual, err := os.ReadFile(actualPath)
	require.NoError(g.t, err, "failed to read actual file %s", actualPath)

	g.AssertGolden(goldenName, actual)
}

// MustReadGolden reads a golden file and returns its content.
// It fails the test if the file cannot be read.
func (g *GoldenHelper) MustReadGolden(name string) []byte {
	g.t.Helper()

	goldenPath := g.GoldenPath(name)
	content, err := os.ReadFile(goldenPath)
	require.NoError(g.t, err, "failed to read golden file %s", goldenPath)

	return content
}

// MustReadGoldenString reads a golden file and returns its content as a string.
func (g *GoldenHelper) MustReadGoldenString(name string) string {
	g.t.Helper()
	return string(g.MustReadGolden(name))
}

// Exists checks if a golden file exists.
func (g *GoldenHelper) Exists(name string) bool {
	goldenPath := g.GoldenPath(name)
	_, err := os.Stat(goldenPath)
	return err == nil
}

// AssertGoldenJSON compares JSON content, ignoring formatting differences.
// Both actual and golden are expected to be valid JSON.
func (g *GoldenHelper) AssertGoldenJSON(name string, actual []byte) {
	g.t.Helper()

	goldenPath := g.GoldenPath(name)

	if g.updateMode {
		// Update the golden file
		err := os.MkdirAll(filepath.Dir(goldenPath), 0o755)
		require.NoError(g.t, err, "failed to create golden file directory")

		err = os.WriteFile(goldenPath, actual, 0o644)
		require.NoError(g.t, err, "failed to update golden file")

		g.t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read the golden file
	golden, err := os.ReadFile(goldenPath)
	require.NoError(g.t, err, "failed to read golden file %s", goldenPath)

	// Use JSONEq for comparison which ignores formatting
	assert.JSONEq(g.t, string(golden), string(actual),
		"JSON content does not match golden file %s", name)
}

// CreateGoldenDir creates the golden directory if it doesn't exist.
// This is useful in update mode when starting with no golden files.
func (g *GoldenHelper) CreateGoldenDir() {
	g.t.Helper()

	err := os.MkdirAll(g.goldenDir, 0o755)
	require.NoError(g.t, err, "failed to create golden directory")
}

// GoldenTest runs a subtest with golden file comparison.
// It handles the boilerplate of reading actual output and comparing with golden.
func (g *GoldenHelper) GoldenTest(name string, goldenFile string, generator func() ([]byte, error)) {
	g.t.Helper()

	g.t.Run(name, func(t *testing.T) {
		// Create a new helper for the subtest
		helper := &GoldenHelper{
			t:          t,
			goldenDir:  g.goldenDir,
			updateMode: g.updateMode,
		}

		actual, err := generator()
		require.NoError(t, err, "generator failed")

		helper.AssertGolden(goldenFile, actual)
	})
}
