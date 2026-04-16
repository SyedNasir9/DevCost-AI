package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/pkg/logger"
)

// CostRepository provides data access for cost data
type CostRepository struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewCostRepository creates a new cost repository
func NewCostRepository(pool *pgxpool.Pool, logger *logger.Logger) *CostRepository {
	return &CostRepository{
		pool:   pool,
		logger: logger,
	}
}

// CostDataDB represents cost data as stored in the database
type CostDataDB struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ResourceID   string          `json:"resource_id" db:"resource_id"`
	ResourceUUID *uuid.UUID      `json:"resource_uuid,omitempty" db:"resource_uuid"`
	Service      string          `json:"service" db:"service"`
	CostAmount   float64         `json:"cost_amount" db:"cost_amount"`
	Currency     string          `json:"currency" db:"currency"`
	StartDate    time.Time       `json:"start_date" db:"start_date"`
	EndDate      time.Time       `json:"end_date" db:"end_date"`
	Timestamp    time.Time       `json:"timestamp" db:"timestamp"`
	UsageType    *string         `json:"usage_type,omitempty" db:"usage_type"`
	Region       *string         `json:"region,omitempty" db:"region"`
	AccountID    *string         `json:"account_id,omitempty" db:"account_id"`
	Metadata     json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// SaveCostData saves a single cost data entry with upsert logic
