package datatypes

import (
	"encoding/json"
	"fmt"
)

type EnergySource struct {
	Type               string
	FuelCategories     []string
	Effectivity        *float64
	FuelInventorySize  *int64
	BurntInventorySize *int64
	UsagePriority      *string
	Drain              *float64
	BurnsFluid         *bool
	ScaleFluidUsage    *bool
	FluidUsagePerTick  *float64
	MaximumTemperature *float64
	EmissionsPerMinute *float64
}

func (e *EnergySource) UnmarshalJSON(b []byte) error {
	var raw struct {
		Type               string             `json:"type"`
		FuelCategories     []string           `json:"fuel_categories"`
		FuelCategory       string             `json:"fuel_category"`
		Effectivity        *float64           `json:"effectivity"`
		FuelInventorySize  *int64             `json:"fuel_inventory_size"`
		BurntInventorySize *int64             `json:"burnt_inventory_size"`
		UsagePriority      *string            `json:"usage_priority"`
		Drain              *string            `json:"drain"`
		BurnsFluid         *bool              `json:"burns_fluid"`
		ScaleFluidUsage    *bool              `json:"scale_fluid_usage"`
		FluidUsagePerTick  *float64           `json:"fluid_usage_per_tick"`
		MaximumTemperature *float64           `json:"maximum_temperature"`
		EmissionsPerMinute map[string]float64 `json:"emissions_per_minute"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	cats := raw.FuelCategories
	if len(cats) == 0 && raw.FuelCategory != "" {
		cats = []string{raw.FuelCategory}
	}

	var drain *float64
	if raw.Drain != nil && *raw.Drain != "" {
		w, err := parsePower(*raw.Drain)
		if err != nil {
			return fmt.Errorf("energy_source.drain: %w", err)
		}
		drain = &w
	}

	var pollution *float64
	if raw.EmissionsPerMinute != nil {
		if v, ok := raw.EmissionsPerMinute["pollution"]; ok {
			pollution = &v
		}
	}

	*e = EnergySource{
		Type:               raw.Type,
		FuelCategories:     cats,
		Effectivity:        raw.Effectivity,
		FuelInventorySize:  raw.FuelInventorySize,
		BurntInventorySize: raw.BurntInventorySize,
		UsagePriority:      raw.UsagePriority,
		Drain:              drain,
		BurnsFluid:         raw.BurnsFluid,
		ScaleFluidUsage:    raw.ScaleFluidUsage,
		FluidUsagePerTick:  raw.FluidUsagePerTick,
		MaximumTemperature: raw.MaximumTemperature,
		EmissionsPerMinute: pollution,
	}
	return nil
}
