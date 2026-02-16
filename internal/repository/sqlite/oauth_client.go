package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gatheryourdeals/data/internal/model"
)

const clientColumns = "id, secret, domain, created_at"

// ClientRepo implements repository.ClientRepository backed by SQLite.
type ClientRepo struct {
	db *DB
}

// NewClientRepo creates a new SQLite-backed client repository.
func NewClientRepo(db *DB) *ClientRepo {
	return &ClientRepo{db: db}
}

func (r *ClientRepo) CreateClient(ctx context.Context, client *model.OAuthClient) error {
	query := `INSERT INTO oauth_clients (` + clientColumns + `) VALUES (?, ?, ?, ?)`
	client.CreatedAt = time.Now().UTC()
	_, err := r.db.conn.ExecContext(ctx, query,
		client.ID, client.Secret, client.Domain, client.CreatedAt)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	return nil
}

func (r *ClientRepo) GetClientByID(ctx context.Context, id string) (*model.OAuthClient, error) {
	query := "SELECT " + clientColumns + " FROM oauth_clients WHERE id = ?"
	row := r.db.conn.QueryRowContext(ctx, query, id)
	var c model.OAuthClient
	err := row.Scan(&c.ID, &c.Secret, &c.Domain, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	return &c, nil
}

func (r *ClientRepo) ListClients(ctx context.Context) ([]*model.OAuthClient, error) {
	rows, err := r.db.conn.QueryContext(ctx, "SELECT "+clientColumns+" FROM oauth_clients")
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()

	var clients []*model.OAuthClient
	for rows.Next() {
		var c model.OAuthClient
		if err := rows.Scan(&c.ID, &c.Secret, &c.Domain, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan client: %w", err)
		}
		clients = append(clients, &c)
	}
	return clients, rows.Err()
}

func (r *ClientRepo) DeleteClient(ctx context.Context, id string) error {
	_, err := r.db.conn.ExecContext(ctx, "DELETE FROM oauth_clients WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete client: %w", err)
	}
	return nil
}

func (r *ClientRepo) HasClients(ctx context.Context) (bool, error) {
	var count int
	err := r.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_clients").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has clients: %w", err)
	}
	return count > 0, nil
}
