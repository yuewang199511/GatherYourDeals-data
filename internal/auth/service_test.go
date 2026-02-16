package auth_test

import (
	"context"
	"testing"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

func newTestService(t *testing.T) *auth.Service {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	return auth.NewService(repo)
}

func TestCreateAdmin(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	user, err := svc.CreateAdmin(ctx, "admin", "password123")
	if err != nil {
		t.Fatalf("CreateAdmin failed: %v", err)
	}
	if user.Username != "admin" {
		t.Errorf("expected username 'admin', got '%s'", user.Username)
	}
	if user.Role != "admin" {
		t.Errorf("expected role 'admin', got '%s'", user.Role)
	}
}

func TestCreateAdmin_AlreadyExists(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateAdmin(ctx, "admin", "password123"); err != nil {
		t.Fatalf("first CreateAdmin failed: %v", err)
	}

	_, err := svc.CreateAdmin(ctx, "admin2", "password456")
	if err != auth.ErrAdminExists {
		t.Fatalf("expected ErrAdminExists, got %v", err)
	}
}

func TestRegister(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected username 'alice', got '%s'", user.Username)
	}
	if user.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", user.Role)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "alice", "password123"); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	_, err := svc.Register(ctx, "alice", "password456")
	if err != auth.ErrUsernameExists {
		t.Fatalf("expected ErrUsernameExists, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "alice", "password123"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	user, err := svc.Login(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected username 'alice', got '%s'", user.Username)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "alice", "password123"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, err := svc.Login(ctx, "alice", "wrongpassword")
	if err != auth.ErrInvalidCredential {
		t.Fatalf("expected ErrInvalidCredential, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Login(ctx, "nonexistent", "password123")
	if err != auth.ErrInvalidCredential {
		t.Fatalf("expected ErrInvalidCredential, got %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "alice", "oldpassword1"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := svc.ResetPassword(ctx, "alice", "newpassword1"); err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	// Old password should fail
	_, err := svc.Login(ctx, "alice", "oldpassword1")
	if err != auth.ErrInvalidCredential {
		t.Fatal("expected old password to fail after reset")
	}

	// New password should work
	user, err := svc.Login(ctx, "alice", "newpassword1")
	if err != nil {
		t.Fatalf("Login with new password failed: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected username 'alice', got '%s'", user.Username)
	}
}

func TestResetPassword_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	err := svc.ResetPassword(ctx, "nonexistent", "newpassword1")
	if err != auth.ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestHasAdmin(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	has, err := svc.HasAdmin(ctx)
	if err != nil {
		t.Fatalf("HasAdmin failed: %v", err)
	}
	if has {
		t.Fatal("expected no admin")
	}

	if _, err := svc.CreateAdmin(ctx, "admin", "password123"); err != nil {
		t.Fatalf("CreateAdmin failed: %v", err)
	}

	has, err = svc.HasAdmin(ctx)
	if err != nil {
		t.Fatalf("HasAdmin failed: %v", err)
	}
	if !has {
		t.Fatal("expected admin to exist")
	}
}
