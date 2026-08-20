-- name: AddGenerator :one
INSERT INTO generators(
  entity_id, energy_source, effectivity, fluid_usage_per_tick, maximum_temperature, burns_fluid, input_fluid
) VALUES (
  @entity_id, @energy_source, @effectivity, @fluid_usage_per_tick, @maximum_temperature, @burns_fluid, @input_fluid
) RETURNING *;

-- name: GetAllGenerators :many
SELECT *
FROM generators
LEFT JOIN entities
ON generators.entity_id = entities.id;

-- name: GetAllGeneratorValues :many
SELECT * FROM generators;

-- name: GetGeneratorByEntityID :one
SELECT *
FROM generators
LEFT JOIN entities
ON generators.entity_id = entities.id
WHERE entities.id = @id;

-- name: GetGeneratorByGeneratorID :one
SELECT *
FROM generators
LEFT JOIN entities
ON generators.entity_id = entities.id
WHERE generators.id = @id;

-- name: GetGeneratorByName :one
SELECT *
FROM generators
LEFT JOIN entities
ON generators.entity_id = entities.id
WHERE entities.name = @name;

-- name: UpdateGeneratorByID :one
UPDATE generators
SET entity_id = @entity_id,
energy_source = @energy_source,
effectivity = @effectivity,
fluid_usage_per_tick = @fluid_usage_per_tick,
maximum_temperature = @maximum_temperature,
burns_fluid = @burns_fluid,
input_fluid = @input_fluid
WHERE id = @id
RETURNING *;
