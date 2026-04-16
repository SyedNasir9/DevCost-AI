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

// WasteDetectionService detects wasteful resource usage across AWS resources
type WasteDetectionService struct {
	logger     *logger.Logger
	repository ResourceUsageRepository
	config     *WasteDetectionConfig
}

// WasteDetectionConfig holds configuration for waste detection rules
type WasteDetectionConfig struct {
	// EC2 idle detection
	EC2IdleCPUThreshold    float64       // CPU threshold for idle (default: 5%)
	EC2IdleTimeWindow    time.Duration // Time window to check (default: 24h)
	EC2MinInstanceAge    time.Duration // Minimum instance age to consider (default: 1h)

	// EBS unattached detection
	EBSUnattachedDays    int // Days to consider volume as unattached (default: 7)

	// RDS underutilization
	RDSLowConnectionThreshold int     // Low connection threshold (default: 5)
	RDSLowCPUThreshold        float64 // CPU threshold (default: 10%)
	RDSLowIOPSThreshold       float64 // IOPS threshold (default: 100)
	RDSMinDBAge             time.Duration // Minimum DB age (default: 24h)
}

// DefaultWasteDetectionConfig returns default configuration
func DefaultWasteDetectionConfig() *WasteDetectionConfig {
	return &WasteDetectionConfig{
		EC2IdleCPUThreshold:       5.0,
		EC2IdleTimeWindow:         24 * time.Hour,
		EC2MinInstanceAge:         1 * time.Hour,
		EBSUnattachedDays:         7,
		RDSLowConnectionThreshold: 5,
		RDSLowCPUThreshold:        10.0,
		RDSLowIOPSThreshold:       100.0,
		RDSMinDBAge:               24 * time.Hour,
	}
}

// WasteResult represents a detected waste resource
type WasteResult struct {
	ResourceID      string              `json:"resource_id"`
	ResourceUUID    string              `json:"resource_uuid,omitempty"`
	ResourceType    string              `json:"resource_type"`
	ResourceName    string              `json:"resource_name,omitempty"`
	WasteType       WasteType           `json:"waste_type"`
	Reason          string              `json:"reason"`
	Severity        WasteSeverity       `json:"severity"`
	Confidence      float64             `json:"confidence"`
	EstimatedSavings float64            `json:"estimated_savings_usd,omitempty"`
	Details         *WasteDetails       `json:"details,omitempty"`
	DetectedAt      time.Time           `json:"detected_at"`
	Recommendation  string              `json:"recommendation"`
}

// WasteType represents the type of waste detected
type WasteType string

const (
	WasteTypeIdleEC2         WasteType = "idle_ec2"
	WasteTypeUnattachedEBS   WasteType = "unattached_ebs"
	WasteTypeUnderutilizedRDS WasteType = "underutilized_rds"
)

// WasteSeverity represents the severity of waste
type WasteSeverity string

const (
	WasteSeverityLow      WasteSeverity = "low"
	WasteSeverityMedium   WasteSeverity = "medium"
	WasteSeverityHigh     WasteSeverity = "high"
	WasteSeverityCritical WasteSeverity = "critical"
)

// WasteDetails contains detailed information about the waste
type WasteDetails struct {
	// EC2 idle details
	AvgCPUUtilization    float64   `json:"avg_cpu_utilization,omitempty"`
	MaxCPUUtilization    float64   `json:"max_cpu_utilization,omitempty"`
	MinCPUUtilization    float64   `json:"min_cpu_utilization,omitempty"`
	IdleDuration         time.Duration `json:"idle_duration,omitempty"`
	LastActivityTime     *time.Time  `json:"last_activity_time,omitempty"`

	// EBS unattached details
	UnattachedSince      *time.Time  `json:"unattached_since,omitempty"`
	VolumeSize           int         `json:"volume_size_gb,omitempty"`
	VolumeType           string      `json:"volume_type,omitempty"`
	DaysUnattached       int         `json:"days_unattached,omitempty"`

	// RDS underutilization details
	AvgConnections       float64   `json:"avg_connections,omitempty"`
	AvgCPUUtilization    float64   `json:"avg_cpu_utilization,omitempty"`
	AvgIOPS              float64   `json:"avg_iops,omitempty"`
	ConnectionCount      int         `json:"connection_count,omitempty"`
	LowActivityPeriods   int         `json:"low_activity_periods,omitempty"`
}

