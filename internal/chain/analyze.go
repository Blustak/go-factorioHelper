package chain

import "fmt"

const (
	// DefaultFluidHeatCapacity is vanilla water/steam heat capacity in J per unit per °C.
	DefaultFluidHeatCapacity = 200.0
	// DefaultFluidAmbientTemp is Factorio's 15°C zero-energy point for steam.
	DefaultFluidAmbientTemp = 15.0
	defaultRecipeEnergy     = 0.5
	defaultCraftingSpeed    = 1.0
	defaultEffectivity      = 1.0
)

type Analysis struct {
	InputEnergy      float64      `json:"input_energy"`
	OutputGross      float64      `json:"output_gross"`
	ProductionEnergy float64      `json:"production_energy"`
	OutputEnergy     float64      `json:"output_energy"`
	Gain             float64      `json:"gain"`
	Inputs           []EnergyTerm `json:"inputs"`
	Outputs          []EnergyTerm `json:"outputs"`
	Production       []EnergyTerm `json:"production"`
	Warnings         []string     `json:"warnings"`
}

type EnergyTerm struct {
	NodeID   string  `json:"node_id"`
	Item     string  `json:"item"`
	Quantity float64 `json:"quantity"`
	Energy   float64 `json:"energy"`
}

func Analyze(g Graph, cat Catalog) Result {
	res := Validate(g, cat)
	if !res.OK {
		return res
	}
	a := computeAnalysis(g, cat)
	res.Analysis = &a
	return res
}

func computeAnalysis(g Graph, cat Catalog) Analysis {
	a := Analysis{
		Inputs:     []EnergyTerm{},
		Outputs:    []EnergyTerm{},
		Production: []EnergyTerm{},
		Warnings:   []string{},
	}
	nodes := g.nodeByID()
	flow := solveFlows(g, cat, nodes)
	a.Warnings = append(a.Warnings, flow.warnings...)

	hasInput, hasOutput := false, false
	for _, n := range g.Nodes {
		switch n.NodeKind {
		case KindInput:
			hasInput = true
			qty := flow.qty[n.NodeID]
			term, warn := commodityEnergy(n, cat, "input", qty)
			a.Inputs = append(a.Inputs, term)
			a.InputEnergy += term.Energy
			if warn != "" {
				a.Warnings = append(a.Warnings, warn)
			}
		case KindOutput:
			hasOutput = true
			qty := flow.qty[n.NodeID]
			term, warn := outputEnergy(n, nodes, g, cat, qty)
			a.Outputs = append(a.Outputs, term)
			a.OutputGross += term.Energy
			if warn != "" {
				a.Warnings = append(a.Warnings, warn)
			}
		}
	}
	if !hasInput {
		a.Warnings = append(a.Warnings, "no input nodes; input energy is 0")
	}
	if !hasOutput {
		a.Warnings = append(a.Warnings, "no output nodes; output energy is 0")
	}

	for _, n := range g.Nodes {
		if n.NodeKind != KindRecipe {
			continue
		}
		crafts := flow.crafts[n.NodeID]
		if crafts <= 0 {
			continue
		}
		term, warn := recipeProductionEnergy(n, cat, crafts)
		if term != nil {
			a.Production = append(a.Production, *term)
			a.ProductionEnergy += term.Energy
		}
		if warn != "" {
			a.Warnings = append(a.Warnings, warn)
		}
	}

	a.Warnings = uniqStrings(a.Warnings)
	a.OutputEnergy = a.OutputGross - a.ProductionEnergy
	a.Gain = a.OutputEnergy - a.InputEnergy
	return a
}

func commodityEnergy(n NodeDoc, cat Catalog, role string, qty float64) (EnergyTerm, string) {
	term := EnergyTerm{NodeID: n.NodeID, Item: n.ItemName, Quantity: qty}
	if IsElectricity(n.ItemName, n.PrototypeType) {
		return term, fmt.Sprintf("%s node %q is electricity; fuel_value is not used", role, n.NodeID)
	}
	v, ok := cat.FuelValue(n.ItemName, n.PrototypeType)
	if !ok {
		return term, fmt.Sprintf("%s node %q (%s) has no fuel_value", role, n.NodeID, n.ItemName)
	}
	term.Energy = v * qty
	return term, ""
}

