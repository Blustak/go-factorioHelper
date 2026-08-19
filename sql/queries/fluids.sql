-- name: AddFluid :one
INSERT INTO fluids(
  entity_id, fuel_value, gas_temperature, default_temperature, max_temperature
) VALUES (
  @entity_id, @fuel_value, @gas_temperature, @default_temperature, @max_temperature
) RETURNING *;

-- name: GetAllFluids :many
SELECT *
FROM fluids
LEFT JOIN entities
ON fluids.entity_id = entities.id;

-- name: GetAllFluidValues :many
SELECT * FROM fluids;

-- name: GetFluidByEntityID :one
SELECT *
FROM fluids
LEFT JOIN entities
ON fluids.entity_id = entities.id
WHERE entities.id = @id;

-- name: GetFluidByFluidID :one
SELECT *
FROM fluids
LEFT JOIN entities
ON fluids.entity_id = entities.id
WHERE fluids.id = @id;

-- name: GetFluidByName :one
SELECT *
FROM fluids
LEFT JOIN entities
ON fluids.entity_id = entities.id
WHERE entities.name = @name;

-- name: UpdateFluidByID :one
UPDATE fluids
SET entity_id = @entity_id,
fuel_value = @fuel_value,
gas_temperature = @gas_temperature,
default_temperature = @default_temperature,
max_temperature = @max_temperature
WHERE id = @id
RETURNING *;
