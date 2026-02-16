package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUsernameExists    = errors.New("username already exists")
	ErrInvalidCredential = errors.New("invalid username or password")
	ErrAdminExists       = errors.New("admin account already exists")
)

// Service handles authentication and user management business logic.
type Service struct {
	users repository.UserRepository
}

// NewService creates a new auth service.
func NewService(users repository.UserRepository) *Service {
	return &Service{users: users}
}

// CreateAdmin creates the initial admin account.
// Returns ErrAdminExists if an admin already exists.
func (s *Service) CreateAdmin(ctx context.Context, username, password string) (*model.User, error) {
	exists, err := s.users.HasAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAdminExists
	}
	return s.createUser(ctx, username, password, model.RoleAdmin)
}

// Register creates a new regular user account. Open registration, immediately active.
func (s *Service) Register(ctx context.Context, username, password string) (*model.User, error) {
	existing, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameExists
	}
	return s.createUser(ctx, username, password, model.RoleUser)
}

// Login verifies credentials and returns the user if valid.
func (s *Service) Login(ctx context.Context, username, password string) (*model.User, error) {
	user, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredential
	}
	if err := CheckPassword(password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredential
	}
	return user, nil
}

// ResetPassword changes a user's password by username. Used by the admin CLI.
func (s *Service) ResetPassword(ctx context.Context, username, newPassword string) error {
	user, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.users.UpdatePassword(ctx, user.ID, hash)
}

// HasAdmin checks whether an admin account exists in the system.
func (s *Service) HasAdmin(ctx context.Context) (bool, error) {
	return s.users.HasAdmin(ctx)
}

func (s *Service) createUser(ctx context.Context, username, password string, role model.Role) (*model.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: hash,
		Role:         role,
	}
	if err := s.users.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}
