package redis_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/gatheryourdeals/data/internal/model"
	redisstore "github.com/gatheryourdeals/data/internal/repository/redis"
)

// newTestStore spins up a miniredis instance and returns a store backed by it.
func newTestStore(t *testing.T) (*redisstore.RefreshTokenStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	store, err := redisstore.NewWithClient(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
	if err != nil {
		t.Fatalf("NewWithClient: %v", err)
	}
	return store, mr
}

func TestRedisRefreshTokenStore_SaveAndFind(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	exp := time.Now().Add(time.Hour)
	if err := store.Save(ctx, "tok-1", "user-1", exp); err != nil {
		t.Fatalf("Save: %v", err)
	}
	userID, err := store.Find(ctx, "tok-1")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("got userID %q, want %q", userID, "user-1")
	}
}

func TestRedisRefreshTokenStore_Find_NotFound(t *testing.T) {
	store, _ := newTestStore(t)
	_, err := store.Find(context.Background(), "nonexistent")
	if !errors.Is(err, model.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRedisRefreshTokenStore_Find_Expired(t *testing.T) {
	store, mr := newTestStore(t)
	ctx := context.Background()

	exp := time.Now().Add(time.Hour)
	if err := store.Save(ctx, "tok-exp", "user-1", exp); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Fast-forward miniredis clock past TTL.
	mr.FastForward(2 * time.Hour)

	_, err := store.Find(ctx, "tok-exp")
	if !errors.Is(err, model.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for expired token, got %v", err)
	}
}

func TestRedisRefreshTokenStore_Delete(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	exp := time.Now().Add(time.Hour)
	if err := store.Save(ctx, "tok-1", "user-1", exp); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Delete(ctx, "tok-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := store.Find(ctx, "tok-1")
	if !errors.Is(err, model.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after delete, got %v", err)
	}
}

func TestRedisRefreshTokenStore_DeleteAllForUser(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	exp := time.Now().Add(time.Hour)
	for _, tok := range []string{"t1", "t2", "t3"} {
		if err := store.Save(ctx, tok, "user-1", exp); err != nil {
			t.Fatalf("Save %q: %v", tok, err)
		}
	}
	if err := store.Save(ctx, "t-other", "user-2", exp); err != nil {
		t.Fatalf("Save t-other: %v", err)
	}

	if err := store.DeleteAllForUser(ctx, "user-1"); err != nil {
		t.Fatalf("DeleteAllForUser: %v", err)
	}

	for _, tok := range []string{"t1", "t2", "t3"} {
		if _, err := store.Find(ctx, tok); !errors.Is(err, model.ErrInvalidToken) {
			t.Errorf("expected %q to be deleted, got %v", tok, err)
		}
	}
	// user-2 token should be untouched.
	if _, err := store.Find(ctx, "t-other"); err != nil {
		t.Errorf("user-2 token should survive, got %v", err)
	}
}

// --- Consume ---

func TestRedisRefreshTokenStore_Consume_Success(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	exp := time.Now().Add(time.Hour)
	if err := store.Save(ctx, "tok-1", "user-1", exp); err != nil {
		t.Fatalf("Save: %v", err)
	}
	userID, err := store.Consume(ctx, "tok-1")
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("got userID %q, want %q", userID, "user-1")
	}
	// Token must be gone after consume.
	if _, err := store.Find(ctx, "tok-1"); !errors.Is(err, model.ErrInvalidToken) {
		t.Errorf("token should be gone after Consume, got %v", err)
	}
}

func TestRedisRefreshTokenStore_Consume_NotFound(t *testing.T) {
	store, _ := newTestStore(t)
	_, err := store.Consume(context.Background(), "nonexistent")
	if !errors.Is(err, model.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRedisRefreshTokenStore_Consume_ConcurrentSameToken(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	exp := time.Now().Add(time.Hour)
	if err := store.Save(ctx, "tok-race", "user-1", exp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	type result struct{ err error }
	results := make(chan result, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	for range 2 {
		go func() {
			defer wg.Done()
			_, e := store.Consume(ctx, "tok-race")
			results <- result{err: e}
		}()
	}
	wg.Wait()
	close(results)

	successes, invalids := 0, 0
	for r := range results {
		if r.err == nil {
			successes++
		} else if errors.Is(r.err, model.ErrInvalidToken) {
			invalids++
		} else {
			t.Errorf("unexpected error: %v", r.err)
		}
	}
	if successes != 1 || invalids != 1 {
		t.Errorf("want 1 success + 1 ErrInvalidToken, got %d + %d", successes, invalids)
	}
}
