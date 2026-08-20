package datatypes

import (
	"encoding/json"
	"testing"
)

func TestHumanize(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"iron-plate", "Iron plate"},
		{"item-name.iron-plate", "Iron plate"},
		{"assembling-machine-1", "Assembling machine 1"},
		{"entity-name.dino-dig-site", "Dino dig site"},
		{"crude_oil", "Crude oil"},
		{"parameter-x", "Parameter x"},
	}
	for _, tc := range cases {
		if got := Humanize(tc.in); got != tc.want {
			t.Errorf("Humanize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDisplay(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		fallback string
		want     string
	}{
		{"missing", "", "iron-plate", "Iron plate"},
		{"null", "null", "iron-plate", "Iron plate"},
		{"empty array", "[]", "wood", "Wood"},
		{"string key", `"entity-name.uranium-cannon-explosion"`, "proxy", "Uranium cannon explosion"},
		{"literal suffix", `" (Legacy)"`, "x", " (Legacy)"},
		{
			"question fallback",
			`["?", ["recipe-name.iron-plate"], ["item-name.iron-plate"]]`,
			"iron-plate",
			"Iron plate",
		},
		{
			"concat legacy",
			`["", ["entity-name.evaporator"], " (Legacy)"]`,
			"evaporator-legacy",
			"Evaporator (Legacy)",
		},
		{
			"key with arg",
			`["item-name.filled-barrel", ["fluid-name.water"]]`,
			"water-barrel",
			"Filled barrel (Water)",
		},
		{
			"parameter",
			`["parameter-x", "0"]`,
			"parameter-0",
			"Parameter x (0)",
		},
		{
			"different key than id",
			`["entity-name.dino-dig-site"]`,
			"pipette-dino-dig-site",
			"Dino dig site",
		},
		{
			"question skips empty",
			`["?", "", ["item-name.wood"]]`,
			"wood",
			"Wood",
		},
	}
	for _, tc := range cases {
		var raw json.RawMessage
		if tc.raw != "" {
			raw = json.RawMessage(tc.raw)
		}
		if got := Display(raw, tc.fallback); got != tc.want {
			t.Errorf("%s: Display(%s, %q) = %q, want %q", tc.name, tc.raw, tc.fallback, got, tc.want)
		}
	}
}

func TestParseLocalisedName(t *testing.T) {
	got := parseLocalisedName([]byte(`{"name":"wood","localised_name":["item-name.wood"]}`))
	if string(got) != `["item-name.wood"]` {
		t.Errorf("parseLocalisedName = %s", got)
	}
	if parseLocalisedName([]byte(`{"name":"wood"}`)) != nil {
		t.Error("missing localised_name should be nil")
	}
	if parseLocalisedName([]byte(`{"localised_name":null}`)) != nil {
		t.Error("null localised_name should be nil")
	}
}
