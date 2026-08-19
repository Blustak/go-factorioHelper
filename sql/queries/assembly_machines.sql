-- name: AddAssemblyMachine :one
INSERT INTO assembly_machines(
  entity_id, crafting_categories, crafting_speed, energy_source, energy_usage, fixed_recipe
) VALUES (
  @entity_id, @crafting_categories, @crafting_speed, @energy_source, @energy_usage, @fixed_recipe
) RETURNING *;

-- name: GetAllAssemblyMachines :many
SELECT *
FROM assembly_machines
LEFT JOIN entities
ON assembly_machines.entity_id = entities.id;

-- name: GetAllAssemblyMachineValues :many
SELECT * FROM assembly_machines;

-- name: GetAssemblyMachineByEntityID :one
SELECT *
FROM assembly_machines
LEFT JOIN entities
ON assembly_machines.entity_id = entities.id
WHERE entities.id = @id;

-- name: GetAssemblyMachineByAssemblyMachineID :one
SELECT *
FROM assembly_machines
LEFT JOIN entities
ON assembly_machines.entity_id = entities.id
WHERE assembly_machines.id = @id;

-- name: GetAssemblyMachineByName :one
SELECT *
FROM assembly_machines
LEFT JOIN entities
ON assembly_machines.entity_id = entities.id
WHERE entities.name = @name;

-- name: UpdateAssemblyMachineByID :one
UPDATE assembly_machines
SET entity_id = @entity_id,
crafting_categories = @crafting_categories,
crafting_speed = @crafting_speed,
energy_source = @energy_source,
energy_usage = @energy_usage,
fixed_recipe = @fixed_recipe
WHERE id = @id
RETURNING *;
