package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devcost-ai/pkg/logger"
)

// SimulationService provides simulation of recommendation execution
type SimulationService struct {
	logger              *logger.Logger
	recommendationRepo  interface {
		GetActiveRecommendations(ctx context.Context) ([]*Recommendation, error)
		GetRecommendationsByResource(ctx context.Context, resourceID string) ([]*Recommendation, error)
	}
	executionController *ExecutionController
}

// SimulationConfig holds configuration for simulation
type SimulationConfig struct {
	// Limit simulation to first N recommendations (0 = no limit)
	MaxRecommendations int
	
	// Only simulate recommendations with savings above threshold
	MinSavingsThreshold float64
	
	// Include implementation steps in output
	IncludeImplementationSteps bool
	
	// Include alternatives in output
	IncludeAlternatives bool
	
	// Time range for cost analysis
	CostAnalysisWindow time.Duration
}

// DefaultSimulationConfig returns default configuration
func DefaultSimulationConfig() *SimulationConfig {
	return &SimulationConfig{
		MaxRecommendations:         100,
		MinSavingsThreshold:        0.0,
		IncludeImplementationSteps: true,
		IncludeAlternatives:        true,
		CostAnalysisWindow:         7 * 24 * time.Hour,
	}
}

// SimulationResult contains the results of a simulation
type SimulationResult struct {
	SimulationID        uuid.UUID              `json:"simulation_id"`
	Timestamp            time.Time              `json:"timestamp"`
	Duration             time.Duration          `json:"duration"`
	
	// Summary counts
	TotalRecommendations int                    `json:"total_recommendations"`
	SimulatedCount       int                    `json:"simulated_count"`
	ApprovedCount        int                    `json:"approved_count"`
	RejectedCount        int                    `json:"rejected_count"`
	PendingCount         int                    `json:"pending_count"`
	
	// Financial impact
	TotalEstimatedSavings float64               `json:"total_estimated_savings_usd"`
	MonthlySavings      float64               `json:"monthly_savings_usd"`
	AnnualSavings       float64               `json:"annual_savings_usd"`
	
	// Resource impact
	ResourcesAffected     int                   `json:"resources_affected"`
	ResourcesByType       map[string]int        `json:"resources_by_type"`
	
	// Actions
	Actions              []SimulatedAction      `json:"actions"`
	ActionsByType        map[string]int         `json:"actions_by_type"`
	
	// Risk assessment
	HighRiskCount        int                   `json:"high_risk_count"`
	MediumRiskCount      int                   `json:"medium_risk_count"`
	LowRiskCount         int                   `json:"low_risk_count"`
	
	// Implementation timeline
	EstimatedImplementationTime string         `json:"estimated_implementation_time"`
}

// SimulatedAction represents a single action in the simulation
type SimulatedAction struct {
	ActionID            uuid.UUID              `json:"action_id"`
	RecommendationID    uuid.UUID              `json:"recommendation_id"`
	ResourceID          string                 `json:"resource_id"`
	ResourceType        string                 `json:"resource_type"`
	ResourceName        string                 `json:"resource_name,omitempty"`
	ActionType          string                 `json:"action_type"`
	Status              string                 `json:"status"` // approved, rejected, pending
	
	// Financial impact
	EstimatedSavings    float64                `json:"estimated_savings_usd"`
	SavingsPercentage float64                `json:"savings_percentage,omitempty"`
	
	// Risk assessment
	RiskLevel           string                 `json:"risk_level"`
	Priority            string                 `json:"priority"`
	
	// Implementation details
	ImplementationSteps []string             `json:"implementation_steps,omitempty"`
	Alternatives        []string             `json:"alternatives,omitempty"`
	EstimatedDuration   string               `json:"estimated_duration,omitempty"`
	
	// Decision details
	DecisionReason      string               `json:"decision_reason"`
	Warnings            []string             `json:"warnings,omitempty"`
}

// NewSimulationService creates a new simulation service
func NewSimulationService(
	logger *logger.Logger,
	recommendationRepo interface {
		GetActiveRecommendations(ctx context.Context) ([]*Recommendation, error)
		GetRecommendationsByResource(ctx context.Context, resourceID string) ([]*Recommendation, error)
	},
	executionController *ExecutionController,
) *SimulationService {
	return &SimulationService{
		logger:              logger,
		recommendationRepo:  recommendationRepo,
		executionController: executionController,
	}
}

