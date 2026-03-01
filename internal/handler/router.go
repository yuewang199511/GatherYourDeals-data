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
	}

	// Admin-only endpoints
	admin := v1.Group("/admin")
	admin.Use(middleware.Auth(tokens), middleware.RequireAdmin())
	{
		admin.GET("/users", adminHandler.ListUsers)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
	}

	return r
}
