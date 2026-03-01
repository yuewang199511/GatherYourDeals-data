package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/gatheryourdeals/data/internal/model"
)

// ErrInvalidToken is aliased from model so callers that already import auth
// don't need a separate model import just for this error.
var ErrInvalidToken = model.ErrInvalidToken

// Claims is the JWT payload embedded in every access token.
type Claims struct {
	UserID string     `json:"uid"`
	Role   model.Role `json:"role"`
	jwt.RegisteredClaims
}

// TokenService issues and validates JWTs, and manages refresh tokens.
type TokenService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	store         RefreshTokenStore
}

// RefreshTokenStore persists refresh tokens for revocation support.
// A single SQLite table is sufficient.
type RefreshTokenStore interface {
	// Save stores a refresh token bound to a user.
	Save(ctx context.Context, token string, userID string, expiresAt time.Time) error
	// Find returns the userID for a valid, non-expired token.
	// Returns ("", model.ErrInvalidToken) if not found or expired.
	Find(ctx context.Context, token string) (userID string, err error)
	// Delete revokes a refresh token (logout).
	Delete(ctx context.Context, token string) error
	// DeleteAllForUser revokes all refresh tokens for a user (force-logout).
	DeleteAllForUser(ctx context.Context, userID string) error
}

func NewTokenService(secret []byte, accessExpiry, refreshExpiry time.Duration, store RefreshTokenStore) *TokenService {
	return &TokenService{
		secret:        secret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		store:         store,
	}
}

// IssueTokenPair creates a new access + refresh token pair for a user.
func (ts *TokenService) IssueTokenPair(ctx context.Context, user *model.User) (accessToken, refreshToken string, err error) {
	accessToken, err = ts.newAccessToken(user)
	if err != nil {
		return
	}
	refreshToken, err = ts.newRefreshToken(ctx, user.ID)
	return
}

// ValidateAccessToken parses and validates an access token, returning its claims.
func (ts *TokenService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return ts.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// RefreshAccessToken validates a refresh token and issues a new access token.
// The old refresh token is consumed and a new one is issued (rotation).
func (ts *TokenService) RefreshAccessToken(ctx context.Context, refreshToken string, users UserLookup) (newAccess, newRefresh string, err error) {
	userID, err := ts.store.Find(ctx, refreshToken)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	user, err := users.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return "", "", ErrInvalidToken
	}

	// Rotate: delete old token, issue new pair.
	if err = ts.store.Delete(ctx, refreshToken); err != nil {
		return "", "", err
	}
	return ts.IssueTokenPair(ctx, user)
}

// RevokeRefreshToken deletes a refresh token (used on logout).
func (ts *TokenService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	return ts.store.Delete(ctx, refreshToken)
}

// RevokeAllForUser revokes all refresh tokens for a user (used when deleting a user).
func (ts *TokenService) RevokeAllForUser(ctx context.Context, userID string) error {
	return ts.store.DeleteAllForUser(ctx, userID)
}

// UserLookup is satisfied by any type that can retrieve a user by ID.
// Defined here as a minimal interface rather than importing the full repository.
type UserLookup interface {
	GetUserByID(ctx context.Context, id string) (*model.User, error)
}

// --- private helpers ---

func (ts *TokenService) newAccessToken(user *model.User) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(), // jti: unique per token, prevents duplicate tokens
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessExpiry)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ts.secret)
}

func (ts *TokenService) newRefreshToken(ctx context.Context, userID string) (string, error) {
	// Use a signed JWT as the refresh token. A random jti (JWT ID) ensures
	// uniqueness even if two tokens are issued within the same second.
	now := time.Now()
	exp := now.Add(ts.refreshExpiry)
	claims := jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ts.secret)
	if err != nil {
		return "", err
	}
	if err := ts.store.Save(ctx, tokenStr, userID, exp); err != nil {
		return "", err
	}
	return tokenStr, nil
}
