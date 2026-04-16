package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// RecommendationsHandler handles API requests for recommendations
type RecommendationsHandler struct {
	recommendationService *services.RecommendationService
	logger                *logger.Logger
}

// NewRecommendationsHandler creates a new recommendations handler
func NewRecommendationsHandler(recommendationService *services.RecommendationService, logger *logger.Logger) *RecommendationsHandler {
	return &RecommendationsHandler{
		recommendationService: recommendationService,
		logger:                logger,
	}
}

// RecommendationResponse represents a single recommendation in the API response
type RecommendationResponse struct {
	ID                  string   `json:"id"`
	ResourceID          string   `json:"resource_id"`
	ResourceType        string   `json:"resource_type"`
	ResourceName        string   `json:"resource_name,omitempty"`
	Type                string   `json:"type"` // stop, delete, resize, schedule
	Title               string   `json:"title"`
	Reason              string   `json:"reason"`
	Description         string   `json:"description"`
	EstimatedSavings    float64  `json:"estimated_savings_usd"`
	Priority            string   `json:"priority"`
	RiskLevel           string   `json:"risk_level"`
	Status              string   `json:"status"`
	ImplementationSteps []string `json:"implementation_steps,omitempty"`
}

// RecommendationsListResponse represents the response for GET /recommendations
type RecommendationsListResponse struct {
	Success         bool                     `json:"success"`
	Count           int                      `json:"count"`
	Recommendations []RecommendationResponse `json:"recommendations"`
	TotalSavings    float64                  `json:"total_estimated_savings_usd"`
	CriticalCount   int                      `json:"critical_count"`
	HighCount       int                      `json:"high_count"`
	MediumCount     int                      `json:"medium_count"`
	LowCount        int                      `json:"low_count"`
}

// GetRecommendations returns optimization recommendations
// @Summary      Get optimization recommendations
// @Description  Returns cost optimization recommendations with savings estimates
// @Tags         recommendations
// @Accept       json
// @Produce      json
// @Param        status      query   string  false  "Filter by status (active, pending, accepted, implemented)"
// @Param        priority    query   string  false  "Filter by priority (critical, high, medium, low)"
// @Param        type        query   string  false  "Filter by type (stop, delete, resize, schedule)"
// @Param        resource_type query string  false  "Filter by resource type (EC2, EBS, RDS)"
// @Param        limit       query   int     false  "Limit number of results (default: 50, max: 1000)"
// @Success      200  {object}  RecommendationsListResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /recommendations [get]
func (h *RecommendationsHandler) GetRecommendations(c *gin.Context) {
	h.logger.Debug("Processing GET /recommendations request",
		zap.String("status_filter", c.Query("status")),
		zap.String("priority_filter", c.Query("priority")),
		zap.String("type_filter", c.Query("type")),
	)

	// Get filters from query parameters
	statusFilter := c.Query("status")
	priorityFilter := c.Query("priority")
	typeFilter := c.Query("type")
	resourceTypeFilter := c.Query("resource_type")

	// Parse limit with default and validation
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	ctx := c.Request.Context()

	// Get recommendations from service
	var recommendations []*services.Recommendation
	var err error

	// If status filter is provided, use it
	if statusFilter != "" {
		status := services.RecommendationStatus(statusFilter)
		recommendations, err = h.recommendationService.GetRecommendationsByStatus(ctx, status)
	} else {
		// Default to active recommendations
		recommendations, err = h.recommendationService.GetActiveRecommendations(ctx)
	}

	if err != nil {
		h.logger.Error("Failed to get recommendations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve recommendations",
		})
		return
	}

	// Transform to response format
	var recResponses []RecommendationResponse
	var totalSavings float64
	var criticalCount, highCount, mediumCount, lowCount int

	for _, rec := range recommendations {
		// Apply filters
		if priorityFilter != "" && string(rec.Priority) != priorityFilter {
			continue
		}
		if typeFilter != "" && string(rec.RecommendationType) != typeFilter {
			continue
		}
		if resourceTypeFilter != "" && rec.ResourceType != resourceTypeFilter {
			continue
		}

		// Count by priority
		switch rec.Priority {
		case services.RecommendationPriorityCritical:
			criticalCount++
		case services.RecommendationPriorityHigh:
			highCount++
		case services.RecommendationPriorityMedium:
			mediumCount++
		case services.RecommendationPriorityLow:
			lowCount++
		}

		// Build reason from description and rationale
		reason := rec.Description
		if len(reason) > 200 {
			reason = reason[:200] + "..."
		}

		recResponses = append(recResponses, RecommendationResponse{
			ID:                  rec.ID.String(),
			ResourceID:          rec.ResourceID,
			ResourceType:        rec.ResourceType,
			ResourceName:        rec.ResourceName,
			Type:                string(rec.RecommendationType),
			Title:               rec.Title,
			Reason:              reason,
			Description:         rec.Description,
			EstimatedSavings:    rec.EstimatedSavings,
			Priority:            string(rec.Priority),
			RiskLevel:           rec.RiskLevel,
			Status:              string(rec.Status),
			ImplementationSteps: rec.ImplementationSteps,
		})

		totalSavings += rec.EstimatedSavings

		// Apply limit
		if len(recResponses) >= limit {
			break
		}
	}

	response := RecommendationsListResponse{
		Success:         true,
		Count:           len(recResponses),
		Recommendations: recResponses,
		TotalSavings:    totalSavings,
		CriticalCount:   criticalCount,
		HighCount:       highCount,
		MediumCount:     mediumCount,
		LowCount:        lowCount,
	}

	c.JSON(http.StatusOK, response)
}

