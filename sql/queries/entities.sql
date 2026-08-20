-- name: AddEntity :one
INSERT INTO entities(
  name, prototype_type, entity_order, localised_name
) VALUES (
  @name, @prototype_type, @entity_order, @localised_name
) RETURNING *;

-- name: GetEntityByID :one
SELECT * FROM entities WHERE id = @id;

-- name: GetEntityByName :one
SELECT * FROM entities WHERE name = @name AND prototype_type = @prototype_type;

-- name: GetEntitiesByName :many
SELECT * FROM entities WHERE name = @name;

-- name: GetAllEntities :many
SELECT * FROM entities;

-- name: DeleteEntityByID :exec
DELETE FROM entities WHERE id = @id;

-- name: DeleteEntityByName :exec
DELETE FROM entities WHERE name = @name AND prototype_type = @prototype_type;

-- name: UpdateEntityOrderByID :one
UPDATE entities SET entity_order = @order WHERE id = @id RETURNING *;

-- name: UpdateEntityOrderByName :one
UPDATE entities SET entity_order = @order WHERE name = @name AND prototype_type = @prototype_type RETURNING *;

-- name: UpdateEntityLocalisedNameByID :one
UPDATE entities SET localised_name = @localised_name WHERE id = @id RETURNING *;
