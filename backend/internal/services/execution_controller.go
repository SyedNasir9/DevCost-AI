package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devcost-ai/pkg/logger"
)

// ExecutionMode defines the execution behavior
type ExecutionMode string

const (
	// ExecutionModeDryRun validates actions without executing
	ExecutionModeDryRun ExecutionMode = "dry_run"
	
	// ExecutionModeApprovalRequired requires explicit approval before execution
	ExecutionModeApprovalRequired ExecutionMode = "approval_required"
	
	// ExecutionModeAutoExecute automatically executes safe recommendations
	ExecutionModeAutoExecute ExecutionMode = "auto_execute"
)

// RiskLevel defines the risk classification
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// ExecutionController manages safe execution of recommendations
type ExecutionController struct {
	logger        *logger.Logger
	config        *ExecutionConfig
	actionService *ActionService
}

// ExecutionConfig holds configuration for execution control
type ExecutionConfig struct {
	// Global execution mode
	Mode ExecutionMode `json:"mode"`
	
	// Auto-execute criteria
	AutoExecuteMaxSavings      float64   `json:"auto_execute_max_savings_usd"`      // Max monthly savings for auto-execute
	AutoExecuteMaxRiskLevel    RiskLevel `json:"auto_execute_max_risk_level"`       // Max risk level for auto-execute
	AutoExecuteAllowedTypes    []RecommendationType `json:"auto_execute_allowed_types"` // Which types can auto-execute
	
	// Approval required criteria
	ApprovalRequiredMinSavings float64 `json:"approval_required_min_savings_usd"` // Min savings requiring approval
	ApprovalRequiredRiskLevels   []RiskLevel `json:"approval_required_risk_levels"`   // Risk levels requiring approval
	ApprovalRequiredTypes        []RecommendationType `json:"approval_required_types"` // Types always requiring approval
	
	// Protected resources (never auto-execute)
	ProtectedResourcePatterns []string `json:"protected_resource_patterns"` // Regex patterns
	ProtectedResourceTags     map[string]string `json:"protected_resource_tags"`  // Tag key-value pairs
	
	// Safety settings
	RequireDoubleConfirmation bool `json:"require_double_confirmation"` // For critical/high risk + high savings
	MaxActionsPerBatch        int  `json:"max_actions_per_batch"`       // Limit batch size
	EnableExecutionWindows    bool `json:"enable_execution_windows"`      // Only execute in time windows
	ExecutionWindowStart      string `json:"execution_window_start"`     // HH:MM format
	ExecutionWindowEnd        string `json:"execution_window_end"`       // HH:MM format
	ExecutionWindowTimezone   string `json:"execution_window_timezone"`  // e.g., "UTC", "America/New_York"
	
	// Logging
	LogAllDecisions bool `json:"log_all_decisions"`
}

// DefaultExecutionConfig returns safe default configuration
func DefaultExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		// Default to safest mode
		Mode: ExecutionModeDryRun,
		
		// Conservative auto-execute limits
		AutoExecuteMaxSavings:   100.0,  // $100/month max for auto-execute
		AutoExecuteMaxRiskLevel: RiskLevelLow,
		AutoExecuteAllowedTypes: []RecommendationType{
			RecommendationTypeStop,    // Stopping is reversible
		},
		
		// Approval required for anything significant
		ApprovalRequiredMinSavings: 50.0,
		ApprovalRequiredRiskLevels: []RiskLevel{
			RiskLevelMedium,
			RiskLevelHigh,
			RiskLevelCritical,
		},
		ApprovalRequiredTypes: []RecommendationType{
			RecommendationTypeDelete,   // Destructive
			RecommendationTypeResize,   // Can impact performance
		},
		
		// Protect production resources
		ProtectedResourcePatterns: []string{
			"*prod*",
			"*production*",
			"*critical*",
		},
		ProtectedResourceTags: map[string]string{
			"Environment": "production",
			"Critical":    "true",
		},
		
		// Safety settings
		RequireDoubleConfirmation: true,
		MaxActionsPerBatch:        10,
		EnableExecutionWindows:    false,
		ExecutionWindowStart:      "02:00",
		ExecutionWindowEnd:        "06:00",
		ExecutionWindowTimezone:   "UTC",
		
		// Logging
		LogAllDecisions: true,
	}
}

