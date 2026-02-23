-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    token      TEXT    PRIMARY KEY,
    user_id    TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
