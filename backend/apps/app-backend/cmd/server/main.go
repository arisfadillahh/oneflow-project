package main

import (
	"context"
	"log"
	"time"
	_ "time/tzdata" // Minimal runtime images do not include system timezone data.

	"oneflow/app-backend/internal/config"
	"oneflow/app-backend/internal/db"
	"oneflow/app-backend/internal/httpapi"
	"oneflow/app-backend/internal/seed"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pool, err := db.Open(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "/db/migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := seed.Bootstrap(ctx, pool); err != nil {
		log.Fatalf("seed: %v", err)
	}

	log.Printf("app-backend listening on :%s", cfg.AppPort)
	if err := httpapi.ListenAndServe(cfg, pool); err != nil {
		log.Fatal(err)
	}
}
