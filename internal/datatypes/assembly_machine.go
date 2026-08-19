package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type AssemblyMachine struct {
	Entity
	CraftingCategories []string
	CraftingSpeed      *float64
	EnergySource       *EnergySource
	EnergyUsage        *float64
	FixedRecipe        *string
	machineEntry       *database.AssemblyMachine
}

func (m *AssemblyMachine) UnmarshalJSON(b []byte) error {
	var raw struct {
		Name               string        `json:"name"`
		Type               string        `json:"type"`
		Order              *string       `json:"order"`
		CraftingCategories []string      `json:"crafting_categories"`
		CraftingSpeed      *float64      `json:"crafting_speed"`
		EnergySource       *EnergySource `json:"energy_source"`
		EnergyUsage        *string       `json:"energy_usage"`
		FixedRecipe        *string       `json:"fixed_recipe"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	usage, err := parseOptionalPower(raw.EnergyUsage)
	if err != nil {
		return fmt.Errorf("assembling machine %q energy_usage: %w", raw.Name, err)
	}
	*m = AssemblyMachine{
		Entity: Entity{
			Name:        raw.Name,
			Type:        raw.Type,
			EntityOrder: raw.Order,
		},
		CraftingCategories: raw.CraftingCategories,
		CraftingSpeed:      raw.CraftingSpeed,
		EnergySource:       raw.EnergySource,
		EnergyUsage:        usage,
		FixedRecipe:        raw.FixedRecipe,
	}
	return nil
}

func (m *AssemblyMachine) AddToDB(cfg *config.State) (database.AssemblyMachine, error) {
	var machine database.AssemblyMachine
	if m == nil {
		return machine, DatatypeNilError[AssemblyMachine]("Nil assembling machine")
	}
	if _, err := m.Entity.AddToDB(cfg); err != nil {
		return machine, err
	}

	cats, err := gobEncodeSlice(m.CraftingCategories)
	if err != nil {
		return machine, err
	}
	src, err := gobEncodeIfPresent(m.EnergySource)
	if err != nil {
		return machine, err
	}
	fixed, err := stubRecipeID(cfg, m.FixedRecipe)
	if err != nil {
		return machine, err
	}

	params := database.AddAssemblyMachineParams{
		EntityID:           m.entEntry.ID,
		CraftingCategories: cats,
		CraftingSpeed:      toNullFloat64(m.CraftingSpeed),
		EnergySource:       src,
		EnergyUsage:        toNullFloat64(m.EnergyUsage),
		FixedRecipe:        fixed,
	}

	existing, err := cfg.DB.GetAssemblyMachineByEntityID(cfg.CTX, m.entEntry.ID)
	if err == nil {
		machine, err = cfg.DB.UpdateAssemblyMachineByID(cfg.CTX, database.UpdateAssemblyMachineByIDParams{
			EntityID:           params.EntityID,
			CraftingCategories: params.CraftingCategories,
			CraftingSpeed:      params.CraftingSpeed,
			EnergySource:       params.EnergySource,
			EnergyUsage:        params.EnergyUsage,
			FixedRecipe:        params.FixedRecipe,
			ID:                 existing.ID,
		})
		if err != nil {
			return machine, err
		}
		m.machineEntry = &machine
		return machine, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return machine, err
	}

	machine, err = cfg.DB.AddAssemblyMachine(cfg.CTX, params)
	if err != nil {
		return machine, err
	}
	m.machineEntry = &machine
	return machine, nil
}

func (m *AssemblyMachine) GetFromDB(cfg *config.State) (database.AssemblyMachine, error) {
	var machine database.AssemblyMachine
	if m == nil {
		return machine, DatatypeNilError[AssemblyMachine]("Nil assembling machine")
	}
	if m.machineEntry == nil {
		return machine, fmt.Errorf("machineEntry not set")
	}
	if _, err := m.Entity.GetFromDB(cfg); err != nil {
		return machine, err
	}
	row, err := cfg.DB.GetAssemblyMachineByEntityID(cfg.CTX, m.entEntry.ID)
	if err != nil {
		return machine, err
	}
	machine = assemblyMachineFromJoin(row)
	if !rowsEqual(*m.machineEntry, machine) {
		return machine, mismatchError(cfg, "AssemblyMachine", *m.machineEntry, machine)
	}
	return machine, nil
}

func (m *AssemblyMachine) Unwrap() (database.AssemblyMachine, error) {
	if m == nil {
		return database.AssemblyMachine{}, DatatypeNilError[AssemblyMachine]("Nil assembling machine")
	}
	if m.machineEntry == nil {
		return database.AssemblyMachine{}, fmt.Errorf("machineEntry not set")
	}
	return *m.machineEntry, nil
}

func (m *AssemblyMachine) GetEntityID(cfg *config.State) (int64, error) {
	if m == nil {
		return 0, DatatypeNilError[AssemblyMachine]("Cannot get id of nil assembling machine")
	}
	return m.Entity.GetEntityID(cfg)
}

func stubRecipeID(cfg *config.State, name *string) (sql.NullInt64, error) {
	if name == nil || *name == "" {
		return sql.NullInt64{}, nil
	}
	ent, err := ensureEntity(cfg, *name, "recipe")
	if err != nil {
		return sql.NullInt64{}, err
	}
	existing, err := cfg.DB.GetRecipeByEntityID(cfg.CTX, ent.ID)
	if err == nil {
		return sql.NullInt64{Int64: existing.ID, Valid: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return sql.NullInt64{}, err
	}
	recipe, err := cfg.DB.AddRecipe(cfg.CTX, database.AddRecipeParams{
		EntityID: ent.ID,
	})
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: recipe.ID, Valid: true}, nil
}

func assemblyMachineFromJoin(row database.GetAssemblyMachineByEntityIDRow) database.AssemblyMachine {
	return database.AssemblyMachine{
		ID:                 row.ID,
		EntityID:           row.EntityID,
		CraftingCategories: row.CraftingCategories,
		CraftingSpeed:      row.CraftingSpeed,
		EnergySource:       row.EnergySource,
		EnergyUsage:        row.EnergyUsage,
		FixedRecipe:        row.FixedRecipe,
	}
}
