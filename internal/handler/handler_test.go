package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/handler"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type testEnv struct {
	router      *gin.Engine
	userRepo    *sqlite.UserRepo
	metaRepo    *sqlite.MetaFieldRepo
	receiptRepo *sqlite.ReceiptRepo
	authService *auth.Service
	tokens      *auth.TokenService
}

func setupEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	userRepo := sqlite.NewUserRepo(db)
	refreshStore := sqlite.NewRefreshTokenStore(db)
	metaRepo := sqlite.NewMetaFieldRepo(db)
	receiptRepo := sqlite.NewReceiptRepo(db, metaRepo)

	authService := auth.NewService(userRepo)
	tokens := auth.NewTokenService(
		[]byte("test-secret-that-is-long-enough-32c"),
		time.Hour,
		7*24*time.Hour,
		refreshStore,
	)

	authHandler := handler.NewAuthHandler(authService, tokens)
	userHandler := handler.NewUserHandler(userRepo)
	metaHandler := handler.NewMetaHandler(metaRepo)
	receiptHandler := handler.NewReceiptHandler(receiptRepo)
	r := handler.NewRouter(authHandler, userHandler, metaHandler, receiptHandler, tokens, nil)

	return &testEnv{
		router:      r,
		userRepo:    userRepo,
		metaRepo:    metaRepo,
		receiptRepo: receiptRepo,
		authService: authService,
		tokens:      tokens,
	}
}

// getAdminToken creates an admin user and returns a valid access token.
func (e *testEnv) getAdminToken(t *testing.T) string {
	t.Helper()
	user, err := e.authService.CreateAdmin(context.Background(), "admin", "adminpass1")
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	access, _, err := e.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to issue admin token: %v", err)
	}
	return access
}

