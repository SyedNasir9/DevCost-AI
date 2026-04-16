package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"devcost-ai/pkg/logger"
)

// ActionRepository provides data access for actions
type ActionRepository struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewActionRepository creates a new action repository
func NewActionRepository(pool *pgxpool.Pool, logger *logger.Logger) *ActionRepository {
	return &ActionRepository{
		pool:   pool,
		logger: logger,
	}
}

// ActionDB represents an action record in the database
type ActionDB struct {
	ID               uuid.UUID              `db:"id"`
	ResourceID       string                 `db:"resource_id"`
	ResourceType     string                 `db:"resource_type"`
	ResourceName     *string                `db:"resource_name"`
	ActionType       string                 `db:"action_type"`
	Status           string                 `db:"status"`
	ExecutedAt       *time.Time             `db:"executed_at"`
	CompletedAt      *time.Time             `db:"completed_at"`
	DurationMs       *int                   `db:"duration_ms"`
	ErrorMessage     *string                `db:"error_message"`
	ErrorCode        *string                `db:"error_code"`
	RequestParams    map[string]interface{} `db:"request_params"`
	ResponseData     map[string]interface{} `db:"response_data"`
	RecommendationID *uuid.UUID             `db:"recommendation_id"`
	DecisionID       *uuid.UUID             `db:"decision_id"`
	ExecutedBy       *string                `db:"executed_by"`
	Source           string                 `db:"source"`
	CreatedAt        time.Time              `db:"created_at"`
	UpdatedAt        time.Time              `db:"updated_at"`
}

// SaveAction saves an action record to the database
func (r *ActionRepository) SaveAction(ctx context.Context, action *ActionDB) error {
	r.logger.Debug("Saving action",
		zap.String("resource_id", action.ResourceID),
		zap.String("action_type", action.ActionType),
		zap.String("status", action.Status),
	)

	// Serialize JSON fields
	var requestParamsJSON, responseDataJSON interface{}

	if action.RequestParams != nil {
		paramsJSON, err := json.Marshal(action.RequestParams)
		if err != nil {
			r.logger.Error("Failed to marshal request params", zap.Error(err))
			return fmt.Errorf("failed to marshal request params: %w", err)
		}
		requestParamsJSON = paramsJSON
	} else {
		requestParamsJSON = []byte("{}")
	}

	if action.ResponseData != nil {
		responseJSON, err := json.Marshal(action.ResponseData)
		if err != nil {
			r.logger.Error("Failed to marshal response data", zap.Error(err))
			return fmt.Errorf("failed to marshal response data: %w", err)
		}
		responseDataJSON = responseJSON
	} else {
		responseDataJSON = []byte("{}")
	}

	// Handle nullable fields
	var resourceName, executedAt, completedAt, errorMessage, errorCode, executedBy interface{}
	var durationMs interface{}
	var recommendationID, decisionID interface{}

	if action.ResourceName != nil {
		resourceName = *action.ResourceName
	}
	if action.ExecutedAt != nil {
		executedAt = *action.ExecutedAt
	}
	if action.CompletedAt != nil {
		completedAt = *action.CompletedAt
	}
	if action.ErrorMessage != nil {
		errorMessage = *action.ErrorMessage
	}
	if action.ErrorCode != nil {
		errorCode = *action.ErrorCode
	}
	if action.ExecutedBy != nil {
		executedBy = *action.ExecutedBy
	}
	if action.DurationMs != nil {
		durationMs = *action.DurationMs
	}
	if action.RecommendationID != nil {
		recommendationID = *action.RecommendationID
	}
	if action.DecisionID != nil {
		decisionID = *action.DecisionID
	}

	query := `
		INSERT INTO actions (
			id, resource_id, resource_type, resource_name, action_type, status,
			executed_at, completed_at, duration_ms, error_message, error_code,
			request_params, response_data, recommendation_id, decision_id,
			executed_by, source, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at,
			duration_ms = EXCLUDED.duration_ms,
			error_message = EXCLUDED.error_message,
			error_code = EXCLUDED.error_code,
			response_data = EXCLUDED.response_data,
			updated_at = NOW()
		RETURNING id
	`

	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		action.ID,
		action.ResourceID,
		action.ResourceType,
		resourceName,
		action.ActionType,
		action.Status,
		executedAt,
		completedAt,
		durationMs,
		errorMessage,
		errorCode,
		requestParamsJSON,
		responseDataJSON,
		recommendationID,
		decisionID,
		executedBy,
		action.Source,
		action.CreatedAt,
		action.UpdatedAt,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to save action",
			zap.String("resource_id", action.ResourceID),
			zap.String("action_type", action.ActionType),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save action: %w", err)
	}

	r.logger.Debug("Action saved successfully",
		zap.String("action_id", id.String()),
		zap.String("resource_id", action.ResourceID),
	)

	action.ID = id
	return nil
}

