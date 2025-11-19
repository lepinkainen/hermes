// Package testutil provides common test utilities for the hermes project.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnv provides a sandboxed test environment that validates all paths
// stay within a temporary directory. It automatically cleans up when the
// test completes.
type TestEnv struct {
	t       *testing.T
	rootDir string
}

// NewTestEnv creates a new sandboxed test environment.
// The environment is automatically cleaned up when the test completes.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()
	return &TestEnv{
		t:       t,
		rootDir: t.TempDir(),
	}
}

// RootDir returns the root directory of the test environment.
func (e *TestEnv) RootDir() string {
	return e.rootDir
}

// Path returns an absolute path within the test environment.
// It validates that the path does not escape the sandbox.
func (e *TestEnv) Path(elem ...string) string {
	e.t.Helper()

	// Join elements and clean the path
	relPath := filepath.Join(elem...)
	absPath := filepath.Join(e.rootDir, relPath)

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(absPath)

	// Verify the path stays within the sandbox
	if !e.isWithinSandbox(cleanPath) {
		e.t.Fatalf("path %q escapes test sandbox %q", cleanPath, e.rootDir)
	}

	return cleanPath
}

// isWithinSandbox checks if a path is within the test environment root.
func (e *TestEnv) isWithinSandbox(path string) bool {
	// Ensure both paths are clean
	cleanRoot := filepath.Clean(e.rootDir)
	cleanPath := filepath.Clean(path)

	// Check if the path starts with the root directory
	return strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) || cleanPath == cleanRoot
}

// WriteFile writes content to a file within the test environment.
// It creates any necessary parent directories.
func (e *TestEnv) WriteFile(path string, content []byte) {
	e.t.Helper()

	absPath := e.Path(path)

	// Create parent directories
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.t.Fatalf("failed to create directory %q: %v", dir, err)
	}

	// Write the file
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		e.t.Fatalf("failed to write file %q: %v", absPath, err)
	}
}

// WriteFileString writes a string to a file within the test environment.
func (e *TestEnv) WriteFileString(path, content string) {
	e.t.Helper()
	e.WriteFile(path, []byte(content))
}

// ReadFile reads a file from within the test environment.
func (e *TestEnv) ReadFile(path string) []byte {
	e.t.Helper()

	absPath := e.Path(path)

	content, err := os.ReadFile(absPath)
	if err != nil {
		e.t.Fatalf("failed to read file %q: %v", absPath, err)
	}

	return content
}

// ReadFileString reads a file as a string from within the test environment.
func (e *TestEnv) ReadFileString(path string) string {
	e.t.Helper()
	return string(e.ReadFile(path))
}

// MkdirAll creates a directory and all necessary parents within the test environment.
func (e *TestEnv) MkdirAll(path string) {
	e.t.Helper()

	absPath := e.Path(path)

	if err := os.MkdirAll(absPath, 0o755); err != nil {
		e.t.Fatalf("failed to create directory %q: %v", absPath, err)
	}
}

// FileExists checks if a file exists within the test environment.
func (e *TestEnv) FileExists(path string) bool {
	e.t.Helper()

	absPath := e.Path(path)
	_, err := os.Stat(absPath)
	return err == nil
}

// RequireFileExists asserts that a file exists within the test environment.
func (e *TestEnv) RequireFileExists(path string) {
	e.t.Helper()

	if !e.FileExists(path) {
		e.t.Fatalf("expected file %q to exist", e.Path(path))
	}
}

// RequireFileNotExists asserts that a file does not exist within the test environment.
func (e *TestEnv) RequireFileNotExists(path string) {
	e.t.Helper()

	if e.FileExists(path) {
		e.t.Fatalf("expected file %q to not exist", e.Path(path))
	}
}

// Chdir changes the working directory to a path within the test environment.
// It restores the original working directory when the test completes.
func (e *TestEnv) Chdir(path string) {
	e.t.Helper()

	absPath := e.Path(path)

	// Get the current directory
	origDir, err := os.Getwd()
	if err != nil {
		e.t.Fatalf("failed to get current directory: %v", err)
	}

	// Change to the new directory
	if err := os.Chdir(absPath); err != nil {
		e.t.Fatalf("failed to change directory to %q: %v", absPath, err)
	}

	// Restore on cleanup
	e.t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			e.t.Errorf("failed to restore directory to %q: %v", origDir, err)
		}
	})
}

