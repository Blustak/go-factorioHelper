package datatypes

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestAssemblyMachineUnmarshalAndRoundTrip(t *testing.T) {
	cfg := testState(t)
	var machine AssemblyMachine
	if err := json.Unmarshal([]byte(`{
		"type": "assembling-machine",
		"name": "assembling-machine-1",
		"crafting_categories": ["crafting", "basic-crafting"],
		"crafting_speed": 1,
		"energy_source": {
			"type": "burner",
			"fuel_categories": ["chemical", "biomass"],
			"effectivity": 1,
			"fuel_inventory_size": 1,
			"burnt_inventory_size": 1,
			"emissions_per_minute": {"pollution": 12}
		},
		"energy_usage": "75kW"
	}`), &machine); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if machine.EnergyUsage == nil || *machine.EnergyUsage != 75000 {
		t.Errorf("EnergyUsage = %v, want 75000", machine.EnergyUsage)
	}
	if machine.EnergySource == nil || machine.EnergySource.Type != "burner" {
		t.Errorf("EnergySource = %+v", machine.EnergySource)
	}

	inserted, err := machine.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	cats, err := gobDecode[[]string](inserted.CraftingCategories)
	if err != nil {
		t.Fatalf("gobDecode categories: %v", err)
	}
	if len(cats) != 2 || cats[0] != "crafting" {
		t.Errorf("categories = %v", cats)
	}
	src, err := gobDecode[EnergySource](inserted.EnergySource)
	if err != nil {
		t.Fatalf("gobDecode energy source: %v", err)
	}
	if src.Type != "burner" || src.EmissionsPerMinute == nil || *src.EmissionsPerMinute != 12 {
		t.Errorf("energy source = %+v", src)
	}

	got, err := machine.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if !rowsEqual(got, inserted) {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}
}

func TestAssemblyMachineFixedRecipeStub(t *testing.T) {
	cfg := testState(t)
	machine := AssemblyMachine{
		Entity:      Entity{Name: "oil-refinery", Type: "assembling-machine"},
		FixedRecipe: ptr("advanced-oil-processing"),
	}
	row, err := machine.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if !row.FixedRecipe.Valid {
		t.Fatal("FixedRecipe not set")
	}
	ent, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          "advanced-oil-processing",
		PrototypeType: "recipe",
	})
	if err != nil {
		t.Fatalf("recipe entity stub missing: %v", err)
	}
	recipe, err := cfg.DB.GetRecipeByEntityID(cfg.CTX, ent.ID)
	if err != nil {
		t.Fatalf("recipe row stub missing: %v", err)
	}
	if row.FixedRecipe.Int64 != recipe.ID {
		t.Errorf("FixedRecipe = %d, want recipe id %d", row.FixedRecipe.Int64, recipe.ID)
	}

	full := Recipe{
		Entity:         Entity{Name: "advanced-oil-processing", Type: "recipe"},
		EnergyRequired: ptr(5.0),
		Category:       ptr("oil-processing"),
	}
	updated, err := full.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB real recipe: %v", err)
	}
	if updated.ID != recipe.ID {
		t.Errorf("updated recipe ID = %d, want stub id %d", updated.ID, recipe.ID)
	}
	if !updated.EnergyRequired.Valid || updated.EnergyRequired.Float64 != 5 {
		t.Errorf("EnergyRequired = %+v, want 5", updated.EnergyRequired)
	}
}

func TestAssemblyMachineAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var machine *AssemblyMachine
	_, err := machine.AddToDB(cfg)
	var nilErr DatatypeNilError[AssemblyMachine]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}
