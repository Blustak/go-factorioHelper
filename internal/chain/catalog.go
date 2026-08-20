package chain

// Catalog is the prototype lookup Validate needs. The editor catalog implements it.
type Catalog interface {
	Recipe(name string) (RecipeInfo, bool)
	Machine(name string) (MachineInfo, bool)
	Boiler(name string) (BoilerInfo, bool)
	Generator(name string) (GeneratorInfo, bool)
	HasCommodity(name, prototypeType string) bool
	FuelCategory(name, prototypeType string) (string, bool)
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

type BoilerInfo struct {
	Name           string
	InputFluid     string
	OutputFluid    string
	FuelCategories []string
}

type GeneratorInfo struct {
	Name       string
	InputFluid string
}
