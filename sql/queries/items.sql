-- name: AddItem :one
INSERT INTO items(
  entity_id, stack_size, burnt_result, fuel_value, spoil_result, spoil_ticks, weight, fuel_category
) VALUES (
  @entity_id, @stack_size, @burnt_result, @fuel_value, @spoil_result, @spoil_ticks, @weight, @fuel_category
) RETURNING *;

-- name: GetAllItems :many

SELECT
*
FROM items
LEFT JOIN entities
ON items.entity_id = entities.id;

-- name: GetAllItemValues :many
SELECT * FROM items;

-- name: GetItemByEntityID :one
SELECT *
FROM items
LEFT JOIN entities
ON items.entity_id = entities.id
WHERE entities.id = @id;

-- name: GetItemByItemID :one
SELECT *
FROM items
LEFT JOIN entities
ON items.entity_id = entities.id
WHERE items.id = @id;

-- name: GetItemByName :one

SELECT *
FROM items
LEFT JOIN entities
ON items.entity_id = entities.id
WHERE entities.name = @name;

-- name: UpdateItemByID :one
UPDATE items
SET entity_id = @entity_id,
stack_size = @stack_size,
burnt_result = @burnt_result,
fuel_value = @fuel_value,
spoil_result = @spoil_result,
spoil_ticks = @spoil_ticks,
weight = @weight,
fuel_category = @fuel_category
WHERE id = @id
RETURNING *;

-- name: SearchItemsByName :many
SELECT entities.name, items.stack_size, items.fuel_value
FROM items
LEFT JOIN entities
ON entities.id = items.entity_id
WHERE entities.name LIKE @query
ORDER BY entities.name;
