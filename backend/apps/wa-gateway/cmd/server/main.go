package main

import (
	"context"
	"log"
	"time"

	"oneflow/wa-gateway/internal/config"
	"oneflow/wa-gateway/internal/httpapi"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer pool.Close()

	log.Printf("wa-gateway listening on :%s", cfg.Port)
	if err := httpapi.ListenAndServe(cfg, pool); err != nil {
		log.Fatal(err)
	}
}
