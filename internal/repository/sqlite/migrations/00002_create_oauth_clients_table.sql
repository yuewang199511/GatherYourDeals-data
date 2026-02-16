-- +goose Up
CREATE TABLE oauth_clients (
    id         TEXT PRIMARY KEY,
    secret     TEXT NOT NULL DEFAULT '',
    domain     TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS oauth_clients;
