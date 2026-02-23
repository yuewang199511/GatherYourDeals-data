package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

// --- helpers ---

type tokenTestEnv struct {
	tokens *auth.TokenService
	store  *sqlite.RefreshTokenStore
	svc    *auth.Service
}

func newTestTokenEnv(t *testing.T) *tokenTestEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	store := sqlite.NewRefreshTokenStore(db)
	tokens := auth.NewTokenService(
		[]byte("test-secret-that-is-long-enough-32c"),
		time.Hour,
		7*24*time.Hour,
		store,
	)
	svc := auth.NewService(sqlite.NewUserRepo(db))
	return &tokenTestEnv{tokens: tokens, store: store, svc: svc}
}

// newSavedUser creates a user in the DB and returns it.
func newSavedUser(t *testing.T, svc *auth.Service, role model.Role) *model.User {
	t.Helper()
	var user *model.User
	var err error
	if role == model.RoleAdmin {
		user, err = svc.CreateAdmin(context.Background(), "alice", "password123")
	} else {
		user, err = svc.Register(context.Background(), "alice", "password123")
	}
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

// --- IssueTokenPair ---

func TestIssueTokenPair_ReturnsNonEmptyTokens(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleUser)

	access, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}
	if access == "" {
		t.Error("expected non-empty access token")
	}
	if refresh == "" {
		t.Error("expected non-empty refresh token")
	}
	if access == refresh {
		t.Error("access and refresh tokens should be different")
	}
}

func TestIssueTokenPair_AccessTokenContainsClaims(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleAdmin)

	access, _, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	claims, err := env.tokens.ValidateAccessToken(access)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected userID %q, got %q", user.ID, claims.UserID)
	}
	if claims.Role != model.RoleAdmin {
		t.Errorf("expected role admin, got %q", claims.Role)
	}
}

func TestIssueTokenPair_RefreshTokenStoredInDB(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleUser)

	_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	userID, err := env.store.Find(context.Background(), refresh)
	if err != nil {
		t.Fatalf("expected refresh token in store, got error: %v", err)
	}
	if userID != user.ID {
		t.Errorf("expected userID %q, got %q", user.ID, userID)
	}
}

// --- ValidateAccessToken ---

func TestValidateAccessToken_Valid(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleUser)

	access, _, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	claims, err := env.tokens.ValidateAccessToken(access)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected userID %q, got %q", user.ID, claims.UserID)
	}
}

func TestValidateAccessToken_RandomString(t *testing.T) {
	env := newTestTokenEnv(t)

	_, err := env.tokens.ValidateAccessToken("not-a-token")
	if err == nil {
		t.Fatal("expected error for random string, got nil")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleUser)

	access, _, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	otherDB := testutil.NewTestDB(t)
	otherTokens := auth.NewTokenService(
		[]byte("completely-different-secret-32chars!"),
		time.Hour,
		7*24*time.Hour,
		sqlite.NewRefreshTokenStore(otherDB),
	)

	_, err = otherTokens.ValidateAccessToken(access)
	if err == nil {
		t.Fatal("expected error when validating token with wrong secret")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	env := newTestTokenEnv(t)
	// Override with an already-expired access duration
	expiredTokens := auth.NewTokenService(
		[]byte("test-secret-that-is-long-enough-32c"),
		-time.Second,
		7*24*time.Hour,
		env.store,
	)
	user := newSavedUser(t, env.svc, model.RoleUser)

	access, _, err := expiredTokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	_, err = expiredTokens.ValidateAccessToken(access)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

// --- RefreshAccessToken ---

func TestRefreshAccessToken_Success(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleUser)

	_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	newAccess, newRefresh, err := env.tokens.RefreshAccessToken(context.Background(), refresh, env.svc)
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}
	if newAccess == "" {
		t.Error("expected non-empty new access token")
	}
	if newRefresh == "" {
		t.Error("expected non-empty new refresh token")
	}
	if newRefresh == refresh {
		t.Error("new refresh token should differ from old one (rotation)")
	}
}

func TestRefreshAccessToken_OldTokenRevoked(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleUser)

	_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	if _, _, err := env.tokens.RefreshAccessToken(context.Background(), refresh, env.svc); err != nil {
		t.Fatalf("first RefreshAccessToken failed: %v", err)
	}

	// Using it again should fail (rotation consumed it)
	_, _, err = env.tokens.RefreshAccessToken(context.Background(), refresh, env.svc)
	if err == nil {
		t.Fatal("expected error reusing a consumed refresh token, got nil")
	}
}

func TestRefreshAccessToken_InvalidToken(t *testing.T) {
	env := newTestTokenEnv(t)

	_, _, err := env.tokens.RefreshAccessToken(context.Background(), "not-a-real-token", env.svc)
	if err == nil {
		t.Fatal("expected error for invalid refresh token, got nil")
	}
}

// --- RevokeRefreshToken ---

func TestRevokeRefreshToken_PreventsReuse(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleUser)

	_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	if err := env.tokens.RevokeRefreshToken(context.Background(), refresh); err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}

	_, _, err = env.tokens.RefreshAccessToken(context.Background(), refresh, env.svc)
	if err == nil {
		t.Fatal("expected error using revoked refresh token, got nil")
	}
}

// --- RevokeAllForUser ---

func TestRevokeAllForUser_RevokesAllTokens(t *testing.T) {
	env := newTestTokenEnv(t)
	user := newSavedUser(t, env.svc, model.RoleUser)

	// Issue multiple refresh tokens (simulating multiple devices)
	var refreshTokens []string
	for i := 0; i < 3; i++ {
		_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
		if err != nil {
			t.Fatalf("IssueTokenPair failed: %v", err)
		}
		refreshTokens = append(refreshTokens, refresh)
	}

	if err := env.tokens.RevokeAllForUser(context.Background(), user.ID); err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}

	for _, refresh := range refreshTokens {
		_, _, err := env.tokens.RefreshAccessToken(context.Background(), refresh, env.svc)
		if err == nil {
			t.Error("expected error for revoked token, got nil")
		}
	}
}
