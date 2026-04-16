package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/internal/repositories"
	"devcost-ai/pkg/logger"
)

// ExampleResourceSync demonstrates the resource sync service usage
func ExampleResourceSync() {
	fmt.Println("=== Resource Sync Service Example ===")

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	// Create AWS client (in real usage, this would be properly configured)
	awsClient := &aws.Client{
		Region: "us-east-1",
		logger: logger,
	}

	// Create database repository (in real usage, this would use a real database pool)
	repo := &repositories.ResourceRepository{
		logger: logger,
	}

	// Create sync configuration
	config := &SyncConfig{
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

	// Create sync service
	syncService := NewResourceSyncService(awsClient, repo, logger, config)

	fmt.Println("Sync Service Configuration:")
	fmt.Printf("  Interval: %v\n", config.Interval)
	fmt.Printf("  Timeout: %v\n", config.Timeout)
	fmt.Printf("  Max Retries: %d\n", config.MaxRetries)
	fmt.Printf("  Batch Size: %d\n", config.BatchSize)
	fmt.Printf("  Enable EC2: %t\n", config.EnableEC2)
	fmt.Printf("  Enable RDS: %t\n", config.EnableRDS)
	fmt.Printf("  Enable EBS: %t\n", config.EnableEBS)

	// Example: Start the sync service
	fmt.Println("\nStarting sync service...")
	ctx := context.Background()
	err := syncService.Start(ctx)
	if err != nil {
		log.Printf("Failed to start sync service: %v", err)
		return
	}

	// Let it run for a bit
	time.Sleep(10 * time.Second)

	// Check status
	status := syncService.GetStatus()
	fmt.Printf("Sync Status:\n")
	fmt.Printf("  Is Running: %t\n", status.IsRunning)
	fmt.Printf("  Sync Count: %d\n", status.SyncCount)
	fmt.Printf("  Last Sync: %v\n", status.LastSyncTime)

	// Stop the sync service
	fmt.Println("\nStopping sync service...")
	err = syncService.Stop()
	if err != nil {
		log.Printf("Failed to stop sync service: %v", err)
	}

	fmt.Println("Sync service example completed")
}

// ExampleManualSync demonstrates manual sync execution
func ExampleManualSync() {
	fmt.Println("=== Manual Sync Example ===")

	// Create dependencies
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)
	
	awsClient := &aws.Client{
		Region: "us-east-1",
		logger: logger,
	}
	
	repo := &repositories.ResourceRepository{
		logger: logger,
	}
	
	config := DefaultSyncConfig()
	syncService := NewResourceSyncService(awsClient, repo, logger, config)

	// Perform manual sync
	fmt.Println("Performing manual sync...")
	ctx := context.Background()
	result, err := syncService.SyncNow(ctx)
	if err != nil {
		log.Printf("Manual sync failed: %v", err)
		return
	}

	// Display results
	fmt.Printf("Sync Results:\n")
	fmt.Printf("  Duration: %v\n", result.Duration)
	fmt.Printf("  Total Resources: %d\n", result.TotalResources)
	fmt.Printf("  Success Count: %d\n", result.SuccessCount)
	fmt.Printf("  Failure Count: %d\n", result.FailureCount)
	fmt.Printf("  Error Count: %d\n", len(result.Errors))

	// Display service-specific results
	if result.EC2Result != nil {
		fmt.Printf("  EC2: %d resources found, %d saved (%v)\n",
			result.EC2Result.ResourcesFound,
			result.EC2Result.ResourcesSaved,
			result.EC2Result.Duration)
	}

	if result.RDSResult != nil {
		fmt.Printf("  RDS: %d resources found, %d saved (%v)\n",
			result.RDSResult.ResourcesFound,
			result.RDSResult.ResourcesSaved,
			result.RDSResult.Duration)
	}

	if result.EBSResult != nil {
		fmt.Printf("  EBS: %d resources found, %d saved (%v)\n",
			result.EBSResult.ResourcesFound,
			result.EBSResult.ResourcesSaved,
			result.EBSResult.Duration)
	}

	// Display database operations
	if result.DatabaseOps != nil {
		fmt.Printf("  Database: %d operations, %d success, %d failures (%v)\n",
			result.DatabaseOps.TotalOperations,
			result.DatabaseOps.SuccessCount,
			result.DatabaseOps.FailureCount,
			result.DatabaseOps.Duration)
		fmt.Printf("  Batches Processed: %d\n", result.DatabaseOps.BatchesProcessed)
	}

	// Display errors if any
	if len(result.Errors) > 0 {
		fmt.Printf("  Errors:\n")
		for i, err := range result.Errors {
			fmt.Printf("    %d. %s: %s - %s\n", i+1, err.Service, err.Operation, err.Error)
		}
	}

	// Display statistics if available
	if result.Statistics != nil {
		fmt.Printf("  Statistics:\n")
		fmt.Printf("    Total Resources: %d\n", result.Statistics.TotalCount)
		fmt.Printf("    Resource Types: %d\n", result.Statistics.TypeCount)
		fmt.Printf("    Providers: %d\n", result.Statistics.ProviderCount)
		fmt.Printf("    Regions: %d\n", result.Statistics.RegionCount)
		fmt.Printf("    Accounts: %d\n", result.Statistics.AccountCount)
	}
}

