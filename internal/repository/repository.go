package repository

import (
	"context"

	"github.com/gatheryourdeals/data/internal/model"
)

// MetaFieldRepository defines the storage operations for field metadata.
type MetaFieldRepository interface {
	// CreateField registers a new user-defined field.
	CreateField(ctx context.Context, field *model.MetaField) error

	// GetField returns a single field by name.
	GetField(ctx context.Context, fieldName string) (*model.MetaField, error)

	// ListFields returns a paginated list of registered fields (native + user-defined).
	ListFields(ctx context.Context, params model.PaginationParams) (*model.Page[*model.MetaField], error)

	// UpdateDescription updates the description of an existing field.
	UpdateDescription(ctx context.Context, fieldName string, description string) error
}

// ReceiptRepository defines the storage operations for purchase records.
type ReceiptRepository interface {
	// CreateReceipt inserts a new purchase record.
	CreateReceipt(ctx context.Context, receipt *model.Receipt) error

	// GetReceiptByID returns a single receipt by its ID.
	GetReceiptByID(ctx context.Context, id string) (*model.Receipt, error)

	// ListReceiptsByUser returns a paginated list of receipts for a given user.
	ListReceiptsByUser(ctx context.Context, userID string, params model.PaginationParams) (*model.Page[*model.Receipt], error)

	// DeleteReceipt removes a receipt by its ID.
	DeleteReceipt(ctx context.Context, id string) error
}

// UserRepository defines the storage operations for user accounts.
type UserRepository interface {
	// CreateUser inserts a new user into the store.
	CreateUser(ctx context.Context, user *model.User) error

	// GetUserByID returns a user by their unique ID.
	GetUserByID(ctx context.Context, id string) (*model.User, error)

	// GetUserByUsername returns a user by their username.
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)

	// UpdatePassword updates the password hash for a user.
	UpdatePassword(ctx context.Context, id string, passwordHash string) error

	// ListUsers returns a paginated list of registered users.
	ListUsers(ctx context.Context, params model.PaginationParams) (*model.Page[*model.User], error)

	// DeleteUser removes a user by their ID.
	DeleteUser(ctx context.Context, id string) error

	// HasAdmin returns true if at least one admin account exists.
	HasAdmin(ctx context.Context) (bool, error)
}
