package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type Generator struct {
	Entity
	EnergySource       *EnergySource
	Effectivity        *float64
	FluidUsagePerTick  *float64
	MaximumTemperature *float64
	BurnsFluid         *bool
	InputFluid         *string
	generatorEntry     *database.Generator
}

func (g *Generator) UnmarshalJSON(rawJSON []byte) error {
	var raw struct {
		Name               string          `json:"name"`
		Type               string          `json:"type"`
		Order              *string         `json:"order"`
		EnergySource       *EnergySource   `json:"energy_source"`
		Effectivity        *float64        `json:"effectivity"`
		FluidUsagePerTick  *float64        `json:"fluid_usage_per_tick"`
		MaximumTemperature *float64        `json:"maximum_temperature"`
		BurnsFluid         *bool           `json:"burns_fluid"`
		FluidBox           *fluidBoxFilter `json:"fluid_box"`
	}
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return err
	}
	*g = Generator{
		Entity:             newEntity(raw.Name, raw.Type, raw.Order, rawJSON),
		EnergySource:       raw.EnergySource,
		Effectivity:        raw.Effectivity,
		FluidUsagePerTick:  raw.FluidUsagePerTick,
		MaximumTemperature: raw.MaximumTemperature,
		BurnsFluid:         raw.BurnsFluid,
		InputFluid:         filterOf(raw.FluidBox),
	}
	return nil
}

func (g *Generator) AddToDB(cfg *config.State) (database.Generator, error) {
	var generator database.Generator
	if g == nil {
		return generator, DatatypeNilError[Generator]("Nil generator")
	}
	if _, err := g.Entity.AddToDB(cfg); err != nil {
		return generator, err
	}

	src, err := gobEncodeIfPresent(g.EnergySource)
	if err != nil {
		return generator, err
	}
	input, err := stubEntityID(cfg, g.InputFluid, "fluid")
	if err != nil {
		return generator, err
	}

	params := database.AddGeneratorParams{
		EntityID:           g.entEntry.ID,
		EnergySource:       src,
		Effectivity:        toNullFloat64(g.Effectivity),
		FluidUsagePerTick:  toNullFloat64(g.FluidUsagePerTick),
		MaximumTemperature: toNullFloat64(g.MaximumTemperature),
		BurnsFluid:         toNullBoolInt(g.BurnsFluid),
		InputFluid:         input,
	}

	existing, err := cfg.DB.GetGeneratorByEntityID(cfg.CTX, g.entEntry.ID)
	if err == nil {
		generator, err = cfg.DB.UpdateGeneratorByID(cfg.CTX, database.UpdateGeneratorByIDParams{
			EntityID:           params.EntityID,
			EnergySource:       params.EnergySource,
			Effectivity:        params.Effectivity,
			FluidUsagePerTick:  params.FluidUsagePerTick,
			MaximumTemperature: params.MaximumTemperature,
			BurnsFluid:         params.BurnsFluid,
			InputFluid:         params.InputFluid,
			ID:                 existing.ID,
		})
		if err != nil {
			return generator, err
		}
		g.generatorEntry = &generator
		return generator, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return generator, err
	}

	generator, err = cfg.DB.AddGenerator(cfg.CTX, params)
	if err != nil {
		return generator, err
	}
	g.generatorEntry = &generator
	return generator, nil
}

func (g *Generator) GetFromDB(cfg *config.State) (database.Generator, error) {
	var generator database.Generator
	if g == nil {
		return generator, DatatypeNilError[Generator]("Nil generator")
	}
	if g.generatorEntry == nil {
		return generator, fmt.Errorf("generatorEntry not set")
	}
	if _, err := g.Entity.GetFromDB(cfg); err != nil {
		return generator, err
	}
	row, err := cfg.DB.GetGeneratorByEntityID(cfg.CTX, g.entEntry.ID)
	if err != nil {
		return generator, err
	}
	generator = generatorFromJoin(row)
	if !rowsEqual(*g.generatorEntry, generator) {
		return generator, mismatchError(cfg, "Generator", *g.generatorEntry, generator)
	}
	return generator, nil
}

func (g *Generator) Unwrap() (database.Generator, error) {
	if g == nil {
		return database.Generator{}, DatatypeNilError[Generator]("Nil generator")
	}
	if g.generatorEntry == nil {
		return database.Generator{}, fmt.Errorf("generatorEntry not set")
	}
	return *g.generatorEntry, nil
}

func (g *Generator) GetEntityID(cfg *config.State) (int64, error) {
	if g == nil {
		return 0, DatatypeNilError[Generator]("Cannot get id of nil generator")
	}
	return g.Entity.GetEntityID(cfg)
}

func generatorFromJoin(row database.GetGeneratorByEntityIDRow) database.Generator {
	return database.Generator{
		ID:                 row.ID,
		EntityID:           row.EntityID,
		EnergySource:       row.EnergySource,
		Effectivity:        row.Effectivity,
		FluidUsagePerTick:  row.FluidUsagePerTick,
		MaximumTemperature: row.MaximumTemperature,
		BurnsFluid:         row.BurnsFluid,
		InputFluid:         row.InputFluid,
	}
}
