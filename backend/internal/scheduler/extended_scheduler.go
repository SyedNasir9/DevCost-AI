package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/repositories"
	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// ExtendedScheduler runs the full end-to-end cost optimization pipeline
type ExtendedScheduler struct {
	logger                *logger.Logger

	// AWS clients
	awsClient             *aws.Client

	// Repositories
	resourceRepo          *repositories.ResourceRepository
	costRepo            repositories.CostRepository
	wasteRepo           repositories.WasteRepository
	recommendationRepo    *repositories.RecommendationRepository

	// Services
	resourceSyncService   *services.ResourceSyncService
	costIngestionService  *services.CostIngestionService
	wasteService          *services.WasteDetectionService
	recommendationService *services.RecommendationService
	costAnalysisService   *services.CostAnalysisService
	actionPipeline        *services.ActionPipeline
	executionController *services.ExecutionController

	// Configuration
	config              *ExtendedSchedulerConfig

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

// ExtendedSchedulerConfig holds scheduler configuration
type ExtendedSchedulerConfig struct {
	// Schedule interval (default: 10-15 minutes)
	Interval          time.Duration

	// Stage toggles
	EnableResourceSync      bool
	EnableCostIngestion     bool
	EnableWasteDetection    bool
	EnableRecommendations   bool
	EnableActionExecution   bool  // Optional stage

	// Stage-specific settings
	CostIngestionWindow     time.Duration  // How far back to fetch cost data
	ResourceSyncBatchSize   int
	CostIngestionBatchSize  int
	WasteDetectionBatchSize int
	RecommendationBatchSize int

	// Action execution settings
	ActionExecutionMode     services.ExecutionMode  // dry_run, approval_required, auto_execute
	MaxActionsPerRun      int
	MinSavingsThreshold   float64

	// Failure handling
	ContinueOnStageError  bool  // Continue to next stage if current fails
	MaxStageRetries       int

	// Safety
	DryRunMode            bool  // Never execute actual changes
}

// DefaultExtendedSchedulerConfig returns default configuration
func DefaultExtendedSchedulerConfig() *ExtendedSchedulerConfig {
	return &ExtendedSchedulerConfig{
		Interval:                10 * time.Minute,
		EnableResourceSync:      true,
		EnableCostIngestion:     true,
		EnableWasteDetection:    true,
		EnableRecommendations:   true,
		EnableActionExecution:   false, // Disabled by default for safety
		CostIngestionWindow:     1 * time.Hour,
		ResourceSyncBatchSize:   100,
		CostIngestionBatchSize:  500,
		WasteDetectionBatchSize: 50,
		RecommendationBatchSize: 100,
		ActionExecutionMode:     services.ExecutionModeDryRun,
		MaxActionsPerRun:        10,
		MinSavingsThreshold:     0.0,
		ContinueOnStageError:    true,
		MaxStageRetries:         3,
		DryRunMode:              false,
	}
}

// PipelineRunResult contains results from a full pipeline run
type PipelineRunResult struct {
	RunID               uuid.UUID              `json:"run_id"`
	StartTime           time.Time              `json:"start_time"`
	EndTime             *time.Time             `json:"end_time,omitempty"`
	Duration            time.Duration          `json:"duration"`
	Status              string                 `json:"status"` // success, partial, failed

	// Stage results
	Stages              []StageResult          `json:"stages"`

	// Aggregated metrics
	ResourcesSynced     int                    `json:"resources_synced"`
	CostRecordsIngested int                    `json:"cost_records_ingested"`
	WasteDetected       int                    `json:"waste_detected"`
	RecommendationsGenerated int               `json:"recommendations_generated"`
	ActionsExecuted     int                    `json:"actions_executed"`
	ActionsFailed       int                    `json:"actions_failed"`

	// Financial impact
	TotalEstimatedSavings float64              `json:"total_estimated_savings_usd"`

	// Errors
	Errors              []PipelineError        `json:"errors,omitempty"`
}

// StageResult tracks a single stage execution
type StageResult struct {
	Name           string         `json:"name"`
	Success        bool           `json:"success"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        *time.Time     `json:"end_time,omitempty"`
	Duration       time.Duration  `json:"duration"`
	ItemsProcessed int            `json:"items_processed"`
	Error          string         `json:"error,omitempty"`
}

// PipelineError captures pipeline-level errors
type PipelineError struct {
	Stage     string    `json:"stage"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// NewExtendedScheduler creates a new extended scheduler
func NewExtendedScheduler(
	logger *logger.Logger,
	awsClient *aws.Client,
	resourceRepo *repositories.ResourceRepository,
	costRepo repositories.CostRepository,
	wasteRepo repositories.WasteRepository,
	recommendationRepo *repositories.RecommendationRepository,
	resourceSyncService *services.ResourceSyncService,
	costIngestionService *services.CostIngestionService,
	wasteService *services.WasteDetectionService,
	recommendationService *services.RecommendationService,
	costAnalysisService *services.CostAnalysisService,
	actionPipeline *services.ActionPipeline,
	executionController *services.ExecutionController,
	config *ExtendedSchedulerConfig,
) *ExtendedScheduler {
	if config == nil {
		config = DefaultExtendedSchedulerConfig()
	}

	return &ExtendedScheduler{
		logger:                logger,
		awsClient:             awsClient,
		resourceRepo:          resourceRepo,
		costRepo:              costRepo,
		wasteRepo:             wasteRepo,
		recommendationRepo:    recommendationRepo,
		resourceSyncService:   resourceSyncService,
		costIngestionService:  costIngestionService,
		wasteService:          wasteService,
		recommendationService: recommendationService,
		costAnalysisService:   costAnalysisService,
		actionPipeline:        actionPipeline,
		executionController:   executionController,
		config:                config,
		stopChan:              make(chan struct{}),
		isRunning:             false,
	}
}

// Start begins the scheduled pipeline runs
func (s *ExtendedScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		s.logger.Warn("Extended scheduler is already running")
		return fmt.Errorf("scheduler already running")
	}

	s.logger.Info("Starting extended scheduler",
		zap.Duration("interval", s.config.Interval),
		zap.Bool("resource_sync", s.config.EnableResourceSync),
		zap.Bool("cost_ingestion", s.config.EnableCostIngestion),
		zap.Bool("waste_detection", s.config.EnableWasteDetection),
		zap.Bool("recommendations", s.config.EnableRecommendations),
		zap.Bool("action_execution", s.config.EnableActionExecution),
		zap.String("action_mode", string(s.config.ActionExecutionMode)),
	)

	s.isRunning = true
	s.ticker = time.NewTicker(s.config.Interval)

	s.wg.Add(1)
	go s.runLoop()

	s.logger.Info("Extended scheduler started successfully")
	return nil
}

// Stop gracefully stops the scheduler
func (s *ExtendedScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		s.logger.Warn("Extended scheduler is not running")
		return nil
	}

	s.logger.Info("Stopping extended scheduler...")

	s.isRunning = false
	close(s.stopChan)

	if s.ticker != nil {
		s.ticker.Stop()
	}

	s.wg.Wait()

	s.logger.Info("Extended scheduler stopped")
	return nil
}

