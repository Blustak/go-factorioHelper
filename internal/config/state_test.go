package config

import (
	"context"
	"strings"
	"testing"
)

func TestOpenEmptyDBString(t *testing.T) {
	_, err := Open(context.Background(), "")
	if err == nil {
		t.Fatal("Open error = nil, want GOOSE_DBSTRING is not set")
	}
	if !strings.Contains(err.Error(), "GOOSE_DBSTRING is not set") {
		t.Fatalf("Open error = %v, want GOOSE_DBSTRING is not set", err)
	}
}

func TestSqliteDSN(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{in: "data/data.db", want: "file:data/data.db?_foreign_keys=1"},
		{in: "file:data/data.db", want: "file:data/data.db?_foreign_keys=1"},
		{in: "file:data/data.db?cache=shared", want: "file:data/data.db?cache=shared&_foreign_keys=1"},
		{in: "file:data/data.db?_foreign_keys=1", want: "file:data/data.db?_foreign_keys=1"},
	}
	for _, tt := range tests {
		if got := sqliteDSN(tt.in); got != tt.want {
			t.Errorf("sqliteDSN(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
