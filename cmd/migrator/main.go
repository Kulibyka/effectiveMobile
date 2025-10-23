package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Kulibyka/effective-mobile/internal/config"
	"github.com/Kulibyka/effective-mobile/internal/logger"
	"github.com/Kulibyka/effective-mobile/internal/storage/postgresql"
)

const (
	migrationsTable           = "schema_migrations"
	defaultMigrationsPath     = "./migrations"
	migrationStatementTimeout = 30 * time.Second
)

func main() {
	cfg := config.MustLoad()

	log := logger.New(cfg.Env)
	log.Info("starting migrator", slog.String("env", cfg.Env))

	storage, err := postgresql.New(cfg.PostgreSQL)
	if err != nil {
		log.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			log.Warn("failed to close database connection", slog.Any("error", err))
		}
	}()

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = defaultMigrationsPath
	}

	if err := runMigrations(storage.GetDB(), migrationsPath, log); err != nil {
		log.Error("migration failed", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("migrations applied successfully")
}

func runMigrations(db *sql.DB, migrationsPath string, log *slog.Logger) error {
	info, err := os.Stat(migrationsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("migrations directory does not exist: %s", migrationsPath)
		}

		return fmt.Errorf("failed to access migrations directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("migrations path is not a directory: %s", migrationsPath)
	}

	ctx := context.Background()

	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, filepath.Join(migrationsPath, name))
		}
	}

	sort.Strings(files)

	applied, err := loadAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".up.sql")
		if _, ok := applied[version]; ok {
			log.Info("migration already applied", slog.String("version", version))
			continue
		}

		contents, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		log.Info("applying migration", slog.String("version", version), slog.String("file", file))

		if err := execMigration(ctx, db, string(contents)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}

		if err := markMigrationApplied(ctx, db, version); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	execCtx, cancel := context.WithTimeout(ctx, migrationStatementTimeout)
	defer cancel()

	const query = `CREATE TABLE IF NOT EXISTS ` + migrationsTable + ` (
        version TEXT PRIMARY KEY,
        applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

	if _, err := db.ExecContext(execCtx, query); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	return nil
}

func loadAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	queryCtx, cancel := context.WithTimeout(ctx, migrationStatementTimeout)
	defer cancel()

	rows, err := db.QueryContext(queryCtx, "SELECT version FROM "+migrationsTable)
	if err != nil {
		return nil, fmt.Errorf("failed to load applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan applied migration: %w", err)
		}

		applied[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate applied migrations: %w", err)
	}

	return applied, nil
}

func markMigrationApplied(ctx context.Context, db *sql.DB, version string) error {
	execCtx, cancel := context.WithTimeout(ctx, migrationStatementTimeout)
	defer cancel()

	const query = "INSERT INTO " + migrationsTable + " (version) VALUES ($1)"

	if _, err := db.ExecContext(execCtx, query, version); err != nil {
		return fmt.Errorf("failed to mark migration %s as applied: %w", version, err)
	}

	return nil
}

func execMigration(ctx context.Context, db *sql.DB, statement string) error {
	execCtx, cancel := context.WithTimeout(ctx, migrationStatementTimeout)
	defer cancel()

	if _, err := db.ExecContext(execCtx, statement); err != nil {
		return err
	}

	return nil
}
