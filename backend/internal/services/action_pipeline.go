package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devcost-ai/internal/repositories"
	"devcost-ai/pkg/logger"
)

// ActionPipeline orchestrates the end-to-end action execution flow
type ActionPipeline struct {
	logger                *logger.Logger
	recommendationRepo  *repositories.RecommendationRepository
	executionController *ExecutionController
	actionService       *ActionService
	actionRepo          *repositories.ActionRepository
	config              *PipelineConfig
}

// PipelineConfig holds configuration for the action pipeline
type PipelineConfig struct {
	// Batch processing
	BatchSize           int
	MaxConcurrent       int
	
	// Failure handling
	ContinueOnError     bool
	MaxRetriesPerAction int
	
	// Execution control
	RequireExplicitApprove bool  // Even in auto mode, require approval
	DryRunMode            bool  // Override - force all to dry run
	
	// Limits
	MaxActionsPerRun      int
	MinSavingsThreshold   float64
}

// DefaultPipelineConfig returns default configuration
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		BatchSize:              10,
		MaxConcurrent:          5,
		ContinueOnError:        true,
		MaxRetriesPerAction:    3,
		RequireExplicitApprove: false,
		DryRunMode:            false,
		MaxActionsPerRun:      50,
		MinSavingsThreshold:   0.0,
	}
}

// PipelineResult contains the results of a pipeline run
type PipelineResult struct {
	RunID                uuid.UUID            `json:"run_id"`
	StartTime            time.Time            `json:"start_time"`
	EndTime              *time.Time           `json:"end_time,omitempty"`
	Duration             time.Duration        `json:"duration"`
	
	// Step results
	RecommendationsFound int                  `json:"recommendations_found"`
	EvaluatedCount       int                  `json:"evaluated_count"`
	ApprovedCount        int                  `json:"approved_count"`
	ExecutedCount        int                  `json:"executed_count"`
	StoredCount          int                  `json:"stored_count"`
	FailedCount          int                  `json:"failed_count"`
	
	// Financial impact
	TotalEstimatedSavings float64           `json:"total_estimated_savings_usd"`
	
	// Details
	StepResults          []StepResult       `json:"step_results"`
	Errors               []PipelineError      `json:"errors,omitempty"`
}

// StepResult tracks individual step results
type StepResult struct {
	StepName      string        `json:"step_name"`
	Success       bool          `json:"success"`
	ItemsProcessed int          `json:"items_processed"`
	Duration      time.Duration `json:"duration"`
	Error         string        `json:"error,omitempty"`
}

