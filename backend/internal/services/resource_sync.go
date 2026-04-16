package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/internal/repositories"
	"devcost-ai/internal/services/ec2"
	"devcost-ai/internal/services/ebs"
	"devcost-ai/internal/services/rds"
	"devcost-ai/pkg/logger"
)

// ResourceSyncService orchestrates resource discovery and persistence
type ResourceSyncService struct {
	awsClient       *aws.Client
	repository      *repositories.ResourceRepository
	logger          *logger.Logger
	
	// Discovery services
	ec2Discovery    *ec2.DiscoveryService
	rdsDiscovery    *rds.DiscoveryService
	ebsDiscovery    *ebs.DiscoveryService
	unifiedCollector *UnifiedResourceCollector
	
	// Configuration
	config          *SyncConfig
	
	// Sync state
	isRunning       bool
	lastSyncTime    time.Time
	syncCount       int64
	syncErrors      []SyncError
}

// SyncConfig defines configuration for the sync service
type SyncConfig struct {
	Interval        time.Duration `json:"interval"`         // Sync interval (e.g., 5 * time.Minute)
	Timeout         time.Duration `json:"timeout"`          // Timeout for individual service syncs
	MaxRetries      int           `json:"max_retries"`      // Maximum retries per service
	RetryDelay      time.Duration `json:"retry_delay"`      // Delay between retries
	EnableEC2       bool          `json:"enable_ec2"`       // Enable EC2 discovery
	EnableRDS       bool          `json:"enable_rds"`       // Enable RDS discovery
	EnableEBS       bool          `json:"enable_ebs"`       // Enable EBS discovery
	EnableStatistics bool         `json:"enable_statistics"` // Enable statistics collection
	BatchSize       int           `json:"batch_size"`       // Batch size for database operations
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	StartTime       time.Time                    `json:"start_time"`
	EndTime         time.Time                    `json:"end_time"`
	Duration        time.Duration                `json:"duration"`
	TotalResources  int                         `json:"total_resources"`
	SuccessCount    int                         `json:"success_count"`
	FailureCount    int                         `json:"failure_count"`
	
	// Service-specific results
	EC2Result       *ServiceSyncResult         `json:"ec2_result,omitempty"`
	RDSResult       *ServiceSyncResult         `json:"rds_result,omitempty"`
	EBSResult       *ServiceSyncResult         `json:"ebs_result,omitempty"`
	
	// Database operations
	DatabaseOps     *DatabaseSyncResult        `json:"database_ops,omitempty"`
	
	// Errors
	Errors          []SyncError                 `json:"errors"`
	
	// Statistics
	Statistics       *repositories.ResourceStatistics `json:"statistics,omitempty"`
}

// ServiceSyncResult represents the result of a specific service sync
type ServiceSyncResult struct {
	ServiceName     string        `json:"service_name"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	Duration        time.Duration `json:"duration"`
	ResourcesFound  int           `json:"resources_found"`
	ResourcesSaved  int           `json:"resources_saved"`
	Success         bool          `json:"success"`
	Error           error         `json:"error,omitempty"`
	Retries         int           `json:"retries"`
}

// DatabaseSyncResult represents database operation results
type DatabaseSyncResult struct {
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	Duration        time.Duration `json:"duration"`
	TotalOperations int           `json:"total_operations"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	BatchesProcessed int          `json:"batches_processed"`
	Error           error         `json:"error,omitempty"`
}

// SyncError represents an error that occurred during sync
type SyncError struct {
	Timestamp   time.Time `json:"timestamp"`
	Service     string    `json:"service"`
	Operation   string    `json:"operation"`
	Error       string    `json:"error"`
	ResourceID  string    `json:"resource_id,omitempty"`
	Retryable   bool      `json:"retryable"`
}

