package postgres

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	// Use DATABASE_URL if set, otherwise build from individual vars.
	// Individual vars avoid URL special-character encoding issues.
	url := os.Getenv("DATABASE_URL")
	log.Printf("db: DATABASE_URL set=%v DB_HOST=%q", url != "", os.Getenv("DB_HOST"))
	if url == "" {
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASSWORD")
		name := os.Getenv("DB_NAME")
		if port == "" {
			port = "5432"
		}
		cfg, err := pgxpool.ParseConfig("")
		if err != nil {
			return nil, err
		}
		cfg.ConnConfig.Host = host
		cfg.ConnConfig.Port = func() uint16 {
			var p uint16 = 5432
			fmt.Sscanf(port, "%d", &p)
			return p
		}()
		cfg.ConnConfig.User = user
		cfg.ConnConfig.Password = pass
		cfg.ConnConfig.Database = name
		return pgxpool.NewWithConfig(ctx, cfg)
	}
	return pgxpool.New(ctx, url)
}
