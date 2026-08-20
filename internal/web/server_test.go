package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/chain"
	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/dump"
	"github.com/Blustak/go-factorioHelper/sql/schema"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

func testServer(t *testing.T) *Server {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, schema.Migrations)
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	cfg := config.New(ctx, db, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := dump.Load(cfg, dumpFixture(t)); err != nil {
		t.Fatalf("dump.Load: %v", err)
	}
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	return srv
}

func dumpFixture(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "dump", "testdata", "dump.json")
}

func TestGetRecipesDecodesIngredients(t *testing.T) {
	srv := testServer(t)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/recipes", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.Bytes())
	}
	var recipes []struct {
		Name          string `json:"name"`
		LocalisedName string `json:"localised_name"`
		Category      string `json:"category"`
		Ingredients   []struct {
			Name          string `json:"name"`
			Type          string `json:"type"`
			LocalisedName string `json:"localised_name"`
		} `json:"ingredients"`
		Products []struct {
			Name          string `json:"name"`
			Type          string `json:"type"`
			LocalisedName string `json:"localised_name"`
		} `json:"products"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &recipes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(recipes) != 1 || recipes[0].Name != "wood" {
		t.Fatalf("recipes = %+v", recipes)
	}
	if recipes[0].LocalisedName != "Wood" {
		t.Errorf("recipe localised_name = %q, want Wood", recipes[0].LocalisedName)
	}
	if recipes[0].Category != "crafting" {
		t.Errorf("category = %q, want crafting", recipes[0].Category)
	}
	if len(recipes[0].Ingredients) != 1 || recipes[0].Ingredients[0].Name != "wood" || recipes[0].Ingredients[0].LocalisedName != "Wood" {
		t.Errorf("ingredients = %+v", recipes[0].Ingredients)
	}
	if len(recipes[0].Products) != 1 || recipes[0].Products[0].Name != "wood" || recipes[0].Products[0].LocalisedName != "Wood" {
		t.Errorf("products = %+v", recipes[0].Products)
	}
}

func TestGetMachinesByCategory(t *testing.T) {
	srv := testServer(t)

	res := httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/machines?category=crafting", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var machines []struct {
		Name          string `json:"name"`
		LocalisedName string `json:"localised_name"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &machines); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(machines) != 1 || machines[0].Name != "assembling-machine-1" {
		t.Fatalf("crafting machines = %+v", machines)
	}
	if machines[0].LocalisedName != "Assembling machine 1 (Legacy)" {
		t.Errorf("crafting machine localised_name = %q, want Assembling machine 1 (Legacy)", machines[0].LocalisedName)
	}

	res = httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/machines?category=smelting", nil))
	if err := json.Unmarshal(res.Body.Bytes(), &machines); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(machines) != 1 || machines[0].Name != "stone-furnace" {
		t.Fatalf("smelting machines = %+v, want stone-furnace", machines)
	}
}

func TestValidateGraph(t *testing.T) {
	srv := testServer(t)
	valid := chain.Graph{
		Nodes: []chain.NodeDoc{
			{NodeID: "src", NodeKind: chain.KindSource, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: chain.KindRecipe, Recipe: "wood", Machine: "assembling-machine-1"},
			{NodeID: "snk", NodeKind: chain.KindSink, ItemName: "wood", PrototypeType: "item"},
		},
		Edges: []chain.Edge{
			{ID: "e1", FromNode: "src", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e2", FromNode: "rec", FromPort: "out:0", ToNode: "snk", ToPort: "in:0"},
		},
	}
	res := postValidate(t, srv, valid)
	if !res.OK {
		t.Fatalf("valid graph issues = %+v", res.Issues)
	}

	invalid := chain.Graph{
		Nodes: []chain.NodeDoc{
			{NodeID: "rec", NodeKind: chain.KindRecipe, Recipe: "wood"},
		},
	}
	res = postValidate(t, srv, invalid)
	if res.OK {
		t.Fatal("incomplete recipe graph reported ok")
	}
	found := false
	for _, issue := range res.Issues {
		if issue.Code == "required_input" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues = %+v, want required_input", res.Issues)
	}
}

func TestIndexAndStatic(t *testing.T) {
	srv := testServer(t)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("GET / status = %d", res.Code)
	}
	if !bytes.Contains(res.Body.Bytes(), []byte("Supply chain")) {
		t.Fatalf("GET / body missing title: %s", res.Body.Bytes())
	}

	res = httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/static/editor.js", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("GET /static/editor.js status = %d", res.Code)
	}
}

