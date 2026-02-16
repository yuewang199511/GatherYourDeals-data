package handler

import (
	"github.com/gatheryourdeals/data/internal/middleware"
	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4/manage"
)

// NewRouter creates a gin router with all routes registered.
func NewRouter(
	authHandler *AuthHandler,
	adminHandler *AdminHandler,
	oauthManager *manage.Manager,
	users repository.UserRepository,
) *gin.Engine {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	// Public endpoints
	v1.POST("/users", authHandler.Register)
	v1.POST("/oauth/token", authHandler.Token)

	// Protected endpoints (any authenticated user)
	v1protected := v1.Group("")
	v1protected.Use(middleware.Auth(oauthManager, users))
	{
		v1protected.DELETE("/oauth/sessions", authHandler.Logout)
	}

	// Admin-only endpoints
	v1admin := v1.Group("/admin")
	v1admin.Use(middleware.Auth(oauthManager, users), middleware.RequireAdmin())
	{
		v1admin.POST("/clients", adminHandler.CreateClient)
		v1admin.GET("/clients", adminHandler.ListClients)
		v1admin.DELETE("/clients/:id", adminHandler.DeleteClient)
	}

	return r
}
