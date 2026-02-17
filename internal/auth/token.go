package auth

import (
	"context"

	"github.com/gatheryourdeals/data/internal/config"
	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/server"
)

// NewOAuthManager creates a go-oauth2 manager with the given token store
// and a database-backed client store.
func NewOAuthManager(cfg *config.Config, clients repository.ClientRepository, tokenStore oauth2.TokenStore) (*manage.Manager, error) {
	accessExp, err := cfg.OAuth2.GetAccessTokenDuration()
	if err != nil {
		return nil, err
	}
	refreshExp, err := cfg.OAuth2.GetRefreshTokenDuration()
	if err != nil {
		return nil, err
	}

	manager := manage.NewDefaultManager()

	manager.SetPasswordTokenCfg(&manage.Config{
		AccessTokenExp:    accessExp,
		RefreshTokenExp:   refreshExp,
		IsGenerateRefresh: true,
	})

	manager.SetRefreshTokenCfg(&manage.RefreshingConfig{
		AccessTokenExp:    accessExp,
		RefreshTokenExp:   refreshExp,
		IsGenerateRefresh: true,
	})

	manager.MapTokenStorage(tokenStore)

	// Database-backed client store. Clients persist across restarts.
	manager.MapClientStorage(NewDBClientStore(clients))

	return manager, nil
}

// NewRedisTokenStore creates a Redis-backed token store for production use.
func NewRedisTokenStore(cfg *config.Config) oauth2.TokenStore {
	return newRedisTokenStore(cfg)
}

// NewOAuthServer creates a go-oauth2 server configured for the
// resource owner password credentials grant.
func NewOAuthServer(manager *manage.Manager, authService *Service) *server.Server {
	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(false)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	// PasswordAuthorizationHandler is called by the OAuth2 server
	// when processing a password credentials grant.
	// It verifies the username/password and returns the user ID.
	srv.SetPasswordAuthorizationHandler(func(ctx context.Context, clientID, username, password string) (string, error) {
		user, err := authService.Login(ctx, username, password)
		if err != nil {
			return "", err
		}
		return user.ID, nil
	})

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
	})

	return srv
}
