package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/internal/repositories"
	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// CostOptimizationScheduler orchestrates waste detection and recommendation generation
type CostOptimizationScheduler struct {
	logger              *logger.Logger
	
	// Services
	resourceRepo          *repositories.ResourceRepository
	costRepo              repositories.CostRepository
	wasteService          *services.WasteDetectionService
	recommendationService *services.RecommendationService
	costAnalysisService   *services.CostAnalysisService
	
	// Configuration
	config              *CostOptimizationConfig
	
	// Scheduling control
	ticker              *time.Ticker
	stopChan            chan struct{}
	wg                  sync.WaitGroup
	isRunning           bool
	mu                  sync.Mutex
	
	// Metrics and status
	lastRunTime         *time.Time
	runCount            int
	lastRunStatus       string
	lastError           error
}

// CostOptimizationConfig holds scheduler configuration
type CostOptimizationConfig struct {
	// Schedule interval (default: 10-15 minutes)
	Interval          time.Duration
	
	// Analysis window for cost data (default: 7 days)
	CostAnalysisWindow time.Duration
	
	// Batch sizes for processing
	ResourceBatchSize  int
	CostBatchSize      int
	
	// Concurrency limits
	MaxConcurrentWasteDetection int
	MaxConcurrentRecommendations int
	
	// Feature toggles
	EnableWasteDetection       bool
	EnableRecommendations      bool
	EnableCostAnalysis         bool
	
	// Failure handling
	MaxRetries              int
	RetryDelay              time.Duration
	ContinueOnError         bool
}

// DefaultCostOptimizationConfig returns default configuration
func DefaultCostOptimizationConfig() *CostOptimizationConfig {
	return &CostOptimizationConfig{
		Interval:                     10 * time.Minute, // 10 minutes default
		CostAnalysisWindow:           7 * 24 * time.Hour, // 7 days
		ResourceBatchSize:            100,
		CostBatchSize:                500,
		MaxConcurrentWasteDetection:  5,
		MaxConcurrentRecommendations: 5,
		EnableWasteDetection:         true,
		EnableRecommendations:        true,
		EnableCostAnalysis:           true,
		MaxRetries:                   3,
		RetryDelay:                   5 * time.Second,
		ContinueOnError:              true,
	}
}

// OptimizationRunResult contains the results of a single optimization run
type OptimizationRunResult struct {
	RunID               uuid.UUID                    `json:"run_id"`
	StartTime           time.Time                    `json:"start_time"`
	EndTime             *time.Time                   `json:"end_time,omitempty"`
	Duration            time.Duration                `json:"duration"`
	Status              string                       `json:"status"` // success, partial, failed
	
	// Step results
	ResourcesFetched    int                          `json:"resources_fetched"`
	CostDataFetched     int                          `json:"cost_data_fetched"`
	WasteDetected       int                          `json:"waste_detected"`
	RecommendationsGenerated int                     `json:"recommendations_generated"`
	RecommendationsStored  int                       `json:"recommendations_stored"`
	
	// Financial summary
	TotalEstimatedSavings float64                   `json:"total_estimated_savings_usd"`
	
	// Errors by step
	StepErrors          map[string]error             `json:"step_errors,omitempty"`
}

// StepResult tracks the result of an individual step
type StepResult struct {
	StepName    string
	Success     bool
	Duration    time.Duration
	Error       error
	ItemsProcessed int
}

// NewCostOptimizationScheduler creates a new scheduler
func NewCostOptimizationScheduler(
	logger *logger.Logger,
	resourceRepo *repositories.ResourceRepository,
	costRepo repositories.CostRepository,
	wasteService *services.WasteDetectionService,
	recommendationService *services.RecommendationService,
	costAnalysisService *services.CostAnalysisService,
	config *CostOptimizationConfig,
) *CostOptimizationScheduler {
	if config == nil {
		config = DefaultCostOptimizationConfig()
	}

	return &CostOptimizationScheduler{
		logger:                logger,
		resourceRepo:          resourceRepo,
		costRepo:              costRepo,
		wasteService:          wasteService,
		recommendationService: recommendationService,
		costAnalysisService:   costAnalysisService,
		config:                config,
		stopChan:              make(chan struct{}),
		isRunning:             false,
	}
}

