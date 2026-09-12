package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	backendassets "github.com/mistysya/hackathon_2026/backend"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"modernc.org/sqlite"
	_ "modernc.org/sqlite"
)

const DefaultDSN = "file:/data/app.db?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

type Repository struct {
	db *sql.DB
}

var (
	_ ports.EmployeeRepository = (*Repository)(nil)
	_ ports.CampaignRepository = (*Repository)(nil)
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, backendassets.SchemaSQL); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}
	return nil
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var sqliteError *sqlite.Error
	if errors.As(err, &sqliteError) && sqliteError.Code()&0xff == 19 {
		return fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return err
}
