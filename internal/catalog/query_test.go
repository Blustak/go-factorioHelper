package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/dump"
	"github.com/Blustak/go-factorioHelper/sql/schema"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

func testQueryState(t *testing.T) (*config.State, *bytes.Buffer) {
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

	buf := &bytes.Buffer{}
	cfg := config.New(ctx, db, slog.New(slog.NewTextHandler(io.Discard, nil)), buf)
	if _, err := dump.Load(cfg, dumpFixture(t)); err != nil {
		t.Fatalf("dump.Load: %v", err)
	}
	return cfg, buf
}

func dumpFixture(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "dump", "testdata", "dump.json")
}

func TestQueryItems(t *testing.T) {
	cfg, buf := testQueryState(t)
	cat := "items"
	if err := (&Catalog{}).Query(cfg, &cat, "wood"); err != nil {
		t.Fatalf("Query items: %v", err)
	}
	got := buf.String()
	want := "[item] Name: wood, Stack Size: 100, Fuel Value: 2000000.00J\n"
	if got != want {
		t.Fatalf("Query items output = %q, want %q", got, want)
	}
}

func TestQueryRecipes(t *testing.T) {
	cfg, buf := testQueryState(t)
	cat := "recipes"
	if err := (&Catalog{}).Query(cfg, &cat, "wood"); err != nil {
		t.Fatalf("Query recipes: %v", err)
	}
	got := buf.String()
	want := "[recipe] Name: wood, Category: crafting, Energy: 0.50s, Ingredients: wood x1, Results: wood x1\n"
	if got != want {
		t.Fatalf("Query recipes output = %q, want %q", got, want)
	}
}

func TestQueryAll(t *testing.T) {
	cfg, buf := testQueryState(t)
	if err := (&Catalog{}).Query(cfg, nil, "wood"); err != nil {
		t.Fatalf("Query all: %v", err)
	}
	got := buf.String()
	want := "[item] Name: wood, Stack Size: 100, Fuel Value: 2000000.00J\n" +
		"[recipe] Name: wood, Category: crafting, Energy: 0.50s, Ingredients: wood x1, Results: wood x1\n"
	if got != want {
		t.Fatalf("Query all output = %q, want %q", got, want)
	}
}

func TestQueryUnknownCategory(t *testing.T) {
	cfg, _ := testQueryState(t)
	cat := "fluids"
	err := (&Catalog{}).Query(cfg, &cat, "wood")
	if err == nil {
		t.Fatal("Query unknown category error = nil, want unknown category")
	}
	if !strings.Contains(err.Error(), "unknown category") {
		t.Fatalf("Query error = %v, want unknown category", err)
	}
}

func TestQueryNilConfig(t *testing.T) {
	err := (&Catalog{}).Query(nil, nil, "wood")
	if err == nil {
		t.Fatal("Query nil config error = nil, want nil config")
	}
	if !strings.Contains(err.Error(), "nil config") {
		t.Fatalf("Query error = %v, want nil config", err)
	}
}

func TestQueryEmptyMatchesAll(t *testing.T) {
	cfg, buf := testQueryState(t)
	if err := (&Catalog{}).Query(cfg, nil, ""); err != nil {
		t.Fatalf("Query empty: %v", err)
	}
	got := buf.String()
	want := "[item] Name: wood, Stack Size: 100, Fuel Value: 2000000.00J\n" +
		"[recipe] Name: wood, Category: crafting, Energy: 0.50s, Ingredients: wood x1, Results: wood x1\n"
	if got != want {
		t.Fatalf("Query empty output = %q, want %q", got, want)
	}
}

func TestQueryEmptyItems(t *testing.T) {
	cfg, buf := testQueryState(t)
	cat := "items"
	if err := (&Catalog{}).Query(cfg, &cat, ""); err != nil {
		t.Fatalf("Query empty items: %v", err)
	}
	got := buf.String()
	want := "[item] Name: wood, Stack Size: 100, Fuel Value: 2000000.00J\n"
	if got != want {
		t.Fatalf("Query empty items output = %q, want %q", got, want)
	}
}

func TestQueryNoMatches(t *testing.T) {
	cfg, buf := testQueryState(t)
	if err := (&Catalog{}).Query(cfg, nil, "zzzz-missing"); err != nil {
		t.Fatalf("Query no matches: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("Query no matches output = %q, want empty", got)
	}
}
