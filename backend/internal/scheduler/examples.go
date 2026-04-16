package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"

	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// ExampleBackgroundScheduler demonstrates the background scheduler usage
func ExampleBackgroundScheduler() {
	fmt.Println("=== Background Scheduler Example ===")

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	// Create mock sync service (in real usage, this would be a real service)
	syncService := &services.ResourceSyncService{}

	// Create scheduler configuration
	config := DefaultSchedulerConfig()
	config.Interval = 10 * time.Second // 10 minutes as required
	config.EnableLogging = true
	config.LogLevel = "info"

	fmt.Printf("Scheduler Configuration:\n")
	fmt.Printf("  Interval: %v\n", config.Interval)
	fmt.Printf("  Timeout: %v\n", config.Timeout)
	fmt.Printf("  Max Retries: %d\n", config.MaxRetries)
	fmt.Printf("  Retry Delay: %v\n", config.RetryDelay)
	fmt.Printf("  Enable Logging: %t\n", config.EnableLogging)
	fmt.Printf("  Log Level: %s\n", config.LogLevel)

	// Create scheduler
	scheduler := NewBackgroundScheduler(syncService, logger, config)

	// Example 1: Basic scheduler lifecycle
	fmt.Println("\n1. Basic Scheduler Lifecycle:")
	exampleSchedulerLifecycle(scheduler)

	// Example 2: Immediate trigger
	fmt.Println("\n2. Immediate Trigger:")
	exampleImmediateTrigger(scheduler)

	// Example 3: Status monitoring
	fmt.Println("\n3. Status Monitoring:")
	exampleStatusMonitoring(scheduler)

	// Example 4: Different configurations
	fmt.Println("\n4. Different Configurations:")
	exampleConfigurations()

	// Example 5: Error handling and retries
	fmt.Println("\n5. Error Handling and Retries:")
	exampleErrorHandling()

	// Example 6: Integration with main application
	fmt.Println("\n6. Application Integration:")
	exampleApplicationIntegration()
}

// exampleSchedulerLifecycle demonstrates basic start/stop lifecycle
func exampleSchedulerLifecycle(scheduler *BackgroundScheduler) {
	ctx := context.Background()

	fmt.Printf("Starting scheduler...\n")
	
	// Start scheduler
	err := scheduler.Start(ctx)
	if err != nil {
		log.Printf("Failed to start scheduler: %v\n", err)
		return
	}

	fmt.Printf("Scheduler started successfully\n")
	fmt.Printf("Is Running: %t\n", scheduler.IsRunning())

	// Let it run for a bit
	time.Sleep(2 * time.Second)

	fmt.Printf("Stopping scheduler...\n")
	
	// Stop scheduler
	err = scheduler.Stop(ctx)
	if err != nil {
		log.Printf("Failed to stop scheduler: %v\n", err)
		return
	}

	fmt.Printf("Scheduler stopped successfully\n")
	fmt.Printf("Is Running: %t\n", scheduler.IsRunning())
}

// exampleImmediateTrigger demonstrates triggering immediate runs
func exampleImmediateTrigger(scheduler *BackgroundScheduler) {
	ctx := context.Background()

	fmt.Printf("Triggering immediate run...\n")
	
	// Trigger immediate run
	err := scheduler.TriggerNow(ctx)
	if err != nil {
		log.Printf("Failed to trigger immediate run: %v\n", err)
		return
	}

	fmt.Printf("Immediate run triggered successfully\n")

	// Wait for completion
	time.Sleep(1 * time.Second)

	// Check status
	status := scheduler.GetStatus()
	fmt.Printf("Run Count: %d\n", status.RunCount)
	fmt.Printf("Last Run Time: %v\n", status.LastRunTime)
	if status.LastRunDuration > 0 {
		fmt.Printf("Last Run Duration: %v\n", status.LastRunDuration)
	}
}