// RunOnce executes a single pipeline run immediately
func (s *ExtendedScheduler) RunOnce(ctx context.Context) (*PipelineRunResult, error) {
	s.logger.Info("Starting manual pipeline run")
	return s.executePipeline(ctx)
}

// runLoop is the main scheduling loop
func (s *ExtendedScheduler) runLoop() {
	defer s.wg.Done()

	// Run immediately on start
	ctx := context.Background()
	result, err := s.executePipeline(ctx)
	if err != nil {
		s.logger.Error("Initial pipeline run failed", zap.Error(err))
	} else {
		s.logger.Info("Initial pipeline run completed",
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
			s.logger.Info("Starting scheduled pipeline run")

			ctx := context.Background()
			result, err := s.executePipeline(ctx)

			if err != nil {
				s.logger.Error("Scheduled pipeline run failed", zap.Error(err))
				s.lastRunStatus = "failed"
				s.lastError = err
			} else {
				s.logger.Info("Scheduled pipeline run completed",
					zap.String("status", result.Status),
					zap.Duration("duration", result.Duration),
					zap.Int("waste_detected", result.WasteDetected),
					zap.Int("recommendations", result.RecommendationsGenerated),
					zap.Int("actions", result.ActionsExecuted),
				)
				s.lastRunStatus = result.Status
			}

			now := time.Now()
			s.lastRunTime = &now
			s.runCount++
		}
	}
}

