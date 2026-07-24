package migrations

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed *.up.sql
var files embed.FS

type DB interface {
	Begin(context.Context) (pgx.Tx, error)
}

func Up(ctx context.Context, db DB) error {
	entries, err := fs.Glob(files, "*.up.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, name := range entries {
		version, err := strconv.ParseInt(strings.SplitN(name, "_", 2)[0], 10, 64)
		if err != nil {
			return err
		}
		sql, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		if err := apply(ctx, db, version, string(sql)); err != nil {
			return err
		}
	}
	return nil
}

func apply(ctx context.Context, db DB, version int64, migration string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version bigint PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(0x4f70656e54756e)); err != nil {
		return err
	}
	var applied int64
	err = tx.QueryRow(ctx, "SELECT version FROM schema_migrations WHERE version = $1", version).Scan(&applied)
	if err == nil {
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, err := tx.Exec(ctx, migration); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES ($1)", version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
