// testdb prepares only an explicitly named, local integration-test database.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"oneflow/app-backend/internal/db"
	"oneflow/app-backend/internal/seed"
)

func main() {
	dsn := os.Getenv("APP_BACKEND_TEST_DSN")
	if dsn == "" {
		log.Fatal("APP_BACKEND_TEST_DSN is required")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("invalid test database configuration")
	}
	if cfg.ConnConfig.Database != "oneflow_test" || (cfg.ConnConfig.Host != "127.0.0.1" && cfg.ConnConfig.Host != "localhost" && cfg.ConnConfig.Host != "::1") {
		log.Fatal("testdb requires localhost and database oneflow_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, "../../db/migrations"); err != nil {
		log.Fatal(err)
	}
	if err := os.Setenv("ENABLE_DEMO_ACCOUNTS", "true"); err != nil {
		log.Fatal(err)
	}
	if err := seed.Bootstrap(ctx, pool); err != nil {
		log.Fatal(err)
	}
	log.Print("integration test database ready")
}
