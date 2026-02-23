package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

type refreshStoreEnv struct {
	store    *sqlite.RefreshTokenStore
	userRepo *sqlite.UserRepo
	ctx      context.Context
}

func newRefreshStoreEnv(t *testing.T) *refreshStoreEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	return &refreshStoreEnv{
		store:    sqlite.NewRefreshTokenStore(db),
		userRepo: sqlite.NewUserRepo(db),
		ctx:      context.Background(),
	}
}

// seedUser inserts a minimal user row so foreign key constraints are satisfied.
func (e *refreshStoreEnv) seedUser(t *testing.T, id string) {
	t.Helper()
	err := e.userRepo.CreateUser(e.ctx, &model.User{
		ID:           id,
		Username:     id, // use id as username to keep them unique
		PasswordHash: "hash",
		Role:         model.RoleUser,
	})
	if err != nil {
		t.Fatalf("seedUser failed: %v", err)
	}
}

func TestRefreshTokenStore_SaveAndFind(t *testing.T) {
	env := newRefreshStoreEnv(t)
	env.seedUser(t, "user-1")

	exp := time.Now().Add(time.Hour)
	if err := env.store.Save(env.ctx, "token-abc", "user-1", exp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	userID, err := env.store.Find(env.ctx, "token-abc")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("expected userID 'user-1', got %q", userID)
	}
}

func TestRefreshTokenStore_Find_NotFound(t *testing.T) {
	env := newRefreshStoreEnv(t)

	_, err := env.store.Find(env.ctx, "nonexistent-token")
	if err != model.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRefreshTokenStore_Find_Expired(t *testing.T) {
	env := newRefreshStoreEnv(t)
	env.seedUser(t, "user-1")

	exp := time.Now().Add(-time.Second)
	if err := env.store.Save(env.ctx, "expired-token", "user-1", exp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	_, err := env.store.Find(env.ctx, "expired-token")
	if err != model.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for expired token, got %v", err)
	}
}

func TestRefreshTokenStore_Find_ExpiredIsDeleted(t *testing.T) {
	env := newRefreshStoreEnv(t)
	env.seedUser(t, "user-1")

	exp := time.Now().Add(-time.Second)
	if err := env.store.Save(env.ctx, "expired-token", "user-1", exp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// First Find triggers lazy deletion
	_, _ = env.store.Find(env.ctx, "expired-token")

	// Second Find confirms the row is gone
	_, err := env.store.Find(env.ctx, "expired-token")
	if err != model.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken after lazy deletion, got %v", err)
	}
}

func TestRefreshTokenStore_Delete(t *testing.T) {
	env := newRefreshStoreEnv(t)
	env.seedUser(t, "user-1")

	exp := time.Now().Add(time.Hour)
	if err := env.store.Save(env.ctx, "token-abc", "user-1", exp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := env.store.Delete(env.ctx, "token-abc"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := env.store.Find(env.ctx, "token-abc")
	if err != model.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken after delete, got %v", err)
	}
}

func TestRefreshTokenStore_Delete_Nonexistent(t *testing.T) {
	env := newRefreshStoreEnv(t)

	if err := env.store.Delete(env.ctx, "nonexistent"); err != nil {
		t.Fatalf("expected no error deleting nonexistent token, got %v", err)
	}
}

func TestRefreshTokenStore_DeleteAllForUser(t *testing.T) {
	env := newRefreshStoreEnv(t)
	env.seedUser(t, "user-1")
	env.seedUser(t, "user-2")

	exp := time.Now().Add(time.Hour)
	tokens := []string{"token-1", "token-2", "token-3"}
	for _, tok := range tokens {
		if err := env.store.Save(env.ctx, tok, "user-1", exp); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}
	if err := env.store.Save(env.ctx, "token-other", "user-2", exp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := env.store.DeleteAllForUser(env.ctx, "user-1"); err != nil {
		t.Fatalf("DeleteAllForUser failed: %v", err)
	}

	for _, tok := range tokens {
		_, err := env.store.Find(env.ctx, tok)
		if err != model.ErrInvalidToken {
			t.Errorf("expected token %q to be deleted, but Find returned: %v", tok, err)
		}
	}

	// user-2 token should be unaffected
	userID, err := env.store.Find(env.ctx, "token-other")
	if err != nil {
		t.Fatalf("expected user-2 token to survive, got error: %v", err)
	}
	if userID != "user-2" {
		t.Errorf("expected userID 'user-2', got %q", userID)
	}
}

func TestRefreshTokenStore_Save_DuplicateToken(t *testing.T) {
	env := newRefreshStoreEnv(t)
	env.seedUser(t, "user-1")
	env.seedUser(t, "user-2")

	exp := time.Now().Add(time.Hour)
	if err := env.store.Save(env.ctx, "token-abc", "user-1", exp); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	err := env.store.Save(env.ctx, "token-abc", "user-2", exp)
	if err == nil {
		t.Fatal("expected error on duplicate token, got nil")
	}
}
