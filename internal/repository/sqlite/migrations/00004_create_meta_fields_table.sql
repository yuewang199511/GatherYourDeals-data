-- +goose Up
CREATE TABLE meta_fields (
    field_name  TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    field_type  TEXT NOT NULL DEFAULT 'string',
    native      INTEGER NOT NULL DEFAULT 0
);

-- Seed the native fields that every receipt must have.
INSERT INTO meta_fields (field_name, description, field_type, native) VALUES
    ('productName',  'name of the product',                                              'string', 1),
    ('purchaseDate', 'purchase date in Y.M.D format',                                    'string', 1),
    ('price',        'the price for payment',                                             'string', 1),
    ('amount',       'the amount of purchased goods, in the format of number or number(unit)', 'string', 1),
    ('storeName',    'name of the store',                                                 'string', 1),
    ('latitude',     'latitude of the location, this field is optional',                  'float',  1),
    ('longitude',    'longitude of the location, this field is optional',                 'float',  1);

-- +goose Down
DROP TABLE IF EXISTS meta_fields;
