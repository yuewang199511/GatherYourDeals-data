package sqlite_test

import (
	"context"
	"testing"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

func mustCreateClient(t *testing.T, repo *sqlite.ClientRepo, ctx context.Context, client *model.OAuthClient) {
	t.Helper()
	if err := repo.CreateClient(ctx, client); err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}
}

func TestCreateClient(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	client := &model.OAuthClient{
		ID:     "test-client",
		Secret: "secret",
		Domain: "http://localhost",
	}

	mustCreateClient(t, repo, ctx, client)

	got, err := repo.GetClientByID(ctx, "test-client")
	if err != nil {
		t.Fatalf("GetClientByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected client, got nil")
	}
	if got.Secret != "secret" {
		t.Errorf("expected secret 'secret', got '%s'", got.Secret)
	}
	if got.Domain != "http://localhost" {
		t.Errorf("expected domain 'http://localhost', got '%s'", got.Domain)
	}
}

func TestCreateClient_DuplicateID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	c1 := &model.OAuthClient{ID: "client-1", Secret: "", Domain: ""}
	c2 := &model.OAuthClient{ID: "client-1", Secret: "other", Domain: ""}

	mustCreateClient(t, repo, ctx, c1)
	err := repo.CreateClient(ctx, c2)
	if err == nil {
		t.Fatal("expected error on duplicate client ID, got nil")
	}
}

func TestGetClientByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	got, err := repo.GetClientByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got client %v", got)
	}
}

func TestListClients(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	mustCreateClient(t, repo, ctx, &model.OAuthClient{ID: "c1", Secret: "", Domain: ""})
	mustCreateClient(t, repo, ctx, &model.OAuthClient{ID: "c2", Secret: "", Domain: ""})

	clients, err := repo.ListClients(ctx)
	if err != nil {
		t.Fatalf("ListClients failed: %v", err)
	}
	if len(clients) != 2 {
		t.Errorf("expected 2 clients, got %d", len(clients))
	}
}

func TestDeleteClient(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	mustCreateClient(t, repo, ctx, &model.OAuthClient{ID: "c1", Secret: "", Domain: ""})

	if err := repo.DeleteClient(ctx, "c1"); err != nil {
		t.Fatalf("DeleteClient failed: %v", err)
	}

	got, err := repo.GetClientByID(ctx, "c1")
	if err != nil {
		t.Fatalf("GetClientByID failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil after delete, got client")
	}
}

func TestHasClients(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	has, err := repo.HasClients(ctx)
	if err != nil {
		t.Fatalf("HasClients failed: %v", err)
	}
	if has {
		t.Fatal("expected no clients on empty database")
	}

	mustCreateClient(t, repo, ctx, &model.OAuthClient{ID: "c1", Secret: "", Domain: ""})

	has, err = repo.HasClients(ctx)
	if err != nil {
		t.Fatalf("HasClients failed: %v", err)
	}
	if !has {
		t.Fatal("expected clients to exist")
	}
}
