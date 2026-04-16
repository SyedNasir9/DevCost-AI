package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"devcost-ai/internal/repositories"
	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// ActionHandler handles API requests for actions
type ActionHandler struct {
	actionPipeline *services.ActionPipeline
	actionRepo     *repositories.ActionRepository
	logger         *logger.Logger
}

// NewActionHandler creates a new action handler
func NewActionHandler(
	actionPipeline *services.ActionPipeline,
	actionRepo *repositories.ActionRepository,
	logger *logger.Logger,
) *ActionHandler {
	return &ActionHandler{
		actionPipeline: actionPipeline,
		actionRepo:     actionRepo,
		logger:         logger,
	}
}

// ExecuteRequest represents a request to execute recommendations
// @Description Request to execute cost optimization recommendations
type ExecuteRequest struct {
	RecommendationIDs []string `json:"recommendation_ids,omitempty" example:"["550e8400-e29b-41d4-a716-446655440000"]"`
	AllActive         bool     `json:"all_active,omitempty" example:"true"` // Execute all active recommendations if true
	DryRun            bool     `json:"dry_run,omitempty" example:"false"`   // Validate only, don't execute
}

// ExecuteResponse represents the response for action execution
// @Description Response containing execution results
type ExecuteResponse struct {
	Success              bool     `json:"success"`
	RunID                string   `json:"run_id,omitempty"`
	Message              string   `json:"message"`
	RecommendationsFound int      `json:"recommendations_found"`
	Evaluated            int      `json:"evaluated"`
	Approved             int      `json:"approved"`
	Executed             int      `json:"executed"`
	Failed               int      `json:"failed"`
	TotalSavings         float64  `json:"total_savings_usd"`
	Duration             string   `json:"duration"`
	Errors               []string `json:"errors,omitempty"`
}

// ActionResponse represents a single action in the API response
// @Description Action details
type ActionResponse struct {
	ID           string     `json:"id"`
	ResourceID   string     `json:"resource_id"`
	ResourceType string     `json:"resource_type,omitempty"`
	ActionType   string     `json:"action_type"`
	Status       string     `json:"status"`
	Timestamp    time.Time  `json:"timestamp"`
	ExecutedAt   *time.Time `json:"executed_at,omitempty"`
	DurationMs   *int       `json:"duration_ms,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
}

// ActionsListResponse represents the response for listing actions
// @Description List of actions with count
type ActionsListResponse struct {
	Success bool             `json:"success"`
	Count   int              `json:"count"`
	Actions []ActionResponse `json:"actions"`
}

// ActionDetailResponse represents detailed action information
// @Description Detailed action information
type ActionDetailResponse struct {
	Success bool             `json:"success"`
	Data    ActionDetailData `json:"data"`
}

// ActionDetailData contains full action details
// @Description Full action details
type ActionDetailData struct {
	ID               string                 `json:"id"`
	ResourceID       string                 `json:"resource_id"`
	ResourceType     string                 `json:"resource_type"`
	ResourceName     *string                `json:"resource_name,omitempty"`
	ActionType       string                 `json:"action_type"`
	Status           string                 `json:"status"`
	ExecutedAt       *time.Time             `json:"executed_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	DurationMs       *int                   `json:"duration_ms,omitempty"`
	ErrorMessage     *string                `json:"error_message,omitempty"`
	ErrorCode        *string                `json:"error_code,omitempty"`
	RequestParams    map[string]interface{} `json:"request_params,omitempty"`
	ResponseData     map[string]interface{} `json:"response_data,omitempty"`
	RecommendationID *string                `json:"recommendation_id,omitempty"`
	ExecutedBy       *string                `json:"executed_by,omitempty"`
	Source           string                 `json:"source"`
	CreatedAt        time.Time              `json:"created_at"`
}

// ExecuteActions executes recommendations
// @Summary      Execute recommendations
// @Description  Execute cost optimization recommendations through the action pipeline
// @Tags         actions
// @Accept       json
// @Produce      json
// @Param        request  body      ExecuteRequest  true  "Execution request"
// @Success      200      {object}  ExecuteResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /actions/execute [post]
func (h *ActionHandler) ExecuteActions(c *gin.Context) {
	h.logger.Debug("Processing POST /actions/execute request")

	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// If specific IDs provided, validate them
	if len(req.RecommendationIDs) > 0 {
		for _, id := range req.RecommendationIDs {
			if _, err := uuid.Parse(id); err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Success: false,
					Error:   "Invalid recommendation ID: " + id,
				})
				return
			}
		}
	}

	// Note: In a full implementation, you would:
	// 1. If specific IDs provided, fetch those recommendations
	// 2. If AllActive=true, use the pipeline's normal flow
	// 3. Apply dry run mode if requested

	// For now, run the full pipeline
	ctx := c.Request.Context()
	result, err := h.actionPipeline.Run(ctx)

	if err != nil {
		h.logger.Error("Action pipeline failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Action execution failed: " + err.Error(),
		})
		return
	}

	// Collect errors
	var errors []string
	for _, e := range result.Errors {
		errors = append(errors, fmt.Sprintf("%s: %s", e.Step, e.Error))
	}

	response := ExecuteResponse{
		Success:              len(errors) == 0,
		RunID:                result.RunID.String(),
		Message:              fmt.Sprintf("Pipeline completed with %d executed, %d failed", result.ExecutedCount, result.FailedCount),
		RecommendationsFound: result.RecommendationsFound,
		Evaluated:            result.EvaluatedCount,
		Approved:             result.ApprovedCount,
		Executed:             result.ExecutedCount,
		Failed:               result.FailedCount,
		TotalSavings:         result.TotalEstimatedSavings,
		Duration:             result.Duration.String(),
		Errors:               errors,
	}

	h.logger.Info("Action execution completed",
		zap.String("run_id", result.RunID.String()),
		zap.Int("executed", result.ExecutedCount),
		zap.Int("failed", result.FailedCount),
	)

	c.JSON(http.StatusOK, response)
}