// exampleStatusMonitoring demonstrates status monitoring
func exampleStatusMonitoring(scheduler *BackgroundScheduler) {
	// Get current status
	status := scheduler.GetStatus()

	fmt.Printf("Scheduler Status:\n")
	fmt.Printf("  Is Running: %t\n", status.IsRunning)
	fmt.Printf("  Run Count: %d\n", status.RunCount)
	fmt.Printf("  Last Run Time: %v\n", status.LastRunTime)
	fmt.Printf("  Last Run Duration: %v\n", status.LastRunDuration)
	fmt.Printf("  Next Run Time: %v\n", status.NextRunTime)

	if status.LastRunResult != nil {
		fmt.Printf("  Last Run Result:\n")
		fmt.Printf("    Total Resources: %d\n", status.LastRunResult.TotalResources)
		fmt.Printf("    Success Count: %d\n", status.LastRunResult.SuccessCount)
		fmt.Printf("    Failure Count: %d\n", status.LastRunResult.FailureCount)
		fmt.Printf("    Error Count: %d\n", len(status.LastRunResult.Errors))
	}

	if status.LastRunError != "" {
		fmt.Printf("  Last Run Error: %s\n", status.LastRunError)
	}

	// Check if currently running
	fmt.Printf("  Currently Running: %t\n", scheduler.IsCurrentlyRunning())

	// Get next run time
	nextRun := scheduler.GetNextRunTime()
	if !nextRun.IsZero() {
		fmt.Printf("  Time Until Next Run: %v\n", time.Until(nextRun))
	}
}

// exampleConfigurations demonstrates different configuration options
func exampleConfigurations() {
	fmt.Printf("Configuration Examples:\n")

	// Default configuration
	defaultConfig := DefaultSchedulerConfig()
	fmt.Printf("\n1. Default Configuration:\n")
	fmt.Printf("  Interval: %v\n", defaultConfig.Interval)
	fmt.Printf("  Timeout: %v\n", defaultConfig.Timeout)
	fmt.Printf("  Max Retries: %d\n", defaultConfig.MaxRetries)
	fmt.Printf("  Use Case: Production with moderate frequency\n")

	// High-frequency configuration
	highFreqConfig := HighFrequencyConfig()
	fmt.Printf("\n2. High-Frequency Configuration:\n")
	fmt.Printf("  Interval: %v\n", highFreqConfig.Interval)
	fmt.Printf("  Timeout: %v\n", highFreqConfig.Timeout)
	fmt.Printf("  Max Retries: %d\n", highFreqConfig.MaxRetries)
	fmt.Printf("  Log Level: %s\n", highFreqConfig.LogLevel)
	fmt.Printf("  Use Case: Development and testing\n")

	// Production configuration
	prodConfig := ProductionConfig()
	fmt.Printf("\n3. Production Configuration:\n")
	fmt.Printf("  Interval: %v\n", prodConfig.Interval)
	fmt.Printf("  Timeout: %v\n", prodConfig.Timeout)
	fmt.Printf("  Max Retries: %d\n", prodConfig.MaxRetries)
	fmt.Printf("  Graceful Shutdown Timeout: %v\n", prodConfig.GracefulShutdownTimeout)
	fmt.Printf("  Use Case: Production environment with reliability\n")

	// Custom configuration
	customConfig := &SchedulerConfig{
		Interval:                 30 * time.Minute,
		Timeout:                  10 * time.Minute,
		MaxRetries:               5,
		RetryDelay:               60 * time.Second,
		EnableLogging:            true,
		LogLevel:                 "warn",
		GracefulShutdownTimeout:  120 * time.Second,
	}
	fmt.Printf("\n4. Custom Configuration:\n")
	fmt.Printf("  Interval: %v\n", customConfig.Interval)
	fmt.Printf("  Timeout: %v\n", customConfig.Timeout)
	fmt.Printf("  Max Retries: %d\n", customConfig.MaxRetries)
	fmt.Printf("  Use Case: Low-frequency with high reliability\n")
}

