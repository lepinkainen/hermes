package datastore

import (
	"testing"
)

func TestSQLiteStore_CreateTableAndInsert(t *testing.T) {
	dbPath := "file::memory:?cache=shared"
	store := NewSQLiteStore(dbPath)
	if err := store.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() { _ = store.Close() }()

	schema := `CREATE TABLE IF NOT EXISTS test_table (
		id INTEGER PRIMARY KEY,
		name TEXT,
		value INTEGER
	)`
	if err := store.CreateTable(schema); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	records := []map[string]any{
		{"id": 1, "name": "foo", "value": 42},
		{"id": 2, "name": "bar", "value": 99},
	}
	if err := store.BatchInsert("hermes", "test_table", records); err != nil {
		t.Fatalf("failed to batch insert: %v", err)
	}

	// Verify inserted rows
	rows, err := store.db.Query("SELECT id, name, value FROM test_table ORDER BY id")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var count int
	for rows.Next() {
		var id, value int
		var name string
		if err := rows.Scan(&id, &name, &value); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}
}

func TestSQLiteStore_UpsertBehavior(t *testing.T) {
	dbPath := "file::memory:?cache=shared"
	store := NewSQLiteStore(dbPath)
	if err := store.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() { _ = store.Close() }()

	schema := `CREATE TABLE IF NOT EXISTS test_upsert (
		id INTEGER PRIMARY KEY,
		name TEXT,
		value INTEGER
	)`
	if err := store.CreateTable(schema); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert initial records
	records := []map[string]any{
		{"id": 1, "name": "foo", "value": 42},
		{"id": 2, "name": "bar", "value": 99},
	}
	if err := store.BatchInsert("hermes", "test_upsert", records); err != nil {
		t.Fatalf("failed to batch insert: %v", err)
	}

	// Insert records with same primary keys (should replace, not error)
	updatedRecords := []map[string]any{
		{"id": 1, "name": "foo_updated", "value": 100},
		{"id": 2, "name": "bar_updated", "value": 200},
	}
	if err := store.BatchInsert("hermes", "test_upsert", updatedRecords); err != nil {
		t.Fatalf("upsert failed with error: %v", err)
	}

	// Verify records were replaced
	rows, err := store.db.Query("SELECT id, name, value FROM test_upsert ORDER BY id")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	expected := []struct {
		id    int
		name  string
		value int
	}{
		{1, "foo_updated", 100},
		{2, "bar_updated", 200},
	}

	var idx int
	for rows.Next() {
		var id, value int
		var name string
		if err := rows.Scan(&id, &name, &value); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		if id != expected[idx].id || name != expected[idx].name || value != expected[idx].value {
			t.Errorf("row %d: expected {%d, %s, %d}, got {%d, %s, %d}",
				idx, expected[idx].id, expected[idx].name, expected[idx].value, id, name, value)
		}
		idx++
	}
	if idx != 2 {
		t.Errorf("expected 2 rows, got %d", idx)
	}
}
