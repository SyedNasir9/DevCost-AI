package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/internal/services/ec2"
	"devcost-ai/internal/services/ebs"
	"devcost-ai/internal/services/rds"
	"devcost-ai/pkg/logger"
)

// UnifiedResourceCollector collects resources from multiple AWS services
type UnifiedResourceCollector struct {
	client          *aws.Client
	logger          *logger.Logger
	ec2Discovery   *ec2.DiscoveryService
	rdsDiscovery   *rds.DiscoveryService
	ebsDiscovery   *ebs.DiscoveryService
}

// NewUnifiedResourceCollector creates a new unified resource collector
func NewUnifiedResourceCollector(awsClient *aws.Client) *UnifiedResourceCollector {
	return &UnifiedResourceCollector{
		client:          awsClient,
		logger:          awsClient.logger,
		ec2Discovery:    ec2.NewDiscoveryService(awsClient),
		rdsDiscovery:    rds.NewDiscoveryService(awsClient),
		ebsDiscovery:    ebs.NewDiscoveryService(awsClient),
	}
}

// CollectAllResources collects all resources from all supported services
func (c *UnifiedResourceCollector) CollectAllResources(ctx context.Context) (*ResourceCollectionResult, error) {
	c.logger.Info("Starting unified resource collection")
	start := time.Now()

	result := &ResourceCollectionResult{
		EC2Instances: []*models.Resource{},
		RDSInstances: []*models.Resource{},
		EBSVolumes:    []*models.Resource{},
		StartTime:     start,
	}

	// Collect EC2 instances
	c.logger.Info("Collecting EC2 instances")
	ec2StartTime := time.Now()
	ec2Instances, err := c.ec2Discovery.DiscoverAllInstances(ctx)
	if err != nil {
		c.logger.Error("Failed to collect EC2 instances", zap.Error(err))
		result.Errors = append(result.Errors, fmt.Errorf("EC2 discovery failed: %w", err))
	} else {
		// Convert to base Resource model
		for _, instance := range ec2Instances {
			result.EC2Instances = append(result.EC2Instances, instance.ToResource())
		}
		result.EC2Count = len(ec2Instances)
		result.EC2Duration = time.Since(ec2StartTime)
		c.logger.Info("EC2 instances collected",
			zap.Int("count", result.EC2Count),
			zap.Duration("duration", result.EC2Duration),
		)
	}

	// Collect RDS instances
	c.logger.Info("Collecting RDS instances")
	rdsStartTime := time.Now()
	rdsInstances, err := c.rdsDiscovery.DiscoverAllDBInstances(ctx)
	if err != nil {
		c.logger.Error("Failed to collect RDS instances", zap.Error(err))
		result.Errors = append(result.Errors, fmt.Errorf("RDS discovery failed: %w", err))
	} else {
		// Convert to base Resource model
		for _, instance := range rdsInstances {
			result.RDSInstances = append(result.RDSInstances, instance.ToResource())
		}
		result.RDSCount = len(rdsInstances)
		result.RDSDuration = time.Since(rdsStartTime)
		c.logger.Info("RDS instances collected",
			zap.Int("count", result.RDSCount),
			zap.Duration("duration", result.RDSDuration),
		)
	}

	// Collect EBS volumes
	c.logger.Info("Collecting EBS volumes")
	ebsStartTime := time.Now()
	ebsVolumes, err := c.ebsDiscovery.DiscoverAllVolumes(ctx)
	if err != nil {
		c.logger.Error("Failed to collect EBS volumes", zap.Error(err))
		result.Errors = append(result.Errors, fmt.Errorf("EBS discovery failed: %w", err))
	} else {
		// Convert to base Resource model
		for _, volume := range ebsVolumes {
			result.EBSVolumes = append(result.EBSVolumes, volume.ToResource())
		}
		result.EBSCount = len(ebsVolumes)
		result.EBSDuration = time.Since(ebsStartTime)
		c.logger.Info("EBS volumes collected",
			zap.Int("count", result.EBSCount),
			zap.Duration("duration", result.EBSDuration),
		)
	}

	// Calculate totals
	result.TotalCount = result.EC2Count + result.RDSCount + result.EBSCount
	result.Duration = time.Since(start)

	// Generate summary
	result.Summary = c.generateSummary(result)

	c.logger.Info("Unified resource collection completed",
		zap.Int("total_resources", result.TotalCount),
		zap.Duration("total_duration", result.Duration),
		zap.Int("error_count", len(result.Errors)),
	)

	return result, nil
}