// WasteDetectionResult represents the complete detection result
type WasteDetectionResult struct {
	WasteResources    []*WasteResult      `json:"waste_resources"`
	Summary           *WasteSummary         `json:"summary"`
	ScanTimestamp     time.Time             `json:"scan_timestamp"`
	Duration          time.Duration         `json:"scan_duration"`
	ResourcesScanned  int                   `json:"resources_scanned"`
}

// WasteSummary provides statistics about detected waste
type WasteSummary struct {
	TotalWasteCount      int                      `json:"total_waste_count"`
	WasteByType          map[WasteType]int        `json:"waste_by_type"`
	WasteBySeverity      map[WasteSeverity]int    `json:"waste_by_severity"`
	TotalEstimatedSavings float64                 `json:"total_estimated_savings_usd"`
	HighPriorityCount    int                      `json:"high_priority_count"`
	MediumPriorityCount  int                      `json:"medium_priority_count"`
	LowPriorityCount     int                      `json:"low_priority_count"`
}

// ResourceUsageRepository defines the interface for accessing resource usage data
type ResourceUsageRepository interface {
	GetResourceUsage(ctx context.Context, resourceID string, metric string, since time.Time) ([]*ResourceUsage, error)
	GetResourceMetadata(ctx context.Context, resourceID string) (*ResourceMetadata, error)
	GetResourcesByType(ctx context.Context, resourceType models.ResourceType) ([]*models.Resource, error)
	GetAllActiveResources(ctx context.Context) ([]*models.Resource, error)
}

// ResourceUsage represents a single usage metric data point
type ResourceUsage struct {
	ResourceID   string
	MetricName   string
	Value        float64
	Timestamp    time.Time
	Unit         string
}

// ResourceMetadata represents additional resource metadata
type ResourceMetadata struct {
	ResourceID      string
	InstanceType    string
	LaunchTime      *time.Time
	VolumeSize      int
	VolumeType      string
	AttachedTo      string
	AttachmentTime  *time.Time
	DBInstanceClass string
	Engine          string
	ConnectionLimit int
}

// NewWasteDetectionService creates a new waste detection service
func NewWasteDetectionService(logger *logger.Logger, repository ResourceUsageRepository, config *WasteDetectionConfig) *WasteDetectionService {
	if config == nil {
		config = DefaultWasteDetectionConfig()
	}

	return &WasteDetectionService{
		logger:     logger,
		repository: repository,
		config:     config,
	}
}

// DetectWaste performs waste detection across all supported resource types
func (s *WasteDetectionService) DetectWaste(ctx context.Context) (*WasteDetectionResult, error) {
	startTime := time.Now()
	s.logger.Info("Starting waste detection scan",
		zap.Time("start_time", startTime),
	)

	var allWaste []*WasteResult
	
	// Get all active resources
	resources, err := s.repository.GetAllActiveResources(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch active resources", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch resources: %w", err)
	}

	s.logger.Info("Scanning resources for waste",
		zap.Int("resource_count", len(resources)),
	)

	// Detect EC2 waste
	ec2Waste, err := s.detectIdleEC2Instances(ctx, resources)
	if err != nil {
		s.logger.Error("EC2 waste detection failed", zap.Error(err))
	} else {
		allWaste = append(allWaste, ec2Waste...)
	}

	// Detect EBS waste
	ebsWaste, err := s.detectUnattachedVolumes(ctx, resources)
	if err != nil {
		s.logger.Error("EBS waste detection failed", zap.Error(err))
	} else {
		allWaste = append(allWaste, ebsWaste...)
	}

	// Detect RDS waste
	rdsWaste, err := s.detectUnderutilizedRDS(ctx, resources)
	if err != nil {
		s.logger.Error("RDS waste detection failed", zap.Error(err))
	} else {
		allWaste = append(allWaste, rdsWaste...)
	}

	// Generate summary
	summary := s.generateWasteSummary(allWaste)

	result := &WasteDetectionResult{
		WasteResources:   allWaste,
		Summary:          summary,
		ScanTimestamp:    startTime,
		Duration:         time.Since(startTime),
		ResourcesScanned: len(resources),
	}

	s.logger.Info("Waste detection scan completed",
		zap.Duration("duration", result.Duration),
		zap.Int("waste_found", len(allWaste)),
		zap.Float64("estimated_savings", summary.TotalEstimatedSavings),
	)

	return result, nil
}

