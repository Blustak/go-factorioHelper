package datatypes

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestItemUnmarshalJSON(t *testing.T) {
	var item Item
	raw := []byte(`{
		"type": "item",
		"name": "wood",
		"order": "a[wood]",
		"stack_size": 100,
		"weight": 2000,
		"fuel_value": "2MJ",
		"fuel_category": "biomass"
	}`)
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if item.Name != "wood" || item.Type != "item" {
		t.Errorf("entity = %s/%s, want wood/item", item.Name, item.Type)
	}
	if item.StackSize == nil || *item.StackSize != 100 {
		t.Errorf("StackSize = %v, want 100", item.StackSize)
	}
	if item.FuelValue == nil || *item.FuelValue != 2e6 {
		t.Errorf("FuelValue = %v, want 2e6", item.FuelValue)
	}
	if item.FuelCategory == nil || *item.FuelCategory != "biomass" {
		t.Errorf("FuelCategory = %v, want biomass", item.FuelCategory)
	}
}

func TestItemFuelCategoryDefaultsToChemical(t *testing.T) {
	var item Item
	if err := json.Unmarshal([]byte(`{
		"type": "item",
		"name": "coal",
		"fuel_value": "4MJ"
	}`), &item); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if item.FuelCategory == nil || *item.FuelCategory != DefaultFuelCategory {
		t.Errorf("FuelCategory = %v, want %q", item.FuelCategory, DefaultFuelCategory)
	}
}

func TestItemNoFuelOmitsCategory(t *testing.T) {
	var item Item
	if err := json.Unmarshal([]byte(`{
		"type": "item",
		"name": "iron-plate",
		"stack_size": 100
	}`), &item); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if item.FuelCategory != nil {
		t.Errorf("FuelCategory = %v, want nil", item.FuelCategory)
	}
}

func TestItemAddGetRoundTrip(t *testing.T) {
	cfg := testState(t)
	var item Item
	if err := json.Unmarshal([]byte(`{
		"type": "item",
		"name": "coal",
		"order": "b[coal]",
		"stack_size": 50,
		"weight": 2000,
		"fuel_value": "4MJ",
		"burnt_result": "ash"
	}`), &item); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	inserted, err := item.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if !inserted.BurntResult.Valid {
		t.Fatal("BurntResult not set")
	}
	ash, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          "ash",
		PrototypeType: "item",
	})
	if err != nil {
		t.Fatalf("ash stub missing: %v", err)
	}
	if inserted.BurntResult.Int64 != ash.ID {
		t.Errorf("BurntResult = %d, want ash id %d", inserted.BurntResult.Int64, ash.ID)
	}
	if !inserted.FuelCategory.Valid || inserted.FuelCategory.String != DefaultFuelCategory {
		t.Errorf("FuelCategory = %+v, want %q", inserted.FuelCategory, DefaultFuelCategory)
	}

	got, err := item.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if !rowsEqual(got, inserted) {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}

	unwrapped, err := item.Unwrap()
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if !rowsEqual(unwrapped, inserted) {
		t.Errorf("Unwrap() = %+v, want %+v", unwrapped, inserted)
	}
}

func TestItemAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var item *Item
	_, err := item.AddToDB(cfg)
	var nilErr DatatypeNilError[Item]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}

func TestItemGetFromDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var item *Item
	_, err := item.GetFromDB(cfg)
	var nilErr DatatypeNilError[Item]
	if !errors.As(err, &nilErr) {
		t.Fatalf("GetFromDB error = %v, want DatatypeNilError", err)
	}
}

func TestItemStubThenRealInsert(t *testing.T) {
	cfg := testState(t)
	coal := Item{
		Entity:      Entity{Name: "coal", Type: "item"},
		BurntResult: ptr("ash"),
		StackSize:   ptr(int64(50)),
	}
	if _, err := coal.AddToDB(cfg); err != nil {
		t.Fatalf("AddToDB coal: %v", err)
	}

	order := "a[ash]"
	ash := Item{
		Entity:    Entity{Name: "ash", Type: "item", EntityOrder: &order},
		StackSize: ptr(int64(100)),
	}
	row, err := ash.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB ash: %v", err)
	}
	if row.StackSize.Int64 != 100 {
		t.Errorf("ash StackSize = %v, want 100", row.StackSize)
	}
	unwrapped, err := ash.Unwrap()
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if !rowsEqual(unwrapped, row) {
		t.Errorf("Unwrap() = %+v, want %+v", unwrapped, row)
	}
	id, err := ash.GetEntityID(cfg)
	if err != nil {
		t.Fatalf("GetEntityID: %v", err)
	}
	stub, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          "ash",
		PrototypeType: "item",
	})
	if err != nil {
		t.Fatalf("get ash entity: %v", err)
	}
	if id != stub.ID {
		t.Errorf("GetEntityID = %d, want stub id %d", id, stub.ID)
	}
	if !stub.EntityOrder.Valid || stub.EntityOrder.String != order {
		t.Errorf("stub order = %+v, want %q", stub.EntityOrder, order)
	}
}