// CollectResourcesByType collects resources of specific types
func (c *UnifiedResourceCollector) CollectResourcesByType(ctx context.Context, resourceTypes []models.ResourceType) (*ResourceCollectionResult, error) {
	c.logger.Info("Starting filtered resource collection",
		zap.Strings("resource_types", func() []string {
			types := make([]string, len(resourceTypes))
			for i, rt := range resourceTypes {
				types[i] = string(rt)
			}
			return types
		}()),
	)

	result := &ResourceCollectionResult{
		StartTime: time.Now(),
	}

	for _, resourceType := range resourceTypes {
		switch resourceType {
		case models.ResourceTypeEC2:
			ec2Instances, err := c.ec2Discovery.DiscoverAllInstances(ctx)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("EC2 discovery failed: %w", err))
				continue
			}
			for _, instance := range ec2Instances {
				result.EC2Instances = append(result.EC2Instances, instance.ToResource())
			}
			result.EC2Count = len(ec2Instances)

		case models.ResourceTypeRDS:
			rdsInstances, err := c.rdsDiscovery.DiscoverAllDBInstances(ctx)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("RDS discovery failed: %w", err))
				continue
			}
			for _, instance := range rdsInstances {
				result.RDSInstances = append(result.RDSInstances, instance.ToResource())
			}
			result.RDSCount = len(rdsInstances)

		case models.ResourceTypeEBS:
			ebsVolumes, err := c.ebsDiscovery.DiscoverAllVolumes(ctx)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("EBS discovery failed: %w", err))
				continue
			}
			for _, volume := range ebsVolumes {
				result.EBSVolumes = append(result.EBSVolumes, volume.ToResource())
			}
			result.EBSCount = len(ebsVolumes)

		default:
			result.Errors = append(result.Errors, fmt.Errorf("unsupported resource type: %s", resourceType))
		}
	}

	result.TotalCount = result.EC2Count + result.RDSCount + result.EBSCount
	result.Duration = time.Since(result.StartTime)
	result.Summary = c.generateSummary(result)

	return result, nil
}

// GetResourceStatistics provides statistics about all discovered resources
func (c *UnifiedResourceCollector) GetResourceStatistics(ctx context.Context) (*ResourceStatistics, error) {
	c.logger.Info("Collecting resource statistics")

	stats := &ResourceStatistics{
		ByType:     make(map[string]int),
		ByState:    make(map[string]int),
		ByRegion:   make(map[string]int),
		ByProvider: make(map[string]int),
	}

	// Collect EC2 statistics
	ec2Instances, err := c.ec2Discovery.DiscoverAllInstances(ctx)
	if err != nil {
		c.logger.Error("Failed to collect EC2 instances for statistics", zap.Error(err))
	} else {
		for _, instance := range ec2Instances {
			stats.ByType["EC2"]++
			stats.ByState[string(instance.State)]++
			stats.ByRegion[instance.Region]++
			stats.ByProvider["aws"]++
		}
	}

	// Collect RDS statistics
	rdsInstances, err := c.rdsDiscovery.DiscoverAllDBInstances(ctx)
	if err != nil {
		c.logger.Error("Failed to collect RDS instances for statistics", zap.Error(err))
	} else {
		for _, instance := range rdsInstances {
			stats.ByType["RDS"]++
			stats.ByState[string(instance.Status)]++
			stats.ByRegion[instance.Region]++
			stats.ByProvider["aws"]++
		}
	}

	// Collect EBS statistics
	ebsVolumes, err := c.ebsDiscovery.DiscoverAllVolumes(ctx)
	if err != nil {
		c.logger.Error("Failed to collect EBS volumes for statistics", zap.Error(err))
	} else {
		for _, volume := range ebsVolumes {
			stats.ByType["EBS"]++
			stats.ByState[string(volume.State)]++
			stats.ByRegion[volume.AvailabilityZone]++
			stats.ByProvider["aws"]++

			// Additional EBS-specific statistics
			stats.EBSStats.TotalVolumes++
			stats.EBSStats.TotalSizeGB += int(volume.Size)
			if volume.IsAttached {
				stats.EBSStats.AttachedVolumes++
			} else {
				stats.EBSStats.UnattachedVolumes++
			}
			if volume.Encrypted {
				stats.EBSStats.EncryptedVolumes++
			}
		}
	}

	// Calculate totals
	for _, count := range stats.ByType {
		stats.TotalResources += count
	}

	c.logger.Info("Resource statistics collected",
		zap.Int("total_resources", stats.TotalResources),
		zap.Int("ec2_count", stats.ByType["EC2"]),
		zap.Int("rds_count", stats.ByType["RDS"]),
		zap.Int("ebs_count", stats.ByType["EBS"]),
	)

	return stats, nil
}

// generateSummary generates a human-readable summary of the collection result
func (c *UnifiedResourceCollector) generateSummary(result *ResourceCollectionResult) string {
	summary := fmt.Sprintf("Resource Collection Summary:\n")
	summary += fmt.Sprintf("  Total Resources: %d\n", result.TotalCount)
	summary += fmt.Sprintf("  Duration: %v\n", result.Duration)
	summary += fmt.Sprintf("  EC2 Instances: %d (%v)\n", result.EC2Count, result.EC2Duration)
	summary += fmt.Sprintf("  RDS Instances: %d (%v)\n", result.RDSCount, result.RDSDuration)
	summary += fmt.Sprintf("  EBS Volumes: %d (%v)\n", result.EBSCount, result.EBSDuration)

	if len(result.Errors) > 0 {
		summary += fmt.Sprintf("  Errors: %d\n", len(result.Errors))
		for _, err := range result.Errors {
			summary += fmt.Sprintf("    - %v\n", err)
		}
	}

	return summary
}