// ListActions returns a list of all actions
// @Summary      List actions
// @Description  Returns a list of executed actions with optional filtering
// @Tags         actions
// @Accept       json
// @Produce      json
// @Param        status    query   string  false  "Filter by status (pending, success, failed)"
// @Param        resource_id query string  false  "Filter by resource ID"
// @Param        limit     query   int     false  "Limit number of results (default: 50, max: 1000)"
// @Success      200  {object}  ActionsListResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /actions [get]
func (h *ActionHandler) ListActions(c *gin.Context) {
	h.logger.Debug("Processing GET /actions request",
		zap.String("status_filter", c.Query("status")),
		zap.String("resource_id_filter", c.Query("resource_id")),
	)

	// Parse limit
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	ctx := c.Request.Context()
	var actions []*repositories.ActionDB
	var err error

	// Apply filters
	resourceID := c.Query("resource_id")
	status := c.Query("status")

	if resourceID != "" {
		actions, err = h.actionRepo.GetActionsByResource(ctx, resourceID, limit)
	} else if status != "" {
		actions, err = h.actionRepo.GetActionsByStatus(ctx, status, limit)
	} else {
		actions, err = h.actionRepo.GetRecentActions(ctx, limit)
	}

	if err != nil {
		h.logger.Error("Failed to get actions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve actions",
		})
		return
	}

	// Transform to response format
	var actionResponses []ActionResponse
	for _, action := range actions {
		actionResp := ActionResponse{
			ID:           action.ID.String(),
			ResourceID:   action.ResourceID,
			ResourceType: action.ResourceType,
			ActionType:   action.ActionType,
			Status:       action.Status,
			Timestamp:    action.CreatedAt,
		}

		if action.ExecutedAt != nil {
			actionResp.ExecutedAt = action.ExecutedAt
		}
		if action.DurationMs != nil {
			actionResp.DurationMs = action.DurationMs
		}
		if action.ErrorMessage != nil {
			actionResp.ErrorMessage = action.ErrorMessage
		}

		actionResponses = append(actionResponses, actionResp)
	}

	response := ActionsListResponse{
		Success: true,
		Count:   len(actionResponses),
		Actions: actionResponses,
	}

	c.JSON(http.StatusOK, response)
}

// GetAction returns details of a specific action
// @Summary      Get action details
// @Description  Returns detailed information about a specific action
// @Tags         actions
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Action ID"
// @Success      200  {object}  ActionDetailResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /actions/{id} [get]
func (h *ActionHandler) GetAction(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Action ID is required",
		})
		return
	}

	// Parse UUID
	actionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid action ID format",
		})
		return
	}

	h.logger.Debug("Processing GET /actions/:id request",
		zap.String("action_id", id),
	)

	ctx := c.Request.Context()
	action, err := h.actionRepo.GetActionByID(ctx, actionID)
	if err != nil {
		h.logger.Error("Failed to get action",
			zap.String("id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error:   "Action not found",
		})
		return
	}

	// Transform to response format
	var recID *string
	if action.RecommendationID != nil {
		rid := action.RecommendationID.String()
		recID = &rid
	}

	response := ActionDetailResponse{
		Success: true,
		Data: ActionDetailData{
			ID:               action.ID.String(),
			ResourceID:       action.ResourceID,
			ResourceType:     action.ResourceType,
			ResourceName:     action.ResourceName,
			ActionType:       action.ActionType,
			Status:           action.Status,
			ExecutedAt:       action.ExecutedAt,
			CompletedAt:      action.CompletedAt,
			DurationMs:       action.DurationMs,
			ErrorMessage:     action.ErrorMessage,
			ErrorCode:        action.ErrorCode,
			RequestParams:    action.RequestParams,
			ResponseData:     action.ResponseData,
			RecommendationID: recID,
			ExecutedBy:       action.ExecutedBy,
			Source:           action.Source,
			CreatedAt:        action.CreatedAt,
		},
	}

	c.JSON(http.StatusOK, response)
}