// ExampleSyncConfiguration demonstrates different sync configurations
func ExampleSyncConfiguration() {
	fmt.Println("=== Sync Configuration Examples ===")

	// High-frequency configuration (for development/testing)
	highFreqConfig := &SyncConfig{
		Interval:         1 * time.Minute,
		Timeout:          30 * time.Second,
		MaxRetries:       1,
		RetryDelay:       5 * time.Second,
		EnableEC2:        true,
		EnableRDS:        false,
		EnableEBS:        false,
		EnableStatistics: false,
		BatchSize:        50,
	}

	// Production configuration
	productionConfig := &SyncConfig{
		Interval:         15 * time.Minute,
		Timeout:          5 * time.Minute,
		MaxRetries:       5,
		RetryDelay:       30 * time.Second,
		EnableEC2:        true,
		EnableRDS:        true,
		EnableEBS:        true,
		EnableStatistics: true,
		BatchSize:        200,
	}

	// Cost optimization configuration (focus on running resources)
	costOptConfig := &SyncConfig{
		Interval:         10 * time.Minute,
		Timeout:          3 * time.Minute,
		MaxRetries:       3,
		RetryDelay:       15 * time.Second,
		EnableEC2:        true,
		EnableRDS:        true,
		EnableEBS:        true,
		EnableStatistics: true,
		BatchSize:        150,
	}

	configs := []struct {
		name   string
		config *SyncConfig
	}{
		{"High Frequency (Development)", highFreqConfig},
		{"Production", productionConfig},
		{"Cost Optimization", costOptConfig},
	}

	for _, cfg := range configs {
		fmt.Printf("\n%s Configuration:\n", cfg.name)
		fmt.Printf("  Interval: %v\n", cfg.config.Interval)
		fmt.Printf("  Timeout: %v\n", cfg.config.Timeout)
		fmt.Printf("  Max Retries: %d\n", cfg.config.MaxRetries)
		fmt.Printf("  Retry Delay: %v\n", cfg.config.RetryDelay)
		fmt.Printf("  Batch Size: %d\n", cfg.config.BatchSize)
		fmt.Printf("  Services: EC2=%t, RDS=%t, EBS=%t\n",
			cfg.config.EnableEC2, cfg.config.EnableRDS, cfg.config.EnableEBS)
		fmt.Printf("  Statistics: %t\n", cfg.config.EnableStatistics)
	}
}

// ExampleErrorHandling demonstrates error handling scenarios
func ExampleErrorHandling() {
	fmt.Println("=== Error Handling Examples ===")

	fmt.Println("Error Handling Features:")
	fmt.Println("✓ Per-service error isolation")
	fmt.Println("✓ Retry logic with exponential backoff")
	fmt.Println("✓ Detailed error reporting")
	fmt.Println("✓ Graceful degradation")
	fmt.Println("✓ Error categorization (retryable vs non-retryable)")

	// Example error scenarios
	errorScenarios := []struct {
		service   string
		operation string
		error     string
		retryable bool
	}{
		{"EC2", "discovery", "rate limit exceeded", true},
		{"RDS", "discovery", "connection timeout", true},
		{"EBS", "discovery", "authorization failed", false},
		{"database", "save_batch", "connection lost", true},
		{"database", "save_batch", "constraint violation", false},
	}

	fmt.Println("\nError Scenarios:")
	for i, scenario := range errorScenarios {
		fmt.Printf("  %d. %s %s: %s (retryable: %t)\n",
			i+1, scenario.service, scenario.operation, scenario.error, scenario.retryable)
	}

	// Example error handling flow
	fmt.Println("\nError Handling Flow:")
	fmt.Println("1. Service discovery fails with retryable error")
	fmt.Println("2. Retry with exponential backoff")
	fmt.Println("3. If all retries fail, log error and continue")
	fmt.Println("4. Continue with other services")
	fmt.Println("5. Save successful resources to database")
	fmt.Println("6. Report comprehensive results with errors")
	fmt.Println("7. Continue scheduled sync despite partial failures")
}