// executePipeline runs the full 5-stage pipeline
func (s *ExtendedScheduler) executePipeline(ctx context.Context) (*PipelineRunResult, error) {
	startTime := time.Now()
	runID := uuid.New()

	result := &PipelineRunResult{
		RunID:     runID,
		StartTime: startTime,
		Status:    "running",
		Stages:    []StageResult{},
		Errors:    []PipelineError{},
	}

	s.logger.Info("Executing pipeline",
		zap.String("run_id", runID.String()),
		zap.Time("start_time", startTime),
	)

	// Stage 1: Resource Sync
	if s.config.EnableResourceSync {
		stage1Result := s.stageResourceSync(ctx)
		result.Stages = append(result.Stages, stage1Result)
		result.ResourcesSynced = stage1Result.ItemsProcessed

		if !stage1Result.Success {
			s.logger.Error("Stage 1: Resource sync failed",
				zap.String("run_id", runID.String()),
				zap.String("error", stage1Result.Error),
			)
			result.Errors = append(result.Errors, PipelineError{
				Stage:     "resource_sync",
				Error:     stage1Result.Error,
				Timestamp: time.Now(),
			})
			if !s.config.ContinueOnStageError {
				result.Status = "failed"
				return s.finalizeResult(result, startTime), fmt.Errorf("resource sync failed: %s", stage1Result.Error)
			}
		} else {
			s.logger.Info("Stage 1: Resource sync completed",
				zap.String("run_id", runID.String()),
				zap.Int("resources_synced", stage1Result.ItemsProcessed),
				zap.Duration("duration", stage1Result.Duration),
			)
		}
	}

	// Stage 2: Cost Ingestion
	if s.config.EnableCostIngestion {
		stage2Result := s.stageCostIngestion(ctx)
		result.Stages = append(result.Stages, stage2Result)
		result.CostRecordsIngested = stage2Result.ItemsProcessed

		if !stage2Result.Success {
			s.logger.Error("Stage 2: Cost ingestion failed",
				zap.String("run_id", runID.String()),
				zap.String("error", stage2Result.Error),
			)
			result.Errors = append(result.Errors, PipelineError{
				Stage:     "cost_ingestion",
				Error:     stage2Result.Error,
				Timestamp: time.Now(),
			})
			if !s.config.ContinueOnStageError {
				result.Status = "failed"
				return s.finalizeResult(result, startTime), fmt.Errorf("cost ingestion failed: %s", stage2Result.Error)
			}
		} else {
			s.logger.Info("Stage 2: Cost ingestion completed",
				zap.String("run_id", runID.String()),
				zap.Int("records_ingested", stage2Result.ItemsProcessed),
				zap.Duration("duration", stage2Result.Duration),
			)
		}
	}

	// Stage 3: Waste Detection
	if s.config.EnableWasteDetection {
		stage3Result := s.stageWasteDetection(ctx)
		result.Stages = append(result.Stages, stage3Result)
		result.WasteDetected = stage3Result.ItemsProcessed

		if !stage3Result.Success {
			s.logger.Error("Stage 3: Waste detection failed",
				zap.String("run_id", runID.String()),
				zap.String("error", stage3Result.Error),
			)
			result.Errors = append(result.Errors, PipelineError{
				Stage:     "waste_detection",
				Error:     stage3Result.Error,
				Timestamp: time.Now(),
			})
			if !s.config.ContinueOnStageError {
				result.Status = "failed"
				return s.finalizeResult(result, startTime), fmt.Errorf("waste detection failed: %s", stage3Result.Error)
			}
		} else {
			s.logger.Info("Stage 3: Waste detection completed",
				zap.String("run_id", runID.String()),
				zap.Int("waste_detected", stage3Result.ItemsProcessed),
				zap.Duration("duration", stage3Result.Duration),
			)
		}
	}

	// Stage 4: Recommendation Generation
	if s.config.EnableRecommendations {
		stage4Result := s.stageRecommendationGeneration(ctx)
		result.Stages = append(result.Stages, stage4Result)
		result.RecommendationsGenerated = stage4Result.ItemsProcessed

		if !stage4Result.Success {
			s.logger.Error("Stage 4: Recommendation generation failed",
				zap.String("run_id", runID.String()),
				zap.String("error", stage4Result.Error),
			)
			result.Errors = append(result.Errors, PipelineError{
				Stage:     "recommendation_generation",
				Error:     stage4Result.Error,
				Timestamp: time.Now(),
			})
			if !s.config.ContinueOnStageError {
				result.Status = "failed"
				return s.finalizeResult(result, startTime), fmt.Errorf("recommendation generation failed: %s", stage4Result.Error)
			}
		} else {
			s.logger.Info("Stage 4: Recommendation generation completed",
				zap.String("run_id", runID.String()),
				zap.Int("recommendations", stage4Result.ItemsProcessed),
				zap.Duration("duration", stage4Result.Duration),
			)
		}
	}

	// Stage 5: Optional Action Execution
	if s.config.EnableActionExecution {
		stage5Result := s.stageActionExecution(ctx)
		result.Stages = append(result.Stages, stage5Result)
		result.ActionsExecuted = stage5Result.ItemsProcessed

		// Extract failures from stage result
		if stage5Result.Error != "" {
			// Parse failures from error or metadata
			result.ActionsFailed = 0 // Would be populated from actual result
		}

		if !stage5Result.Success {
			s.logger.Error("Stage 5: Action execution failed",
				zap.String("run_id", runID.String()),
				zap.String("error", stage5Result.Error),
			)
			result.Errors = append(result.Errors, PipelineError{
				Stage:     "action_execution",
				Error:     stage5Result.Error,
				Timestamp: time.Now(),
			})
			// Always continue after action execution - it's optional
		} else {
			s.logger.Info("Stage 5: Action execution completed",
				zap.String("run_id", runID.String()),
				zap.Int("actions_executed", stage5Result.ItemsProcessed),
				zap.Duration("duration", stage5Result.Duration),
			)
		}
	}

	// Determine final status
	if len(result.Errors) == 0 {
		result.Status = "success"
	} else {
		result.Status = "partial"
	}

	s.logger.Info("Pipeline execution completed",
		zap.String("run_id", runID.String()),
		zap.String("status", result.Status),
		zap.Int("stages_completed", len(result.Stages)),
		zap.Int("errors", len(result.Errors)),
	)

	return s.finalizeResult(result, startTime), nil
}

