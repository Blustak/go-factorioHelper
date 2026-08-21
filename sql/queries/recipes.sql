-- name: AddRecipe :one
INSERT INTO recipes(
  entity_id, energy_required, category, main_product, ingredient, results
) VALUES (
  @entity_id, @energy_required, @category, @main_product, @ingredient, @results
) RETURNING *;

-- name: GetAllRecipes :many
SELECT *
FROM recipes
LEFT JOIN entities
ON recipes.entity_id = entities.id;

-- name: GetAllRecipeValues :many
SELECT * FROM recipes;

-- name: GetRecipeByEntityID :one
SELECT *
FROM recipes
LEFT JOIN entities
ON recipes.entity_id = entities.id
WHERE entities.id = @id;

-- name: GetRecipeByRecipeID :one
SELECT *
FROM recipes
LEFT JOIN entities
ON recipes.entity_id = entities.id
WHERE recipes.id = @id;

-- name: GetRecipeByName :one
SELECT *
FROM recipes
LEFT JOIN entities
ON recipes.entity_id = entities.id
WHERE entities.name = @name;

-- name: UpdateRecipeByID :one
UPDATE recipes
SET entity_id = @entity_id,
energy_required = @energy_required,
category = @category,
main_product = @main_product,
ingredient = @ingredient,
results = @results
WHERE id = @id
RETURNING *;

-- name: SearchRecipesByName :many
SELECT entities.name, recipes.ingredient, recipes.results, recipes.main_product, recipes.energy_required, recipes.category
FROM recipes
LEFT JOIN entities
ON entities.id = recipes.entity_id
WHERE entities.name LIKE @query
OR entities.localised_name LIKE @query
ORDER BY entities.name;
