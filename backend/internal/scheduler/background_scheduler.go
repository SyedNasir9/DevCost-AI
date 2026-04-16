package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// BackgroundScheduler manages scheduled background tasks
type BackgroundScheduler struct {
	logger          *logger.Logger
	syncService     *services.ResourceSyncService
	config          *SchedulerConfig
	
	// Scheduler state
	isRunning       bool
	isRunningMutex  sync.RWMutex
	
	// Task state
	lastRunTime     time.Time
	lastRunDuration time.Duration
	lastRunResult   *services.SyncResult
	lastRunError    error
	runCount        int64
	runCountMutex   sync.RWMutex
	
	// Ticker and control
	ticker          *time.Ticker
	stopChan        chan struct{}
	wg              sync.WaitGroup
	
	// Execution control
	running         bool
	runningMutex    sync.Mutex
}

// SchedulerConfig defines configuration for the background scheduler
type SchedulerConfig struct {
	Interval        time.Duration `json:"interval"`         // Run interval (e.g., 10 * time.Minute)
	Timeout         time.Duration `json:"timeout"`          // Timeout for each run
	MaxRetries      int           `json:"max_retries"`      // Maximum retries per run
	RetryDelay      time.Duration `json:"retry_delay"`      // Delay between retries
	EnableLogging   bool          `json:"enable_logging"`   // Enable detailed logging
	LogLevel        string        `json:"log_level"`        // Log level (debug, info, warn, error)
	GracefulShutdownTimeout time.Duration `json:"graceful_shutdown_timeout"` // Timeout for graceful shutdown
}

// SchedulerStatus represents the current status of the scheduler
type SchedulerStatus struct {
	IsRunning       bool                    `json:"is_running"`
	LastRunTime     time.Time               `json:"last_run_time"`
	LastRunDuration time.Duration           `json:"last_run_duration"`
	LastRunResult   *services.SyncResult    `json:"last_run_result,omitempty"`
	LastRunError    string                  `json:"last_run_error,omitempty"`
	RunCount        int64                    `json:"run_count"`
	Config          *SchedulerConfig         `json:"config"`
	NextRunTime     time.Time               `json:"next_run_time"`
}

// NewBackgroundScheduler creates a new background scheduler
func NewBackgroundScheduler(
	syncService *services.ResourceSyncService,
	logger *logger.Logger,
	config *SchedulerConfig,
) *BackgroundScheduler {
	return &BackgroundScheduler{
		logger:      logger,
		syncService: syncService,
		config:      config,
		stopChan:    make(chan struct{}),
	}
}

// Start starts the background scheduler
func (s *BackgroundScheduler) Start(ctx context.Context) error {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	s.logger.Info("Starting background scheduler",
		zap.Duration("interval", s.config.Interval),
		zap.Duration("timeout", s.config.Timeout),
		zap.Int("max_retries", s.config.MaxRetries),
		zap.Bool("enable_logging", s.config.EnableLogging),
		zap.String("log_level", s.config.LogLevel),
	)

	s.isRunning = true
	s.ticker = time.NewTicker(s.config.Interval)

	// Start the scheduler goroutine
	s.wg.Add(1)
	go s.run(ctx)

	s.logger.Info("Background scheduler started successfully")
	return nil
}

// Stop stops the background scheduler gracefully
func (s *BackgroundScheduler) Stop(ctx context.Context) error {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	s.logger.Info("Stopping background scheduler")

	// Signal stop
	close(s.stopChan)

	// Stop ticker
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Wait for graceful shutdown with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("Background scheduler stopped gracefully")
	case <-time.After(s.config.GracefulShutdownTimeout):
		s.logger.Warn("Background scheduler shutdown timeout")
	}

	s.isRunning = false
	return nil
}

