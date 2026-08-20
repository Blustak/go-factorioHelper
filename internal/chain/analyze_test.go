package chain

import (
	"math"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func energyCatalog() fakeCatalog {
	c := testCatalog()
	c.fuelValue = map[string]float64{
		"item:wood": 2e6,
		"item:coal": 4e6,
	}
	gear := c.recipes["iron-gear-wheel"]
	gear.EnergyRequired = ptr(0.5)
	c.recipes["iron-gear-wheel"] = gear
	c.recipes["wood"] = RecipeInfo{
		Name:           "wood",
		Category:       "crafting",
		EnergyRequired: ptr(0.5),
		Ingredients:    []Commodity{{Name: "wood", Type: "item", Amount: 1}},
		Products:       []Commodity{{Name: "wood", Type: "item", Amount: 1}},
	}
	c.recipes["compress-wood"] = RecipeInfo{
		Name:           "compress-wood",
		Category:       "crafting",
		EnergyRequired: ptr(0.5),
		Ingredients:    []Commodity{{Name: "wood", Type: "item", Amount: 2}},
		Products:       []Commodity{{Name: "wood", Type: "item", Amount: 1}},
	}
	am := c.machines["assembling-machine-1"]
	am.CraftingSpeed = ptr(0.5)
	am.EnergyUsage = ptr(75000.0)
	c.machines["assembling-machine-1"] = am
	boiler := c.boilers["boiler"]
	boiler.TargetTemperature = ptr(165.0)
	boiler.Effectivity = ptr(1.0)
	c.boilers["boiler"] = boiler
	eng := c.generators["steam-engine"]
	eng.Effectivity = ptr(1.0)
	eng.MaximumTemperature = ptr(165.0)
	c.generators["steam-engine"] = eng
	return c
}

func TestAnalyzeInvalidGraph(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{{NodeID: "out", NodeKind: KindOutput, ItemName: "wood", PrototypeType: "item"}},
	}
	res := Analyze(g, energyCatalog())
	if res.OK || res.Analysis != nil {
		t.Fatalf("invalid graph should not analyze, got %+v", res)
	}
	if hasCode(res, "required_input") == nil {
		t.Fatalf("issues = %+v, want required_input", res.Issues)
	}
}

func TestAnalyzeWoodInWoodOut(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "in", NodeKind: KindInput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "out", NodeKind: KindOutput, ItemName: "wood", PrototypeType: "item"},
		},
		Edges: []Edge{{
			ID: "e1", FromNode: "in", FromPort: "out:0", ToNode: "out", ToPort: "in:0",
		}},
	}
	res := Analyze(g, energyCatalog())
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	a := res.Analysis
	if a.InputEnergy != 2e6 || a.OutputGross != 2e6 || a.ProductionEnergy != 0 || a.Gain != 0 {
		t.Fatalf("analysis = %+v, want 2MJ in/out gain 0", a)
	}
	if len(a.Inputs) != 1 || a.Inputs[0].Quantity != 1 || len(a.Outputs) != 1 || a.Outputs[0].Quantity != 1 {
		t.Fatalf("quantities = in %+v out %+v, want 1", a.Inputs, a.Outputs)
	}
}

func TestAnalyzeIgnoresSourceFuel(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "src", NodeKind: KindSource, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "out", NodeKind: KindOutput, ItemName: "wood", PrototypeType: "item"},
		},
		Edges: []Edge{{
			ID: "e1", FromNode: "src", FromPort: "out:0", ToNode: "out", ToPort: "in:0",
		}},
	}
	res := Analyze(g, energyCatalog())
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	a := res.Analysis
	if a.InputEnergy != 0 {
		t.Fatalf("source should not count as input, got %v", a.InputEnergy)
	}
	if a.OutputGross != 2e6 {
		t.Fatalf("output energy = %v, want 2e6", a.OutputGross)
	}
}

func TestAnalyzeRecipeProductionEnergy(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "in", NodeKind: KindInput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "wood", Machine: "assembling-machine-1"},
			{NodeID: "out", NodeKind: KindOutput, ItemName: "wood", PrototypeType: "item"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "in", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e2", FromNode: "rec", FromPort: "out:0", ToNode: "out", ToPort: "in:0"},
		},
	}
	res := Analyze(g, energyCatalog())
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	a := res.Analysis
	wantProd := 75000.0 * (0.5 / 0.5)
	if a.ProductionEnergy != wantProd {
		t.Fatalf("production = %v, want %v", a.ProductionEnergy, wantProd)
	}
	if a.InputEnergy != 2e6 || a.OutputGross != 2e6 {
		t.Fatalf("io energy in=%v out=%v, want 2e6", a.InputEnergy, a.OutputGross)
	}
	if a.OutputEnergy != 2e6-wantProd || a.Gain != -wantProd {
		t.Fatalf("output=%v gain=%v, want net 2e6-75kJ", a.OutputEnergy, a.Gain)
	}
}