// exampleErrorHandling demonstrates error handling and retry logic
func exampleErrorHandling() {
	fmt.Printf("Error Handling Features:\n")
	fmt.Printf("✓ Automatic retry on failures\n")
	fmt.Printf("✓ Exponential backoff between retries\n")
	fmt.Printf("✓ Detailed error logging\n")
	fmt.Printf("✓ Graceful degradation\n")
	fmt.Printf("✓ Timeout protection\n")
	fmt.Printf("✓ Context cancellation support\n")

	fmt.Printf("\nError Scenarios:\n")
	fmt.Printf("1. Network Timeout: Retry with exponential backoff\n")
	fmt.Printf("2. Database Connection Lost: Retry up to max retries\n")
	fmt.Printf("3. Service Unavailable: Retry with delay\n")
	fmt.Printf("4. Authorization Error: No retry (non-retryable)\n")
	fmt.Printf("5. Resource Not Found: No retry (non-retryable)\n")

	fmt.Printf("\nRetry Logic Flow:\n")
	fmt.Printf("1. Execute sync with timeout\n")
	fmt.Printf("2. If failed and retryable, wait retry delay\n")
	fmt.Printf("3. Retry up to max retries\n")
	fmt.Printf("4. Log all attempts and results\n")
	fmt.Printf("5. Continue scheduler even if all retries fail\n")
}

// exampleApplicationIntegration demonstrates integration with main application
func exampleApplicationIntegration() {
	fmt.Printf("Application Integration:\n")
	fmt.Printf(`
// main.go
func main() {
    // Initialize dependencies
    logger := logger.NewLogger(zap.NewProduction())
    awsClient := aws.NewClient(config.AWS, logger)
    repo := repositories.NewResourceRepository(dbPool, logger)
    syncService := services.NewResourceSyncService(awsClient, repo, logger, syncConfig)
    
    // Create scheduler configuration
    schedulerConfig := &scheduler.SchedulerConfig{
        Interval:                 10 * time.Minute,
        Timeout:                  5 * time.Minute,
        MaxRetries:               3,
        RetryDelay:               30 * time.Second,
        EnableLogging:            true,
        LogLevel:                 "info",
        GracefulShutdownTimeout:  60 * time.Second,
    }
    
    // Create and start scheduler
    scheduler := scheduler.NewBackgroundScheduler(syncService, logger, schedulerConfig)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    if err := scheduler.Start(ctx); err != nil {
        log.Fatalf("Failed to start scheduler: %v", err)
    }
    
    // Setup HTTP endpoints for scheduler control
    router := gin.New()
    router.GET("/scheduler/status", func(c *gin.Context) {
        status := scheduler.GetStatus()
        c.JSON(200, status)
    })
    
    router.POST("/scheduler/trigger", func(c *gin.Context) {
        err := scheduler.TriggerNow(c.Request.Context())
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, gin.H{"message": "Triggered successfully"})
    })
    
    router.POST("/scheduler/stop", func(c *gin.Context) {
        err := scheduler.Stop(c.Request.Context())
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, gin.H{"message": "Stopped successfully"})
    })
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    <-sigChan
    log.Println("Shutting down...")
    
    // Stop scheduler first
    if err := scheduler.Stop(ctx); err != nil {
        log.Printf("Error stopping scheduler: %v", err)
    }
    
    // Stop other services...
    
    log.Println("Shutdown complete")
}
`)

	fmt.Printf("\nHTTP API Endpoints:\n")
	fmt.Printf("  GET  /scheduler/status - Get scheduler status\n")
	fmt.Printf("  POST /scheduler/trigger - Trigger immediate run\n")
	fmt.Printf("  POST /scheduler/stop - Stop scheduler\n")

	fmt.Printf("\nConfiguration via Environment Variables:\n")
	fmt.Printf("  SCHEDULER_INTERVAL=10m\n")
	fmt.Printf("  SCHEDULER_TIMEOUT=5m\n")
	fmt.Printf("  SCHEDULER_MAX_RETRIES=3\n")
	fmt.Printf("  SCHEDULER_RETRY_DELAY=30s\n")
	fmt.Printf("  SCHEDULER_LOG_LEVEL=info\n")
	fmt.Printf("  SCHEDULER_ENABLE_LOGGING=true\n")
}