// getUserToken creates a regular user and returns a valid access token.
func (e *testEnv) getUserToken(t *testing.T, username, password string) string {
	t.Helper()
	user, err := e.authService.Register(context.Background(), username, password)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}
	access, _, err := e.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to issue user token: %v", err)
	}
	return access
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
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["username"] != "alice" {
		t.Errorf("expected username 'alice', got '%v'", resp["username"])
	}
	if resp["role"] != "user" {
		t.Errorf("expected role 'user', got '%v'", resp["role"])
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{"username": "alice", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), req)

	body = jsonBody(t, map[string]string{"username": "alice", "password": "password456"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{"username": "alice", "password": "short"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Login tests ---

func TestLogin_Success(t *testing.T) {
	env := setupEnv(t)
	if _, err := env.authService.Register(context.Background(), "alice", "password123"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	body := jsonBody(t, map[string]string{"username": "alice", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["access_token"] == "" || resp["access_token"] == nil {
		t.Error("expected non-empty access_token")
	}
	if resp["refresh_token"] == "" || resp["refresh_token"] == nil {
		t.Error("expected non-empty refresh_token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	env := setupEnv(t)
	if _, err := env.authService.Register(context.Background(), "alice", "password123"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	body := jsonBody(t, map[string]string{"username": "alice", "password": "wrongpassword"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Logout tests ---

func TestLogout_Success(t *testing.T) {
	env := setupEnv(t)
	user, err := env.authService.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}
	access, _, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	body := jsonBody(t, map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", body)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogout_NoToken(t *testing.T) {
	env := setupEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- Me tests ---

func TestMe_Success(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["role"] != string(model.RoleUser) {
		t.Errorf("expected role 'user', got '%v'", resp["role"])
	}
}

// --- Refresh tests ---

func TestRefresh_Success(t *testing.T) {
	env := setupEnv(t)
	user, err := env.authService.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	body := jsonBody(t, map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Error("expected non-empty access_token")
	}
	newRefresh, _ := resp["refresh_token"].(string)
	if newRefresh == "" {
		t.Error("expected non-empty refresh_token")
	}
	if newRefresh == refresh {
		t.Error("expected rotated refresh token, got same value")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{"refresh_token": "not-a-real-token"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRefresh_ConsumedToken(t *testing.T) {
	env := setupEnv(t)
	user, err := env.authService.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	// Use the refresh token once
	body := jsonBody(t, map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), req)

	// Using it again should fail
	body = jsonBody(t, map[string]string{"refresh_token": refresh})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on reuse of consumed token, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Me tests (additional) ---

func TestMe_NoToken(t *testing.T) {
	env := setupEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- Login tests (additional) ---

func TestLogin_MissingFields(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{"username": "alice"}) // missing password
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{"username": "ghost", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Logout tests (additional) ---

func TestLogout_RefreshTokenRevokedAfterLogout(t *testing.T) {
	env := setupEnv(t)
	user, err := env.authService.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	access, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	// Logout
	body := jsonBody(t, map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", body)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), req)

	// Try to use the refresh token — should now be rejected
	body = jsonBody(t, map[string]string{"refresh_token": refresh})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Admin user management tests ---

func TestAdminListUsers(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)
	env.getUserToken(t, "alice", "password123") // create a regular user too

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(data))
	}
}

func TestAdminDeleteUser(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	user, err := env.authService.Register(context.Background(), "todelete", "password123")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+user.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminDeleteUser_NotFound(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/nonexistent-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminDeleteUser_RevokesRefreshTokens(t *testing.T) {
	env := setupEnv(t)
	adminToken := env.getAdminToken(t)

	user, err := env.authService.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	_, refresh, err := env.tokens.IssueTokenPair(context.Background(), user)
	if err != nil {
		t.Fatalf("IssueTokenPair failed: %v", err)
	}

	// Delete the user
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+user.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	env.router.ServeHTTP(httptest.NewRecorder(), req)

	// The deleted user's refresh token should no longer work
	body := jsonBody(t, map[string]string{"refresh_token": refresh})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after user deletion, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminEndpoints_ForbiddenForRegularUser(t *testing.T) {
	env := setupEnv(t)
	env.getAdminToken(t) // ensure admin exists
	userToken := env.getUserToken(t, "alice", "password123")

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/users"},
		{http.MethodDelete, "/api/v1/users/some-id"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d", ep.method, ep.path, w.Code)
		}
	}
}

// ===========================================================================
// Meta field handler tests
// ===========================================================================

func TestMeta_ListFields(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 7 {
		t.Errorf("expected 7 native fields, got %d", len(data))
	}
}

func TestMeta_ListFields_Unauthenticated(t *testing.T) {
	env := setupEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMeta_CreateField(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	body := jsonBody(t, map[string]string{
		"fieldName":   "brand",
		"description": "brand of the product",
		"type":        "string",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meta", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["fieldName"] != "brand" {
		t.Errorf("expected fieldName 'brand', got %v", resp["fieldName"])
	}
}

func TestMeta_CreateField_Duplicate(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	body := jsonBody(t, map[string]string{
		"fieldName":   "brand",
		"description": "brand of the product",
		"type":        "string",
	})

	// First create
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meta", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), req)

	// Duplicate
	body = jsonBody(t, map[string]string{
		"fieldName":   "brand",
		"description": "duplicate",
		"type":        "string",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/meta", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMeta_CreateField_MissingFields(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	body := jsonBody(t, map[string]string{"fieldName": "brand"}) // missing description and type
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meta", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMeta_CreateField_Unauthenticated(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]string{
		"fieldName":   "brand",
		"description": "brand of the product",
		"type":        "string",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meta", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMeta_UpdateDescription(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	// Create a field first (any authenticated user can do this)
	body := jsonBody(t, map[string]string{
		"fieldName":   "brand",
		"description": "original",
		"type":        "string",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meta", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), req)

	// Update
	body = jsonBody(t, map[string]string{"description": "updated description"})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/meta/brand", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMeta_UpdateDescription_NotFound(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	body := jsonBody(t, map[string]string{"description": "nope"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meta/nonexistent", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMeta_UpdateDescription_ForbiddenForRegularUser(t *testing.T) {
	env := setupEnv(t)
	env.getAdminToken(t)
	userToken := env.getUserToken(t, "alice", "password123")

	body := jsonBody(t, map[string]string{"description": "nope"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meta/productName", body)
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// ===========================================================================
// Receipt handler tests
// ===========================================================================

func TestReceipt_Create(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	body := jsonBody(t, map[string]interface{}{
		"productName":  "Milk 2%",
		"purchaseDate": "2025.04.05",
		"price":        "5.49CAD",
		"amount":       "1",
		"storeName":    "Costco",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["productName"] != "Milk 2%" {
		t.Errorf("expected productName 'Milk 2%%', got %v", resp["productName"])
	}
	if resp["userId"] == nil || resp["userId"] == "" {
		t.Error("expected userId to be set")
	}
	if resp["uploadTime"] == nil || resp["uploadTime"].(float64) == 0 {
		t.Error("expected uploadTime to be set")
	}
}

func TestReceipt_Create_WithExtras(t *testing.T) {
	env := setupEnv(t)
	userToken := env.getUserToken(t, "alice", "password123")

	// User registers the custom field
	body := jsonBody(t, map[string]string{
		"fieldName":   "brand",
		"description": "brand of the product",
		"type":        "string",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meta", body)
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), req)

	// User creates a receipt with the extra field — flat, no "extras" wrapper
	body = jsonBody(t, map[string]interface{}{
		"productName":  "Milk 2%",
		"purchaseDate": "2025.04.05",
		"price":        "5.49CAD",
		"amount":       "1",
		"storeName":    "Costco",
		"brand":        "Kirkland",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// The response should contain "brand" at the top level
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["brand"] != "Kirkland" {
		t.Errorf("expected brand 'Kirkland' at top level, got %v", resp["brand"])
	}
	if _, hasExtras := resp["extras"]; hasExtras {
		t.Error("expected no 'extras' key in response, but found one")
	}
}

func TestReceipt_Create_UnregisteredExtra(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	body := jsonBody(t, map[string]interface{}{
		"productName":  "Milk 2%",
		"purchaseDate": "2025.04.05",
		"price":        "5.49CAD",
		"amount":       "1",
		"storeName":    "Costco",
		"unknownField": "value",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceipt_Create_MissingFields(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	body := jsonBody(t, map[string]string{"productName": "Milk"}) // missing required fields
	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceipt_Create_Unauthenticated(t *testing.T) {
	env := setupEnv(t)

	body := jsonBody(t, map[string]interface{}{
		"productName":  "Milk",
		"purchaseDate": "2025.04.05",
		"price":        "5.49CAD",
		"amount":       "1",
		"storeName":    "Costco",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestReceipt_GetByID(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	// Create a receipt
	body := jsonBody(t, map[string]interface{}{
		"productName":  "Milk 2%",
		"purchaseDate": "2025.04.05",
		"price":        "5.49CAD",
		"amount":       "1",
		"storeName":    "Costco",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var created map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	id := created["id"].(string)

	// Get it back
	req = httptest.NewRequest(http.MethodGet, "/api/v1/receipts/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["productName"] != "Milk 2%" {
		t.Errorf("expected 'Milk 2%%', got %v", resp["productName"])
	}
}

func TestReceipt_GetByID_NotFound(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceipt_List(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	// Create two receipts
	for _, name := range []string{"Milk", "Bread"} {
		body := jsonBody(t, map[string]interface{}{
			"productName":  name,
			"purchaseDate": "2025.04.05",
			"price":        "3.00CAD",
			"amount":       "1",
			"storeName":    "Costco",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		env.router.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("expected 2 receipts, got %d", len(data))
	}
}

func TestReceipt_List_Empty(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("expected empty list, got %d", len(data))
	}
}

func TestReceipt_List_OnlyOwnReceipts(t *testing.T) {
	env := setupEnv(t)
	aliceToken := env.getUserToken(t, "alice", "password123")
	bobToken := env.getUserToken(t, "bob", "password123")

	// Alice creates a receipt
	body := jsonBody(t, map[string]interface{}{
		"productName":  "Milk",
		"purchaseDate": "2025.04.05",
		"price":        "5.49CAD",
		"amount":       "1",
		"storeName":    "Costco",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Authorization", "Bearer "+aliceToken)
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), req)

	// Bob should see an empty list
	req = httptest.NewRequest(http.MethodGet, "/api/v1/receipts", nil)
	req.Header.Set("Authorization", "Bearer "+bobToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("expected 0 receipts for bob, got %d", len(data))
	}
}

func TestReceipt_Delete(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	// Create
	body := jsonBody(t, map[string]interface{}{
		"productName":  "Milk",
		"purchaseDate": "2025.04.05",
		"price":        "5.49CAD",
		"amount":       "1",
		"storeName":    "Costco",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var created map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	id := created["id"].(string)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/receipts/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	req = httptest.NewRequest(http.MethodGet, "/api/v1/receipts/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

func TestReceipt_Delete_NotFound(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/receipts/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ===========================================================================
// Receipt pagination tests (T012)
// ===========================================================================

func createReceipt(t *testing.T, env *testEnv, token, productName, date string) {
	t.Helper()
	body := jsonBody(t, map[string]interface{}{
		"productName":  productName,
		"purchaseDate": date,
		"price":        "5.49CAD",
		"amount":       "1",
		"storeName":    "Costco",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create receipt: %d %s", w.Code, w.Body.String())
	}
}

func TestReceipt_Pagination_Envelope(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")
	createReceipt(t, env, token, "Milk", "2025.01.01")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected 'data' key in response")
	}
	if _, ok := resp["total"]; !ok {
		t.Error("expected 'total' key in response")
	}
	if _, ok := resp["offset"]; !ok {
		t.Error("expected 'offset' key in response")
	}
	if _, ok := resp["limit"]; !ok {
		t.Error("expected 'limit' key in response")
	}
	if _, ok := resp["total_pages"]; !ok {
		t.Error("expected 'total_pages' key in response")
	}
}

func TestReceipt_Pagination_LimitOffset(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")
	for i, name := range []string{"Milk", "Bread", "Eggs", "Butter", "Juice", "Cheese", "Yogurt"} {
		createReceipt(t, env, token, name, "2025.01.01")
		_ = i
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts?limit=5&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 5 {
		t.Errorf("expected 5 items in page, got %d", len(data))
	}
	if resp["total"].(float64) != 7 {
		t.Errorf("expected total 7, got %v", resp["total"])
	}
	if resp["total_pages"].(float64) != 2 {
		t.Errorf("expected total_pages 2, got %v", resp["total_pages"])
	}
}

func TestReceipt_Pagination_SortByPurchaseDate(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")
	createReceipt(t, env, token, "A", "2025.03.01")
	createReceipt(t, env, token, "B", "2025.01.01")
	createReceipt(t, env, token, "C", "2025.02.01")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts?sort_by=purchase_date&sort_order=asc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 3 {
		t.Fatalf("expected 3 receipts, got %d", len(data))
	}
	first := data[0].(map[string]interface{})
	if first["purchaseDate"] != "2025.01.01" {
		t.Errorf("expected first purchaseDate '2025.01.01', got %v", first["purchaseDate"])
	}
}

func TestReceipt_Pagination_InvalidSortBy(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts?sort_by=invalid_field", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceipt_Pagination_InvalidLimit(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts?limit=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceipt_Pagination_InvalidSortOrder(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts?sort_order=sideways", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceipt_Pagination_LimitCappedAt100(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts?limit=500", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["limit"].(float64) != 100 {
		t.Errorf("expected limit capped at 100, got %v", resp["limit"])
	}
}

// ===========================================================================
// User pagination tests (T015)
// ===========================================================================

func TestAdminListUsers_Pagination_Envelope(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	for _, key := range []string{"data", "total", "offset", "limit", "total_pages"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("expected '%s' key in response", key)
		}
	}
}

func TestAdminListUsers_Pagination_LimitOffset(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)
	env.getUserToken(t, "alice", "password123")
	env.getUserToken(t, "bob", "password456")

	// 3 users total (admin + alice + bob); request limit=2&offset=0
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=2&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("expected 2 users in page, got %d", len(data))
	}
	if resp["total"].(float64) != 3 {
		t.Errorf("expected total 3, got %v", resp["total"])
	}
}

func TestAdminListUsers_Pagination_SortByUsername(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)
	env.getUserToken(t, "zebra", "password123")
	env.getUserToken(t, "apple", "password456")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?sort_by=username&sort_order=asc&limit=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) < 2 {
		t.Fatalf("expected at least 2 users, got %d", len(data))
	}
	first := data[0].(map[string]interface{})["username"].(string)
	last := data[len(data)-1].(map[string]interface{})["username"].(string)
	if first >= last {
		t.Errorf("expected ascending sort, got first=%q last=%q", first, last)
	}
}

func TestAdminListUsers_Pagination_InvalidSortBy(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?sort_by=email", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminListUsers_Pagination_InvalidLimit(t *testing.T) {
	env := setupEnv(t)
	token := env.getAdminToken(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ===========================================================================
// Meta pagination tests (T018)
// ===========================================================================

func TestMeta_Pagination_Envelope_AscDefault(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	for _, key := range []string{"data", "total", "offset", "limit", "total_pages"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("expected '%s' key in response", key)
		}
	}
	// Verify default ASC ordering: first name alphabetically should be 'amount'
	data := resp["data"].([]interface{})
	if len(data) < 2 {
		t.Fatalf("expected at least 2 fields, got %d", len(data))
	}
	first := data[0].(map[string]interface{})["fieldName"].(string)
	second := data[1].(map[string]interface{})["fieldName"].(string)
	if first >= second {
		t.Errorf("expected ASC order by default, got %q then %q", first, second)
	}
}

func TestMeta_Pagination_LimitOffset(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	// 7 native fields seeded; request limit=3&offset=3 → 3 items, total=7, total_pages=3
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta?limit=3&offset=3", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 3 {
		t.Errorf("expected 3 items in page, got %d", len(data))
	}
	if resp["total"].(float64) != 7 {
		t.Errorf("expected total 7, got %v", resp["total"])
	}
	if resp["total_pages"].(float64) != 3 {
		t.Errorf("expected total_pages 3, got %v", resp["total_pages"])
	}
}

func TestMeta_Pagination_SortByNameDesc(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta?sort_by=name&sort_order=desc&limit=2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(data))
	}
	first := data[0].(map[string]interface{})["fieldName"].(string)
	second := data[1].(map[string]interface{})["fieldName"].(string)
	if first <= second {
		t.Errorf("expected DESC order, got %q then %q", first, second)
	}
}

func TestMeta_Pagination_InvalidSortBy(t *testing.T) {
	env := setupEnv(t)
	token := env.getUserToken(t, "alice", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta?sort_by=created_at", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
