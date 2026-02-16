package testutil

import (
	"testing"

	"github.com/gatheryourdeals/data/internal/repository/sqlite"
)

// NewTestDB creates a temporary in-memory SQLite database for testing.
// It runs migrations automatically and closes the database when the test finishes.
func NewTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