// ExampleLoggingOutput demonstrates the logging output format
func ExampleLoggingOutput() {
	fmt.Printf("Logging Output Examples:\n")
	fmt.Printf("\n1. Scheduler Start:\n")
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:00:00Z","msg":"Starting background scheduler","interval":600000000000,"timeout":300000000000,"max_retries":2,"enable_logging":true,"log_level":"info"}`)

	fmt.Printf("\n2. Scheduled Sync Start:\n")
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:10:00Z","msg":"Starting scheduled resource sync","start_time":"2024-03-26T10:10:00Z","run_count":1}`)

	fmt.Printf("\n3. Service Results:\n")
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:10:05Z","msg":"EC2 service results","success":true,"resources_found":45,"duration":1200000000,"retries":0}`)
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:10:07Z","msg":"RDS service results","success":true,"resources_found":8,"duration":2100000000,"retries":0}`)
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:10:09Z","msg":"EBS service results","success":true,"resources_found":120,"duration":1800000000,"retries":0}`)

	fmt.Printf("\n4. Database Operations:\n")
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:10:15Z","msg":"Database operations results","success":true,"total_operations":173,"success_count":170,"failure_count":3,"batches_processed":2,"duration":6200000000}`)

	fmt.Printf("\n5. Sync Completion:\n")
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:10:15Z","msg":"Scheduled sync completed successfully","duration":15300000000,"total_resources":173,"success_count":170,"failure_count":3,"error_count":3}`)

	fmt.Printf("\n6. Error and Retry:\n")
	fmt.Printf(`{"level":"warn","timestamp":"2024-03-26T10:20:00Z","msg":"Scheduled sync failed","error":"database connection lost","duration":5000000000}`)
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:20:30Z","msg":"Retrying scheduled sync","attempt":1,"max_retries":3,"retry_delay":30000000000}`)
	fmt.Printf(`{"level":"error","timestamp":"2024-03-26T10:20:35Z","msg":"Scheduled sync retry failed","attempt":1,"error":"database connection lost","duration":2000000000}`)
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:20:40Z","msg":"Retrying scheduled sync","attempt":2,"max_retries":3,"retry_delay":30000000000}`)
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T10:20:45Z","msg":"Scheduled sync retry succeeded","attempt":2,"duration":15000000000,"total_resources":150,"success_count":150,"failure_count":0}`)

	fmt.Printf("\n7. Scheduler Shutdown:\n")
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T11:00:00Z","msg":"Stopping background scheduler"}`)
	fmt.Printf(`{"level":"info","timestamp":"2024-03-26T11:00:01Z","msg":"Background scheduler stopped gracefully"}`)
}

// ExampleSchedulerMetrics demonstrates monitoring and metrics
func ExampleSchedulerMetrics() {
	fmt.Printf("Scheduler Metrics and Monitoring:\n")
	fmt.Printf("\nKey Metrics:\n")
	fmt.Printf("✓ Run Count: Total number of sync runs\n")
	fmt.Printf("✓ Success Rate: Percentage of successful runs\n")
	fmt.Printf("✓ Average Duration: Average time per run\n")
	fmt.Printf("✓ Resource Count: Average resources per run\n")
	fmt.Printf("✓ Error Rate: Percentage of failed runs\n")
	fmt.Printf("✓ Retry Rate: Percentage of runs requiring retries\n")

	fmt.Printf("\nMonitoring Examples:\n")
	fmt.Printf("1. Health Check: GET /scheduler/status\n")
	fmt.Printf("2. Metrics Endpoint: GET /scheduler/metrics\n")
	fmt.Printf("3. Alerting: Based on error rate and duration\n")
	fmt.Printf("4. Dashboard: Real-time scheduler status\n")

	fmt.Printf("\nAlert Conditions:\n")
	fmt.Printf("• High Error Rate (>10%%): Trigger alert\n")
	fmt.Printf("• Long Duration (>5 minutes): Trigger alert\n")
	fmt.Printf("• Continuous Failures (>3 consecutive): Trigger alert\n")
	fmt.Printf("• Scheduler Not Running: Trigger alert\n")
	fmt.Printf("• Resource Count Anomaly (>50%% change): Trigger alert\n")
}

