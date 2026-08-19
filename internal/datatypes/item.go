package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type Item struct {
	Entity
	StackSize   *int64
	BurntResult *string
	FuelValue   *float64
	SpoilResult *string
	SpoilTicks  *int64
	Weight      *int64
	itemEntry   *database.Item
}

func (i *Item) UnmarshalJSON(b []byte) error {
	var raw struct {
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Order       *string `json:"order"`
		StackSize   *int64  `json:"stack_size"`
		BurntResult *string `json:"burnt_result"`
		FuelValue   *string `json:"fuel_value"`
		SpoilResult *string `json:"spoil_result"`
		SpoilTicks  *int64  `json:"spoil_ticks"`
		Weight      *int64  `json:"weight"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	fuel, err := parseOptionalEnergy(raw.FuelValue)
	if err != nil {
		return fmt.Errorf("item %q fuel_value: %w", raw.Name, err)
	}
	*i = Item{
		Entity: Entity{
			Name:        raw.Name,
			Type:        raw.Type,
			EntityOrder: raw.Order,
		},
		StackSize:   raw.StackSize,
		BurntResult: raw.BurntResult,
		FuelValue:   fuel,
		SpoilResult: raw.SpoilResult,
		SpoilTicks:  raw.SpoilTicks,
		Weight:      raw.Weight,
	}
	return nil
}

func (i *Item) AddToDB(cfg *config.State) (database.Item, error) {
	var item database.Item
	if i == nil {
		return item, DatatypeNilError[Item]("Nil item")
	}
	if _, err := i.Entity.AddToDB(cfg); err != nil {
		return item, err
	}

	burnt, err := stubEntityID(cfg, i.BurntResult, "item")
	if err != nil {
		return item, err
	}
	spoil, err := stubEntityID(cfg, i.SpoilResult, "item")
	if err != nil {
		return item, err
	}

	params := database.AddItemParams{
		EntityID:    i.entEntry.ID,
		StackSize:   toNullInt64(i.StackSize),
		BurntResult: burnt,
		FuelValue:   toNullFloat64(i.FuelValue),
		SpoilResult: spoil,
		SpoilTicks:  toNullInt64(i.SpoilTicks),
		Weight:      toNullInt64(i.Weight),
	}

	existing, err := cfg.DB.GetItemByEntityID(cfg.CTX, i.entEntry.ID)
	if err == nil {
		item, err = cfg.DB.UpdateItemByID(cfg.CTX, database.UpdateItemByIDParams{
			EntityID:    params.EntityID,
			StackSize:   params.StackSize,
			BurntResult: params.BurntResult,
			FuelValue:   params.FuelValue,
			SpoilResult: params.SpoilResult,
			SpoilTicks:  params.SpoilTicks,
			Weight:      params.Weight,
			ID:          existing.ID,
		})
		if err != nil {
			return item, err
		}
		i.itemEntry = &item
		return item, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return item, err
	}

	item, err = cfg.DB.AddItem(cfg.CTX, params)
	if err != nil {
		return item, err
	}
	i.itemEntry = &item
	return item, nil
}

func (i *Item) GetFromDB(cfg *config.State) (database.Item, error) {
	var item database.Item
	if i == nil {
		return item, DatatypeNilError[Item]("Nil item")
	}
	if i.itemEntry == nil {
		return item, fmt.Errorf("itemEntry not set")
	}
	if _, err := i.Entity.GetFromDB(cfg); err != nil {
		return item, err
	}
	row, err := cfg.DB.GetItemByEntityID(cfg.CTX, i.entEntry.ID)
	if err != nil {
		return item, err
	}
	item = itemFromJoin(row)
	if !rowsEqual(*i.itemEntry, item) {
		return item, mismatchError(cfg, "Item", *i.itemEntry, item)
	}
	return item, nil
}

func (i *Item) Unwrap() (database.Item, error) {
	if i == nil {
		return database.Item{}, DatatypeNilError[Item]("Nil item")
	}
	if i.itemEntry == nil {
		return database.Item{}, fmt.Errorf("itemEntry not set")
	}
	return *i.itemEntry, nil
}

func (i *Item) GetEntityID(cfg *config.State) (int64, error) {
	if i == nil {
		return 0, DatatypeNilError[Item]("Cannot get id of nil item")
	}
	return i.Entity.GetEntityID(cfg)
}

func itemFromJoin(row database.GetItemByEntityIDRow) database.Item {
	return database.Item{
		ID:          row.ID,
		EntityID:    row.EntityID,
		StackSize:   row.StackSize,
		BurntResult: row.BurntResult,
		FuelValue:   row.FuelValue,
		SpoilResult: row.SpoilResult,
		SpoilTicks:  row.SpoilTicks,
		Weight:      row.Weight,
	}
}
