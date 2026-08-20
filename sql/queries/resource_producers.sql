-- name: AddResourceProducer :one
INSERT INTO resource_producers(
  entity_id, resource_categories, mining_speed, pumping_speed, produced_fluid, energy_source, energy_usage
) VALUES (
  @entity_id, @resource_categories, @mining_speed, @pumping_speed, @produced_fluid, @energy_source, @energy_usage
) RETURNING *;

-- name: GetAllResourceProducers :many
SELECT *
FROM resource_producers
LEFT JOIN entities
ON resource_producers.entity_id = entities.id;

-- name: GetAllResourceProducerValues :many
SELECT * FROM resource_producers;

-- name: GetResourceProducerByEntityID :one
SELECT *
FROM resource_producers
LEFT JOIN entities
ON resource_producers.entity_id = entities.id
WHERE entities.id = @id;

-- name: GetResourceProducerByResourceProducerID :one
SELECT *
FROM resource_producers
LEFT JOIN entities
ON resource_producers.entity_id = entities.id
WHERE resource_producers.id = @id;

-- name: GetResourceProducerByName :one
SELECT *
FROM resource_producers
LEFT JOIN entities
ON resource_producers.entity_id = entities.id
WHERE entities.name = @name;

-- name: UpdateResourceProducerByID :one
UPDATE resource_producers
SET entity_id = @entity_id,
resource_categories = @resource_categories,
mining_speed = @mining_speed,
pumping_speed = @pumping_speed,
produced_fluid = @produced_fluid,
energy_source = @energy_source,
energy_usage = @energy_usage
WHERE id = @id
RETURNING *;
