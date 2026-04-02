package sqlite

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/gatheryourdeals/data/internal/model"
)

// RefreshTokenStore is a SQLite-backed implementation of auth.RefreshTokenStore.
type RefreshTokenStore struct {
	db *DB
}

// NewRefreshTokenStore creates a new SQLite-backed refresh token store.
func NewRefreshTokenStore(db *DB) *RefreshTokenStore {
	return &RefreshTokenStore{db: db}
}

func (s *RefreshTokenStore) Save(ctx context.Context, token, userID string, expiresAt time.Time) error {
	_, err := s.db.conn.ExecContext(ctx,
		`INSERT INTO refresh_tokens (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt.Unix(),
	)
	return err
}

func (s *RefreshTokenStore) Find(ctx context.Context, token string) (string, error) {
	var userID string
	var expiresAt int64
	err := s.db.conn.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token = ?`, token,
	).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return "", model.ErrInvalidToken
	}
	if err != nil {
		return "", err
	}
	if time.Now().Unix() > expiresAt {
		if _, derr := s.db.conn.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token = ?`, token); derr != nil {
			slog.Warn("failed to delete expired refresh token", "error", derr)
		}
		return "", model.ErrInvalidToken
	}
	return userID, nil
}

// Consume atomically deletes the token and returns the userID in one statement.
// If the token is missing or expired, sql.ErrNoRows maps to ErrInvalidToken.
// Because SQLite serialises writers, the DELETE is the exclusive "claim" — a
// concurrent call that races here will find no rows and get ErrInvalidToken.
func (s *RefreshTokenStore) Consume(ctx context.Context, token string) (string, error) {
	var userID string
	err := s.db.conn.QueryRowContext(ctx,
		`DELETE FROM refresh_tokens WHERE token = ? AND expires_at > ? RETURNING user_id`,
		token, time.Now().Unix(),
	).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", model.ErrInvalidToken
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *RefreshTokenStore) Delete(ctx context.Context, token string) error {
	_, err := s.db.conn.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token = ?`, token)
	return err
}

func (s *RefreshTokenStore) DeleteAllForUser(ctx context.Context, userID string) error {
	_, err := s.db.conn.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	return err
}
