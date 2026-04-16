package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devcost-ai/internal/models"
	"devcost-ai/pkg/logger"
)

// RecommendationType defines the type of recommendation
type RecommendationType string

const (
	RecommendationTypeStop     RecommendationType = "stop"
	RecommendationTypeDelete   RecommendationType = "delete"
	RecommendationTypeResize   RecommendationType = "resize"
	RecommendationTypeSchedule RecommendationType = "schedule"
	RecommendationTypeSnapshot RecommendationType = "snapshot"
)

// RecommendationStatus defines the status of a recommendation
type RecommendationStatus string

const (
	RecommendationStatusActive      RecommendationStatus = "active"
	RecommendationStatusPending     RecommendationStatus = "pending"
	RecommendationStatusAccepted    RecommendationStatus = "accepted"
	RecommendationStatusRejected    RecommendationStatus = "rejected"
	RecommendationStatusImplemented RecommendationStatus = "implemented"
	RecommendationStatusExpired     RecommendationStatus = "expired"
)

// RecommendationPriority defines the priority level
type RecommendationPriority string

const (
	RecommendationPriorityCritical RecommendationPriority = "critical"
	RecommendationPriorityHigh     RecommendationPriority = "high"
	RecommendationPriorityMedium   RecommendationPriority = "medium"
	RecommendationPriorityLow      RecommendationPriority = "low"
)

// Recommendation represents a resource optimization recommendation
type Recommendation struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ResourceID   string    `json:"resource_id" db:"resource_id"`
	ResourceUUID uuid.UUID `json:"resource_uuid" db:"resource_uuid"`
	ResourceType string    `json:"resource_type" db:"resource_type"`
	ResourceName string    `json:"resource_name" db:"resource_name"`

	RecommendationType RecommendationType     `json:"recommendation_type" db:"recommendation_type"`
	Status             RecommendationStatus   `json:"status" db:"status"`
	Priority           RecommendationPriority `json:"priority" db:"priority"`

	Title       string `json:"title" db:"title"`
	Description string `json:"description" db:"description"`
	Rationale   string `json:"rationale" db:"rationale"`

	CurrentState  *ResourceState `json:"current_state,omitempty" db:"current_state"`
	ProposedState *ResourceState `json:"proposed_state,omitempty" db:"proposed_state"`

	EstimatedSavings    float64  `json:"estimated_savings_usd" db:"estimated_savings_usd"`
	SavingsCurrency     string   `json:"savings_currency" db:"savings_currency"`
	RiskLevel           string   `json:"risk_level" db:"risk_level"`
	ImplementationSteps []string `json:"implementation_steps" db:"implementation_steps"`
	Alternatives        []string `json:"alternatives,omitempty" db:"alternatives"`

	WasteID    *uuid.UUID `json:"waste_id,omitempty" db:"waste_id"`
	CostDataID *uuid.UUID `json:"cost_data_id,omitempty" db:"cost_data_id"`

	ValidFrom     time.Time  `json:"valid_from" db:"valid_from"`
	ValidUntil    *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	ImplementedAt *time.Time `json:"implemented_at,omitempty" db:"implemented_at"`
	ImplementedBy *string    `json:"implemented_by,omitempty" db:"implemented_by"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ResourceState represents the current or proposed state of a resource
type ResourceState struct {
	InstanceType  string                 `json:"instance_type,omitempty"`
	Size          int                    `json:"size_gb,omitempty"`
	State         string                 `json:"state,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

// RecommendationInput combines waste detection and cost data for recommendation generation
type RecommendationInput struct {
	WasteResults     []*WasteResult
	CostData         map[string]*ResourceCostInfo // resource_id -> cost info
	ResourceMetadata map[string]*models.Resource
}

// ResourceCostInfo contains cost-related information for a resource
type ResourceCostInfo struct {
	ResourceID       string
	MonthlyCost      float64
	DailyCost        float64
	CurrentSpend     float64
	ProjectedSavings float64
}

