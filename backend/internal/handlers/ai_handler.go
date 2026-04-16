package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// AIHandler handles AI-related API requests
type AIHandler struct {
	aiService *services.AIService
	validator *services.AISafetyValidator
	logger    *logger.Logger
}

// NewAIHandler creates a new AI handler
func NewAIHandler(aiService *services.AIService, log *logger.Logger) *AIHandler {
	return &AIHandler{
		aiService: aiService,
		validator: services.NewAISafetyValidator(),
		logger:    log,
	}
}

// AnalyzeRequest represents a cost analysis request
type AnalyzeRequest struct {
	TotalCost       float64                   `json:"total_cost" binding:"required"`
	PreviousCost    float64                   `json:"previous_cost" binding:"required"`
	TopServices     []services.ServiceCost    `json:"top_services"`
	ResourceChanges []services.ResourceChange `json:"resource_changes"`
}

// AnalyzeResponse represents a cost analysis response
type AnalyzeResponse struct {
	Summary     string   `json:"summary"`
	CostDrivers []string `json:"cost_drivers"`
	Suggestions []string `json:"suggestions"`
	RiskLevel   string   `json:"risk_level"`
	AIEnabled   bool     `json:"ai_enabled"`
	Disclaimer  string   `json:"disclaimer"`
}

// ExplainRequest represents a recommendation explanation request
type ExplainRequest struct {
	ResourceID       string  `json:"resource_id" binding:"required"`
	ResourceType     string  `json:"resource_type" binding:"required"`
	Title            string  `json:"title" binding:"required"`
	Rationale        string  `json:"rationale"`
	EstimatedSavings float64 `json:"estimated_savings"`
	CurrentState     string  `json:"current_state"`
	ProposedState    string  `json:"proposed_state"`
	RiskLevel        string  `json:"risk_level"`
}

// ExplainResponse represents a recommendation explanation response
type ExplainResponse struct {
	Explanation      string `json:"explanation"`
	SavingsRationale string `json:"savings_rationale"`
	ImpactSummary    string `json:"impact_summary"`
	Confidence       string `json:"confidence"`
	AIEnabled        bool   `json:"ai_enabled"`
	Disclaimer       string `json:"disclaimer"`
}

// AnomalyRequest represents an anomaly detection request
type AnomalyRequest struct {
	CostData []services.DailyCost `json:"cost_data" binding:"required,min=2"`
}

// AnomalyResponse represents an anomaly detection response
type AnomalyResponse struct {
	IsAnomaly   bool     `json:"is_anomaly"`
	Confidence  float64  `json:"confidence"`
	Explanation string   `json:"explanation"`
	Factors     []string `json:"factors,omitempty"`
	AIEnabled   bool     `json:"ai_enabled"`
	Disclaimer  string   `json:"disclaimer"`
}

// Analyze handles POST /api/v1/ai/analyze
// @Summary Analyze cost data
// @Description Analyzes cost data and provides AI-generated insights
// @Tags AI
// @Accept json
// @Produce json
// @Param request body AnalyzeRequest true "Cost analysis request"
// @Success 200 {object} AnalyzeResponse
// @Failure 400 {object} ErrorResponse
// @Router /ai/analyze [post]
func (h *AIHandler) Analyze(c *gin.Context) {
	start := time.Now()

	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid analyze request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Convert to service input
	input := services.CostAnalysisInput{
		TotalCost:       req.TotalCost,
		PreviousCost:    req.PreviousCost,
		TopServices:     req.TopServices,
		ResourceChanges: req.ResourceChanges,
	}

	// Call AI service with timeout context
	ctx := c.Request.Context()
	result, err := h.aiService.AnalyzeCost(ctx, input)
	if err != nil {
		h.logger.Error("AI analysis failed", zap.Error(err))
		// Return fallback response instead of error
		result = h.fallbackAnalysis(input)
	}

	// Validate and sanitize output
	validatedSummary := h.validator.ValidateOutput(result.Summary)
	if !validatedSummary.IsValid {
		h.logger.Warn("AI output failed validation",
			zap.Strings("violations", validatedSummary.Violations))
	}

	response := AnalyzeResponse{
		Summary:     validatedSummary.Sanitized,
		CostDrivers: result.CostDrivers,
		Suggestions: result.Suggestions,
		RiskLevel:   result.RiskLevel,
		AIEnabled:   h.aiService.IsEnabled(),
		Disclaimer:  services.APIDisclaimer,
	}

	h.logger.Info("Cost analysis completed",
		zap.Duration("duration", time.Since(start)),
		zap.Bool("ai_enabled", h.aiService.IsEnabled()),
	)

	c.JSON(http.StatusOK, response)
}

