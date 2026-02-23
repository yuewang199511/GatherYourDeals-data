-- +goose Up
CREATE TABLE users (
    id            TEXT    PRIMARY KEY,
    username      TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    role          TEXT    NOT NULL DEFAULT 'user',
    created_at    INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at    INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

-- +goose Down
DROP TABLE IF EXISTS users;
