package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gatheryourdeals/data/internal/middleware"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository"
)

// UserHandler handles HTTP requests for user management endpoints.
type UserHandler struct {
	users repository.UserRepository
}

// NewUserHandler creates a new user handler.
func NewUserHandler(users repository.UserRepository) *UserHandler {
	return &UserHandler{users: users}
}

// ListUsers handles GET /api/v1/users — admin only.
func (h *UserHandler) ListUsers(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	users, err := h.users.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// DeleteUser handles DELETE /api/v1/users/:id — admin only.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	userID := c.Param("id")

	user, err := h.users.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to look up user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := h.users.DeleteUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

// requireAdmin checks if the current user has admin role.
// Returns false and sends a 403 response if not.
func requireAdmin(c *gin.Context) bool {
	role, exists := c.Get(middleware.ContextKeyRole)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return false
	}
	if role.(model.Role) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return false
	}
	return true
}