// RecommendationResult contains the generated recommendations
type RecommendationResult struct {
	Recommendations []*Recommendation
	Summary         *RecommendationSummary
	GeneratedAt     time.Time
	Duration        time.Duration
}

// RecommendationSummary provides statistics about recommendations
type RecommendationSummary struct {
	TotalCount            int                            `json:"total_count"`
	ByType                map[RecommendationType]int     `json:"by_type"`
	ByPriority            map[RecommendationPriority]int `json:"by_priority"`
	ByStatus              map[RecommendationStatus]int   `json:"by_status"`
	TotalEstimatedSavings float64                        `json:"total_estimated_savings_usd"`
	HighPriorityCount     int                            `json:"high_priority_count"`
	CriticalCount         int                            `json:"critical_count"`
	ImplementationRate    float64                        `json:"implementation_rate"`
}

// RecommendationService generates optimization recommendations based on waste and cost data
type RecommendationService struct {
	logger     *logger.Logger
	repository RecommendationRepository
}

// RecommendationRepository defines the interface for recommendation persistence
type RecommendationRepository interface {
	SaveRecommendation(ctx context.Context, rec *Recommendation) error
	SaveRecommendations(ctx context.Context, recs []*Recommendation) error
	GetRecommendationByID(ctx context.Context, id uuid.UUID) (*Recommendation, error)
	GetRecommendationsByResource(ctx context.Context, resourceID string) ([]*Recommendation, error)
	GetRecommendationsByStatus(ctx context.Context, status RecommendationStatus) ([]*Recommendation, error)
	UpdateRecommendationStatus(ctx context.Context, id uuid.UUID, status RecommendationStatus) error
	GetActiveRecommendations(ctx context.Context) ([]*Recommendation, error)
	GetRecommendationSummary(ctx context.Context) (*RecommendationSummary, error)
}

// NewRecommendationService creates a new recommendation service
func NewRecommendationService(logger *logger.Logger, repository RecommendationRepository) *RecommendationService {
	return &RecommendationService{
		logger:     logger,
		repository: repository,
	}
}