// NewResourceSyncService creates a new resource sync service
func NewResourceSyncService(
	awsClient *aws.Client,
	repository *repositories.ResourceRepository,
	logger *logger.Logger,
	config *SyncConfig,
) *ResourceSyncService {
	return &ResourceSyncService{
		awsClient:          awsClient,
		repository:         repository,
		logger:             logger,
		ec2Discovery:       ec2.NewDiscoveryService(awsClient),
		rdsDiscovery:       rds.NewDiscoveryService(awsClient),
		ebsDiscovery:       ebs.NewDiscoveryService(awsClient),
		unifiedCollector:   NewUnifiedResourceCollector(awsClient),
		config:             config,
		syncErrors:         []SyncError{},
	}
}

// Start starts the scheduled sync service
func (s *ResourceSyncService) Start(ctx context.Context) error {
	if s.isRunning {
		return fmt.Errorf("sync service is already running")
	}

	s.logger.Info("Starting resource sync service",
		zap.Duration("interval", s.config.Interval),
		zap.Duration("timeout", s.config.Timeout),
		zap.Bool("enable_ec2", s.config.EnableEC2),
		zap.Bool("enable_rds", s.config.EnableRDS),
		zap.Bool("enable_ebs", s.config.EnableEBS),
	)

	s.isRunning = true
	s.lastSyncTime = time.Now()

	// Run initial sync immediately
	go func() {
		if err := s.runSync(ctx); err != nil {
			s.logger.Error("Initial sync failed", zap.Error(err))
		}
	}()

	// Start scheduled sync
	go s.scheduleSync(ctx)

	s.logger.Info("Resource sync service started successfully")
	return nil
}

// Stop stops the scheduled sync service
func (s *ResourceSyncService) Stop() error {
	if !s.isRunning {
		return fmt.Errorf("sync service is not running")
	}

	s.logger.Info("Stopping resource sync service")
	s.isRunning = false

	s.logger.Info("Resource sync service stopped")
	return nil
}

// SyncNow performs an immediate sync operation
func (s *ResourceSyncService) SyncNow(ctx context.Context) (*SyncResult, error) {
	s.logger.Info("Performing immediate sync")
	return s.runSync(ctx)
}

// scheduleSync runs the sync operation on a schedule
func (s *ResourceSyncService) scheduleSync(ctx context.Context) {
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Sync service context cancelled, stopping")
			return
		case <-ticker.C:
			if !s.isRunning {
				s.logger.Info("Sync service stopped, exiting scheduler")
				return
			}

			s.logger.Debug("Starting scheduled sync")
			if err := s.runSync(ctx); err != nil {
				s.logger.Error("Scheduled sync failed", zap.Error(err))
			}
		}
	}
}

// runSync performs a complete sync operation
func (s *ResourceSyncService) runSync(ctx context.Context) (*SyncResult, error) {
	syncStartTime := time.Now()
	s.logger.Info("Starting resource sync operation")

	result := &SyncResult{
		StartTime: syncStartTime,
		Errors:    []SyncError{},
	}

	// Discover resources from all enabled services
	allResources, err := s.discoverResources(ctx, result)
	if err != nil {
		s.logger.Error("Resource discovery failed", zap.Error(err))
		result.Errors = append(result.Errors, SyncError{
			Timestamp: time.Now(),
			Service:   "discovery",
			Operation: "discover_resources",
			Error:     err.Error(),
			Retryable: true,
		})
		return result, err
	}

	// Save resources to database
	if err := s.saveResources(ctx, allResources, result); err != nil {
		s.logger.Error("Resource persistence failed", zap.Error(err))
		result.Errors = append(result.Errors, SyncError{
			Timestamp: time.Now(),
			Service:   "database",
			Operation: "save_resources",
			Error:     err.Error(),
			Retryable: false,
		})
		return result, err
	}

	// Collect statistics if enabled
	if s.config.EnableStatistics {
		stats, err := s.collectStatistics(ctx)
		if err != nil {
			s.logger.Warn("Failed to collect statistics", zap.Error(err))
		} else {
			result.Statistics = stats
		}
	}

	// Update sync state
	s.lastSyncTime = time.Now()
	s.syncCount++
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.TotalResources = len(allResources)
	result.SuccessCount = result.TotalResources - result.FailureCount

	s.logger.Info("Resource sync completed",
		zap.Duration("duration", result.Duration),
		zap.Int("total_resources", result.TotalResources),
		zap.Int("success_count", result.SuccessCount),
		zap.Int("failure_count", result.FailureCount),
		zap.Int("error_count", len(result.Errors)),
	)

	return result, nil
}