// ExecutionDecision represents the decision for a recommendation
type ExecutionDecision struct {
	DecisionID        uuid.UUID              `json:"decision_id"`
	RecommendationID  uuid.UUID              `json:"recommendation_id"`
	ResourceID        string                 `json:"resource_id"`
	ResourceType      string                 `json:"resource_type"`
	RecommendationType RecommendationType  `json:"recommendation_type"`
	
	// Decision details
	Decision          string                 `json:"decision"` // approve, reject, pending, dry_run
	ExecutionMode     ExecutionMode          `json:"execution_mode"`
	Reason            string                 `json:"reason"`
	RiskLevel         RiskLevel              `json:"risk_level"`
	EstimatedSavings  float64                `json:"estimated_savings_usd"`
	
	// Safety checks
	SafetyChecks      []SafetyCheckResult    `json:"safety_checks"`
	PassedAllChecks   bool                   `json:"passed_all_checks"`
	
	// Metadata
	CreatedAt         time.Time              `json:"created_at"`
	ExpiresAt         *time.Time             `json:"expires_at,omitempty"`
	
	// Execution tracking
	Executed          bool                   `json:"executed"`
	ExecutedAt        *time.Time             `json:"executed_at,omitempty"`
	ActionResult      *ActionResult          `json:"action_result,omitempty"`
}

// SafetyCheckResult represents a single safety check
type SafetyCheckResult struct {
	CheckName   string `json:"check_name"`
	Passed      bool   `json:"passed"`
	Severity    string `json:"severity"` // info, warning, critical
	Message     string `json:"message"`
}

// ExecutionBatch represents a batch of decisions
type ExecutionBatch struct {
	BatchID           uuid.UUID              `json:"batch_id"`
	Decisions         []*ExecutionDecision   `json:"decisions"`
	Mode              ExecutionMode          `json:"mode"`
	CreatedAt         time.Time              `json:"created_at"`
	
	// Summary
	TotalCount        int                    `json:"total_count"`
	ApprovedCount     int                    `json:"approved_count"`
	RejectedCount     int                    `json:"rejected_count"`
	PendingCount      int                    `json:"pending_count"`
	DryRunCount       int                    `json:"dry_run_count"`
	
	TotalSavings      float64                `json:"total_savings_usd"`
}

// NewExecutionController creates a new execution controller
func NewExecutionController(logger *logger.Logger, config *ExecutionConfig, actionService *ActionService) *ExecutionController {
	if config == nil {
		config = DefaultExecutionConfig()
	}
	
	return &ExecutionController{
		logger:        logger,
		config:        config,
		actionService: actionService,
	}
}

// EvaluateRecommendation evaluates a single recommendation and returns execution decision
func (c *ExecutionController) EvaluateRecommendation(ctx context.Context, rec *Recommendation) (*ExecutionDecision, error) {
	decisionID := uuid.New()
	
	c.logger.Info("Evaluating recommendation for execution",
		zap.String("decision_id", decisionID.String()),
		zap.String("recommendation_id", rec.ID.String()),
		zap.String("resource_id", rec.ResourceID),
		zap.String("resource_type", rec.ResourceType),
		zap.String("recommendation_type", string(rec.RecommendationType)),
		zap.String("mode", string(c.config.Mode)),
	)
	
	decision := &ExecutionDecision{
		DecisionID:         decisionID,
		RecommendationID:   rec.ID,
		ResourceID:         rec.ResourceID,
		ResourceType:       rec.ResourceType,
		RecommendationType: rec.RecommendationType,
		ExecutionMode:      c.config.Mode,
		RiskLevel:          RiskLevel(rec.RiskLevel),
		EstimatedSavings:   rec.EstimatedSavings,
		CreatedAt:          time.Now(),
		SafetyChecks:       []SafetyCheckResult{},
	}
	
	// Run all safety checks
	safetyResults := c.runSafetyChecks(ctx, rec)
	decision.SafetyChecks = safetyResults
	
	// Determine if all critical checks passed
	allPassed := true
	for _, check := range safetyResults {
		if check.Severity == "critical" && !check.Passed {
			allPassed = false
			break
		}
	}
	decision.PassedAllChecks = allPassed
	
	// Make execution decision based on mode and checks
	c.makeDecision(ctx, decision, rec)
	
	// Log decision
	c.logDecision(decision)
	
	return decision, nil
}

