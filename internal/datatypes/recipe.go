package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type Recipe struct {
	Entity
	EnergyRequired *float64
	Category       *string
	MainProduct    *string
	Ingredients    []Ingredient
	Results        []Product
	recipeEntry    *database.Recipe
}

func (r *Recipe) UnmarshalJSON(b []byte) error {
	var raw struct {
		Name           string       `json:"name"`
		Type           string       `json:"type"`
		Order          *string      `json:"order"`
		EnergyRequired *float64     `json:"energy_required"`
		Category       *string      `json:"category"`
		MainProduct    *string      `json:"main_product"`
		Ingredients    []Ingredient `json:"ingredients"`
		Results        []Product    `json:"results"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	mainProduct := raw.MainProduct
	if mainProduct != nil && *mainProduct == "" {
		mainProduct = nil
	}
	*r = Recipe{
		Entity: Entity{
			Name:        raw.Name,
			Type:        raw.Type,
			EntityOrder: raw.Order,
		},
		EnergyRequired: raw.EnergyRequired,
		Category:       raw.Category,
		MainProduct:    mainProduct,
		Ingredients:    raw.Ingredients,
		Results:        raw.Results,
	}
	return nil
}

func (r *Recipe) AddToDB(cfg *config.State) (database.Recipe, error) {
	var recipe database.Recipe
	if r == nil {
		return recipe, DatatypeNilError[Recipe]("Nil recipe")
	}
	if _, err := r.Entity.AddToDB(cfg); err != nil {
		return recipe, err
	}
	if err := stubIngredients(cfg, r.Ingredients); err != nil {
		return recipe, err
	}
	if err := stubProducts(cfg, r.Results); err != nil {
		return recipe, err
	}
	mainProduct, err := stubMainProduct(cfg, r.MainProduct)
	if err != nil {
		return recipe, err
	}
	ingBlob, err := gobEncodeSlice(r.Ingredients)
	if err != nil {
		return recipe, err
	}
	resBlob, err := gobEncodeSlice(r.Results)
	if err != nil {
		return recipe, err
	}

	params := database.AddRecipeParams{
		EntityID:       r.entEntry.ID,
		EnergyRequired: toNullFloat64(r.EnergyRequired),
		Category:       toNullString(r.Category),
		MainProduct:    mainProduct,
		Ingredient:     ingBlob,
		Results:        resBlob,
	}

	existing, err := cfg.DB.GetRecipeByEntityID(cfg.CTX, r.entEntry.ID)
	if err == nil {
		recipe, err = cfg.DB.UpdateRecipeByID(cfg.CTX, database.UpdateRecipeByIDParams{
			EntityID:       params.EntityID,
			EnergyRequired: params.EnergyRequired,
			Category:       params.Category,
			MainProduct:    params.MainProduct,
			Ingredient:     params.Ingredient,
			Results:        params.Results,
			ID:             existing.ID,
		})
		if err != nil {
			return recipe, err
		}
		r.recipeEntry = &recipe
		return recipe, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return recipe, err
	}

	recipe, err = cfg.DB.AddRecipe(cfg.CTX, params)
	if err != nil {
		return recipe, err
	}
	r.recipeEntry = &recipe
	return recipe, nil
}

func (r *Recipe) GetFromDB(cfg *config.State) (database.Recipe, error) {
	var recipe database.Recipe
	if r == nil {
		return recipe, DatatypeNilError[Recipe]("Nil recipe")
	}
	if r.recipeEntry == nil {
		return recipe, fmt.Errorf("recipeEntry not set")
	}
	if _, err := r.Entity.GetFromDB(cfg); err != nil {
		return recipe, err
	}
	row, err := cfg.DB.GetRecipeByEntityID(cfg.CTX, r.entEntry.ID)
	if err != nil {
		return recipe, err
	}
	recipe = recipeFromJoin(row)
	if !rowsEqual(*r.recipeEntry, recipe) {
		return recipe, mismatchError(cfg, "Recipe", *r.recipeEntry, recipe)
	}
	return recipe, nil
}

func (r *Recipe) Unwrap() (database.Recipe, error) {
	if r == nil {
		return database.Recipe{}, DatatypeNilError[Recipe]("Nil recipe")
	}
	if r.recipeEntry == nil {
		return database.Recipe{}, fmt.Errorf("recipeEntry not set")
	}
	return *r.recipeEntry, nil
}

func (r *Recipe) GetEntityID(cfg *config.State) (int64, error) {
	if r == nil {
		return 0, DatatypeNilError[Recipe]("Cannot get id of nil recipe")
	}
	return r.Entity.GetEntityID(cfg)
}

func stubMainProduct(cfg *config.State, name *string) (sql.NullInt64, error) {
	if name == nil || *name == "" {
		return sql.NullInt64{}, nil
	}
	item, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          *name,
		PrototypeType: "item",
	})
	if err == nil {
		return sql.NullInt64{Int64: item.ID, Valid: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return sql.NullInt64{}, err
	}
	fluid, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          *name,
		PrototypeType: "fluid",
	})
	if err == nil {
		return sql.NullInt64{Int64: fluid.ID, Valid: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return sql.NullInt64{}, err
	}
	ent, err := ensureEntity(cfg, *name, "item")
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: ent.ID, Valid: true}, nil
}

func recipeFromJoin(row database.GetRecipeByEntityIDRow) database.Recipe {
	return database.Recipe{
		ID:             row.ID,
		EntityID:       row.EntityID,
		EnergyRequired: row.EnergyRequired,
		Category:       row.Category,
		MainProduct:    row.MainProduct,
		Ingredient:     row.Ingredient,
		Results:        row.Results,
	}
}
