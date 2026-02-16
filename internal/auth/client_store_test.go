package auth_test

import (
	"context"
	"testing"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

func mustCreateTestClient(t *testing.T, repo *sqlite.ClientRepo, ctx context.Context, client *model.OAuthClient) {
	t.Helper()
	if err := repo.CreateClient(ctx, client); err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
}

func TestDBClientStore_GetByID_Found(t *testing.T) {
	db := testutil.NewTestDB(t)
	clientRepo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	mustCreateTestClient(t, clientRepo, ctx, &model.OAuthClient{
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
	info, err := store.GetByID(ctx, "new-client")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if info != nil {
		t.Fatal("expected nil before creation")
	}

	// Create via repo (simulates admin API)
	mustCreateTestClient(t, clientRepo, ctx, &model.OAuthClient{
		ID:     "new-client",
		Secret: "",
		Domain: "",
	})

	// Store should see it immediately
	info, err = store.GetByID(ctx, "new-client")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected client after creation, got nil")
	}

	// Delete via repo (simulates admin API)
	if err := clientRepo.DeleteClient(ctx, "new-client"); err != nil {
		t.Fatalf("DeleteClient failed: %v", err)
	}

	// Store should no longer see it
	info, err = store.GetByID(ctx, "new-client")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if info != nil {
		t.Fatal("expected nil after deletion")
	}
}
