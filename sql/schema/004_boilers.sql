-- +goose Up
ALTER TABLE items ADD COLUMN fuel_category TEXT;

CREATE TABLE boilers(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  entity_id INTEGER NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
  energy_source BLOB,
  energy_consumption REAL,
  target_temperature REAL,
  mode TEXT,
  input_fluid INTEGER REFERENCES entities(id) ON DELETE CASCADE,
  output_fluid INTEGER REFERENCES entities(id) ON DELETE CASCADE
) STRICT;

CREATE TABLE generators(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  entity_id INTEGER NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
  energy_source BLOB,
  effectivity REAL,
  fluid_usage_per_tick REAL,
  maximum_temperature REAL,
  burns_fluid INTEGER,
  input_fluid INTEGER REFERENCES entities(id) ON DELETE CASCADE
) STRICT;

-- +goose Down
DROP TABLE generators;
DROP TABLE boilers;
ALTER TABLE items DROP COLUMN fuel_category;
