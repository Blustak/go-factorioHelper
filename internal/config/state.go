package config

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/Blustak/go-factorioHelper/internal/database"
)

type State struct{
  DB *database.Queries
  Log *slog.Logger
  CTX context.Context

  dbFile *sql.DB
  logFile *os.File
}