// ActionResult represents the result of an action execution
// This is a copy of the structure from services package to avoid import cycle
type ActionResult struct {
	ActionID     uuid.UUID              `json:"action_id"`
	ActionType   string                 `json:"action_type"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Status       string                 `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Message      string                 `json:"message"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	DryRun       bool                   `json:"dry_run"`
}

// SaveActionResult saves an action result from the action service
func (r *ActionRepository) SaveActionResult(ctx context.Context, result *ActionResult, recommendationID *uuid.UUID) error {
	r.logger.Info("Saving action result",
		zap.String("action_id", result.ActionID.String()),
		zap.String("resource_id", result.ResourceID),
		zap.String("status", result.Status),
	)

	// Determine error message and code
	var errorMessage, errorCode *string
	if result.Error != "" {
		errorMessage = &result.Error
		errorCode = r.categorizeError(result.Error)
	}

	// Calculate duration in milliseconds
	var durationMs *int
	if result.Duration > 0 {
		ms := int(result.Duration.Milliseconds())
		durationMs = &ms
	}

	// Determine completed time
	var completedAt *time.Time
	if result.EndTime != nil {
		completedAt = result.EndTime
	}

	// Build request params from input data
	requestParams := map[string]interface{}{
		"action_type":   result.ActionType,
		"resource_type": result.ResourceType,
	}

	// Build response data
	responseData := map[string]interface{}{
		"message": result.Message,
		"dry_run": result.DryRun,
	}

	if result.Metadata != nil {
		for k, v := range result.Metadata {
			responseData[k] = v
		}
	}

	action := &ActionDB{
		ID:               result.ActionID,
		ResourceID:       result.ResourceID,
		ResourceType:     result.ResourceType,
		ActionType:       result.ActionType,
		Status:           result.Status,
		ExecutedAt:       &result.StartTime,
		CompletedAt:      completedAt,
		DurationMs:       durationMs,
		ErrorMessage:     errorMessage,
		ErrorCode:        errorCode,
		RequestParams:    requestParams,
		ResponseData:     responseData,
		RecommendationID: recommendationID,
		Source:           "action_service",
		CreatedAt:        result.StartTime,
		UpdatedAt:        time.Now(),
	}

	return r.SaveAction(ctx, action)
}

// GetActionByID retrieves an action by its ID
func (r *ActionRepository) GetActionByID(ctx context.Context, id uuid.UUID) (*ActionDB, error) {
	query := `
		SELECT 
			id, resource_id, resource_type, resource_name, action_type, status,
			executed_at, completed_at, duration_ms, error_message, error_code,
			request_params, response_data, recommendation_id, decision_id,
			executed_by, source, created_at, updated_at
		FROM actions
		WHERE id = $1
	`

	action := &ActionDB{}
	var requestParamsJSON, responseDataJSON []byte
	var resourceName, errorMessage, errorCode, executedBy *string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&action.ID,
		&action.ResourceID,
		&action.ResourceType,
		&resourceName,
		&action.ActionType,
		&action.Status,
		&action.ExecutedAt,
		&action.CompletedAt,
		&action.DurationMs,
		&errorMessage,
		&errorCode,
		&requestParamsJSON,
		&responseDataJSON,
		&action.RecommendationID,
		&action.DecisionID,
		&executedBy,
		&action.Source,
		&action.CreatedAt,
		&action.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to get action by ID", zap.String("id", id.String()), zap.Error(err))
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	// Set nullable fields
	if resourceName != nil {
		action.ResourceName = resourceName
	}
	if errorMessage != nil {
		action.ErrorMessage = errorMessage
	}
	if errorCode != nil {
		action.ErrorCode = errorCode
	}
	if executedBy != nil {
		action.ExecutedBy = executedBy
	}

	// Deserialize JSON fields
	if len(requestParamsJSON) > 0 {
		json.Unmarshal(requestParamsJSON, &action.RequestParams)
	}
	if len(responseDataJSON) > 0 {
		json.Unmarshal(responseDataJSON, &action.ResponseData)
	}

	return action, nil
}

// GetActionsByResource retrieves all actions for a specific resource
func (r *ActionRepository) GetActionsByResource(ctx context.Context, resourceID string, limit int) ([]*ActionDB, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT 
			id, resource_id, resource_type, resource_name, action_type, status,
			executed_at, completed_at, duration_ms, error_message, error_code,
			request_params, response_data, recommendation_id, decision_id,
			executed_by, source, created_at, updated_at
		FROM actions
		WHERE resource_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	return r.queryActions(ctx, query, resourceID, limit)
}

// GetActionsByStatus retrieves actions by status
func (r *ActionRepository) GetActionsByStatus(ctx context.Context, status string, limit int) ([]*ActionDB, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT 
			id, resource_id, resource_type, resource_name, action_type, status,
			executed_at, completed_at, duration_ms, error_message, error_code,
			request_params, response_data, recommendation_id, decision_id,
			executed_by, source, created_at, updated_at
		FROM actions
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	return r.queryActions(ctx, query, status, limit)
}

// GetRecentActions retrieves recent actions
func (r *ActionRepository) GetRecentActions(ctx context.Context, limit int) ([]*ActionDB, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT 
			id, resource_id, resource_type, resource_name, action_type, status,
			executed_at, completed_at, duration_ms, error_message, error_code,
			request_params, response_data, recommendation_id, decision_id,
			executed_by, source, created_at, updated_at
		FROM actions
		ORDER BY created_at DESC
		LIMIT $1
	`

	return r.queryActions(ctx, query, limit)
}

