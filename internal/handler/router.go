package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/middleware"
)

// NewRouter creates a gin router with all routes registered.
func NewRouter(
	authHandler *AuthHandler,
	adminHandler *AdminHandler,
	metaHandler *MetaHandler,
	receiptHandler *ReceiptHandler,
	tokens *auth.TokenService,
) *gin.Engine {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	// Public endpoints
	v1.POST("/users", authHandler.Register)       // register
	v1.POST("/auth/login", authHandler.Login)     // login → returns token pair
	v1.POST("/auth/refresh", authHandler.Refresh) // refresh access token

	// Authenticated endpoints
	protected := v1.Group("")
	protected.Use(middleware.Auth(tokens))
	{
		protected.POST("/auth/logout", authHandler.Logout) // logout (revoke refresh token)
		protected.GET("/auth/me", authHandler.Me)          // whoami

		// Meta — any authenticated user can list; admin can create/update
		protected.GET("/meta", metaHandler.ListFields)

		// Receipts
		protected.POST("/receipts", receiptHandler.CreateReceipt)
		protected.GET("/receipts", receiptHandler.ListReceipts)
		protected.GET("/receipts/:id", receiptHandler.GetReceipt)
		protected.DELETE("/receipts/:id", receiptHandler.DeleteReceipt)
	}

	// Admin-only endpoints
	admin := v1.Group("/admin")
	admin.Use(middleware.Auth(tokens), middleware.RequireAdmin())
	{
		admin.GET("/users", adminHandler.ListUsers)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)

		// Meta management — admin only
		admin.POST("/meta", metaHandler.CreateField)
		admin.PUT("/meta/:fieldName", metaHandler.UpdateDescription)
	}

	return r
}
