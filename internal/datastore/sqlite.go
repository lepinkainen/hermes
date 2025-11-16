package datastore

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements the Store interface for local SQLite storage
type SQLiteStore struct {
	db     *sql.DB
	dbPath string
}

// NewSQLiteStore creates a new SQLiteStore instance
func NewSQLiteStore(dbPath string) *SQLiteStore {
	return &SQLiteStore{
		dbPath: dbPath,
	}
}

// Connect opens a connection to the SQLite database
func (s *SQLiteStore) Connect() error {
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db
	return nil
}

// CreateTable creates a new table with the given schema if it doesn't exist
func (s *SQLiteStore) CreateTable(schema string) error {
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

// BatchInsert inserts multiple records into the specified table
func (s *SQLiteStore) BatchInsert(database string, table string, records []map[string]any) error {
	if len(records) == 0 {
		return nil
	}

	// Start a transaction for batch insert
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		// Rollback if we don't commit - ignore errors as they're expected if transaction was committed
		_ = tx.Rollback()
	}()

	// Get column names from the first record
	var columns []string
	for col := range records[0] {
		columns = append(columns, col)
	}

	// Create the prepared statement
	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = "?"
	}
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	// Insert all records
	for _, record := range records {
		values := make([]any, len(columns))
		for i, col := range columns {
			values[i] = record[col]
		}

		if _, err := stmt.Exec(values...); err != nil {
			return fmt.Errorf("failed to insert record: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
