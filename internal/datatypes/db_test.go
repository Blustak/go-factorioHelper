package datatypes

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
	"github.com/Blustak/go-factorioHelper/sql/schema"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

func testState(t *testing.T) *config.State {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, schema.Migrations)
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	return &config.State{
		DB:  database.New(db),
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		CTX: ctx,
	}
}