func (r *CostRepository) SaveCostData(ctx context.Context, cost *aws.CostData) error {
	if cost == nil {
		return fmt.Errorf("cost data cannot be nil")
	}

	r.logger.Debug("Saving cost data",
		zap.String("resource_id", cost.ResourceID),
		zap.String("service", cost.Service),
		zap.Float64("cost_amount", cost.CostAmount),
	)

	// Prepare metadata
	metadata := r.prepareMetadata(cost)
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		r.logger.Error("Failed to marshal metadata", zap.Error(err))
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", cost.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", cost.EndDate)
	if err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}

	// Execute upsert query
	query := `
		INSERT INTO cost_data (
			id, resource_id, service, cost_amount, currency,
			start_date, end_date, timestamp, usage_type, region,
			account_id, metadata, created_at, updated_at
		) VALUES (
			uuid_generate_v4(), $1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, NOW(), NOW()
		)
		ON CONFLICT (resource_id, service, start_date, end_date, COALESCE(region, ''))
		DO UPDATE SET
			cost_amount = EXCLUDED.cost_amount,
			currency = EXCLUDED.currency,
			usage_type = EXCLUDED.usage_type,
			account_id = EXCLUDED.account_id,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	var id uuid.UUID
	var createdAt, updatedAt time.Time

	err = r.pool.QueryRow(ctx, query,
		cost.ResourceID,
		cost.Service,
		cost.CostAmount,
		cost.Currency,
		startDate,
		endDate,
		cost.Timestamp,
		r.stringPtr(cost.UsageType),
		r.stringPtr(cost.Region),
		r.stringPtr(cost.AccountID),
		metadataJSON,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		r.logger.Error("Failed to save cost data",
			zap.String("resource_id", cost.ResourceID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save cost data: %w", err)
	}

	r.logger.Debug("Cost data saved successfully",
		zap.String("id", id.String()),
		zap.String("resource_id", cost.ResourceID),
		zap.Time("created_at", createdAt),
	)

	return nil
}

// SaveCostDataBatch saves multiple cost data entries using bulk insert
func (r *CostRepository) SaveCostDataBatch(ctx context.Context, costs []*aws.CostData) error {
	if len(costs) == 0 {
		r.logger.Debug("No cost data to save")
		return nil
	}

	r.logger.Info("Saving cost data batch",
		zap.Int("count", len(costs)),
	)

	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare statement for upsert
	query := `
		INSERT INTO cost_data (
			id, resource_id, service, cost_amount, currency,
			start_date, end_date, timestamp, usage_type, region,
			account_id, metadata, created_at, updated_at
		) VALUES (
			uuid_generate_v4(), $1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, NOW(), NOW()
		)
		ON CONFLICT (resource_id, service, start_date, end_date, COALESCE(region, ''))
		DO UPDATE SET
			cost_amount = EXCLUDED.cost_amount,
			currency = EXCLUDED.currency,
			usage_type = EXCLUDED.usage_type,
			account_id = EXCLUDED.account_id,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`

	// Prepare batch
	batch := &pgx.Batch{}
	
	for _, cost := range costs {
		if cost == nil {
			continue
		}

		// Prepare metadata
		metadata := r.prepareMetadata(cost)
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			r.logger.Warn("Failed to marshal metadata, skipping entry",
				zap.String("resource_id", cost.ResourceID),
				zap.Error(err),
			)
			continue
		}

		// Parse dates
		startDate, err := time.Parse("2006-01-02", cost.StartDate)
		if err != nil {
			r.logger.Warn("Invalid start date, skipping entry",
				zap.String("resource_id", cost.ResourceID),
				zap.Error(err),
			)
			continue
		}

		endDate, err := time.Parse("2006-01-02", cost.EndDate)
		if err != nil {
			r.logger.Warn("Invalid end date, skipping entry",
				zap.String("resource_id", cost.ResourceID),
				zap.Error(err),
			)
			continue
		}

		// Queue batch item
		batch.Queue(query,
			cost.ResourceID,
			cost.Service,
			cost.CostAmount,
			cost.Currency,
			startDate,
			endDate,
			cost.Timestamp,
			r.stringPtr(cost.UsageType),
			r.stringPtr(cost.Region),
			r.stringPtr(cost.AccountID),
			metadataJSON,
		)
	}

	// Execute batch
	batchResults := tx.SendBatch(ctx, batch)
	
	// Process results and collect errors
	var errors []error
	for i := 0; i < batch.Len(); i++ {
		_, err := batchResults.Exec()
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok {
				r.logger.Warn("Failed to save cost data item",
					zap.Int("index", i),
					zap.String("pg_error", pgErr.Message),
					zap.String("detail", pgErr.Detail),
				)
			} else {
				r.logger.Warn("Failed to save cost data item",
					zap.Int("index", i),
					zap.Error(err),
				)
			}
			errors = append(errors, err)
		}
	}

	// Close batch results
	if err := batchResults.Close(); err != nil {
		r.logger.Error("Failed to close batch results", zap.Error(err))
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	successCount := batch.Len() - len(errors)
	r.logger.Info("Cost data batch saved",
		zap.Int("total", batch.Len()),
		zap.Int("success", successCount),
		zap.Int("failed", len(errors)),
	)

	if len(errors) > 0 {
		return fmt.Errorf("failed to save %d out of %d cost data entries", len(errors), batch.Len())
	}

	return nil
}

// GetCostDataByResourceID retrieves cost data for a specific resource
func (r *CostRepository) GetCostDataByResourceID(ctx context.Context, resourceID string) ([]*CostDataDB, error) {
	r.logger.Debug("Fetching cost data by resource ID",
		zap.String("resource_id", resourceID),
	)

	query := `
		SELECT 
			id, resource_id, resource_uuid, service, cost_amount, currency,
			start_date, end_date, timestamp, usage_type, region,
			account_id, metadata, created_at, updated_at
		FROM cost_data
		WHERE resource_id = $1
		ORDER BY timestamp DESC
	`

	rows, err := r.pool.Query(ctx, query, resourceID)
	if err != nil {
		r.logger.Error("Failed to fetch cost data",
			zap.String("resource_id", resourceID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch cost data: %w", err)
	}
	defer rows.Close()

	return r.scanCostDataRows(rows)
}

// GetCostDataByTimeRange retrieves cost data for a time range
func (r *CostRepository) GetCostDataByTimeRange(ctx context.Context, startDate, endDate string) ([]*CostDataDB, error) {
	r.logger.Debug("Fetching cost data by time range",
		zap.String("start_date", startDate),
		zap.String("end_date", endDate),
	)

	query := `
		SELECT 
			id, resource_id, resource_uuid, service, cost_amount, currency,
			start_date, end_date, timestamp, usage_type, region,
			account_id, metadata, created_at, updated_at
		FROM cost_data
		WHERE start_date >= $1 AND end_date <= $2
		ORDER BY timestamp DESC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		r.logger.Error("Failed to fetch cost data",
			zap.String("start_date", startDate),
			zap.String("end_date", endDate),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch cost data: %w", err)
	}
	defer rows.Close()

	return r.scanCostDataRows(rows)
}

// GetCostSummaryByService retrieves cost summary grouped by service
func (r *CostRepository) GetCostSummaryByService(ctx context.Context, startDate, endDate string) (map[string]float64, error) {
	r.logger.Debug("Fetching cost summary by service",
		zap.String("start_date", startDate),
		zap.String("end_date", endDate),
	)

	query := `
		SELECT service, SUM(cost_amount) as total_cost
		FROM cost_data
		WHERE start_date >= $1 AND end_date <= $2
		GROUP BY service
		ORDER BY total_cost DESC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		r.logger.Error("Failed to fetch cost summary",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch cost summary: %w", err)
	}
	defer rows.Close()

	summary := make(map[string]float64)
	for rows.Next() {
		var service string
		var totalCost float64
		if err := rows.Scan(&service, &totalCost); err != nil {
			r.logger.Warn("Failed to scan cost summary row", zap.Error(err))
			continue
		}
		summary[service] = totalCost
	}

	return summary, nil
}

// GetTotalCost retrieves total cost for a time range
func (r *CostRepository) GetTotalCost(ctx context.Context, startDate, endDate string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(cost_amount), 0)
		FROM cost_data
		WHERE start_date >= $1 AND end_date <= $2
	`

	var totalCost float64
	err := r.pool.QueryRow(ctx, query, startDate, endDate).Scan(&totalCost)
	if err != nil {
		r.logger.Error("Failed to fetch total cost", zap.Error(err))
		return 0, fmt.Errorf("failed to fetch total cost: %w", err)
	}

	return totalCost, nil
}

