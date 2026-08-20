package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

func TestEntityAddToDBNameOnly(t *testing.T) {
	cfg := testState(t)
	e := &Entity{Name: "iron-plate", Type: "item"}

	got, err := e.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if got.Name != "iron-plate" {
		t.Errorf("Name = %q, want iron-plate", got.Name)
	}
	if got.LocalisedName != "Iron plate" {
		t.Errorf("LocalisedName = %q, want Iron plate", got.LocalisedName)
	}
	if got.EntityOrder.Valid {
		t.Errorf("EntityOrder.Valid = true, want false")
	}
	if got.ID == 0 {
		t.Error("ID is 0, want non-zero")
	}
	unwrapped, err := e.Unwrap()
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if unwrapped != got {
		t.Errorf("Unwrap() = %+v, want %+v", unwrapped, got)
	}
}

func TestEntityAddToDBWithOrder(t *testing.T) {
	cfg := testState(t)
	order := "a[iron-plate]"
	e := &Entity{Name: "iron-plate", Type: "item", EntityOrder: &order}

	got, err := e.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}
	if got.Name != "iron-plate" {
		t.Errorf("Name = %q, want iron-plate", got.Name)
	}
	if !got.EntityOrder.Valid || got.EntityOrder.String != order {
		t.Errorf("EntityOrder = %+v, want valid %q", got.EntityOrder, order)
	}
}

func TestEntityAddToDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var e *Entity
	_, err := e.AddToDB(cfg)
	var nilErr DatatypeNilError[Entity]
	if !errors.As(err, &nilErr) {
		t.Fatalf("AddToDB error = %v, want DatatypeNilError", err)
	}
}

func TestEntityGetFromDBRoundTrip(t *testing.T) {
	cfg := testState(t)
	order := "a[copper-plate]"
	e := &Entity{Name: "copper-plate", Type: "item", EntityOrder: &order}
	inserted, err := e.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}

	got, err := e.GetFromDB(cfg)
	if err != nil {
		t.Fatalf("GetFromDB: %v", err)
	}
	if got != inserted {
		t.Errorf("GetFromDB() = %+v, want %+v", got, inserted)
	}
}

func TestEntityGetFromDBMismatch(t *testing.T) {
	cfg := testState(t)
	order := "a[steel-plate]"
	e := &Entity{Name: "steel-plate", Type: "item", EntityOrder: &order}
	if _, err := e.AddToDB(cfg); err != nil {
		t.Fatalf("AddToDB: %v", err)
	}

	updatedOrder := sql.NullString{String: "b[steel-plate]", Valid: true}
	if _, err := cfg.DB.UpdateEntityOrderByName(cfg.CTX, database.UpdateEntityOrderByNameParams{
		Order:         updatedOrder,
		Name:          "steel-plate",
		PrototypeType: "item",
	}); err != nil {
		t.Fatalf("UpdateEntityOrderByName: %v", err)
	}

	_, err := e.GetFromDB(cfg)
	if err == nil {
		t.Fatal("GetFromDB error = nil, want mismatch")
	}
}

func TestEntityGetFromDBMissingName(t *testing.T) {
	cfg := testState(t)
	e := &Entity{Name: "does-not-exist", Type: "item"}
	_, err := e.GetFromDB(cfg)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetFromDB error = %v, want sql.ErrNoRows", err)
	}
}

func TestEntityGetFromDBNilReceiver(t *testing.T) {
	cfg := testState(t)
	var e *Entity
	_, err := e.GetFromDB(cfg)
	var nilErr DatatypeNilError[Entity]
	if !errors.As(err, &nilErr) {
		t.Fatalf("GetFromDB error = %v, want DatatypeNilError", err)
	}
}

func TestEntityUnwrapAndGetEntityID(t *testing.T) {
	cfg := testState(t)
	e := &Entity{Name: "electronic-circuit", Type: "item"}
	inserted, err := e.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB: %v", err)
	}

	unwrapped, err := e.Unwrap()
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if unwrapped != inserted {
		t.Errorf("Unwrap() = %+v, want %+v", unwrapped, inserted)
	}

	id, err := e.GetEntityID(cfg)
	if err != nil {
		t.Fatalf("GetEntityID: %v", err)
	}
	if id != inserted.ID {
		t.Errorf("GetEntityID() = %d, want %d", id, inserted.ID)
	}
}

func TestEntityUnwrapUnsetEntry(t *testing.T) {
	e := &Entity{Name: "advanced-circuit"}
	_, err := e.Unwrap()
	if err == nil {
		t.Fatal("Unwrap error = nil, want entEntry not set")
	}
}

func TestEntityUnwrapNilReceiver(t *testing.T) {
	var e *Entity
	_, err := e.Unwrap()
	var nilErr DatatypeNilError[Entity]
	if !errors.As(err, &nilErr) {
		t.Fatalf("Unwrap error = %v, want DatatypeNilError", err)
	}
}

