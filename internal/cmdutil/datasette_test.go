package cmdutil

import (
	"database/sql"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type datasetteRecord struct {
	ID    int
	Title string
}

const datasetteSchema = `
CREATE TABLE IF NOT EXISTS test_items (
	id INTEGER PRIMARY KEY,
	title TEXT NOT NULL
);
`

func TestWriteToDatastore_Disabled(t *testing.T) {
	env := testutil.NewTestEnv(t)
	viper.Reset()
	viper.Set("datasette.enabled", false)
	viper.Set("datasette.dbfile", env.Path("test.db"))
	t.Cleanup(viper.Reset)

	records := []datasetteRecord{{ID: 1, Title: "Matrix"}}
	err := WriteToDatastore(records, datasetteSchema, "test_items", "test records", func(item datasetteRecord) map[string]any {
		return map[string]any{"id": item.ID, "title": item.Title}
	})
	require.NoError(t, err)

	assert.False(t, env.FileExists("test.db"))
}

func TestWriteToDatastore_WritesRows(t *testing.T) {
	env := testutil.NewTestEnv(t)
	viper.Reset()
	viper.Set("datasette.enabled", true)
	viper.Set("datasette.dbfile", env.Path("test.db"))
	t.Cleanup(viper.Reset)

	records := []datasetteRecord{{ID: 1, Title: "Matrix"}, {ID: 2, Title: "Inception"}}
	err := WriteToDatastore(records, datasetteSchema, "test_items", "test records", func(item datasetteRecord) map[string]any {
		return map[string]any{"id": item.ID, "title": item.Title}
	})
	require.NoError(t, err)

	db, err := sql.Open("sqlite", env.Path("test.db"))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_items").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