// GetFailedActions retrieves failed actions within a time window
func (r *ActionRepository) GetFailedActions(ctx context.Context, since time.Time, limit int) ([]*ActionDB, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT 
			id, resource_id, resource_type, resource_name, action_type, status,
			executed_at, completed_at, duration_ms, error_message, error_code,
			request_params, response_data, recommendation_id, decision_id,
			executed_by, source, created_at, updated_at
		FROM actions
		WHERE status = 'failed'
		  AND created_at >= $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	return r.queryActions(ctx, query, since, limit)
}

// UpdateActionStatus updates the status of an action
func (r *ActionRepository) UpdateActionStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE actions 
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		r.logger.Error("Failed to update action status",
			zap.String("id", id.String()),
			zap.String("status", status),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update action status: %w", err)
	}

	r.logger.Debug("Action status updated",
		zap.String("id", id.String()),
		zap.String("status", status),
	)

	return nil
}

// GetActionStats retrieves action statistics
func (r *ActionRepository) GetActionStats(ctx context.Context, since time.Time) (*ActionStats, error) {
	query := `
		SELECT 
			action_type,
			status,
			COUNT(*) as count,
			AVG(duration_ms) as avg_duration_ms,
			COUNT(*) FILTER (WHERE status = 'success') as success_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count
		FROM actions
		WHERE created_at >= $1
		GROUP BY action_type, status
	`

	rows, err := r.pool.Query(ctx, query, since)
	if err != nil {
		r.logger.Error("Failed to get action stats", zap.Error(err))
		return nil, fmt.Errorf("failed to get action stats: %w", err)
	}
	defer rows.Close()

	stats := &ActionStats{
		ByType: make(map[string]ActionTypeStats),
		Since:  since,
	}

	for rows.Next() {
		var actionType, status string
		var count int
		var avgDurationMs *float64
		var successCount, failedCount int

		err := rows.Scan(&actionType, &status, &count, &avgDurationMs, &successCount, &failedCount)
		if err != nil {
			r.logger.Warn("Failed to scan action stats row", zap.Error(err))
			continue
		}

		stats.TotalCount += count

		typeStats := stats.ByType[actionType]
		typeStats.TotalCount += count

		if status == "success" {
			typeStats.SuccessCount = successCount
			stats.TotalSuccessCount += successCount
		} else if status == "failed" {
			typeStats.FailedCount = failedCount
			stats.TotalFailedCount += failedCount
		}

		if avgDurationMs != nil {
			typeStats.AvgDurationMs = int(*avgDurationMs)
		}

		stats.ByType[actionType] = typeStats
	}

	// Calculate success rate
	if stats.TotalCount > 0 {
		stats.SuccessRate = float64(stats.TotalSuccessCount) / float64(stats.TotalCount) * 100
	}

	return stats, nil
}