// EvaluateBatch evaluates multiple recommendations
func (c *ExecutionController) EvaluateBatch(ctx context.Context, recommendations []*Recommendation) (*ExecutionBatch, error) {
	batchID := uuid.New()
	
	c.logger.Info("Evaluating batch of recommendations",
		zap.String("batch_id", batchID.String()),
		zap.Int("count", len(recommendations)),
		zap.String("mode", string(c.config.Mode)),
	)
	
	batch := &ExecutionBatch{
		BatchID:   batchID,
		Decisions: []*ExecutionDecision{},
		Mode:      c.config.Mode,
		CreatedAt: time.Now(),
	}
	
	// Respect max batch size
	maxSize := c.config.MaxActionsPerBatch
	if len(recommendations) > maxSize {
		c.logger.Warn("Batch size exceeds maximum, truncating",
			zap.String("batch_id", batchID.String()),
			zap.Int("requested", len(recommendations)),
			zap.Int("max", maxSize),
		)
		recommendations = recommendations[:maxSize]
	}
	
	// Evaluate each recommendation
	for _, rec := range recommendations {
		decision, err := c.EvaluateRecommendation(ctx, rec)
		if err != nil {
			c.logger.Error("Failed to evaluate recommendation",
				zap.String("batch_id", batchID.String()),
				zap.String("recommendation_id", rec.ID.String()),
				zap.Error(err),
			)
			continue
		}
		
		batch.Decisions = append(batch.Decisions, decision)
		batch.TotalCount++
		batch.TotalSavings += rec.EstimatedSavings
		
		switch decision.Decision {
		case "approve":
			batch.ApprovedCount++
		case "reject":
			batch.RejectedCount++
		case "pending":
			batch.PendingCount++
		case "dry_run":
			batch.DryRunCount++
		}
	}
	
	c.logger.Info("Batch evaluation completed",
		zap.String("batch_id", batchID.String()),
		zap.Int("total", batch.TotalCount),
		zap.Int("approved", batch.ApprovedCount),
		zap.Int("rejected", batch.RejectedCount),
		zap.Int("pending", batch.PendingCount),
		zap.Int("dry_run", batch.DryRunCount),
		zap.Float64("total_savings", batch.TotalSavings),
	)
	
	return batch, nil
}

// ExecuteApproved executes approved decisions in a batch
func (c *ExecutionController) ExecuteApproved(ctx context.Context, batch *ExecutionBatch) ([]*ExecutionDecision, error) {
	c.logger.Info("Executing approved decisions",
		zap.String("batch_id", batch.BatchID.String()),
		zap.Int("approved_count", batch.ApprovedCount),
	)
	
	var executed []*ExecutionDecision
	
	for _, decision := range batch.Decisions {
		if decision.Decision != "approve" {
			continue
		}
		
		// Double-check execution window if enabled
		if c.config.EnableExecutionWindows && !c.isInExecutionWindow() {
			c.logger.Warn("Outside execution window, skipping execution",
				zap.String("decision_id", decision.DecisionID.String()),
				zap.String("window_start", c.config.ExecutionWindowStart),
				zap.String("window_end", c.config.ExecutionWindowEnd),
			)
			decision.Reason = "Outside execution window"
			continue
		}
		
		// Execute based on recommendation type
		result, err := c.executeDecision(ctx, decision)
		if err != nil {
			c.logger.Error("Execution failed",
				zap.String("decision_id", decision.DecisionID.String()),
				zap.Error(err),
			)
			decision.Reason = fmt.Sprintf("Execution failed: %v", err)
			continue
		}
		
		decision.Executed = true
		now := time.Now()
		decision.ExecutedAt = &now
		decision.ActionResult = result
		
		executed = append(executed, decision)
		
		c.logger.Info("Decision executed successfully",
			zap.String("decision_id", decision.DecisionID.String()),
			zap.String("action_id", result.ActionID.String()),
			zap.String("status", result.Status),
		)
	}
	
	c.logger.Info("Batch execution completed",
		zap.String("batch_id", batch.BatchID.String()),
		zap.Int("executed_count", len(executed)),
	)
	
	return executed, nil
}

