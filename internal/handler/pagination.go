package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gatheryourdeals/data/internal/model"
)

const (
	defaultLimit     = 20
	maxLimit         = 100
	defaultSortOrder = "DESC"
)

// receiptSortFields maps API sort_by values to receipt DB column names.
// "created_at" maps to "upload_time" — the column used as the insertion timestamp.
var receiptSortFields = map[string]string{
	"purchase_date": "purchase_date",
	"price":         "price",
	"store_name":    "store_name",
	"product_name":  "product_name",
	"created_at":    "upload_time",
}

// userSortFields maps API sort_by values to user DB column names.
// Note: "email" is intentionally absent — the users table has no email column.
var userSortFields = map[string]string{
	"username":   "username",
	"role":       "role",
	"created_at": "created_at",
}

// metaSortFields maps API sort_by values to meta_fields DB column names.
// Note: "created_at" is intentionally absent — meta_fields has no timestamp column.
var metaSortFields = map[string]string{
	"name": "field_name",
}

// parsePaginationParams parses and validates the four pagination query parameters
// (offset, limit, sort_by, sort_order) from the request.
//
// defaultSortBy must already be a DB column name (post-mapping).
// allowedSortFields maps API param values → DB column names.
//
// On validation error, this function writes a 400 JSON response and returns a
// non-nil error; the caller must return immediately without writing further output.
func parsePaginationParams(
	c *gin.Context,
	defaultSortBy string,
	defaultSortOrderOverride string,
	allowedSortFields map[string]string,
) (model.PaginationParams, error) {
	// --- offset ---
	offset := 0
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be a non-negative integer"})
			return model.PaginationParams{}, fmt.Errorf("invalid offset")
		}
		offset = v
	}

	// --- limit ---
	limit := defaultLimit
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return model.PaginationParams{}, fmt.Errorf("invalid limit")
		}
		if v > maxLimit {
			v = maxLimit
		}
		limit = v
	}

	// --- sort_order ---
	sortOrder := defaultSortOrder
	if defaultSortOrderOverride != "" {
		sortOrder = defaultSortOrderOverride
	}
	if raw := c.Query("sort_order"); raw != "" {
		upper := strings.ToUpper(raw)
		if upper != "ASC" && upper != "DESC" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sort_order: allowed values are asc, desc"})
			return model.PaginationParams{}, fmt.Errorf("invalid sort_order")
		}
		sortOrder = upper
	}

	// --- sort_by ---
	sortBy := defaultSortBy
	if raw := c.Query("sort_by"); raw != "" {
		col, ok := allowedSortFields[raw]
		if !ok {
			allowed := make([]string, 0, len(allowedSortFields))
			for k := range allowedSortFields {
				allowed = append(allowed, k)
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid sort_by: allowed values are %s", strings.Join(allowed, ", ")),
			})
			return model.PaginationParams{}, fmt.Errorf("invalid sort_by")
		}
		sortBy = col
	}

	return model.PaginationParams{
		Offset:    offset,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}, nil
}
