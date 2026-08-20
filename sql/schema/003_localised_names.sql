-- +goose Up
ALTER TABLE entities ADD COLUMN localised_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE entities DROP COLUMN localised_name;
