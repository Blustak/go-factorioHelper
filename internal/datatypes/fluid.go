package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type Fluid struct {
	Entity
	FuelValue          *float64
	GasTemperature     *int64
	DefaultTemperature *int64
	MaxTemperature     *int64
	fluidEntry         *database.Fluid
}

func (f *Fluid) UnmarshalJSON(b []byte) error {
	var raw struct {
		Name               string   `json:"name"`
		Type               string   `json:"type"`
		Order              *string  `json:"order"`
		FuelValue          *string  `json:"fuel_value"`
		GasTemperature     *float64 `json:"gas_temperature"`
		DefaultTemperature *float64 `json:"default_temperature"`
		MaxTemperature     *float64 `json:"max_temperature"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	fuel, err := parseOptionalEnergy(raw.FuelValue)
	if err != nil {
		return fmt.Errorf("fluid %q fuel_value: %w", raw.Name, err)
	}
	*f = Fluid{
		Entity: Entity{
			Name:        raw.Name,
			Type:        raw.Type,
			EntityOrder: raw.Order,
		},
		FuelValue:          fuel,
		GasTemperature:     int64FromFloat(raw.GasTemperature),
		DefaultTemperature: int64FromFloat(raw.DefaultTemperature),
		MaxTemperature:     int64FromFloat(raw.MaxTemperature),
	}
	return nil
}

func (f *Fluid) AddToDB(cfg *config.State) (database.Fluid, error) {
	var fluid database.Fluid
	if f == nil {
		return fluid, DatatypeNilError[Fluid]("Nil fluid")
	}
	if _, err := f.Entity.AddToDB(cfg); err != nil {
		return fluid, err
	}

	params := database.AddFluidParams{
		EntityID:           f.entEntry.ID,
		FuelValue:          toNullFloat64(f.FuelValue),
		GasTemperature:     toNullInt64(f.GasTemperature),
		DefaultTemperature: toNullInt64(f.DefaultTemperature),
		MaxTemperature:     toNullInt64(f.MaxTemperature),
	}

	existing, err := cfg.DB.GetFluidByEntityID(cfg.CTX, f.entEntry.ID)
	if err == nil {
		fluid, err = cfg.DB.UpdateFluidByID(cfg.CTX, database.UpdateFluidByIDParams{
			EntityID:           params.EntityID,
			FuelValue:          params.FuelValue,
			GasTemperature:     params.GasTemperature,
			DefaultTemperature: params.DefaultTemperature,
			MaxTemperature:     params.MaxTemperature,
			ID:                 existing.ID,
		})
		if err != nil {
			return fluid, err
		}
		f.fluidEntry = &fluid
		return fluid, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fluid, err
	}

	fluid, err = cfg.DB.AddFluid(cfg.CTX, params)
	if err != nil {
		return fluid, err
	}
	f.fluidEntry = &fluid
	return fluid, nil
}

func (f *Fluid) GetFromDB(cfg *config.State) (database.Fluid, error) {
	var fluid database.Fluid
	if f == nil {
		return fluid, DatatypeNilError[Fluid]("Nil fluid")
	}
	if f.fluidEntry == nil {
		return fluid, fmt.Errorf("fluidEntry not set")
	}
	if _, err := f.Entity.GetFromDB(cfg); err != nil {
		return fluid, err
	}
	row, err := cfg.DB.GetFluidByEntityID(cfg.CTX, f.entEntry.ID)
	if err != nil {
		return fluid, err
	}
	fluid = fluidFromJoin(row)
	if !rowsEqual(*f.fluidEntry, fluid) {
		return fluid, mismatchError(cfg, "Fluid", *f.fluidEntry, fluid)
	}
	return fluid, nil
}

func (f *Fluid) Unwrap() (database.Fluid, error) {
	if f == nil {
		return database.Fluid{}, DatatypeNilError[Fluid]("Nil fluid")
	}
	if f.fluidEntry == nil {
		return database.Fluid{}, fmt.Errorf("fluidEntry not set")
	}
	return *f.fluidEntry, nil
}

func (f *Fluid) GetEntityID(cfg *config.State) (int64, error) {
	if f == nil {
		return 0, DatatypeNilError[Fluid]("Cannot get id of nil fluid")
	}
	return f.Entity.GetEntityID(cfg)
}

func fluidFromJoin(row database.GetFluidByEntityIDRow) database.Fluid {
	return database.Fluid{
		ID:                 row.ID,
		EntityID:           row.EntityID,
		FuelValue:          row.FuelValue,
		GasTemperature:     row.GasTemperature,
		DefaultTemperature: row.DefaultTemperature,
		MaxTemperature:     row.MaxTemperature,
	}
}