func TestAnalyzeRecipeOffPathSkipped(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "in", NodeKind: KindInput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "out", NodeKind: KindOutput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "src", NodeKind: KindSource, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "wood", Machine: "assembling-machine-1"},
			{NodeID: "snk", NodeKind: KindSink, ItemName: "wood", PrototypeType: "item"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "in", FromPort: "out:0", ToNode: "out", ToPort: "in:0"},
			{ID: "e2", FromNode: "src", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e3", FromNode: "rec", FromPort: "out:0", ToNode: "snk", ToPort: "in:0"},
		},
	}
	res := Analyze(g, energyCatalog())
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	if res.Analysis.ProductionEnergy != 0 {
		t.Fatalf("off-path recipe should not count, production = %v", res.Analysis.ProductionEnergy)
	}
}

func TestAnalyzeMissingFuelValueWarns(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "in", NodeKind: KindInput, ItemName: "iron-plate", PrototypeType: "item"},
			{NodeID: "out", NodeKind: KindOutput, ItemName: "iron-plate", PrototypeType: "item"},
		},
		Edges: []Edge{{
			ID: "e1", FromNode: "in", FromPort: "out:0", ToNode: "out", ToPort: "in:0",
		}},
	}
	res := Analyze(g, energyCatalog())
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	if res.Analysis.InputEnergy != 0 || res.Analysis.OutputGross != 0 {
		t.Fatalf("missing fuel_value should be 0, got %+v", res.Analysis)
	}
	if len(res.Analysis.Warnings) < 2 {
		t.Fatalf("warnings = %v, want missing fuel_value on in and out", res.Analysis.Warnings)
	}
}

func TestAnalyzeElectricityFromBoilerSteam(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "fuel", NodeKind: KindInput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "water", NodeKind: KindSource, ItemName: "water", PrototypeType: "fluid"},
			{NodeID: "b", NodeKind: KindBoiler, Boiler: "boiler"},
			{NodeID: "eng", NodeKind: KindGenerator, Generator: "steam-engine"},
			{NodeID: "out", NodeKind: KindOutput, ItemName: ElectricityName, PrototypeType: ElectricityType},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "fuel", FromPort: "out:0", ToNode: "b", ToPort: "in:0"},
			{ID: "e2", FromNode: "water", FromPort: "out:0", ToNode: "b", ToPort: "in:1"},
			{ID: "e3", FromNode: "b", FromPort: "out:0", ToNode: "eng", ToPort: "in:0"},
			{ID: "e4", FromNode: "eng", FromPort: "out:0", ToNode: "out", ToPort: "in:0"},
		},
	}
	res := Analyze(g, energyCatalog())
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	a := res.Analysis
	wantOut := 2e6
	if a.InputEnergy != 2e6 {
		t.Fatalf("input = %v, want 2e6", a.InputEnergy)
	}
	if !almostEq(a.OutputGross, wantOut) {
		t.Fatalf("electricity = %v, want %v", a.OutputGross, wantOut)
	}
	if a.ProductionEnergy != 0 {
		t.Fatalf("production = %v, want 0", a.ProductionEnergy)
	}
	if !almostEq(a.Gain, 0) {
		t.Fatalf("gain = %v, want 0", a.Gain)
	}
}

func TestAnalyzeRecipeWithoutMachineWarns(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "in", NodeKind: KindInput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "wood"},
			{NodeID: "out", NodeKind: KindOutput, ItemName: "wood", PrototypeType: "item"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "in", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e2", FromNode: "rec", FromPort: "out:0", ToNode: "out", ToPort: "in:0"},
		},
	}
	res := Analyze(g, energyCatalog())
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	if res.Analysis.ProductionEnergy != 0 {
		t.Fatalf("production = %v, want 0", res.Analysis.ProductionEnergy)
	}
	if len(res.Analysis.Warnings) == 0 {
		t.Fatal("want warning for missing machine")
	}
}

func TestAnalyzeRecipeIngredientAmounts(t *testing.T) {
	g := Graph{
		Nodes: []NodeDoc{
			{NodeID: "in", NodeKind: KindInput, ItemName: "wood", PrototypeType: "item"},
			{NodeID: "rec", NodeKind: KindRecipe, Recipe: "compress-wood", Machine: "assembling-machine-1"},
			{NodeID: "out", NodeKind: KindOutput, ItemName: "wood", PrototypeType: "item"},
		},
		Edges: []Edge{
			{ID: "e1", FromNode: "in", FromPort: "out:0", ToNode: "rec", ToPort: "in:0"},
			{ID: "e2", FromNode: "rec", FromPort: "out:0", ToNode: "out", ToPort: "in:0"},
		},
	}
	res := Analyze(g, energyCatalog())
	if !res.OK || res.Analysis == nil {
		t.Fatalf("analyze = %+v", res)
	}
	a := res.Analysis
	if !almostEq(a.Inputs[0].Quantity, 2) || a.InputEnergy != 4e6 {
		t.Fatalf("input qty/energy = %+v / %v, want 2 × 2MJ", a.Inputs, a.InputEnergy)
	}
	if a.Outputs[0].Quantity != 1 || a.OutputGross != 2e6 {
		t.Fatalf("output qty/energy = %+v / %v, want 1 × 2MJ", a.Outputs, a.OutputGross)
	}
	wantProd := 75000.0
	if a.ProductionEnergy != wantProd {
		t.Fatalf("production = %v, want 1 craft (%v)", a.ProductionEnergy, wantProd)
	}
}

func almostEq(a, b float64) bool {
	return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(b))
}
