package repository

import (
	"context"

	"github.com/gatheryourdeals/data/internal/model"
)

// ClientRepository defines the storage operations for OAuth2 clients.
// Implementations can use SQLite, PostgreSQL, or any other backend.
type ClientRepository interface {
	// CreateClient inserts a new OAuth2 client.
	CreateClient(ctx context.Context, client *model.OAuthClient) error

	// GetClientByID returns a client by its ID.
	GetClientByID(ctx context.Context, id string) (*model.OAuthClient, error)

	// ListClients returns all registered clients.
	ListClients(ctx context.Context) ([]*model.OAuthClient, error)

	// DeleteClient removes a client by its ID.
	DeleteClient(ctx context.Context, id string) error

	// HasClients returns true if at least one client exists.
	HasClients(ctx context.Context) (bool, error)
}

// UserRepository defines the storage operations for user accounts.
// Implementations can use SQLite, PostgreSQL, or any other backend.
type UserRepository interface {
	// CreateUser inserts a new user into the store.
	CreateUser(ctx context.Context, user *model.User) error

	// GetUserByID returns a user by their unique ID.
	GetUserByID(ctx context.Context, id string) (*model.User, error)

	// GetUserByUsername returns a user by their username.
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)

	// UpdatePassword updates the password hash for a user.
	UpdatePassword(ctx context.Context, id string, passwordHash string) error

	// ListUsers returns all registered users.
	ListUsers(ctx context.Context) ([]*model.User, error)

	// DeleteUser removes a user by their ID.
	DeleteUser(ctx context.Context, id string) error

	// HasAdmin returns true if at least one admin account exists.
	HasAdmin(ctx context.Context) (bool, error)
}