// discoverResources discovers resources from all enabled services
func (s *ResourceSyncService) discoverResources(ctx context.Context, result *SyncResult) ([]*models.Resource, error) {
	s.logger.Info("Starting resource discovery")

	var allResources []*models.Resource

	// Discover EC2 resources
	if s.config.EnableEC2 {
		ec2Result, err := s.syncEC2Resources(ctx)
		result.EC2Result = ec2Result
		if err != nil {
			s.logger.Error("EC2 discovery failed", zap.Error(err))
			result.Errors = append(result.Errors, SyncError{
				Timestamp: time.Now(),
				Service:   "EC2",
				Operation: "discovery",
				Error:     err.Error(),
				Retryable: true,
			})
		} else {
			allResources = append(allResources, ec2Result.Resources...)
			s.logger.Info("EC2 discovery completed",
				zap.Int("resources_found", ec2Result.ResourcesFound),
				zap.Duration("duration", ec2Result.Duration),
			)
		}
	}

	// Discover RDS resources
	if s.config.EnableRDS {
		rdsResult, err := s.syncRDSResources(ctx)
		result.RDSResult = rdsResult
		if err != nil {
			s.logger.Error("RDS discovery failed", zap.Error(err))
			result.Errors = append(result.Errors, SyncError{
				Timestamp: time.Now(),
				Service:   "RDS",
				Operation: "discovery",
				Error:     err.Error(),
				Retryable: true,
			})
		} else {
			allResources = append(allResources, rdsResult.Resources...)
			s.logger.Info("RDS discovery completed",
				zap.Int("resources_found", rdsResult.ResourcesFound),
				zap.Duration("duration", rdsResult.Duration),
			)
		}
	}

	// Discover EBS resources
	if s.config.EnableEBS {
		ebsResult, err := s.syncEBSResources(ctx)
		result.EBSResult = ebsResult
		if err != nil {
			s.logger.Error("EBS discovery failed", zap.Error(err))
			result.Errors = append(result.Errors, SyncError{
				Timestamp: time.Now(),
				Service:   "EBS",
				Operation: "discovery",
				Error:     err.Error(),
				Retryable: true,
			})
		} else {
			allResources = append(allResources, ebsResult.Resources...)
			s.logger.Info("EBS discovery completed",
				zap.Int("resources_found", ebsResult.ResourcesFound),
				zap.Duration("duration", ebsResult.Duration),
			)
		}
	}

	s.logger.Info("Resource discovery completed",
		zap.Int("total_resources", len(allResources)),
		zap.Int("ec2_count", len(result.EC2Result.Resources)),
		zap.Int("rds_count", len(result.RDSResult.Resources)),
		zap.Int("ebs_count", len(result.EBSResult.Resources)),
	)

	return allResources, nil
}

