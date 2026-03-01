package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gatheryourdeals/data/internal/middleware"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository"
)

// ReceiptHandler handles HTTP requests for purchase receipt endpoints.
type ReceiptHandler struct {
	receipts repository.ReceiptRepository
}

// NewReceiptHandler creates a new receipt handler.
func NewReceiptHandler(receipts repository.ReceiptRepository) *ReceiptHandler {
	return &ReceiptHandler{receipts: receipts}
}

// CreateReceipt handles POST /api/v1/receipts
// Accepts a flat JSON object. Native fields become columns; the rest go into extras.
func (h *ReceiptHandler) CreateReceipt(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receipt, extras := model.ParseReceiptFromMap(raw)

	// Validate required native fields.
	if receipt.ProductName == "" || receipt.PurchaseDate == "" ||
		receipt.Price == "" || receipt.Amount == "" || receipt.StoreName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "productName, purchaseDate, price, amount, and storeName are required"})
		return
	}

	receipt.ID = uuid.New().String()
	receipt.Extras = extras
	receipt.UserID = userID.(string)

	if err := h.receipts.CreateReceipt(c.Request.Context(), receipt); err != nil {
		if errors.Is(err, model.ErrFieldNotRegistered) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create receipt"})
		return
	}

	c.JSON(http.StatusCreated, receipt)
}

// GetReceipt handles GET /api/v1/receipts/:id
// Returns a single receipt by ID.
func (h *ReceiptHandler) GetReceipt(c *gin.Context) {
	id := c.Param("id")

	receipt, err := h.receipts.GetReceiptByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get receipt"})
		return
	}
	if receipt == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "receipt not found"})
		return
	}

	c.JSON(http.StatusOK, receipt)
}

// ListReceipts handles GET /api/v1/receipts
// Returns all receipts for the authenticated user.
func (h *ReceiptHandler) ListReceipts(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	receipts, err := h.receipts.ListReceiptsByUser(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list receipts"})
		return
	}

	if receipts == nil {
		receipts = []*model.Receipt{}
	}

	c.JSON(http.StatusOK, receipts)
}

// DeleteReceipt handles DELETE /api/v1/receipts/:id
// Deletes a receipt by ID.
func (h *ReceiptHandler) DeleteReceipt(c *gin.Context) {
	id := c.Param("id")

	receipt, err := h.receipts.GetReceiptByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to look up receipt"})
		return
	}
	if receipt == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "receipt not found"})
		return
	}

	if err := h.receipts.DeleteReceipt(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete receipt"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "receipt deleted"})
}
