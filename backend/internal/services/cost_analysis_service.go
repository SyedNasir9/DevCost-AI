package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/pkg/logger"
)

// CostAnalysisService calculates cost estimates and savings projections
type CostAnalysisService struct {
	logger     *logger.Logger
	repository CostDataRepository
}

// CostDataRepository defines the interface for accessing cost data
type CostDataRepository interface {
	GetCostDataByResourceID(ctx context.Context, resourceID string, since time.Time) ([]*aws.CostData, error)
	GetCostDataByTimeRange(ctx context.Context, resourceID string, startDate, endDate string) ([]*aws.CostData, error)
	GetAverageDailyCost(ctx context.Context, resourceID string, days int) (float64, error)
}

// CostEstimate represents calculated cost estimates for a resource
type CostEstimate struct {
	ResourceID       string    `json:"resource_id"`
	HourlyCost       float64   `json:"hourly_cost_usd"`
	DailyCost        float64   `json:"daily_cost_usd"`
	MonthlyCost      float64   `json:"monthly_cost_usd"`
	AnnualCost       float64   `json:"annual_cost_usd"`
	
	// Calculation metadata
	CalculationBasis string    `json:"calculation_basis"` // "actual", "projected", "estimated"
	DataPoints       int       `json:"data_points"`
	PeriodDays       int       `json:"period_days"`
	Confidence       float64   `json:"confidence"` // 0.0 - 1.0
	
	// Currency
	Currency         string    `json:"currency"`
	
	// Last updated
	CalculatedAt     time.Time `json:"calculated_at"`
}

// SavingsEstimate represents estimated savings for a recommendation
type SavingsEstimate struct {
	RecommendationType  string    `json:"recommendation_type"`
	
	// Current costs
	CurrentHourlyCost   float64   `json:"current_hourly_cost_usd"`
	CurrentDailyCost    float64   `json:"current_daily_cost_usd"`
	CurrentMonthlyCost  float64   `json:"current_monthly_cost_usd"`
	CurrentAnnualCost   float64   `json:"current_annual_cost_usd"`
	
	// Projected costs after implementation
	ProjectedHourlyCost   float64   `json:"projected_hourly_cost_usd"`
	ProjectedDailyCost    float64   `json:"projected_daily_cost_usd"`
	ProjectedMonthlyCost  float64   `json:"projected_monthly_cost_usd"`
	ProjectedAnnualCost   float64   `json:"projected_annual_cost_usd"`
	
	// Savings
	HourlySavings       float64   `json:"hourly_savings_usd"`
	DailySavings        float64   `json:"daily_savings_usd"`
	MonthlySavings      float64   `json:"monthly_savings_usd"`
	AnnualSavings       float64   `json:"annual_savings_usd"`
	SavingsPercentage   float64   `json:"savings_percentage"` // 0.0 - 100.0
	
	// Payback period (if applicable)
	PaybackPeriodHours  *float64  `json:"payback_period_hours,omitempty"`
	
	// Metadata
	Currency            string    `json:"currency"`
	Confidence          float64   `json:"confidence"`
	CalculatedAt        time.Time `json:"calculated_at"`
	
	// Calculation notes
	Notes               []string  `json:"notes,omitempty"`
}

// CostAnalysisConfig holds configuration for cost analysis
type CostAnalysisConfig struct {
	DefaultAnalysisPeriod    time.Duration // Default period for cost analysis (default: 7 days)
	MinimumDataPoints        int           // Minimum cost records for reliable estimate (default: 3)
	ConfidenceThreshold      float64       // Minimum confidence for reliable estimate (default: 0.7)
	HoursPerMonth            float64       // Hours in a month for calculations (default: 730)
	DaysPerMonth             float64       // Days in a month for calculations (default: 30.44)
}

// DefaultCostAnalysisConfig returns default configuration
func DefaultCostAnalysisConfig() *CostAnalysisConfig {
	return &CostAnalysisConfig{
		DefaultAnalysisPeriod: 7 * 24 * time.Hour, // 7 days
		MinimumDataPoints:     3,
		ConfidenceThreshold:   0.7,
		HoursPerMonth:         730.0,    // 365 days / 12 months * 24 hours
		DaysPerMonth:          30.44,    // 365 days / 12 months
	}
}

