package dump

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/sql/schema"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

func testState(t *testing.T) *config.State {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, schema.Migrations)
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	return config.New(ctx, db, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func TestLoadFixture(t *testing.T) {
	cfg := testState(t)
	stats, err := Load(cfg, testdataPath(t, "dump.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stats.Items != 1 || stats.Fluids != 1 || stats.Resources != 1 || stats.Recipes != 1 || stats.AssemblyMachines != 1 || stats.Furnaces != 1 || stats.ResourceProducers != 3 {
		t.Errorf("stats typed = %+v, want 1 of each plus 3 producers", stats)
	}
	if stats.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", stats.Skipped)
	}

	assertTypedCounts(t, cfg)

	ents, err := cfg.DB.GetAllEntities(cfg.CTX)
	if err != nil {
		t.Fatalf("GetAllEntities: %v", err)
	}
	var sawFurnace, sawDrill bool
	wantLocalised := map[string]string{
		"recipe:wood": "Wood",
		"item:wood":   "Wood",
		"assembling-machine:assembling-machine-1": "Assembling machine 1 (Legacy)",
		"furnace:stone-furnace":                   "Stone furnace",
	}
	for _, ent := range ents {
		key := ent.PrototypeType + ":" + ent.Name
		if want, ok := wantLocalised[key]; ok && ent.LocalisedName != want {
			t.Errorf("%s localised_name = %q, want %q", key, ent.LocalisedName, want)
		}
		if ent.PrototypeType == "furnace" && ent.Name == "stone-furnace" {
			sawFurnace = true
		}
		if ent.PrototypeType == "mining-drill" && ent.Name == "electric-mining-drill" {
			sawDrill = true
		}
		if ent.PrototypeType == "lab" {
			t.Fatalf("lab prototype stored as entity: %+v", ent)
		}
	}
	if !sawFurnace {
		t.Fatal("stone-furnace entity missing")
	}
	if !sawDrill {
		t.Fatal("electric-mining-drill entity missing")
	}

	again, err := Load(cfg, testdataPath(t, "dump.json"))
	if err != nil {
		t.Fatalf("Load upsert: %v", err)
	}
	if again.Items != 1 || again.Fluids != 1 || again.Resources != 1 || again.Recipes != 1 || again.AssemblyMachines != 1 || again.Furnaces != 1 || again.ResourceProducers != 3 {
		t.Errorf("upsert stats typed = %+v, want 1 of each plus 3 producers", again)
	}
	assertTypedCounts(t, cfg)
}

func TestLoadMissingFile(t *testing.T) {
	cfg := testState(t)
	_, err := Load(cfg, filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("Load error = nil, want missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load error = %v, want os.ErrNotExist", err)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	cfg := testState(t)
	_, err := Load(cfg, testdataPath(t, "invalid.json"))
	if err == nil {
		t.Fatal("Load error = nil, want invalid JSON")
	}
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("Load error = %v, want json.SyntaxError", err)
	}
}

func assertTypedCounts(t *testing.T, cfg *config.State) {
	t.Helper()
	items, err := cfg.DB.GetAllItemValues(cfg.CTX)
	assertLen(t, "items", 1, items, err)
	fluids, err := cfg.DB.GetAllFluidValues(cfg.CTX)
	assertLen(t, "fluids", 1, fluids, err)
	resources, err := cfg.DB.GetAllResourceValues(cfg.CTX)
	assertLen(t, "resources", 1, resources, err)
	recipes, err := cfg.DB.GetAllRecipeValues(cfg.CTX)
	assertLen(t, "recipes", 1, recipes, err)
	machines, err := cfg.DB.GetAllAssemblyMachineValues(cfg.CTX)
	assertLen(t, "assembling machines", 2, machines, err)
	producers, err := cfg.DB.GetAllResourceProducerValues(cfg.CTX)
	assertLen(t, "resource producers", 3, producers, err)
}

func assertLen[T any](t *testing.T, kind string, want int, rows []T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", kind, err)
	}
	if len(rows) != want {
		t.Errorf("%s count = %d, want %d", kind, len(rows), want)
	}
}
