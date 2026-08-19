-- +goose Up
CREATE TABLE entities(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  name TEXT NOT NULL,
  prototype_type TEXT NOT NULL,
  entity_order TEXT,
  UNIQUE(name, prototype_type)
) STRICT;

CREATE TABLE items(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  entity_id INTEGER NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
  stack_size INTEGER,
  burnt_result INTEGER REFERENCES entities(id) ON DELETE CASCADE,
  fuel_value REAL,
  spoil_result INTEGER REFERENCES entities(id) ON DELETE CASCADE,
  spoil_ticks INTEGER,
  weight INTEGER
) STRICT;

CREATE TABLE fluids(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  entity_id INTEGER NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
  fuel_value REAL,
  gas_temperature INTEGER,
  default_temperature INTEGER,
  max_temperature INTEGER
) STRICT;

CREATE TABLE resources(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  entity_id INTEGER NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
  mining_time REAL,
  results BLOB,
  required_fluid INTEGER REFERENCES entities(id) ON DELETE CASCADE
) STRICT;

CREATE TABLE recipes(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  entity_id INTEGER NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
  energy_required REAL,
  category TEXT,
  main_product INTEGER REFERENCES entities(id) ON DELETE CASCADE,
  ingredient BLOB,
  results BLOB
) STRICT;

CREATE TABLE assembly_machines(
  id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  entity_id INTEGER NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
  crafting_categories BLOB,
  crafting_speed REAL,
  energy_source BLOB,
  energy_usage REAL,
  fixed_recipe INTEGER REFERENCES recipes(id)
) STRICT;

-- +goose Down
DROP TABLE assembly_machines;
DROP TABLE recipes;
DROP TABLE resources;
DROP TABLE fluids;
DROP TABLE items;
DROP TABLE entities;