// ExampleBatchProcessing demonstrates batch processing logic
func ExampleBatchProcessing() {
	fmt.Println("=== Batch Processing Example ===")

	// Create test resources
	resources := createMockResources(250)
	batchSize := 100

	fmt.Printf("Processing %d resources with batch size %d\n", len(resources), batchSize)

	// Calculate batches
	batches := calculateMockBatches(resources, batchSize)
	fmt.Printf("Will process %d batches\n", len(batches))

	// Simulate batch processing
	var totalSuccess, totalFailures int
	for i, batch := range batches {
		fmt.Printf("Processing batch %d/%d (%d resources)\n", i+1, len(batches), len(batch))
		
		// Simulate some failures
		successCount := len(batch)
		failureCount := 0
		
		// Simulate 5% failure rate
		for j, resource := range batch {
			if j%20 == 0 { // Every 20th resource fails
				successCount--
				failureCount++
				fmt.Printf("  Failed to save resource %s\n", resource.ResourceID)
			}
		}
		
		totalSuccess += successCount
		totalFailures += failureCount
		
		fmt.Printf("  Batch %d: %d success, %d failures\n", i+1, successCount, failureCount)
	}

	fmt.Printf("Batch Processing Summary:\n")
	fmt.Printf("  Total Resources: %d\n", len(resources))
	fmt.Printf("  Total Success: %d\n", totalSuccess)
	fmt.Printf("  Total Failures: %d\n", totalFailures)
	fmt.Printf("  Success Rate: %.1f%%\n", float64(totalSuccess)/float64(len(resources))*100)
}

// ExampleMonitoring demonstrates monitoring and observability
func ExampleMonitoring() {
	fmt.Println("=== Monitoring and Observability Example ===")

	fmt.Println("Monitoring Features:")
	fmt.Println("✓ Structured logging with Zap")
	fmt.Println("✓ Performance metrics per service")
	fmt.Println("✓ Error tracking and categorization")
	fmt.Println("✓ Sync statistics and trends")
	fmt.Println("✓ Resource count tracking")
	fmt.Println("✓ Database operation metrics")

	// Example monitoring output
	fmt.Println("\nSample Monitoring Output:")
	fmt.Println(`{"level":"info","timestamp":"2024-03-26T10:00:00Z","msg":"Starting resource sync operation"}`)
	fmt.Println(`{"level":"info","timestamp":"2024-03-26T10:00:05Z","msg":"EC2 discovery completed","resources_found":45,"duration":1.2s}`)
	fmt.Println(`{"level":"info","timestamp":"2024-03-26T10:00:07Z","msg":"RDS discovery completed","resources_found":8,"duration":2.1s}`)
	fmt.Println(`{"level":"info","timestamp":"2024-03-26T10:00:09Z","msg":"EBS discovery completed","resources_found":120,"duration":1.8s}`)
	fmt.Println(`{"level":"info","timestamp":"2024-03-26T10:00:15Z","msg":"Resource persistence completed","duration":6.2s,"total_operations":173,"success_count":170,"failure_count":3}`)
	fmt.Println(`{"level":"info","timestamp":"2024-03-26T10:00:15Z","msg":"Resource sync completed","duration":15.3s","total_resources":173,"success_count":170,"failure_count":3,"error_count":3}`)

	// Example metrics
	fmt.Println("\nKey Metrics:")
	fmt.Println("  Sync Frequency: Every 5 minutes")
	fmt.Println("  Average Duration: 15.3 seconds")
	fmt.Println("  Resources per Sync: 173")
	fmt.Println("  Success Rate: 98.3%")
	fmt.Println("  Error Rate: 1.7%")
	fmt.Println("  Database Operations: 173")
	fmt.Println("  Batch Processing: 2 batches of 100 resources")

	// Example alerts
	fmt.Println("\nAlert Conditions:")
	fmt.Println("  High Error Rate (>5%): Trigger alert")
	fmt.Println("  Long Duration (>5 minutes): Trigger alert")
	fmt.Println("  Service Unavailable: Trigger alert")
	fmt.Println("  Database Connection Issues: Trigger alert")
	fmt.Println("  Resource Count Anomaly (>50% change): Trigger alert")
}

