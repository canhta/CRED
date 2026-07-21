package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/canhta/cred/internal/store"
)

// MigrationResult describes what a migration run did.
type MigrationResult struct {
	FromVersion int64
	ToVersion   int64
	Applied     int
}

// Migrate applies every pending migration.
//
// Migration failure is not atomic. goose reports a *goose.PartialError
// carrying the migrations that succeeded before the failure, and treating a
// failure as all-or-nothing is a real bug rather than a style question — the
// database is left at an intermediate version and the operator needs to know
// which one.
func (s *Store) Migrate(ctx context.Context) (MigrationResult, error) {
	db := stdlib.OpenDBFromPool(s.pool)
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(store.Migrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return MigrationResult{}, fmt.Errorf("goose dialect: %w", err)
	}

	from, err := currentVersion(ctx, db)
	if err != nil {
		return MigrationResult{}, err
	}

	if uperr := goose.UpContext(ctx, db, store.MigrationsDir); uperr != nil {
		var partial *goose.PartialError
		if errors.As(uperr, &partial) {
			at, verr := currentVersion(ctx, db)
			if verr != nil {
				at = from
			}
			return MigrationResult{
					FromVersion: from,
					ToVersion:   at,
					Applied:     len(partial.Applied),
				}, fmt.Errorf(
					"migration %d failed after %d applied; database is at version %d: %w",
					partial.Failed.Source.Version, len(partial.Applied), at, translate(partial.Err))
		}
		return MigrationResult{FromVersion: from}, translate(uperr)
	}

	to, err := currentVersion(ctx, db)
	if err != nil {
		return MigrationResult{}, err
	}
	return MigrationResult{FromVersion: from, ToVersion: to}, nil
}

// SchemaVersion reports the applied migration version, or 0 if the migration
// table does not exist yet.
func (s *Store) SchemaVersion(ctx context.Context) (int64, error) {
	db := stdlib.OpenDBFromPool(s.pool)
	defer func() { _ = db.Close() }()
	return currentVersion(ctx, db)
}

// PendingMigrations reports how many migrations have not been applied.
func (s *Store) PendingMigrations(ctx context.Context) (int, error) {
	db := stdlib.OpenDBFromPool(s.pool)
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(store.Migrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return 0, fmt.Errorf("goose dialect: %w", err)
	}
	pending, err := goose.CollectMigrations(store.MigrationsDir, 0, goose.MaxVersion)
	if err != nil {
		return 0, fmt.Errorf("collect migrations: %w", err)
	}
	at, err := currentVersion(ctx, db)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, m := range pending {
		if m.Version > at {
			n++
		}
	}
	return n, nil
}

func currentVersion(ctx context.Context, db *sql.DB) (int64, error) {
	goose.SetBaseFS(store.Migrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return 0, fmt.Errorf("goose dialect: %w", err)
	}
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists)
	if err != nil {
		return 0, translate(err)
	}
	if !exists {
		return 0, nil
	}
	v, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return 0, translate(err)
	}
	return v, nil
}
