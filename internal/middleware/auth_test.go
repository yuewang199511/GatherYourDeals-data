package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/config"
	"github.com/gatheryourdeals/data/internal/middleware"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
	"github.com/gin-gonic/gin"
	oauth2 "github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/store"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestEnv creates a test database, user, client, and OAuth2 manager.
// Returns the manager, user repo, and a valid access token for the created user.
func setupTestEnv(t *testing.T, role model.Role) (*manage.Manager, *sqlite.UserRepo, string) {
	t.Helper()
	db := testutil.NewTestDB(t)
	userRepo := sqlite.NewUserRepo(db)
	clientRepo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	// Create a test client
	if err := clientRepo.CreateClient(ctx, &model.OAuthClient{ID: "test-client", Secret: "", Domain: ""}); err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}

	// Create a test user
	authService := auth.NewService(userRepo)
	var user *model.User
	var err error
	if role == model.RoleAdmin {
		user, err = authService.CreateAdmin(ctx, "testuser", "password123")
	} else {
		user, err = authService.Register(ctx, "testuser", "password123")
	}
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create OAuth2 manager and get a token
	cfg := &config.Config{
		OAuth2: config.OAuth2Config{
			AccessTokenExp:  "1h",
			RefreshTokenExp: "168h",
		},
	}
	tokenStore, _ := store.NewMemoryTokenStore()
	oauthManager, err := auth.NewOAuthManager(cfg, clientRepo, tokenStore)
	if err != nil {
		t.Fatalf("failed to create oauth manager: %v", err)
	}
	oauthServer := auth.NewOAuthServer(oauthManager, authService)

	// Issue a token via the password grant
	// We simulate this by calling the manager directly
	tgr := &oauth2.TokenGenerateRequest{
		ClientID: "test-client",
		UserID:   user.ID,
	}
	_ = oauthServer // ensure server is configured
	ti, err := oauthManager.GenerateAccessToken(ctx, oauth2.PasswordCredentials, tgr)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return oauthManager, userRepo, ti.GetAccess()
}

func TestAuth_ValidToken(t *testing.T) {
	oauthManager, userRepo, token := setupTestEnv(t, model.RoleUser)

	r := gin.New()
	r.GET("/test", middleware.Auth(oauthManager, userRepo), func(c *gin.Context) {
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
	oauthManager, userRepo, _ := setupTestEnv(t, model.RoleUser)

	r := gin.New()
	r.GET("/test", middleware.Auth(oauthManager, userRepo), func(c *gin.Context) {
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
	oauthManager, userRepo, _ := setupTestEnv(t, model.RoleUser)

	r := gin.New()
	r.GET("/test", middleware.Auth(oauthManager, userRepo), func(c *gin.Context) {
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
	oauthManager, userRepo, _ := setupTestEnv(t, model.RoleUser)

	r := gin.New()
	r.GET("/test", middleware.Auth(oauthManager, userRepo), func(c *gin.Context) {
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
	oauthManager, userRepo, token := setupTestEnv(t, model.RoleAdmin)

	r := gin.New()
	r.GET("/test", middleware.Auth(oauthManager, userRepo), middleware.RequireAdmin(), func(c *gin.Context) {
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
	oauthManager, userRepo, token := setupTestEnv(t, model.RoleUser)

	r := gin.New()
	r.GET("/test", middleware.Auth(oauthManager, userRepo), middleware.RequireAdmin(), func(c *gin.Context) {
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

func TestRequireAdmin_NoRole(t *testing.T) {
	r := gin.New()
	// Skip Auth middleware — go straight to RequireAdmin with no role set
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
