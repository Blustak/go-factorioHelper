package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

// DefaultResourceCategory is Factorio's implicit category when a resource omits one.
const DefaultResourceCategory = "basic-solid"

type Resource struct {
	Entity
	MiningTime    *float64
	RequiredFluid *string
	Category      *string
	Minable       MinableResults
	resourceEntry *database.Resource
}

func (r *Resource) UnmarshalJSON(b []byte) error {
	var raw struct {
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		Order    *string `json:"order"`
		Category *string `json:"category"`
		Minable  *struct {
			MiningTime    *float64        `json:"mining_time"`
			Results       json.RawMessage `json:"results"`
			Result        *string         `json:"result"`
			ResultCount   *float64        `json:"result_count"`
			RequiredFluid *string         `json:"required_fluid"`
			FluidAmount   *float64        `json:"fluid_amount"`
		} `json:"minable"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	category := raw.Category
	if category == nil || *category == "" {
		cat := DefaultResourceCategory
		category = &cat
	}
	res := Resource{
		Entity:   newEntity(raw.Name, raw.Type, raw.Order, b),
		Category: category,
	}
	if raw.Minable != nil {
		results, err := unmarshalFactorioArray[Product](raw.Minable.Results)
		if err != nil {
			return fmt.Errorf("resource %q minable.results: %w", raw.Name, err)
		}
		res.MiningTime = raw.Minable.MiningTime
		res.RequiredFluid = raw.Minable.RequiredFluid
		res.Minable = MinableResults{
			Results:     results,
			FluidAmount: raw.Minable.FluidAmount,
		}
		if len(res.Minable.Results) == 0 && raw.Minable.Result != nil && *raw.Minable.Result != "" {
			amount := 1.0
			if raw.Minable.ResultCount != nil {
				amount = *raw.Minable.ResultCount
			}
			res.Minable.Results = []Product{
				{
					Entity: Entity{Name: *raw.Minable.Result, Type: "item"},
					Amount: &amount,
				},
			}
		}
	}
	*r = res
	return nil
}

func (r *Resource) AddToDB(cfg *config.State) (database.Resource, error) {
	var resource database.Resource
	if r == nil {
		return resource, DatatypeNilError[Resource]("Nil resource")
	}
	if _, err := r.Entity.AddToDB(cfg); err != nil {
		return resource, err
	}
	if err := stubProducts(cfg, r.Minable.Results); err != nil {
		return resource, err
	}
	required, err := stubEntityID(cfg, r.RequiredFluid, "fluid")
	if err != nil {
		return resource, err
	}
	blob, err := gobEncode(r.Minable)
	if err != nil {
		return resource, err
	}

	params := database.AddResourceParams{
		EntityID:      r.entEntry.ID,
		MiningTime:    toNullFloat64(r.MiningTime),
		Results:       blob,
		RequiredFluid: required,
		Category:      toNullString(r.Category),
	}

	existing, err := cfg.DB.GetResourceByEntityID(cfg.CTX, r.entEntry.ID)
	if err == nil {
		resource, err = cfg.DB.UpdateResourceByID(cfg.CTX, database.UpdateResourceByIDParams{
			EntityID:      params.EntityID,
			MiningTime:    params.MiningTime,
			Results:       params.Results,
			RequiredFluid: params.RequiredFluid,
			Category:      params.Category,
			ID:            existing.ID,
		})
		if err != nil {
			return resource, err
		}
		r.resourceEntry = &resource
		return resource, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return resource, err
	}

	resource, err = cfg.DB.AddResource(cfg.CTX, params)
	if err != nil {
		return resource, err
	}
	r.resourceEntry = &resource
	return resource, nil
}

func (r *Resource) GetFromDB(cfg *config.State) (database.Resource, error) {
	var resource database.Resource
	if r == nil {
		return resource, DatatypeNilError[Resource]("Nil resource")
	}
	if r.resourceEntry == nil {
		return resource, fmt.Errorf("resourceEntry not set")
	}
	if _, err := r.Entity.GetFromDB(cfg); err != nil {
		return resource, err
	}
	row, err := cfg.DB.GetResourceByEntityID(cfg.CTX, r.entEntry.ID)
	if err != nil {
		return resource, err
	}
	resource = resourceFromJoin(row)
	if !rowsEqual(*r.resourceEntry, resource) {
		return resource, mismatchError(cfg, "Resource", *r.resourceEntry, resource)
	}
	return resource, nil
}

func (r *Resource) Unwrap() (database.Resource, error) {
	if r == nil {
		return database.Resource{}, DatatypeNilError[Resource]("Nil resource")
	}
	if r.resourceEntry == nil {
		return database.Resource{}, fmt.Errorf("resourceEntry not set")
	}
	return *r.resourceEntry, nil
}

func (r *Resource) GetEntityID(cfg *config.State) (int64, error) {
	if r == nil {
		return 0, DatatypeNilError[Resource]("Cannot get id of nil resource")
	}
	return r.Entity.GetEntityID(cfg)
}

func resourceFromJoin(row database.GetResourceByEntityIDRow) database.Resource {
	return database.Resource{
		ID:            row.ID,
		EntityID:      row.EntityID,
		MiningTime:    row.MiningTime,
		Results:       row.Results,
		RequiredFluid: row.RequiredFluid,
		Category:      row.Category,
	}
}
