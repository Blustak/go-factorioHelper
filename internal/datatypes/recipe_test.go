package datatypes

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestRecipeUnmarshalAndRoundTrip(t *testing.T) {
	cfg := testState(t)
	var recipe Recipe
	if err := json.Unmarshal([]byte(`{
		"type": "recipe",
		"name": "electronic-circuit",
		"order": "aab",
		"energy_required": 4,
		"category": "chip",
		"ingredients": [
			{"type": "item", "name": "pcb1", "amount": 1},
			{"type": "item", "name": "solder", "amount": 2}
		],
		"results": [
			{"type": "item", "name": "electronic-circuit", "amount": 3}
		],
		"main_product": "electronic-circuit"
	}`), &recipe); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if len(recipe.Ingredients) != 2 {
		t.Fatalf("Ingredients len = %d, want 2", len(recipe.Ingredients))
	}
	if recipe.MainProduct == nil || *recipe.MainProduct != "electronic-circuit" {
		t.Errorf("MainProduct = %v", recipe.MainProduct)
	}

	inserted, err := recipe.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if _, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          "pcb1",
		PrototypeType: "item",
	}); err != nil {
		t.Fatalf("pcb1 stub missing: %v", err)
	}

	ings, err := gobDecode[[]Ingredient](inserted.Ingredient)
	if err != nil {
		t.Fatalf("gobDecode ingredients: %v", err)
	}
	if len(ings) != 2 || ings[0].Name != "pcb1" {
		t.Errorf("decoded ingredients = %+v", ings)
	}
	results, err := gobDecode[[]Product](inserted.Results)
	if err != nil {
		t.Fatalf("gobDecode results: %v", err)
	}
	if len(results) != 1 || results[0].Name != "electronic-circuit" {
		t.Errorf("decoded results = %+v", results)
	}

	got, err := recipe.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if !rowsEqual(got, inserted) {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}
}

func TestRecipeEmptyMainProduct(t *testing.T) {
	var recipe Recipe
	if err := json.Unmarshal([]byte(`{
		"type": "recipe",
		"name": "kovarex-enrichment-process",
		"energy_required": 10,
		"ingredients": [],
		"results": [],
		"main_product": ""
	}`), &recipe); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if recipe.MainProduct != nil {
		t.Errorf("MainProduct = %v, want nil", recipe.MainProduct)
	}
}

func TestRecipeItemNameCollision(t *testing.T) {
	cfg := testState(t)
	item := Item{
		Entity:    Entity{Name: "iron-plate", Type: "item"},
		StackSize: ptr(int64(100)),
	}
	recipe := Recipe{
		Entity:         Entity{Name: "iron-plate", Type: "recipe"},
		EnergyRequired: ptr(3.2),
		Ingredients: []Ingredient{{
			Entity: Entity{Name: "iron-ore", Type: "item"},
			Amount: ptr(8.0),
		}},
		Results: []Product{{
			Entity: Entity{Name: "iron-plate", Type: "item"},
			Amount: ptr(1.0),
		}},
	}
	itemRow, err := item.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB item: %v", err)
	}
	recipeRow, err := recipe.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB recipe: %v", err)
	}
	if itemRow.EntityID == recipeRow.EntityID {
		t.Fatal("item and recipe share entity_id, want distinct entities")
	}
}

func TestRecipeAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var recipe *Recipe
	_, err := recipe.AddToDB(cfg)
	var nilErr DatatypeNilError[Recipe]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}

func TestRecipeMainProductPrefersExistingFluid(t *testing.T) {
	cfg := testState(t)
	fluid := Fluid{Entity: Entity{Name: "petroleum-gas", Type: "fluid"}}
	if _, err := fluid.AddToDB(cfg); err != nil {
		t.Fatalf("AddToDB fluid: %v", err)
	}
	recipe := Recipe{
		Entity:      Entity{Name: "advanced-oil-processing", Type: "recipe"},
		MainProduct: ptr("petroleum-gas"),
	}
	row, err := recipe.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB recipe: %v", err)
	}
	if !row.MainProduct.Valid || row.MainProduct.Int64 != fluid.entEntry.ID {
		t.Errorf("MainProduct = %+v, want fluid id %d", row.MainProduct, fluid.entEntry.ID)
	}
}
