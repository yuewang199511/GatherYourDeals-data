package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gatheryourdeals/data/internal/model"
)

const userColumns = "id, username, password_hash, role, created_at, updated_at"

// UserRepo implements repository.UserRepository backed by PostgreSQL.
type UserRepo struct {
	db *DB
}

// NewUserRepo creates a new PostgreSQL-backed user repository.
func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (` + userColumns + `) VALUES ($1, $2, $3, $4, $5, $6)`
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
	return r.scanUser(ctx, "SELECT "+userColumns+" FROM users WHERE id = $1", id)
}

func (r *UserRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.scanUser(ctx, "SELECT "+userColumns+" FROM users WHERE username = $1", username)
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.conn.ExecContext(ctx, query, passwordHash, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *UserRepo) ListUsers(ctx context.Context, params model.PaginationParams) (*model.Page[*model.User], error) {
	// Count total users.
	var total int
	if err := r.db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	page := &model.Page[*model.User]{
		Data:   []*model.User{},
		Total:  total,
		Offset: params.Offset,
		Limit:  params.Limit,
	}
	if total > 0 {
		page.TotalPages = (total + params.Limit - 1) / params.Limit
	}

	if total == 0 || params.Offset >= total {
		page.Data = []*model.User{}
		return page, nil
	}

	// Fetch paginated data. SortBy and SortOrder are validated by the handler.
	query := fmt.Sprintf(
		`SELECT `+userColumns+` FROM users ORDER BY %s %s LIMIT $1 OFFSET $2`,
		params.SortBy, params.SortOrder,
	)
	rows, err := r.db.conn.QueryContext(ctx, query, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []*model.User
	for rows.Next() {
		u, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if users == nil {
		users = []*model.User{}
	}
	page.Data = users
	return page, nil
}

func (r *UserRepo) DeleteUser(ctx context.Context, id string) error {
	_, err := r.db.conn.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *UserRepo) HasAdmin(ctx context.Context) (bool, error) {
	var count int
	err := r.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = $1", string(model.RoleAdmin)).Scan(&count)
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
