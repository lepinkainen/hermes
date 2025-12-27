package enhance

import (
	"bytes"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestFindMarkdownFiles(t *testing.T) {
	env := testutil.NewTestEnv(t)

	env.WriteFileString("Movie.md", "ok")
	env.WriteFileString("Readme.txt", "ignore")
	env.MkdirAll("sub")
	env.WriteFileString("sub/Show.md", "ok")

	dir := env.RootDir()
	files, err := findMarkdownFiles(dir, false)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "Movie.md")}, files)

	files, err = findMarkdownFiles(dir, true)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		filepath.Join(dir, "Movie.md"),
		filepath.Join(dir, "sub", "Show.md"),
	}, files)
}

func TestFindMarkdownFiles_WithParentheses(t *testing.T) {
	env := testutil.NewTestEnv(t)

	env.WriteFileString("Plain.md", "ok")
	env.WriteFileString("Red Sonja (2025).md", "ok")
	env.WriteFileString("ignore.txt", "nope")
	env.MkdirAll("subdir")
	env.WriteFileString("subdir/Series (Pilot).md", "ok")

	gh := testutil.NewGoldenHelper(t, filepath.Join("testdata", "find_markdown_files"))

	files, err := findMarkdownFiles(env.RootDir(), false)
	require.NoError(t, err)
	gh.AssertGoldenString("non_recursive.txt", strings.Join(relPaths(t, env.RootDir(), files), "\n")+"\n")

	files, err = findMarkdownFiles(env.RootDir(), true)
	require.NoError(t, err)
	gh.AssertGoldenString("recursive.txt", strings.Join(relPaths(t, env.RootDir(), files), "\n")+"\n")
}

func TestEnhanceNotes_ProcessesFilesWithParentheses(t *testing.T) {
	testutil.SetTestConfigWithOptions(t, testutil.WithTMDBAPIKey("test-key"))
	env := testutil.NewTestEnv(t)

	env.WriteFileString("Plain.md", completeNote("Plain", 1001))
	env.WriteFileString("Red Sonja (2025).md", completeNote("Red Sonja", 2002))

	// Create the cover files so NeedsCover() returns false
	env.WriteFileString("attachments/Plain - cover.jpg", "fake cover data")
	env.WriteFileString("attachments/Red Sonja - cover.jpg", "fake cover data")

	var buf bytes.Buffer
	origLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	t.Cleanup(func() {
		slog.SetDefault(origLogger)
	})

	opts := Options{
		InputDir: env.RootDir(),
	}

	err := EnhanceNotes(opts)
	require.NoError(t, err)

	logs := buf.String()
	require.Contains(t, logs, "Skipping file (already has all TMDB data)")
	require.Contains(t, logs, "Plain.md")
	require.Contains(t, logs, "Red Sonja (2025).md")
}

func TestEnhanceNotes_DryRunIncludesParenthesesFiles(t *testing.T) {
	testutil.SetTestConfigWithOptions(t, testutil.WithTMDBAPIKey("test-key"))
	env := testutil.NewTestEnv(t)

	minimal := `---
---

Body`
	env.WriteFileString("Plain.md", minimal)
	env.WriteFileString("Red Sonja (2025).md", minimal)

	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	t.Cleanup(func() { slog.SetDefault(orig) })

	err := EnhanceNotes(Options{
		InputDir:          env.RootDir(),
		DryRun:            true,
		Force:             true,
		TMDBDownloadCover: false,
	})
	require.NoError(t, err)

	logs := buf.String()
	require.Contains(t, logs, "Would enhance")
	require.Contains(t, logs, "Plain.md")
	require.Contains(t, logs, "Red Sonja (2025).md")
}

