package handler

import (
	"net/http"
	"strings"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/middleware"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles HTTP requests for authentication endpoints.
type AuthHandler struct {
	service *auth.Service
	tokens  *auth.TokenService
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(service *auth.Service, tokens *auth.TokenService) *AuthHandler {
	return &AuthHandler{service: service, tokens: tokens}
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register handles POST /api/v1/users
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

// Login handles POST /api/v1/sessions
// Returns an access token and a refresh token on success.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.service.Login(c.Request.Context(), req.Username, req.Password)
	if err == auth.ErrInvalidCredential {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	access, refresh, err := h.tokens.IssueTokenPair(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue tokens"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "Bearer",
	})
}

// Refresh handles POST /api/v1/sessions/refresh
// Exchanges a valid refresh token for a new access + refresh token pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	access, refresh, err := h.tokens.RefreshAccessToken(c.Request.Context(), req.RefreshToken, h.service)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "Bearer",
	})
}

// Logout handles DELETE /api/v1/sessions
// Revokes the refresh token sent in the request body.
func (h *AuthHandler) Logout(c *gin.Context) {
	_, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// The client must send the refresh token to revoke it.
	// The access token will expire naturally on its own.
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Graceful: if no refresh token provided, just return OK (access token will expire).
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}
	_ = h.tokens.RevokeRefreshToken(c.Request.Context(), req.RefreshToken)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// Me handles GET /api/v1/sessions/me — returns the current user's info from the token.
func (h *AuthHandler) Me(c *gin.Context) {
	header := c.GetHeader("Authorization")
	parts := strings.SplitN(header, " ", 2)
	claims, err := h.tokens.ValidateAccessToken(parts[1])
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":   claims.UserID,
		"role": claims.Role,
	})
}
