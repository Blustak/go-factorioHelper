package chain

// Catalog is the prototype lookup Validate needs. The editor catalog implements it.
type Catalog interface {
	Recipe(name string) (RecipeInfo, bool)
	Machine(name string) (MachineInfo, bool)
	HasCommodity(name, prototypeType string) bool
}

type Commodity struct {
	Name string
	Type string
}

type RecipeInfo struct {
	Name        string
	Category    string
	Ingredients []Commodity
	Products    []Commodity
}

type MachineInfo struct {
	Name       string
	Categories []string
}
