package sqlite

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/XSAM/otelsql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

//go:embed migrations/*.sql
var migrations embed.FS

// DB wraps a sql.DB connection to a SQLite database.
type DB struct {
	conn *sql.DB
}

// New opens a SQLite database at the given path and runs migrations.
func New(dbPath string) (*DB, error) {
	conn, err := otelsql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on",
		otelsql.WithAttributes(semconv.DBSystemSqlite),
		otelsql.WithSpanOptions(otelsql.SpanOptions{DisableQuery: true}),
	)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// SetMaxOpenConns forwards to the underlying sql.DB. Useful in tests that use
// an in-memory SQLite database: :memory: is per-connection, so pinning to one
// connection ensures all goroutines share the same database.
func (db *DB) SetMaxOpenConns(n int) {
	db.conn.SetMaxOpenConns(n)
}

// migrate runs all pending goose migrations from the embedded SQL files.
// follwing example in https://github.com/pressly/goose?tab=readme-ov-file#go-migrations
func (db *DB) migrate() error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db.conn, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
