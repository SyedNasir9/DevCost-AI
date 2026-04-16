package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// RecommendationRepository provides data access for recommendations
type RecommendationRepository struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewRecommendationRepository creates a new recommendation repository
func NewRecommendationRepository(pool *pgxpool.Pool, logger *logger.Logger) *RecommendationRepository {
	return &RecommendationRepository{
		pool:   pool,
		logger: logger,
	}
}

// SaveRecommendation saves a single recommendation to the database
func (r *RecommendationRepository) SaveRecommendation(ctx context.Context, rec *services.Recommendation) error {
	r.logger.Debug("Saving recommendation",
		zap.String("resource_id", rec.ResourceID),
		zap.String("type", string(rec.RecommendationType)),
	)

	// Serialize JSON fields
	currentStateJSON, err := json.Marshal(rec.CurrentState)
	if err != nil {
		r.logger.Error("Failed to marshal current state", zap.Error(err))
		return fmt.Errorf("failed to marshal current state: %w", err)
	}

	proposedStateJSON, err := json.Marshal(rec.ProposedState)
	if err != nil {
		r.logger.Error("Failed to marshal proposed state", zap.Error(err))
		return fmt.Errorf("failed to marshal proposed state: %w", err)
	}

	implementationStepsJSON, err := json.Marshal(rec.ImplementationSteps)
	if err != nil {
		r.logger.Error("Failed to marshal implementation steps", zap.Error(err))
		return fmt.Errorf("failed to marshal implementation steps: %w", err)
	}

	alternativesJSON, err := json.Marshal(rec.Alternatives)
	if err != nil {
		r.logger.Error("Failed to marshal alternatives", zap.Error(err))
		return fmt.Errorf("failed to marshal alternatives: %w", err)
	}

	query := `
		INSERT INTO recommendations (
			id, resource_id, resource_uuid, resource_type, resource_name,
			recommendation_type, status, priority, title, description, rationale,
			current_state, proposed_state, estimated_savings_usd, savings_currency,
			risk_level, implementation_steps, alternatives, waste_id, cost_data_id,
			valid_from, valid_until, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			current_state = EXCLUDED.current_state,
			proposed_state = EXCLUDED.proposed_state,
			estimated_savings_usd = EXCLUDED.estimated_savings_usd,
			risk_level = EXCLUDED.risk_level,
			implementation_steps = EXCLUDED.implementation_steps,
			alternatives = EXCLUDED.alternatives,
			valid_until = EXCLUDED.valid_until,
			implemented_at = EXCLUDED.implemented_at,
			implemented_by = EXCLUDED.implemented_by,
			updated_at = NOW()
	`

	var resourceUUID interface{}
	if rec.ResourceUUID != uuid.Nil {
		resourceUUID = rec.ResourceUUID
	}

	var wasteID, costDataID interface{}
	if rec.WasteID != nil {
		wasteID = *rec.WasteID
	}
	if rec.CostDataID != nil {
		costDataID = *rec.CostDataID
	}

	_, err = r.pool.Exec(ctx, query,
		rec.ID,
		rec.ResourceID,
		resourceUUID,
		rec.ResourceType,
		rec.ResourceName,
		rec.RecommendationType,
		rec.Status,
		rec.Priority,
		rec.Title,
		rec.Description,
		rec.Rationale,
		currentStateJSON,
		proposedStateJSON,
		rec.EstimatedSavings,
		rec.SavingsCurrency,
		rec.RiskLevel,
		implementationStepsJSON,
		alternativesJSON,
		wasteID,
		costDataID,
		rec.ValidFrom,
		rec.ValidUntil,
		rec.CreatedAt,
		rec.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save recommendation",
			zap.String("resource_id", rec.ResourceID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save recommendation: %w", err)
	}

	r.logger.Debug("Recommendation saved successfully",
		zap.String("id", rec.ID.String()),
	)

	return nil
}

// SaveRecommendations saves multiple recommendations using batch insert
func (r *RecommendationRepository) SaveRecommendations(ctx context.Context, recs []*services.Recommendation) error {
	if len(recs) == 0 {
		r.logger.Debug("No recommendations to save")
		return nil
	}

	r.logger.Info("Saving recommendations batch",
		zap.Int("count", len(recs)),
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
		INSERT INTO recommendations (
			id, resource_id, resource_uuid, resource_type, resource_name,
			recommendation_type, status, priority, title, description, rationale,
			current_state, proposed_state, estimated_savings_usd, savings_currency,
			risk_level, implementation_steps, alternatives, waste_id, cost_data_id,
			valid_from, valid_until, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			current_state = EXCLUDED.current_state,
			proposed_state = EXCLUDED.proposed_state,
			estimated_savings_usd = EXCLUDED.estimated_savings_usd,
			risk_level = EXCLUDED.risk_level,
			implementation_steps = EXCLUDED.implementation_steps,
			alternatives = EXCLUDED.alternatives,
			valid_until = EXCLUDED.valid_until,
			implemented_at = EXCLUDED.implemented_at,
			implemented_by = EXCLUDED.implemented_by,
			updated_at = NOW()
	`

	batch := &pgx.Batch{}

	for _, rec := range recs {
		// Serialize JSON fields
		currentStateJSON, _ := json.Marshal(rec.CurrentState)
		proposedStateJSON, _ := json.Marshal(rec.ProposedState)
		implementationStepsJSON, _ := json.Marshal(rec.ImplementationSteps)
		alternativesJSON, _ := json.Marshal(rec.Alternatives)

		var resourceUUID interface{}
		if rec.ResourceUUID != uuid.Nil {
			resourceUUID = rec.ResourceUUID
		}

		var wasteID, costDataID interface{}
		if rec.WasteID != nil {
			wasteID = *rec.WasteID
		}
		if rec.CostDataID != nil {
			costDataID = *rec.CostDataID
		}

		batch.Queue(query,
			rec.ID,
			rec.ResourceID,
			resourceUUID,
			rec.ResourceType,
			rec.ResourceName,
			rec.RecommendationType,
			rec.Status,
			rec.Priority,
			rec.Title,
			rec.Description,
			rec.Rationale,
			currentStateJSON,
			proposedStateJSON,
			rec.EstimatedSavings,
			rec.SavingsCurrency,
			rec.RiskLevel,
			implementationStepsJSON,
			alternativesJSON,
			wasteID,
			costDataID,
			rec.ValidFrom,
			rec.ValidUntil,
			rec.CreatedAt,
			rec.UpdatedAt,
		)
	}

	// Execute batch
	batchResults := tx.SendBatch(ctx, batch)
	
	// Process results
	for i := 0; i < batch.Len(); i++ {
		_, err := batchResults.Exec()
		if err != nil {
			r.logger.Warn("Failed to save recommendation batch item",
				zap.Int("index", i),
				zap.Error(err),
			)
		}
	}

	if err := batchResults.Close(); err != nil {
		r.logger.Error("Failed to close batch results", zap.Error(err))
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Recommendations batch saved successfully",
		zap.Int("count", len(recs)),
	)

	return nil
}

// GetRecommendationByID retrieves a recommendation by its ID
func (r *RecommendationRepository) GetRecommendationByID(ctx context.Context, id uuid.UUID) (*services.Recommendation, error) {
	query := `
		SELECT 
			id, resource_id, resource_uuid, resource_type, resource_name,
			recommendation_type, status, priority, title, description, rationale,
			current_state, proposed_state, estimated_savings_usd, savings_currency,
			risk_level, implementation_steps, alternatives, waste_id, cost_data_id,
			valid_from, valid_until, implemented_at, implemented_by,
			created_at, updated_at
		FROM recommendations
		WHERE id = $1
	`

	rec := &services.Recommendation{}
	var currentStateJSON, proposedStateJSON, implementationStepsJSON, alternativesJSON []byte
	var wasteID, costDataID uuid.UUID

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rec.ID,
		&rec.ResourceID,
		&rec.ResourceUUID,
		&rec.ResourceType,
		&rec.ResourceName,
		&rec.RecommendationType,
		&rec.Status,
		&rec.Priority,
		&rec.Title,
		&rec.Description,
		&rec.Rationale,
		&currentStateJSON,
		&proposedStateJSON,
		&rec.EstimatedSavings,
		&rec.SavingsCurrency,
		&rec.RiskLevel,
		&implementationStepsJSON,
		&alternativesJSON,
		&wasteID,
		&costDataID,
		&rec.ValidFrom,
		&rec.ValidUntil,
		&rec.ImplementedAt,
		&rec.ImplementedBy,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to get recommendation by ID",
			zap.String("id", id.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get recommendation: %w", err)
	}

	// Deserialize JSON fields
	if len(currentStateJSON) > 0 {
		json.Unmarshal(currentStateJSON, &rec.CurrentState)
	}
	if len(proposedStateJSON) > 0 {
		json.Unmarshal(proposedStateJSON, &rec.ProposedState)
	}
	if len(implementationStepsJSON) > 0 {
		json.Unmarshal(implementationStepsJSON, &rec.ImplementationSteps)
	}
	if len(alternativesJSON) > 0 {
		json.Unmarshal(alternativesJSON, &rec.Alternatives)
	}

	// Set UUID pointers if not nil
	if wasteID != uuid.Nil {
		rec.WasteID = &wasteID
	}
	if costDataID != uuid.Nil {
		rec.CostDataID = &costDataID
	}

	return rec, nil
}

// GetRecommendationsByResource retrieves recommendations for a specific resource
func (r *RecommendationRepository) GetRecommendationsByResource(ctx context.Context, resourceID string) ([]*services.Recommendation, error) {
	query := `
		SELECT 
			id, resource_id, resource_uuid, resource_type, resource_name,
			recommendation_type, status, priority, title, description, rationale,
			current_state, proposed_state, estimated_savings_usd, savings_currency,
			risk_level, implementation_steps, alternatives, waste_id, cost_data_id,
			valid_from, valid_until, implemented_at, implemented_by,
			created_at, updated_at
		FROM recommendations
		WHERE resource_id = $1
		ORDER BY created_at DESC
	`

	return r.scanRecommendations(ctx, query, resourceID)
}

// GetRecommendationsByStatus retrieves recommendations by status
func (r *RecommendationRepository) GetRecommendationsByStatus(ctx context.Context, status services.RecommendationStatus) ([]*services.Recommendation, error) {
	query := `
		SELECT 
			id, resource_id, resource_uuid, resource_type, resource_name,
			recommendation_type, status, priority, title, description, rationale,
			current_state, proposed_state, estimated_savings_usd, savings_currency,
			risk_level, implementation_steps, alternatives, waste_id, cost_data_id,
			valid_from, valid_until, implemented_at, implemented_by,
			created_at, updated_at
		FROM recommendations
		WHERE status = $1
		ORDER BY estimated_savings_usd DESC, created_at DESC
	`

	return r.scanRecommendations(ctx, query, status)
}

// GetActiveRecommendations retrieves all active recommendations
func (r *RecommendationRepository) GetActiveRecommendations(ctx context.Context) ([]*services.Recommendation, error) {
	query := `
		SELECT 
			id, resource_id, resource_uuid, resource_type, resource_name,
			recommendation_type, status, priority, title, description, rationale,
			current_state, proposed_state, estimated_savings_usd, savings_currency,
			risk_level, implementation_steps, alternatives, waste_id, cost_data_id,
			valid_from, valid_until, implemented_at, implemented_by,
			created_at, updated_at
		FROM recommendations
		WHERE status = 'active'
		ORDER BY 
			CASE priority 
				WHEN 'critical' THEN 4
				WHEN 'high' THEN 3
				WHEN 'medium' THEN 2
				WHEN 'low' THEN 1
			END DESC,
			estimated_savings_usd DESC
	`

	return r.scanRecommendations(ctx, query)
}

// UpdateRecommendationStatus updates the status of a recommendation
func (r *RecommendationRepository) UpdateRecommendationStatus(ctx context.Context, id uuid.UUID, status services.RecommendationStatus) error {
	query := `
		UPDATE recommendations 
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		r.logger.Error("Failed to update recommendation status",
			zap.String("id", id.String()),
			zap.String("status", string(status)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update recommendation status: %w", err)
	}

	r.logger.Debug("Recommendation status updated",
		zap.String("id", id.String()),
		zap.String("status", string(status)),
	)

	return nil
}

// GetRecommendationSummary retrieves a summary of recommendations
func (r *RecommendationRepository) GetRecommendationSummary(ctx context.Context) (*services.RecommendationSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_count,
			recommendation_type,
			priority,
			status,
			SUM(estimated_savings_usd) as total_savings
		FROM recommendations
		GROUP BY recommendation_type, priority, status
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		r.logger.Error("Failed to get recommendation summary", zap.Error(err))
		return nil, fmt.Errorf("failed to get recommendation summary: %w", err)
	}
	defer rows.Close()

	summary := &services.RecommendationSummary{
		ByType:     make(map[services.RecommendationType]int),
		ByPriority: make(map[services.RecommendationPriority]int),
		ByStatus:   make(map[services.RecommendationStatus]int),
	}

	for rows.Next() {
		var count int
		var recType services.RecommendationType
		var priority services.RecommendationPriority
		var status services.RecommendationStatus
		var savings float64

		err := rows.Scan(&count, &recType, &priority, &status, &savings)
		if err != nil {
			r.logger.Warn("Failed to scan recommendation summary row", zap.Error(err))
			continue
		}

		summary.TotalCount += count
		summary.ByType[recType] += count
		summary.ByPriority[priority] += count
		summary.ByStatus[status] += count
		summary.TotalEstimatedSavings += savings

		if priority == services.RecommendationPriorityHigh || 
		   priority == services.RecommendationPriorityCritical {
			summary.HighPriorityCount += count
		}

		if priority == services.RecommendationPriorityCritical {
			summary.CriticalCount += count
		}
	}

	// Calculate implementation rate
	if summary.TotalCount > 0 {
		implementedCount := summary.ByStatus[services.RecommendationStatusImplemented]
		summary.ImplementationRate = float64(implementedCount) / float64(summary.TotalCount) * 100
	}

	return summary, nil
}

// scanRecommendations scans multiple recommendation rows
func (r *RecommendationRepository) scanRecommendations(ctx context.Context, query string, args ...interface{}) ([]*services.Recommendation, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recommendations []*services.Recommendation

	for rows.Next() {
		rec := &services.Recommendation{}
		var currentStateJSON, proposedStateJSON, implementationStepsJSON, alternativesJSON []byte
		var wasteID, costDataID uuid.UUID

		err := rows.Scan(
			&rec.ID,
			&rec.ResourceID,
			&rec.ResourceUUID,
			&rec.ResourceType,
			&rec.ResourceName,
			&rec.RecommendationType,
			&rec.Status,
			&rec.Priority,
			&rec.Title,
			&rec.Description,
			&rec.Rationale,
			&currentStateJSON,
			&proposedStateJSON,
			&rec.EstimatedSavings,
			&rec.SavingsCurrency,
			&rec.RiskLevel,
			&implementationStepsJSON,
			&alternativesJSON,
			&wasteID,
			&costDataID,
			&rec.ValidFrom,
			&rec.ValidUntil,
			&rec.ImplementedAt,
			&rec.ImplementedBy,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		)
		if err != nil {
			r.logger.Warn("Failed to scan recommendation row", zap.Error(err))
			continue
		}

		// Deserialize JSON fields
		if len(currentStateJSON) > 0 {
			json.Unmarshal(currentStateJSON, &rec.CurrentState)
		}
		if len(proposedStateJSON) > 0 {
			json.Unmarshal(proposedStateJSON, &rec.ProposedState)
		}
		if len(implementationStepsJSON) > 0 {
			json.Unmarshal(implementationStepsJSON, &rec.ImplementationSteps)
		}
		if len(alternativesJSON) > 0 {
			json.Unmarshal(alternativesJSON, &rec.Alternatives)
		}

		// Set UUID pointers if not nil
		if wasteID != uuid.Nil {
			rec.WasteID = &wasteID
		}
		if costDataID != uuid.Nil {
			rec.CostDataID = &costDataID
		}

		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}