func TestGetItemsAndFluids(t *testing.T) {
	srv := testServer(t)

	res := httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/items", nil))
	var items []struct {
		Name          string   `json:"name"`
		Type          string   `json:"type"`
		LocalisedName string   `json:"localised_name"`
		FuelCategory  string   `json:"fuel_category"`
		FuelValue     *float64 `json:"fuel_value"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &items); err != nil {
		t.Fatalf("items: %v", err)
	}
	if len(items) != 1 || items[0].Name != "wood" || items[0].LocalisedName != "Wood" || items[0].FuelCategory != "chemical" {
		t.Fatalf("items = %+v", items)
	}
	if items[0].FuelValue == nil || *items[0].FuelValue != 2e6 {
		t.Fatalf("wood fuel_value = %v, want 2e6", items[0].FuelValue)
	}

	res = httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/fluids", nil))
	var fluids []struct {
		Name          string `json:"name"`
		Type          string `json:"type"`
		LocalisedName string `json:"localised_name"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &fluids); err != nil {
		t.Fatalf("fluids: %v", err)
	}
	if len(fluids) != 3 {
		t.Fatalf("fluids = %+v, want 3", fluids)
	}
	foundOil := false
	for _, f := range fluids {
		if f.Name == "crude-oil" {
			foundOil = true
			if f.Type != "fluid" || f.LocalisedName != "Crude oil" {
				t.Fatalf("crude-oil = %+v", f)
			}
		}
	}
	if !foundOil {
		t.Fatal("crude-oil missing from fluids")
	}
}

