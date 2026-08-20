-- +goose Up
ALTER TABLE resources ADD COLUMN category TEXT;

CREATE TABLE resource_producers(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  entity_id INTEGER NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
  resource_categories BLOB,
  mining_speed REAL,
  pumping_speed REAL,
  produced_fluid INTEGER REFERENCES entities(id) ON DELETE CASCADE,
  energy_source BLOB,
  energy_usage REAL
) STRICT;

-- +goose Down
DROP TABLE resource_producers;
ALTER TABLE resources DROP COLUMN category;