// TestEnhanceNotes_FilesWithParenthesesFullWorkflow demonstrates the complete lifecycle
// of files with parentheses in their names:
// 1. Discovery via findMarkdownFiles
// 2. Parsing via parseNoteFile
// 3. Title extraction from filename
// 4. Full processing through enhance workflow
// This is a regression test for hermes-zihb.
func TestEnhanceNotes_FilesWithParenthesesFullWorkflow(t *testing.T) {
	testutil.SetTestConfigWithOptions(t, testutil.WithTMDBAPIKey("test-key"))
	env := testutil.NewTestEnv(t)

	// Create files with various parentheses patterns
	testFiles := map[string]string{
		"Movie (2025).md":                     "title: Movie\nyear: 2025",
		"Show (Season 1).md":                  "title: Show\nyear: 2024",
		"Complex (2023) - Special Edition.md": "title: Complex\nyear: 2023",
		"Plain.md":                            "title: Plain\nyear: 2024",
	}

	for filename, content := range testFiles {
		env.WriteFileString(filename, fmt.Sprintf("---\n%s\n---\n\nTest content", content))
	}

	// Step 1: Verify file discovery finds all files including those with parentheses
	files, err := findMarkdownFiles(env.RootDir(), false)
	require.NoError(t, err)
	require.Len(t, files, 4, "Should find all 4 markdown files")

	// Verify specific files with parentheses are in the list
	foundParenthesesFiles := 0
	for _, file := range files {
		if strings.Contains(filepath.Base(file), "(") {
			foundParenthesesFiles++
		}
	}
	require.Equal(t, 3, foundParenthesesFiles, "Should find 3 files with parentheses in name")

	// Step 2: Verify parsing works for files with parentheses
	for _, file := range files {
		note, err := parseNoteFile(file)
		require.NoError(t, err, "Should successfully parse file: %s", filepath.Base(file))
		require.NotEmpty(t, note.Title, "Title should be extracted for file: %s", filepath.Base(file))
	}

	// Step 3: Verify title extraction from filenames with parentheses
	testCases := []struct {
		path          string
		expectedTitle string
	}{
		{filepath.Join(env.RootDir(), "Movie (2025).md"), "Movie (2025)"},
		{filepath.Join(env.RootDir(), "Show (Season 1).md"), "Show (Season 1)"},
		{filepath.Join(env.RootDir(), "Complex (2023) - Special Edition.md"), "Complex (2023) - Special Edition"},
	}

	for _, tc := range testCases {
		title := extractTitleFromPath(tc.path)
		require.Equal(t, tc.expectedTitle, title, "Title extraction should preserve parentheses")
	}

	// Step 4: Verify full enhance workflow with dry-run
	var buf bytes.Buffer
	origLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	t.Cleanup(func() { slog.SetDefault(origLogger) })

	err = EnhanceNotes(Options{
		InputDir:          env.RootDir(),
		DryRun:            true,
		Force:             true,
		TMDBDownloadCover: false,
	})
	require.NoError(t, err, "Enhance should complete successfully")

	// Verify all files including those with parentheses were processed
	logs := buf.String()
	require.Contains(t, logs, "Found markdown files to process")
	require.Contains(t, logs, "count=4")

	// Verify specific files with parentheses appear in logs
	require.Contains(t, logs, "Movie (2025).md", "File with parentheses should be processed")
	require.Contains(t, logs, "Show (Season 1).md", "File with parentheses should be processed")
	require.Contains(t, logs, "Complex (2023) - Special Edition.md", "File with complex parentheses pattern should be processed")
}

func TestUpdateNoteWithTMDBData(t *testing.T) {
	env := testutil.NewTestEnv(t)

	content := `---
title: Heat
tmdb_type: movie
year: 1995
---

Body`
	env.WriteFileString("Heat.md", content)
	path := env.Path("Heat.md")

	note, err := parseNoteFile(path)
	require.NoError(t, err)

	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:          949,
		TMDBType:        "movie",
		RuntimeMins:     170,
		TotalEpisodes:   0,
		GenreTags:       []string{"Action", "Crime"},
		CoverPath:       "attachments/Heat - cover.jpg",
		ContentMarkdown: "## Overview\n\nDetailed plot.",
	}

	err = updateNoteWithTMDBData(path, note, tmdbData, true)
	require.NoError(t, err)

	body := env.ReadFileString("Heat.md")
	require.Contains(t, body, "tmdb_id: 949")
	require.Contains(t, body, "runtime: 170")
	require.Contains(t, body, "tags:")
	require.Contains(t, body, "<!-- TMDB_DATA_START -->")
	require.Contains(t, body, "## Overview")
}