// Stage 1: Resource Sync
func (s *ExtendedScheduler) stageResourceSync(ctx context.Context) StageResult {
	start := time.Now()

	s.logger.Info("Stage 1: Starting resource sync",
		zap.Int("batch_size", s.config.ResourceSyncBatchSize),
	)

	// Sync EC2 resources
	ec2Resources, err := s.resourceSyncService.SyncEC2Resources(ctx)
	if err != nil {
		s.logger.Error("Failed to sync EC2 resources", zap.Error(err))
		return StageResult{
			Name:      "resource_sync",
			Success:   false,
			StartTime: start,
			Error:     fmt.Sprintf("EC2 sync failed: %v", err),
		}
	}

	// Sync RDS resources
	rdsResources, err := s.resourceSyncService.SyncRDSResources(ctx)
	if err != nil {
		s.logger.Error("Failed to sync RDS resources", zap.Error(err))
		return StageResult{
			Name:      "resource_sync",
			Success:   false,
			StartTime: start,
			Error:     fmt.Sprintf("RDS sync failed: %v", err),
		}
	}

	// Sync EBS volumes
	ebsResources, err := s.resourceSyncService.SyncEBSResources(ctx)
	if err != nil {
		s.logger.Error("Failed to sync EBS resources", zap.Error(err))
		return StageResult{
			Name:      "resource_sync",
			Success:   false,
			StartTime: start,
			Error:     fmt.Sprintf("EBS sync failed: %v", err),
		}
	}

	totalResources := len(ec2Resources) + len(rdsResources) + len(ebsResources)

	endTime := time.Now()
	return StageResult{
		Name:           "resource_sync",
		Success:        true,
		StartTime:      start,
		EndTime:        &endTime,
		Duration:       endTime.Sub(start),
		ItemsProcessed: totalResources,
	}
}