func outputEnergy(n NodeDoc, nodes map[string]NodeDoc, g Graph, cat Catalog, qty float64) (EnergyTerm, string) {
	if IsElectricity(n.ItemName, n.PrototypeType) {
		return electricityOutput(n, nodes, g, cat, qty)
	}
	return commodityEnergy(n, cat, "output", qty)
}

func electricityOutput(n NodeDoc, nodes map[string]NodeDoc, g Graph, cat Catalog, fluidQty float64) (EnergyTerm, string) {
	term := EnergyTerm{NodeID: n.NodeID, Item: ElectricityName, Quantity: fluidQty}
	from, ok := incomingNode(g, nodes, n.NodeID)
	if !ok || from.NodeKind != KindGenerator {
		return term, fmt.Sprintf("output node %q is not fed by a generator", n.NodeID)
	}
	per, warn := energyPerFluid(from, nodes, g, cat)
	term.Energy = per * fluidQty
	return term, warn
}

func energyPerFluid(gen NodeDoc, nodes map[string]NodeDoc, g Graph, cat Catalog) (float64, string) {
	info, ok := cat.Generator(gen.Generator)
	if !ok {
		return 0, fmt.Sprintf("generator node %q is unknown", gen.NodeID)
	}
	eff := floatOr(info.Effectivity, defaultEffectivity)
	if info.BurnsFluid {
		v, ok := cat.FuelValue(info.InputFluid, "fluid")
		if !ok {
			return 0, fmt.Sprintf("generator %q input fluid %q has no fuel_value", gen.Generator, info.InputFluid)
		}
		return v * eff, ""
	}
	temp := generatorFluidTemp(gen, nodes, g, cat, info)
	delta := temp - DefaultFluidAmbientTemp
	if delta < 0 {
		delta = 0
	}
	return DefaultFluidHeatCapacity * delta * eff, ""
}

func generatorFluidTemp(gen NodeDoc, nodes map[string]NodeDoc, g Graph, cat Catalog, info GeneratorInfo) float64 {
	maxT := floatOr(info.MaximumTemperature, DefaultFluidAmbientTemp)
	from, ok := incomingNode(g, nodes, gen.NodeID)
	if !ok || from.NodeKind != KindBoiler {
		return maxT
	}
	boiler, ok := cat.Boiler(from.Boiler)
	if !ok || boiler.TargetTemperature == nil {
		return maxT
	}
	t := *boiler.TargetTemperature
	if t > maxT {
		return maxT
	}
	return t
}

func recipeProductionEnergy(n NodeDoc, cat Catalog, crafts float64) (*EnergyTerm, string) {
	if n.Machine == "" {
		return nil, fmt.Sprintf("recipe node %q has no machine; craft energy skipped", n.NodeID)
	}
	recipe, ok := cat.Recipe(n.Recipe)
	if !ok {
		return nil, fmt.Sprintf("recipe node %q has unknown recipe", n.NodeID)
	}
	machine, ok := cat.Machine(n.Machine)
	if !ok {
		return nil, fmt.Sprintf("recipe node %q has unknown machine %q", n.NodeID, n.Machine)
	}
	if machine.EnergyUsage == nil {
		return nil, fmt.Sprintf("machine %q on node %q has no energy_usage; craft energy skipped", n.Machine, n.NodeID)
	}
	required := floatOr(recipe.EnergyRequired, defaultRecipeEnergy)
	speed := floatOr(machine.CraftingSpeed, defaultCraftingSpeed)
	if speed <= 0 {
		return nil, fmt.Sprintf("machine %q on node %q has no crafting speed; craft energy skipped", n.Machine, n.NodeID)
	}
	energy := *machine.EnergyUsage * (required / speed) * crafts
	return &EnergyTerm{NodeID: n.NodeID, Item: n.Recipe, Quantity: crafts, Energy: energy}, ""
}

type flowResult struct {
	qty      map[string]float64
	crafts   map[string]float64
	warnings []string
}