// ResourceCollectionResult represents the result of a resource collection operation
type ResourceCollectionResult struct {
	EC2Instances []*models.Resource `json:"ec2_instances"`
	RDSInstances []*models.Resource `json:"rds_instances"`
	EBSVolumes    []*models.Resource `json:"ebs_volumes"`

	// Statistics
	EC2Count    int           `json:"ec2_count"`
	RDSCount    int           `json:"rds_count"`
	EBSCount    int           `json:"ebs_count"`
	TotalCount  int           `json:"total_count"`

	// Timing
	StartTime     time.Time `json:"start_time"`
	Duration      time.Duration `json:"duration"`
	EC2Duration   time.Duration `json:"ec2_duration"`
	RDSDuration   time.Duration `json:"rds_duration"`
	EBSDuration   time.Duration `json:"ebs_duration"`

	// Errors
	Errors  []error `json:"errors"`
	Summary string  `json:"summary"`
}

// GetAllResources returns all resources as a single slice
func (r *ResourceCollectionResult) GetAllResources() []*models.Resource {
	var allResources []*models.Resource
	allResources = append(allResources, r.EC2Instances...)
	allResources = append(allResources, r.RDSInstances...)
	allResources = append(allResources, r.EBSVolumes...)
	return allResources
}

// GetResourcesByType returns resources of a specific type
func (r *ResourceCollectionResult) GetResourcesByType(resourceType models.ResourceType) []*models.Resource {
	switch resourceType {
	case models.ResourceTypeEC2:
		return r.EC2Instances
	case models.ResourceTypeRDS:
		return r.RDSInstances
	case models.ResourceTypeEBS:
		return r.EBSVolumes
	default:
		return []*models.Resource{}
	}
}

// ResourceStatistics represents statistics about discovered resources
type ResourceStatistics struct {
	TotalResources int                       `json:"total_resources"`
	ByType         map[string]int            `json:"by_type"`
	ByState        map[string]int            `json:"by_state"`
	ByRegion       map[string]int            `json:"by_region"`
	ByProvider     map[string]int            `json:"by_provider"`
	EBSStats       EBSStatistics             `json:"ebs_stats"`
}

// EBSStatistics represents EBS-specific statistics
type EBSStatistics struct {
	TotalVolumes      int `json:"total_volumes"`
	AttachedVolumes   int `json:"attached_volumes"`
	UnattachedVolumes int `json:"unattached_volumes"`
	TotalSizeGB       int `json:"total_size_gb"`
	EncryptedVolumes  int `json:"encrypted_volumes"`
}

// ExampleUsage demonstrates how to use the unified resource collector
func ExampleUsage() {
	fmt.Println("=== Unified Resource Collector Example ===")

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	// Create AWS client (in real usage, this would be properly configured)
	awsClient := &aws.Client{
		Region: "us-east-1",
		logger: logger,
	}

	// Create unified collector
	collector := NewUnifiedResourceCollector(awsClient)

	// Collect all resources
	ctx := context.Background()
	result, err := collector.CollectAllResources(ctx)
	if err != nil {
		log.Printf("Failed to collect resources: %v", err)
		return
	}

	fmt.Printf("Collection Summary:\n%s\n", result.Summary)

	// Get all resources as a single slice
	allResources := result.GetAllResources()
	fmt.Printf("Total resources collected: %d\n", len(allResources))

	// Get resources by type
	ec2Resources := result.GetResourcesByType(models.ResourceTypeEC2)
	fmt.Printf("EC2 resources: %d\n", len(ec2Resources))

	// Get statistics
	stats, err := collector.GetResourceStatistics(ctx)
	if err != nil {
		log.Printf("Failed to get statistics: %v", err)
		return
	}

	fmt.Printf("Resource Statistics:\n")
	fmt.Printf("  Total: %d\n", stats.TotalResources)
	fmt.Printf("  By Type: %v\n", stats.ByType)
	fmt.Printf("  By State: %v\n", stats.ByState)
	fmt.Printf("  By Region: %v\n", stats.ByRegion)
	fmt.Printf("  EBS Stats: %+v\n", stats.EBSStats)
}

// StartPeriodicCollection starts a background goroutine for periodic resource collection
func (c *UnifiedResourceCollector) StartPeriodicCollection(ctx context.Context, interval time.Duration, callback func(*ResourceCollectionResult)) {
	c.logger.Info("Starting periodic resource collection",
		zap.Duration("interval", interval),
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping periodic resource collection")
			return
		case <-ticker.C:
			c.logger.Debug("Starting periodic resource collection")
			result, err := c.CollectAllResources(ctx)
			if err != nil {
				c.logger.Error("Periodic resource collection failed", zap.Error(err))
				continue
			}

			// Call the callback function with the result
			if callback != nil {
				callback(result)
			}

			c.logger.Debug("Periodic resource collection completed",
				zap.Int("total_resources", result.TotalCount),
				zap.Duration("duration", result.Duration),
			)
		}
	}
}
