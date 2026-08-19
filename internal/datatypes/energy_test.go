package datatypes

import "testing"

func TestParseEnergy(t *testing.T) {
	tests := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{in: "2MJ", want: 2e6},
		{in: "82.5kJ", want: 82500},
		{in: "4GJ", want: 4e9},
		{in: "1J", want: 1},
		{in: "75kW", wantErr: true},
		{in: "not-a-quantity", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseEnergy(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseEnergy(%q) error = nil, want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseEnergy(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseEnergy(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParsePower(t *testing.T) {
	tests := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{in: "75kW", want: 75000},
		{in: "150kW", want: 150000},
		{in: "1MW", want: 1e6},
		{in: "18W", want: 18},
		{in: "2MJ", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parsePower(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parsePower(%q) error = nil, want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePower(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parsePower(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
