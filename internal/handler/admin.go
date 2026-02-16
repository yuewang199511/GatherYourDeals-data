package handler

import (
	"net/http"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/gin-gonic/gin"
)

// AdminHandler handles HTTP requests for admin-only endpoints.
type AdminHandler struct {
	clients repository.ClientRepository
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(clients repository.ClientRepository) *AdminHandler {
	return &AdminHandler{clients: clients}
}

type createClientRequest struct {
	ID     string `json:"id" binding:"required"`
	Secret string `json:"secret"`
	Domain string `json:"domain"`
}

// CreateClient handles POST /api/v1/admin/clients
// Creates a new OAuth2 client. Admin only.
func (h *AdminHandler) CreateClient(c *gin.Context) {
	var req createClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.clients.GetClientByID(c.Request.Context(), req.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check client"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "client ID already exists"})
		return
	}

	client := &model.OAuthClient{
		ID:     req.ID,
		Secret: req.Secret,
		Domain: req.Domain,
	}
	if err := h.clients.CreateClient(c.Request.Context(), client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create client"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        client.ID,
		"domain":    client.Domain,
		"createdAt": client.CreatedAt,
	})
}

// ListClients handles GET /api/v1/admin/clients
// Returns all registered OAuth2 clients. Admin only.
func (h *AdminHandler) ListClients(c *gin.Context) {
	clients, err := h.clients.ListClients(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}
	c.JSON(http.StatusOK, clients)
}

// DeleteClient handles DELETE /api/v1/admin/clients/:id
// Revokes an OAuth2 client. Admin only.
func (h *AdminHandler) DeleteClient(c *gin.Context) {
	clientID := c.Param("id")

	existing, err := h.clients.GetClientByID(c.Request.Context(), clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check client"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	if err := h.clients.DeleteClient(c.Request.Context(), clientID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete client"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "client revoked"})
}
