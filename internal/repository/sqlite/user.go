package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gatheryourdeals/data/internal/model"
)

// return all columns from user, need to change this when changing the schema
const userColumns = "id, username, password_hash, role, created_at, updated_at"

// UserRepo implements repository.UserRepository backed by SQLite.
type UserRepo struct {
	db *DB
}

// NewUserRepo creates a new SQLite-backed user repository.
func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (` + userColumns + `) VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	_, err := r.db.conn.ExecContext(ctx, query,
		user.ID, user.Username, user.PasswordHash, string(user.Role), user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return r.scanUser(ctx, "SELECT "+userColumns+" FROM users WHERE id = ?", id)
}

func (r *UserRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.scanUser(ctx, "SELECT "+userColumns+" FROM users WHERE username = ?", username)
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.conn.ExecContext(ctx, query, passwordHash, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *UserRepo) ListUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := r.db.conn.QueryContext(ctx, "SELECT "+userColumns+" FROM users")
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) DeleteUser(ctx context.Context, id string) error {
	_, err := r.db.conn.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *UserRepo) HasAdmin(ctx context.Context) (bool, error) {
	var count int
	err := r.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = ?", string(model.RoleAdmin)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has admin: %w", err)
	}
	return count > 0, nil
}

// scanUser executes a query expected to return a single user row.
func (r *UserRepo) scanUser(ctx context.Context, query string, args ...interface{}) (*model.User, error) {
	row := r.db.conn.QueryRowContext(ctx, query, args...)
	var u model.User
	// role is stored as string in the database, need to convert to model.Role
	var role string
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &role, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Role = model.Role(role)
	return &u, nil
}

// scanRow scans a single row from a multi-row result set.
func scanRow(rows *sql.Rows) (*model.User, error) {
	var u model.User
	var role string
	err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan row: %w", err)
	}
	u.Role = model.Role(role)
	return &u, nil
}