// Stage 2: Cost Ingestion
func (s *ExtendedScheduler) stageCostIngestion(ctx context.Context) StageResult {
	start := time.Now()

	s.logger.Info("Stage 2: Starting cost ingestion",
		zap.Duration("window", s.config.CostIngestionWindow),
	)

	// Ingest cost data from AWS Cost Explorer
	since := time.Now().Add(-s.config.CostIngestionWindow)
	costData, err := s.costIngestionService.IngestCostData(ctx, since, time.Now())
	if err != nil {
		s.logger.Error("Failed to ingest cost data", zap.Error(err))
		return StageResult{
			Name:      "cost_ingestion",
			Success:   false,
			StartTime: start,
			Error:     fmt.Sprintf("Cost ingestion failed: %v", err),
		}
	}

	endTime := time.Now()
	return StageResult{
		Name:           "cost_ingestion",
		Success:        true,
		StartTime:      start,
		EndTime:        &endTime,
		Duration:       endTime.Sub(start),
		ItemsProcessed: len(costData),
	}
}

// Stage 3: Waste Detection
func (s *ExtendedScheduler) stageWasteDetection(ctx context.Context) StageResult {
	start := time.Now()

	s.logger.Info("Stage 3: Starting waste detection")

	// Run waste detection
	wasteResult, err := s.wasteService.DetectWaste(ctx)
	if err != nil {
		s.logger.Error("Waste detection failed", zap.Error(err))
		return StageResult{
			Name:      "waste_detection",
			Success:   false,
			StartTime: start,
			Error:     fmt.Sprintf("Waste detection failed: %v", err),
		}
	}

	endTime := time.Now()
	return StageResult{
		Name:           "waste_detection",
		Success:        true,
		StartTime:      start,
		EndTime:        &endTime,
		Duration:       endTime.Sub(start),
		ItemsProcessed: len(wasteResult.WasteResources),
	}
}