// DetectIdleEC2Instances detects EC2 instances with low CPU utilization
func (s *WasteDetectionService) DetectIdleEC2Instances(ctx context.Context, resources []*models.Resource) ([]*WasteResult, error) {
	return s.detectIdleEC2Instances(ctx, resources)
}

// DetectUnattachedVolumes detects EBS volumes not attached to any instance
func (s *WasteDetectionService) DetectUnattachedVolumes(ctx context.Context, resources []*models.Resource) ([]*WasteResult, error) {
	return s.detectUnattachedVolumes(ctx, resources)
}

// DetectUnderutilizedRDS detects RDS instances with low utilization
func (s *WasteDetectionService) DetectUnderutilizedRDS(ctx context.Context, resources []*models.Resource) ([]*WasteResult, error) {
	return s.detectUnderutilizedRDS(ctx, resources)
}

// detectIdleEC2Instances detects EC2 instances that are idle
func (s *WasteDetectionService) detectIdleEC2Instances(ctx context.Context, resources []*models.Resource) ([]*WasteResult, error) {
	s.logger.Info("Detecting idle EC2 instances",
		zap.Float64("cpu_threshold", s.config.EC2IdleCPUThreshold),
		zap.Duration("time_window", s.config.EC2IdleTimeWindow),
	)

	var results []*WasteResult
	since := time.Now().Add(-s.config.EC2IdleTimeWindow)

	for _, resource := range resources {
		if resource.ResourceType != models.ResourceTypeEC2 {
			continue
		}

		// Skip if instance is too new
		if time.Since(resource.CreatedAt) < s.config.EC2MinInstanceAge {
			continue
		}

		// Skip if not in running state
		if resource.State != models.ResourceStateRunning {
			continue
		}

		// Get CPU utilization data
		usage, err := s.repository.GetResourceUsage(ctx, resource.ResourceID, "CPUUtilization", since)
		if err != nil {
			s.logger.Warn("Failed to get CPU usage for instance",
				zap.String("resource_id", resource.ResourceID),
				zap.Error(err),
			)
			continue
		}

		if len(usage) == 0 {
			// No data available - might be a new instance
			continue
		}

		// Calculate average CPU
		avgCPU, maxCPU, minCPU := calculateCPUStats(usage)

		// Check if instance is idle
		if avgCPU < s.config.EC2IdleCPUThreshold {
			waste := &WasteResult{
				ResourceID:       resource.ResourceID,
				ResourceUUID:     resource.ID.String(),
				ResourceType:     string(resource.ResourceType),
				ResourceName:     resource.Name,
				WasteType:        WasteTypeIdleEC2,
				Reason:           fmt.Sprintf("Average CPU utilization %.2f%% below threshold %.2f%% over last %v", avgCPU, s.config.EC2IdleCPUThreshold, s.config.EC2IdleTimeWindow),
				Severity:         determineIdleSeverity(avgCPU, s.config.EC2IdleCPUThreshold),
				Confidence:       calculateIdleConfidence(avgCPU, len(usage)),
				DetectedAt:       time.Now(),
				Recommendation:   "Consider stopping or terminating this idle instance",
				Details: &WasteDetails{
					AvgCPUUtilization: avgCPU,
					MaxCPUUtilization: maxCPU,
					MinCPUUtilization: minCPU,
					IdleDuration:     s.config.EC2IdleTimeWindow,
				},
			}

			// Estimate savings (rough calculation: $0.05 per hour for t3.micro equivalent)
			waste.EstimatedSavings = estimateEC2Savings(resource, s.config.EC2IdleTimeWindow)
			results = append(results, waste)

			s.logger.Debug("Idle EC2 detected",
				zap.String("resource_id", resource.ResourceID),
				zap.Float64("avg_cpu", avgCPU),
			)
		}
	}

	s.logger.Info("Idle EC2 detection complete",
		zap.Int("idle_instances_found", len(results)),
	)

	return results, nil
}

