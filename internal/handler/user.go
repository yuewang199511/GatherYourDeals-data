package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

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
// Returns a paginated list of all registered users.
func (h *UserHandler) ListUsers(c *gin.Context) {
	params, err := parsePaginationParams(c, "created_at", "", userSortFields)
	if err != nil {
		return
	}

	page, err := h.users.ListUsers(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, page)
}

// DeleteUser handles DELETE /api/v1/users/:id — admin only.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	if err := h.users.DeleteUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}
