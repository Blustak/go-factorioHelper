-- name: AddEntity :one

INSERT INTO entities(
  name, entity_order
) VALUES (
  @name, @entity_order
) RETURNING *;

-- name: GetEntityByID :one
SELECT * FROM entities WHERE id = @id;

-- name: GetEntityByName :one
SELECT * FROM entities WHERE name = @name;

-- name: GetAllEntities :many

SELECT * FROM entities;

-- name: DeleteEntityByID :exec
DELETE FROM entities WHERE id = @id;

-- name: DeleteEntityByName :exec
DELETE FROM entities WHERE name = @name;

-- name: UpdateEntityOrderByID :one
UPDATE entities SET entity_order = @order WHERE id = @id RETURNING *;

-- name: UpdateEntityOrderByName :one
UPDATE entities SET entity_order = @order WHERE name = @name RETURNING *;




