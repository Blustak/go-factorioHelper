package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type ResourceProducer struct {
	Entity
	ResourceCategories []string
	MiningSpeed        *float64
	PumpingSpeed       *float64
	ProducedFluid      *string
	EnergySource       *EnergySource
	EnergyUsage        *float64
	producerEntry      *database.ResourceProducer
}

func (p *ResourceProducer) UnmarshalJSON(b []byte) error {
	var raw struct {
		Name               string   `json:"name"`
		Type               string   `json:"type"`
		Order              *string  `json:"order"`
		ResourceCategories []string `json:"resource_categories"`
		MiningSpeed        *float64 `json:"mining_speed"`
		PumpingSpeed       *float64 `json:"pumping_speed"`
		Fluid              *string  `json:"fluid"`
		FluidBox           *struct {
			Filter *string `json:"filter"`
		} `json:"fluid_box"`
		EnergySource *EnergySource `json:"energy_source"`
		EnergyUsage  *string       `json:"energy_usage"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	usage, err := parseOptionalPower(raw.EnergyUsage)
	if err != nil {
		return fmt.Errorf("resource producer %q energy_usage: %w", raw.Name, err)
	}
	produced := raw.Fluid
	if (produced == nil || *produced == "") && raw.FluidBox != nil {
		produced = raw.FluidBox.Filter
	}
	*p = ResourceProducer{
		Entity:             newEntity(raw.Name, raw.Type, raw.Order, b),
		ResourceCategories: raw.ResourceCategories,
		MiningSpeed:        raw.MiningSpeed,
		PumpingSpeed:       raw.PumpingSpeed,
		ProducedFluid:      produced,
		EnergySource:       raw.EnergySource,
		EnergyUsage:        usage,
	}
	return nil
}

func (p *ResourceProducer) AddToDB(cfg *config.State) (database.ResourceProducer, error) {
	var producer database.ResourceProducer
	if p == nil {
		return producer, DatatypeNilError[ResourceProducer]("Nil resource producer")
	}
	if _, err := p.Entity.AddToDB(cfg); err != nil {
		return producer, err
	}

	cats, err := gobEncodeSlice(p.ResourceCategories)
	if err != nil {
		return producer, err
	}
	src, err := gobEncodeIfPresent(p.EnergySource)
	if err != nil {
		return producer, err
	}
	fluid, err := stubEntityID(cfg, p.ProducedFluid, "fluid")
	if err != nil {
		return producer, err
	}

	params := database.AddResourceProducerParams{
		EntityID:           p.entEntry.ID,
		ResourceCategories: cats,
		MiningSpeed:        toNullFloat64(p.MiningSpeed),
		PumpingSpeed:       toNullFloat64(p.PumpingSpeed),
		ProducedFluid:      fluid,
		EnergySource:       src,
		EnergyUsage:        toNullFloat64(p.EnergyUsage),
	}

	existing, err := cfg.DB.GetResourceProducerByEntityID(cfg.CTX, p.entEntry.ID)
	if err == nil {
		producer, err = cfg.DB.UpdateResourceProducerByID(cfg.CTX, database.UpdateResourceProducerByIDParams{
			EntityID:           params.EntityID,
			ResourceCategories: params.ResourceCategories,
			MiningSpeed:        params.MiningSpeed,
			PumpingSpeed:       params.PumpingSpeed,
			ProducedFluid:      params.ProducedFluid,
			EnergySource:       params.EnergySource,
			EnergyUsage:        params.EnergyUsage,
			ID:                 existing.ID,
		})
		if err != nil {
			return producer, err
		}
		p.producerEntry = &producer
		return producer, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return producer, err
	}

	producer, err = cfg.DB.AddResourceProducer(cfg.CTX, params)
	if err != nil {
		return producer, err
	}
	p.producerEntry = &producer
	return producer, nil
}

func (p *ResourceProducer) GetFromDB(cfg *config.State) (database.ResourceProducer, error) {
	var producer database.ResourceProducer
	if p == nil {
		return producer, DatatypeNilError[ResourceProducer]("Nil resource producer")
	}
	if p.producerEntry == nil {
		return producer, fmt.Errorf("producerEntry not set")
	}
	if _, err := p.Entity.GetFromDB(cfg); err != nil {
		return producer, err
	}
	row, err := cfg.DB.GetResourceProducerByEntityID(cfg.CTX, p.entEntry.ID)
	if err != nil {
		return producer, err
	}
	producer = resourceProducerFromJoin(row)
	if !rowsEqual(*p.producerEntry, producer) {
		return producer, mismatchError(cfg, "ResourceProducer", *p.producerEntry, producer)
	}
	return producer, nil
}

func (p *ResourceProducer) Unwrap() (database.ResourceProducer, error) {
	if p == nil {
		return database.ResourceProducer{}, DatatypeNilError[ResourceProducer]("Nil resource producer")
	}
	if p.producerEntry == nil {
		return database.ResourceProducer{}, fmt.Errorf("producerEntry not set")
	}
	return *p.producerEntry, nil
}

func (p *ResourceProducer) GetEntityID(cfg *config.State) (int64, error) {
	if p == nil {
		return 0, DatatypeNilError[ResourceProducer]("Cannot get id of nil resource producer")
	}
	return p.Entity.GetEntityID(cfg)
}

func resourceProducerFromJoin(row database.GetResourceProducerByEntityIDRow) database.ResourceProducer {
	return database.ResourceProducer{
		ID:                 row.ID,
		EntityID:           row.EntityID,
		ResourceCategories: row.ResourceCategories,
		MiningSpeed:        row.MiningSpeed,
		PumpingSpeed:       row.PumpingSpeed,
		ProducedFluid:      row.ProducedFluid,
		EnergySource:       row.EnergySource,
		EnergyUsage:        row.EnergyUsage,
	}
}
