package handler

import (
	"io"
	"os"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/middleware"
)

// NewRouter creates a gin router with all routes registered.
// The logWriter is used for Gin's own request logging so it goes to
// the same destination as application logs (stdout + rotating file).
// tracerProvider is used by the OTel middleware; pass a noop provider when
// tracing is disabled (FR-001, FR-002, FR-003).
func NewRouter(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	metaHandler *MetaHandler,
	receiptHandler *ReceiptHandler,
	tokens *auth.TokenService,
	logWriter io.Writer,
	tracerProvider oteltrace.TracerProvider,
) *gin.Engine {
	if logWriter != nil {
		gin.DefaultWriter = logWriter
	}

	if tracerProvider == nil {
		tracerProvider = tracenoop.NewTracerProvider()
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "gatheryourdeals"
	}

	r := gin.Default()

	// OTel HTTP instrumentation: creates one span per request, injects trace
	// context into c.Request.Context(). MUST be first middleware (FR-001).
	r.Use(otelgin.Middleware(serviceName, otelgin.WithTracerProvider(tracerProvider)))

	// Test metadata: reads X-Test-* headers → sets test.* span attributes.
	// Must run after otelgin so the span exists (FR-006, FR-007).
	r.Use(middleware.TestMetadata())

	// Health check — unauthenticated, outside /api/v1
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

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