func TestGetProducersByCategory(t *testing.T) {
	srv := testServer(t)

	res := httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/producers", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var producers []struct {
		Name               string   `json:"name"`
		Type               string   `json:"type"`
		ResourceCategories []string `json:"resource_categories"`
		ProducedFluid      *string  `json:"produced_fluid"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &producers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(producers) != 3 {
		t.Fatalf("producers = %+v, want 3", producers)
	}

	res = httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/producers?category=basic-solid", nil))
	if err := json.Unmarshal(res.Body.Bytes(), &producers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(producers) != 1 || producers[0].Name != "electric-mining-drill" {
		t.Fatalf("basic-solid producers = %+v, want electric-mining-drill", producers)
	}

	foundPump := false
	res = httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/producers", nil))
	if err := json.Unmarshal(res.Body.Bytes(), &producers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, p := range producers {
		if p.Name == "offshore-pump" {
			foundPump = true
			if p.Type != "offshore-pump" {
				t.Errorf("offshore-pump type = %q", p.Type)
			}
			if p.ProducedFluid == nil || *p.ProducedFluid != "water" {
				t.Errorf("offshore-pump produced_fluid = %v, want water", p.ProducedFluid)
			}
		}
	}
	if !foundPump {
		t.Fatal("offshore-pump missing from producers")
	}
}

func TestGetBoilersAndGenerators(t *testing.T) {
	srv := testServer(t)

	res := httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/boilers", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var boilers []struct {
		Name           string   `json:"name"`
		FuelCategories []string `json:"fuel_categories"`
		InputFluid     *string  `json:"input_fluid"`
		OutputFluid    *string  `json:"output_fluid"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &boilers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(boilers) != 2 {
		t.Fatalf("boilers = %+v, want 2", boilers)
	}
	foundBurner := false
	foundHeat := false
	for _, b := range boilers {
		switch b.Name {
		case "boiler":
			foundBurner = true
			if len(b.FuelCategories) != 1 || b.FuelCategories[0] != "chemical" {
				t.Errorf("boiler fuel_categories = %v", b.FuelCategories)
			}
			if b.InputFluid == nil || *b.InputFluid != "water" || b.OutputFluid == nil || *b.OutputFluid != "steam" {
				t.Errorf("boiler fluids = %v -> %v", b.InputFluid, b.OutputFluid)
			}
		case "heat-exchanger":
			foundHeat = true
			if len(b.FuelCategories) != 0 {
				t.Errorf("heat-exchanger fuel_categories = %v", b.FuelCategories)
			}
		}
	}
	if !foundBurner || !foundHeat {
		t.Fatalf("boilers = %+v", boilers)
	}

	res = httptest.NewRecorder()
	srv.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/generators", nil))
	var generators []struct {
		Name       string  `json:"name"`
		InputFluid *string `json:"input_fluid"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &generators); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(generators) != 1 || generators[0].Name != "steam-engine" {
		t.Fatalf("generators = %+v", generators)
	}
	if generators[0].InputFluid == nil || *generators[0].InputFluid != "steam" {
		t.Errorf("steam-engine input_fluid = %v", generators[0].InputFluid)
	}
}

func postValidate(t *testing.T, srv *Server, g chain.Graph) chain.Result {
	t.Helper()
	body, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/graph/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate status = %d, body = %s", rec.Code, rec.Body.Bytes())
	}
	var out chain.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	return out
}

func TestAnalyzeGraph(t *testing.T) {
	srv := testServer(t)
	g := chain.Graph{
		Nodes: []chain.NodeDoc{
			{NodeID: "in", NodeKind: chain.KindInput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: chain.KindRecipe, Recipe: "wood", Machine: "assembling-machine-1"},
			{NodeID: "out", NodeKind: chain.KindOutput, ItemName: "wood", PrototypeType: "item"},
		},
		Edges: []chain.Edge{
			{ID: "e1", FromNode: "in", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e2", FromNode: "rec", FromPort: "out:0", ToNode: "out", ToPort: "in:0"},
		},
	}
	res := postAnalyze(t, srv, g)
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	wantProd := 75000.0 * (0.5 / 0.5)
	if res.Analysis.InputEnergy != 2e6 {
		t.Errorf("input_energy = %v, want 2e6", res.Analysis.InputEnergy)
	}
	if res.Analysis.OutputGross != 2e6 {
		t.Errorf("output_gross = %v, want 2e6", res.Analysis.OutputGross)
	}
	if res.Analysis.ProductionEnergy != wantProd {
		t.Errorf("production_energy = %v, want %v", res.Analysis.ProductionEnergy, wantProd)
	}
	if res.Analysis.Gain != -wantProd {
		t.Errorf("gain = %v, want %v", res.Analysis.Gain, -wantProd)
	}
}

func TestAnalyzeElectricity(t *testing.T) {
	srv := testServer(t)
	g := chain.Graph{
		Nodes: []chain.NodeDoc{
			{NodeID: "fuel", NodeKind: chain.KindInput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "water", NodeKind: chain.KindSource, ItemName: "water", PrototypeType: "fluid"},
			{NodeID: "b", NodeKind: chain.KindBoiler, Boiler: "boiler"},
			{NodeID: "eng", NodeKind: chain.KindGenerator, Generator: "steam-engine"},
			{NodeID: "out", NodeKind: chain.KindOutput, ItemName: chain.ElectricityName, PrototypeType: chain.ElectricityType},
		},
		Edges: []chain.Edge{
			{ID: "e1", FromNode: "fuel", FromPort: "out:0", ToNode: "b", ToPort: "in:0"},
			{ID: "e2", FromNode: "water", FromPort: "out:0", ToNode: "b", ToPort: "in:1"},
			{ID: "e3", FromNode: "b", FromPort: "out:0", ToNode: "eng", ToPort: "in:0"},
			{ID: "e4", FromNode: "eng", FromPort: "out:0", ToNode: "out", ToPort: "in:0"},
		},
	}
	res := postAnalyze(t, srv, g)
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	wantOut := 2e6
	if res.Analysis.InputEnergy != 2e6 {
		t.Errorf("input_energy = %v, want 2e6", res.Analysis.InputEnergy)
	}
	if math.Abs(res.Analysis.OutputGross-wantOut) > 1e-6*wantOut {
		t.Errorf("output_gross = %v, want %v", res.Analysis.OutputGross, wantOut)
	}
}

func TestAnalyzeInvalidGraph(t *testing.T) {
	srv := testServer(t)
	g := chain.Graph{
		Nodes: []chain.NodeDoc{{
			NodeID: "out", NodeKind: chain.KindOutput, ItemName: "wood", PrototypeType: "item",
		}},
	}
	res := postAnalyze(t, srv, g)
	if res.OK || res.Analysis != nil {
		t.Fatalf("invalid graph should skip analysis, got %+v", res)
	}
}

func postAnalyze(t *testing.T, srv *Server, g chain.Graph) chain.Result {
	t.Helper()
	body, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/graph/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("analyze status = %d, body = %s", rec.Code, rec.Body.Bytes())
	}
	var out chain.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	return out
}