// Start begins the scheduled optimization runs
func (s *CostOptimizationScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		s.logger.Warn("Cost optimization scheduler is already running")
		return fmt.Errorf("scheduler already running")
	}

	s.logger.Info("Starting cost optimization scheduler",
		zap.Duration("interval", s.config.Interval),
		zap.Bool("waste_detection_enabled", s.config.EnableWasteDetection),
		zap.Bool("recommendations_enabled", s.config.EnableRecommendations),
	)

	s.isRunning = true
	s.ticker = time.NewTicker(s.config.Interval)

	// Start the scheduling goroutine
	s.wg.Add(1)
	go s.runLoop()

	s.logger.Info("Cost optimization scheduler started successfully")
	return nil
}

// Stop gracefully stops the scheduler
func (s *CostOptimizationScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		s.logger.Warn("Cost optimization scheduler is not running")
		return nil
	}

	s.logger.Info("Stopping cost optimization scheduler...")

	s.isRunning = false
	close(s.stopChan)

	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Wait for current run to complete
	s.wg.Wait()

	s.logger.Info("Cost optimization scheduler stopped")
	return nil
}

// RunOnce executes a single optimization run immediately
func (s *CostOptimizationScheduler) RunOnce(ctx context.Context) (*OptimizationRunResult, error) {
	s.logger.Info("Starting manual cost optimization run")
	return s.executeOptimizationRun(ctx)
}

// runLoop is the main scheduling loop
func (s *CostOptimizationScheduler) runLoop() {
	defer s.wg.Done()

	// Run immediately on start
	ctx := context.Background()
	result, err := s.executeOptimizationRun(ctx)
	if err != nil {
		s.logger.Error("Initial optimization run failed", zap.Error(err))
	} else {
		s.logger.Info("Initial optimization run completed",
			zap.String("status", result.Status),
			zap.Duration("duration", result.Duration),
		)
	}

	// Then run on ticker
	for {
		select {
		case <-s.stopChan:
			s.logger.Debug("Scheduler received stop signal")
			return
		case <-s.ticker.C:
			s.logger.Info("Starting scheduled optimization run")
			
			ctx := context.Background()
			result, err := s.executeOptimizationRun(ctx)
			
			if err != nil {
				s.logger.Error("Scheduled optimization run failed", zap.Error(err))
				s.lastRunStatus = "failed"
				s.lastError = err
			} else {
				s.logger.Info("Scheduled optimization run completed",
					zap.String("status", result.Status),
					zap.Duration("duration", result.Duration),
					zap.Int("waste_detected", result.WasteDetected),
					zap.Int("recommendations_generated", result.RecommendationsGenerated),
				)
				s.lastRunStatus = result.Status
			}
			
			now := time.Now()
			s.lastRunTime = &now
			s.runCount++
		}
	}
}

