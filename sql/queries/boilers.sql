-- name: AddBoiler :one
INSERT INTO boilers(
  entity_id, energy_source, energy_consumption, target_temperature, mode, input_fluid, output_fluid
) VALUES (
  @entity_id, @energy_source, @energy_consumption, @target_temperature, @mode, @input_fluid, @output_fluid
) RETURNING *;

-- name: GetAllBoilers :many
SELECT *
FROM boilers
LEFT JOIN entities
ON boilers.entity_id = entities.id;

-- name: GetAllBoilerValues :many
SELECT * FROM boilers;

-- name: GetBoilerByEntityID :one
SELECT *
FROM boilers
LEFT JOIN entities
ON boilers.entity_id = entities.id
WHERE entities.id = @id;

-- name: GetBoilerByBoilerID :one
SELECT *
FROM boilers
LEFT JOIN entities
ON boilers.entity_id = entities.id
WHERE boilers.id = @id;

-- name: GetBoilerByName :one
SELECT *
FROM boilers
LEFT JOIN entities
ON boilers.entity_id = entities.id
WHERE entities.name = @name;

-- name: UpdateBoilerByID :one
UPDATE boilers
SET entity_id = @entity_id,
energy_source = @energy_source,
energy_consumption = @energy_consumption,
target_temperature = @target_temperature,
mode = @mode,
input_fluid = @input_fluid,
output_fluid = @output_fluid
WHERE id = @id
RETURNING *;
