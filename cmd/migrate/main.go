package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/config"
	"github.com/rs/zerolog"
)

func main() {
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Host,
		cfg.Postgres.Port, cfg.Postgres.DB, cfg.Postgres.SSLMode,
	)

	log.Info().Str("host", cfg.Postgres.Host).Str("db", cfg.Postgres.DB).Msg("Connecting to PostgreSQL...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect to PostgreSQL. Is PostgreSQL running on localhost:5432 with news_db?")
	}
	defer conn.Close(ctx)

	log.Info().Msg("✓ Connected to PostgreSQL database")

	// Find all .up.sql files in migrations directory
	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = filepath.Join("..", "migrations")
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read migrations directory")
	}

	var upFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	// Create schema_migrations table if not exists
	_, _ = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT NOW())`)

	var tablesExist bool
	_ = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='categories')`).Scan(&tablesExist)
	if tablesExist {
		_, _ = conn.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ('001_tenants.up.sql'), ('002_users_auth.up.sql'), ('003_iam_rbac.up.sql'), ('004_content.up.sql'), ('005_rls_policies.up.sql'), ('006_ads_seo.up.sql') ON CONFLICT DO NOTHING`)
	}

	for _, filename := range upFiles {
		var alreadyApplied bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", filename).Scan(&alreadyApplied)

		if alreadyApplied {
			log.Info().Str("file", filename).Msg("• Already applied, skipping")
			continue
		}

		filePath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatal().Err(err).Str("file", filename).Msg("Failed to read migration file")
		}

		log.Info().Str("file", filename).Msg("Running migration...")
		_, err = conn.Exec(ctx, string(content))
		if err != nil {
			log.Fatal().Err(err).Str("file", filename).Msg("Migration failed")
		}

		_, _ = conn.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING", filename)
		log.Info().Str("file", filename).Msg("✓ Applied successfully")
	}

	// Verify categories, users, roles count
	var categoryCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM categories").Scan(&categoryCount)

	var userCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)

	var roleCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM roles").Scan(&roleCount)

	log.Info().
		Int("categories", categoryCount).
		Int("users", userCount).
		Int("roles", roleCount).
		Msg("🎉 Database migrations and seeding completed successfully!")
}
