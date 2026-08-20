package datatypes

import (
	"encoding/json"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestRecipeFromDBDecodesBlobsAndDefaultsCategory(t *testing.T) {
	cfg := testState(t)
	var recipe Recipe
	if err := json.Unmarshal([]byte(`{
		"type": "recipe",
		"name": "electronic-circuit",
		"energy_required": 0.5,
		"ingredients": [
			{"type": "item", "name": "pcb1", "amount": 1}
		],
		"results": [
			{"type": "item", "name": "electronic-circuit", "amount": 1}
		]
	}`), &recipe); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	inserted, err := recipe.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}

	got, err := RecipeFromDB(recipe.Name, recipe.Type, toNullString(recipe.EntityOrder), inserted)
	if err != nil {
		t.Fatalf("RecipeFromDB: %v", err)
	}
	if got.Category == nil || *got.Category != DefaultRecipeCategory {
		t.Errorf("Category = %v, want %q", got.Category, DefaultRecipeCategory)
	}
	if len(got.Ingredients) != 1 || got.Ingredients[0].Name != "pcb1" {
		t.Errorf("Ingredients = %+v", got.Ingredients)
	}
	if len(got.Results) != 1 || got.Results[0].Name != "electronic-circuit" {
		t.Errorf("Results = %+v", got.Results)
	}
	if got.EnergyRequired == nil || *got.EnergyRequired != 0.5 {
		t.Errorf("EnergyRequired = %v", got.EnergyRequired)
	}
}

func TestAssemblyMachineFromDBDecodesCategories(t *testing.T) {
	cfg := testState(t)
	var machine AssemblyMachine
	if err := json.Unmarshal([]byte(`{
		"type": "assembling-machine",
		"name": "assembling-machine-1",
		"crafting_categories": ["crafting", "basic-crafting"],
		"crafting_speed": 0.5,
		"energy_usage": "75kW"
	}`), &machine); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	inserted, err := machine.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}

	got, err := AssemblyMachineFromDB(machine.Name, machine.Type, toNullString(machine.EntityOrder), inserted)
	if err != nil {
		t.Fatalf("AssemblyMachineFromDB: %v", err)
	}
	if len(got.CraftingCategories) != 2 || got.CraftingCategories[0] != "crafting" {
		t.Errorf("CraftingCategories = %v", got.CraftingCategories)
	}
	if got.CraftingSpeed == nil || *got.CraftingSpeed != 0.5 {
		t.Errorf("CraftingSpeed = %v", got.CraftingSpeed)
	}
}

func TestRecipeFromDBRequiresName(t *testing.T) {
	_, err := RecipeFromDB("", "recipe", toNullString(nil), database.Recipe{ID: 1})
	if err == nil {
		t.Fatal("RecipeFromDB error = nil, want missing name")
	}
}
