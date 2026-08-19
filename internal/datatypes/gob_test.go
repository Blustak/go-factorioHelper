package datatypes

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGobIngredientRoundTrip(t *testing.T) {
	amount := 8.0
	in := []Ingredient{{
		Entity: Entity{Name: "iron-ore", Type: "item"},
		Amount: &amount,
	}}
	b, err := gobEncode(in)
	if err != nil {
		t.Fatalf("gobEncode: %v", err)
	}
	got, err := gobDecode[[]Ingredient](b)
	if err != nil {
		t.Fatalf("gobDecode: %v", err)
	}
	if !reflect.DeepEqual(got, in) {
		t.Errorf("gob round-trip = %+v, want %+v", got, in)
	}
}

func TestGobMinableResultsRoundTrip(t *testing.T) {
	amount := 1.0
	fluid := 25.0
	in := MinableResults{
		Results: []Product{{
			Entity: Entity{Name: "iron-ore", Type: "item"},
			Amount: &amount,
		}},
		FluidAmount: &fluid,
	}
	b, err := gobEncode(in)
	if err != nil {
		t.Fatalf("gobEncode: %v", err)
	}
	got, err := gobDecode[MinableResults](b)
	if err != nil {
		t.Fatalf("gobDecode: %v", err)
	}
	if !reflect.DeepEqual(got, in) {
		t.Errorf("gob round-trip = %+v, want %+v", got, in)
	}
}

func TestGobEnergySourceRoundTrip(t *testing.T) {
	raw := []byte(`{
		"type": "burner",
		"fuel_categories": ["chemical", "biomass"],
		"effectivity": 1,
		"fuel_inventory_size": 1,
		"burnt_inventory_size": 1,
		"emissions_per_minute": {"pollution": 12}
	}`)
	var src EnergySource
	if err := json.Unmarshal(raw, &src); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	b, err := gobEncode(src)
	if err != nil {
		t.Fatalf("gobEncode: %v", err)
	}
	got, err := gobDecode[EnergySource](b)
	if err != nil {
		t.Fatalf("gobDecode: %v", err)
	}
	if !reflect.DeepEqual(got, src) {
		t.Errorf("gob round-trip = %+v, want %+v", got, src)
	}
}

func TestGobDecodeEmpty(t *testing.T) {
	got, err := gobDecode[[]Ingredient](nil)
	if err != nil {
		t.Fatalf("gobDecode nil: %v", err)
	}
	if got != nil {
		t.Errorf("gobDecode(nil) = %+v, want nil", got)
	}
}
