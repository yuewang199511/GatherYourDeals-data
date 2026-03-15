package sqlite_test

import (
	"context"
	"testing"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

func mustCreateUser(t *testing.T, repo *sqlite.UserRepo, ctx context.Context, user *model.User) {
	t.Helper()
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
}

// defaultUserParams returns pagination params suitable for tests that don't need
// to test pagination specifically (fetches all, default sort).
func defaultUserParams() model.PaginationParams {
	return model.PaginationParams{Offset: 0, Limit: 100, SortBy: "created_at", SortOrder: "DESC"}
}

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

	mustCreateUser(t, repo, ctx, user)

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

	mustCreateUser(t, repo, ctx, user1)
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
	mustCreateUser(t, repo, ctx, user)

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
	mustCreateUser(t, repo, ctx, user)

	if err := repo.UpdatePassword(ctx, "u1", "new"); err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	got, err := repo.GetUserByID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got.PasswordHash != "new" {
		t.Errorf("expected password hash 'new', got '%s'", got.PasswordHash)
	}
}

func TestListUsers(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	mustCreateUser(t, repo, ctx, &model.User{ID: "u1", Username: "alice", PasswordHash: "h", Role: model.RoleUser})
	mustCreateUser(t, repo, ctx, &model.User{ID: "u2", Username: "bob", PasswordHash: "h", Role: model.RoleAdmin})

	page, err := repo.ListUsers(ctx, defaultUserParams())
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if page.Total != 2 {
		t.Errorf("expected total 2, got %d", page.Total)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 users in data, got %d", len(page.Data))
	}
}

func TestListUsers_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	page, err := repo.ListUsers(ctx, defaultUserParams())
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if page.Total != 0 {
		t.Errorf("expected total 0, got %d", page.Total)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(page.Data))
	}
	if page.Data == nil {
		t.Error("expected non-nil Data slice, got nil")
	}
	if page.TotalPages != 0 {
		t.Errorf("expected total_pages 0, got %d", page.TotalPages)
	}
}

func TestListUsers_Pagination(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	mustCreateUser(t, repo, ctx, &model.User{ID: "u1", Username: "alice", PasswordHash: "h", Role: model.RoleUser})
	mustCreateUser(t, repo, ctx, &model.User{ID: "u2", Username: "bob", PasswordHash: "h", Role: model.RoleUser})
	mustCreateUser(t, repo, ctx, &model.User{ID: "u3", Username: "carol", PasswordHash: "h", Role: model.RoleUser})

	params := model.PaginationParams{Offset: 1, Limit: 2, SortBy: "username", SortOrder: "ASC"}
	page, err := repo.ListUsers(ctx, params)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if page.Total != 3 {
		t.Errorf("expected total 3, got %d", page.Total)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 users in page, got %d", len(page.Data))
	}
	if page.TotalPages != 2 {
		t.Errorf("expected total_pages 2, got %d", page.TotalPages)
	}
}

func TestListUsers_SortByUsername(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	mustCreateUser(t, repo, ctx, &model.User{ID: "u1", Username: "zebra", PasswordHash: "h", Role: model.RoleUser})
	mustCreateUser(t, repo, ctx, &model.User{ID: "u2", Username: "apple", PasswordHash: "h", Role: model.RoleUser})

	params := model.PaginationParams{Offset: 0, Limit: 10, SortBy: "username", SortOrder: "ASC"}
	page, err := repo.ListUsers(ctx, params)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 users, got %d", len(page.Data))
	}
	if page.Data[0].Username != "apple" {
		t.Errorf("expected first user 'apple', got %q", page.Data[0].Username)
	}
	if page.Data[1].Username != "zebra" {
		t.Errorf("expected second user 'zebra', got %q", page.Data[1].Username)
	}
}

func TestListUsers_OffsetBeyondTotal(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	mustCreateUser(t, repo, ctx, &model.User{ID: "u1", Username: "alice", PasswordHash: "h", Role: model.RoleUser})

	params := model.PaginationParams{Offset: 100, Limit: 10, SortBy: "created_at", SortOrder: "DESC"}
	page, err := repo.ListUsers(ctx, params)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("expected total 1, got %d", page.Total)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected empty data when offset > total, got %d items", len(page.Data))
	}
}

func TestDeleteUser(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	mustCreateUser(t, repo, ctx, &model.User{ID: "u1", Username: "alice", PasswordHash: "h", Role: model.RoleUser})

	if err := repo.DeleteUser(ctx, "u1"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	got, err := repo.GetUserByID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
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

	mustCreateUser(t, repo, ctx, &model.User{ID: "u1", Username: "admin", PasswordHash: "h", Role: model.RoleAdmin})

	has, err = repo.HasAdmin(ctx)
	if err != nil {
		t.Fatalf("HasAdmin failed: %v", err)
	}
	if !has {
		t.Fatal("expected admin to exist")
	}
}