// TestUpdateNoteWithTMDBData_NoTypeField is a regression test to ensure notes
// without a type field can still be processed. The type is detected from TMDB
// search results, so filtering based on missing type would break anime and
// other content that doesn't have a pre-set type.
func TestUpdateNoteWithTMDBData_NoTypeField(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Note: intentionally NO type field - TMDB will detect it as TV
	content := `---
title: Cowboy Bebop
year: 1998
---

An anime series.`
	env.WriteFileString("Cowboy Bebop.md", content)
	path := env.Path("Cowboy Bebop.md")

	note, err := parseNoteFile(path)
	require.NoError(t, err)
	require.Empty(t, note.Type, "note should have no type field initially")

	// Simulate TMDB returning TV show data
	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:          30991,
		TMDBType:        "tv",
		TotalEpisodes:   26,
		GenreTags:       []string{"Animation", "Action", "Sci-Fi"},
		ContentMarkdown: "## Overview\n\nSpace bounty hunters.",
	}

	// This must succeed - notes without type should still be processable
	err = updateNoteWithTMDBData(path, note, tmdbData, true)
	require.NoError(t, err)

	body := env.ReadFileString("Cowboy Bebop.md")
	require.Contains(t, body, "tmdb_id: 30991")
	require.Contains(t, body, "tmdb_type: tv")
	require.Contains(t, body, "total_episodes: 26")
}

func TestGenerateLetterboxdSearchURL(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "simple title",
			title:    "Heat",
			expected: "https://letterboxd.com/search/Heat/",
		},
		{
			name:     "title with spaces",
			title:    "The Dark Knight",
			expected: "https://letterboxd.com/search/The%20Dark%20Knight/",
		},
		{
			name:     "title with special characters",
			title:    "The Lord of the Rings: The Fellowship of the Ring",
			expected: "https://letterboxd.com/search/The%20Lord%20of%20the%20Rings:%20The%20Fellowship%20of%20the%20Ring/",
		},
		{
			name:     "title with ampersand",
			title:    "Bonnie & Clyde",
			expected: "https://letterboxd.com/search/Bonnie%20&%20Clyde/",
		},
		{
			name:     "title with parentheses",
			title:    "Red Sonja (2025)",
			expected: "https://letterboxd.com/search/Red%20Sonja%20%282025%29/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateLetterboxdSearchURL(tt.title)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveLetterboxdURI(t *testing.T) {
	testutil.SetTestConfig(t)

	tests := []struct {
		name         string
		note         *Note
		storedType   string
		expectedType string
		setup        func(t *testing.T)
		cleanup      func(t *testing.T)
		wantURI      string
		wantContains string
	}{
		{
			name: "tier 1: URI from frontmatter",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("letterboxd_uri", "https://letterboxd.com/film/heat-1995/")
				return &Note{
					Title:       "Heat",
					Frontmatter: fm,
				}
			}(),
			storedType:   "",
			expectedType: "",
			wantURI:      "https://letterboxd.com/film/heat-1995/",
		},
		{
			name: "tier 3: generate search URL when no TMDB ID",
			note: &Note{
				Title:       "The Dark Knight",
				TMDBID:      0,
				Frontmatter: obsidian.NewFrontmatter(),
			},
			storedType:   "",
			expectedType: "",
			wantContains: "https://letterboxd.com/search/The%20Dark%20Knight/",
		},
		{
			name: "empty title returns empty string",
			note: &Note{
				Title:       "",
				TMDBID:      0,
				Frontmatter: obsidian.NewFrontmatter(),
			},
			storedType:   "",
			expectedType: "",
			wantURI:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			if tt.cleanup != nil {
				t.Cleanup(func() { tt.cleanup(t) })
			}

			result := resolveLetterboxdURI(tt.note, tt.storedType, tt.expectedType)

			if tt.wantURI != "" {
				require.Equal(t, tt.wantURI, result)
			}
			if tt.wantContains != "" {
				require.Contains(t, result, tt.wantContains)
			}
		})
	}
}

func TestEnhanceCmd_Run_Success(t *testing.T) {
	testutil.SetTestConfigWithOptions(t, testutil.WithTMDBAPIKey("test-key"))
	env := testutil.NewTestEnv(t)

	// Create test markdown files
	env.WriteFileString("Movie1.md", `---
title: Movie 1
---
Body`)
	env.WriteFileString("Movie2.md", `---
title: Movie 2
---
Body`)

	mockCalled := false
	mockFunc := func(opts Options) error {
		mockCalled = true
		assert.Equal(t, env.RootDir(), opts.InputDir)
		assert.False(t, opts.Recursive)
		assert.False(t, opts.DryRun)
		assert.True(t, opts.TMDBDownloadCover)
		assert.True(t, opts.TMDBInteractive)
		return nil
	}

	origFunc := EnhanceNotesFunc
	EnhanceNotesFunc = mockFunc
	defer func() { EnhanceNotesFunc = origFunc }()

	cmd := EnhanceCmd{
		InputDirs: []string{env.RootDir()},
	}

	err := cmd.Run()
	require.NoError(t, err)
	assert.True(t, mockCalled, "EnhanceNotesFunc should have been called")
}