// Explain handles POST /api/v1/ai/explain
// @Summary Explain a recommendation
// @Description Provides an AI-generated explanation for a recommendation
// @Tags AI
// @Accept json
// @Produce json
// @Param request body ExplainRequest true "Explanation request"
// @Success 200 {object} ExplainResponse
// @Failure 400 {object} ErrorResponse
// @Router /ai/explain [post]
func (h *AIHandler) Explain(c *gin.Context) {
	start := time.Now()

	var req ExplainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid explain request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Convert to service input
	input := services.RecommendationInput{
		ResourceID:       req.ResourceID,
		ResourceType:     req.ResourceType,
		Title:            req.Title,
		Rationale:        req.Rationale,
		EstimatedSavings: req.EstimatedSavings,
		CurrentState:     req.CurrentState,
		ProposedState:    req.ProposedState,
		RiskLevel:        req.RiskLevel,
	}

	// Generate explanation (uses static templates first, falls back to AI)
	enhanced := services.GenerateExplanation(h.aiService, input, nil)

	// Validate output
	validatedExplanation := h.validator.ValidateOutput(enhanced.Explanation)
	if !validatedExplanation.IsValid {
		h.logger.Warn("Explanation failed validation",
			zap.Strings("violations", validatedExplanation.Violations))
	}

	response := ExplainResponse{
		Explanation:      validatedExplanation.Sanitized,
		SavingsRationale: enhanced.SavingsRationale,
		ImpactSummary:    enhanced.ImpactSummary,
		Confidence:       enhanced.Confidence,
		AIEnabled:        h.aiService.IsEnabled(),
		Disclaimer:       services.APIDisclaimer,
	}

	h.logger.Info("Recommendation explanation generated",
		zap.String("resource_id", req.ResourceID),
		zap.Duration("duration", time.Since(start)),
	)

	c.JSON(http.StatusOK, response)
}

// DetectAnomaly handles POST /api/v1/ai/anomaly
// @Summary Detect cost anomalies
// @Description Analyzes cost data for anomalies
// @Tags AI
// @Accept json
// @Produce json
// @Param request body AnomalyRequest true "Anomaly detection request"
// @Success 200 {object} AnomalyResponse
// @Failure 400 {object} ErrorResponse
// @Router /ai/anomaly [post]
func (h *AIHandler) DetectAnomaly(c *gin.Context) {
	var req AnomalyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid anomaly request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	result, err := h.aiService.DetectAnomaly(ctx, req.CostData)
	if err != nil {
		h.logger.Error("Anomaly detection failed", zap.Error(err))
	}

	// Validate explanation
	validatedExplanation := h.validator.ValidateOutput(result.Explanation)

	response := AnomalyResponse{
		IsAnomaly:   result.IsAnomaly,
		Confidence:  result.Confidence,
		Explanation: validatedExplanation.Sanitized,
		Factors:     result.Factors,
		AIEnabled:   h.aiService.IsEnabled(),
		Disclaimer:  services.APIDisclaimer,
	}

	c.JSON(http.StatusOK, response)
}

// Health handles GET /api/v1/ai/health
func (h *AIHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	err := h.aiService.Health(ctx)

	status := "healthy"
	if err != nil {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     status,
		"ai_enabled": h.aiService.IsEnabled(),
		"error":      errorToString(err),
	})
}

func (h *AIHandler) fallbackAnalysis(input services.CostAnalysisInput) *services.CostAnalysisResult {
	change := ((input.TotalCost - input.PreviousCost) / input.PreviousCost) * 100

	result := &services.CostAnalysisResult{
		RiskLevel:   "low",
		CostDrivers: []string{},
		Suggestions: []string{"Review cost data for optimization opportunities"},
	}

	if change > 20 {
		result.Summary = "Significant cost increase detected. Review top services."
		result.RiskLevel = "high"
	} else if change > 10 {
		result.Summary = "Moderate cost increase. Monitor closely."
		result.RiskLevel = "medium"
	} else {
		result.Summary = "Costs are within normal range."
	}

	return result
}

func errorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