// executeOptimizationRun executes the full optimization pipeline
func (s *CostOptimizationScheduler) executeOptimizationRun(ctx context.Context) (*OptimizationRunResult, error) {
	startTime := time.Now()
	runID := uuid.New()
	
	result := &OptimizationRunResult{
		RunID:      runID,
		StartTime:  startTime,
		Status:     "running",
		StepErrors: make(map[string]error),
	}

	s.logger.Info("Executing optimization run",
		zap.String("run_id", runID.String()),
		zap.Time("start_time", startTime),
	)

	// Step 1: Fetch Resources
	step1Result := s.stepFetchResources(ctx)
	result.ResourcesFetched = step1Result.ItemsProcessed
	if !step1Result.Success {
		result.StepErrors["fetch_resources"] = step1Result.Error
		s.logger.Error("Step 1: Fetch resources failed",
			zap.String("run_id", runID.String()),
			zap.Error(step1Result.Error),
		)
		if !s.config.ContinueOnError {
			result.Status = "failed"
			return s.finalizeResult(result, startTime), step1Result.Error
		}
	} else {
		s.logger.Info("Step 1: Fetch resources completed",
			zap.String("run_id", runID.String()),
			zap.Int("count", step1Result.ItemsProcessed),
			zap.Duration("duration", step1Result.Duration),
		)
	}

	// Step 2: Fetch Cost Data
	step2Result := s.stepFetchCostData(ctx)
	result.CostDataFetched = step2Result.ItemsProcessed
	if !step2Result.Success {
		result.StepErrors["fetch_cost_data"] = step2Result.Error
		s.logger.Error("Step 2: Fetch cost data failed",
			zap.String("run_id", runID.String()),
			zap.Error(step2Result.Error),
		)
		if !s.config.ContinueOnError {
			result.Status = "failed"
			return s.finalizeResult(result, startTime), step2Result.Error
		}
	} else {
		s.logger.Info("Step 2: Fetch cost data completed",
			zap.String("run_id", runID.String()),
			zap.Int("count", step2Result.ItemsProcessed),
			zap.Duration("duration", step2Result.Duration),
		)
	}

	// Step 3: Run Waste Detection
	var wasteResult *services.WasteDetectionResult
	if s.config.EnableWasteDetection {
		step3Result := s.stepRunWasteDetection(ctx)
		result.WasteDetected = step3Result.ItemsProcessed
		
		if !step3Result.Success {
			result.StepErrors["waste_detection"] = step3Result.Error
			s.logger.Error("Step 3: Waste detection failed",
				zap.String("run_id", runID.String()),
				zap.Error(step3Result.Error),
			)
			if !s.config.ContinueOnError {
				result.Status = "failed"
				return s.finalizeResult(result, startTime), step3Result.Error
			}
		} else {
			wasteResult = step3Result.Data.(*services.WasteDetectionResult)
			result.WasteDetected = len(wasteResult.WasteResources)
			
			s.logger.Info("Step 3: Waste detection completed",
				zap.String("run_id", runID.String()),
				zap.Int("waste_count", result.WasteDetected),
				zap.Duration("duration", step3Result.Duration),
			)
		}
	}

	// Step 4: Generate Recommendations
	var recommendationResult *services.RecommendationResult
	if s.config.EnableRecommendations && wasteResult != nil {
		step4Result := s.stepGenerateRecommendations(ctx, wasteResult)
		
		if !step4Result.Success {
			result.StepErrors["generate_recommendations"] = step4Result.Error
			s.logger.Error("Step 4: Generate recommendations failed",
				zap.String("run_id", runID.String()),
				zap.Error(step4Result.Error),
			)
			if !s.config.ContinueOnError {
				result.Status = "failed"
				return s.finalizeResult(result, startTime), step4Result.Error
			}
		} else {
			recommendationResult = step4Result.Data.(*services.RecommendationResult)
			result.RecommendationsGenerated = len(recommendationResult.Recommendations)
			result.TotalEstimatedSavings = recommendationResult.Summary.TotalEstimatedSavings
			
			s.logger.Info("Step 4: Generate recommendations completed",
				zap.String("run_id", runID.String()),
				zap.Int("recommendations_count", result.RecommendationsGenerated),
				zap.Float64("total_savings", result.TotalEstimatedSavings),
				zap.Duration("duration", step4Result.Duration),
			)
		}
	}

	// Step 5: Store Results
	if recommendationResult != nil {
		step5Result := s.stepStoreResults(ctx, recommendationResult)
		result.RecommendationsStored = step5Result.ItemsProcessed
		
		if !step5Result.Success {
			result.StepErrors["store_results"] = step5Result.Error
			s.logger.Error("Step 5: Store results failed",
				zap.String("run_id", runID.String()),
				zap.Error(step5Result.Error),
			)
			if !s.config.ContinueOnError {
				result.Status = "failed"
				return s.finalizeResult(result, startTime), step5Result.Error
			}
		} else {
			s.logger.Info("Step 5: Store results completed",
				zap.String("run_id", runID.String()),
				zap.Int("stored_count", result.RecommendationsStored),
				zap.Duration("duration", step5Result.Duration),
			)
		}
	}

	// Determine final status
	if len(result.StepErrors) == 0 {
		result.Status = "success"
	} else {
		result.Status = "partial"
	}

	return s.finalizeResult(result, startTime), nil
}

// finalizeResult completes the run result with timing information
func (s *CostOptimizationScheduler) finalizeResult(result *OptimizationRunResult, startTime time.Time) *OptimizationRunResult {
	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(startTime)
	return result
}

// Step 1: Fetch Resources
func (s *CostOptimizationScheduler) stepFetchResources(ctx context.Context) StepResult {
	start := time.Now()
	s.logger.Info("Step 1: Fetching resources...")

	filter := &repositories.ResourceFilter{
		Limit: s.config.ResourceBatchSize,
	}

	resources, err := s.resourceRepo.GetResourcesByFilter(ctx, filter)
	if err != nil {
		return StepResult{
			StepName: "fetch_resources",
			Success:  false,
			Duration: time.Since(start),
			Error:    err,
		}
	}

	return StepResult{
		StepName:     "fetch_resources",
		Success:      true,
		Duration:     time.Since(start),
		ItemsProcessed: len(resources),
	}
}

