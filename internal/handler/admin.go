package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gatheryourdeals/data/internal/repository"
)

// AdminHandler handles HTTP requests for admin-only endpoints.
type AdminHandler struct {
	users repository.UserRepository
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(users repository.UserRepository) *AdminHandler {
	return &AdminHandler{users: users}
}

// ListUsers handles GET /api/v1/admin/users
// Returns all registered users. Admin only.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.users.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// DeleteUser handles DELETE /api/v1/admin/users/:id
// Deletes a user and revokes all their refresh tokens. Admin only.
func (h *AdminHandler) DeleteUser(c *gin.Context) {
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