// syncEC2Resources syncs EC2 resources
func (s *ResourceSyncService) syncEC2Resources(ctx context.Context) (*ServiceSyncResult, error) {
	result := &ServiceSyncResult{
		ServiceName: "EC2",
		StartTime:   time.Now(),
		Success:     false,
	}

	// Create timeout context
	serviceCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Discover EC2 instances with retry logic
	var instances []*models.EC2Resource
	var err error
	
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Info("Retrying EC2 discovery",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", s.config.MaxRetries),
			)
			
			select {
			case <-serviceCtx.Done():
				return result, serviceCtx.Err()
			case <-time.After(s.config.RetryDelay):
			}
		}

		instances, err = s.ec2Discovery.DiscoverAllInstances(serviceCtx)
		if err == nil {
			break
		}
		
		result.Retries = attempt + 1
		s.logger.Warn("EC2 discovery attempt failed",
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Error = err
		return result, fmt.Errorf("EC2 discovery failed after %d attempts: %w", result.Retries, err)
	}

	// Convert to base Resource model
	result.ResourcesFound = len(instances)
	result.Resources = make([]*models.Resource, len(instances))
	for i, instance := range instances {
		result.Resources[i] = instance.ToResource()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true

	return result, nil
}

// syncRDSResources syncs RDS resources
func (s *ResourceSyncService) syncRDSResources(ctx context.Context) (*ServiceSyncResult, error) {
	result := &ServiceSyncResult{
		ServiceName: "RDS",
		StartTime:   time.Now(),
		Success:     false,
	}

	// Create timeout context
	serviceCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Discover RDS instances with retry logic
	var instances []*models.RDSResource
	var err error
	
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Info("Retrying RDS discovery",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", s.config.MaxRetries),
			)
			
			select {
			case <-serviceCtx.Done():
				return result, serviceCtx.Err()
			case <-time.After(s.config.RetryDelay):
			}
		}

		instances, err = s.rdsDiscovery.DiscoverAllDBInstances(serviceCtx)
		if err == nil {
			break
		}
		
		result.Retries = attempt + 1
		s.logger.Warn("RDS discovery attempt failed",
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Error = err
		return result, fmt.Errorf("RDS discovery failed after %d attempts: %w", result.Retries, err)
	}

	// Convert to base Resource model
	result.ResourcesFound = len(instances)
	result.Resources = make([]*models.Resource, len(instances))
	for i, instance := range instances {
		result.Resources[i] = instance.ToResource()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true

	return result, nil
}

// syncEBSResources syncs EBS resources
func (s *ResourceSyncService) syncEBSResources(ctx context.Context) (*ServiceSyncResult, error) {
	result := &ServiceSyncResult{
		ServiceName: "EBS",
		StartTime:   time.Now(),
		Success:     false,
	}

	// Create timeout context
	serviceCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Discover EBS volumes with retry logic
	var volumes []*models.EBSResource
	var err error
	
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Info("Retrying EBS discovery",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", s.config.MaxRetries),
			)
			
			select {
			case <-serviceCtx.Done():
				return result, serviceCtx.Err()
			case <-time.After(s.config.RetryDelay):
			}
		}

		volumes, err = s.ebsDiscovery.DiscoverAllVolumes(serviceCtx)
		if err == nil {
			break
		}
		
		result.Retries = attempt + 1
		s.logger.Warn("EBS discovery attempt failed",
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Error = err
		return result, fmt.Errorf("EBS discovery failed after %d attempts: %w", result.Retries, err)
	}

	// Convert to base Resource model
	result.ResourcesFound = len(volumes)
	result.Resources = make([]*models.Resource, len(volumes))
	for i, volume := range volumes {
		result.Resources[i] = volume.ToResource()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true

	return result, nil
}