func solveFlows(g Graph, cat Catalog, nodes map[string]NodeDoc) flowResult {
	var commodityOut, elecOut int
	for _, n := range g.Nodes {
		if n.NodeKind != KindOutput {
			continue
		}
		if IsElectricity(n.ItemName, n.PrototypeType) {
			elecOut++
		} else {
			commodityOut++
		}
	}
	if commodityOut > 0 {
		return solveBackward(g, cat, nodes)
	}
	if elecOut > 0 {
		return solveForward(g, cat, nodes)
	}
	return flowResult{qty: map[string]float64{}, crafts: map[string]float64{}}
}

func solveBackward(g Graph, cat Catalog, nodes map[string]NodeDoc) flowResult {
	out := flowResult{qty: map[string]float64{}, crafts: map[string]float64{}}
	iters := len(g.Nodes)*2 + 8
	demandIn := map[string]float64{}
	for i := 0; i < iters; i++ {
		seed := outputCommodityDemand(g)
		demandOut := pullUpstream(g, demandIn)
		demandIn = seed
		for _, n := range g.Nodes {
			switch n.NodeKind {
			case KindRecipe:
				applyRecipeDemand(n, cat, demandOut, demandIn, out.crafts)
			case KindBoiler:
				applyBoilerDemand(n, g, nodes, cat, demandOut, demandIn, &out.warnings)
			case KindGenerator:
				applyGeneratorDemand(n, cat, demandOut, demandIn)
			}
		}
	}
	demandOut := pullUpstream(g, demandIn)
	for _, n := range g.Nodes {
		switch n.NodeKind {
		case KindInput, KindSource:
			out.qty[n.NodeID] = demandOut[portKey(n.NodeID, PortID(DirOut, 0))]
		case KindOutput:
			if !IsElectricity(n.ItemName, n.PrototypeType) {
				out.qty[n.NodeID] = 1
			} else {
				from, ok := incomingNode(g, nodes, n.NodeID)
				if ok {
					out.qty[n.NodeID] = demandIn[portKey(from.NodeID, PortID(DirIn, 0))]
				}
			}
		}
	}
	return out
}

func outputCommodityDemand(g Graph) map[string]float64 {
	demand := map[string]float64{}
	for _, n := range g.Nodes {
		if n.NodeKind == KindOutput && !IsElectricity(n.ItemName, n.PrototypeType) {
			demand[portKey(n.NodeID, PortID(DirIn, 0))] = 1
		}
	}
	return demand
}

func pullUpstream(g Graph, demandIn map[string]float64) map[string]float64 {
	demandOut := map[string]float64{}
	for _, e := range g.Edges {
		d := demandIn[portKey(e.ToNode, e.ToPort)]
		if d != 0 {
			demandOut[portKey(e.FromNode, e.FromPort)] += d
		}
	}
	return demandOut
}

func applyRecipeDemand(n NodeDoc, cat Catalog, demandOut, demandIn map[string]float64, crafts map[string]float64) {
	info, ok := cat.Recipe(n.Recipe)
	if !ok {
		return
	}
	c := 0.0
	for i, p := range info.Products {
		d := demandOut[portKey(n.NodeID, PortID(DirOut, i))]
		amt := commodityQty(p)
		if amt > 0 {
			c = max(c, d/amt)
		}
	}
	crafts[n.NodeID] = c
	for i, ing := range info.Ingredients {
		demandIn[portKey(n.NodeID, PortID(DirIn, i))] = c * commodityQty(ing)
	}
}

func applyBoilerDemand(n NodeDoc, g Graph, nodes map[string]NodeDoc, cat Catalog, demandOut, demandIn map[string]float64, warnings *[]string) {
	info, ok := cat.Boiler(n.Boiler)
	if !ok {
		return
	}
	steam := demandOut[portKey(n.NodeID, PortID(DirOut, 0))]
	for _, p := range n.Ports(cat) {
		if p.Direction != DirIn {
			continue
		}
		if len(p.FuelCategories) > 0 {
			demandIn[portKey(n.NodeID, p.ID)] = fuelItemsForSteam(n, g, nodes, cat, info, steam, warnings)
			continue
		}
		demandIn[portKey(n.NodeID, p.ID)] = steam
	}
}

func applyGeneratorDemand(n NodeDoc, cat Catalog, demandOut, demandIn map[string]float64) {
	info, ok := cat.Generator(n.Generator)
	if !ok || info.InputFluid == "" {
		return
	}
	// Electricity has no item quantity; fluid demand is set only when something
	// requires the electricity port (not used for 1-unit commodity outputs).
	elec := demandOut[portKey(n.NodeID, PortID(DirOut, 0))]
	if elec != 0 {
		demandIn[portKey(n.NodeID, PortID(DirIn, 0))] = elec
	}
}

