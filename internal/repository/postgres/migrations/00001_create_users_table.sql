-- +goose Up
CREATE TABLE users (
    id            TEXT   PRIMARY KEY,
    username      TEXT   NOT NULL UNIQUE,
    password_hash TEXT   NOT NULL,
    role          TEXT   NOT NULL DEFAULT 'user',
    created_at    BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    updated_at    BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
);

-- +goose Down
DROP TABLE IF EXISTS users;
