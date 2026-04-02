package testutil

import (
	"testing"

	"github.com/gatheryourdeals/data/internal/repository/sqlite"
)

// NewTestDB creates a temporary in-memory SQLite database for testing.
// It runs migrations automatically and closes the database when the test finishes.
// MaxOpenConns is set to 1 because :memory: databases are per-connection in
// SQLite — without this, concurrent goroutines may get a second connection
// that sees an empty database (no tables).
func NewTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
