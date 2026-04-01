package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/redis/go-redis/extra/redisotel/v9"
	goredis "github.com/redis/go-redis/v9"
)

// RefreshTokenStore is a Redis-backed implementation of auth.RefreshTokenStore.
//
// Key layout:
//
//	rt:{token}    → userID (string, TTL = token expiry)
//	ut:{userID}   → SET of active token strings (for DeleteAllForUser)
type RefreshTokenStore struct {
	client *goredis.Client
}

// New parses the Redis URL and returns a connected RefreshTokenStore.
func New(redisURL string) (*RefreshTokenStore, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}
	client := goredis.NewClient(opts)
	if err := redisotel.InstrumentTracing(client); err != nil {
		return nil, fmt.Errorf("instrument redis tracing: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RefreshTokenStore{client: client}, nil
}

// Close closes the underlying Redis connection.
func (s *RefreshTokenStore) Close() error {
	return s.client.Close()
}

func (s *RefreshTokenStore) Save(ctx context.Context, token, userID string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // already expired, nothing to save
	}
	pipe := s.client.Pipeline()
	pipe.Set(ctx, rtKey(token), userID, ttl)
	pipe.SAdd(ctx, utKey(userID), token)
	// Extend the user-set TTL so it lives at least as long as this token.
	pipe.Expire(ctx, utKey(userID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RefreshTokenStore) Find(ctx context.Context, token string) (string, error) {
	userID, err := s.client.Get(ctx, rtKey(token)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", model.ErrInvalidToken
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *RefreshTokenStore) Delete(ctx context.Context, token string) error {
	// Look up the owner so we can clean the user-set too.
	userID, err := s.client.Get(ctx, rtKey(token)).Result()
	if errors.Is(err, goredis.Nil) {
		return nil // already gone — idempotent
	}
	if err != nil {
		return err
	}
	pipe := s.client.Pipeline()
	pipe.Del(ctx, rtKey(token))
	pipe.SRem(ctx, utKey(userID), token)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RefreshTokenStore) DeleteAllForUser(ctx context.Context, userID string) error {
	tokens, err := s.client.SMembers(ctx, utKey(userID)).Result()
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}
	pipe := s.client.Pipeline()
	for _, t := range tokens {
		pipe.Del(ctx, rtKey(t))
	}
	pipe.Del(ctx, utKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

func rtKey(token string) string  { return "rt:" + token }
func utKey(userID string) string { return "ut:" + userID }