// GenerateRecommendations creates recommendations from waste detection and cost data
func (s *RecommendationService) GenerateRecommendations(
	ctx context.Context,
	input *RecommendationInput,
) (*RecommendationResult, error) {
	startTime := time.Now()
	s.logger.Info("Generating recommendations",
		zap.Int("waste_count", len(input.WasteResults)),
		zap.Int("cost_data_count", len(input.CostData)),
	)

	var recommendations []*Recommendation

	// Generate recommendations from waste results
	for _, waste := range input.WasteResults {
		rec := s.generateRecommendationFromWaste(waste, input.CostData[waste.ResourceID])
		if rec != nil {
			recommendations = append(recommendations, rec)
		}
	}

	// Generate additional recommendations from cost analysis
	costBasedRecs := s.generateCostBasedRecommendations(input.CostData, input.ResourceMetadata)
	recommendations = append(recommendations, costBasedRecs...)

	// Deduplicate recommendations
	recommendations = s.deduplicateRecommendations(recommendations)

	// Prioritize recommendations
	recommendations = s.prioritizeRecommendations(recommendations)

	// Generate summary
	summary := s.generateRecommendationSummary(recommendations)

	result := &RecommendationResult{
		Recommendations: recommendations,
		Summary:         summary,
		GeneratedAt:     startTime,
		Duration:        time.Since(startTime),
	}

	s.logger.Info("Recommendation generation completed",
		zap.Int("recommendations_generated", len(recommendations)),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// generateRecommendationFromWaste creates a recommendation from a waste detection result
func (s *RecommendationService) generateRecommendationFromWaste(
	waste *WasteResult,
	costInfo *ResourceCostInfo,
) *Recommendation {
	rec := &Recommendation{
		ID:              uuid.New(),
		ResourceID:      waste.ResourceID,
		ResourceType:    waste.ResourceType,
		ResourceName:    waste.ResourceName,
		Status:          RecommendationStatusActive,
		SavingsCurrency: "USD",
		RiskLevel:       "low",
		ValidFrom:       time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if waste.ResourceUUID != "" {
		if id, err := uuid.Parse(waste.ResourceUUID); err == nil {
			rec.ResourceUUID = id
		}
	}

	// Set recommendation type and details based on waste type
	switch waste.WasteType {
	case WasteTypeIdleEC2:
		s.populateIdleEC2Recommendation(rec, waste, costInfo)
	case WasteTypeUnattachedEBS:
		s.populateUnattachedEBSRecommendation(rec, waste, costInfo)
	case WasteTypeUnderutilizedRDS:
		s.populateUnderutilizedRDSRecommendation(rec, waste, costInfo)
	}

	// Set priority based on severity and savings
	rec.Priority = s.determinePriority(waste.Severity, rec.EstimatedSavings)

	return rec
}

// populateIdleEC2Recommendation populates recommendation for idle EC2
func (s *RecommendationService) populateIdleEC2Recommendation(
	rec *Recommendation,
	waste *WasteResult,
	costInfo *ResourceCostInfo,
) {
	rec.RecommendationType = RecommendationTypeStop
	rec.Title = fmt.Sprintf("Stop idle EC2 instance: %s", waste.ResourceName)
	rec.Description = fmt.Sprintf(
		"This EC2 instance has been idle with average CPU utilization of %.1f%% over the past %v. "+
			"Stopping this instance will eliminate the compute charges while preserving the EBS volumes.",
		waste.Details.AvgCPUUtilization,
		waste.Details.IdleDuration,
	)
	rec.Rationale = "Idle instances incur unnecessary compute costs without providing value"
	rec.EstimatedSavings = waste.EstimatedSavings
	if costInfo != nil {
		rec.EstimatedSavings = costInfo.MonthlyCost
	}

	rec.CurrentState = &ResourceState{
		State: "running",
	}
	rec.ProposedState = &ResourceState{
		State: "stopped",
	}

	rec.ImplementationSteps = []string{
		"1. Verify no critical processes are running",
		"2. Create AMI backup if needed (optional)",
		"3. Stop the instance using AWS Console or CLI: aws ec2 stop-instances --instance-id " + waste.ResourceID,
		"4. Verify instance status changed to 'stopped'",
		"5. Tag instance with 'CostOptimized=true' for tracking",
	}

	rec.Alternatives = []string{
		"Terminate instance if no longer needed (higher savings but irreversible)",
		"Downsize to smaller instance type if occasional use is needed",
		"Use AWS Instance Scheduler for automatic start/stop based on schedule",
	}
}

// populateUnattachedEBSRecommendation populates recommendation for unattached EBS
func (s *RecommendationService) populateUnattachedEBSRecommendation(
	rec *Recommendation,
	waste *WasteResult,
	costInfo *ResourceCostInfo,
) {
	rec.RecommendationType = RecommendationTypeDelete
	rec.Title = fmt.Sprintf("Delete unattached EBS volume: %s", waste.ResourceName)

	volumeInfo := ""
	if waste.Details.VolumeSize > 0 {
		volumeInfo = fmt.Sprintf(" (%d GB %s)", waste.Details.VolumeSize, waste.Details.VolumeType)
	}

	rec.Description = fmt.Sprintf(
		"This EBS volume has been unattached for %d days%s. "+
			"Unattached volumes incur storage charges without providing any value.",
		waste.Details.DaysUnattached,
		volumeInfo,
	)
	rec.Rationale = "Unattached volumes are pure waste - storage costs with no benefit"
	rec.EstimatedSavings = waste.EstimatedSavings
	if costInfo != nil {
		rec.EstimatedSavings = costInfo.MonthlyCost
	}

	rec.RiskLevel = "medium" // Requires backup consideration
	rec.CurrentState = &ResourceState{
		State: "available",
		Size:  waste.Details.VolumeSize,
	}
	rec.ProposedState = &ResourceState{
		State: "deleted",
	}

	rec.ImplementationSteps = []string{
		"1. Create snapshot as backup: aws ec2 create-snapshot --volume-id " + waste.ResourceID,
		"2. Wait for snapshot completion",
		"3. Verify snapshot exists in AWS Console",
		"4. Delete the volume: aws ec2 delete-volume --volume-id " + waste.ResourceID,
		"5. Tag snapshot with retention policy",
	}

	rec.Alternatives = []string{
		"Keep volume attached to a placeholder instance (if future use planned)",
		"Archive data to S3 Glacier for long-term storage (lower cost)",
		"Create AMI from volume if OS/boot volume",
	}
}

// populateUnderutilizedRDSRecommendation populates recommendation for underutilized RDS
func (s *RecommendationService) populateUnderutilizedRDSRecommendation(
	rec *Recommendation,
	waste *WasteResult,
	costInfo *ResourceCostInfo,
) {
	rec.RecommendationType = RecommendationTypeResize
	rec.Title = fmt.Sprintf("Downsize underutilized RDS instance: %s", waste.ResourceName)
	rec.Description = fmt.Sprintf(
		"This RDS instance is underutilized with average of %.1f connections and %.1f%% CPU utilization. "+
			"Downsizing to a smaller instance class will significantly reduce costs while maintaining functionality.",
		waste.Details.AvgConnections,
		waste.Details.AvgCPUUtilization,
	)
	rec.Rationale = "Overprovisioned databases waste money - rightsize for actual workload"
	rec.EstimatedSavings = waste.EstimatedSavings * 0.5 // Resize typically saves 50%
	if costInfo != nil {
		rec.EstimatedSavings = costInfo.MonthlyCost * 0.5
	}

	rec.RiskLevel = "medium" // Requires performance testing
	rec.CurrentState = &ResourceState{
		State: "available",
	}
	rec.ProposedState = &ResourceState{
		State: "available",
	}

	rec.ImplementationSteps = []string{
		"1. Review current instance class in RDS Console",
		"2. Create read replica for testing (optional but recommended)",
		"3. Identify target smaller instance class (e.g., db.t3.small → db.t3.micro)",
		"4. Schedule maintenance window for resize",
		"5. Modify instance: aws rds modify-db-instance --db-instance-identifier " + waste.ResourceID + " --db-instance-class <target-class> --apply-immediately",
		"6. Monitor performance after resize",
	}

	rec.Alternatives = []string{
		"Use Aurora Serverless for variable workloads (pay per use)",
		"Enable Reserved Instances for predictable long-term usage",
		"Terminate and migrate to smaller database if truly minimal usage",
	}
}

// generateCostBasedRecommendations creates recommendations from cost analysis
func (s *RecommendationService) generateCostBasedRecommendations(
	costData map[string]*ResourceCostInfo,
	resourceMetadata map[string]*models.Resource,
) []*Recommendation {
	var recommendations []*Recommendation

	for resourceID, costInfo := range costData {
		resource, exists := resourceMetadata[resourceID]
		if !exists {
			continue
		}

		// Check for scheduling opportunities (resources running 24/7 but only used during business hours)
		if s.isCandidateForScheduling(resource, costInfo) {
			rec := s.createSchedulingRecommendation(resource, costInfo)
			if rec != nil {
				recommendations = append(recommendations, rec)
			}
		}
	}

	return recommendations
}

// isCandidateForScheduling determines if a resource should use scheduling
func (s *RecommendationService) isCandidateForScheduling(
	resource *models.Resource,
	costInfo *ResourceCostInfo,
) bool {
	// Only EC2 and RDS are candidates for scheduling
	if resource.ResourceType != models.ResourceTypeEC2 &&
		resource.ResourceType != models.ResourceTypeRDS {
		return false
	}

	// Must be in running/available state
	if resource.State != models.ResourceStateRunning &&
		resource.State != models.ResourceStateAvailable {
		return false
	}

	// Higher monthly cost indicates good scheduling candidate
	if costInfo.MonthlyCost < 50.0 {
		return false
	}

	return true
}

// createSchedulingRecommendation creates a scheduling-based recommendation
func (s *RecommendationService) createSchedulingRecommendation(
	resource *models.Resource,
	costInfo *ResourceCostInfo,
) *Recommendation {
	rec := &Recommendation{
		ID:                 uuid.New(),
		ResourceID:         resource.ResourceID,
		ResourceType:       string(resource.ResourceType),
		ResourceName:       resource.Name,
		RecommendationType: RecommendationTypeSchedule,
		Status:             RecommendationStatusActive,
		Priority:           RecommendationPriorityMedium,
		SavingsCurrency:    "USD",
		RiskLevel:          "low",
		ValidFrom:          time.Now(),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Parse resource ID to UUID
	if id, err := uuid.Parse(resource.ID); err == nil {
		rec.ResourceUUID = id
	}

	savings := costInfo.MonthlyCost * 0.65 // ~65% savings for business hours only (8am-6pm, 5 days)

	rec.Title = fmt.Sprintf("Implement auto-scheduling for %s: %s", resource.ResourceType, resource.Name)
	rec.Description = fmt.Sprintf(
		"This %s is running 24/7 but can be automatically stopped during non-business hours. "+
			"Implementing auto-scheduling (stop at 6 PM, start at 8 AM, weekdays only) can reduce costs by ~65%%.",
		resource.ResourceType,
	)
	rec.Rationale = "Many resources are only needed during business hours - scheduling eliminates waste"
	rec.EstimatedSavings = savings

	rec.CurrentState = &ResourceState{
		State: string(resource.State),
		Configuration: map[string]interface{}{
			"schedule": "24/7",
		},
	}
	rec.ProposedState = &ResourceState{
		State: string(resource.State),
		Configuration: map[string]interface{}{
			"schedule":   "business-hours-only",
			"stop_time":  "18:00",
			"start_time": "08:00",
			"days":       "monday-friday",
		},
	}

	rec.ImplementationSteps = []string{
		"1. Analyze CloudWatch metrics to confirm usage pattern",
		"2. Choose scheduling solution:",
		"   - Option A: AWS Instance Scheduler (CloudFormation template)",
		"   - Option B: AWS Systems Manager Maintenance Windows",
		"   - Option C: Custom Lambda with EventBridge",
		"3. Configure schedule: stop at 6 PM, start at 8 AM (weekdays)",
		"4. Set up SNS notifications for start/stop events",
		"5. Test scheduling for 1 week before full deployment",
		"6. Monitor and adjust schedule as needed",
	}

	rec.Alternatives = []string{
		"Use AWS Auto Scaling for variable workloads",
		"Implement custom automation scripts",
		"Use Spot instances for non-critical workloads (higher savings, interruption risk)",
	}

	return rec
}

// determinePriority determines recommendation priority based on severity and savings
func (s *RecommendationService) determinePriority(severity WasteSeverity, savings float64) RecommendationPriority {
	// Critical severity always gets critical priority
	if severity == WasteSeverityCritical {
		return RecommendationPriorityCritical
	}

	// High severity or high savings gets high priority
	if severity == WasteSeverityHigh || savings > 500.0 {
		return RecommendationPriorityHigh
	}

	// Medium severity or moderate savings gets medium priority
	if severity == WasteSeverityMedium || savings > 100.0 {
		return RecommendationPriorityMedium
	}

	return RecommendationPriorityLow
}

// deduplicateRecommendations removes duplicate recommendations for same resource
func (s *RecommendationService) deduplicateRecommendations(recs []*Recommendation) []*Recommendation {
	seen := make(map[string]bool)
	var unique []*Recommendation

	for _, rec := range recs {
		key := rec.ResourceID + string(rec.RecommendationType)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, rec)
		}
	}

	return unique
}

// prioritizeRecommendations sorts recommendations by priority and savings
func (s *RecommendationService) prioritizeRecommendations(recs []*Recommendation) []*Recommendation {
	// Simple bubble sort for demonstration
	// In production, use sort.Slice
	for i := 0; i < len(recs); i++ {
		for j := i + 1; j < len(recs); j++ {
			if s.shouldSwap(recs[i], recs[j]) {
				recs[i], recs[j] = recs[j], recs[i]
			}
		}
	}
	return recs
}

// shouldSwap determines if two recommendations should be swapped for sorting
func (s *RecommendationService) shouldSwap(a, b *Recommendation) bool {
	priorityOrder := map[RecommendationPriority]int{
		RecommendationPriorityCritical: 4,
		RecommendationPriorityHigh:     3,
		RecommendationPriorityMedium:   2,
		RecommendationPriorityLow:      1,
	}

	aPriority := priorityOrder[a.Priority]
	bPriority := priorityOrder[b.Priority]

	// Higher priority comes first
	if aPriority != bPriority {
		return aPriority < bPriority
	}

	// Higher savings comes first for same priority
	return a.EstimatedSavings < b.EstimatedSavings
}

// generateRecommendationSummary creates a summary of recommendations
func (s *RecommendationService) generateRecommendationSummary(recs []*Recommendation) *RecommendationSummary {
	summary := &RecommendationSummary{
		ByType:     make(map[RecommendationType]int),
		ByPriority: make(map[RecommendationPriority]int),
		ByStatus:   make(map[RecommendationStatus]int),
	}

	for _, rec := range recs {
		summary.TotalCount++
		summary.ByType[rec.RecommendationType]++
		summary.ByPriority[rec.Priority]++
		summary.ByStatus[rec.Status]++
		summary.TotalEstimatedSavings += rec.EstimatedSavings

		if rec.Priority == RecommendationPriorityHigh ||
			rec.Priority == RecommendationPriorityCritical {
			summary.HighPriorityCount++
		}

		if rec.Priority == RecommendationPriorityCritical {
			summary.CriticalCount++
		}
	}

	return summary
}

// SaveRecommendations persists recommendations to the database
func (s *RecommendationService) SaveRecommendations(ctx context.Context, result *RecommendationResult) error {
	if len(result.Recommendations) == 0 {
		s.logger.Info("No recommendations to save")
		return nil
	}

	s.logger.Info("Saving recommendations",
		zap.Int("count", len(result.Recommendations)),
	)

	if err := s.repository.SaveRecommendations(ctx, result.Recommendations); err != nil {
		s.logger.Error("Failed to save recommendations", zap.Error(err))
		return fmt.Errorf("failed to save recommendations: %w", err)
	}

	s.logger.Info("Recommendations saved successfully")
	return nil
}

// GetActiveRecommendations retrieves all active recommendations
func (s *RecommendationService) GetActiveRecommendations(ctx context.Context) ([]*Recommendation, error) {
	return s.repository.GetActiveRecommendations(ctx)
}

// GetRecommendationsByResource retrieves recommendations for a specific resource
func (s *RecommendationService) GetRecommendationsByResource(ctx context.Context, resourceID string) ([]*Recommendation, error) {
	return s.repository.GetRecommendationsByResource(ctx, resourceID)
}

// UpdateRecommendationStatus updates the status of a recommendation
func (s *RecommendationService) UpdateRecommendationStatus(
	ctx context.Context,
	id uuid.UUID,
	status RecommendationStatus,
) error {
	if err := s.repository.UpdateRecommendationStatus(ctx, id, status); err != nil {
		return err
	}

	s.logger.Info("Recommendation status updated",
		zap.String("id", id.String()),
		zap.String("status", string(status)),
	)

	return nil
}
