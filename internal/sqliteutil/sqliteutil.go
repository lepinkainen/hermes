// Package sqliteutil provides shared helpers for opening modernc.org/sqlite
// database connections.
package sqliteutil

import "strings"

// pragma is a single per-connection SQLite pragma to append to a DSN.
type pragma struct {
	name string
	args string
}

// dsnPragmas are applied to every connection the pool opens (pragmas are
// per-connection, not per-file).
var dsnPragmas = []pragma{
	{"foreign_keys", "1"},
	{"busy_timeout", "5000"},
	{"journal_mode", "WAL"},
	{"synchronous", "NORMAL"},
}

// DSN builds a modernc.org/sqlite DSN with pragmas applied to every
// connection the pool opens (pragmas are per-connection, not per-file).
// dbPath may already carry its own query string (e.g. "file::memory:?cache=shared").
// A pragma already present in dbPath is left alone rather than appended a
// second time - the modernc.org/sqlite driver does not guarantee the
// ordering of duplicate pragmas, so a user-supplied value should win.
func DSN(dbPath string) string {
	sep := "?"
	if strings.Contains(dbPath, "?") {
		sep = "&"
	}

	var dsn strings.Builder
	dsn.WriteString(dbPath)
	for _, p := range dsnPragmas {
		if strings.Contains(dbPath, "_pragma="+p.name+"(") {
			continue
		}
		dsn.WriteString(sep + "_pragma=" + p.name + "(" + p.args + ")")
		sep = "&"
	}
	return dsn.String()
}