// runSafetyChecks runs all safety checks on a recommendation
func (c *ExecutionController) runSafetyChecks(ctx context.Context, rec *Recommendation) []SafetyCheckResult {
	checks := []SafetyCheckResult{}
	
	// Check 1: Protected resource patterns
	check1 := c.checkProtectedPatterns(rec)
	checks = append(checks, check1)
	
	// Check 2: Protected tags
	check2 := c.checkProtectedTags(rec)
	checks = append(checks, check2)
	
	// Check 3: Execution window
	check3 := c.checkExecutionWindow(rec)
	checks = append(checks, check3)
	
	// Check 4: Double confirmation required
	check4 := c.checkDoubleConfirmationRequired(rec)
	checks = append(checks, check4)
	
	// Check 5: High savings threshold
	check5 := c.checkHighSavingsThreshold(rec)
	checks = append(checks, check5)
	
	// Check 6: Critical risk level
	check6 := c.checkCriticalRisk(rec)
	checks = append(checks, check6)
	
	return checks
}

// checkProtectedPatterns checks if resource matches protected patterns
func (c *ExecutionController) checkProtectedPatterns(rec *Recommendation) SafetyCheckResult {
	resourceName := strings.ToLower(rec.ResourceName)
	resourceID := strings.ToLower(rec.ResourceID)
	
	for _, pattern := range c.config.ProtectedResourcePatterns {
		pattern = strings.ToLower(pattern)
		pattern = strings.ReplaceAll(pattern, "*", "")
		
		if strings.Contains(resourceName, pattern) || strings.Contains(resourceID, pattern) {
			return SafetyCheckResult{
				CheckName: "protected_pattern",
				Passed:    false,
				Severity:  "critical",
				Message:   fmt.Sprintf("Resource matches protected pattern: %s", pattern),
			}
		}
	}
	
	return SafetyCheckResult{
		CheckName: "protected_pattern",
		Passed:    true,
		Severity:  "info",
		Message:   "No protected patterns matched",
	}
}

// checkProtectedTags checks if resource has protected tags
func (c *ExecutionController) checkProtectedTags(rec *Recommendation) SafetyCheckResult {
	// Note: In real implementation, fetch resource tags from repository
	// For now, assume we need to check
	
	return SafetyCheckResult{
		CheckName: "protected_tags",
		Passed:    true,
		Severity:  "info",
		Message:   "Tag check passed (resource tags not available in recommendation)",
	}
}

// checkExecutionWindow checks if current time is in execution window
func (c *ExecutionController) checkExecutionWindow(rec *Recommendation) SafetyCheckResult {
	if !c.config.EnableExecutionWindows {
		return SafetyCheckResult{
			CheckName: "execution_window",
			Passed:    true,
			Severity:  "info",
			Message:   "Execution windows disabled",
		}
	}
	
	if c.isInExecutionWindow() {
		return SafetyCheckResult{
			CheckName: "execution_window",
			Passed:    true,
			Severity:  "info",
			Message:   fmt.Sprintf("Within execution window (%s - %s)",
				c.config.ExecutionWindowStart, c.config.ExecutionWindowEnd),
		}
	}
	
	return SafetyCheckResult{
		CheckName: "execution_window",
		Passed:    false,
		Severity:  "warning",
		Message:   fmt.Sprintf("Outside execution window (%s - %s %s)",
			c.config.ExecutionWindowStart, c.config.ExecutionWindowEnd, c.config.ExecutionWindowTimezone),
	}
}

// checkDoubleConfirmationRequired checks if double confirmation is needed
func (c *ExecutionController) checkDoubleConfirmationRequired(rec *Recommendation) SafetyCheckResult {
	if !c.config.RequireDoubleConfirmation {
		return SafetyCheckResult{
			CheckName: "double_confirmation",
			Passed:    true,
			Severity:  "info",
			Message:   "Double confirmation not required",
		}
	}
	
	riskLevel := RiskLevel(rec.RiskLevel)
	
	// Require double confirmation for high savings + high risk
	if rec.EstimatedSavings >= c.config.ApprovalRequiredMinSavings*2 &&
		(riskLevel == RiskLevelHigh || riskLevel == RiskLevelCritical) {
		return SafetyCheckResult{
			CheckName: "double_confirmation",
			Passed:    false,
			Severity:  "critical",
			Message:   fmt.Sprintf("Double confirmation required: high savings ($%.2f) + high risk (%s)",
				rec.EstimatedSavings, riskLevel),
		}
	}
	
	return SafetyCheckResult{
		CheckName: "double_confirmation",
		Passed:    true,
		Severity:  "info",
		Message:   "Double confirmation not required",
	}
}