// TriggerNow triggers an immediate run (if not already running)
func (s *BackgroundScheduler) TriggerNow(ctx context.Context) error {
	s.runningMutex.Lock()
	if s.running {
		s.runningMutex.Unlock()
		return fmt.Errorf("scheduler is already running")
	}
	s.running = true
	s.runningMutex.Unlock()

	s.logger.Info("Triggering immediate scheduler run")

	// Run in goroutine to avoid blocking
	go func() {
		defer func() {
			s.runningMutex.Lock()
			s.running = false
			s.runningMutex.Unlock()
		}()

		s.runOnce(ctx)
	}()

	return nil
}

// run is the main scheduler loop
func (s *BackgroundScheduler) run(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler context cancelled, stopping")
			return
		case <-s.stopChan:
			s.logger.Info("Scheduler stop signal received")
			return
		case <-s.ticker.C:
			s.logger.Debug("Scheduler tick received")
			s.runOnce(ctx)
		}
	}
}

// runOnce executes a single sync run
func (s *BackgroundScheduler) runOnce(ctx context.Context) {
	startTime := time.Now()

	s.logger.Info("Starting scheduled resource sync",
		zap.Time("start_time", startTime),
		zap.Int64("run_count", s.getRunCount()+1),
	)

	// Update run count
	s.incrementRunCount()

	// Execute sync with timeout
	syncCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	result, err := s.syncService.SyncNow(syncCtx)

	// Update state
	duration := time.Since(startTime)
	s.updateRunState(startTime, duration, result, err)

	// Log results
	s.logRunResults(result, err, duration)

	// Handle errors
	if err != nil {
		s.logger.Error("Scheduled sync failed",
			zap.Error(err),
			zap.Duration("duration", duration),
		)

		// Retry logic if configured
		if s.config.MaxRetries > 0 {
			s.retryRun(ctx, startTime)
		}
	} else {
		s.logger.Info("Scheduled sync completed successfully",
			zap.Duration("duration", duration),
			zap.Int("total_resources", result.TotalResources),
			zap.Int("success_count", result.SuccessCount),
			zap.Int("failure_count", result.FailureCount),
		)
	}
}

// retryRun implements retry logic for failed runs
func (s *BackgroundScheduler) retryRun(ctx context.Context, originalStartTime time.Time) {
	for attempt := 1; attempt <= s.config.MaxRetries; attempt++ {
		s.logger.Info("Retrying scheduled sync",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", s.config.MaxRetries),
			zap.Duration("retry_delay", s.config.RetryDelay),
		)

		// Wait before retry
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler context cancelled during retry")
			return
		case <-s.stopChan:
			s.logger.Info("Scheduler stop signal received during retry")
			return
		case <-time.After(s.config.RetryDelay):
		}

		retryStartTime := time.Now()
		
		// Execute retry with timeout
		syncCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		result, err := s.syncService.SyncNow(syncCtx)
		duration := time.Since(retryStartTime)

		// Update state with retry result
		s.updateRunState(retryStartTime, duration, result, err)

		if err != nil {
			s.logger.Error("Scheduled sync retry failed",
				zap.Int("attempt", attempt),
				zap.Error(err),
				zap.Duration("duration", duration),
			)
		} else {
			s.logger.Info("Scheduled sync retry succeeded",
				zap.Int("attempt", attempt),
				zap.Duration("duration", duration),
				zap.Int("total_resources", result.TotalResources),
				zap.Int("success_count", result.SuccessCount),
			)
			return // Success, exit retry loop
		}
	}

	s.logger.Error("All retry attempts failed",
		zap.Int("max_retries", s.config.MaxRetries),
		zap.Duration("total_duration", time.Since(originalStartTime)),
	)
}

