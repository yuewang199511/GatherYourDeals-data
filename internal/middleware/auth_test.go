package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/middleware"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTokenService(t *testing.T) (*auth.TokenService, *sqlite.UserRepo) {
	t.Helper()
	db := testutil.NewTestDB(t)
	userRepo := sqlite.NewUserRepo(db)
	refreshStore := sqlite.NewRefreshTokenStore(db)
	tokens := auth.NewTokenService(
		[]byte("test-secret-that-is-long-enough-32c"),
		time.Hour,
		7*24*time.Hour,
		refreshStore,
	)
	return tokens, userRepo
}

func issueToken(t *testing.T, tokens *auth.TokenService, userRepo *sqlite.UserRepo, role model.Role) string {
	t.Helper()
	svc := auth.NewService(userRepo)
	var user *model.User
	var err error
	if role == model.RoleAdmin {
		user, err = svc.CreateAdmin(t.Context(), "testuser", "password123")
	} else {
		user, err = svc.Register(t.Context(), "testuser", "password123")
	}
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	access, _, err := tokens.IssueTokenPair(t.Context(), user)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}
	return access
}

func TestAuth_ValidToken(t *testing.T) {
	tokens, userRepo := newTokenService(t)
	token := issueToken(t, tokens, userRepo, model.RoleUser)

	r := gin.New()
	r.GET("/test", middleware.Auth(tokens), func(c *gin.Context) {
		userID, _ := c.Get(middleware.ContextKeyUserID)
		role, _ := c.Get(middleware.ContextKeyRole)
		c.JSON(http.StatusOK, gin.H{"userID": userID, "role": role})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	tokens, _ := newTokenService(t)

	r := gin.New()
	r.GET("/test", middleware.Auth(tokens), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	tokens, _ := newTokenService(t)

	r := gin.New()
	r.GET("/test", middleware.Auth(tokens), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_MalformedHeader(t *testing.T) {
	tokens, _ := newTokenService(t)

	r := gin.New()
	r.GET("/test", middleware.Auth(tokens), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "NotBearer something")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAdmin_AsAdmin(t *testing.T) {
	tokens, userRepo := newTokenService(t)
	token := issueToken(t, tokens, userRepo, model.RoleAdmin)

	r := gin.New()
	r.GET("/test", middleware.Auth(tokens), middleware.RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireAdmin_AsUser(t *testing.T) {
	tokens, userRepo := newTokenService(t)
	token := issueToken(t, tokens, userRepo, model.RoleUser)

	r := gin.New()
	r.GET("/test", middleware.Auth(tokens), middleware.RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAuth_TamperedClaims(t *testing.T) {
	tokens, userRepo := newTokenService(t)
	// Issue a regular user token
	userToken := issueToken(t, tokens, userRepo, model.RoleUser)

	// A different TokenService with a different secret cannot produce a token
	// that the original service accepts — forging is impossible
	attackerDB := testutil.NewTestDB(t)
	attackerRepo := sqlite.NewUserRepo(attackerDB)
	attackerTokens := auth.NewTokenService(
		[]byte("attacker-secret-long-enough-here!!"),
		time.Hour,
		7*24*time.Hour,
		sqlite.NewRefreshTokenStore(attackerDB),
	)
	attackerSvc := auth.NewService(attackerRepo)
	attackerUser, err := attackerSvc.Register(context.Background(), "attacker", "password123")
	if err != nil {
		t.Fatalf("failed to create attacker user: %v", err)
	}
	// Manually set role to admin to simulate a privilege escalation attempt
	attackerUser.Role = model.RoleAdmin
	attackerToken, _, err := attackerTokens.IssueTokenPair(context.Background(), attackerUser)
	if err != nil {
		t.Fatalf("failed to issue attacker token: %v", err)
	}

	// The original service should reject the attacker's token
	_, err = tokens.ValidateAccessToken(attackerToken)
	if err == nil {
		t.Fatal("expected rejection of token signed with wrong secret, got nil")
	}

	// And the middleware should reject it too
	r := gin.New()
	r.GET("/test", middleware.Auth(tokens), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for _, tok := range []string{userToken + "tampered", attackerToken, "eyJhbGciOiJub25lIn0.eyJ1aWQiOiJoYWNrZXIiLCJyb2xlIjoiYWRtaW4ifQ."} {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("token %q: expected 401, got %d", tok[:20], w.Code)
		}
	}
}

func TestRequireAdmin_NoRole(t *testing.T) {
	r := gin.New()
	r.GET("/test", middleware.RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
