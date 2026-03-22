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
	now := time.Now().Unix()
	user.CreatedAt = now
	user.UpdatedAt = now
	_, err := r.db.conn.ExecContext(ctx, query,
		user.ID, user.Username, user.PasswordHash, string(user.Role), now, now)
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
	_, err := r.db.conn.ExecContext(ctx, query, passwordHash, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *UserRepo) ListUsers(ctx context.Context, params model.PaginationParams) (*model.Page[*model.User], error) {
	// Single query: window function returns total alongside each row.
	// SortBy and SortOrder are validated by the handler.
	query := fmt.Sprintf(
		`SELECT `+userColumns+`, COUNT(*) OVER() FROM users ORDER BY %s %s LIMIT ? OFFSET ?`,
		params.SortBy, params.SortOrder,
	)
	rows, err := r.db.conn.QueryContext(ctx, query, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []*model.User
	var total int
	for rows.Next() {
		var u model.User
		var role string
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &role, &u.CreatedAt, &u.UpdatedAt, &total); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.Role = model.Role(role)
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// If no rows returned with a non-zero offset, the offset is past the end;
	// run a separate count to fill in the total for the response envelope.
	if len(users) == 0 && params.Offset > 0 {
		if err := r.db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
			return nil, fmt.Errorf("count users: %w", err)
		}
	}

	page := &model.Page[*model.User]{
		Data:   users,
		Total:  total,
		Offset: params.Offset,
		Limit:  params.Limit,
	}
	if total > 0 {
		page.TotalPages = (total + params.Limit - 1) / params.Limit
	}
	if page.Data == nil {
		page.Data = []*model.User{}
	}
	return page, nil
}

func (r *UserRepo) DeleteUser(ctx context.Context, id string) error {
	result, err := r.db.conn.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
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
