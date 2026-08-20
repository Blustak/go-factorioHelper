package datatypes

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestGeneratorSteamEngineRoundTrip(t *testing.T) {
	cfg := testState(t)
	var generator Generator
	if err := json.Unmarshal([]byte(`{
		"type": "generator",
		"name": "steam-engine",
		"effectivity": 1,
		"fluid_usage_per_tick": 0.5,
		"maximum_temperature": 165,
		"burns_fluid": false,
		"fluid_box": {"filter": "steam"},
		"energy_source": {
			"type": "electric",
			"usage_priority": "secondary-output"
		}
	}`), &generator); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if generator.InputFluid == nil || *generator.InputFluid != "steam" {
		t.Errorf("InputFluid = %v, want steam", generator.InputFluid)
	}
	if generator.FluidUsagePerTick == nil || *generator.FluidUsagePerTick != 0.5 {
		t.Errorf("FluidUsagePerTick = %v, want 0.5", generator.FluidUsagePerTick)
	}
	if generator.BurnsFluid == nil || *generator.BurnsFluid {
		t.Errorf("BurnsFluid = %v, want false", generator.BurnsFluid)
	}

	inserted, err := generator.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if !inserted.InputFluid.Valid {
		t.Fatal("InputFluid not set")
	}
	steam, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name: "steam", PrototypeType: "fluid",
	})
	if err != nil {
		t.Fatalf("steam stub: %v", err)
	}
	if inserted.InputFluid.Int64 != steam.ID {
		t.Errorf("InputFluid = %d, want %d", inserted.InputFluid.Int64, steam.ID)
	}
	if !inserted.BurnsFluid.Valid || inserted.BurnsFluid.Int64 != 0 {
		t.Errorf("BurnsFluid = %+v, want 0", inserted.BurnsFluid)
	}

	got, err := generator.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if !rowsEqual(got, inserted) {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}
}

func TestGeneratorAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var generator *Generator
	_, err := generator.AddToDB(cfg)
	var nilErr DatatypeNilError[Generator]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}