// Step 2: Fetch Cost Data
func (s *CostOptimizationScheduler) stepFetchCostData(ctx context.Context) StepResult {
	start := time.Now()
	s.logger.Info("Step 2: Fetching cost data...")

	// Fetch recent cost data
	since := time.Now().Add(-s.config.CostAnalysisWindow)
	costData, err := s.costRepo.GetCostDataByTimeRange(ctx, since, time.Now())
	if err != nil {
		return StepResult{
			StepName: "fetch_cost_data",
			Success:  false,
			Duration: time.Since(start),
			Error:    err,
		}
	}

	return StepResult{
		StepName:     "fetch_cost_data",
		Success:      true,
		Duration:     time.Since(start),
		ItemsProcessed: len(costData),
	}
}

// Step 3: Run Waste Detection
func (s *CostOptimizationScheduler) stepRunWasteDetection(ctx context.Context) StepResult {
	start := time.Now()
	s.logger.Info("Step 3: Running waste detection...")

	result, err := s.wasteService.DetectWaste(ctx)
	if err != nil {
		return StepResult{
			StepName: "waste_detection",
			Success:  false,
			Duration: time.Since(start),
			Error:    err,
		}
	}

	return StepResult{
		StepName:     "waste_detection",
		Success:      true,
		Duration:     time.Since(start),
		ItemsProcessed: len(result.WasteResources),
		Data:         result,
	}
}

// Step 4: Generate Recommendations
func (s *CostOptimizationScheduler) stepGenerateRecommendations(
	ctx context.Context,
	wasteResult *services.WasteDetectionResult,
) StepResult {
	start := time.Now()
	s.logger.Info("Step 4: Generating recommendations...")

	// Get cost data for resources with waste
	costDataMap := make(map[string]*services.ResourceCostInfo)
	for _, waste := range wasteResult.WasteResources {
		estimate, err := s.costAnalysisService.CalculateCostEstimates(
			ctx,
			waste.ResourceID,
			s.config.CostAnalysisWindow,
		)
		if err != nil {
			s.logger.Warn("Failed to calculate cost estimate for waste resource",
				zap.String("resource_id", waste.ResourceID),
				zap.Error(err),
			)
			continue
		}

		costDataMap[waste.ResourceID] = &services.ResourceCostInfo{
			ResourceID:   waste.ResourceID,
			MonthlyCost:  estimate.MonthlyCost,
			DailyCost:    estimate.DailyCost,
		}
	}

	// Create recommendation input
	input := &services.RecommendationInput{
		WasteResults:      wasteResult.WasteResources,
		CostData:          costDataMap,
		ResourceMetadata:  make(map[string]*models.Resource),
	}

	result, err := s.recommendationService.GenerateRecommendations(ctx, input)
	if err != nil {
		return StepResult{
			StepName: "generate_recommendations",
			Success:  false,
			Duration: time.Since(start),
			Error:    err,
		}
	}

	return StepResult{
		StepName:     "generate_recommendations",
		Success:      true,
		Duration:     time.Since(start),
		ItemsProcessed: len(result.Recommendations),
		Data:         result,
	}
}

// Step 5: Store Results
func (s *CostOptimizationScheduler) stepStoreResults(
	ctx context.Context,
	recommendationResult *services.RecommendationResult,
) StepResult {
	start := time.Now()
	s.logger.Info("Step 5: Storing results...")

	// Enrich recommendations with accurate savings calculations
	for _, rec := range recommendationResult.Recommendations {
		err := s.costAnalysisService.AttachSavingsToRecommendation(
			ctx,
			rec,
			s.config.CostAnalysisWindow,
		)
		if err != nil {
			s.logger.Warn("Failed to attach savings to recommendation",
				zap.String("resource_id", rec.ResourceID),
				zap.Error(err),
			)
		}
	}

	// Save recommendations to repository
	err := s.recommendationService.SaveRecommendations(ctx, recommendationResult)
	if err != nil {
		return StepResult{
			StepName: "store_results",
			Success:  false,
			Duration: time.Since(start),
			Error:    err,
		}
	}

	return StepResult{
		StepName:     "store_results",
		Success:      true,
		Duration:     time.Since(start),
		ItemsProcessed: len(recommendationResult.Recommendations),
	}
}

// GetStatus returns the current scheduler status
func (s *CostOptimizationScheduler) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"is_running":        s.isRunning,
		"run_count":         s.runCount,
		"last_run_time":     s.lastRunTime,
		"last_run_status":   s.lastRunStatus,
		"last_error":        s.lastError,
		"interval":          s.config.Interval,
		"waste_enabled":     s.config.EnableWasteDetection,
		"recommendations_enabled": s.config.EnableRecommendations,
	}
}
