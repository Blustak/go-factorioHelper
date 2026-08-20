package datatypes

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Display turns a Factorio LocalisedString (from dump localised_name) into a
// single English label without locale .cfg files. Missing or empty input falls
// back to Humanize(fallback).
func Display(localised json.RawMessage, fallback string) string {
	if s := resolveRaw(localised); s != "" {
		return s
	}
	return Humanize(fallback)
}

// Humanize turns a prototype ID or locale key into a display label: the
// segment after the last ".", hyphens/underscores to spaces, first letter
// capitalized (iron-plate → "Iron plate", item-name.iron-plate → "Iron plate").
func Humanize(s string) string {
	if s == "" {
		return ""
	}
	if i := strings.LastIndex(s, "."); i >= 0 && i < len(s)-1 {
		s = s[i+1:]
	}
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

func parseLocalisedName(b []byte) json.RawMessage {
	var raw struct {
		LocalisedName json.RawMessage `json:"localised_name"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil
	}
	msg := bytes.TrimSpace(raw.LocalisedName)
	if len(msg) == 0 || bytes.Equal(msg, []byte("null")) {
		return nil
	}
	return append(json.RawMessage(nil), msg...)
}

func newEntity(name, typ string, order *string, dump []byte) Entity {
	return Entity{
		Name:         name,
		Type:         typ,
		EntityOrder:  order,
		localisedRaw: parseLocalisedName(dump),
	}
}

func resolveRaw(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return resolveValue(v)
}

func resolveValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		if looksLikeKeyOrID(val) {
			return Humanize(val)
		}
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'g', -1, 64)
	case json.Number:
		return val.String()
	case bool:
		return strconv.FormatBool(val)
	case []any:
		if len(val) == 0 {
			return ""
		}
		first, ok := val[0].(string)
		if !ok {
			return resolveValue(val[0])
		}
		switch first {
		case "?":
			for _, alt := range val[1:] {
				if s := resolveValue(alt); s != "" {
					return s
				}
			}
			return ""
		case "":
			var b strings.Builder
			for _, p := range val[1:] {
				b.WriteString(resolveValue(p))
			}
			return b.String()
		default:
			head := Humanize(first)
			var args []string
			for _, p := range val[1:] {
				if a := resolveValue(p); a != "" {
					args = append(args, a)
				}
			}
			if len(args) == 0 {
				return head
			}
			return head + " (" + strings.Join(args, ", ") + ")"
		}
	default:
		return ""
	}
}

func looksLikeKeyOrID(s string) bool {
	if s == "" {
		return false
	}
	if strings.Contains(s, ".") {
		return true
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
