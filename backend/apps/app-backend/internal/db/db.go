package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return err
	}

	var appliedCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&appliedCount); err != nil {
		return err
	}
	if appliedCount == 0 {
		legacyComplete, err := hasLegacyCompleteSchema(ctx, pool)
		if err != nil {
			return err
		}
		if legacyComplete {
			for _, file := range files {
				if _, err := pool.Exec(ctx, `
					INSERT INTO schema_migrations (version)
					VALUES ($1)
					ON CONFLICT (version) DO NOTHING
				`, filepath.Base(file)); err != nil {
					return err
				}
			}
			return nil
		}
	}

	for _, file := range files {
		version := filepath.Base(file)
		var alreadyApplied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&alreadyApplied); err != nil {
			return err
		}
		if alreadyApplied {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("migrate %s: %w", version, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return err
		}
	}

	return nil
}

func hasLegacyCompleteSchema(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	requiredTables := []string{
		"organizations",
		"organization_members",
		"organization_invites",
		"whatsapp_sessions",
		"subscription_plans",
		"audit_logs",
	}
	for _, table := range requiredTables {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists); err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}
	return true, nil
}