// SimulateAll runs simulation on all active recommendations
func (s *SimulationService) SimulateAll(ctx context.Context, config *SimulationConfig) (*SimulationResult, error) {
	startTime := time.Now()
	simulationID := uuid.New()
	
	if config == nil {
		config = DefaultSimulationConfig()
	}
	
	s.logger.Info("Starting simulation",
		zap.String("simulation_id", simulationID.String()),
		zap.Int("max_recommendations", config.MaxRecommendations),
		zap.Float64("min_savings", config.MinSavingsThreshold),
	)
	
	// Get active recommendations
	recommendations, err := s.recommendationRepo.GetActiveRecommendations(ctx)
	if err != nil {
		s.logger.Error("Failed to get recommendations", zap.Error(err))
		return nil, fmt.Errorf("failed to get recommendations: %w", err)
	}
	
	s.logger.Info("Retrieved recommendations for simulation",
		zap.String("simulation_id", simulationID.String()),
		zap.Int("count", len(recommendations)),
	)
	
	// Filter recommendations
	filtered := s.filterRecommendations(recommendations, config)
	
	// Run simulation
	result := s.runSimulation(ctx, simulationID, filtered, config)
	result.Duration = time.Since(startTime)
	
	s.logger.Info("Simulation completed",
		zap.String("simulation_id", simulationID.String()),
		zap.Int("simulated", result.SimulatedCount),
		zap.Int("approved", result.ApprovedCount),
		zap.Float64("total_savings", result.TotalEstimatedSavings),
		zap.Duration("duration", result.Duration),
	)
	
	return result, nil
}

// SimulateRecommendations runs simulation on specific recommendations
func (s *SimulationService) SimulateRecommendations(
	ctx context.Context,
	recommendationIDs []uuid.UUID,
	config *SimulationConfig,
) (*SimulationResult, error) {
	startTime := time.Now()
	simulationID := uuid.New()
	
	if config == nil {
		config = DefaultSimulationConfig()
	}
	
	s.logger.Info("Starting simulation for specific recommendations",
		zap.String("simulation_id", simulationID.String()),
		zap.Int("recommendation_count", len(recommendationIDs)),
	)
	
	// Fetch recommendations by ID
	var recommendations []*Recommendation
	for _, id := range recommendationIDs {
		recs, err := s.recommendationRepo.GetRecommendationsByResource(ctx, id.String())
		if err != nil {
			s.logger.Warn("Failed to get recommendation",
				zap.String("id", id.String()),
				zap.Error(err),
			)
			continue
		}
		recommendations = append(recommendations, recs...)
	}
	
	// Filter recommendations
	filtered := s.filterRecommendations(recommendations, config)
	
	// Run simulation
	result := s.runSimulation(ctx, simulationID, filtered, config)
	result.Duration = time.Since(startTime)
	
	return result, nil
}

// SimulateResource runs simulation for a specific resource
func (s *SimulationService) SimulateResource(
	ctx context.Context,
	resourceID string,
	config *SimulationConfig,
) (*SimulationResult, error) {
	startTime := time.Now()
	simulationID := uuid.New()
	
	if config == nil {
		config = DefaultSimulationConfig()
	}
	
	s.logger.Info("Starting simulation for resource",
		zap.String("simulation_id", simulationID.String()),
		zap.String("resource_id", resourceID),
	)
	
	// Get recommendations for resource
	recommendations, err := s.recommendationRepo.GetRecommendationsByResource(ctx, resourceID)
	if err != nil {
		s.logger.Error("Failed to get recommendations for resource", zap.Error(err))
		return nil, fmt.Errorf("failed to get recommendations: %w", err)
	}
	
	// Filter recommendations
	filtered := s.filterRecommendations(recommendations, config)
	
	// Run simulation
	result := s.runSimulation(ctx, simulationID, filtered, config)
	result.Duration = time.Since(startTime)
	
	return result, nil
}

// filterRecommendations filters recommendations based on config
func (s *SimulationService) filterRecommendations(
	recommendations []*Recommendation,
	config *SimulationConfig,
) []*Recommendation {
	var filtered []*Recommendation
	
	for _, rec := range recommendations {
		// Apply savings threshold
		if rec.EstimatedSavings < config.MinSavingsThreshold {
			continue
		}
		
		filtered = append(filtered, rec)
	}
	
	// Apply max limit
	if config.MaxRecommendations > 0 && len(filtered) > config.MaxRecommendations {
		s.logger.Warn("Truncating recommendations to max limit",
			zap.Int("original", len(filtered)),
			zap.Int("max", config.MaxRecommendations),
		)
		filtered = filtered[:config.MaxRecommendations]
	}
	
	return filtered
}

// runSimulation performs the actual simulation
func (s *SimulationService) runSimulation(
	ctx context.Context,
	simulationID uuid.UUID,
	recommendations []*Recommendation,
	config *SimulationConfig,
) *SimulationResult {
	result := &SimulationResult{
		SimulationID:         simulationID,
		Timestamp:            time.Now(),
		TotalRecommendations: len(recommendations),
		ResourcesByType:      make(map[string]int),
		ActionsByType:        make(map[string]int),
	}
	
	// Track unique resources
	resourceMap := make(map[string]bool)
	
	// Simulate each recommendation
	for _, rec := range recommendations {
		simulatedAction := s.simulateAction(ctx, rec, config)
		result.Actions = append(result.Actions, simulatedAction)
		result.SimulatedCount++
		
		// Update counts based on decision
		switch simulatedAction.Status {
		case "approved":
			result.ApprovedCount++
			result.TotalEstimatedSavings += simulatedAction.EstimatedSavings
		case "rejected":
			result.RejectedCount++
		case "pending":
			result.PendingCount++
		}
		
		// Update resource tracking
		resourceMap[rec.ResourceID] = true
		result.ResourcesByType[rec.ResourceType]++
		
		// Update action type tracking
		result.ActionsByType[simulatedAction.ActionType]++
		
		// Update risk counts
		switch rec.RiskLevel {
		case "high":
			result.HighRiskCount++
		case "medium":
			result.MediumRiskCount++
		case "low":
			result.LowRiskCount++
		}
	}
	
	// Calculate totals
	result.ResourcesAffected = len(resourceMap)
	result.MonthlySavings = result.TotalEstimatedSavings
	result.AnnualSavings = result.TotalEstimatedSavings * 12
	
	// Estimate implementation time
	totalMinutes := result.ApprovedCount * 5 // Assume 5 minutes per action on average
	result.EstimatedImplementationTime = formatDuration(totalMinutes)
	
	return result
}

