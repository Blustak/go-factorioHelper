package catalog

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/Blustak/go-factorioHelper/internal/chain"
	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
	"github.com/Blustak/go-factorioHelper/internal/datatypes"
)

type Commodity struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	LocalisedName string `json:"localised_name"`
}

type Recipe struct {
	Name           string      `json:"name"`
	LocalisedName  string      `json:"localised_name"`
	Category       string      `json:"category"`
	EnergyRequired *float64    `json:"energy_required,omitempty"`
	Ingredients    []Commodity `json:"ingredients"`
	Products       []Commodity `json:"products"`
}

type Machine struct {
	Name               string   `json:"name"`
	LocalisedName      string   `json:"localised_name"`
	CraftingCategories []string `json:"crafting_categories"`
	CraftingSpeed      *float64 `json:"crafting_speed,omitempty"`
}

type Producer struct {
	Name               string   `json:"name"`
	LocalisedName      string   `json:"localised_name"`
	Type               string   `json:"type"`
	ResourceCategories []string `json:"resource_categories"`
	MiningSpeed        *float64 `json:"mining_speed,omitempty"`
	PumpingSpeed       *float64 `json:"pumping_speed,omitempty"`
	ProducedFluid      *string  `json:"produced_fluid,omitempty"`
}

type Catalog struct {
	Recipes   []Recipe    `json:"recipes"`
	Items     []Commodity `json:"items"`
	Fluids    []Commodity `json:"fluids"`
	Machines  []Machine   `json:"machines"`
	Producers []Producer  `json:"producers"`

	recipeByName  map[string]Recipe
	machineByName map[string]Machine
	commodities   map[string]Commodity
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
		Producers:     []Producer{},
		recipeByName:  map[string]Recipe{},
		machineByName: map[string]Machine{},
		commodities:   map[string]Commodity{},
	}
	if err := c.loadItems(cfg); err != nil {
		return nil, err
	}
	if err := c.loadFluids(cfg); err != nil {
		return nil, err
	}
	if err := c.loadRecipes(cfg); err != nil {
		return nil, err
	}
	if err := c.loadMachines(cfg); err != nil {
		return nil, err
	}
	if err := c.loadProducers(cfg); err != nil {
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
			LocalisedName:  rowLocalised(rec.Name, row.LocalisedName),
			Category:       "",
			EnergyRequired: rec.EnergyRequired,
			Ingredients:    c.lookupIngredients(rec.Ingredients),
			Products:       c.lookupProducts(rec.Results),
		}
		if rec.Category != nil {
			dto.Category = *rec.Category
		}
		c.Recipes = append(c.Recipes, dto)
		c.recipeByName[dto.Name] = dto
	}
	sort.Slice(c.Recipes, func(i, j int) bool {
		return displayLess(c.Recipes[i].Name, c.Recipes[i].LocalisedName, c.Recipes[j].Name, c.Recipes[j].LocalisedName)
	})
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
			LocalisedName:      rowLocalised(m.Name, row.LocalisedName),
			CraftingCategories: cats,
			CraftingSpeed:      m.CraftingSpeed,
		}
		c.Machines = append(c.Machines, dto)
		c.machineByName[dto.Name] = dto
	}
	sort.Slice(c.Machines, func(i, j int) bool {
		return displayLess(c.Machines[i].Name, c.Machines[i].LocalisedName, c.Machines[j].Name, c.Machines[j].LocalisedName)
	})
	return nil
}