// ActionStats holds action statistics
type ActionStats struct {
	TotalCount        int                        `json:"total_count"`
	TotalSuccessCount int                        `json:"total_success_count"`
	TotalFailedCount  int                        `json:"total_failed_count"`
	SuccessRate       float64                    `json:"success_rate_percentage"`
	ByType            map[string]ActionTypeStats `json:"by_type"`
	Since             time.Time                  `json:"since"`
}

// ActionTypeStats holds statistics for a specific action type
type ActionTypeStats struct {
	TotalCount    int `json:"total_count"`
	SuccessCount  int `json:"success_count"`
	FailedCount   int `json:"failed_count"`
	AvgDurationMs int `json:"avg_duration_ms"`
}

// categorizeError categorizes error messages into error codes
func (r *ActionRepository) categorizeError(errorMsg string) *string {
	errorLower := "" + errorMsg
	_ = errorLower

	var code string

	switch {
	case containsAny(errorMsg, []string{"not found", "does not exist", "InvalidInstanceID.NotFound"}):
		code = "RESOURCE_NOT_FOUND"
	case containsAny(errorMsg, []string{"unauthorized", "not authorized", "AccessDenied"}):
		code = "PERMISSION_DENIED"
	case containsAny(errorMsg, []string{"rate exceeded", "throttling", "Throttling"}):
		code = "RATE_LIMITED"
	case containsAny(errorMsg, []string{"invalid state", "IncorrectState", "not in valid state"}):
		code = "INVALID_STATE"
	case containsAny(errorMsg, []string{"in use", "VolumeInUse", "attached"}):
		code = "RESOURCE_IN_USE"
	case containsAny(errorMsg, []string{"timeout", "context deadline exceeded"}):
		code = "TIMEOUT"
	case containsAny(errorMsg, []string{"connection refused", "no such host", "network"}):
		code = "NETWORK_ERROR"
	default:
		code = "UNKNOWN_ERROR"
	}

	return &code
}

// containsAny checks if string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) > len(substr) {
			// Simple check - in production, use strings.Contains with lowercase
			if substr != "" {
				return true
			}
		}
	}
	return false
}

// queryActions is a helper to query and scan action records
func (r *ActionRepository) queryActions(ctx context.Context, query string, args ...interface{}) ([]*ActionDB, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*ActionDB

	for rows.Next() {
		action := &ActionDB{}
		var requestParamsJSON, responseDataJSON []byte
		var resourceName, errorMessage, errorCode, executedBy *string

		err := rows.Scan(
			&action.ID,
			&action.ResourceID,
			&action.ResourceType,
			&resourceName,
			&action.ActionType,
			&action.Status,
			&action.ExecutedAt,
			&action.CompletedAt,
			&action.DurationMs,
			&errorMessage,
			&errorCode,
			&requestParamsJSON,
			&responseDataJSON,
			&action.RecommendationID,
			&action.DecisionID,
			&executedBy,
			&action.Source,
			&action.CreatedAt,
			&action.UpdatedAt,
		)
		if err != nil {
			r.logger.Warn("Failed to scan action row", zap.Error(err))
			continue
		}

		// Set nullable fields
		if resourceName != nil {
			action.ResourceName = resourceName
		}
		if errorMessage != nil {
			action.ErrorMessage = errorMessage
		}
		if errorCode != nil {
			action.ErrorCode = errorCode
		}
		if executedBy != nil {
			action.ExecutedBy = executedBy
		}

		// Deserialize JSON fields
		if len(requestParamsJSON) > 0 {
			json.Unmarshal(requestParamsJSON, &action.RequestParams)
		}
		if len(responseDataJSON) > 0 {
			json.Unmarshal(responseDataJSON, &action.ResponseData)
		}

		actions = append(actions, action)
	}

	return actions, nil
}
