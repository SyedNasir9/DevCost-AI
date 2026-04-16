package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"devcost-ai/internal/config"
	"devcost-ai/pkg/logger"
)

// PostgreSQL represents a PostgreSQL database connection
type PostgreSQL struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
	config *config.DatabaseConfig
}

// NewPostgreSQL creates a new PostgreSQL connection with connection pooling
func NewPostgreSQL(cfg *config.DatabaseConfig, log *logger.Logger) (*PostgreSQL, error) {
	log.Info("Initializing PostgreSQL connection",
		zap.String("host", cfg.Host),
		zap.String("port", cfg.Port),
		zap.String("database", cfg.Name),
		zap.String("user", cfg.User),
	)

	// Build connection string
	connString := buildConnectionString(cfg)

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Error("Failed to parse connection string", zap.Error(err))
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure pool settings
	configurePool(poolConfig, cfg, log)

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Error("Failed to create connection pool", zap.Error(err))
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := testConnection(ctx, pool, log); err != nil {
		log.Error("Database connection test failed", zap.Error(err))
		return nil, fmt.Errorf("database connection test failed: %w", err)
	}

	log.Info("PostgreSQL connection established successfully")

	return &PostgreSQL{
		pool:   pool,
		logger: log,
		config: cfg,
	}, nil
}

// buildConnectionString builds PostgreSQL connection string
func buildConnectionString(cfg *config.DatabaseConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)
}

// configurePool configures pgx connection pool settings
func configurePool(config *pgxpool.Config, cfg *config.DatabaseConfig, log interface{ Info(string, ...zap.Field) }) {
	// Connection pool settings
	config.MaxConns = 25                       // Maximum number of connections
	config.MinConns = 5                        // Minimum number of connections
	config.MaxConnLifetime = 1 * time.Hour     // Maximum lifetime of a connection
	config.MaxConnIdleTime = 30 * time.Minute  // Maximum idle time for a connection
	config.HealthCheckPeriod = 1 * time.Minute // Health check frequency

	// Configure connection settings
	config.ConnConfig.ConnectTimeout = 10 * time.Second
	config.ConnConfig.RuntimeParams["application_name"] = "devcost-ai"
	config.ConnConfig.RuntimeParams["timezone"] = "UTC"

	// Configure logger for pgx (simplified for now)
	// config.ConnConfig.Tracer = &pgxTracer{logger: log}
}

// testConnection tests database connection
func testConnection(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Ping database
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Test a simple query
	var result int
	err := pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to execute test query: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected test query result: %d", result)
	}

	log.Info("Database connection test passed")
	return nil
}

// GetPool returns pgx connection pool
func (p *PostgreSQL) GetPool() *pgxpool.Pool {
	return p.pool
}

// GetSQLDB returns a standard database/sql DB for compatibility with migrations
func (p *PostgreSQL) GetSQLDB() *sql.DB {
	return stdlib.OpenDBFromPool(p.pool)
}

// Health checks database connection
func (p *PostgreSQL) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.pool.Ping(ctx); err != nil {
		p.logger.Error("Database health check failed", zap.Error(err))
		return fmt.Errorf("database health check failed: %w", err)
	}

	p.logger.Debug("Database health check passed")
	return nil
}

// Close closes the database connection pool
func (p *PostgreSQL) Close(ctx context.Context) error {
	p.logger.Info("Closing PostgreSQL connection pool")

	p.pool.Close()
	return nil
}

// Stats returns connection pool statistics
func (p *PostgreSQL) Stats() *pgxpool.Stat {
	if p.pool == nil {
		// Return zero-value stats when pool is nil
		return &pgxpool.Stat{}
	}
	return p.pool.Stat()
}

// LogStats logs current connection pool statistics
func (p *PostgreSQL) LogStats() {
	stats := p.Stats()
	p.logger.Info("Connection pool statistics",
		zap.Int32("total_conns", stats.TotalConns()),
		zap.Int32("idle_conns", stats.IdleConns()),
		zap.Int32("acquired_conns", stats.AcquiredConns()),
		zap.Int32("max_conns", int32(stats.MaxConns())),
	)
}
