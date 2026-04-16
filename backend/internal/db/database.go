package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"devcost-ai/internal/config"
	"devcost-ai/pkg/logger"
)

// Database interface defines all database operations
type Database interface {
	Health(ctx context.Context) error
	Close(ctx context.Context) error
	GetPool() *pgxpool.Pool
	GetSQLDB() *sql.DB
	Stats() *pgxpool.Stat
	LogStats()
}

// Client represents the database client
type Client struct {
	postgres Database
	logger   *logger.Logger
	config   *config.DatabaseConfig
}

// NewClient creates a new database client
func NewClient(cfg *config.DatabaseConfig, log *logger.Logger) (*Client, error) {
	log.Info("Initializing database client")

	// Create PostgreSQL connection
	postgres, err := NewPostgreSQL(cfg, log)
	if err != nil {
		log.Error("Failed to initialize PostgreSQL connection", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	client := &Client{
		postgres: postgres,
		logger:   log,
		config:   cfg,
	}

	log.Info("Database client initialized successfully")
	return client, nil
}

// Initialize performs database initialization tasks
func (c *Client) Initialize(ctx context.Context) error {
	c.logger.Info("Initializing database schema and data")

	// Test database connection first
	if err := c.Health(ctx); err != nil {
		c.logger.Error("Database health check failed during initialization", zap.Error(err))
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Run database migrations
	if err := c.runMigrations(ctx); err != nil {
		c.logger.Error("Failed to run database migrations", zap.Error(err))
		return fmt.Errorf("database migrations failed: %w", err)
	}

	// Validate schema
	if err := c.validateSchema(ctx); err != nil {
		c.logger.Error("Schema validation failed", zap.Error(err))
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Create initial data if needed
	if err := c.seedInitialData(ctx); err != nil {
		c.logger.Error("Failed to seed initial data", zap.Error(err))
		return fmt.Errorf("failed to seed initial data: %w", err)
	}

	c.logger.Info("Database initialization completed successfully")
	return nil
}

// Health checks database connection
func (c *Client) Health(ctx context.Context) error {
	return c.postgres.Health(ctx)
}

// Close closes the database connection
func (c *Client) Close(ctx context.Context) error {
	c.logger.Info("Closing database client")
	return c.postgres.Close(ctx)
}

// GetPool returns the pgx connection pool
func (c *Client) GetPool() *pgxpool.Pool {
	return c.postgres.GetPool()
}

// GetSQLDB returns a standard database/sql DB for compatibility
func (c *Client) GetSQLDB() *sql.DB {
	return c.postgres.GetSQLDB()
}

// Stats returns connection pool statistics
func (c *Client) Stats() *pgxpool.Stat {
	return c.postgres.Stats()
}

// LogStats logs current connection pool statistics
func (c *Client) LogStats() {
	c.postgres.LogStats()
}

// runMigrations runs database migrations using golang-migrate
func (c *Client) runMigrations(ctx context.Context) error {
	c.logger.Info("Running database migrations")

	// Get SQL database for migrations
	sqlDB := c.GetSQLDB()
	if sqlDB == nil {
		return fmt.Errorf("failed to get SQL database for migrations")
	}

	// Create migrator instance
	migrator := NewMigrator(sqlDB, c.logger)

	// Validate migration files
	migrationsPath := "migrations"
	if err := migrator.ValidateMigrations(migrationsPath); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
	}

	// Run migrations
	if err := migrator.RunMigrations(ctx, migrationsPath); err != nil {
		return fmt.Errorf("migration execution failed: %w", err)
	}

	// Get migration statistics
	stats, err := migrator.GetStats(migrationsPath)
	if err != nil {
		c.logger.Warn("Failed to get migration statistics", zap.Error(err))
	} else {
		c.logger.Info("Migration statistics",
			zap.Int64("current_version", stats.CurrentVersion),
			zap.Bool("is_dirty", stats.IsDirty),
			zap.Int("available_count", stats.Available),
			zap.Bool("needs_update", stats.NeedsUpdate),
		)
	}

	return nil
}

// seedInitialData seeds the database with initial data
func (c *Client) seedInitialData(ctx context.Context) error {
	c.logger.Info("Seeding initial data")

	// Check if we already have users
	pool := c.GetPool()
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing users: %w", err)
	}

	if count > 0 {
		c.logger.Info("Database already contains data, skipping seeding")
		return nil
	}

	// Skip admin user creation in demo mode - admin can be created via API or CLI
	// In production, use proper user management tools
	c.logger.Info("Initial data seeded successfully (no default admin user created)")
	return nil
}

// StartStatsLogging starts a goroutine to log connection pool statistics periodically
func (c *Client) StartStatsLogging(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.LogStats()
		}
	}
}

// validateSchema validates that all required tables and indexes exist
func (c *Client) validateSchema(ctx context.Context) error {
	c.logger.Info("Validating database schema")

	requiredTables := []string{
		"resources",
		"cost_data",
		"resource_usage",
		"recommendations",
		"actions",
		"users",
	}

	pool := c.GetPool()
	var missing []string

	for _, table := range requiredTables {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)
		`, table).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}

		if exists {
			c.logger.Info("Table exists", zap.String("table", table))
		} else {
			missing = append(missing, table)
			c.logger.Warn("Table missing", zap.String("table", table))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tables: %v", missing)
	}

	// Check indexes on critical tables
	criticalIndexes := map[string][]string{
		"resources": {"resources_resource_id_idx", "resources_resource_type_idx"},
		"recommendations": {"recommendations_resource_id_idx", "recommendations_status_idx"},
		"actions": {"actions_resource_id_idx", "actions_status_idx"},
	}

	for table, indexes := range criticalIndexes {
		for _, idx := range indexes {
			var exists bool
			err := pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT FROM pg_indexes 
					WHERE tablename = $1 
					AND indexname = $2
				)
			`, table, idx).Scan(&exists)

			if err != nil {
				c.logger.Warn("Failed to check index", 
					zap.String("table", table),
					zap.String("index", idx),
					zap.Error(err))
				continue
			}

			if exists {
				c.logger.Debug("Index exists", zap.String("index", idx))
			} else {
				c.logger.Warn("Index missing (non-critical)", zap.String("index", idx))
			}
		}
	}

	c.logger.Info("Schema validation completed",
		zap.Int("tables_checked", len(requiredTables)),
		zap.Int("tables_found", len(requiredTables)-len(missing)))

	return nil
}