// detectUnattachedVolumes detects EBS volumes that are not attached
func (s *WasteDetectionService) detectUnattachedVolumes(ctx context.Context, resources []*models.Resource) ([]*WasteResult, error) {
	s.logger.Info("Detecting unattached EBS volumes",
		zap.Int("unattached_days_threshold", s.config.EBSUnattachedDays),
	)

	var results []*WasteResult
	cutoffTime := time.Now().AddDate(0, 0, -s.config.EBSUnattachedDays)

	for _, resource := range resources {
		if resource.ResourceType != models.ResourceTypeEBS {
			continue
		}

		// Check if volume is available (not attached)
		if resource.State != models.ResourceStateAvailable {
			continue
		}

		// Get metadata for volume details
		metadata, err := s.repository.GetResourceMetadata(ctx, resource.ResourceID)
		if err != nil {
			s.logger.Warn("Failed to get metadata for volume",
				zap.String("resource_id", resource.ResourceID),
				zap.Error(err),
			)
			// Continue with basic detection
		}

		// Check if volume has been unattached for threshold period
		unattachedSince := resource.UpdatedAt
		if metadata != nil && metadata.AttachmentTime != nil {
			unattachedSince = *metadata.AttachmentTime
		}

		if unattachedSince.Before(cutoffTime) {
			waste := &WasteResult{
				ResourceID:       resource.ResourceID,
				ResourceUUID:     resource.ID.String(),
				ResourceType:     string(resource.ResourceType),
				ResourceName:     resource.Name,
				WasteType:        WasteTypeUnattachedEBS,
				Reason:           fmt.Sprintf("Volume unattached for more than %d days", s.config.EBSUnattachedDays),
				Severity:         WasteSeverityMedium,
				Confidence:       0.95,
				DetectedAt:       time.Now(),
				Recommendation:   "Delete this unattached volume if no longer needed, or create a snapshot for backup",
				Details: &WasteDetails{
					UnattachedSince: &unattachedSince,
					DaysUnattached:  int(time.Since(unattachedSince).Hours() / 24),
				},
			}

			// Add volume details if available
			if metadata != nil {
				waste.Details.VolumeSize = metadata.VolumeSize
				waste.Details.VolumeType = metadata.VolumeType
			}

			// Estimate savings (rough calculation: $0.10 per GB per month)
			if waste.Details.VolumeSize > 0 {
				waste.EstimatedSavings = float64(waste.Details.VolumeSize) * 0.10
			}

			results = append(results, waste)

			s.logger.Debug("Unattached EBS detected",
				zap.String("resource_id", resource.ResourceID),
				zap.Time("unattached_since", unattachedSince),
			)
		}
	}

	s.logger.Info("Unattached EBS detection complete",
		zap.Int("unattached_volumes_found", len(results)),
	)

	return results, nil
}

