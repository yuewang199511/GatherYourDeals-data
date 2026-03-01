package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository"
)

// MetaHandler handles HTTP requests for field metadata endpoints.
type MetaHandler struct {
	meta repository.MetaFieldRepository
}

// NewMetaHandler creates a new meta field handler.
func NewMetaHandler(meta repository.MetaFieldRepository) *MetaHandler {
	return &MetaHandler{meta: meta}
}

type createFieldRequest struct {
	FieldName   string `json:"fieldName" binding:"required"`
	Description string `json:"description" binding:"required"`
	FieldType   string `json:"type" binding:"required"`
}

type updateDescriptionRequest struct {
	Description string `json:"description" binding:"required"`
}

// ListFields handles GET /api/v1/meta
// Returns all registered fields (native + user-defined).
func (h *MetaHandler) ListFields(c *gin.Context) {
	fields, err := h.meta.ListFields(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list fields"})
		return
	}
	c.JSON(http.StatusOK, fields)
}

// CreateField handles POST /api/v1/meta
// Registers a new user-defined field.
func (h *MetaHandler) CreateField(c *gin.Context) {
	var req createFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	field := &model.MetaField{
		FieldName:   req.FieldName,
		Description: req.Description,
		FieldType:   req.FieldType,
		Native:      false,
	}

	if err := h.meta.CreateField(c.Request.Context(), field); err != nil {
		// SQLite returns a UNIQUE constraint error for duplicates.
		c.JSON(http.StatusConflict, gin.H{"error": "field already exists"})
		return
	}

	c.JSON(http.StatusCreated, field)
}

// UpdateDescription handles PUT /api/v1/meta/:fieldName
// Updates the description of an existing field. Admin only.
func (h *MetaHandler) UpdateDescription(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	fieldName := c.Param("fieldName")

	var req updateDescriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.meta.UpdateDescription(c.Request.Context(), fieldName, req.Description); err != nil {
		if errors.Is(err, model.ErrFieldNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "field not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update field"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "description updated"})
}
