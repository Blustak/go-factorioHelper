package chain

import (
	"strings"
	"testing"
)

var _ Node = NodeDoc{}

type fakeCatalog struct {
	recipes    map[string]RecipeInfo
	machines   map[string]MachineInfo
	boilers    map[string]BoilerInfo
	generators map[string]GeneratorInfo
	commodity  map[string]struct{}
	fuel       map[string]string
}

func (f fakeCatalog) Recipe(name string) (RecipeInfo, bool) {
	r, ok := f.recipes[name]
	return r, ok
}

func (f fakeCatalog) Machine(name string) (MachineInfo, bool) {
	m, ok := f.machines[name]
	return m, ok
}

func (f fakeCatalog) Boiler(name string) (BoilerInfo, bool) {
	b, ok := f.boilers[name]
	return b, ok
}

func (f fakeCatalog) Generator(name string) (GeneratorInfo, bool) {
	g, ok := f.generators[name]
	return g, ok
}

func (f fakeCatalog) HasCommodity(name, prototypeType string) bool {
	if prototypeType == "" {
		prototypeType = "item"
	}
	_, ok := f.commodity[prototypeType+":"+name]
	return ok
}

func (f fakeCatalog) FuelCategory(name, prototypeType string) (string, bool) {
	if prototypeType == "" {
		prototypeType = "item"
	}
	c, ok := f.fuel[prototypeType+":"+name]
	return c, ok
}

func testCatalog() fakeCatalog {
	return fakeCatalog{
		recipes: map[string]RecipeInfo{
			"iron-gear-wheel": {
				Name:     "iron-gear-wheel",
				Category: "crafting",
				Ingredients: []Commodity{
					{Name: "iron-plate", Type: "item"},
				},
				Products: []Commodity{
					{Name: "iron-gear-wheel", Type: "item"},
				},
			},
			"advanced-oil-processing": {
				Name:     "advanced-oil-processing",
				Category: "oil-processing",
				Ingredients: []Commodity{
					{Name: "crude-oil", Type: "fluid"},
				},
				Products: []Commodity{
					{Name: "heavy-oil", Type: "fluid"},
					{Name: "light-oil", Type: "fluid"},
					{Name: "petroleum-gas", Type: "fluid"},
				},
			},
		},
		machines: map[string]MachineInfo{
			"assembling-machine-1": {Name: "assembling-machine-1", Categories: []string{"crafting"}},
			"oil-refinery":         {Name: "oil-refinery", Categories: []string{"oil-processing"}},
		},
		boilers: map[string]BoilerInfo{
			"boiler": {
				Name:           "boiler",
				InputFluid:     "water",
				OutputFluid:    "steam",
				FuelCategories: []string{"chemical"},
			},
			"heat-exchanger": {
				Name:        "heat-exchanger",
				InputFluid:  "water",
				OutputFluid: "steam",
			},
		},
		generators: map[string]GeneratorInfo{
			"steam-engine": {Name: "steam-engine", InputFluid: "steam"},
		},
		commodity: map[string]struct{}{
			"item:iron-plate":      {},
			"item:iron-gear-wheel": {},
			"item:wood":            {},
			"item:coal":            {},
			"fluid:crude-oil":      {},
			"fluid:heavy-oil":      {},
			"fluid:light-oil":      {},
			"fluid:petroleum-gas":  {},
			"fluid:water":          {},
			"fluid:steam":          {},
		},
		fuel: map[string]string{
			"item:wood": "chemical",
			"item:coal": "chemical",
		},
	}
}

func TestValidateEmptyGraph(t *testing.T) {
	res := Validate(Graph{}, testCatalog())
	if !res.OK {
		t.Fatalf("empty graph issues = %+v", res.Issues)
	}
}

func TestValidateMissingInput(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{{
			NodeID:   "rec",
			NodeKind: KindRecipe,
			Recipe:   "iron-gear-wheel",
		}},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "required_input") == nil {
		t.Fatalf("issues = %+v, want required_input", res.Issues)
	}
}

func TestValidateWastedOutputOK(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "src", NodeKind: KindSource, ItemName: "iron-plate", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "iron-gear-wheel", Machine: "assembling-machine-1"},
		},
		Edges: []Edge{{
			ID: "e1", FromNode: "src", FromPort: "out:0", ToNode: "rec", ToPort: "in:0",
		}},
	}
	res := Validate(g, testCatalog())
	if !res.OK {
		t.Fatalf("wasted output should be valid, issues = %+v", res.Issues)
	}
}

func TestValidateTypeMismatch(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "src", NodeKind: KindSource, ItemName: "crude-oil", PrototypeType: "fluid"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "iron-gear-wheel"},
		},
		Edges: []Edge{{
			ID: "e1", FromNode: "src", FromPort: "out:0", ToNode: "rec", ToPort: "in:0",
		}},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "type_mismatch") == nil {
		t.Fatalf("issues = %+v, want type_mismatch", res.Issues)
	}
}

func TestValidateMachineCategory(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "src", NodeKind: KindSource, ItemName: "iron-plate", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "iron-gear-wheel", Machine: "oil-refinery"},
		},
		Edges: []Edge{{
			ID: "e1", FromNode: "src", FromPort: "out:0", ToNode: "rec", ToPort: "in:0",
		}},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "machine_category") == nil {
		t.Fatalf("issues = %+v, want machine_category", res.Issues)
	}

	g.Nodes[1].Machine = "assembling-machine-1"
	res = Validate(g, testCatalog())
	if !res.OK {
		t.Fatalf("matching machine should pass, issues = %+v", res.Issues)
	}
}

