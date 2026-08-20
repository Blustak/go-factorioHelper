package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type fluidBoxFilter struct {
	Filter *string `json:"filter"`
}

type Boiler struct {
	Entity
	EnergySource      *EnergySource
	EnergyConsumption *float64
	TargetTemperature *float64
	Mode              *string
	InputFluid        *string
	OutputFluid       *string
	boilerEntry       *database.Boiler
}

func (b *Boiler) UnmarshalJSON(rawJSON []byte) error {
	var raw struct {
		Name              string          `json:"name"`
		Type              string          `json:"type"`
		Order             *string         `json:"order"`
		EnergySource      *EnergySource   `json:"energy_source"`
		EnergyConsumption *string         `json:"energy_consumption"`
		TargetTemperature *float64        `json:"target_temperature"`
		Mode              *string         `json:"mode"`
		FluidBox          *fluidBoxFilter `json:"fluid_box"`
		OutputFluidBox    *fluidBoxFilter `json:"output_fluid_box"`
	}
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return err
	}
	consumption, err := parseOptionalPower(raw.EnergyConsumption)
	if err != nil {
		return fmt.Errorf("boiler %q energy_consumption: %w", raw.Name, err)
	}
	*b = Boiler{
		Entity:            newEntity(raw.Name, raw.Type, raw.Order, rawJSON),
		EnergySource:      raw.EnergySource,
		EnergyConsumption: consumption,
		TargetTemperature: raw.TargetTemperature,
		Mode:              raw.Mode,
		InputFluid:        filterOf(raw.FluidBox),
		OutputFluid:       filterOf(raw.OutputFluidBox),
	}
	return nil
}

func filterOf(box *fluidBoxFilter) *string {
	if box == nil {
		return nil
	}
	if box.Filter == nil || *box.Filter == "" {
		return nil
	}
	return box.Filter
}

func (b *Boiler) AddToDB(cfg *config.State) (database.Boiler, error) {
	var boiler database.Boiler
	if b == nil {
		return boiler, DatatypeNilError[Boiler]("Nil boiler")
	}
	if _, err := b.Entity.AddToDB(cfg); err != nil {
		return boiler, err
	}

	src, err := gobEncodeIfPresent(b.EnergySource)
	if err != nil {
		return boiler, err
	}
	input, err := stubEntityID(cfg, b.InputFluid, "fluid")
	if err != nil {
		return boiler, err
	}
	output, err := stubEntityID(cfg, b.OutputFluid, "fluid")
	if err != nil {
		return boiler, err
	}

	params := database.AddBoilerParams{
		EntityID:          b.entEntry.ID,
		EnergySource:      src,
		EnergyConsumption: toNullFloat64(b.EnergyConsumption),
		TargetTemperature: toNullFloat64(b.TargetTemperature),
		Mode:              toNullString(b.Mode),
		InputFluid:        input,
		OutputFluid:       output,
	}

	existing, err := cfg.DB.GetBoilerByEntityID(cfg.CTX, b.entEntry.ID)
	if err == nil {
		boiler, err = cfg.DB.UpdateBoilerByID(cfg.CTX, database.UpdateBoilerByIDParams{
			EntityID:          params.EntityID,
			EnergySource:      params.EnergySource,
			EnergyConsumption: params.EnergyConsumption,
			TargetTemperature: params.TargetTemperature,
			Mode:              params.Mode,
			InputFluid:        params.InputFluid,
			OutputFluid:       params.OutputFluid,
			ID:                existing.ID,
		})
		if err != nil {
			return boiler, err
		}
		b.boilerEntry = &boiler
		return boiler, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return boiler, err
	}

	boiler, err = cfg.DB.AddBoiler(cfg.CTX, params)
	if err != nil {
		return boiler, err
	}
	b.boilerEntry = &boiler
	return boiler, nil
}

func (b *Boiler) GetFromDB(cfg *config.State) (database.Boiler, error) {
	var boiler database.Boiler
	if b == nil {
		return boiler, DatatypeNilError[Boiler]("Nil boiler")
	}
	if b.boilerEntry == nil {
		return boiler, fmt.Errorf("boilerEntry not set")
	}
	if _, err := b.Entity.GetFromDB(cfg); err != nil {
		return boiler, err
	}
	row, err := cfg.DB.GetBoilerByEntityID(cfg.CTX, b.entEntry.ID)
	if err != nil {
		return boiler, err
	}
	boiler = boilerFromJoin(row)
	if !rowsEqual(*b.boilerEntry, boiler) {
		return boiler, mismatchError(cfg, "Boiler", *b.boilerEntry, boiler)
	}
	return boiler, nil
}

func (b *Boiler) Unwrap() (database.Boiler, error) {
	if b == nil {
		return database.Boiler{}, DatatypeNilError[Boiler]("Nil boiler")
	}
	if b.boilerEntry == nil {
		return database.Boiler{}, fmt.Errorf("boilerEntry not set")
	}
	return *b.boilerEntry, nil
}

func (b *Boiler) GetEntityID(cfg *config.State) (int64, error) {
	if b == nil {
		return 0, DatatypeNilError[Boiler]("Cannot get id of nil boiler")
	}
	return b.Entity.GetEntityID(cfg)
}

func boilerFromJoin(row database.GetBoilerByEntityIDRow) database.Boiler {
	return database.Boiler{
		ID:                row.ID,
		EntityID:          row.EntityID,
		EnergySource:      row.EnergySource,
		EnergyConsumption: row.EnergyConsumption,
		TargetTemperature: row.TargetTemperature,
		Mode:              row.Mode,
		InputFluid:        row.InputFluid,
		OutputFluid:       row.OutputFluid,
	}
}
