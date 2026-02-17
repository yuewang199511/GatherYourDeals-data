package auth

import (
	"github.com/gatheryourdeals/data/internal/config"
	oauth2 "github.com/go-oauth2/oauth2/v4"
	oredis "github.com/go-oauth2/redis/v4"
	"github.com/go-redis/redis/v8"
)

// newRedisTokenStore creates a Redis-backed OAuth2 token store.
func newRedisTokenStore(cfg *config.Config) oauth2.TokenStore {
	return oredis.NewRedisStore(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}