func fuelItemsForSteam(n NodeDoc, g Graph, nodes map[string]NodeDoc, cat Catalog, info BoilerInfo, steam float64, warnings *[]string) float64 {
	if steam <= 0 {
		return 0
	}
	name, proto, ok := incomingCommodity(g, nodes, cat, n.NodeID, boilerFuelPort(n, cat))
	if !ok {
		return 0
	}
	fuel, ok := cat.FuelValue(name, proto)
	if !ok || fuel <= 0 {
		*warnings = append(*warnings, fmt.Sprintf("boiler node %q fuel %q has no fuel_value", n.NodeID, name))
		return 0
	}
	eff := floatOr(info.Effectivity, defaultEffectivity)
	dT := boilerDeltaT(info)
	if dT <= 0 || eff <= 0 {
		return 0
	}
	return steam * DefaultFluidHeatCapacity * dT / (eff * fuel)
}

func boilerDeltaT(info BoilerInfo) float64 {
	t := floatOr(info.TargetTemperature, DefaultFluidAmbientTemp)
	dT := t - DefaultFluidAmbientTemp
	if dT < 0 {
		return 0
	}
	return dT
}

func boilerFuelPort(n NodeDoc, cat Catalog) string {
	for _, p := range n.Ports(cat) {
		if p.Direction == DirIn && len(p.FuelCategories) > 0 {
			return p.ID
		}
	}
	return ""
}

func solveForward(g Graph, cat Catalog, nodes map[string]NodeDoc) flowResult {
	out := flowResult{qty: map[string]float64{}, crafts: map[string]float64{}}
	supplyOut := seedForward(g)
	iters := len(g.Nodes)*2 + 8
	for i := 0; i < iters; i++ {
		supplyIn := pushDownstream(g, supplyOut)
		next := seedForward(g)
		for _, n := range g.Nodes {
			switch n.NodeKind {
			case KindRecipe:
				applyRecipeSupply(n, cat, supplyIn, next, out.crafts)
			case KindBoiler:
				applyBoilerSupply(n, g, nodes, cat, supplyIn, next, &out.warnings)
			case KindGenerator:
				applyGeneratorSupply(n, cat, supplyIn, next)
			}
		}
		supplyOut = next
	}
	supplyIn := pushDownstream(g, supplyOut)
	for _, n := range g.Nodes {
		switch n.NodeKind {
		case KindInput, KindSource:
			out.qty[n.NodeID] = supplyOut[portKey(n.NodeID, PortID(DirOut, 0))]
		case KindOutput:
			out.qty[n.NodeID] = supplyIn[portKey(n.NodeID, PortID(DirIn, 0))]
		}
	}
	return out
}

func seedForward(g Graph) map[string]float64 {
	supply := map[string]float64{}
	hasInput := false
	for _, n := range g.Nodes {
		if n.NodeKind == KindInput {
			hasInput = true
			supply[portKey(n.NodeID, PortID(DirOut, 0))] = 1
		}
	}
	if hasInput {
		return supply
	}
	for _, n := range g.Nodes {
		if n.NodeKind == KindSource {
			supply[portKey(n.NodeID, PortID(DirOut, 0))] = 1
		}
	}
	return supply
}

func pushDownstream(g Graph, supplyOut map[string]float64) map[string]float64 {
	fanout := map[string]int{}
	for _, e := range g.Edges {
		fanout[portKey(e.FromNode, e.FromPort)]++
	}
	supplyIn := map[string]float64{}
	for _, e := range g.Edges {
		key := portKey(e.FromNode, e.FromPort)
		n := fanout[key]
		if n == 0 {
			continue
		}
		supplyIn[portKey(e.ToNode, e.ToPort)] += supplyOut[key] / float64(n)
	}
	return supplyIn
}

