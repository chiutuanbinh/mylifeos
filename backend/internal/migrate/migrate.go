package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed *.sql
var migrationsFS embed.FS

// Run applies pending migrations using a direct (non-pooled) connection.
// Supabase pgBouncer (port 6543) runs in transaction mode which blocks DDL;
// we open a plain pgx.Conn using MIGRATE_DATABASE_URL (direct, port 5432)
// falling back to DATABASE_URL if not set.
func Run(ctx context.Context) error {
	connStr := os.Getenv("MIGRATE_DATABASE_URL")
	if connStr == "" {
		connStr = os.Getenv("DATABASE_URL")
	}
	if connStr == "" {
		return fmt.Errorf("MIGRATE_DATABASE_URL or DATABASE_URL required")
	}

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("migrate connect: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.schema_migrations (
			filename   TEXT PRIMARY KEY,
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
		conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.schema_migrations WHERE filename=$1)`, name).Scan(&exists)
		if exists {
			continue
		}

		sql, err := migrationsFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}

		conn.Exec(ctx, `INSERT INTO public.schema_migrations (filename) VALUES ($1)`, name)
		log.Printf("migration applied: %s", name)
	}

	return nil
}