// GetExpensiveResources retrieves resources with high costs
func (r *CostRepository) GetExpensiveResources(ctx context.Context, minCost float64, limit int) ([]*CostDataDB, error) {
	r.logger.Debug("Fetching expensive resources",
		zap.Float64("min_cost", minCost),
		zap.Int("limit", limit),
	)

	query := `
		SELECT 
			id, resource_id, resource_uuid, service, cost_amount, currency,
			start_date, end_date, timestamp, usage_type, region,
			account_id, metadata, created_at, updated_at
		FROM cost_data
		WHERE cost_amount > $1
		ORDER BY cost_amount DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, minCost, limit)
	if err != nil {
		r.logger.Error("Failed to fetch expensive resources", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch expensive resources: %w", err)
	}
	defer rows.Close()

	return r.scanCostDataRows(rows)
}

// DeleteOldCostData deletes cost data older than the specified date
func (r *CostRepository) DeleteOldCostData(ctx context.Context, beforeDate string) (int64, error) {
	r.logger.Info("Deleting old cost data",
		zap.String("before_date", beforeDate),
	)

	query := `DELETE FROM cost_data WHERE end_date < $1`

	result, err := r.pool.Exec(ctx, query, beforeDate)
	if err != nil {
		r.logger.Error("Failed to delete old cost data", zap.Error(err))
		return 0, fmt.Errorf("failed to delete old cost data: %w", err)
	}

	deletedCount := result.RowsAffected()
	r.logger.Info("Deleted old cost data",
		zap.Int64("count", deletedCount),
		zap.String("before_date", beforeDate),
	)

	return deletedCount, nil
}

// Helper methods

func (r *CostRepository) prepareMetadata(cost *aws.CostData) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	if cost.UsageType != "" {
		metadata["usage_type"] = cost.UsageType
	}
	if cost.Region != "" {
		metadata["region"] = cost.Region
	}
	if cost.AccountID != "" {
		metadata["account_id"] = cost.AccountID
	}
	
	return metadata
}

func (r *CostRepository) stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *CostRepository) scanCostDataRows(rows pgx.Rows) ([]*CostDataDB, error) {
	var costs []*CostDataDB
	
	for rows.Next() {
		var cost CostDataDB
		err := rows.Scan(
			&cost.ID,
			&cost.ResourceID,
			&cost.ResourceUUID,
			&cost.Service,
			&cost.CostAmount,
			&cost.Currency,
			&cost.StartDate,
			&cost.EndDate,
			&cost.Timestamp,
			&cost.UsageType,
			&cost.Region,
			&cost.AccountID,
			&cost.Metadata,
			&cost.CreatedAt,
			&cost.UpdatedAt,
		)
		if err != nil {
			r.logger.Warn("Failed to scan cost data row", zap.Error(err))
			continue
		}
		costs = append(costs, &cost)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cost data rows: %w", err)
	}

	return costs, nil
}

// LinkCostDataToResource links cost data to a resource UUID
func (r *CostRepository) LinkCostDataToResource(ctx context.Context, resourceID string, resourceUUID uuid.UUID) (int64, error) {
	r.logger.Info("Linking cost data to resource",
		zap.String("resource_id", resourceID),
		zap.String("resource_uuid", resourceUUID.String()),
	)

	query := `
		UPDATE cost_data 
		SET resource_uuid = $1, updated_at = NOW()
		WHERE resource_id = $2 AND (resource_uuid IS NULL OR resource_uuid != $1)
	`

	result, err := r.pool.Exec(ctx, query, resourceUUID, resourceID)
	if err != nil {
		r.logger.Error("Failed to link cost data to resource",
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to link cost data to resource: %w", err)
	}

	updatedCount := result.RowsAffected()
	r.logger.Info("Linked cost data to resource",
		zap.Int64("count", updatedCount),
		zap.String("resource_id", resourceID),
	)

	return updatedCount, nil
}
