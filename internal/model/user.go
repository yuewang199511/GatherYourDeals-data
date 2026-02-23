package model

import "errors"

// ErrInvalidToken is returned when a refresh token is missing, expired, or revoked.
var ErrInvalidToken = errors.New("invalid or expired token")

// Role represents the authorization level of a user.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// User represents a registered account in the system.
// Timestamps are Unix epoch seconds (UTC). Conversion to a display format
// is the responsibility of the caller.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         Role   `json:"role"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
}
