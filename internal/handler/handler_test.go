package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/config"
	"github.com/gatheryourdeals/data/internal/handler"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
	"github.com/gin-gonic/gin"
	oauth2 "github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/manage"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type testEnv struct {
	router      *gin.Engine
	userRepo    *sqlite.UserRepo
	clientRepo  *sqlite.ClientRepo
	authService *auth.Service
	oauthMgr    *manage.Manager
}

func setupEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	userRepo := sqlite.NewUserRepo(db)
	clientRepo := sqlite.NewClientRepo(db)
	ctx := context.Background()

	// Seed a test client
	clientRepo.CreateClient(ctx, &model.OAuthClient{ID: "test-client", Secret: "", Domain: ""})

	authService := auth.NewService(userRepo)

	cfg := &config.Config{
		OAuth2: config.OAuth2Config{
			AccessTokenExp:  "1h",
			RefreshTokenExp: "168h",
		},
	}
	oauthMgr, err := auth.NewOAuthManager(cfg, clientRepo)
	if err != nil {
		t.Fatalf("failed to create oauth manager: %v", err)
	}
	oauthSrv := auth.NewOAuthServer(oauthMgr, authService)

	authHandler := handler.NewAuthHandler(authService, oauthSrv, clientRepo)
	adminHandler := handler.NewAdminHandler(clientRepo)
	r := handler.NewRouter(authHandler, adminHandler, oauthMgr, userRepo)

	return &testEnv{
		router:      r,
		userRepo:    userRepo,
		clientRepo:  clientRepo,
		authService: authService,
		oauthMgr:    oauthMgr,
	}
}

// getAdminToken creates an admin user and returns a valid access token.
func (e *testEnv) getAdminToken(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	user, err := e.authService.CreateAdmin(ctx, "admin", "adminpass1")
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	tgr := &oauth2.TokenGenerateRequest{ClientID: "test-client", UserID: user.ID}
	ti, err := e.oauthMgr.GenerateAccessToken(ctx, oauth2.PasswordCredentials, tgr)
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}
	return ti.GetAccess()
}

// getUserToken creates a regular user and returns a valid access token.
func (e *testEnv) getUserToken(t *testing.T, username, password string) string {
	t.Helper()
	ctx := context.Background()
	user, err := e.authService.Register(ctx, username, password)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}
	tgr := &oauth2.TokenGenerateRequest{ClientID: "test-client", UserID: user.ID}
	ti, err := e.oauthMgr.GenerateAccessToken(ctx, oauth2.PasswordCredentials, tgr)
	if err != nil {
		t.Fatalf("failed to generate user token: %v", err)
	}
	return ti.GetAccess()
}

func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	return bytes.NewBuffer(b)
}

// --- Register tests ---

func TestRegister_Success(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{
		"username": "alice",
		"password": "password123",
		"clientId": "test-client",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["username"] != "alice" {
		t.Errorf("expected username 'alice', got '%s'", resp["username"])
	}
	if resp["role"] != "user" {
		t.Errorf("expected role 'user', got '%s'", resp["role"])
	}
}

func TestRegister_InvalidClient(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{
		"username": "alice",
		"password": "password123",
		"clientId": "nonexistent-client",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{
		"username": "alice", "password": "password123", "clientId": "test-client",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	// Register again with same username
	body = jsonBody(t, map[string]string{
		"username": "alice", "password": "password456", "clientId": "test-client",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{
		"username": "alice", "password": "short", "clientId": "test-client",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Logout tests ---

func TestLogout_Success(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/oauth/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogout_NoToken(t *testing.T) {
	env := setupEnv(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/oauth/sessions", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- Admin client management tests ---

func TestAdminCreateClient(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	body := jsonBody(t, map[string]string{
		"id": "new-client", "secret": "s3cret", "domain": "https://example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/clients", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminCreateClient_Duplicate(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	body := jsonBody(t, map[string]string{"id": "test-client"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/clients", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminListClients(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/clients", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var clients []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &clients)
	if len(clients) < 1 {
		t.Fatal("expected at least 1 client")
	}
}

func TestAdminDeleteClient(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	// Create a client to delete
	ctx := context.Background()
	env.clientRepo.CreateClient(ctx, &model.OAuthClient{ID: "to-delete", Secret: "", Domain: ""})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/clients/to-delete", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminDeleteClient_NotFound(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/clients/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminEndpoints_ForbiddenForRegularUser(t *testing.T) {
	env := setupEnv(t)
	// Need an admin to exist first (HasAdmin check), then create a regular user
	env.getAdminToken(t) // creates admin as side effect
	userToken := env.getUserToken(t, "alice", "password123")

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/clients"},
		{http.MethodPost, "/api/v1/admin/clients"},
		{http.MethodDelete, "/api/v1/admin/clients/test-client"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d", ep.method, ep.path, w.Code)
		}
	}
}