// simulateAction simulates a single action
func (s *SimulationService) simulateAction(
	ctx context.Context,
	rec *Recommendation,
	config *SimulationConfig,
) SimulatedAction {
	action := SimulatedAction{
		ActionID:         uuid.New(),
		RecommendationID: rec.ID,
		ResourceID:       rec.ResourceID,
		ResourceType:     rec.ResourceType,
		ResourceName:     rec.ResourceName,
		ActionType:       string(rec.RecommendationType),
		EstimatedSavings: rec.EstimatedSavings,
		RiskLevel:        rec.RiskLevel,
		Priority:         string(rec.Priority),
	}
	
	// Get decision from execution controller
	decision, err := s.executionController.EvaluateRecommendation(ctx, rec)
	if err != nil {
		s.logger.Warn("Failed to evaluate recommendation in simulation",
			zap.String("recommendation_id", rec.ID.String()),
			zap.Error(err),
		)
		action.Status = "pending"
		action.DecisionReason = fmt.Sprintf("Evaluation error: %v", err)
		action.Warnings = append(action.Warnings, "Could not fully evaluate recommendation")
	} else {
		action.Status = decision.Decision
		action.DecisionReason = decision.Reason
		
		// Add safety check warnings
		for _, check := range decision.SafetyChecks {
			if !check.Passed {
				action.Warnings = append(action.Warnings, check.Message)
			}
		}
	}
	
	// Add implementation details if configured
	if config.IncludeImplementationSteps {
		action.ImplementationSteps = rec.ImplementationSteps
	}
	
	if config.IncludeAlternatives {
		action.Alternatives = rec.Alternatives
	}
	
	// Estimate duration based on action type
	action.EstimatedDuration = estimateActionDuration(rec.RecommendationType)
	
	return action
}

// estimateActionDuration estimates how long an action takes
func estimateActionDuration(actionType RecommendationType) string {
	switch actionType {
	case RecommendationTypeStop:
		return "1-2 minutes"
	case RecommendationTypeDelete:
		return "2-3 minutes (includes snapshot)"
	case RecommendationTypeResize:
		return "5-10 minutes (may require downtime)"
	case RecommendationTypeSchedule:
		return "3-5 minutes"
	case RecommendationTypeSnapshot:
		return "1-2 minutes"
	default:
		return "2-5 minutes"
	}
}

// formatDuration formats minutes into readable string
func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	
	hours := minutes / 60
	remainingMinutes := minutes % 60
	
	if remainingMinutes == 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	
	return fmt.Sprintf("%d hours %d minutes", hours, remainingMinutes)
}

// GetQuickSummary provides a quick summary of potential savings
func (s *SimulationService) GetQuickSummary(ctx context.Context) (*QuickSummary, error) {
	config := &SimulationConfig{
		MaxRecommendations:  1000, // No limit for summary
		MinSavingsThreshold: 0.0,
	}
	
	result, err := s.SimulateAll(ctx, config)
	if err != nil {
		return nil, err
	}
	
	return &QuickSummary{
		TotalRecommendations: result.TotalRecommendations,
		ReadyToExecute:       result.ApprovedCount,
		RequiresApproval:     result.PendingCount,
		MonthlySavings:       result.MonthlySavings,
		AnnualSavings:        result.AnnualSavings,
		ResourcesAffected:    result.ResourcesAffected,
		HighRiskCount:        result.HighRiskCount,
		MediumRiskCount:      result.MediumRiskCount,
		LowRiskCount:         result.LowRiskCount,
	}, nil
}

// QuickSummary provides a condensed summary for dashboards
type QuickSummary struct {
	TotalRecommendations int     `json:"total_recommendations"`
	ReadyToExecute       int     `json:"ready_to_execute"`
	RequiresApproval     int     `json:"requires_approval"`
	MonthlySavings       float64 `json:"monthly_savings_usd"`
	AnnualSavings        float64 `json:"annual_savings_usd"`
	ResourcesAffected    int     `json:"resources_affected"`
	HighRiskCount        int     `json:"high_risk_count"`
	MediumRiskCount      int     `json:"medium_risk_count"`
	LowRiskCount         int     `json:"low_risk_count"`
}
