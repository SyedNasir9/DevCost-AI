package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"devcost-ai/internal/models"
	"devcost-ai/pkg/logger"
)

// ResourceRepository handles database operations for resources
type ResourceRepository struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewResourceRepository creates a new resource repository
func NewResourceRepository(pool *pgxpool.Pool, log *logger.Logger) *ResourceRepository {
	return &ResourceRepository{
		pool:   pool,
		logger: log,
	}
}

// SaveResources performs bulk upsert of resources using ON CONFLICT
func (r *ResourceRepository) SaveResources(ctx context.Context, resources []*models.Resource) error {
	if len(resources) == 0 {
		r.logger.Debug("No resources to save")
		return nil
	}

	r.logger.Info("Saving resources",
		zap.Int("count", len(resources)),
	)

	start := time.Now()
	defer func() {
		r.logger.Info("Resources saved",
			zap.Int("count", len(resources)),
			zap.Duration("duration", time.Since(start)),
		)
	}()

	// Begin transaction for bulk operations
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare the upsert statement
	query := `
		INSERT INTO resources (
			id, resource_id, resource_type, provider, region, account_id,
			name, state, instance_type, tags, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (resource_id) 
		DO UPDATE SET
			resource_type = EXCLUDED.resource_type,
			provider = EXCLUDED.provider,
			region = EXCLUDED.region,
			account_id = EXCLUDED.account_id,
			name = EXCLUDED.name,
			state = EXCLUDED.state,
			instance_type = EXCLUDED.instance_type,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`

	// Prepare statement
	stmt, err := tx.Prepare(ctx, query)
	if err != nil {
		r.logger.Error("Failed to prepare statement", zap.Error(err))
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute bulk upsert
	for _, resource := range resources {
		// Convert tags to JSONB
		tagsJSON, err := json.Marshal(resource.Tags)
		if err != nil {
			r.logger.Error("Failed to marshal tags",
				zap.String("resource_id", resource.ResourceID),
				zap.Error(err),
			)
			continue // Skip this resource but continue with others
		}

		// Convert metadata to JSONB
		metadataJSON, err := json.Marshal(resource.Metadata)
		if err != nil {
			r.logger.Error("Failed to marshal metadata",
				zap.String("resource_id", resource.ResourceID),
				zap.Error(err),
			)
			continue // Skip this resource but continue with others
		}

		// Set updated_at if not already set
		if resource.UpdatedAt.IsZero() {
			resource.UpdatedAt = time.Now()
		}

		// Execute upsert
		_, err = stmt.Exec(ctx,
			resource.ID,
			resource.ResourceID,
			resource.ResourceType,
			resource.Provider,
			resource.Region,
			resource.AccountID,
			resource.Name,
			resource.State,
			resource.InstanceType,
			tagsJSON,
			metadataJSON,
			resource.CreatedAt,
			resource.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to upsert resource",
				zap.String("resource_id", resource.ResourceID),
				zap.Error(err),
			)
			continue // Continue with other resources
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Resources saved successfully",
		zap.Int("count", len(resources)),
	)

	return nil
}

// SaveResource performs upsert of a single resource
func (r *ResourceRepository) SaveResource(ctx context.Context, resource *models.Resource) error {
	return r.SaveResources(ctx, []*models.Resource{resource})
}

// GetResourceByResourceID retrieves a resource by its cloud provider ID
func (r *ResourceRepository) GetResourceByResourceID(ctx context.Context, resourceID string) (*models.Resource, error) {
	r.logger.Debug("Getting resource by resource ID",
		zap.String("resource_id", resourceID),
	)

	query := `
		SELECT id, resource_id, resource_type, provider, region, account_id,
			   name, state, instance_type, tags, metadata, created_at, updated_at
		FROM resources
		WHERE resource_id = $1
	`

	var resource models.Resource
	var tagsJSON, metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, resourceID).Scan(
		&resource.ID,
		&resource.ResourceID,
		&resource.ResourceType,
		&resource.Provider,
		&resource.Region,
		&resource.AccountID,
		&resource.Name,
		&resource.State,
		&resource.InstanceType,
		&tagsJSON,
		&metadataJSON,
		&resource.CreatedAt,
		&resource.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("resource with ID %s not found", resourceID)
		}
		r.logger.Error("Failed to get resource",
			zap.String("resource_id", resourceID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	// Unmarshal tags
	if err := json.Unmarshal(tagsJSON, &resource.Tags); err != nil {
		r.logger.Error("Failed to unmarshal tags",
			zap.String("resource_id", resourceID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	// Unmarshal metadata
	if err := json.Unmarshal(metadataJSON, &resource.Metadata); err != nil {
		r.logger.Error("Failed to unmarshal metadata",
			zap.String("resource_id", resourceID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	r.logger.Debug("Resource retrieved successfully",
		zap.String("resource_id", resourceID),
		zap.String("resource_type", string(resource.ResourceType)),
	)

	return &resource, nil
}

// GetResourcesByType retrieves resources by type
func (r *ResourceRepository) GetResourcesByType(ctx context.Context, resourceType models.ResourceType) ([]*models.Resource, error) {
	r.logger.Debug("Getting resources by type",
		zap.String("resource_type", string(resourceType)),
	)

	query := `
		SELECT id, resource_id, resource_type, provider, region, account_id,
			   name, state, instance_type, tags, metadata, created_at, updated_at
		FROM resources
		WHERE resource_type = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, resourceType)
	if err != nil {
		r.logger.Error("Failed to query resources by type",
			zap.String("resource_type", string(resourceType)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to query resources by type: %w", err)
	}
	defer rows.Close()

	var resources []*models.Resource
	for rows.Next() {
		var resource models.Resource
		var tagsJSON, metadataJSON []byte

		err := rows.Scan(
			&resource.ID,
			&resource.ResourceID,
			&resource.ResourceType,
			&resource.Provider,
			&resource.Region,
			&resource.AccountID,
			&resource.Name,
			&resource.State,
			&resource.InstanceType,
			&tagsJSON,
			&metadataJSON,
			&resource.CreatedAt,
			&resource.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan resource row", zap.Error(err))
			continue // Skip this row but continue with others
		}

		// Unmarshal tags
		if err := json.Unmarshal(tagsJSON, &resource.Tags); err != nil {
			r.logger.Error("Failed to unmarshal tags", zap.Error(err))
			continue
		}

		// Unmarshal metadata
		if err := json.Unmarshal(metadataJSON, &resource.Metadata); err != nil {
			r.logger.Error("Failed to unmarshal metadata", zap.Error(err))
			continue
		}

		resources = append(resources, &resource)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Row iteration error", zap.Error(err))
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	r.logger.Debug("Resources retrieved by type",
		zap.String("resource_type", string(resourceType)),
		zap.Int("count", len(resources)),
	)

	return resources, nil
}

// GetResourcesByProvider retrieves resources by provider
func (r *ResourceRepository) GetResourcesByProvider(ctx context.Context, provider string) ([]*models.Resource, error) {
	r.logger.Debug("Getting resources by provider",
		zap.String("provider", provider),
	)

	query := `
		SELECT id, resource_id, resource_type, provider, region, account_id,
			   name, state, instance_type, tags, metadata, created_at, updated_at
		FROM resources
		WHERE provider = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, provider)
	if err != nil {
		r.logger.Error("Failed to query resources by provider",
			zap.String("provider", provider),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to query resources by provider: %w", err)
	}
	defer rows.Close()

	var resources []*models.Resource
	for rows.Next() {
		var resource models.Resource
		var tagsJSON, metadataJSON []byte

		err := rows.Scan(
			&resource.ID,
			&resource.ResourceID,
			&resource.ResourceType,
			&resource.Provider,
			&resource.Region,
			&resource.AccountID,
			&resource.Name,
			&resource.State,
			&resource.InstanceType,
			&tagsJSON,
			&metadataJSON,
			&resource.CreatedAt,
			&resource.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan resource row", zap.Error(err))
			continue
		}

		// Unmarshal tags and metadata
		if err := json.Unmarshal(tagsJSON, &resource.Tags); err != nil {
			r.logger.Error("Failed to unmarshal tags", zap.Error(err))
			continue
		}

		if err := json.Unmarshal(metadataJSON, &resource.Metadata); err != nil {
			r.logger.Error("Failed to unmarshal metadata", zap.Error(err))
			continue
		}

		resources = append(resources, &resource)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Row iteration error", zap.Error(err))
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	r.logger.Debug("Resources retrieved by provider",
		zap.String("provider", provider),
		zap.Int("count", len(resources)),
	)

	return resources, nil
}

// GetResourcesByFilter retrieves resources with custom filters
func (r *ResourceRepository) GetResourcesByFilter(ctx context.Context, filter *ResourceFilter) ([]*models.Resource, error) {
	r.logger.Debug("Getting resources by filter",
		zap.Any("filter", filter),
	)

	// Build dynamic query
	query := `
		SELECT id, resource_id, resource_type, provider, region, account_id,
			   name, state, instance_type, tags, metadata, created_at, updated_at
		FROM resources
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add resource type filter
	if len(filter.ResourceTypes) > 0 {
		placeholders := make([]string, len(filter.ResourceTypes))
		for i, rt := range filter.ResourceTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex+i)
			args = append(args, rt)
		}
		query += fmt.Sprintf(" AND resource_type IN (%s)", 
			fmt.Sprintf("%s", placeholders))
		argIndex += len(filter.ResourceTypes)
	}

	// Add state filter
	if len(filter.States) > 0 {
		placeholders := make([]string, len(filter.States))
		for i, state := range filter.States {
			placeholders[i] = fmt.Sprintf("$%d", argIndex+i)
			args = append(args, state)
		}
		query += fmt.Sprintf(" AND state IN (%s)", 
			fmt.Sprintf("%s", placeholders))
		argIndex += len(filter.States)
	}

	// Add region filter
	if len(filter.Regions) > 0 {
		placeholders := make([]string, len(filter.Regions))
		for i, region := range filter.Regions {
			placeholders[i] = fmt.Sprintf("$%d", argIndex+i)
			args = append(args, region)
		}
		query += fmt.Sprintf(" AND region IN (%s)", 
			fmt.Sprintf("%s", placeholders))
		argIndex += len(filter.Regions)
	}

	// Add account ID filter
	if len(filter.AccountIDs) > 0 {
		placeholders := make([]string, len(filter.AccountIDs))
		for i, accountID := range filter.AccountIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex+i)
			args = append(args, accountID)
		}
		query += fmt.Sprintf(" AND account_id IN (%s)", 
			fmt.Sprintf("%s", placeholders))
		argIndex += len(filter.AccountIDs)
	}

	// Add tag filter (JSONB contains)
	if len(filter.Tags) > 0 {
		for key, value := range filter.Tags {
			query += fmt.Sprintf(" AND tags->>'%s' = $%d", key, argIndex)
			args = append(args, value)
			argIndex++
		}
	}

	// Add ordering and limit
	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to query resources by filter", zap.Error(err))
		return nil, fmt.Errorf("failed to query resources by filter: %w", err)
	}
	defer rows.Close()

	var resources []*models.Resource
	for rows.Next() {
		var resource models.Resource
		var tagsJSON, metadataJSON []byte

		err := rows.Scan(
			&resource.ID,
			&resource.ResourceID,
			&resource.ResourceType,
			&resource.Provider,
			&resource.Region,
			&resource.AccountID,
			&resource.Name,
			&resource.State,
			&resource.InstanceType,
			&tagsJSON,
			&metadataJSON,
			&resource.CreatedAt,
			&resource.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan resource row", zap.Error(err))
			continue
		}

		// Unmarshal tags and metadata
		if err := json.Unmarshal(tagsJSON, &resource.Tags); err != nil {
			r.logger.Error("Failed to unmarshal tags", zap.Error(err))
			continue
		}

		if err := json.Unmarshal(metadataJSON, &resource.Metadata); err != nil {
			r.logger.Error("Failed to unmarshal metadata", zap.Error(err))
			continue
		}

		resources = append(resources, &resource)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Row iteration error", zap.Error(err))
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	r.logger.Debug("Resources retrieved by filter",
		zap.Int("count", len(resources)),
	)

	return resources, nil
}

// DeleteResourceByResourceID deletes a resource by its cloud provider ID
func (r *ResourceRepository) DeleteResourceByResourceID(ctx context.Context, resourceID string) error {
	r.logger.Info("Deleting resource",
		zap.String("resource_id", resourceID),
	)

	query := `DELETE FROM resources WHERE resource_id = $1`

	result, err := r.pool.Exec(ctx, query, resourceID)
	if err != nil {
		r.logger.Error("Failed to delete resource",
			zap.String("resource_id", resourceID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.logger.Warn("No resource found to delete",
			zap.String("resource_id", resourceID),
		)
		return fmt.Errorf("resource with ID %s not found", resourceID)
	}

	r.logger.Info("Resource deleted successfully",
		zap.String("resource_id", resourceID),
		zap.Int64("rows_affected", rowsAffected),
	)

	return nil
}

// GetResourceCount returns the total count of resources
func (r *ResourceRepository) GetResourceCount(ctx context.Context) (int64, error) {
	r.logger.Debug("Getting resource count")

	query := `SELECT COUNT(*) FROM resources`

	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		r.logger.Error("Failed to get resource count", zap.Error(err))
		return 0, fmt.Errorf("failed to get resource count: %w", err)
	}

	r.logger.Debug("Resource count retrieved", zap.Int64("count", count))
	return count, nil
}

// GetResourceStatistics returns resource statistics
func (r *ResourceRepository) GetResourceStatistics(ctx context.Context) (*ResourceStatistics, error) {
	r.logger.Debug("Getting resource statistics")

	query := `
		SELECT 
			COUNT(*) as total_count,
			COUNT(DISTINCT resource_type) as type_count,
			COUNT(DISTINCT provider) as provider_count,
			COUNT(DISTINCT region) as region_count,
			COUNT(DISTINCT account_id) as account_count
		FROM resources
	`

	var stats ResourceStatistics
	err := r.pool.QueryRow(ctx, query).Scan(
		&stats.TotalCount,
		&stats.TypeCount,
		&stats.ProviderCount,
		&stats.RegionCount,
		&stats.AccountCount,
	)

	if err != nil {
		r.logger.Error("Failed to get resource statistics", zap.Error(err))
		return nil, fmt.Errorf("failed to get resource statistics: %w", err)
	}

	// Get counts by type
	typeQuery := `
		SELECT resource_type, COUNT(*) as count
		FROM resources
		GROUP BY resource_type
		ORDER BY count DESC
	`

	rows, err := r.pool.Query(ctx, typeQuery)
	if err != nil {
		r.logger.Error("Failed to get resource type statistics", zap.Error(err))
		return nil, fmt.Errorf("failed to get resource type statistics: %w", err)
	}
	defer rows.Close()

	stats.ByType = make(map[string]int64)
	for rows.Next() {
		var resourceType string
		var count int64
		if err := rows.Scan(&resourceType, &count); err != nil {
			r.logger.Error("Failed to scan type statistics row", zap.Error(err))
			continue
		}
		stats.ByType[resourceType] = count
	}

	r.logger.Debug("Resource statistics retrieved",
		zap.Int64("total_count", stats.TotalCount),
		zap.Int("type_count", stats.TypeCount),
	)

	return &stats, nil
}

// ResourceFilter represents filtering options for resource queries
type ResourceFilter struct {
	ResourceTypes []models.ResourceType `json:"resource_types"`
	States        []models.ResourceState `json:"states"`
	Regions       []string              `json:"regions"`
	AccountIDs    []string              `json:"account_ids"`
	Tags          map[string]string      `json:"tags"`
	Limit         int                   `json:"limit"`
	Offset        int                   `json:"offset"`
}

// ResourceStatistics represents resource statistics
type ResourceStatistics struct {
	TotalCount    int64            `json:"total_count"`
	TypeCount     int64            `json:"type_count"`
	ProviderCount int64            `json:"provider_count"`
	RegionCount   int64            `json:"region_count"`
	AccountCount  int64            `json:"account_count"`
	ByType        map[string]int64 `json:"by_type"`
}