func TestEnhanceCmd_Run_MultipleDirectories(t *testing.T) {
	testutil.SetTestConfigWithOptions(t, testutil.WithTMDBAPIKey("test-key"))
	env := testutil.NewTestEnv(t)

	// Create subdirectories
	env.MkdirAll("dir1")
	env.MkdirAll("dir2")

	callCount := 0
	var calledDirs []string

	mockFunc := func(opts Options) error {
		callCount++
		calledDirs = append(calledDirs, opts.InputDir)
		return nil
	}

	origFunc := EnhanceNotesFunc
	EnhanceNotesFunc = mockFunc
	defer func() { EnhanceNotesFunc = origFunc }()

	cmd := EnhanceCmd{
		InputDirs: []string{
			filepath.Join(env.RootDir(), "dir1"),
			filepath.Join(env.RootDir(), "dir2"),
		},
	}

	err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "EnhanceNotesFunc should be called twice")
	assert.Contains(t, calledDirs[0], "dir1")
	assert.Contains(t, calledDirs[1], "dir2")
}

func TestEnhanceCmd_Run_WithOptions(t *testing.T) {
	testutil.SetTestConfigWithOptions(t, testutil.WithTMDBAPIKey("test-key"))
	env := testutil.NewTestEnv(t)

	tests := []struct {
		name            string
		cmd             EnhanceCmd
		wantRecursive   bool
		wantDryRun      bool
		wantOverwrite   bool
		wantForce       bool
		wantInteractive bool
	}{
		{
			name: "with recursive flag",
			cmd: EnhanceCmd{
				InputDirs: []string{env.RootDir()},
				Recursive: true,
			},
			wantRecursive:   true,
			wantInteractive: true,
		},
		{
			name: "with dry-run flag",
			cmd: EnhanceCmd{
				InputDirs: []string{env.RootDir()},
				DryRun:    true,
			},
			wantDryRun:      true,
			wantInteractive: true,
		},
		{
			name: "with regenerate data flag",
			cmd: EnhanceCmd{
				InputDirs:      []string{env.RootDir()},
				RegenerateData: true,
			},
			wantOverwrite:   true,
			wantInteractive: true,
		},
		{
			name: "with force flag",
			cmd: EnhanceCmd{
				InputDirs: []string{env.RootDir()},
				Force:     true,
			},
			wantForce:       true,
			wantInteractive: true,
		},
		{
			name: "with non-interactive flag",
			cmd: EnhanceCmd{
				InputDirs:         []string{env.RootDir()},
				TMDBNoInteractive: true,
			},
			wantInteractive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCalled := false
			mockFunc := func(opts Options) error {
				mockCalled = true
				assert.Equal(t, tt.wantRecursive, opts.Recursive, "Recursive flag")
				assert.Equal(t, tt.wantDryRun, opts.DryRun, "DryRun flag")
				assert.Equal(t, tt.wantOverwrite, opts.RegenerateData, "RegenerateData flag")
				assert.Equal(t, tt.wantForce, opts.Force, "Force flag")
				assert.Equal(t, tt.wantInteractive, opts.TMDBInteractive, "TMDBInteractive flag")
				return nil
			}

			origFunc := EnhanceNotesFunc
			EnhanceNotesFunc = mockFunc
			defer func() { EnhanceNotesFunc = origFunc }()

			err := tt.cmd.Run()
			require.NoError(t, err)
			assert.True(t, mockCalled)
		})
	}
}

func TestEnhanceCmd_Run_PropagatesError(t *testing.T) {
	testutil.SetTestConfigWithOptions(t, testutil.WithTMDBAPIKey("test-key"))
	env := testutil.NewTestEnv(t)

	expectedErr := fmt.Errorf("mock error from enhance")
	mockFunc := func(opts Options) error {
		return expectedErr
	}

	origFunc := EnhanceNotesFunc
	EnhanceNotesFunc = mockFunc
	defer func() { EnhanceNotesFunc = origFunc }()

	cmd := EnhanceCmd{
		InputDirs: []string{env.RootDir()},
	}

	err := cmd.Run()
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
