package datatypes

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestResourceUnmarshalLegacyResult(t *testing.T) {
	var res Resource
	if err := json.Unmarshal([]byte(`{
		"type": "resource",
		"name": "coal",
		"order": "a-b-b",
		"minable": {"mining_time": 1, "result": "coal"}
	}`), &res); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if res.MiningTime == nil || *res.MiningTime != 1 {
		t.Errorf("MiningTime = %v, want 1", res.MiningTime)
	}
	if len(res.Minable.Results) != 1 || res.Minable.Results[0].Name != "coal" {
		t.Errorf("Results = %+v, want coal product", res.Minable.Results)
	}
}

func TestResourceAddGetRoundTripWithFluid(t *testing.T) {
	cfg := testState(t)
	var res Resource
	if err := json.Unmarshal([]byte(`{
		"type": "resource",
		"name": "borax",
		"minable": {
			"mining_time": 1.5,
			"fluid_amount": 25,
			"required_fluid": "syngas",
			"results": [{"type": "item", "name": "borax", "amount": 1}]
		}
	}`), &res); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	inserted, err := res.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if !inserted.RequiredFluid.Valid {
		t.Fatal("RequiredFluid not set")
	}
	fluid, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          "syngas",
		PrototypeType: "fluid",
	})
	if err != nil {
		t.Fatalf("syngas stub missing: %v", err)
	}
	if inserted.RequiredFluid.Int64 != fluid.ID {
		t.Errorf("RequiredFluid = %d, want %d", inserted.RequiredFluid.Int64, fluid.ID)
	}

	decoded, err := gobDecode[MinableResults](inserted.Results)
	if err != nil {
		t.Fatalf("gobDecode results: %v", err)
	}
	if decoded.FluidAmount == nil || *decoded.FluidAmount != 25 {
		t.Errorf("FluidAmount = %v, want 25", decoded.FluidAmount)
	}
	if len(decoded.Results) != 1 || decoded.Results[0].Name != "borax" {
		t.Errorf("decoded results = %+v", decoded.Results)
	}

	got, err := res.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if !rowsEqual(got, inserted) {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}
}

func TestResourceAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var res *Resource
	_, err := res.AddToDB(cfg)
	var nilErr DatatypeNilError[Resource]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}
