package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// WasteHandler handles API requests for waste detection
type WasteHandler struct {
	wasteService *services.WasteDetectionService
	logger       *logger.Logger
}

// NewWasteHandler creates a new waste handler
func NewWasteHandler(wasteService *services.WasteDetectionService, logger *logger.Logger) *WasteHandler {
	return &WasteHandler{
		wasteService: wasteService,
		logger:       logger,
	}
}

// WasteResponse represents a single waste resource in the API response
type WasteResponse struct {
	ResourceID       string  `json:"resource_id"`
	ResourceType     string  `json:"resource_type"`
	ResourceName     string  `json:"resource_name,omitempty"`
	WasteType        string  `json:"type"`
	Reason           string  `json:"reason"`
	Severity         string  `json:"severity"`
	EstimatedSavings float64 `json:"estimated_savings_usd"`
	Confidence       float64 `json:"confidence"`
}

// WasteListResponse represents the response for GET /waste
type WasteListResponse struct {
	Success        bool            `json:"success"`
	Count          int             `json:"count"`
	Waste          []WasteResponse `json:"waste"`
	TotalSavings   float64         `json:"total_estimated_savings_usd"`
	HighPriority   int             `json:"high_priority_count"`
	MediumPriority int             `json:"medium_priority_count"`
	LowPriority    int             `json:"low_priority_count"`
}

// GetWaste returns detected waste resources
// @Summary      Get waste resources
// @Description  Returns detected waste resources with severity and savings estimates
// @Tags         waste
// @Accept       json
// @Produce      json
// @Param        severity    query   string  false  "Filter by severity (critical, high, medium, low)"
// @Param        type        query   string  false  "Filter by waste type (idle_ec2, unattached_ebs, underutilized_rds)"
// @Param        limit       query   int     false  "Limit number of results (default: 50)"
// @Success      200  {object}  WasteListResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /waste [get]
func (h *WasteHandler) GetWaste(c *gin.Context) {
	h.logger.Debug("Processing GET /waste request",
		zap.String("severity_filter", c.Query("severity")),
		zap.String("type_filter", c.Query("type")),
	)

	// Get filters from query parameters
	severityFilter := c.Query("severity")
	typeFilter := c.Query("type")

	// Parse limit with default
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	// Get waste detection results
	// In production, this would come from the repository
	// For now, we'll assume the waste service returns cached results
	ctx := c.Request.Context()
	result, err := h.wasteService.DetectWaste(ctx)
	if err != nil {
		h.logger.Error("Failed to detect waste", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve waste detection results",
		})
		return
	}

	// Transform to response format
	var wasteResponses []WasteResponse
	var totalSavings float64
	var highCount, mediumCount, lowCount int

	for _, w := range result.WasteResources {
		// Apply filters
		if severityFilter != "" && string(w.Severity) != severityFilter {
			continue
		}
		if typeFilter != "" && string(w.WasteType) != typeFilter {
			continue
		}

		// Count by priority
		switch w.Severity {
		case services.WasteSeverityCritical, services.WasteSeverityHigh:
			highCount++
		case services.WasteSeverityMedium:
			mediumCount++
		case services.WasteSeverityLow:
			lowCount++
		}

		wasteResponses = append(wasteResponses, WasteResponse{
			ResourceID:       w.ResourceID,
			ResourceType:     w.ResourceType,
			ResourceName:     w.ResourceName,
			WasteType:        string(w.WasteType),
			Reason:           w.Reason,
			Severity:         string(w.Severity),
			EstimatedSavings: w.EstimatedSavings,
			Confidence:       w.Confidence,
		})

		totalSavings += w.EstimatedSavings

		// Apply limit
		if len(wasteResponses) >= limit {
			break
		}
	}

	response := WasteListResponse{
		Success:        true,
		Count:          len(wasteResponses),
		Waste:          wasteResponses,
		TotalSavings:   totalSavings,
		HighPriority:   highCount,
		MediumPriority: mediumCount,
		LowPriority:    lowCount,
	}

	c.JSON(http.StatusOK, response)
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// SuccessResponse represents a simple success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}