// saveResources saves resources to the database in batches
func (s *ResourceSyncService) saveResources(ctx context.Context, resources []*models.Resource, result *SyncResult) error {
	if len(resources) == 0 {
		s.logger.Info("No resources to save")
		return nil
	}

	s.logger.Info("Saving resources to database",
		zap.Int("total_resources", len(resources)),
		zap.Int("batch_size", s.config.BatchSize),
	)

	dbResult := &DatabaseSyncResult{
		StartTime: time.Now(),
	}

	// Process resources in batches
	for i := 0; i < len(resources); i += s.config.BatchSize {
		end := i + s.config.BatchSize
		if end > len(resources) {
			end = len(resources)
		}

		batch := resources[i:end]
		s.logger.Debug("Processing batch",
			zap.Int("batch_number", i/s.config.BatchSize+1),
			zap.Int("batch_size", len(batch)),
			zap.Int("total_batches", (len(resources)+s.config.BatchSize-1)/s.config.BatchSize),
		)

		// Save batch to database
		err := s.repository.SaveResources(ctx, batch)
		if err != nil {
			s.logger.Error("Failed to save batch",
				zap.Int("batch_number", i/s.config.BatchSize+1),
				zap.Error(err),
			)
			dbResult.FailureCount += len(batch)
			result.FailureCount += len(batch)
			
			// Add error for each resource in failed batch
			for _, resource := range batch {
				result.Errors = append(result.Errors, SyncError{
					Timestamp: time.Now(),
					Service:   "database",
					Operation: "save_batch",
					Error:     err.Error(),
					ResourceID: resource.ResourceID,
					Retryable: false,
				})
			}
		} else {
			dbResult.SuccessCount += len(batch)
			result.SuccessCount += len(batch)
		}

		dbResult.BatchesProcessed++
	}

	dbResult.EndTime = time.Now()
	dbResult.Duration = dbResult.EndTime.Sub(dbResult.StartTime)
	dbResult.TotalOperations = len(resources)
	result.DatabaseOps = dbResult

	s.logger.Info("Resource persistence completed",
		zap.Duration("duration", dbResult.Duration),
		zap.Int("total_operations", dbResult.TotalOperations),
		zap.Int("success_count", dbResult.SuccessCount),
		zap.Int("failure_count", dbResult.FailureCount),
		zap.Int("batches_processed", dbResult.BatchesProcessed),
	)

	return nil
}

// collectStatistics collects resource statistics
func (s *ResourceSyncService) collectStatistics(ctx context.Context) (*repositories.ResourceStatistics, error) {
	s.logger.Debug("Collecting resource statistics")

	stats, err := s.repository.GetResourceStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect statistics: %w", err)
	}

	s.logger.Debug("Statistics collected",
		zap.Int64("total_resources", stats.TotalCount),
		zap.Int64("type_count", stats.TypeCount),
		zap.Int64("provider_count", stats.ProviderCount),
	)

	return stats, nil
}

// GetStatus returns the current status of the sync service
func (s *ResourceSyncService) GetStatus() *SyncStatus {
	return &SyncStatus{
		IsRunning:    s.isRunning,
		LastSyncTime: s.lastSyncTime,
		SyncCount:    s.syncCount,
		Config:       s.config,
		RecentErrors: s.getRecentErrors(10),
	}
}

// SyncStatus represents the current status of the sync service
type SyncStatus struct {
	IsRunning    bool         `json:"is_running"`
	LastSyncTime time.Time    `json:"last_sync_time"`
	SyncCount    int64        `json:"sync_count"`
	Config       *SyncConfig  `json:"config"`
	RecentErrors []SyncError  `json:"recent_errors"`
}

// getRecentErrors returns the most recent errors
func (s *ResourceSyncService) getRecentErrors(count int) []SyncError {
	if len(s.syncErrors) == 0 {
		return []SyncError{}
	}

	if count > len(s.syncErrors) {
		count = len(s.syncErrors)
	}

	// Return the most recent errors (last N)
	start := len(s.syncErrors) - count
	return s.syncErrors[start:]
}

// DefaultSyncConfig returns a default configuration for the sync service
func DefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		Interval:         5 * time.Minute,
		Timeout:          2 * time.Minute,
		MaxRetries:       3,
		RetryDelay:       10 * time.Second,
		EnableEC2:        true,
		EnableRDS:        true,
		EnableEBS:        true,
		EnableStatistics: true,
		BatchSize:        100,
	}
}
