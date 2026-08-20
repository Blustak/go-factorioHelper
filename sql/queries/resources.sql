-- name: AddResource :one
INSERT INTO resources(
  entity_id, mining_time, results, required_fluid, category
) VALUES (
  @entity_id, @mining_time, @results, @required_fluid, @category
) RETURNING *;

-- name: GetAllResources :many
SELECT *
FROM resources
LEFT JOIN entities
ON resources.entity_id = entities.id;

-- name: GetAllResourceValues :many
SELECT * FROM resources;

-- name: GetResourceByEntityID :one
SELECT *
FROM resources
LEFT JOIN entities
ON resources.entity_id = entities.id
WHERE entities.id = @id;

-- name: GetResourceByResourceID :one
SELECT *
FROM resources
LEFT JOIN entities
ON resources.entity_id = entities.id
WHERE resources.id = @id;

-- name: GetResourceByName :one
SELECT *
FROM resources
LEFT JOIN entities
ON resources.entity_id = entities.id
WHERE entities.name = @name;

-- name: UpdateResourceByID :one
UPDATE resources
SET entity_id = @entity_id,
mining_time = @mining_time,
results = @results,
required_fluid = @required_fluid,
category = @category
WHERE id = @id
RETURNING *;
