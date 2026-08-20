package catalog

import (
	"fmt"
	"sort"

	"github.com/Blustak/go-factorioHelper/internal/chain"
	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
	"github.com/Blustak/go-factorioHelper/internal/datatypes"
)

type Commodity struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Recipe struct {
	Name           string      `json:"name"`
	Category       string      `json:"category"`
	EnergyRequired *float64    `json:"energy_required,omitempty"`
	Ingredients    []Commodity `json:"ingredients"`
	Products       []Commodity `json:"products"`
}

type Machine struct {
	Name               string   `json:"name"`
	CraftingCategories []string `json:"crafting_categories"`
	CraftingSpeed      *float64 `json:"crafting_speed,omitempty"`
}

type Catalog struct {
	Recipes  []Recipe    `json:"recipes"`
	Items    []Commodity `json:"items"`
	Fluids   []Commodity `json:"fluids"`
	Machines []Machine   `json:"machines"`

	recipeByName  map[string]Recipe
	machineByName map[string]Machine
	commodities   map[string]struct{}
}

var _ chain.Catalog = (*Catalog)(nil)

func Load(cfg *config.State) (*Catalog, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	c := &Catalog{
		Recipes:       []Recipe{},
		Items:         []Commodity{},
		Fluids:        []Commodity{},
		Machines:      []Machine{},
		recipeByName:  map[string]Recipe{},
		machineByName: map[string]Machine{},
		commodities:   map[string]struct{}{},
	}
	if err := c.loadRecipes(cfg); err != nil {
		return nil, err
	}
	if err := c.loadMachines(cfg); err != nil {
		return nil, err
	}
	if err := c.loadItems(cfg); err != nil {
		return nil, err
	}
	if err := c.loadFluids(cfg); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Catalog) loadRecipes(cfg *config.State) error {
	rows, err := cfg.DB.GetAllRecipes(cfg.CTX)
	if err != nil {
		return fmt.Errorf("list recipes: %w", err)
	}
	for _, row := range rows {
		if !row.Name.Valid {
			return fmt.Errorf("recipe %d has no entity name", row.ID)
		}
		dbRow := database.Recipe{
			ID:             row.ID,
			EntityID:       row.EntityID,
			EnergyRequired: row.EnergyRequired,
			Category:       row.Category,
			MainProduct:    row.MainProduct,
			Ingredient:     row.Ingredient,
			Results:        row.Results,
		}
		rec, err := datatypes.RecipeFromDB(row.Name.String, row.PrototypeType.String, row.EntityOrder, dbRow)
		if err != nil {
			return err
		}
		dto := Recipe{
			Name:           rec.Name,
			Category:       "",
			EnergyRequired: rec.EnergyRequired,
			Ingredients:    ingredientsOf(rec.Ingredients),
			Products:       productsOf(rec.Results),
		}
		if rec.Category != nil {
			dto.Category = *rec.Category
		}
		c.Recipes = append(c.Recipes, dto)
		c.recipeByName[dto.Name] = dto
	}
	sort.Slice(c.Recipes, func(i, j int) bool { return c.Recipes[i].Name < c.Recipes[j].Name })
	return nil
}

func (c *Catalog) loadMachines(cfg *config.State) error {
	rows, err := cfg.DB.GetAllAssemblyMachines(cfg.CTX)
	if err != nil {
		return fmt.Errorf("list assembling machines: %w", err)
	}
	for _, row := range rows {
		if !row.Name.Valid {
			return fmt.Errorf("assembly machine %d has no entity name", row.ID)
		}
		dbRow := database.AssemblyMachine{
			ID:                 row.ID,
			EntityID:           row.EntityID,
			CraftingCategories: row.CraftingCategories,
			CraftingSpeed:      row.CraftingSpeed,
			EnergySource:       row.EnergySource,
			EnergyUsage:        row.EnergyUsage,
			FixedRecipe:        row.FixedRecipe,
		}
		m, err := datatypes.AssemblyMachineFromDB(row.Name.String, row.PrototypeType.String, row.EntityOrder, dbRow)
		if err != nil {
			return err
		}
		cats := m.CraftingCategories
		if cats == nil {
			cats = []string{}
		}
		dto := Machine{
			Name:               m.Name,
			CraftingCategories: cats,
			CraftingSpeed:      m.CraftingSpeed,
		}
		c.Machines = append(c.Machines, dto)
		c.machineByName[dto.Name] = dto
	}
	sort.Slice(c.Machines, func(i, j int) bool { return c.Machines[i].Name < c.Machines[j].Name })
	return nil
}

