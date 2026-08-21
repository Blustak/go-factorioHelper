package config

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/Blustak/go-factorioHelper/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

type State struct {
	DB     *database.Queries
	Log    *slog.Logger
	CTX    context.Context
	Writer *bufio.Writer

	dbFile  *sql.DB
	logFile *os.File
}

func New(ctx context.Context, db *sql.DB, log *slog.Logger, w io.Writer) *State {
	if ctx == nil {
		ctx = context.Background()
	}
	if log == nil {
		log = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if w == nil {
		w = os.Stdout
	}
	return &State{
		DB:     database.New(db),
		Log:    log,
		CTX:    ctx,
		Writer: bufio.NewWriter(w),
		dbFile: db,
	}
}

func Open(ctx context.Context, dbString string) (*State, error) {
	if dbString == "" {
		return nil, fmt.Errorf("GOOSE_DBSTRING is not set")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	db, err := sql.Open("sqlite3", sqliteDSN(dbString))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return New(ctx, db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil), nil
}

func (s *State) Close() error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.Writer != nil {
		errs = append(errs, s.Writer.Flush())
	}
	if s.logFile != nil {
		errs = append(errs, s.logFile.Close())
		s.logFile = nil
	}
	if s.dbFile != nil {
		errs = append(errs, s.dbFile.Close())
		s.dbFile = nil
	}
	return errors.Join(errs...)
}

func (s *State) RunInTx(fn func(*State) error) error {
	if s == nil || s.dbFile == nil {
		return fmt.Errorf("database handle is not set")
	}

	tx, err := s.dbFile.BeginTx(s.CTX, nil)
	if err != nil {
		return err
	}

	txState := *s
	txState.DB = s.DB.WithTx(tx)
	if err := fn(&txState); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func sqliteDSN(dbString string) string {
	if strings.Contains(dbString, "_foreign_keys=") {
		return dbString
	}
	if strings.HasPrefix(dbString, "file:") {
		if strings.Contains(dbString, "?") {
			return dbString + "&_foreign_keys=1"
		}
		return dbString + "?_foreign_keys=1"
	}
	return "file:" + dbString + "?_foreign_keys=1"
}