// logRunResults logs detailed results of a sync run
func (s *BackgroundScheduler) logRunResults(result *services.SyncResult, err error, duration time.Duration) {
	if !s.config.EnableLogging {
		return
	}

	// Basic run information
	s.logger.Info("Sync run completed",
		zap.Duration("duration", duration),
		zap.Bool("success", err == nil),
	)

	if err != nil {
		s.logger.Error("Sync run error details",
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		return
	}

	// Detailed success logging
	if result != nil {
		s.logger.Info("Sync run results",
			zap.Int("total_resources", result.TotalResources),
			zap.Int("success_count", result.SuccessCount),
			zap.Int("failure_count", result.FailureCount),
			zap.Int("error_count", len(result.Errors)),
			zap.Duration("duration", duration),
		)

		// Service-specific results
		if result.EC2Result != nil {
			s.logger.Info("EC2 service results",
				zap.Bool("success", result.EC2Result.Success),
				zap.Int("resources_found", result.EC2Result.ResourcesFound),
				zap.Duration("duration", result.EC2Result.Duration),
				zap.Int("retries", result.EC2Result.Retries),
			)
		}

		if result.RDSResult != nil {
			s.logger.Info("RDS service results",
				zap.Bool("success", result.RDSResult.Success),
				zap.Int("resources_found", result.RDSResult.ResourcesFound),
				zap.Duration("duration", result.RDSResult.Duration),
				zap.Int("retries", result.RDSResult.Retries),
			)
		}

		if result.EBSResult != nil {
			s.logger.Info("EBS service results",
				zap.Bool("success", result.EBSResult.Success),
				zap.Int("resources_found", result.EBSResult.ResourcesFound),
				zap.Duration("duration", result.EBSResult.Duration),
				zap.Int("retries", result.EBSResult.Retries),
			)
		}

		// Database operations
		if result.DatabaseOps != nil {
			s.logger.Info("Database operations results",
				zap.Bool("success", result.DatabaseOps.Success),
				zap.Int("total_operations", result.DatabaseOps.TotalOperations),
				zap.Int("success_count", result.DatabaseOps.SuccessCount),
				zap.Int("failure_count", result.DatabaseOps.FailureCount),
				zap.Int("batches_processed", result.DatabaseOps.BatchesProcessed),
				zap.Duration("duration", result.DatabaseOps.Duration),
			)
		}

		// Statistics
		if result.Statistics != nil {
			s.logger.Info("Resource statistics",
				zap.Int64("total_resources", result.Statistics.TotalCount),
				zap.Int64("type_count", result.Statistics.TypeCount),
				zap.Int64("provider_count", result.Statistics.ProviderCount),
				zap.Int64("region_count", result.Statistics.RegionCount),
				zap.Int64("account_count", result.Statistics.AccountCount),
			)

			// Resource type breakdown
			for resourceType, count := range result.Statistics.ByType {
				s.logger.Debug("Resource type count",
					zap.String("resource_type", resourceType),
					zap.Int64("count", count),
				)
			}
		}

		// Log errors if any
		if len(result.Errors) > 0 {
			s.logger.Warn("Sync run had errors",
				zap.Int("error_count", len(result.Errors)),
			)

			for i, syncError := range result.Errors {
				s.logger.Error("Sync error details",
					zap.Int("error_index", i+1),
					zap.String("service", syncError.Service),
					zap.String("operation", syncError.Operation),
					zap.String("error", syncError.Error),
					zap.String("resource_id", syncError.ResourceID),
					zap.Bool("retryable", syncError.Retryable),
					zap.Time("timestamp", syncError.Timestamp),
				)
			}
		}
	}
}

// updateRunState updates the scheduler state after a run
func (s *BackgroundScheduler) updateRunState(startTime time.Time, duration time.Duration, result *services.SyncResult, err error) {
	s.lastRunTime = startTime
	s.lastRunDuration = duration
	s.lastRunResult = result
	s.lastRunError = err
}

// incrementRunCount safely increments the run counter
func (s *BackgroundScheduler) incrementRunCount() {
	s.runCountMutex.Lock()
	defer s.runCountMutex.Unlock()
	s.runCount++
}

// getRunCount safely returns the current run count
func (s *BackgroundScheduler) getRunCount() int64 {
	s.runCountMutex.RLock()
	defer s.runCountMutex.RUnlock()
	return s.runCount
}

// GetStatus returns the current status of the scheduler
func (s *BackgroundScheduler) GetStatus() *SchedulerStatus {
	s.isRunningMutex.RLock()
	defer s.isRunningMutex.RUnlock()

	s.runCountMutex.RLock()
	defer s.runCountMutex.RUnlock()

	status := &SchedulerStatus{
		IsRunning:       s.isRunning,
		LastRunTime:     s.lastRunTime,
		LastRunDuration: s.lastRunDuration,
		LastRunResult:   s.lastRunResult,
		RunCount:        s.runCount,
		Config:          s.config,
	}

	if s.lastRunError != nil {
		status.LastRunError = s.lastRunError.Error()
	}

	// Calculate next run time
	if s.isRunning && s.ticker != nil {
		status.NextRunTime = s.lastRunTime.Add(s.config.Interval)
	}

	return status
}

// IsRunning returns whether the scheduler is currently running
func (s *BackgroundScheduler) IsRunning() bool {
	s.isRunningMutex.RLock()
	defer s.isRunningMutex.RUnlock()
	return s.isRunning
}

// IsCurrentlyRunning returns whether a sync run is currently in progress
func (s *BackgroundScheduler) IsCurrentlyRunning() bool {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()
	return s.running
}

// GetNextRunTime returns when the next run is scheduled
func (s *BackgroundScheduler) GetNextRunTime() time.Time {
	if !s.IsRunning() {
		return time.Time{}
	}

	s.runCountMutex.RLock()
	defer s.runCountMutex.RUnlock()

	if s.lastRunTime.IsZero() {
		return time.Now().Add(s.config.Interval)
	}

	return s.lastRunTime.Add(s.config.Interval)
}

// GetRunHistory returns a summary of recent runs
func (s *BackgroundScheduler) GetRunHistory(limit int) *RunHistory {
	s.runCountMutex.RLock()
	defer s.runCountMutex.RUnlock()

	history := &RunHistory{
		RunCount:        s.runCount,
		LastRunTime:     s.lastRunTime,
		LastRunDuration: s.lastRunDuration,
		LastRunResult:   s.lastRunResult,
		LastRunError:    s.lastRunError,
		Config:          s.config,
	}

	// In a real implementation, you might store more detailed history
	// For now, we'll just return the most recent run
	return history
}

// RunHistory represents the history of scheduler runs
type RunHistory struct {
	RunCount        int64                   `json:"run_count"`
	LastRunTime     time.Time               `json:"last_run_time"`
	LastRunDuration time.Duration           `json:"last_run_duration"`
	LastRunResult   *services.SyncResult    `json:"last_run_result,omitempty"`
	LastRunError    string                  `json:"last_run_error,omitempty"`
	Config          *SchedulerConfig         `json:"config"`
}

// DefaultSchedulerConfig returns a default configuration for the scheduler
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		Interval:                 10 * time.Minute,
		Timeout:                  5 * time.Minute,
		MaxRetries:               2,
		RetryDelay:               30 * time.Second,
		EnableLogging:            true,
		LogLevel:                 "info",
		GracefulShutdownTimeout:  30 * time.Second,
	}
}

// HighFrequencyConfig returns a configuration for high-frequency scheduling (development/testing)
func HighFrequencyConfig() *SchedulerConfig {
	return &SchedulerConfig{
		Interval:                 1 * time.Minute,
		Timeout:                  2 * time.Minute,
		MaxRetries:               1,
		RetryDelay:               10 * time.Second,
		EnableLogging:            true,
		LogLevel:                 "debug",
		GracefulShutdownTimeout:  10 * time.Second,
	}
}

// ProductionConfig returns a configuration optimized for production
func ProductionConfig() *SchedulerConfig {
	return &SchedulerConfig{
		Interval:                 10 * time.Minute,
		Timeout:                  5 * time.Minute,
		MaxRetries:               3,
		RetryDelay:               30 * time.Second,
		EnableLogging:            true,
		LogLevel:                 "info",
		GracefulShutdownTimeout:  60 * time.Second,
	}
}
