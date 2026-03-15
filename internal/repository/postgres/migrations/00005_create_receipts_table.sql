-- +goose Up
CREATE TABLE receipts (
    id            TEXT   PRIMARY KEY,
    product_name  TEXT   NOT NULL,
    purchase_date TEXT   NOT NULL,
    price         TEXT   NOT NULL,
    amount        TEXT   NOT NULL,
    store_name    TEXT   NOT NULL,
    latitude      REAL,
    longitude     REAL,
    extras        TEXT   NOT NULL DEFAULT '{}',
    upload_time   BIGINT NOT NULL,
    user_id       TEXT   NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS receipts;
