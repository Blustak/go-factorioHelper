package datatypes

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestResourceProducerMiningDrillRoundTrip(t *testing.T) {
	cfg := testState(t)
	var producer ResourceProducer
	if err := json.Unmarshal([]byte(`{
		"type": "mining-drill",
		"name": "electric-mining-drill",
		"resource_categories": ["basic-solid"],
		"mining_speed": 0.5,
		"energy_source": {
			"type": "electric",
			"usage_priority": "secondary-input"
		},
		"energy_usage": "90kW"
	}`), &producer); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if producer.EnergyUsage == nil || *producer.EnergyUsage != 90000 {
		t.Errorf("EnergyUsage = %v, want 90000", producer.EnergyUsage)
	}
	if producer.MiningSpeed == nil || *producer.MiningSpeed != 0.5 {
		t.Errorf("MiningSpeed = %v, want 0.5", producer.MiningSpeed)
	}

	inserted, err := producer.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	cats, err := gobDecode[[]string](inserted.ResourceCategories)
	if err != nil {
		t.Fatalf("gobDecode categories: %v", err)
	}
	if len(cats) != 1 || cats[0] != "basic-solid" {
		t.Errorf("categories = %v", cats)
	}

	got, err := producer.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if !rowsEqual(got, inserted) {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}
}

func TestResourceProducerOffshorePumpFluidBoxFilter(t *testing.T) {
	cfg := testState(t)
	var producer ResourceProducer
	if err := json.Unmarshal([]byte(`{
		"type": "offshore-pump",
		"name": "offshore-pump",
		"pumping_speed": 20,
		"fluid_box": {"filter": "water"},
		"energy_source": {"type": "electric", "usage_priority": "secondary-input"},
		"energy_usage": "60kW"
	}`), &producer); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if producer.ProducedFluid == nil || *producer.ProducedFluid != "water" {
		t.Errorf("ProducedFluid = %v, want water", producer.ProducedFluid)
	}
	if producer.PumpingSpeed == nil || *producer.PumpingSpeed != 20 {
		t.Errorf("PumpingSpeed = %v, want 20", producer.PumpingSpeed)
	}

	row, err := producer.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if !row.ProducedFluid.Valid {
		t.Fatal("ProducedFluid not set")
	}
	ent, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          "water",
		PrototypeType: "fluid",
	})
	if err != nil {
		t.Fatalf("water stub missing: %v", err)
	}
	if row.ProducedFluid.Int64 != ent.ID {
		t.Errorf("ProducedFluid = %d, want %d", row.ProducedFluid.Int64, ent.ID)
	}
}

func TestResourceProducerLegacyFluidField(t *testing.T) {
	var producer ResourceProducer
	if err := json.Unmarshal([]byte(`{
		"type": "offshore-pump",
		"name": "offshore-pump",
		"pumping_speed": 20,
		"fluid": "water"
	}`), &producer); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if producer.ProducedFluid == nil || *producer.ProducedFluid != "water" {
		t.Errorf("ProducedFluid = %v, want water", producer.ProducedFluid)
	}
}

func TestResourceProducerAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var producer *ResourceProducer
	_, err := producer.AddToDB(cfg)
	var nilErr DatatypeNilError[ResourceProducer]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}