func TestValidateSourceSinkWiring(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "src", NodeKind: KindSource, ItemName: "crude-oil", PrototypeType: "fluid"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "advanced-oil-processing", Machine: "oil-refinery"},
			{NodeID: "snk", NodeKind: KindSink, ItemName: "petroleum-gas", PrototypeType: "fluid"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "src", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e2", FromNode: "rec", FromPort: "out:2", ToNode: "snk", ToPort: "in:0"},
		},
	}
	res := Validate(g, testCatalog())
	if !res.OK {
		t.Fatalf("oil graph with wasted by-products should pass, issues = %+v", res.Issues)
	}
}

func TestValidateUnknownCommodity(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{{
			NodeID: "src", NodeKind: KindSource, ItemName: "unobtainium", PrototypeType: "item",
		}},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "unknown_commodity") == nil {
		t.Fatalf("issues = %+v, want unknown_commodity", res.Issues)
	}
}

func TestValidateSinkRequiresInput(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{{
			NodeID: "snk", NodeKind: KindSink, ItemName: "iron-plate", PrototypeType: "item",
		}},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "required_input") == nil {
		t.Fatalf("issues = %+v, want required_input on sink", res.Issues)
	}
}

func TestValidateFanOut(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "src", NodeKind: KindSource, ItemName: "iron-plate", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "iron-gear-wheel"},
			{NodeID: "a", NodeKind: KindSink, ItemName: "iron-gear-wheel", PrototypeType: "item"},
			{NodeID: "b", NodeKind: KindSink, ItemName: "iron-gear-wheel", PrototypeType: "item"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "src", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e2", FromNode: "rec", FromPort: "out:0", ToNode: "a", ToPort: "in:0"},
			{ID: "e3", FromNode: "rec", FromPort: "out:0", ToNode: "b", ToPort: "in:0"},
		},
	}
	res := Validate(g, testCatalog())
	if !res.OK {
		t.Fatalf("fan-out should pass, issues = %+v", res.Issues)
	}
}

func TestValidateMultipleInputs(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "s1", NodeKind: KindSource, ItemName: "iron-plate", PrototypeType: "item"},
			{NodeID: "s2", NodeKind: KindSource, ItemName: "iron-plate", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "iron-gear-wheel"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "s1", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e2", FromNode: "s2", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
		},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "multiple_inputs") == nil {
		t.Fatalf("issues = %+v, want multiple_inputs", res.Issues)
	}
}

func hasCode(res Result, code string) *Issue {
	for i := range res.Issues {
		if res.Issues[i].Code == code {
			return &res.Issues[i]
		}
	}
	return nil
}

func TestValidateMissingCatalog(t *testing.T) {
	res := Validate(Graph{}, nil)
	if res.OK || hasCode(res, "missing_catalog") == nil {
		t.Fatalf("issues = %+v, want missing_catalog", res.Issues)
	}
}

func TestIssueMessagesMentionPorts(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{{NodeID: "rec", NodeKind: KindRecipe, Recipe: "iron-gear-wheel"}},
	}
	res := Validate(g, testCatalog())
	issue := hasCode(res, "required_input")
	if issue == nil || !strings.Contains(issue.Message, "iron-plate") {
		t.Fatalf("required_input message = %+v", issue)
	}
}

func TestValidateBoilerFuelAndWaterRequired(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{{NodeID: "b", NodeKind: KindBoiler, Boiler: "boiler"}},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "required_input") == nil {
		t.Fatalf("issues = %+v, want required_input", res.Issues)
	}
}

func TestValidateBoilerWoodFuelOK(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "fuel", NodeKind: KindSource, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "water", NodeKind: KindSource, ItemName: "water", PrototypeType: "fluid"},
			{NodeID: "b", NodeKind: KindBoiler, Boiler: "boiler"},
			{NodeID: "eng", NodeKind: KindGenerator, Generator: "steam-engine"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "fuel", FromPort: "out:0", ToNode: "b", ToPort: "in:0"},
			{ID: "e2", FromNode: "water", FromPort: "out:0", ToNode: "b", ToPort: "in:1"},
			{ID: "e3", FromNode: "b", FromPort: "out:0", ToNode: "eng", ToPort: "in:0"},
		},
	}
	res := Validate(g, testCatalog())
	if !res.OK {
		t.Fatalf("issues = %+v", res.Issues)
	}
}

func TestValidateBoilerIronPlateFuelMismatch(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "fuel", NodeKind: KindSource, ItemName: "iron-plate", PrototypeType: "item"},
			{NodeID: "water", NodeKind: KindSource, ItemName: "water", PrototypeType: "fluid"},
			{NodeID: "b", NodeKind: KindBoiler, Boiler: "boiler"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "fuel", FromPort: "out:0", ToNode: "b", ToPort: "in:0"},
			{ID: "e2", FromNode: "water", FromPort: "out:0", ToNode: "b", ToPort: "in:1"},
		},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "type_mismatch") == nil {
		t.Fatalf("issues = %+v, want type_mismatch", res.Issues)
	}
}

func TestValidateHeatExchangerNoFuelPort(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "water", NodeKind: KindSource, ItemName: "water", PrototypeType: "fluid"},
			{NodeID: "hx", NodeKind: KindBoiler, Boiler: "heat-exchanger"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "water", FromPort: "out:0", ToNode: "hx", ToPort: "in:0"},
		},
	}
	res := Validate(g, testCatalog())
	if !res.OK {
		t.Fatalf("issues = %+v", res.Issues)
	}
}

func TestValidateUnknownBoiler(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{{NodeID: "b", NodeKind: KindBoiler, Boiler: "missing"}},
	}
	res := Validate(g, testCatalog())
	if hasCode(res, "unknown_boiler") == nil {
		t.Fatalf("issues = %+v, want unknown_boiler", res.Issues)
	}
}
