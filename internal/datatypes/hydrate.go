package datatypes

import (
	"database/sql"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

// DefaultRecipeCategory is Factorio's implicit category when a recipe omits one.
const DefaultRecipeCategory = "crafting"

// RecipeFromDB reconstructs a Recipe from a database row, decoding gob-encoded
// ingredients and results. A missing category is filled with DefaultRecipeCategory.
func RecipeFromDB(name, prototypeType string, entityOrder sql.NullString, row database.Recipe) (*Recipe, error) {
	if name == "" {
		return nil, fmt.Errorf("recipe row %d missing entity name", row.ID)
	}
	ings, err := gobDecode[[]Ingredient](row.Ingredient)
	if err != nil {
		return nil, fmt.Errorf("recipe %q ingredients: %w", name, err)
	}
	results, err := gobDecode[[]Product](row.Results)
	if err != nil {
		return nil, fmt.Errorf("recipe %q results: %w", name, err)
	}
	normalizeCommodities(ings, results)

	category := fromNullString(row.Category)
	if category == nil || *category == "" {
		cat := DefaultRecipeCategory
		category = &cat
	}
	proto := prototypeType
	if proto == "" {
		proto = "recipe"
	}
	rec := &Recipe{
		Entity: Entity{
			Name:        name,
			Type:        proto,
			EntityOrder: fromNullString(entityOrder),
		},
		EnergyRequired: fromNullFloat64(row.EnergyRequired),
		Category:       category,
		Ingredients:    ings,
		Results:        results,
		recipeEntry:    &row,
	}
	return rec, nil
}

// AssemblyMachineFromDB reconstructs an AssemblyMachine from a database row,
// decoding gob-encoded crafting categories and energy source.
func AssemblyMachineFromDB(name, prototypeType string, entityOrder sql.NullString, row database.AssemblyMachine) (*AssemblyMachine, error) {
	if name == "" {
		return nil, fmt.Errorf("assembly machine row %d missing entity name", row.ID)
	}
	cats, err := gobDecode[[]string](row.CraftingCategories)
	if err != nil {
		return nil, fmt.Errorf("machine %q categories: %w", name, err)
	}
	var src *EnergySource
	if len(row.EnergySource) > 0 {
		decoded, err := gobDecode[EnergySource](row.EnergySource)
		if err != nil {
			return nil, fmt.Errorf("machine %q energy source: %w", name, err)
		}
		src = &decoded
	}
	proto := prototypeType
	if proto == "" {
		proto = "assembling-machine"
	}
	return &AssemblyMachine{
		Entity: Entity{
			Name:        name,
			Type:        proto,
			EntityOrder: fromNullString(entityOrder),
		},
		CraftingCategories: cats,
		CraftingSpeed:      fromNullFloat64(row.CraftingSpeed),
		EnergySource:       src,
		EnergyUsage:        fromNullFloat64(row.EnergyUsage),
		machineEntry:       &row,
	}, nil
}

func normalizeCommodities(ings []Ingredient, results []Product) {
	for i := range ings {
		if ings[i].Type == "" {
			ings[i].Type = "item"
		}
	}
	for i := range results {
		if results[i].Type == "" {
			results[i].Type = "item"
		}
	}
}
