package datatypes

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestBoilerBurnerRoundTrip(t *testing.T) {
	cfg := testState(t)
	var boiler Boiler
	if err := json.Unmarshal([]byte(`{
		"type": "boiler",
		"name": "boiler",
		"energy_source": {
			"type": "burner",
			"fuel_categories": ["chemical"],
			"effectivity": 1,
			"fuel_inventory_size": 1,
			"emissions_per_minute": {"pollution": 30}
		},
		"energy_consumption": "1.8MW",
		"target_temperature": 165,
		"mode": "output-to-separate-pipe",
		"fluid_box": {"filter": "water"},
		"output_fluid_box": {"filter": "steam"}
	}`), &boiler); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if boiler.EnergyConsumption == nil || *boiler.EnergyConsumption != 1.8e6 {
		t.Errorf("EnergyConsumption = %v, want 1.8e6", boiler.EnergyConsumption)
	}
	if boiler.InputFluid == nil || *boiler.InputFluid != "water" {
		t.Errorf("InputFluid = %v, want water", boiler.InputFluid)
	}
	if boiler.OutputFluid == nil || *boiler.OutputFluid != "steam" {
		t.Errorf("OutputFluid = %v, want steam", boiler.OutputFluid)
	}
	if boiler.EnergySource == nil || boiler.EnergySource.Type != "burner" {
		t.Errorf("EnergySource = %+v", boiler.EnergySource)
	}
	if len(boiler.EnergySource.FuelCategories) != 1 || boiler.EnergySource.FuelCategories[0] != "chemical" {
		t.Errorf("FuelCategories = %v, want [chemical]", boiler.EnergySource.FuelCategories)
	}

	inserted, err := boiler.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if !inserted.InputFluid.Valid || !inserted.OutputFluid.Valid {
		t.Fatal("fluid FKs not set")
	}
	water, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name: "water", PrototypeType: "fluid",
	})
	if err != nil {
		t.Fatalf("water stub: %v", err)
	}
	if inserted.InputFluid.Int64 != water.ID {
		t.Errorf("InputFluid = %d, want %d", inserted.InputFluid.Int64, water.ID)
	}

	got, err := boiler.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if !rowsEqual(got, inserted) {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}
}

func TestBoilerHeatExchangerHasNoFuelCategories(t *testing.T) {
	var boiler Boiler
	if err := json.Unmarshal([]byte(`{
		"type": "boiler",
		"name": "heat-exchanger",
		"energy_source": {"type": "heat", "max_temperature": 1000},
		"energy_consumption": "10MW",
		"target_temperature": 500,
		"fluid_box": {"filter": "water"},
		"output_fluid_box": {"filter": "steam"}
	}`), &boiler); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if boiler.EnergySource == nil || boiler.EnergySource.Type != "heat" {
		t.Errorf("EnergySource = %+v, want heat", boiler.EnergySource)
	}
	if len(boiler.EnergySource.FuelCategories) != 0 {
		t.Errorf("FuelCategories = %v, want empty", boiler.EnergySource.FuelCategories)
	}
}

func TestBoilerAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var boiler *Boiler
	_, err := boiler.AddToDB(cfg)
	var nilErr DatatypeNilError[Boiler]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}
