package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/latoulicious/Tsugi/migrations"
)

// ponytail: minimal forward/last-step runner — no dirty-state or down-to-version.
const ensureMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version    TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`

// MigrateUp applies every pending *.up.sql in lexical order, each in its own tx.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, ensureMigrationsTable); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}
	files, err := fs.Glob(migrations.FS, "*.up.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)
	for _, name := range files {
		ver := strings.TrimSuffix(name, ".up.sql")
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE version = $1`, ver).Scan(&n); err != nil {
			return fmt.Errorf("check %s: %w", ver, err)
		}
		if n > 0 {
			continue
		}
		if err := apply(ctx, pool, name, ver, `INSERT INTO schema_migrations (version) VALUES ($1)`); err != nil {
			return err
		}
	}
	return nil
}

// MigrateDown rolls back the most recently applied migration (one step).
func MigrateDown(ctx context.Context, pool *pgxpool.Pool) error {
	var ver string
	err := pool.QueryRow(ctx, `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&ver)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("last migration: %w", err)
	}
	return apply(ctx, pool, ver+".down.sql", ver, `DELETE FROM schema_migrations WHERE version = $1`)
}

// apply runs a migration file and records/removes its version in one tx.
func apply(ctx context.Context, pool *pgxpool.Pool, name, ver, ledger string) error {
	body, err := migrations.FS.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", ver, err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, string(body)); err != nil {
		return fmt.Errorf("apply %s: %w", ver, err)
	}
	if _, err := tx.Exec(ctx, ledger, ver); err != nil {
		return fmt.Errorf("record %s: %w", ver, err)
	}
	return tx.Commit(ctx)
}
