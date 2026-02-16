package model

import "time"

// OAuthClient represents a registered OAuth2 client application.
type OAuthClient struct {
	ID        string    `json:"id"`
	Secret    string    `json:"-"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"createdAt"`
}
