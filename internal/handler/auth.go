package handler

import (
	"net/http"
	"strings"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/middleware"
	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4/server"
)

// AuthHandler handles HTTP requests for authentication endpoints.
type AuthHandler struct {
	service     *auth.Service
	oauthServer *server.Server
	clients     repository.ClientRepository
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(service *auth.Service, oauthServer *server.Server, clients repository.ClientRepository) *AuthHandler {
	return &AuthHandler{service: service, oauthServer: oauthServer, clients: clients}
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	ClientID string `json:"clientId" binding:"required"`
}

// Register handles POST /api/v1/users
// Creates a new user resource. Requires a valid client_id to prevent unauthorized account creation.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.clients.GetClientByID(c.Request.Context(), req.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate client"})
		return
	}
	if client == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid client_id"})
		return
	}

	user, err := h.service.Register(c.Request.Context(), req.Username, req.Password)
	if err == auth.ErrUsernameExists {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

// Token handles POST /api/v1/oauth/token
// This is the standard OAuth2 token endpoint.
// It handles both password credentials grant (login) and refresh token grant.
// For login: grant_type=password&username=xxx&password=xxx&client_id=<id>
// For refresh: grant_type=refresh_token&refresh_token=xxx&client_id=<id>
// If the client has a secret, it must also include client_secret.
// Clients are managed via the admin API and stored in the database.
func (h *AuthHandler) Token(c *gin.Context) {
	err := h.oauthServer.HandleTokenRequest(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// Logout handles DELETE /api/v1/oauth/sessions
// Deletes the current session by revoking the access token.
func (h *AuthHandler) Logout(c *gin.Context) {
	_, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Extract the access token from the Authorization header.
	// The Auth middleware already validated the format, so we know it's "Bearer <token>".
	header := c.GetHeader("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse authorization header"})
		return
	}

	err := h.oauthServer.Manager.RemoveAccessToken(c.Request.Context(), parts[1])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
