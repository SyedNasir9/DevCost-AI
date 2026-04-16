package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Postgres driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // File source
	"go.uber.org/zap"

	"devcost-ai/pkg/logger"
)

// Migrator handles database migrations
type Migrator struct {
	db     *sql.DB
	logger *logger.Logger
	m      *migrate.Migrate
}

// NewMigrator creates a new migrator instance
func NewMigrator(database *sql.DB, log *logger.Logger) *Migrator {
	return &Migrator{
		db:     database,
		logger: log,
		m:      nil, // Will be initialized in RunMigrations
	}
}

// RunMigrations runs all pending migrations
func (m *Migrator) RunMigrations(ctx context.Context, migrationsPath string) error {
	m.logger.Info("Starting database migrations",
		zap.String("migrations_path", migrationsPath),
	)

	// Create migration instance
	instance, err := migrate.New(
		migrate.WithDatabaseInstance(m.db),
		migrate.WithSourceInstance(migrate.WithFileSystem(os.DirFS(migrationsPath))),
	)
	if err != nil {
		m.logger.Error("Failed to create migration instance", zap.Error(err))
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Store the instance for later use
	m.m = instance

	// Get current migration version
	version, dirty, err := m.m.Version()
	if err != nil {
		m.logger.Error("Failed to get migration version", zap.Error(err))
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	m.logger.Info("Current migration state",
		zap.Int64("version", version),
		zap.Bool("dirty", dirty),
	)

	if dirty {
		m.logger.Error("Database is in dirty state, manual intervention required")
		return fmt.Errorf("database migration state is dirty, version: %d", version)
	}

	// Run migrations with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := m.m.Up(); err != nil {
		if err.Error() == "no change" {
			m.logger.Info("No new migrations to apply")
			return nil
		}

		m.logger.Error("Failed to run migrations", zap.Error(err))
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get final version after migration
	finalVersion, _, err := m.m.Version()
	if err != nil {
		m.logger.Error("Failed to get final migration version", zap.Error(err))
		return fmt.Errorf("failed to get final migration version: %w", err)
	}

	m.logger.Info("Migrations completed successfully",
		zap.Int64("final_version", finalVersion),
	)

	return nil
}

// GetMigrationVersion returns the current migration version
func (m *Migrator) GetMigrationVersion() (int64, bool, error) {
	m, err := migrate.New(
		migrate.WithDatabaseInstance(m.db),
		migrate.WithSourceInstance(migrate.WithFileSystem(fs.OSS{}, "migrations")),
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration instance: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// CreateMigrationTable creates the schema_migrations table if it doesn't exist
func (m *Migrator) CreateMigrationTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		dirty BOOLEAN NOT NULL,
		applied_at TIMESTAMP WITH TIME ZONE
	);`

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	return nil
}

// ListMigrations returns all available migration files
func (m *Migrator) ListMigrations(migrationsPath string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.up.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read migration files: %w", err)
	}

	return files, nil
}

// ValidateMigrations checks if migration files are valid
func (m *Migrator) ValidateMigrations(migrationsPath string) error {
	files, err := m.ListMigrations(migrationsPath)
	if err != nil {
		return err
	}

	m.logger.Info("Found migration files",
		zap.Strings("files", files),
		zap.Int("count", len(files)),
	)

	if len(files) == 0 {
		m.logger.Warn("No migration files found")
		return nil
	}

	return nil
}

// MigrationStats provides migration statistics
type MigrationStats struct {
	CurrentVersion int64 `json:"current_version"`
	IsDirty        bool  `json:"is_dirty"`
	Available      int   `json:"available_count"`
	NeedsUpdate    bool  `json:"needs_update"`
}

// GetStats returns migration statistics
func (m *Migrator) GetStats(migrationsPath string) (*MigrationStats, error) {
	stats := &MigrationStats{}

	// Get current version
	version, dirty, err := m.GetMigrationVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	stats.CurrentVersion = version
	stats.IsDirty = dirty

	// Count available migrations
	files, err := m.ListMigrations(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list migrations: %w", err)
	}

	stats.Available = len(files)
	stats.NeedsUpdate = len(files) > int(version)

	return stats, nil
}