func (c *Catalog) loadProducers(cfg *config.State) error {
	rows, err := cfg.DB.GetAllResourceProducers(cfg.CTX)
	if err != nil {
		return fmt.Errorf("list resource producers: %w", err)
	}
	for _, row := range rows {
		if !row.Name.Valid {
			return fmt.Errorf("resource producer %d has no entity name", row.ID)
		}
		dbRow := database.ResourceProducer{
			ID:                 row.ID,
			EntityID:           row.EntityID,
			ResourceCategories: row.ResourceCategories,
			MiningSpeed:        row.MiningSpeed,
			PumpingSpeed:       row.PumpingSpeed,
			ProducedFluid:      row.ProducedFluid,
			EnergySource:       row.EnergySource,
			EnergyUsage:        row.EnergyUsage,
		}
		p, err := datatypes.ResourceProducerFromDB(row.Name.String, row.PrototypeType.String, row.EntityOrder, dbRow)
		if err != nil {
			return err
		}
		cats := p.ResourceCategories
		if cats == nil {
			cats = []string{}
		}
		dto := Producer{
			Name:               p.Name,
			LocalisedName:      rowLocalised(p.Name, row.LocalisedName),
			Type:               p.Type,
			ResourceCategories: cats,
			MiningSpeed:        p.MiningSpeed,
			PumpingSpeed:       p.PumpingSpeed,
		}
		if row.ProducedFluid.Valid {
			ent, err := cfg.DB.GetEntityByID(cfg.CTX, row.ProducedFluid.Int64)
			if err != nil {
				return fmt.Errorf("producer %q produced fluid: %w", p.Name, err)
			}
			name := ent.Name
			dto.ProducedFluid = &name
		}
		c.Producers = append(c.Producers, dto)
	}
	sort.Slice(c.Producers, func(i, j int) bool {
		return displayLess(c.Producers[i].Name, c.Producers[i].LocalisedName, c.Producers[j].Name, c.Producers[j].LocalisedName)
	})
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
		dto := Commodity{Name: row.Name.String, Type: proto, LocalisedName: rowLocalised(row.Name.String, row.LocalisedName)}
		c.Items = append(c.Items, dto)
		c.commodities[commodityKey(dto.Name, dto.Type)] = dto
	}
	sort.Slice(c.Items, func(i, j int) bool {
		return displayLess(c.Items[i].Name, c.Items[i].LocalisedName, c.Items[j].Name, c.Items[j].LocalisedName)
	})
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
		dto := Commodity{Name: row.Name.String, Type: proto, LocalisedName: rowLocalised(row.Name.String, row.LocalisedName)}
		c.Fluids = append(c.Fluids, dto)
		c.commodities[commodityKey(dto.Name, dto.Type)] = dto
	}
	sort.Slice(c.Fluids, func(i, j int) bool {
		return displayLess(c.Fluids[i].Name, c.Fluids[i].LocalisedName, c.Fluids[j].Name, c.Fluids[j].LocalisedName)
	})
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

func (c *Catalog) ProducersForCategory(category string) []Producer {
	if c == nil {
		return []Producer{}
	}
	if category == "" {
		out := make([]Producer, len(c.Producers))
		copy(out, c.Producers)
		return out
	}
	out := []Producer{}
	for _, p := range c.Producers {
		for _, cat := range p.ResourceCategories {
			if cat == category {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func (c *Catalog) lookupIngredients(ings []datatypes.Ingredient) []Commodity {
	out := make([]Commodity, 0, len(ings))
	for _, in := range ings {
		out = append(out, c.lookupCommodity(in.Name, in.Type))
	}
	return out
}

func (c *Catalog) lookupProducts(products []datatypes.Product) []Commodity {
	out := make([]Commodity, 0, len(products))
	for _, p := range products {
		out = append(out, c.lookupCommodity(p.Name, p.Type))
	}
	return out
}

func (c *Catalog) lookupCommodity(name, typ string) Commodity {
	if typ == "" {
		typ = "item"
	}
	if found, ok := c.commodities[commodityKey(name, typ)]; ok {
		return found
	}
	return Commodity{Name: name, Type: typ, LocalisedName: datatypes.Humanize(name)}
}

func rowLocalised(name string, ns sql.NullString) string {
	if ns.Valid && ns.String != "" {
		return ns.String
	}
	return datatypes.Humanize(name)
}

func displayLess(aName, aDisp, bName, bDisp string) bool {
	if aDisp != bDisp {
		return aDisp < bDisp
	}
	return aName < bName
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
