package chain

import "fmt"

type NodeKind string

const (
	KindRecipe    NodeKind = "recipe"
	KindSource    NodeKind = "source"
	KindSink      NodeKind = "sink"
	KindInput     NodeKind = "input"
	KindOutput    NodeKind = "output"
	KindBoiler    NodeKind = "boiler"
	KindGenerator NodeKind = "generator"
)

const (
	ElectricityName = "electricity"
	ElectricityType = "electricity"
)

type Direction string

const (
	DirIn  Direction = "in"
	DirOut Direction = "out"
)

// Node is a process in the supply-chain graph. Ports are derived from the
// catalog (recipe ingredients/products, or the chosen source/sink item).
type Node interface {
	ID() string
	Kind() NodeKind
	Ports(cat Catalog) []Port
}

type Port struct {
	ID             string    `json:"id"`
	ItemName       string    `json:"item_name"`
	PrototypeType  string    `json:"prototype_type"`
	Direction      Direction `json:"direction"`
	Required       bool      `json:"required"`
	FuelCategories []string  `json:"fuel_categories,omitempty"`
}

type Edge struct {
	ID       string `json:"id"`
	FromNode string `json:"from_node"`
	FromPort string `json:"from_port"`
	ToNode   string `json:"to_node"`
	ToPort   string `json:"to_port"`
}

// Graph is the JSON document the editor sends. Node kinds share one struct
// with a discriminator; unused fields are omitted.
type Graph struct {
	Nodes []NodeDoc `json:"nodes"`
	Edges []Edge    `json:"edges"`
}

type NodeDoc struct {
	NodeID        string   `json:"id"`
	NodeKind      NodeKind `json:"kind"`
	X             float64  `json:"x"`
	Y             float64  `json:"y"`
	Recipe        string   `json:"recipe,omitempty"`
	Machine       string   `json:"machine,omitempty"`
	Boiler        string   `json:"boiler,omitempty"`
	Generator     string   `json:"generator,omitempty"`
	ItemName      string   `json:"item_name,omitempty"`
	PrototypeType string   `json:"prototype_type,omitempty"`
}

func (n NodeDoc) ID() string     { return n.NodeID }
func (n NodeDoc) Kind() NodeKind { return n.NodeKind }

func (n NodeDoc) Ports(cat Catalog) []Port {
	switch n.NodeKind {
	case KindRecipe:
		info, ok := cat.Recipe(n.Recipe)
		if !ok {
			return nil
		}
		return recipePorts(info)
	case KindBoiler:
		info, ok := cat.Boiler(n.Boiler)
		if !ok {
			return nil
		}
		return boilerPorts(info)
	case KindGenerator:
		info, ok := cat.Generator(n.Generator)
		if !ok {
			return nil
		}
		return generatorPorts(info)
	case KindSource, KindSink, KindInput, KindOutput:
		return ioPorts(n.NodeKind, n.ItemName, n.PrototypeType)
	default:
		return nil
	}
}

func PortID(dir Direction, i int) string {
	return fmt.Sprintf("%s:%d", dir, i)
}

func recipePorts(info RecipeInfo) []Port {
	ports := make([]Port, 0, len(info.Ingredients)+len(info.Products))
	for i, in := range info.Ingredients {
		ports = append(ports, Port{
			ID:            PortID(DirIn, i),
			ItemName:      in.Name,
			PrototypeType: normalizeType(in.Type),
			Direction:     DirIn,
			Required:      true,
		})
	}
	for i, p := range info.Products {
		ports = append(ports, Port{
			ID:            PortID(DirOut, i),
			ItemName:      p.Name,
			PrototypeType: normalizeType(p.Type),
			Direction:     DirOut,
			Required:      false,
		})
	}
	return ports
}

func boilerPorts(info BoilerInfo) []Port {
	ports := make([]Port, 0, 3)
	in := 0
	if len(info.FuelCategories) > 0 {
		ports = append(ports, Port{
			ID:             PortID(DirIn, in),
			ItemName:       "fuel",
			PrototypeType:  "item",
			Direction:      DirIn,
			Required:       true,
			FuelCategories: info.FuelCategories,
		})
		in++
	}
	if info.InputFluid != "" {
		ports = append(ports, Port{
			ID:            PortID(DirIn, in),
			ItemName:      info.InputFluid,
			PrototypeType: "fluid",
			Direction:     DirIn,
			Required:      true,
		})
	}
	if info.OutputFluid != "" {
		ports = append(ports, Port{
			ID:            PortID(DirOut, 0),
			ItemName:      info.OutputFluid,
			PrototypeType: "fluid",
			Direction:     DirOut,
			Required:      false,
		})
	}
	return ports
}

func generatorPorts(info GeneratorInfo) []Port {
	var ports []Port
	if info.InputFluid != "" {
		ports = append(ports, Port{
			ID:            PortID(DirIn, 0),
			ItemName:      info.InputFluid,
			PrototypeType: "fluid",
			Direction:     DirIn,
			Required:      true,
		})
	}
	ports = append(ports, Port{
		ID:            PortID(DirOut, 0),
		ItemName:      ElectricityName,
		PrototypeType: ElectricityType,
		Direction:     DirOut,
		Required:      false,
	})
	return ports
}

func ioPorts(kind NodeKind, name, protoType string) []Port {
	if !IsElectricity(name, protoType) {
		protoType = normalizeType(protoType)
	}
	if kind == KindSource || kind == KindInput {
		return []Port{{
			ID:            PortID(DirOut, 0),
			ItemName:      name,
			PrototypeType: protoType,
			Direction:     DirOut,
			Required:      false,
		}}
	}
	return []Port{{
		ID:            PortID(DirIn, 0),
		ItemName:      name,
		PrototypeType: protoType,
		Direction:     DirIn,
		Required:      true,
	}}
}

func IsElectricity(name, protoType string) bool {
	return name == ElectricityName && protoType == ElectricityType
}

func normalizeType(t string) string {
	if t == "" {
		return "item"
	}
	return t
}

func (g Graph) nodeByID() map[string]NodeDoc {
	out := make(map[string]NodeDoc, len(g.Nodes))
	for _, n := range g.Nodes {
		out[n.NodeID] = n
	}
	return out
}

func portByID(ports []Port) map[string]Port {
	out := make(map[string]Port, len(ports))
	for _, p := range ports {
		out[p.ID] = p
	}
	return out
}
