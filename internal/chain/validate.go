package chain

import "fmt"

type Issue struct {
	NodeID  string `json:"node_id,omitempty"`
	PortID  string `json:"port_id,omitempty"`
	EdgeID  string `json:"edge_id,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	OK     bool    `json:"ok"`
	Issues []Issue `json:"issues"`
}

func (r Result) okIfClean() Result {
	r.OK = len(r.Issues) == 0
	if r.Issues == nil {
		r.Issues = []Issue{}
	}
	return r
}

func Validate(g Graph, cat Catalog) Result {
	var issues []Issue
	if cat == nil {
		return Result{OK: false, Issues: []Issue{{
			Code:    "missing_catalog",
			Message: "catalog is required",
		}}}
	}

	seenNodes := make(map[string]int, len(g.Nodes))
	for _, n := range g.Nodes {
		if n.NodeID == "" {
			issues = append(issues, Issue{
				Code:    "missing_node_id",
				Message: "node is missing an id",
			})
			continue
		}
		seenNodes[n.NodeID]++
		issues = append(issues, validateNode(n, cat)...)
	}
	for id, n := range seenNodes {
		if n > 1 {
			issues = append(issues, Issue{
				NodeID:  id,
				Code:    "duplicate_node_id",
				Message: fmt.Sprintf("duplicate node id %q", id),
			})
		}
	}

	nodes := g.nodeByID()
	portIndex := make(map[string]map[string]Port, len(g.Nodes))
	for _, n := range g.Nodes {
		if n.NodeID == "" {
			continue
		}
		portIndex[n.NodeID] = portByID(n.Ports(cat))
	}

	incoming := make(map[string]int)
	seenEdges := make(map[string]int, len(g.Edges))
	seenPair := make(map[string]struct{}, len(g.Edges))
	for _, e := range g.Edges {
		if e.ID == "" {
			issues = append(issues, Issue{
				Code:    "missing_edge_id",
				Message: "edge is missing an id",
			})
		} else {
			seenEdges[e.ID]++
		}
		issues = append(issues, validateEdge(e, nodes, portIndex, seenPair, incoming, cat)...)
	}
	for id, n := range seenEdges {
		if n > 1 {
			issues = append(issues, Issue{
				EdgeID:  id,
				Code:    "duplicate_edge_id",
				Message: fmt.Sprintf("duplicate edge id %q", id),
			})
		}
	}

	for _, n := range g.Nodes {
		if n.NodeID == "" {
			continue
		}
		for _, p := range n.Ports(cat) {
			if !p.Required {
				continue
			}
			count := incoming[portKey(n.NodeID, p.ID)]
			switch {
			case count == 0:
				issues = append(issues, Issue{
					NodeID:  n.NodeID,
					PortID:  p.ID,
					Code:    "required_input",
					Message: fmt.Sprintf("input %s (%s) must be connected", p.ID, p.ItemName),
				})
			case count > 1:
				issues = append(issues, Issue{
					NodeID:  n.NodeID,
					PortID:  p.ID,
					Code:    "multiple_inputs",
					Message: fmt.Sprintf("input %s (%s) has %d incoming edges; want 1", p.ID, p.ItemName, count),
				})
			}
		}
	}

	return Result{OK: len(issues) == 0, Issues: issues}.okIfClean()
}

func validateNode(n NodeDoc, cat Catalog) []Issue {
	switch n.NodeKind {
	case KindRecipe:
		return validateRecipeNode(n, cat)
	case KindBoiler:
		return validateNamedNode(n.NodeID, n.Boiler, "boiler", func(name string) bool {
			_, ok := cat.Boiler(name)
			return ok
		})
	case KindGenerator:
		return validateNamedNode(n.NodeID, n.Generator, "generator", func(name string) bool {
			_, ok := cat.Generator(name)
			return ok
		})
	case KindSource, KindSink:
		return validateIONode(n, cat)
	default:
		return []Issue{{
			NodeID:  n.NodeID,
			Code:    "unknown_kind",
			Message: fmt.Sprintf("unknown node kind %q", n.NodeKind),
		}}
	}
}

func validateNamedNode(nodeID, name, kind string, exists func(string) bool) []Issue {
	code := "unknown_" + kind
	if name == "" {
		return []Issue{{
			NodeID:  nodeID,
			Code:    code,
			Message: kind + " node has no " + kind,
		}}
	}
	if !exists(name) {
		return []Issue{{
			NodeID:  nodeID,
			Code:    code,
			Message: fmt.Sprintf("unknown %s %q", kind, name),
		}}
	}
	return nil
}

func validateRecipeNode(n NodeDoc, cat Catalog) []Issue {
	if n.Recipe == "" {
		return []Issue{{
			NodeID:  n.NodeID,
			Code:    "unknown_recipe",
			Message: "recipe node has no recipe",
		}}
	}
	info, ok := cat.Recipe(n.Recipe)
	if !ok {
		return []Issue{{
			NodeID:  n.NodeID,
			Code:    "unknown_recipe",
			Message: fmt.Sprintf("unknown recipe %q", n.Recipe),
		}}
	}
	if n.Machine == "" {
		return nil
	}
	machine, ok := cat.Machine(n.Machine)
	if !ok {
		return []Issue{{
			NodeID:  n.NodeID,
			Code:    "unknown_machine",
			Message: fmt.Sprintf("unknown machine %q", n.Machine),
		}}
	}
	if !machineSupports(machine, info.Category) {
		return []Issue{{
			NodeID:  n.NodeID,
			Code:    "machine_category",
			Message: fmt.Sprintf("machine %q cannot craft category %q", n.Machine, info.Category),
		}}
	}
	return nil
}

func validateIONode(n NodeDoc, cat Catalog) []Issue {
	if n.ItemName == "" {
		return []Issue{{
			NodeID:  n.NodeID,
			Code:    "missing_item",
			Message: fmt.Sprintf("%s node has no item", n.NodeKind),
		}}
	}
	proto := normalizeType(n.PrototypeType)
	if !cat.HasCommodity(n.ItemName, proto) {
		return []Issue{{
			NodeID:  n.NodeID,
			Code:    "unknown_commodity",
			Message: fmt.Sprintf("unknown %s %q", proto, n.ItemName),
		}}
	}
	return nil
}

func validateEdge(e Edge, nodes map[string]NodeDoc, portIndex map[string]map[string]Port, seenPair map[string]struct{}, incoming map[string]int, cat Catalog) []Issue {
	var issues []Issue
	if e.FromNode == "" || e.ToNode == "" || e.FromPort == "" || e.ToPort == "" {
		issues = append(issues, Issue{
			EdgeID:  e.ID,
			Code:    "dangling_edge",
			Message: "edge is missing an endpoint",
		})
		return issues
	}
	_, fromOK := nodes[e.FromNode]
	_, toOK := nodes[e.ToNode]
	if !fromOK || !toOK {
		issues = append(issues, Issue{
			EdgeID:  e.ID,
			Code:    "unknown_edge_endpoint",
			Message: "edge refers to a missing node",
		})
		return issues
	}

	fromPort, fromPortOK := portIndex[e.FromNode][e.FromPort]
	toPort, toPortOK := portIndex[e.ToNode][e.ToPort]
	if !fromPortOK || !toPortOK {
		issues = append(issues, Issue{
			EdgeID:  e.ID,
			Code:    "unknown_port",
			Message: "edge refers to a missing port",
		})
		return issues
	}
	if fromPort.Direction != DirOut || toPort.Direction != DirIn {
		issues = append(issues, Issue{
			EdgeID:  e.ID,
			Code:    "direction",
			Message: "edges must run from an output port to an input port",
		})
		return issues
	}
	if !portsCompatible(fromPort, toPort, cat) {
		issues = append(issues, Issue{
			EdgeID: e.ID,
			Code:   "type_mismatch",
			Message: fmt.Sprintf("cannot connect %s %s to %s %s",
				fromPort.PrototypeType, fromPort.ItemName, toPort.PrototypeType, toPort.ItemName),
		})
	}
	pair := e.FromNode + "/" + e.FromPort + "->" + e.ToNode + "/" + e.ToPort
	if _, dup := seenPair[pair]; dup {
		issues = append(issues, Issue{
			EdgeID:  e.ID,
			Code:    "duplicate_edge",
			Message: "duplicate edge between the same ports",
		})
	} else {
		seenPair[pair] = struct{}{}
	}
	incoming[portKey(e.ToNode, e.ToPort)]++
	return issues
}

func portsCompatible(from, to Port, cat Catalog) bool {
	if len(to.FuelCategories) > 0 {
		fuel, ok := cat.FuelCategory(from.ItemName, from.PrototypeType)
		if !ok {
			return false
		}
		return containsString(to.FuelCategories, fuel)
	}
	return from.ItemName == to.ItemName && from.PrototypeType == to.PrototypeType
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func machineSupports(machine MachineInfo, category string) bool {
	for _, c := range machine.Categories {
		if c == category {
			return true
		}
	}
	return false
}

func portKey(nodeID, portID string) string {
	return nodeID + "/" + portID
}
