package auth_test

import (
	"context"
	"testing"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

func TestDBClientStore_GetByID_Found(t *testing.T) {
	db := testutil.NewTestDB(t)
	clientRepo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	clientRepo.CreateClient(ctx, &model.OAuthClient{
		ID:     "test-client",
		Secret: "secret123",
		Domain: "http://localhost",
	})

	store := auth.NewDBClientStore(clientRepo)

	info, err := store.GetByID(ctx, "test-client")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected client, got nil")
	}
	if info.GetID() != "test-client" {
		t.Errorf("expected ID 'test-client', got '%s'", info.GetID())
	}
	if info.GetSecret() != "secret123" {
		t.Errorf("expected secret 'secret123', got '%s'", info.GetSecret())
	}
	if info.GetDomain() != "http://localhost" {
		t.Errorf("expected domain 'http://localhost', got '%s'", info.GetDomain())
	}
}

func TestDBClientStore_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	clientRepo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	store := auth.NewDBClientStore(clientRepo)

	info, err := store.GetByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Fatalf("expected nil, got client %v", info)
	}
}

func TestDBClientStore_ReflectsChanges(t *testing.T) {
	db := testutil.NewTestDB(t)
	clientRepo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	store := auth.NewDBClientStore(clientRepo)

	// Client doesn't exist yet
	info, _ := store.GetByID(ctx, "new-client")
	if info != nil {
		t.Fatal("expected nil before creation")
	}

	// Create via repo (simulates admin API)
	clientRepo.CreateClient(ctx, &model.OAuthClient{
		ID:     "new-client",
		Secret: "",
		Domain: "",
	})

	// Store should see it immediately
	info, err := store.GetByID(ctx, "new-client")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected client after creation, got nil")
	}

	// Delete via repo (simulates admin API)
	clientRepo.DeleteClient(ctx, "new-client")

	// Store should no longer see it
	info, _ = store.GetByID(ctx, "new-client")
	if info != nil {
		t.Fatal("expected nil after deletion")
	}
}