func TestEntityGetEntityIDUnsetEntry(t *testing.T) {
	cfg := testState(t)
	e := &Entity{Name: "processing-unit"}
	_, err := e.GetEntityID(cfg)
	if err == nil {
		t.Fatal("GetEntityID error = nil, want unset entEntry")
	}
}

func TestEntityGetEntityIDNilReceiver(t *testing.T) {
	cfg := testState(t)
	var e *Entity
	_, err := e.GetEntityID(cfg)
	var nilErr DatatypeNilError[Entity]
	if !errors.As(err, &nilErr) {
		t.Fatalf("GetEntityID error = %v, want DatatypeNilError", err)
	}
}

func TestEntityUnmarshalJSONNameOnly(t *testing.T) {
	var e Entity
	if err := json.Unmarshal([]byte(`{"name":"iron-ore"}`), &e); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if e.Name != "iron-ore" {
		t.Errorf("Name = %q, want iron-ore", e.Name)
	}
	if e.EntityOrder != nil {
		t.Errorf("EntityOrder = %v, want nil", e.EntityOrder)
	}
}

func TestEntityUnmarshalJSONWithLocalisedName(t *testing.T) {
	var e Entity
	raw := []byte(`{"name":"pipette-dino-dig-site","type":"mining-drill","localised_name":["entity-name.dino-dig-site"]}`)
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if Display(e.localisedRaw, e.Name) != "Dino dig site" {
		t.Errorf("display = %q, want Dino dig site", Display(e.localisedRaw, e.Name))
	}
}

func TestEntityAddToDBFillsLocalisedNameFromDump(t *testing.T) {
	cfg := testState(t)
	stub := &Entity{Name: "pipette-dino-dig-site", Type: "mining-drill"}
	first, err := stub.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB stub: %v", err)
	}
	if first.LocalisedName != "Pipette dino dig site" {
		t.Errorf("stub LocalisedName = %q, want Pipette dino dig site", first.LocalisedName)
	}

	var full Entity
	if err := json.Unmarshal([]byte(`{
		"name": "pipette-dino-dig-site",
		"type": "mining-drill",
		"localised_name": ["entity-name.dino-dig-site"]
	}`), &full); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	second, err := full.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB full: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("ID = %d, want %d", second.ID, first.ID)
	}
	if second.LocalisedName != "Dino dig site" {
		t.Errorf("LocalisedName = %q, want Dino dig site", second.LocalisedName)
	}

	clobber := &Entity{Name: "pipette-dino-dig-site", Type: "mining-drill"}
	third, err := clobber.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB stub again: %v", err)
	}
	if third.LocalisedName != "Dino dig site" {
		t.Errorf("stub clobber LocalisedName = %q, want Dino dig site", third.LocalisedName)
	}
}

func TestEntityUnmarshalJSONWithOrder(t *testing.T) {
	var e Entity
	if err := json.Unmarshal([]byte(`{"name":"copper-ore","order":"a[copper-ore]"}`), &e); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if e.Name != "copper-ore" {
		t.Errorf("Name = %q, want copper-ore", e.Name)
	}
	if e.EntityOrder == nil || *e.EntityOrder != "a[copper-ore]" {
		t.Errorf("EntityOrder = %v, want a[copper-ore]", e.EntityOrder)
	}
}

func TestEntityAddToDBRequiresType(t *testing.T) {
	cfg := testState(t)
	e := &Entity{Name: "iron-plate"}
	if _, err := e.AddToDB(cfg); err == nil {
		t.Fatal("AddToDB error = nil, want type required")
	}
}

func TestEntityAddToDBGetOrCreateFillsOrder(t *testing.T) {
	cfg := testState(t)
	stub := &Entity{Name: "ash", Type: "item"}
	first, err := stub.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB stub: %v", err)
	}

	order := "a[ash]"
	full := &Entity{Name: "ash", Type: "item", EntityOrder: &order}
	second, err := full.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB fill: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("ID = %d, want %d (same entity)", second.ID, first.ID)
	}
	if !second.EntityOrder.Valid || second.EntityOrder.String != order {
		t.Errorf("EntityOrder = %+v, want valid %q", second.EntityOrder, order)
	}
}

func TestEntitySameNameDifferentType(t *testing.T) {
	cfg := testState(t)
	item := &Entity{Name: "iron-plate", Type: "item"}
	recipe := &Entity{Name: "iron-plate", Type: "recipe"}
	itemRow, err := item.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB item: %v", err)
	}
	recipeRow, err := recipe.AddToDB(cfg)
	if err != nil {
		t.Fatalf("AddToDB recipe: %v", err)
	}
	if itemRow.ID == recipeRow.ID {
		t.Fatal("item and recipe entities share an ID, want distinct rows")
	}
}
