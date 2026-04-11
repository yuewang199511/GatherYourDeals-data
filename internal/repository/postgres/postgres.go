package postgres

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

//go:embed migrations/*.sql
var migrations embed.FS

// DB wraps a sql.DB connection to a PostgreSQL database.
type DB struct {
	conn *sql.DB
}

// withSimpleProtocol appends default_query_exec_mode=simple_protocol to the DSN.
// Required for PgBouncer transaction mode — prepared statements are not preserved
// across connections in transaction mode, so pgx must use the simple wire protocol.
// Safe to apply in session mode too (no functional difference, minor perf overhead).
func withSimpleProtocol(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse DSN: %w", err)
	}
	q := u.Query()
	q.Set("default_query_exec_mode", "simple_protocol")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// New opens a PostgreSQL database at the given DSN and runs migrations.
// dsn may be a URI (postgres://...) or a key-value DSN (host=... port=...).
func New(dsn string) (*DB, error) {
	dsn, err := withSimpleProtocol(dsn)
	if err != nil {
		return nil, err
	}
	conn, err := otelsql.Open("pgx", dsn,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{DisableQuery: true}),
	)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
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

// migrate runs all pending goose migrations from the embedded SQL files.
func (db *DB) migrate() error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db.conn, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