// NewCostAnalysisService creates a new cost analysis service
func NewCostAnalysisService(logger *logger.Logger, repository CostDataRepository) *CostAnalysisService {
	return &CostAnalysisService{
		logger:     logger,
		repository: repository,
	}
}

// CalculateCostEstimates calculates hourly, daily, and monthly cost estimates for a resource
func (s *CostAnalysisService) CalculateCostEstimates(
	ctx context.Context,
	resourceID string,
	analysisPeriod time.Duration,
) (*CostEstimate, error) {
	if analysisPeriod == 0 {
		analysisPeriod = DefaultCostAnalysisConfig().DefaultAnalysisPeriod
	}

	s.logger.Debug("Calculating cost estimates",
		zap.String("resource_id", resourceID),
		zap.Duration("analysis_period", analysisPeriod),
	)

	// Get cost data for the analysis period
	since := time.Now().Add(-analysisPeriod)
	costData, err := s.repository.GetCostDataByResourceID(ctx, resourceID, since)
	if err != nil {
		s.logger.Error("Failed to get cost data",
			zap.String("resource_id", resourceID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get cost data: %w", err)
	}

	if len(costData) == 0 {
		s.logger.Warn("No cost data found for resource",
			zap.String("resource_id", resourceID),
		)
		return s.createDefaultEstimate(resourceID), nil
	}

	// Calculate cost estimates from data
	estimate := s.calculateFromCostData(resourceID, costData, analysisPeriod)

	s.logger.Debug("Cost estimates calculated",
		zap.String("resource_id", resourceID),
		zap.Float64("hourly", estimate.HourlyCost),
		zap.Float64("daily", estimate.DailyCost),
		zap.Float64("monthly", estimate.MonthlyCost),
	)

	return estimate, nil
}

// CalculateSavingsEstimate calculates savings estimate for a recommendation
func (s *CostAnalysisService) CalculateSavingsEstimate(
	ctx context.Context,
	resourceID string,
	recommendationType RecommendationType,
	analysisPeriod time.Duration,
) (*SavingsEstimate, error) {
	// Get current cost estimates
	currentCosts, err := s.CalculateCostEstimates(ctx, resourceID, analysisPeriod)
	if err != nil {
		return nil, err
	}

	// Calculate projected costs based on recommendation type
	projectedCosts := s.calculateProjectedCosts(currentCosts, recommendationType)

	// Build savings estimate
	savings := &SavingsEstimate{
		RecommendationType:   string(recommendationType),
		CurrentHourlyCost:    currentCosts.HourlyCost,
		CurrentDailyCost:     currentCosts.DailyCost,
		CurrentMonthlyCost:   currentCosts.MonthlyCost,
		CurrentAnnualCost:    currentCosts.AnnualCost,
		ProjectedHourlyCost:  projectedCosts.HourlyCost,
		ProjectedDailyCost:   projectedCosts.DailyCost,
		ProjectedMonthlyCost: projectedCosts.MonthlyCost,
		ProjectedAnnualCost:  projectedCosts.AnnualCost,
		Currency:             currentCosts.Currency,
		Confidence:           currentCosts.Confidence,
		CalculatedAt:         time.Now(),
		Notes:                []string{},
	}

	// Calculate savings amounts
	savings.HourlySavings = savings.CurrentHourlyCost - savings.ProjectedHourlyCost
	savings.DailySavings = savings.CurrentDailyCost - savings.ProjectedDailyCost
	savings.MonthlySavings = savings.CurrentMonthlyCost - savings.ProjectedMonthlyCost
	savings.AnnualSavings = savings.CurrentAnnualCost - savings.ProjectedAnnualCost

	// Calculate savings percentage
	if savings.CurrentMonthlyCost > 0 {
		savings.SavingsPercentage = (savings.MonthlySavings / savings.CurrentMonthlyCost) * 100
	}

	// Add calculation notes
	s.addCalculationNotes(savings, recommendationType, currentCosts)

	s.logger.Info("Savings estimate calculated",
		zap.String("resource_id", resourceID),
		zap.String("recommendation_type", string(recommendationType)),
		zap.Float64("monthly_savings", savings.MonthlySavings),
		zap.Float64("savings_percentage", savings.SavingsPercentage),
	)

	return savings, nil
}

// BatchCalculateCostEstimates calculates cost estimates for multiple resources
func (s *CostAnalysisService) BatchCalculateCostEstimates(
	ctx context.Context,
	resourceIDs []string,
	analysisPeriod time.Duration,
) map[string]*CostEstimate {
	results := make(map[string]*CostEstimate)

	for _, resourceID := range resourceIDs {
		estimate, err := s.CalculateCostEstimates(ctx, resourceID, analysisPeriod)
		if err != nil {
			s.logger.Warn("Failed to calculate cost estimate",
				zap.String("resource_id", resourceID),
				zap.Error(err),
			)
			continue
		}
		results[resourceID] = estimate
	}

	return results
}

// AttachSavingsToRecommendation attaches calculated savings to a recommendation
func (s *CostAnalysisService) AttachSavingsToRecommendation(
	ctx context.Context,
	rec *Recommendation,
	analysisPeriod time.Duration,
) error {
	savings, err := s.CalculateSavingsEstimate(ctx, rec.ResourceID, rec.RecommendationType, analysisPeriod)
	if err != nil {
		s.logger.Error("Failed to calculate savings for recommendation",
			zap.String("resource_id", rec.ResourceID),
			zap.String("recommendation_id", rec.ID.String()),
			zap.Error(err),
		)
		return err
	}

	// Update recommendation with calculated savings
	rec.EstimatedSavings = savings.MonthlySavings
	rec.CurrentState = &ResourceState{
		Configuration: map[string]interface{}{
			"monthly_cost": savings.CurrentMonthlyCost,
			"daily_cost":   savings.CurrentDailyCost,
			"hourly_cost":  savings.CurrentHourlyCost,
		},
	}
	rec.ProposedState = &ResourceState{
		Configuration: map[string]interface{}{
			"monthly_cost": savings.ProjectedMonthlyCost,
			"daily_cost":   savings.ProjectedDailyCost,
			"hourly_cost":  savings.ProjectedHourlyCost,
			"monthly_savings": savings.MonthlySavings,
			"savings_percentage": savings.SavingsPercentage,
		},
	}

	// Add savings notes to rationale
	if len(savings.Notes) > 0 {
		rec.Rationale += fmt.Sprintf("\n\nSavings Analysis:\n")
		for _, note := range savings.Notes {
			rec.Rationale += fmt.Sprintf("- %s\n", note)
		}
	}

	s.logger.Debug("Savings attached to recommendation",
		zap.String("recommendation_id", rec.ID.String()),
		zap.Float64("monthly_savings", rec.EstimatedSavings),
	)

	return nil
}

// calculateFromCostData calculates cost estimates from actual cost data
func (s *CostAnalysisService) calculateFromCostData(
	resourceID string,
	costData []*aws.CostData,
	analysisPeriod time.Duration,
) *CostEstimate {
	config := DefaultCostAnalysisConfig()
	
	if len(costData) == 0 {
		return s.createDefaultEstimate(resourceID)
	}

	// Sum up total cost from all data points
	var totalCost float64
	for _, cost := range costData {
		totalCost += cost.CostAmount
	}

	// Calculate actual period covered by data
	actualPeriodDays := analysisPeriod.Hours() / 24
	if len(costData) > 1 {
		// Use actual date range from data if available
		firstDate := costData[0].Timestamp
		lastDate := costData[len(costData)-1].Timestamp
		actualPeriodDays = lastDate.Sub(firstDate).Hours() / 24
		if actualPeriodDays < 1 {
			actualPeriodDays = 1
		}
	}

	// Calculate daily average
	dailyCost := totalCost / actualPeriodDays

	// Calculate hourly average
	hourlyCost := dailyCost / 24

	// Calculate monthly projection
	monthlyCost := dailyCost * config.DaysPerMonth

	// Calculate annual projection
	annualCost := monthlyCost * 12

	// Calculate confidence based on data points
	confidence := s.calculateConfidence(len(costData), actualPeriodDays)

	estimate := &CostEstimate{
		ResourceID:       resourceID,
		HourlyCost:       round(hourlyCost, 4),
		DailyCost:        round(dailyCost, 2),
		MonthlyCost:      round(monthlyCost, 2),
		AnnualCost:       round(annualCost, 2),
		CalculationBasis: "actual",
		DataPoints:       len(costData),
		PeriodDays:       int(actualPeriodDays),
		Confidence:       confidence,
		Currency:         "USD",
		CalculatedAt:     time.Now(),
	}

	return estimate
}

// calculateProjectedCosts calculates projected costs after recommendation implementation
func (s *CostAnalysisService) calculateProjectedCosts(
	currentCosts *CostEstimate,
	recommendationType RecommendationType,
) *CostEstimate {
	config := DefaultCostAnalysisConfig()
	
	projected := &CostEstimate{
		ResourceID:       currentCosts.ResourceID,
		Currency:         currentCosts.Currency,
		CalculationBasis: "projected",
		CalculatedAt:     time.Now(),
	}

	// Apply savings multipliers based on recommendation type
	switch recommendationType {
	case RecommendationTypeStop:
		// Stopping EC2 saves compute costs (~100% of instance cost)
		// EBS volumes remain (typically ~20% of total cost)
		projected.HourlyCost = currentCosts.HourlyCost * 0.15  // 15% for EBS
		projected.DailyCost = currentCosts.DailyCost * 0.15
		projected.MonthlyCost = currentCosts.MonthlyCost * 0.15
		projected.AnnualCost = currentCosts.AnnualCost * 0.15

	case RecommendationTypeDelete:
		// Deleting removes all costs (100% savings)
		projected.HourlyCost = 0
		projected.DailyCost = 0
		projected.MonthlyCost = 0
		projected.AnnualCost = 0

	case RecommendationTypeResize:
		// Resizing typically saves 30-50% depending on target size
		// Conservative estimate: 50% savings
		projected.HourlyCost = currentCosts.HourlyCost * 0.5
		projected.DailyCost = currentCosts.DailyCost * 0.5
		projected.MonthlyCost = currentCosts.MonthlyCost * 0.5
		projected.AnnualCost = currentCosts.AnnualCost * 0.5

	case RecommendationTypeSchedule:
		// Business hours only (8am-6pm, 5 days = 50 hours/week)
		// vs 24/7 (168 hours/week) = ~70% of time running
		// But usually more efficient = ~65% cost reduction
		projected.HourlyCost = currentCosts.HourlyCost
		projected.DailyCost = currentCosts.DailyCost * 0.35  // Run only 35% of time
		projected.MonthlyCost = currentCosts.MonthlyCost * 0.35
		projected.AnnualCost = currentCosts.AnnualCost * 0.35

	case RecommendationTypeSnapshot:
		// Snapshot costs are ~20% of volume cost (storage only)
		projected.HourlyCost = currentCosts.HourlyCost * 0.2
		projected.DailyCost = currentCosts.DailyCost * 0.2
		projected.MonthlyCost = currentCosts.MonthlyCost * 0.2
		projected.AnnualCost = currentCosts.AnnualCost * 0.2

	default:
		// Unknown type, assume no savings
		projected.HourlyCost = currentCosts.HourlyCost
		projected.DailyCost = currentCosts.DailyCost
		projected.MonthlyCost = currentCosts.MonthlyCost
		projected.AnnualCost = currentCosts.AnnualCost
	}

	// Round values
	projected.HourlyCost = round(projected.HourlyCost, 4)
	projected.DailyCost = round(projected.DailyCost, 2)
	projected.MonthlyCost = round(projected.MonthlyCost, 2)
	projected.AnnualCost = round(projected.AnnualCost, 2)

	return projected
}

// calculateConfidence calculates confidence score based on data quality
func (s *CostAnalysisService) calculateConfidence(dataPoints int, periodDays float64) float64 {
	config := DefaultCostAnalysisConfig()
	
	confidence := 0.5 // Base confidence
	
	// More data points increase confidence
	if dataPoints >= config.MinimumDataPoints {
		confidence += 0.2
	}
	if dataPoints >= 7 {
		confidence += 0.1
	}
	if dataPoints >= 30 {
		confidence += 0.1
	}
	
	// Longer period increases confidence
	if periodDays >= 7 {
		confidence += 0.1
	}
	
	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// addCalculationNotes adds explanatory notes to savings estimate
func (s *CostAnalysisService) addCalculationNotes(
	savings *SavingsEstimate,
	recType RecommendationType,
	currentCosts *CostEstimate,
) {
	config := DefaultCostAnalysisConfig()
	
	savings.Notes = append(savings.Notes,
		fmt.Sprintf("Based on %.1f days of actual cost data (%d data points)",
			currentCosts.PeriodDays,
			currentCosts.DataPoints),
	)

	switch recType {
	case RecommendationTypeStop:
		savings.Notes = append(savings.Notes,
			"Stopping instance eliminates compute charges (100% of instance cost)",
			"EBS volume storage charges remain (~15% of total cost)",
		)
		
	case RecommendationTypeDelete:
		savings.Notes = append(savings.Notes,
			"Deleting resource eliminates all associated costs (100% savings)",
			"Ensure data is backed up before deletion",
		)
		
	case RecommendationTypeResize:
		savings.Notes = append(savings.Notes,
			"Downsizing to smaller instance class reduces compute costs",
			"Typical savings range: 30-70% depending on target size",
			"Used conservative estimate of 50% savings",
		)
		
	case RecommendationTypeSchedule:
		businessHoursPerWeek := 50.0 // 10 hours/day * 5 days
		totalHoursPerWeek := 168.0     // 24 hours * 7 days
		savings.Notes = append(savings.Notes,
			fmt.Sprintf("Business hours schedule: %.0f hours/week vs %.0f hours/week (24/7)",
				businessHoursPerWeek,
				totalHoursPerWeek),
			"Running only during business hours (8am-6pm, weekdays)",
			"Additional savings from reduced data transfer during off-hours",
		)
		
	case RecommendationTypeSnapshot:
		savings.Notes = append(savings.Notes,
			"Snapshot storage costs ~20% of active volume storage",
			"Snapshots are incremental and more cost-effective",
		)
	}

	// Add monthly calculation note
	savings.Notes = append(savings.Notes,
		fmt.Sprintf("Monthly projection based on %.1f days per month", config.DaysPerMonth),
	)
}

// createDefaultEstimate creates a default estimate when no data is available
func (s *CostAnalysisService) createDefaultEstimate(resourceID string) *CostEstimate {
	return &CostEstimate{
		ResourceID:       resourceID,
		HourlyCost:       0,
		DailyCost:        0,
		MonthlyCost:      0,
		AnnualCost:       0,
		CalculationBasis: "estimated",
		DataPoints:       0,
		PeriodDays:       0,
		Confidence:       0,
		Currency:         "USD",
		CalculatedAt:     time.Now(),
	}
}

// GetCostSummaryForResources generates a summary of costs for multiple resources
func (s *CostAnalysisService) GetCostSummaryForResources(
	ctx context.Context,
	resourceIDs []string,
) (*CostSummary, error) {
	totalHourly := 0.0
	totalDaily := 0.0
	totalMonthly := 0.0
	totalAnnual := 0.0
	resourceCount := 0
	
	for _, resourceID := range resourceIDs {
		estimate, err := s.CalculateCostEstimates(ctx, resourceID, 0)
		if err != nil {
			s.logger.Warn("Failed to get cost estimate for resource",
				zap.String("resource_id", resourceID),
				zap.Error(err),
			)
			continue
		}
		
		totalHourly += estimate.HourlyCost
		totalDaily += estimate.DailyCost
		totalMonthly += estimate.MonthlyCost
		totalAnnual += estimate.AnnualCost
		resourceCount++
	}
	
	summary := &CostSummary{
		ResourceCount:    resourceCount,
		TotalHourlyCost:  round(totalHourly, 4),
		TotalDailyCost:   round(totalDaily, 2),
		TotalMonthlyCost: round(totalMonthly, 2),
		TotalAnnualCost:  round(totalAnnual, 2),
		CalculatedAt:     time.Now(),
	}
	
	return summary, nil
}

// CostSummary provides aggregated cost information
type CostSummary struct {
	ResourceCount    int       `json:"resource_count"`
	TotalHourlyCost  float64   `json:"total_hourly_cost_usd"`
	TotalDailyCost   float64   `json:"total_daily_cost_usd"`
	TotalMonthlyCost float64   `json:"total_monthly_cost_usd"`
	TotalAnnualCost  float64   `json:"total_annual_cost_usd"`
	CalculatedAt     time.Time `json:"calculated_at"`
}

// Helper function to round float to specified decimal places
func round(value float64, decimals int) float64 {
	shift := 1.0
	for i := 0; i < decimals; i++ {
		shift *= 10
	}
	return float64(int(value*shift+0.5)) / shift
}
