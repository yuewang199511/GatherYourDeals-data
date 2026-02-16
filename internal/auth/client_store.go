package auth

import (
	"context"

	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/models"
)

// DBClientStore implements go-oauth2's ClientStore interface
// backed by the database, so clients persist across restarts
// and can be added/revoked while the server is running.
type DBClientStore struct {
	clients repository.ClientRepository
}

// NewDBClientStore creates a new database-backed client store.
func NewDBClientStore(clients repository.ClientRepository) *DBClientStore {
	return &DBClientStore{clients: clients}
}

// GetByID implements the go-oauth2 ClientStore interface.
// It is called by the OAuth2 server to validate a client_id on every token request.
func (s *DBClientStore) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	client, err := s.clients.GetClientByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, nil
	}
	return &models.Client{
		ID:     client.ID,
		Secret: client.Secret,
		Domain: client.Domain,
	}, nil
}