func (c *Catalog) loadItems(cfg *config.State) error {
	rows, err := cfg.DB.GetAllItems(cfg.CTX)
	if err != nil {
		return fmt.Errorf("list items: %w", err)
	}
	for _, row := range rows {
		if !row.Name.Valid {
			return fmt.Errorf("item %d has no entity name", row.ID)
		}
		proto := row.PrototypeType.String
		if proto == "" {
			proto = "item"
		}
		dto := Commodity{Name: row.Name.String, Type: proto}
		c.Items = append(c.Items, dto)
		c.commodities[commodityKey(dto.Name, dto.Type)] = struct{}{}
	}
	sort.Slice(c.Items, func(i, j int) bool { return c.Items[i].Name < c.Items[j].Name })
	return nil
}

func (c *Catalog) loadFluids(cfg *config.State) error {
	rows, err := cfg.DB.GetAllFluids(cfg.CTX)
	if err != nil {
		return fmt.Errorf("list fluids: %w", err)
	}
	for _, row := range rows {
		if !row.Name.Valid {
			return fmt.Errorf("fluid %d has no entity name", row.ID)
		}
		proto := row.PrototypeType.String
		if proto == "" {
			proto = "fluid"
		}
		dto := Commodity{Name: row.Name.String, Type: proto}
		c.Fluids = append(c.Fluids, dto)
		c.commodities[commodityKey(dto.Name, dto.Type)] = struct{}{}
	}
	sort.Slice(c.Fluids, func(i, j int) bool { return c.Fluids[i].Name < c.Fluids[j].Name })
	return nil
}

func (c *Catalog) Recipe(name string) (chain.RecipeInfo, bool) {
	if c == nil {
		return chain.RecipeInfo{}, false
	}
	r, ok := c.recipeByName[name]
	if !ok {
		return chain.RecipeInfo{}, false
	}
	return chain.RecipeInfo{
		Name:        r.Name,
		Category:    r.Category,
		Ingredients: toChain(r.Ingredients),
		Products:    toChain(r.Products),
	}, true
}

func (c *Catalog) Machine(name string) (chain.MachineInfo, bool) {
	if c == nil {
		return chain.MachineInfo{}, false
	}
	m, ok := c.machineByName[name]
	if !ok {
		return chain.MachineInfo{}, false
	}
	return chain.MachineInfo{Name: m.Name, Categories: m.CraftingCategories}, true
}

func (c *Catalog) HasCommodity(name, prototypeType string) bool {
	if c == nil {
		return false
	}
	if prototypeType == "" {
		prototypeType = "item"
	}
	_, ok := c.commodities[commodityKey(name, prototypeType)]
	return ok
}

func (c *Catalog) MachinesForCategory(category string) []Machine {
	if c == nil {
		return []Machine{}
	}
	if category == "" {
		out := make([]Machine, len(c.Machines))
		copy(out, c.Machines)
		return out
	}
	out := []Machine{}
	for _, m := range c.Machines {
		for _, cat := range m.CraftingCategories {
			if cat == category {
				out = append(out, m)
				break
			}
		}
	}
	return out
}

func ingredientsOf(ings []datatypes.Ingredient) []Commodity {
	out := make([]Commodity, 0, len(ings))
	for _, in := range ings {
		out = append(out, Commodity{Name: in.Name, Type: in.Type})
	}
	return out
}

func productsOf(products []datatypes.Product) []Commodity {
	out := make([]Commodity, 0, len(products))
	for _, p := range products {
		out = append(out, Commodity{Name: p.Name, Type: p.Type})
	}
	return out
}

func toChain(cs []Commodity) []chain.Commodity {
	out := make([]chain.Commodity, len(cs))
	for i, c := range cs {
		out[i] = chain.Commodity{Name: c.Name, Type: c.Type}
	}
	return out
}

func commodityKey(name, proto string) string {
	return proto + ":" + name
}
