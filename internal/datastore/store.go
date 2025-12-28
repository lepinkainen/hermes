package datastore

// Store defines the interface for local SQLite storage
type Store interface {
	// Connect establishes a connection to the data store
	Connect() error

	// CreateTable creates a new table with the given schema if it doesn't exist
	CreateTable(schema string) error

	// BatchInsert inserts multiple records into the specified table
	BatchInsert(database string, table string, records []map[string]any) error

	// Close closes the connection to the data store
	Close() error
}