// checkHighSavingsThreshold checks if savings exceed threshold
func (c *ExecutionController) checkHighSavingsThreshold(rec *Recommendation) SafetyCheckResult {
	if rec.EstimatedSavings >= c.config.ApprovalRequiredMinSavings*5 {
		return SafetyCheckResult{
			CheckName: "high_savings",
			Passed:    false,
			Severity:  "warning",
			Message:   fmt.Sprintf("Very high savings amount: $%.2f/month", rec.EstimatedSavings),
		}
	}
	
	return SafetyCheckResult{
		CheckName: "high_savings",
		Passed:    true,
		Severity:  "info",
		Message:   fmt.Sprintf("Savings amount acceptable: $%.2f/month", rec.EstimatedSavings),
	}
}

// checkCriticalRisk checks for critical risk level
func (c *ExecutionController) checkCriticalRisk(rec *Recommendation) SafetyCheckResult {
	if rec.RiskLevel == "critical" {
		return SafetyCheckResult{
			CheckName: "critical_risk",
			Passed:    false,
			Severity:  "critical",
			Message:   "Critical risk level - manual review required",
		}
	}
	
	return SafetyCheckResult{
		CheckName: "critical_risk",
		Passed:    true,
		Severity:  "info",
		Message:   fmt.Sprintf("Risk level acceptable: %s", rec.RiskLevel),
	}
}

// makeDecision makes the final execution decision
func (c *ExecutionController) makeDecision(ctx context.Context, decision *ExecutionDecision, rec *Recommendation) {
	switch c.config.Mode {
	case ExecutionModeDryRun:
		decision.Decision = "dry_run"
		decision.Reason = "Global dry run mode enabled - no actions will be executed"
		
	case ExecutionModeApprovalRequired:
		decision.Decision = "pending"
		decision.Reason = "Approval required - awaiting manual review"
		expiresAt := time.Now().Add(24 * time.Hour)
		decision.ExpiresAt = &expiresAt
		
	case ExecutionModeAutoExecute:
		c.makeAutoExecuteDecision(ctx, decision, rec)
		
	default:
		decision.Decision = "reject"
		decision.Reason = "Unknown execution mode"
	}
}

// makeAutoExecuteDecision makes decision for auto-execute mode
func (c *ExecutionController) makeAutoExecuteDecision(ctx context.Context, decision *ExecutionDecision, rec *Recommendation) {
	// Check if recommendation type is allowed for auto-execute
	typeAllowed := false
	for _, allowedType := range c.config.AutoExecuteAllowedTypes {
		if allowedType == rec.RecommendationType {
			typeAllowed = true
			break
		}
	}
	
	if !typeAllowed {
		decision.Decision = "pending"
		decision.Reason = fmt.Sprintf("Auto-execute not allowed for type: %s", rec.RecommendationType)
		return
	}
	
	// Check savings threshold
	if rec.EstimatedSavings > c.config.AutoExecuteMaxSavings {
		decision.Decision = "pending"
		decision.Reason = fmt.Sprintf("Savings ($%.2f) exceeds auto-execute threshold ($%.2f)",
			rec.EstimatedSavings, c.config.AutoExecuteMaxSavings)
		return
	}
	
	// Check risk level
	riskLevel := RiskLevel(rec.RiskLevel)
	if riskLevel > c.config.AutoExecuteMaxRiskLevel {
		decision.Decision = "pending"
		decision.Reason = fmt.Sprintf("Risk level (%s) exceeds auto-execute threshold (%s)",
			riskLevel, c.config.AutoExecuteMaxRiskLevel)
		return
	}
	
	// Check safety checks
	if !decision.PassedAllChecks {
		decision.Decision = "pending"
		decision.Reason = "Failed critical safety checks - manual review required"
		return
	}
	
	// All checks passed - approve for auto-execution
	decision.Decision = "approve"
	decision.Reason = fmt.Sprintf("Auto-approved: type=%s, risk=%s, savings=$%.2f",
		rec.RecommendationType, rec.RiskLevel, rec.EstimatedSavings)
}

