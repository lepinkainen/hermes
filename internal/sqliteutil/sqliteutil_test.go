package sqliteutil

import (
	"strings"
	"testing"
)

func TestDSN(t *testing.T) {
	tests := []struct {
		name        string
		dbPath      string
		wantContain []string
		wantNotHave []string
	}{
		{
			name:   "plain path",
			dbPath: "./cache.db",
			wantContain: []string{
				"./cache.db?",
				"_pragma=foreign_keys(1)",
				"_pragma=busy_timeout(5000)",
				"_pragma=journal_mode(WAL)",
				"_pragma=synchronous(NORMAL)",
			},
		},
		{
			name:   "path with existing query string uses &",
			dbPath: "file:x.db?cache=shared",
			wantContain: []string{
				"file:x.db?cache=shared&_pragma=foreign_keys(1)",
				"_pragma=busy_timeout(5000)",
				"_pragma=journal_mode(WAL)",
				"_pragma=synchronous(NORMAL)",
			},
		},
		{
			name:   "path with existing busy_timeout pragma is not duplicated",
			dbPath: "file:x.db?_pragma=busy_timeout(30000)",
			wantContain: []string{
				"_pragma=busy_timeout(30000)",
				"_pragma=foreign_keys(1)",
				"_pragma=journal_mode(WAL)",
				"_pragma=synchronous(NORMAL)",
			},
			wantNotHave: []string{
				"_pragma=busy_timeout(5000)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DSN(tt.dbPath)
			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("DSN(%q) = %q, want it to contain %q", tt.dbPath, got, want)
				}
			}
			for _, notWant := range tt.wantNotHave {
				if strings.Contains(got, notWant) {
					t.Errorf("DSN(%q) = %q, want it NOT to contain %q", tt.dbPath, got, notWant)
				}
			}
			if tt.name == "path with existing busy_timeout pragma is not duplicated" {
				if strings.Count(got, "_pragma=busy_timeout(") != 1 {
					t.Errorf("DSN(%q) = %q, want exactly one busy_timeout pragma", tt.dbPath, got)
				}
			}
		})
	}
}