// detectUnderutilizedRDS detects RDS instances with low utilization
func (s *WasteDetectionService) detectUnderutilizedRDS(ctx context.Context, resources []*models.Resource) ([]*WasteResult, error) {
	s.logger.Info("Detecting underutilized RDS instances",
		zap.Int("connection_threshold", s.config.RDSLowConnectionThreshold),
		zap.Float64("cpu_threshold", s.config.RDSLowCPUThreshold),
	)

	var results []*WasteResult
	since := time.Now().Add(-24 * time.Hour) // Check last 24 hours

	for _, resource := range resources {
		if resource.ResourceType != models.ResourceTypeRDS {
			continue
		}

		// Skip if DB is too new
		if time.Since(resource.CreatedAt) < s.config.RDSMinDBAge {
			continue
		}

		// Only check available databases
		if resource.State != models.ResourceStateAvailable {
			continue
		}

		// Get connection count data
		connectionUsage, err := s.repository.GetResourceUsage(ctx, resource.ResourceID, "DatabaseConnections", since)
		if err != nil {
			s.logger.Warn("Failed to get connection count",
				zap.String("resource_id", resource.ResourceID),
				zap.Error(err),
			)
			continue
		}

		// Get CPU utilization data
		cpuUsage, err := s.repository.GetResourceUsage(ctx, resource.ResourceID, "CPUUtilization", since)
		if err != nil {
			s.logger.Warn("Failed to get CPU usage for DB",
				zap.String("resource_id", resource.ResourceID),
				zap.Error(err),
			)
			continue
		}

		if len(connectionUsage) == 0 || len(cpuUsage) == 0 {
			continue
		}

		// Calculate metrics
		avgConnections := calculateAverage(connectionUsage)
		avgCPU, _, _ := calculateCPUStats(cpuUsage)
		lowActivityPeriods := countLowActivityPeriods(connectionUsage, float64(s.config.RDSLowConnectionThreshold))

		// Determine if underutilized
		isUnderutilized := avgConnections < float64(s.config.RDSLowConnectionThreshold) ||
			(avgCPU < s.config.RDSLowCPUThreshold && avgConnections < 10)

		if isUnderutilized {
			waste := &WasteResult{
				ResourceID:       resource.ResourceID,
				ResourceUUID:     resource.ID.String(),
				ResourceType:     string(resource.ResourceType),
				ResourceName:     resource.Name,
				WasteType:        WasteTypeUnderutilizedRDS,
				Reason:           fmt.Sprintf("Low activity detected: avg %.1f connections, %.1f%% CPU", avgConnections, avgCPU),
				Severity:         determineRDSSeverity(avgConnections, avgCPU),
				Confidence:       calculateRDSConfidence(avgConnections, avgCPU, len(connectionUsage)),
				DetectedAt:       time.Now(),
				Recommendation:   "Consider downsizing this database or using Aurora Serverless for variable workloads",
				Details: &WasteDetails{
					AvgConnections:   avgConnections,
					AvgCPUUtilization: avgCPU,
					ConnectionCount:  len(connectionUsage),
					LowActivityPeriods: lowActivityPeriods,
				},
			}

			// Estimate savings (rough calculation: depends on instance class)
			waste.EstimatedSavings = estimateRDSSavings(resource)
			results = append(results, waste)

			s.logger.Debug("Underutilized RDS detected",
				zap.String("resource_id", resource.ResourceID),
				zap.Float64("avg_connections", avgConnections),
				zap.Float64("avg_cpu", avgCPU),
			)
		}
	}

	s.logger.Info("Underutilized RDS detection complete",
		zap.Int("underutilized_dbs_found", len(results)),
	)

	return results, nil
}

// generateWasteSummary creates a summary of detected waste
func (s *WasteDetectionService) generateWasteSummary(wasteResults []*WasteResult) *WasteSummary {
	summary := &WasteSummary{
		WasteByType:     make(map[WasteType]int),
		WasteBySeverity: make(map[WasteSeverity]int),
	}

	for _, waste := range wasteResults {
		summary.TotalWasteCount++
		summary.WasteByType[waste.WasteType]++
		summary.WasteBySeverity[waste.Severity]++
		summary.TotalEstimatedSavings += waste.EstimatedSavings

		switch waste.Severity {
		case WasteSeverityHigh, WasteSeverityCritical:
			summary.HighPriorityCount++
		case WasteSeverityMedium:
			summary.MediumPriorityCount++
		case WasteSeverityLow:
			summary.LowPriorityCount++
		}
	}

	return summary
}

// Helper functions