// CopyFile copies a file from src (absolute path) to dst (relative to test env).
func (e *TestEnv) CopyFile(src, dst string) {
	e.t.Helper()

	content, err := os.ReadFile(src)
	if err != nil {
		e.t.Fatalf("failed to read source file %q: %v", src, err)
	}

	e.WriteFile(dst, content)
}

// ListFiles returns a list of files in a directory within the test environment.
func (e *TestEnv) ListFiles(path string) []string {
	e.t.Helper()

	absPath := e.Path(path)

	entries, err := os.ReadDir(absPath)
	if err != nil {
		e.t.Fatalf("failed to read directory %q: %v", absPath, err)
	}

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}

	return files
}

// AssertFileContains checks if a file contains the expected string.
func (e *TestEnv) AssertFileContains(path, expected string) {
	e.t.Helper()

	content := e.ReadFileString(path)
	if !strings.Contains(content, expected) {
		e.t.Errorf("file %q does not contain expected string %q", path, expected)
	}
}

// AssertFileEquals checks if a file exactly matches the expected content.
func (e *TestEnv) AssertFileEquals(path, expected string) {
	e.t.Helper()

	content := e.ReadFileString(path)
	if content != expected {
		e.t.Errorf("file %q content mismatch:\ngot:\n%s\n\nwant:\n%s", path, content, expected)
	}
}

// Symlink creates a symbolic link within the test environment.
func (e *TestEnv) Symlink(target, link string) {
	e.t.Helper()

	linkPath := e.Path(link)

	// Create parent directories for the link
	dir := filepath.Dir(linkPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.t.Fatalf("failed to create directory %q: %v", dir, err)
	}

	if err := os.Symlink(target, linkPath); err != nil {
		e.t.Fatalf("failed to create symlink %q -> %q: %v", linkPath, target, err)
	}
}

// TempFile creates a temporary file within the test environment and returns its path.
func (e *TestEnv) TempFile(pattern string) string {
	e.t.Helper()

	f, err := os.CreateTemp(e.rootDir, pattern)
	if err != nil {
		e.t.Fatalf("failed to create temp file: %v", err)
	}

	name := f.Name()
	if err := f.Close(); err != nil {
		e.t.Fatalf("failed to close temp file: %v", err)
	}

	return name
}

// TempDir creates a temporary directory within the test environment and returns its path.
func (e *TestEnv) TempDir(pattern string) string {
	e.t.Helper()

	dir, err := os.MkdirTemp(e.rootDir, pattern)
	if err != nil {
		e.t.Fatalf("failed to create temp directory: %v", err)
	}

	return dir
}

// Stat returns file info for a path within the test environment.
func (e *TestEnv) Stat(path string) os.FileInfo {
	e.t.Helper()

	absPath := e.Path(path)

	info, err := os.Stat(absPath)
	if err != nil {
		e.t.Fatalf("failed to stat %q: %v", absPath, err)
	}

	return info
}

// Remove removes a file or empty directory within the test environment.
func (e *TestEnv) Remove(path string) {
	e.t.Helper()

	absPath := e.Path(path)

	if err := os.Remove(absPath); err != nil {
		e.t.Fatalf("failed to remove %q: %v", absPath, err)
	}
}

// RemoveAll removes a file or directory and all its contents within the test environment.
func (e *TestEnv) RemoveAll(path string) {
	e.t.Helper()

	absPath := e.Path(path)

	if err := os.RemoveAll(absPath); err != nil {
		e.t.Fatalf("failed to remove all %q: %v", absPath, err)
	}
}

// SetEnv sets an environment variable and restores it when the test completes.
func (e *TestEnv) SetEnv(key, value string) {
	e.t.Helper()

	oldValue, hadValue := os.LookupEnv(key)

	if err := os.Setenv(key, value); err != nil {
		e.t.Fatalf("failed to set environment variable %q: %v", key, err)
	}

	e.t.Cleanup(func() {
		if hadValue {
			_ = os.Setenv(key, oldValue)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

// String returns a string representation of the test environment for debugging.
func (e *TestEnv) String() string {
	return fmt.Sprintf("TestEnv{rootDir: %q}", e.rootDir)
}