// PipelineError captures pipeline errors
type PipelineError struct {
	Step      string    `json:"step"`
	ResourceID string   `json:"resource_id,omitempty"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// NewActionPipeline creates a new action pipeline
func NewActionPipeline(
	logger *logger.Logger,
	recommendationRepo *repositories.RecommendationRepository,
	executionController *ExecutionController,
	actionService *ActionService,
	actionRepo *repositories.ActionRepository,
	config *PipelineConfig,
) *ActionPipeline {
	if config == nil {
		config = DefaultPipelineConfig()
	}
	
	return &ActionPipeline{
		logger:                logger,
		recommendationRepo:    recommendationRepo,
		executionController:   executionController,
		actionService:         actionService,
		actionRepo:           actionRepo,
		config:               config,
	}
}

// Run executes the full action pipeline
func (p *ActionPipeline) Run(ctx context.Context) (*PipelineResult, error) {
	startTime := time.Now()
	runID := uuid.New()
	
	result := &PipelineResult{
		RunID:     runID,
		StartTime: startTime,
		StepResults: []StepResult{},
		Errors:      []PipelineError{},
	}
	
	p.logger.Info("Starting action pipeline",
		zap.String("run_id", runID.String()),
		zap.Int("batch_size", p.config.BatchSize),
		zap.Bool("continue_on_error", p.config.ContinueOnError),
	)
	
	// Step 1: Get Recommendations
	step1Start := time.Now()
	recommendations, err := p.stepGetRecommendations(ctx)
	step1Duration := time.Since(step1Start)
	
	if err != nil {
		p.logger.Error("Step 1: Failed to get recommendations",
			zap.String("run_id", runID.String()),
			zap.Error(err),
		)
		result.StepResults = append(result.StepResults, StepResult{
			StepName: "get_recommendations",
			Success:  false,
			Duration: step1Duration,
			Error:    err.Error(),
		})
		result.Errors = append(result.Errors, PipelineError{
			Step:      "get_recommendations",
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return p.finalizeResult(result, startTime), err
	}
	
	result.RecommendationsFound = len(recommendations)
	result.StepResults = append(result.StepResults, StepResult{
		StepName:       "get_recommendations",
		Success:        true,
		ItemsProcessed: len(recommendations),
		Duration:       step1Duration,
	})
	
	p.logger.Info("Step 1: Get recommendations completed",
		zap.String("run_id", runID.String()),
		zap.Int("count", len(recommendations)),
		zap.Duration("duration", step1Duration),
	)
	
	// Check if no recommendations
	if len(recommendations) == 0 {
		p.logger.Info("No recommendations to process, pipeline complete",
			zap.String("run_id", runID.String()),
		)
		return p.finalizeResult(result, startTime), nil
	}
	
	// Step 2: Evaluate through Execution Controller
	step2Start := time.Now()
	decisions, err := p.stepEvaluateDecisions(ctx, recommendations)
	step2Duration := time.Since(step2Start)
	
	if err != nil {
		p.logger.Error("Step 2: Failed to evaluate decisions",
			zap.String("run_id", runID.String()),
			zap.Error(err),
		)
		result.StepResults = append(result.StepResults, StepResult{
			StepName: "evaluate_decisions",
			Success:  false,
			Duration: step2Duration,
			Error:    err.Error(),
		})
		result.Errors = append(result.Errors, PipelineError{
			Step:      "evaluate_decisions",
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		
		if !p.config.ContinueOnError {
			return p.finalizeResult(result, startTime), err
		}
	} else {
		result.EvaluatedCount = len(decisions)
		result.StepResults = append(result.StepResults, StepResult{
			StepName:       "evaluate_decisions",
			Success:        true,
			ItemsProcessed: len(decisions),
			Duration:       step2Duration,
		})
		
		// Count approved
		for _, d := range decisions {
			if d.Decision == "approve" {
				result.ApprovedCount++
				result.TotalEstimatedSavings += d.EstimatedSavings
			}
		}
		
		p.logger.Info("Step 2: Evaluate decisions completed",
			zap.String("run_id", runID.String()),
			zap.Int("evaluated", result.EvaluatedCount),
			zap.Int("approved", result.ApprovedCount),
			zap.Duration("duration", step2Duration),
		)
	}
	
	// Check if no approved decisions
	if result.ApprovedCount == 0 {
		p.logger.Info("No approved decisions to execute, pipeline complete",
			zap.String("run_id", runID.String()),
		)
		return p.finalizeResult(result, startTime), nil
	}
	
	// Step 3: Execute Actions
	step3Start := time.Now()
	executionResults, err := p.stepExecuteActions(ctx, decisions)
	step3Duration := time.Since(step3Start)
	
	if err != nil {
		p.logger.Error("Step 3: Error during action execution",
			zap.String("run_id", runID.String()),
			zap.Error(err),
		)
		result.StepResults = append(result.StepResults, StepResult{
			StepName: "execute_actions",
			Success:  false,
			Duration: step3Duration,
			Error:    err.Error(),
		})
		result.Errors = append(result.Errors, PipelineError{
			Step:      "execute_actions",
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		
		if !p.config.ContinueOnError {
			return p.finalizeResult(result, startTime), err
		}
	} else {
		result.ExecutedCount = len(executionResults)
		result.StepResults = append(result.StepResults, StepResult{
			StepName:       "execute_actions",
			Success:        true,
			ItemsProcessed: len(executionResults),
			Duration:       step3Duration,
		})
		
		// Count failures
		for _, r := range executionResults {
			if r.Status == "failed" {
				result.FailedCount++
			}
		}
		
		p.logger.Info("Step 3: Execute actions completed",
			zap.String("run_id", runID.String()),
			zap.Int("executed", result.ExecutedCount),
			zap.Int("failed", result.FailedCount),
			zap.Duration("duration", step3Duration),
		)
	}
	
	// Step 4: Store Results
	step4Start := time.Now()
	storedCount, err := p.stepStoreResults(ctx, executionResults, decisions)
	step4Duration := time.Since(step4Start)
	
	if err != nil {
		p.logger.Error("Step 4: Failed to store results",
			zap.String("run_id", runID.String()),
			zap.Error(err),
		)
		result.StepResults = append(result.StepResults, StepResult{
			StepName: "store_results",
			Success:  false,
			Duration: step4Duration,
			Error:    err.Error(),
		})
		result.Errors = append(result.Errors, PipelineError{
			Step:      "store_results",
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		
		if !p.config.ContinueOnError {
			return p.finalizeResult(result, startTime), err
		}
	} else {
		result.StoredCount = storedCount
		result.StepResults = append(result.StepResults, StepResult{
			StepName:       "store_results",
			Success:        true,
			ItemsProcessed: storedCount,
			Duration:       step4Duration,
		})
		
		p.logger.Info("Step 4: Store results completed",
			zap.String("run_id", runID.String()),
			zap.Int("stored", storedCount),
			zap.Duration("duration", step4Duration),
		)
	}
	
	// Final summary
	p.logger.Info("Action pipeline completed",
		zap.String("run_id", runID.String()),
		zap.Int("recommendations_found", result.RecommendationsFound),
		zap.Int("evaluated", result.EvaluatedCount),
		zap.Int("approved", result.ApprovedCount),
		zap.Int("executed", result.ExecutedCount),
		zap.Int("failed", result.FailedCount),
		zap.Int("stored", result.StoredCount),
		zap.Float64("total_savings", result.TotalEstimatedSavings),
		zap.Int("errors", len(result.Errors)),
	)
	
	return p.finalizeResult(result, startTime), nil
}

// stepGetRecommendations retrieves active recommendations
func (p *ActionPipeline) stepGetRecommendations(ctx context.Context) ([]*Recommendation, error) {
	p.logger.Debug("Step 1: Getting active recommendations")
	
	recommendations, err := p.recommendationRepo.GetActiveRecommendations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recommendations: %w", err)
	}
	
	// Apply savings threshold filter
	var filtered []*Recommendation
	for _, rec := range recommendations {
		if rec.EstimatedSavings >= p.config.MinSavingsThreshold {
			filtered = append(filtered, rec)
		}
	}
	
	// Apply max limit
	if len(filtered) > p.config.MaxActionsPerRun {
		p.logger.Warn("Truncating recommendations to max limit",
			zap.Int("original", len(filtered)),
			zap.Int("max", p.config.MaxActionsPerRun),
		)
		filtered = filtered[:p.config.MaxActionsPerRun]
	}
	
	p.logger.Debug("Step 1: Retrieved recommendations",
		zap.Int("total", len(recommendations)),
		zap.Int("after_filter", len(filtered)),
	)
	
	return filtered, nil
}

// stepEvaluateDecisions evaluates recommendations through execution controller
func (p *ActionPipeline) stepEvaluateDecisions(ctx context.Context, recommendations []*Recommendation) ([]*ExecutionDecision, error) {
	p.logger.Debug("Step 2: Evaluating decisions through execution controller",
		zap.Int("count", len(recommendations)),
	)
	
	var decisions []*ExecutionDecision
	
	for _, rec := range recommendations {
		decision, err := p.executionController.EvaluateRecommendation(ctx, rec)
		if err != nil {
			p.logger.Warn("Failed to evaluate recommendation",
				zap.String("recommendation_id", rec.ID.String()),
				zap.String("resource_id", rec.ResourceID),
				zap.Error(err),
			)
			
			if p.config.ContinueOnError {
				// Create a rejection decision for this item
				decisions = append(decisions, &ExecutionDecision{
					DecisionID:         uuid.New(),
					RecommendationID:   rec.ID,
					ResourceID:         rec.ResourceID,
					Decision:           "reject",
					Reason:             fmt.Sprintf("Evaluation failed: %v", err),
					CreatedAt:          time.Now(),
				})
				continue
			} else {
				return nil, err
			}
		}
		
		decisions = append(decisions, decision)
		
		p.logger.Debug("Recommendation evaluated",
			zap.String("recommendation_id", rec.ID.String()),
			zap.String("decision", decision.Decision),
			zap.String("reason", decision.Reason),
		)
	}
	
	return decisions, nil
}

// stepExecuteActions executes approved decisions
func (p *ActionPipeline) stepExecuteActions(ctx context.Context, decisions []*ExecutionDecision) ([]*ActionResult, error) {
	p.logger.Debug("Step 3: Executing approved actions",
		zap.Int("total_decisions", len(decisions)),
	)
	
	var results []*ActionResult
	
	for _, decision := range decisions {
		if decision.Decision != "approve" {
			p.logger.Debug("Skipping non-approved decision",
				zap.String("decision_id", decision.DecisionID.String()),
				zap.String("decision", decision.Decision),
			)
			continue
		}
		
		// Apply dry run override if configured
		if p.config.DryRunMode {
			p.logger.Info("Dry run mode override - action not executed",
				zap.String("resource_id", decision.ResourceID),
			)
			
			result := &ActionResult{
				ActionID:     uuid.New(),
				ActionType:   string(decision.RecommendationType),
				ResourceID:   decision.ResourceID,
				ResourceType: decision.ResourceType,
				Status:       "success",
				StartTime:    time.Now(),
				Message:      "Dry run mode - action not executed",
				DryRun:       true,
			}
			
			endTime := time.Now()
			result.EndTime = &endTime
			result.Duration = time.Since(result.StartTime)
			
			results = append(results, result)
			continue
		}
		
		// Execute the action
		p.logger.Info("Executing action",
			zap.String("decision_id", decision.DecisionID.String()),
			zap.String("resource_id", decision.ResourceID),
			zap.String("action_type", string(decision.RecommendationType)),
		)
		
		result, err := p.executeDecision(ctx, decision)
		if err != nil {
			p.logger.Error("Action execution failed",
				zap.String("decision_id", decision.DecisionID.String()),
				zap.String("resource_id", decision.ResourceID),
				zap.Error(err),
			)
			
			// Create failure result
			failureResult := &ActionResult{
				ActionID:     uuid.New(),
				ActionType:   string(decision.RecommendationType),
				ResourceID:   decision.ResourceID,
				ResourceType: decision.ResourceType,
				Status:       "failed",
				StartTime:    time.Now(),
				Error:        err.Error(),
				Message:      fmt.Sprintf("Execution failed: %v", err),
			}
			
			endTime := time.Now()
			failureResult.EndTime = &endTime
			failureResult.Duration = time.Since(failureResult.StartTime)
			
			results = append(results, failureResult)
			
			if !p.config.ContinueOnError {
				return results, err
			}
			continue
		}
		
		results = append(results, result)
		
		p.logger.Info("Action executed",
			zap.String("action_id", result.ActionID.String()),
			zap.String("resource_id", result.ResourceID),
			zap.String("status", result.Status),
			zap.Duration("duration", result.Duration),
		)
	}
	
	return results, nil
}

// stepStoreResults stores action results in database
func (p *ActionPipeline) stepStoreResults(ctx context.Context, actionResults []*ActionResult, decisions []*ExecutionDecision) (int, error) {
	p.logger.Debug("Step 4: Storing action results in database",
		zap.Int("count", len(actionResults)),
	)
	
	storedCount := 0
	
	for i, result := range actionResults {
		// Find corresponding decision for recommendation link
		var recID *uuid.UUID
		if i < len(decisions) {
			recID = &decisions[i].RecommendationID
		}
		
		// Convert to repository ActionResult
		repoResult := &repositories.ActionResult{
			ActionID:     result.ActionID,
			ActionType:   result.ActionType,
			ResourceID:   result.ResourceID,
			ResourceType: result.ResourceType,
			Status:       result.Status,
			StartTime:    result.StartTime,
			EndTime:      result.EndTime,
			Duration:     result.Duration,
			Message:      result.Message,
			Error:        result.Error,
			Metadata:     result.Metadata,
			DryRun:       result.DryRun,
		}
		
		err := p.actionRepo.SaveActionResult(ctx, repoResult, recID)
		if err != nil {
			p.logger.Error("Failed to store action result",
				zap.String("action_id", result.ActionID.String()),
				zap.Error(err),
			)
			
			if !p.config.ContinueOnError {
				return storedCount, err
			}
			continue
		}
		
		storedCount++
		
		p.logger.Debug("Action result stored",
			zap.String("action_id", result.ActionID.String()),
			zap.String("resource_id", result.ResourceID),
		)
	}
	
	return storedCount, nil
}

// executeDecision executes a single decision
func (p *ActionPipeline) executeDecision(ctx context.Context, decision *ExecutionDecision) (*ActionResult, error) {
	switch decision.RecommendationType {
	case RecommendationTypeStop:
		input := EC2StopInput{
			InstanceID: decision.ResourceID,
			Force:      false,
			DryRun:     false,
		}
		return p.actionService.StopEC2(ctx, input)
		
	case RecommendationTypeDelete:
		input := EBSDeleteInput{
			VolumeID:       decision.ResourceID,
			CreateSnapshot: true,
			DryRun:         false,
		}
		return p.actionService.DeleteEBS(ctx, input)
		
	case RecommendationTypeResize:
		// Need to determine target instance class from recommendation
		input := RDSResizeInput{
			InstanceID:       decision.ResourceID,
			NewInstanceClass: "db.t3.small", // Should be extracted from decision/proposed state
			ApplyImmediately: false,
			DryRun:           false,
		}
		return p.actionService.ResizeRDS(ctx, input)
		
	default:
		return nil, fmt.Errorf("unsupported recommendation type: %s", decision.RecommendationType)
	}
}

// finalizeResult finalizes the pipeline result with timing
func (p *ActionPipeline) finalizeResult(result *PipelineResult, startTime time.Time) *PipelineResult {
	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(startTime)
	return result
}

// GetStatus returns current pipeline status
func (p *ActionPipeline) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"batch_size":         p.config.BatchSize,
		"max_concurrent":     p.config.MaxConcurrent,
		"continue_on_error":  p.config.ContinueOnError,
		"dry_run_mode":       p.config.DryRunMode,
		"max_actions_per_run": p.config.MaxActionsPerRun,
		"execution_mode":     p.executionController.GetMode(),
	}
}