// ExampleIntegration demonstrates integration with the main application
func ExampleIntegration() {
	fmt.Println("=== Application Integration Example ===")

	fmt.Println("Integration Pattern:")
	fmt.Println("1. Initialize sync service in main application")
	fmt.Println("2. Start sync service in background goroutine")
	fmt.Println("3. Expose status endpoints for monitoring")
	fmt.Println("4. Handle graceful shutdown")
	fmt.Println("5. Configure based on environment")

	// Example integration code
	fmt.Println("\nIntegration Code:")
	fmt.Println(`
// main.go
func main() {
    // Initialize dependencies
    logger := logger.NewLogger(zap.NewProduction())
    awsClient := aws.NewClient(config.AWS, logger)
    dbPool := database.NewPool(config.Database, logger)
    repo := repositories.NewResourceRepository(dbPool, logger)
    
    // Create sync configuration
    syncConfig := &services.SyncConfig{
        Interval:         time.Duration(config.Sync.Interval) * time.Minute,
        Timeout:          time.Duration(config.Sync.Timeout) * time.Minute,
        MaxRetries:       config.Sync.MaxRetries,
        EnableEC2:        config.Sync.EnableEC2,
        EnableRDS:        config.Sync.EnableRDS,
        EnableEBS:        config.Sync.EnableEBS,
        BatchSize:        config.Sync.BatchSize,
    }
    
    // Create and start sync service
    syncService := services.NewResourceSyncService(awsClient, repo, logger, syncConfig)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    if err := syncService.Start(ctx); err != nil {
        log.Fatalf("Failed to start sync service: %v", err)
    }
    
    // Setup HTTP endpoints
    router := gin.New()
    router.GET("/sync/status", func(c *gin.Context) {
        status := syncService.GetStatus()
        c.JSON(200, status)
    })
    
    router.POST("/sync/now", func(c *gin.Context) {
        result, err := syncService.SyncNow(c.Request.Context())
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, result)
    })
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    <-sigChan
    log.Println("Shutting down...")
    
    if err := syncService.Stop(); err != nil {
        log.Printf("Error stopping sync service: %v", err)
    }
    
    log.Println("Shutdown complete")
}
`)

	fmt.Println("\nHTTP API Endpoints:")
	fmt.Println("  GET  /sync/status - Get sync service status")
	fmt.Println("  POST /sync/now    - Trigger immediate sync")
	fmt.Println("  GET  /sync/history - Get sync history")
	fmt.Println("  GET  /sync/metrics - Get sync metrics")

	fmt.Println("\nConfiguration:")
	fmt.Println("  SYNC_INTERVAL=5m")
	fmt.Println("  SYNC_TIMEOUT=2m")
	fmt.Println("  SYNC_MAX_RETRIES=3")
	fmt.Println("  SYNC_BATCH_SIZE=100")
	fmt.Println("  SYNC_ENABLE_EC2=true")
	fmt.Println("  SYNC_ENABLE_RDS=true")
	fmt.Println("  SYNC_ENABLE_EBS=true")
}

// Helper functions for examples

func createMockResources(count int) []*models.Resource {
	resources := make([]*models.Resource, count)
	
	for i := 0; i < count; i++ {
		resource := models.NewResource(
			fmt.Sprintf("resource-%d", i),
			"EC2",
			"aws",
			"us-east-1",
			"123456789012",
		)
		resource.Name = fmt.Sprintf("test-resource-%d", i)
		resource.State = models.ResourceStateRunning
		resource.Tags = map[string]string{
			"Environment": "test",
			"Index":       fmt.Sprintf("%d", i),
		}
		resource.Metadata = map[string]interface{}{
			"created_by": "test_suite",
			"index":      i,
		}
		
		resources[i] = resource
	}
	
	return resources
}

func calculateMockBatches(resources []*models.Resource, batchSize int) [][]*models.Resource {
	var batches [][]*models.Resource
	
	for i := 0; i < len(resources); i += batchSize {
		end := i + batchSize
		if end > len(resources) {
			end = len(resources)
		}
		
		batch := resources[i:end]
		batches = append(batches, batch)
	}
	
	return batches
}