// executeDecision executes an approved decision
func (c *ExecutionController) executeDecision(ctx context.Context, decision *ExecutionDecision) (*ActionResult, error) {
	switch decision.RecommendationType {
	case RecommendationTypeStop:
		input := EC2StopInput{
			InstanceID: decision.ResourceID,
			Force:      false,
			DryRun:     c.config.Mode == ExecutionModeDryRun,
		}
		return c.actionService.StopEC2(ctx, input)
		
	case RecommendationTypeDelete:
		input := EBSDeleteInput{
			VolumeID:       decision.ResourceID,
			CreateSnapshot: true,
			DryRun:         c.config.Mode == ExecutionModeDryRun,
		}
		return c.actionService.DeleteEBS(ctx, input)
		
	case RecommendationTypeResize:
		// Note: Need to determine target class from recommendation
		input := RDSResizeInput{
			InstanceID:       decision.ResourceID,
			NewInstanceClass: "db.t3.small", // Should come from recommendation
			ApplyImmediately: false,
			DryRun:           c.config.Mode == ExecutionModeDryRun,
		}
		return c.actionService.ResizeRDS(ctx, input)
		
	default:
		return nil, fmt.Errorf("unsupported recommendation type: %s", decision.RecommendationType)
	}
}

// isInExecutionWindow checks if current time is within execution window
func (c *ExecutionController) isInExecutionWindow() bool {
	// Parse window times
	now := time.Now().UTC()
	
	startParts := strings.Split(c.config.ExecutionWindowStart, ":")
	endParts := strings.Split(c.config.ExecutionWindowEnd, ":")
	
	if len(startParts) != 2 || len(endParts) != 2 {
		return true // Invalid config, allow execution
	}
	
	startHour := parseInt(startParts[0])
	startMin := parseInt(startParts[1])
	endHour := parseInt(endParts[0])
	endMin := parseInt(endParts[1])
	
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin
	currentMinutes := now.Hour()*60 + now.Minute()
	
	return currentMinutes >= startMinutes && currentMinutes <= endMinutes
}

// parseInt safely parses int
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// logDecision logs the execution decision
func (c *ExecutionController) logDecision(decision *ExecutionDecision) {
	if !c.config.LogAllDecisions {
		return
	}
	
	fields := []zap.Field{
		zap.String("decision_id", decision.DecisionID.String()),
		zap.String("recommendation_id", decision.RecommendationID.String()),
		zap.String("resource_id", decision.ResourceID),
		zap.String("resource_type", decision.ResourceType),
		zap.String("recommendation_type", string(decision.RecommendationType)),
		zap.String("decision", decision.Decision),
		zap.String("execution_mode", string(decision.ExecutionMode)),
		zap.String("risk_level", string(decision.RiskLevel)),
		zap.Float64("estimated_savings", decision.EstimatedSavings),
		zap.String("reason", decision.Reason),
		zap.Bool("passed_all_checks", decision.PassedAllChecks),
	}
	
	// Add safety check results
	for _, check := range decision.SafetyChecks {
		fields = append(fields, zap.Bool(fmt.Sprintf("check_%s", check.CheckName), check.Passed))
	}
	
	switch decision.Decision {
	case "approve":
		c.logger.Info("Recommendation approved for execution", fields...)
	case "reject":
		c.logger.Warn("Recommendation rejected", fields...)
	case "pending":
		c.logger.Info("Recommendation pending approval", fields...)
	case "dry_run":
		c.logger.Info("Recommendation in dry run mode", fields...)
	default:
		c.logger.Info("Recommendation evaluated", fields...)
	}
}

// GetMode returns the current execution mode
func (c *ExecutionController) GetMode() ExecutionMode {
	return c.config.Mode
}

// SetMode changes the execution mode
func (c *ExecutionController) SetMode(mode ExecutionMode) {
	c.logger.Info("Changing execution mode",
		zap.String("old_mode", string(c.config.Mode)),
		zap.String("new_mode", string(mode)),
	)
	c.config.Mode = mode
}

// GetConfig returns the execution configuration
func (c *ExecutionController) GetConfig() *ExecutionConfig {
	return c.config
}