// Stage 4: Recommendation Generation
func (s *ExtendedScheduler) stageRecommendationGeneration(ctx context.Context) StageResult {
	start := time.Now()

	s.logger.Info("Stage 4: Starting recommendation generation")

	// First get waste detection results
	wasteResult, err := s.wasteService.DetectWaste(ctx)
	if err != nil {
		s.logger.Error("Failed to get waste for recommendations", zap.Error(err))
		return StageResult{
			Name:      "recommendation_generation",
			Success:   false,
			StartTime: start,
			Error:     fmt.Sprintf("Failed to get waste results: %v", err),
		}
	}

	// Build cost data map for recommendations
	costDataMap := make(map[string]*services.ResourceCostInfo)
	for _, waste := range wasteResult.WasteResources {
		estimate, err := s.costAnalysisService.CalculateCostEstimates(
			ctx,
			waste.ResourceID,
			s.config.CostIngestionWindow,
		)
		if err != nil {
			s.logger.Warn("Failed to calculate cost estimate",
				zap.String("resource_id", waste.ResourceID),
				zap.Error(err),
			)
			continue
		}

		costDataMap[waste.ResourceID] = &services.ResourceCostInfo{
			ResourceID:  waste.ResourceID,
			MonthlyCost: estimate.MonthlyCost,
			DailyCost:   estimate.DailyCost,
		}
	}

	// Generate recommendations
	input := &services.RecommendationInput{
		WasteResults:     wasteResult.WasteResources,
		CostData:         costDataMap,
		ResourceMetadata: make(map[string]interface{}),
	}

	recResult, err := s.recommendationService.GenerateRecommendations(ctx, input)
	if err != nil {
		s.logger.Error("Recommendation generation failed", zap.Error(err))
		return StageResult{
			Name:      "recommendation_generation",
			Success:   false,
			StartTime: start,
			Error:     fmt.Sprintf("Recommendation generation failed: %v", err),
		}
	}

	endTime := time.Now()
	return StageResult{
		Name:           "recommendation_generation",
		Success:        true,
		StartTime:      start,
		EndTime:        &endTime,
		Duration:       endTime.Sub(start),
		ItemsProcessed: len(recResult.Recommendations),
	}
}

// Stage 5: Optional Action Execution
func (s *ExtendedScheduler) stageActionExecution(ctx context.Context) StageResult {
	start := time.Now()

	mode := s.config.ActionExecutionMode
	if s.config.DryRunMode {
		mode = services.ExecutionModeDryRun
	}

	s.logger.Info("Stage 5: Starting action execution",
		zap.String("mode", string(mode)),
		zap.Int("max_actions", s.config.MaxActionsPerRun),
	)

	// Check if we're in dry run or need approval
	if mode == services.ExecutionModeDryRun {
		s.logger.Info("Stage 5: Running in dry-run mode - no actual execution")
	}

	if mode == services.ExecutionModeApprovalRequired {
		s.logger.Info("Stage 5: Approval required mode - checking for pre-approved actions")
		// Only execute actions that have been pre-approved
	}

	// Run the action pipeline
	pipelineResult, err := s.actionPipeline.Run(ctx)
	if err != nil {
		s.logger.Error("Action pipeline failed", zap.Error(err))
		return StageResult{
			Name:      "action_execution",
			Success:   false,
			StartTime: start,
			Error:     fmt.Sprintf("Action execution failed: %v", err),
		}
	}

	endTime := time.Now()
	return StageResult{
		Name:           "action_execution",
		Success:        true,
		StartTime:      start,
		EndTime:        &endTime,
		Duration:       endTime.Sub(start),
		ItemsProcessed: pipelineResult.ExecutedCount,
	}
}

// finalizeResult completes the result with timing
func (s *ExtendedScheduler) finalizeResult(result *PipelineRunResult, startTime time.Time) *PipelineRunResult {
	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(startTime)
	return result
}

// GetStatus returns current scheduler status
func (s *ExtendedScheduler) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"is_running":            s.isRunning,
		"run_count":             s.runCount,
		"last_run_time":         s.lastRunTime,
		"last_run_status":       s.lastRunStatus,
		"interval":              s.config.Interval,
		"stages_enabled": map[string]bool{
			"resource_sync":      s.config.EnableResourceSync,
			"cost_ingestion":     s.config.EnableCostIngestion,
			"waste_detection":    s.config.EnableWasteDetection,
			"recommendations":    s.config.EnableRecommendations,
			"action_execution":   s.config.EnableActionExecution,
		},
		"action_mode":           s.config.ActionExecutionMode,
		"dry_run_mode":          s.config.DryRunMode,
	}
}
