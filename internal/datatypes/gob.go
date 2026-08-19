package datatypes

import (
	"bytes"
	"encoding/gob"
)

func gobEncode(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gobDecode[T any](b []byte) (T, error) {
	var v T
	if len(b) == 0 {
		return v, nil
	}
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&v)
	return v, err
}

func gobEncodeIfPresent[T any](v *T) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return gobEncode(*v)
}

func gobEncodeSlice[T any](v []T) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return gobEncode(v)
}
