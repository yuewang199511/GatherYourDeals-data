package sqlite_test

import (
	"context"
	"testing"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

func TestCreateUser(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	user := &model.User{
		ID:           "user-1",
		Username:     "alice",
		PasswordHash: "hashed",
		Role:         model.RoleUser,
	}

	err := repo.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	got, err := repo.GetUserByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.Username != "alice" {
		t.Errorf("expected username alice, got %s", got.Username)
	}
	if got.Role != model.RoleUser {
		t.Errorf("expected role user, got %s", got.Role)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	user1 := &model.User{ID: "u1", Username: "alice", PasswordHash: "h1", Role: model.RoleUser}
	user2 := &model.User{ID: "u2", Username: "alice", PasswordHash: "h2", Role: model.RoleUser}

	if err := repo.CreateUser(ctx, user1); err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}
	err := repo.CreateUser(ctx, user2)
	if err == nil {
		t.Fatal("expected error on duplicate username, got nil")
	}
}

func TestGetUserByUsername(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	user := &model.User{ID: "u1", Username: "bob", PasswordHash: "h", Role: model.RoleAdmin}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	got, err := repo.GetUserByUsername(ctx, "bob")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != "u1" {
		t.Errorf("expected ID u1, got %s", got.ID)
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	got, err := repo.GetUserByUsername(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got user %v", got)
	}
}

func TestUpdatePassword(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	user := &model.User{ID: "u1", Username: "alice", PasswordHash: "old", Role: model.RoleUser}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if err := repo.UpdatePassword(ctx, "u1", "new"); err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	got, _ := repo.GetUserByID(ctx, "u1")
	if got.PasswordHash != "new" {
		t.Errorf("expected password hash 'new', got '%s'", got.PasswordHash)
	}
}

func TestListUsers(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	repo.CreateUser(ctx, &model.User{ID: "u1", Username: "alice", PasswordHash: "h", Role: model.RoleUser})
	repo.CreateUser(ctx, &model.User{ID: "u2", Username: "bob", PasswordHash: "h", Role: model.RoleAdmin})

	users, err := repo.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestDeleteUser(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	repo.CreateUser(ctx, &model.User{ID: "u1", Username: "alice", PasswordHash: "h", Role: model.RoleUser})

	if err := repo.DeleteUser(ctx, "u1"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	got, _ := repo.GetUserByID(ctx, "u1")
	if got != nil {
		t.Fatal("expected nil after delete, got user")
	}
}

func TestHasAdmin(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	has, err := repo.HasAdmin(ctx)
	if err != nil {
		t.Fatalf("HasAdmin failed: %v", err)
	}
	if has {
		t.Fatal("expected no admin on empty database")
	}

	repo.CreateUser(ctx, &model.User{ID: "u1", Username: "admin", PasswordHash: "h", Role: model.RoleAdmin})

	has, err = repo.HasAdmin(ctx)
	if err != nil {
		t.Fatalf("HasAdmin failed: %v", err)
	}
	if !has {
		t.Fatal("expected admin to exist")
	}
}