// ExamplePerformanceOptimization demonstrates performance considerations
func ExamplePerformanceOptimization() {
	fmt.Printf("Performance Optimization:\n")
	fmt.Printf("\n1. Interval Tuning:\n")
	fmt.Printf("   • High Frequency (1-5 min): Development/testing\n")
	fmt.Printf("   • Standard (10-15 min): Production default\n")
	fmt.Printf("   • Low Frequency (30-60 min): Large deployments\n")
	fmt.Printf("   • Very Low (2-4 hours): Cost optimization\n")

	fmt.Printf("\n2. Timeout Configuration:\n")
	fmt.Printf("   • Short (1-2 min): Fast environments\n")
	fmt.Printf("   • Standard (5 min): Balanced approach\n")
	fmt.Printf("   • Long (10+ min): Large deployments\n")

	fmt.Printf("\n3. Retry Strategy:\n")
	fmt.Printf("   • Conservative (1-2 retries): Stable environments\n")
	fmt.Printf("   • Standard (3 retries): Production default\n")
	fmt.Printf("   • Aggressive (5+ retries): Unstable environments\n")

	fmt.Printf("\n4. Resource Usage:\n")
	fmt.Printf("   • Memory: Minimal (scheduler state only)\n")
	fmt.Printf("   • CPU: Low (ticker-based, event-driven)\n")
	fmt.Printf("   • Network: During sync operations only\n")
	fmt.Printf("   • Database: During persistence only\n")

	fmt.Printf("\n5. Best Practices:\n")
	fmt.Printf("   ✓ Use appropriate intervals for your use case\n")
	fmt.Printf("   ✓ Set reasonable timeouts to prevent hanging\n")
	fmt.Printf("   ✓ Enable detailed logging for debugging\n")
	fmt.Printf("   ✓ Monitor scheduler health and performance\n")
	fmt.Printf("   ✓ Implement graceful shutdown handling\n")
	fmt.Printf("   ✓ Use context cancellation for clean shutdown\n")
}

// ExampleTroubleshooting demonstrates common issues and solutions
func ExampleTroubleshooting() {
	fmt.Printf("Troubleshooting Guide:\n")
	fmt.Printf("\n1. Scheduler Not Starting:\n")
	fmt.Printf("   • Check if already running\n")
	fmt.Printf("   • Verify sync service initialization\n")
	fmt.Printf("   • Check configuration parameters\n")
	fmt.Printf("   • Review logs for initialization errors\n")

	fmt.Printf("\n2. Runs Not Executing:\n")
	fmt.Printf("   • Verify ticker is created\n")
	fmt.Printf("   • Check context cancellation\n")
	fmt.Printf("   • Review stop signal handling\n")
	fmt.Printf("   • Monitor goroutine lifecycle\n")

	fmt.Printf("\n3. High Error Rate:\n")
	fmt.Printf("   • Check network connectivity\n")
	fmt.Printf("   • Verify AWS credentials\n")
	fmt.Printf("   • Review database connection\n")
	fmt.Printf("   • Increase retry delay\n")
	fmt.Printf("   • Check service availability\n")

	fmt.Printf("\n4. Long Duration:\n")
	fmt.Printf("   • Increase timeout value\n")
	fmt.Printf("   • Check resource volume\n")
	fmt.Printf("   • Optimize database queries\n")
	fmt.Printf("   • Review batch size configuration\n")
	fmt.Printf("   • Monitor system resources\n")

	fmt.Printf("\n5. Memory Leaks:\n")
	fmt.Printf("   • Monitor goroutine count\n")
	fmt.Printf("   • Check for circular references\n")
	fmt.Printf("   • Review context usage\n")
	fmt.Printf("   • Profile memory usage\n")
	fmt.Printf("   • Ensure proper cleanup\n")

	fmt.Printf("\nDebug Commands:\n")
	fmt.Printf("• curl http://localhost:8080/scheduler/status\n")
	fmt.Printf("• curl -X POST http://localhost:8080/scheduler/trigger\n")
	fmt.Printf("• Check application logs for scheduler entries\n")
	fmt.Printf("• Monitor system resources (CPU, memory)\n")
	fmt.Printf("• Review database connection pool status\n")
	fmt.Printf("• Verify AWS service availability\n")
}

// Helper function to create mock sync result for examples
func createMockSyncResult() *services.SyncResult {
	return &services.SyncResult{
		TotalResources: 150,
		SuccessCount:   145,
		FailureCount:   5,
		StartTime:     time.Now().Add(-15 * time.Second),
		EndTime:       time.Now(),
		Duration:      15 * time.Second,
		Errors: []services.SyncError{
			{
				Timestamp: time.Now().Add(-2 * time.Second),
				Service:   "EC2",
				Operation: "discovery",
				Error:     "rate limit exceeded",
				Retryable: true,
			},
		},
	}
}