// GetRecommendationByID returns a single recommendation by ID
// @Summary      Get recommendation by ID
// @Description  Returns a single recommendation with full details
// @Tags         recommendations
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Recommendation ID"
// @Success      200  {object}  RecommendationDetailResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /recommendations/{id} [get]
func (h *RecommendationsHandler) GetRecommendationByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Recommendation ID is required",
		})
		return
	}

	// Parse UUID
	recID, err := parseUUID(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid recommendation ID format",
		})
		return
	}

	ctx := c.Request.Context()
	rec, err := h.recommendationService.GetRecommendationsByResource(ctx, recID.String())
	if err != nil || len(rec) == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error:   "Recommendation not found",
		})
		return
	}

	// Return the first matching recommendation (should be unique by ID)
	recommendation := rec[0]

	// Build reason
	reason := recommendation.Description
	if len(reason) > 200 {
		reason = reason[:200] + "..."
	}

	response := RecommendationDetailResponse{
		Success: true,
		Data: DetailedRecommendationResponse{
			ID:                  recommendation.ID.String(),
			ResourceID:          recommendation.ResourceID,
			ResourceType:        recommendation.ResourceType,
			ResourceName:        recommendation.ResourceName,
			Type:                string(recommendation.RecommendationType),
			Title:               recommendation.Title,
			Reason:              reason,
			Description:         recommendation.Description,
			Rationale:           recommendation.Rationale,
			EstimatedSavings:    recommendation.EstimatedSavings,
			Priority:            string(recommendation.Priority),
			RiskLevel:           recommendation.RiskLevel,
			Status:              string(recommendation.Status),
			ImplementationSteps: recommendation.ImplementationSteps,
			Alternatives:        recommendation.Alternatives,
			CurrentState:        recommendation.CurrentState,
			ProposedState:       recommendation.ProposedState,
			CreatedAt:           recommendation.CreatedAt,
		},
	}

	c.JSON(http.StatusOK, response)
}

// RecommendationDetailResponse represents the detailed response
type RecommendationDetailResponse struct {
	Success bool                           `json:"success"`
	Data    DetailedRecommendationResponse `json:"data"`
}

// DetailedRecommendationResponse includes full details
type DetailedRecommendationResponse struct {
	ID                  string                  `json:"id"`
	ResourceID          string                  `json:"resource_id"`
	ResourceType        string                  `json:"resource_type"`
	ResourceName        string                  `json:"resource_name,omitempty"`
	Type                string                  `json:"type"`
	Title               string                  `json:"title"`
	Reason              string                  `json:"reason"`
	Description         string                  `json:"description"`
	Rationale           string                  `json:"rationale"`
	EstimatedSavings    float64                 `json:"estimated_savings_usd"`
	Priority            string                  `json:"priority"`
	RiskLevel           string                  `json:"risk_level"`
	Status              string                  `json:"status"`
	ImplementationSteps []string                `json:"implementation_steps"`
	Alternatives        []string                `json:"alternatives,omitempty"`
	CurrentState        *services.ResourceState `json:"current_state,omitempty"`
	ProposedState       *services.ResourceState `json:"proposed_state,omitempty"`
	CreatedAt           interface{}             `json:"created_at"`
}
