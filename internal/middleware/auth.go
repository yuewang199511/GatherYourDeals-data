package middleware

import (
	"net/http"
	"strings"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4/manage"
)

const (
	// ContextKeyUserID is the gin context key for the authenticated user's ID.
	ContextKeyUserID = "userID"
	// ContextKeyRole is the gin context key for the authenticated user's role.
	ContextKeyRole = "userRole"
)

// Auth returns a gin middleware that validates the Bearer token
// using the go-oauth2 manager, and looks up the user's role from the repository.
func Auth(manager *manage.Manager, users repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}
		accessToken := parts[1]

		// Validate the token via go-oauth2 manager.
		tokenInfo, err := manager.LoadAccessToken(c.Request.Context(), accessToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		userID := tokenInfo.GetUserID()

		// Look up the user's role from the database.
		user, err := users.GetUserByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeyRole, user.Role)
		c.Next()
	}
}

// RequireAdmin returns a gin middleware that rejects non-admin users.
// Must be used after the Auth middleware.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "role not set in context"})
			return
		}
		if role.(model.Role) != model.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}
