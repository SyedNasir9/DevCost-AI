package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"devcost-ai/internal/config"
	"devcost-ai/internal/db"
	"devcost-ai/pkg/logger"
)

var (
	action     = flag.String("action", "", "Migration action: up, down, version, force")
	migrations = flag.String("path", "migrations", "Path to migration files")
	steps      = flag.Int("steps", 0, "Number of migration steps to apply")
	database   = flag.String("database", "", "Database connection string")
)

func main() {
	flag.Parse()

	if *database == "" {
		// Load configuration from environment if no database string provided
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
		*database = cfg.Database.DatabaseURL()
	}

	// Initialize logger
	zapConfig, err := logger.NewProductionConfig()
	if err != nil {
		log.Fatalf("Failed to create logger config: %v", err)
	}

	zapLogger, err := logger.NewLogger(zapConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync()

	zapLogger.Info("Starting migration tool",
		zap.String("action", *action),
		zap.String("migrations_path", *migrations),
		zap.String("database", *database),
	)

	// Run migration command
	if err := runMigration(*action, *migrations, *database, *steps, zapLogger); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	zapLogger.Info("Migration completed successfully")
}

// runMigration executes the specified migration action
func runMigration(action, migrationsPath, database string, steps int, log *logger.Logger) error {
	// Create database connection
	db, err := sql.Open("postgres", database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create migration instance
	m, err := migrate.New(
		migrate.WithDatabaseInstance(db),
		migrate.WithSourceInstance(migrate.WithFileSystem(os.DirFS(migrationsPath))),
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	switch action {
	case "up":
		return runUpMigration(ctx, m, steps, log)
	case "down":
		return runDownMigration(ctx, m, steps, log)
	case "version":
		return showMigrationVersion(ctx, m, log)
	case "force":
		return forceMigration(ctx, m, steps, log)
	default:
		return fmt.Errorf("unknown action: %s (use: up, down, version, force)", action)
	}
}

// runUpMigration applies pending migrations
func runUpMigration(ctx context.Context, m *migrate.Migrate, steps int, log *logger.Logger) error {
	log.Info("Running up migrations")
	
	if steps > 0 {
		if err := m.Steps(steps); err != nil {
			return fmt.Errorf("failed to run %d migration steps: %w", steps, err)
		}
		log.Info("Successfully applied migration steps", zap.Int("steps", steps))
	} else {
		if err := m.Up(); err != nil {
			if err.Error() == "no change" {
				log.Info("No new migrations to apply")
				return nil
			}
			return fmt.Errorf("failed to run up migrations: %w", err)
		}
		log.Info("All pending migrations applied successfully")
	}

	return nil
}

// runDownMigration rolls back migrations
func runDownMigration(ctx context.Context, m *migrate.Migrate, steps int, log *logger.Logger) error {
	log.Info("Running down migrations")
	
	if steps > 0 {
		if err := m.Steps(-steps); err != nil {
			return fmt.Errorf("failed to rollback %d migration steps: %w", steps, err)
		}
		log.Info("Successfully rolled back migration steps", zap.Int("steps", steps))
	} else {
		if err := m.Down(); err != nil {
			if err.Error() == "no change" {
				log.Info("No migrations to rollback")
				return nil
			}
			return fmt.Errorf("failed to run down migrations: %w", err)
		}
		log.Info("All migrations rolled back successfully")
	}

	return nil
}

// showMigrationVersion displays current migration version
func showMigrationVersion(ctx context.Context, m *migrate.Migrate, log *logger.Logger) error {
	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	log.Info("Migration version information",
		zap.Int64("version", version),
		zap.Bool("dirty", dirty),
	)

	if dirty {
		log.Error("Database is in dirty state - manual intervention required")
		return fmt.Errorf("database migration state is dirty, version: %d", version)
	}

	return nil
}

// forceMigration forces a specific migration version
func forceMigration(ctx context.Context, m *migrate.Migrate, steps int, log *logger.Logger) error {
	log.Info("Forcing migration version", zap.Int("version", steps))
	
	if err := m.Force(int64(steps)); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	
	log.Info("Migration version forced successfully", zap.Int("version", steps))
	return nil
}
