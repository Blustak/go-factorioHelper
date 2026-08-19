package datatypes

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestFluidUnmarshalAndRoundTrip(t *testing.T) {
	cfg := testState(t)
	var fluid Fluid
	if err := json.Unmarshal([]byte(`{
		"type": "fluid",
		"name": "crude-oil",
		"order": "a[fluid]-b[oil]-a[crude-oil]",
		"fuel_value": "82.5kJ",
		"default_temperature": 25
	}`), &fluid); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if fluid.FuelValue == nil || *fluid.FuelValue != 82500 {
		t.Errorf("FuelValue = %v, want 82500", fluid.FuelValue)
	}
	if fluid.DefaultTemperature == nil || *fluid.DefaultTemperature != 25 {
		t.Errorf("DefaultTemperature = %v, want 25", fluid.DefaultTemperature)
	}

	inserted, err := fluid.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	got, err := fluid.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if got != inserted {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}
}

func TestFluidAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var fluid *Fluid
	_, err := fluid.AddToDB(cfg)
	var nilErr DatatypeNilError[Fluid]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}
