package handler

import (
	"io"

	"github.com/gin-gonic/gin"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/middleware"
)

// NewRouter creates a gin router with all routes registered.
// The logWriter is used for Gin's own request logging so it goes to
// the same destination as application logs (stdout + rotating file).
func NewRouter(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	metaHandler *MetaHandler,
	receiptHandler *ReceiptHandler,
	tokens *auth.TokenService,
	logWriter io.Writer,
) *gin.Engine {
	if logWriter != nil {
		gin.DefaultWriter = logWriter
	}

	r := gin.Default()
	v1 := r.Group("/api/v1")

	// Public endpoints
	v1.POST("/users", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)

	// Authenticated endpoints — role checks happen inside each handler
	protected := v1.Group("")
	protected.Use(middleware.Auth(tokens))
	{
		// Auth
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/auth/me", authHandler.Me)

		// Users (admin-only checks inside handler)
		protected.GET("/users", userHandler.ListUsers)
		protected.DELETE("/users/:id", userHandler.DeleteUser)

		// Meta (update description has admin check inside handler)
		protected.GET("/meta", metaHandler.ListFields)
		protected.POST("/meta", metaHandler.CreateField)
		protected.PUT("/meta/:fieldName", metaHandler.UpdateDescription)

		// Receipts
		protected.POST("/receipts", receiptHandler.CreateReceipt)
		protected.GET("/receipts", receiptHandler.ListReceipts)
		protected.GET("/receipts/:id", receiptHandler.GetReceipt)
		protected.DELETE("/receipts/:id", receiptHandler.DeleteReceipt)
	}

	return r
}
