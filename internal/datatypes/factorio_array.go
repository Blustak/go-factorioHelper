package datatypes

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// unmarshalFactorioArray treats a Lua empty table encoded as JSON {} as an empty slice.
func unmarshalFactorioArray[T any](raw json.RawMessage) ([]T, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	if raw[0] == '{' {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, err
		}
		if len(obj) == 0 {
			return []T{}, nil
		}
		return nil, fmt.Errorf("expected array or empty object, got object with %d keys", len(obj))
	}
	var out []T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