func calculateCPUStats(usage []*ResourceUsage) (avg, max, min float64) {
	if len(usage) == 0 {
		return 0, 0, 0
	}

	var sum float64
	max = usage[0].Value
	min = usage[0].Value

	for _, u := range usage {
		sum += u.Value
		if u.Value > max {
			max = u.Value
		}
		if u.Value < min {
			min = u.Value
		}
	}

	avg = sum / float64(len(usage))
	return
}

func calculateAverage(usage []*ResourceUsage) float64 {
	if len(usage) == 0 {
		return 0
	}

	var sum float64
	for _, u := range usage {
		sum += u.Value
	}

	return sum / float64(len(usage))
}

func countLowActivityPeriods(usage []*ResourceUsage, threshold float64) int {
	count := 0
	for _, u := range usage {
		if u.Value < threshold {
			count++
		}
	}
	return count
}

func determineIdleSeverity(avgCPU, threshold float64) WasteSeverity {
	percentage := (avgCPU / threshold) * 100
	switch {
	case percentage < 10:
		return WasteSeverityCritical
	case percentage < 30:
		return WasteSeverityHigh
	case percentage < 60:
		return WasteSeverityMedium
	default:
		return WasteSeverityLow
	}
}

func determineRDSSeverity(connections, cpu float64) WasteSeverity {
	if connections < 1 && cpu < 5 {
		return WasteSeverityHigh
	} else if connections < 3 && cpu < 10 {
		return WasteSeverityMedium
	}
	return WasteSeverityLow
}

func calculateIdleConfidence(avgCPU float64, dataPoints int) float64 {
	// Higher confidence with more data points and lower CPU
	confidence := 0.5

	// Adjust for CPU level
	if avgCPU < 1.0 {
		confidence += 0.3
	} else if avgCPU < 3.0 {
		confidence += 0.2
	}

	// Adjust for data sample size
	if dataPoints > 100 {
		confidence += 0.2
	} else if dataPoints > 50 {
		confidence += 0.1
	}

	return min(confidence, 1.0)
}

func calculateRDSConfidence(connections, cpu float64, dataPoints int) float64 {
	confidence := 0.5

	// Lower connections = higher confidence of underutilization
	if connections < 1 {
		confidence += 0.3
	} else if connections < 3 {
		confidence += 0.2
	}

	// Low CPU adds confidence
	if cpu < 5 {
		confidence += 0.2
	}

	// More data points = higher confidence
	if dataPoints > 20 {
		confidence += 0.15
	}

	return min(confidence, 1.0)
}

func estimateEC2Savings(resource *models.Resource, idleDuration time.Duration) float64 {
	// Rough estimation based on instance running hours
	// Assuming $0.05/hour for small instances, scaled by instance type
	hourlyRate := 0.05

	// Adjust based on instance type if available
	if itype, ok := resource.Metadata["instance_type"]; ok {
		switch itype {
		case "t3.micro", "t2.micro":
			hourlyRate = 0.0104
		case "t3.small", "t2.small":
			hourlyRate = 0.0208
		case "t3.medium", "t2.medium":
			hourlyRate = 0.0416
		case "t3.large", "t2.large":
			hourlyRate = 0.0832
		case "m5.large":
			hourlyRate = 0.096
		case "m5.xlarge":
			hourlyRate = 0.192
		}
	}

	hours := idleDuration.Hours()
	return hourlyRate * hours
}

func estimateRDSSavings(resource *models.Resource) float64 {
	// Rough monthly cost estimation based on instance class
	monthlyRate := 15.0 // Default for small instances

	if dbClass, ok := resource.Metadata["db_instance_class"]; ok {
		switch dbClass {
		case "db.t3.micro":
			monthlyRate = 12.0
		case "db.t3.small":
			monthlyRate = 25.0
		case "db.t3.medium":
			monthlyRate = 50.0
		case "db.t3.large":
			monthlyRate = 100.0
		case "db.m5.large":
			monthlyRate = 150.0
		}
	}

	return monthlyRate
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