func applyRecipeSupply(n NodeDoc, cat Catalog, supplyIn, supplyOut map[string]float64, crafts map[string]float64) {
	info, ok := cat.Recipe(n.Recipe)
	if !ok {
		return
	}
	c := 0.0
	first := true
	for i, ing := range info.Ingredients {
		amt := commodityQty(ing)
		if amt <= 0 {
			c = 0
			first = false
			break
		}
		got := supplyIn[portKey(n.NodeID, PortID(DirIn, i))] / amt
		if first || got < c {
			c = got
			first = false
		}
	}
	if first {
		c = 0
	}
	if c < 0 {
		c = 0
	}
	crafts[n.NodeID] = c
	for i, p := range info.Products {
		supplyOut[portKey(n.NodeID, PortID(DirOut, i))] = c * commodityQty(p)
	}
}

func applyBoilerSupply(n NodeDoc, g Graph, nodes map[string]NodeDoc, cat Catalog, supplyIn, supplyOut map[string]float64, warnings *[]string) {
	info, ok := cat.Boiler(n.Boiler)
	if !ok {
		return
	}
	var steam float64
	unlimitedWater := false
	waterQty := 0.0
	for _, p := range n.Ports(cat) {
		if p.Direction != DirIn {
			continue
		}
		got := supplyIn[portKey(n.NodeID, p.ID)]
		if len(p.FuelCategories) > 0 {
			name, proto, ok := incomingCommodity(g, nodes, cat, n.NodeID, p.ID)
			if !ok {
				continue
			}
			fuel, ok := cat.FuelValue(name, proto)
			if !ok || fuel <= 0 {
				*warnings = append(*warnings, fmt.Sprintf("boiler node %q fuel %q has no fuel_value", n.NodeID, name))
				continue
			}
			eff := floatOr(info.Effectivity, defaultEffectivity)
			dT := boilerDeltaT(info)
			if dT > 0 && eff > 0 {
				steam = got * fuel * eff / (DefaultFluidHeatCapacity * dT)
			}
			continue
		}
		from, ok := incomingNodeToPort(g, nodes, n.NodeID, p.ID)
		if ok && from.NodeKind == KindSource {
			unlimitedWater = true
		}
		waterQty = got
	}
	if info.FuelCategories == nil || len(info.FuelCategories) == 0 {
		steam = waterQty
		if unlimitedWater && waterQty == 0 {
			steam = 1
		}
	} else if !unlimitedWater && waterQty < steam {
		steam = waterQty
	}
	if info.OutputFluid != "" {
		supplyOut[portKey(n.NodeID, PortID(DirOut, 0))] = steam
	}
}

func applyGeneratorSupply(n NodeDoc, cat Catalog, supplyIn, supplyOut map[string]float64) {
	info, ok := cat.Generator(n.Generator)
	if !ok || info.InputFluid == "" {
		return
	}
	fluid := supplyIn[portKey(n.NodeID, PortID(DirIn, 0))]
	supplyOut[portKey(n.NodeID, PortID(DirOut, 0))] = fluid
}

func incomingNode(g Graph, nodes map[string]NodeDoc, nodeID string) (NodeDoc, bool) {
	for _, e := range g.Edges {
		if e.ToNode == nodeID {
			n, ok := nodes[e.FromNode]
			return n, ok
		}
	}
	return NodeDoc{}, false
}

func incomingNodeToPort(g Graph, nodes map[string]NodeDoc, nodeID, portID string) (NodeDoc, bool) {
	for _, e := range g.Edges {
		if e.ToNode == nodeID && e.ToPort == portID {
			n, ok := nodes[e.FromNode]
			return n, ok
		}
	}
	return NodeDoc{}, false
}

func incomingCommodity(g Graph, nodes map[string]NodeDoc, cat Catalog, nodeID, portID string) (string, string, bool) {
	for _, e := range g.Edges {
		if e.ToNode != nodeID || e.ToPort != portID {
			continue
		}
		from, ok := nodes[e.FromNode]
		if !ok {
			return "", "", false
		}
		for _, p := range from.Ports(cat) {
			if p.ID == e.FromPort {
				return p.ItemName, p.PrototypeType, true
			}
		}
	}
	return "", "", false
}

func commodityQty(c Commodity) float64 {
	if c.Amount > 0 {
		return c.Amount
	}
	return 1
}

func floatOr(v *float64, fallback float64) float64 {
	if v == nil {
		return fallback
	}
	return *v
}

func uniqStrings(ss []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range ss {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
