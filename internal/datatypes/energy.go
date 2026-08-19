package datatypes

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var quantityRe = regexp.MustCompile(`(?i)^([0-9]*\.?[0-9]+(?:[eE][+-]?[0-9]+)?)\s*([kMGT]?)([jJwW])$`)

var siPrefix = map[byte]float64{
	'k': 1e3,
	'K': 1e3,
	'M': 1e6,
	'G': 1e9,
	'T': 1e12,
}

func parseFactorioQuantity(s string, wantJoule bool) (float64, error) {
	s = strings.TrimSpace(s)
	m := quantityRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid Factorio quantity %q", s)
	}
	value, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid Factorio quantity %q: %w", s, err)
	}
	if m[2] != "" {
		mult, ok := siPrefix[m[2][0]]
		if !ok {
			return 0, fmt.Errorf("unknown SI prefix in %q", s)
		}
		value *= mult
	}
	unit := m[3][0]
	isJoule := unit == 'j' || unit == 'J'
	if wantJoule && !isJoule {
		return 0, fmt.Errorf("expected energy in joules, got %q", s)
	}
	if !wantJoule && isJoule {
		return 0, fmt.Errorf("expected power in watts, got %q", s)
	}
	return value, nil
}

func parseEnergy(s string) (float64, error) {
	return parseFactorioQuantity(s, true)
}

func parsePower(s string) (float64, error) {
	return parseFactorioQuantity(s, false)
}

func parseOptionalEnergy(s *string) (*float64, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	v, err := parseEnergy(*s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseOptionalPower(s *string) (*float64, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	v, err := parsePower(*s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
