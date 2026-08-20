package chain

// Catalog is the prototype lookup Validate needs. The editor catalog implements it.
type Catalog interface {
	Recipe(name string) (RecipeInfo, bool)
	Machine(name string) (MachineInfo, bool)
	Boiler(name string) (BoilerInfo, bool)
	Generator(name string) (GeneratorInfo, bool)
	HasCommodity(name, prototypeType string) bool
	FuelCategory(name, prototypeType string) (string, bool)
	FuelValue(name, prototypeType string) (float64, bool)
}

type Commodity struct {
	Name   string
	Type   string
	Amount float64
}

type RecipeInfo struct {
	Name           string
	Category       string
	EnergyRequired *float64
	Ingredients    []Commodity
	Products       []Commodity
}

type MachineInfo struct {
	Name          string
	Categories    []string
	CraftingSpeed *float64
	EnergyUsage   *float64
}

type BoilerInfo struct {
	Name              string
	InputFluid        string
	OutputFluid       string
	FuelCategories    []string
	TargetTemperature *float64
	Effectivity       *float64
}

type GeneratorInfo struct {
	Name               string
	InputFluid         string
	Effectivity        *float64
	MaximumTemperature *float64
	BurnsFluid         bool
}
