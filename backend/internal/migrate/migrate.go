package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var migrationsFS embed.FS

func Run(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		var exists bool
		db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)`, name).Scan(&exists)
		if exists {
			continue
		}

		sql, err := migrationsFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		if _, err := db.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}

		db.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name)
		log.Printf("migration applied: %s", name)
	}

	return nil
}
