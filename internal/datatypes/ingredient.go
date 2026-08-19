package datatypes

import (
	"encoding/json"

	"github.com/Blustak/go-factorioHelper/internal/config"
)

type Ingredient struct {
	Entity
	Amount         *float64 `json:"amount"`
	Probability    *float64 `json:"probability"`
	Temperature    *float64 `json:"temperature"`
	FluidboxIndex  *int64   `json:"fluidbox_index"`
	IgnoredByStats *int64   `json:"ignored_by_stats"`
}

func (in *Ingredient) UnmarshalJSON(b []byte) error {
	var raw struct {
		Name           string   `json:"name"`
		Type           string   `json:"type"`
		Order          *string  `json:"order"`
		Amount         *float64 `json:"amount"`
		Probability    *float64 `json:"probability"`
		Temperature    *float64 `json:"temperature"`
		FluidboxIndex  *int64   `json:"fluidbox_index"`
		IgnoredByStats *int64   `json:"ignored_by_stats"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*in = Ingredient{
		Entity: Entity{
			Name:        raw.Name,
			Type:        raw.Type,
			EntityOrder: raw.Order,
		},
		Amount:         raw.Amount,
		Probability:    raw.Probability,
		Temperature:    raw.Temperature,
		FluidboxIndex:  raw.FluidboxIndex,
		IgnoredByStats: raw.IgnoredByStats,
	}
	return nil
}

type Product struct {
	Entity
	Amount                *float64 `json:"amount"`
	AmountMin             *float64 `json:"amount_min"`
	AmountMax             *float64 `json:"amount_max"`
	Probability           *float64 `json:"probability"`
	Temperature           *float64 `json:"temperature"`
	FluidboxIndex         *int64   `json:"fluidbox_index"`
	IgnoredByStats        *int64   `json:"ignored_by_stats"`
	IgnoredByProductivity *int64   `json:"ignored_by_productivity"`
	ExtraCountFraction    *float64 `json:"extra_count_fraction"`
}

func (p *Product) UnmarshalJSON(b []byte) error {
	var raw struct {
		Name                  string   `json:"name"`
		Type                  string   `json:"type"`
		Order                 *string  `json:"order"`
		Amount                *float64 `json:"amount"`
		AmountMin             *float64 `json:"amount_min"`
		AmountMax             *float64 `json:"amount_max"`
		Probability           *float64 `json:"probability"`
		Temperature           *float64 `json:"temperature"`
		FluidboxIndex         *int64   `json:"fluidbox_index"`
		IgnoredByStats        *int64   `json:"ignored_by_stats"`
		IgnoredByProductivity *int64   `json:"ignored_by_productivity"`
		ExtraCountFraction    *float64 `json:"extra_count_fraction"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*p = Product{
		Entity: Entity{
			Name:        raw.Name,
			Type:        raw.Type,
			EntityOrder: raw.Order,
		},
		Amount:                raw.Amount,
		AmountMin:             raw.AmountMin,
		AmountMax:             raw.AmountMax,
		Probability:           raw.Probability,
		Temperature:           raw.Temperature,
		FluidboxIndex:         raw.FluidboxIndex,
		IgnoredByStats:        raw.IgnoredByStats,
		IgnoredByProductivity: raw.IgnoredByProductivity,
		ExtraCountFraction:    raw.ExtraCountFraction,
	}
	return nil
}

type MinableResults struct {
	Results     []Product
	FluidAmount *float64
}

func stubIngredients(cfg *config.State, ings []Ingredient) error {
	for i := range ings {
		if err := stubProductEntity(cfg, ings[i].Name, ings[i].Type); err != nil {
			return err
		}
	}
	return nil
}

func stubProducts(cfg *config.State, products []Product) error {
	for i := range products {
		if err := stubProductEntity(cfg, products[i].Name, products[i].Type); err != nil {
			return err
		}
	}
	return nil
}

func stubProductEntity(cfg *config.State, name, prototypeType string) error {
	if name == "" {
		return nil
	}
	if prototypeType == "" {
		prototypeType = "item"
	}
	_, err := ensureEntity(cfg, name, prototypeType)
	return err
}
