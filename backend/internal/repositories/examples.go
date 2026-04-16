package repositories

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"

	"devcost-ai/internal/models"
	"devcost-ai/pkg/logger"
)

// ExampleBasicUsage demonstrates basic repository operations
func ExampleBasicUsage() {
	fmt.Println("=== Resource Repository Basic Usage Example ===")

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	// Create repository (in real usage, this would use a real database pool)
	// repo := NewResourceRepository(dbPool, logger)

	fmt.Println("Repository operations:")
	fmt.Println("1. SaveResources - Bulk upsert of resources")
	fmt.Println("2. SaveResource - Single resource upsert")
	fmt.Println("3. GetResourceByResourceID - Retrieve by cloud ID")
	fmt.Println("4. GetResourcesByType - Filter by resource type")
	fmt.Println("5. GetResourcesByProvider - Filter by provider")
	fmt.Println("6. GetResourcesByFilter - Complex filtering")
	fmt.Println("7. DeleteResourceByResourceID - Delete resource")
	fmt.Println("8. GetResourceCount - Count resources")
	fmt.Println("9. GetResourceStatistics - Resource analytics")

	// Example of creating resources
	ec2Resource := models.NewResource("i-1234567890abcdef0", string(models.ResourceTypeEC2), "aws", "us-east-1", "123456789012")
	ec2Resource.Name = "web-server-01"
	ec2Resource.State = models.ResourceStateRunning
	ec2Resource.InstanceType = "t3.micro"
	ec2Resource.Tags = map[string]string{
		"Environment": "production",
		"Owner":       "team-a",
		"Project":     "devcost-ai",
	}
	ec2Resource.Metadata = map[string]interface{}{
		"availability_zone": "us-east-1a",
		"public_ip":        "203.0.113.12",
		"private_ip":       "10.0.1.123",
	}

	ebsResource := models.NewResource("vol-1234567890abcdef0", string(models.ResourceTypeEBS), "aws", "us-east-1", "123456789012")
	ebsResource.Name = "data-volume-01"
	ebsResource.State = models.ResourceStateAvailable
	ebsResource.Tags = map[string]string{
		"Environment": "production",
		"Backup":       "daily",
	}
	ebsResource.Metadata = map[string]interface{}{
		"size_gb":     100,
		"volume_type": "gp3",
		"encrypted":   true,
		"attached":     false,
	}

	rdsResource := models.NewResource("test-db-instance", string(models.ResourceTypeRDS), "aws", "us-east-1", "123456789012")
	rdsResource.Name = "production-db"
	rdsResource.State = models.ResourceStateAvailable
	rdsResource.InstanceType = "db.t3.micro"
	rdsResource.Tags = map[string]string{
		"Environment": "production",
		"Backup":       "daily",
		"Retention":    "30-days",
	}
	rdsResource.Metadata = map[string]interface{}{
		"engine":         "mysql",
		"engine_version":  "8.0.35",
		"allocated_storage": 20,
		"multi_az":        false,
		"encrypted":       true,
	}

	resources := []*models.Resource{ec2Resource, ebsResource, rdsResource}

	fmt.Printf("\nCreated %d resources:\n", len(resources))
	for _, resource := range resources {
		fmt.Printf("- %s (%s): %s [%s]\n",
			resource.Name,
			resource.ResourceType,
			resource.ResourceID,
			resource.State,
		)
	}

	// In real usage, you would save them like this:
	// err := repo.SaveResources(ctx, resources)
	// if err != nil {
	//     log.Printf("Failed to save resources: %v", err)
	//     return
	// }
	//
	// fmt.Println("Resources saved successfully!")
}

// ExampleUpsertLogic demonstrates the upsert behavior
func ExampleUpsertLogic() {
	fmt.Println("=== Upsert Logic Example ===")

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	fmt.Println("Upsert Logic with ON CONFLICT:")
	fmt.Println("1. INSERT if resource_id doesn't exist")
	fmt.Println("2. UPDATE if resource_id exists")
	fmt.Println("3. Preserves created_at timestamp")
	fmt.Println("4. Updates all other fields including updated_at")

	// Example scenario:
	fmt.Println("\nScenario 1: New Resource (INSERT)")
	fmt.Println("- Resource ID: i-1234567890abcdef0")
	fmt.Println("- Action: INSERT new record")
	fmt.Println("- Result: New resource created with current timestamp")

	fmt.Println("\nScenario 2: Existing Resource (UPDATE)")
	fmt.Println("- Resource ID: i-1234567890abcdef0")
	fmt.Println("- Action: UPDATE existing record")
	fmt.Println("- Result: All fields updated, created_at preserved")

	// SQL upsert query:
	sqlQuery := `
	INSERT INTO resources (
		id, resource_id, resource_type, provider, region, account_id,
		name, state, instance_type, tags, metadata, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	ON CONFLICT (resource_id) 
	DO UPDATE SET
		resource_type = EXCLUDED.resource_type,
		provider = EXCLUDED.provider,
		region = EXCLUDED.region,
		account_id = EXCLUDED.account_id,
		name = EXCLUDED.name,
		state = EXCLUDED.state,
		instance_type = EXCLUDED.instance_type,
		tags = EXCLUDED.tags,
		metadata = EXCLUDED.metadata,
		updated_at = EXCLUDED.updated_at
	`

	fmt.Printf("\nSQL Query:\n%s\n", sqlQuery)

	// Benefits of this approach:
	fmt.Println("\nBenefits of ON CONFLICT upsert:")
	fmt.Println("✓ Atomic operation - no race conditions")
	fmt.Println("✓ Single round trip to database")
	fmt.Println("✓ Handles both insert and update cases")
	fmt.Println("✓ Preserves creation timestamp")
	fmt.Println("✓ Efficient for bulk operations")
	fmt.Println("✓ No duplicate records")
}

// ExampleFiltering demonstrates various filtering options
func ExampleFiltering() {
	fmt.Println("=== Resource Filtering Examples ===")

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	fmt.Println("Filtering Options:")

	fmt.Println("\n1. Filter by Resource Type")
	filter := &ResourceFilter{
		ResourceTypes: []models.ResourceType{
			models.ResourceTypeEC2,
			models.ResourceTypeRDS,
		},
	}
	fmt.Printf("Filter: %+v\n", filter)
	fmt.Println("Result: EC2 and RDS resources only")

	fmt.Println("\n2. Filter by State")
	filter = &ResourceFilter{
		States: []models.ResourceState{
			models.ResourceStateRunning,
			models.ResourceStateAvailable,
		},
	}
	fmt.Printf("Filter: %+v\n", filter)
	fmt.Println("Result: Only running/available resources")

	fmt.Println("\n3. Filter by Region")
	filter = &ResourceFilter{
		Regions: []string{"us-east-1", "us-west-2"},
	}
	fmt.Printf("Filter: %+v\n", filter)
	fmt.Println("Result: Resources in specific regions")

	fmt.Println("\n4. Filter by Tags")
	filter = &ResourceFilter{
		Tags: map[string]string{
			"Environment": "production",
			"Owner":       "team-a",
		},
	}
	fmt.Printf("Filter: %+v\n", filter)
	fmt.Println("Result: Production resources owned by team-a")

	fmt.Println("\n5. Complex Filter (Multiple Criteria)")
	filter = &ResourceFilter{
		ResourceTypes: []models.ResourceType{models.ResourceTypeEC2},
		States:        []models.ResourceState{models.ResourceStateRunning},
		Regions:       []string{"us-east-1"},
		Tags: map[string]string{
			"Environment": "production",
		},
		Limit:  100,
		Offset: 0,
	}
	fmt.Printf("Filter: %+v\n", filter)
	fmt.Println("Result: Running EC2 instances in us-east-1, production tag, limited to 100")

	// SQL generation example:
	fmt.Println("\nGenerated SQL Example:")
	sqlQuery := `
	SELECT id, resource_id, resource_type, provider, region, account_id,
		   name, state, instance_type, tags, metadata, created_at, updated_at
	FROM resources
	WHERE 1=1
	  AND resource_type IN ($1, $2)
	  AND state IN ($3, $4)
	  AND region IN ($5, $6)
	  AND tags->>'Environment' = $7
	  AND tags->>'Owner' = $8
	ORDER BY created_at DESC
	LIMIT $9 OFFSET $10
	`
	fmt.Printf("%s\n", sqlQuery)

	// JSONB tag filtering benefits:
	fmt.Println("\nJSONB Tag Filtering Benefits:")
	fmt.Println("✓ Fast indexing with GIN indexes")
	fmt.Println("✓ Flexible key-value queries")
	fmt.Println("✓ No need for separate tag tables")
	fmt.Println("✓ Complex tag queries supported")
	fmt.Println("✓ Efficient storage for sparse data")
}

// ExampleBulkOperations demonstrates bulk operations
func ExampleBulkOperations() {
	fmt.Println("=== Bulk Operations Example ===")

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	fmt.Println("Bulk Operations Benefits:")
	fmt.Println("✓ Single transaction for all operations")
	fmt.Println("✓ Reduced network round trips")
	fmt.Println("✓ Atomicity - all succeed or all fail")
	fmt.Println("✓ Performance optimization")
	fmt.Println("✓ Error handling per resource")

	// Example of bulk save with mixed success/failure
	fmt.Println("\nBulk Save Scenario:")
	resources := createBulkTestResources(1000) // 1000 resources

	fmt.Printf("Attempting to save %d resources...\n", len(resources))

	// In real usage:
	start := time.Now()
	// err := repo.SaveResources(ctx, resources)
	duration := time.Since(start)

	fmt.Printf("Bulk save completed in %v\n", duration)
	fmt.Printf("Success rate: %d/%d (%.1f%%)\n", 
		950, len(resources), 950.0/float64(len(resources))*100)

	fmt.Println("\nError Handling:")
	fmt.Println("✓ Individual resource errors don't stop bulk operation")
	fmt.Println("✓ Detailed error logging per resource")
	fmt.Println("✓ Continue processing other resources")
	fmt.Println("✓ Summary of success/failure rates")

	// Transaction benefits:
	fmt.Println("\nTransaction Benefits:")
	fmt.Println("✓ ACID compliance")
	fmt.Println("✓ Rollback on failure")
	fmt.Println("✓ Consistent database state")
	fmt.Println("✓ Isolation from other operations")
}

// ExampleJSONBUsage demonstrates JSONB tag storage
func ExampleJSONBUsage() {
	fmt.Println("=== JSONB Tag Storage Example ===")

	// Example tags as JSONB
	tags := map[string]string{
		"Environment": "production",
		"Owner":       "team-a",
		"Project":     "devcost-ai",
		"CostCenter":  "engineering",
		"Backup":      "daily",
		"Monitoring":  "enabled",
	}

	fmt.Println("Tags stored as JSONB:")
	for key, value := range tags {
		fmt.Printf("  %s: %s\n", key, value)
	}

	// JSON representation
	jsonTags := `{
		"Environment": "production",
		"Owner": "team-a",
		"Project": "devcost-ai",
		"CostCenter": "engineering",
		"Backup": "daily",
		"Monitoring": "enabled"
	}`

	fmt.Printf("\nJSONB Storage:\n%s\n", jsonTags)

	// Query examples:
	fmt.Println("\nJSONB Query Examples:")
	fmt.Println("1. Exact tag match:")
	fmt.Println("   WHERE tags->>'Environment' = 'production'")

	fmt.Println("2. Tag exists:")
	fmt.Println("   WHERE tags ? 'Environment'")

	fmt.Println("3. Multiple tag conditions:")
	fmt.Println("   WHERE tags->>'Environment' = 'production' AND tags->>'Owner' = 'team-a'")

	fmt.Println("4. Tag key contains:")
	fmt.Println("   WHERE tags::text LIKE '%Environment%'")

	// Indexing strategy:
	fmt.Println("\nIndexing Strategy:")
	fmt.Println("✓ GIN index on tags column for fast queries")
	fmt.Println("✓ Partial indexes for common tag queries")
	fmt.Println("✓ Expression indexes for specific tag keys")

	// Performance benefits:
	fmt.Println("\nPerformance Benefits:")
	fmt.Println("✓ Faster than EAV (Entity-Attribute-Value) model")
	fmt.Println("✓ No JOIN operations needed for tag queries")
	fmt.Println("✓ Efficient storage for sparse tag data")
	fmt.Println("✓ Flexible schema - no migrations needed for new tags")
}

// ExampleStatistics demonstrates statistics and analytics
func ExampleStatistics() {
	fmt.Println("=== Resource Statistics Example ===")

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	fmt.Println("Resource Analytics:")

	// Example statistics result
	stats := &ResourceStatistics{
		TotalCount:    1250,
		TypeCount:     3,
		ProviderCount: 2,
		RegionCount:   4,
		AccountCount:  3,
		ByType: map[string]int64{
			"EC2": 800,
			"EBS": 350,
			"RDS": 100,
		},
	}

	fmt.Printf("Resource Statistics:\n")
	fmt.Printf("  Total Resources: %d\n", stats.TotalCount)
	fmt.Printf("  Resource Types: %d\n", stats.TypeCount)
	fmt.Printf("  Providers: %d\n", stats.ProviderCount)
	fmt.Printf("  Regions: %d\n", stats.RegionCount)
	fmt.Printf("  Accounts: %d\n", stats.AccountCount)

	fmt.Printf("\nResources by Type:\n")
	for resourceType, count := range stats.ByType {
		fmt.Printf("  %s: %d (%.1f%%)\n", 
			resourceType, count, float64(count)/float64(stats.TotalCount)*100)
	}

	// Additional analytics examples:
	fmt.Println("\nAdvanced Analytics:")
	fmt.Println("✓ Resource utilization by type")
	fmt.Println("✓ Cost breakdown by region")
	fmt.Println("✓ Tag distribution analysis")
	fmt.Println("✓ Resource lifecycle tracking")
	fmt.Println("✓ Multi-account aggregation")

	// SQL for statistics:
	fmt.Println("\nStatistics SQL Queries:")
	fmt.Println("1. Total count:")
	fmt.Println("   SELECT COUNT(*) FROM resources")

	fmt.Println("2. Count by type:")
	fmt.Println("   SELECT resource_type, COUNT(*) FROM resources GROUP BY resource_type")

	fmt.Println("3. Complex statistics:")
	fmt.Println("   SELECT provider, region, resource_type, COUNT(*)")
	fmt.Println("   FROM resources GROUP BY provider, region, resource_type")
}

// ExampleIntegration shows how to integrate with the discovery services
func ExampleIntegration() {
	fmt.Println("=== Repository Integration Example ===")

	fmt.Println("Integration Workflow:")
	fmt.Println("1. Discover resources from AWS services")
	fmt.Println("2. Convert to internal resource models")
	fmt.Println("3. Bulk save to database using repository")
	fmt.Println("4. Generate statistics and analytics")
	fmt.Println("5. Provide API endpoints for resource data")

	// Example integration code:
	fmt.Println("\nIntegration Code Example:")
	fmt.Println(`
// 1. Discover resources
collector := services.NewUnifiedResourceCollector(awsClient)
result, err := collector.CollectAllResources(ctx)
if err != nil {
    log.Printf("Discovery failed: %v", err)
    return
}

// 2. Save to database
repo := repositories.NewResourceRepository(dbPool, logger)
err = repo.SaveResources(ctx, result.GetAllResources())
if err != nil {
    log.Printf("Failed to save resources: %v", err)
    return
}

// 3. Generate statistics
stats, err := repo.GetResourceStatistics(ctx)
if err != nil {
    log.Printf("Failed to get statistics: %v", err)
    return
}

log.Printf("Saved %d resources", result.TotalCount)
log.Printf("Resource breakdown: %+v", stats.ByType)
`)

	fmt.Println("\nBenefits of This Integration:")
	fmt.Println("✓ Decoupled discovery and persistence")
	fmt.Println("✓ Repository abstraction for easy testing")
	fmt.Println("✓ Bulk operations for performance")
	fmt.Println("✓ Comprehensive error handling")
	fmt.Println("✓ Statistics and analytics built-in")
	fmt.Println("✓ Flexible filtering and querying")
}

// Helper functions for examples

func createBulkTestResources(count int) []*models.Resource {
	resources := make([]*models.Resource, count)
	
	for i := 0; i < count; i++ {
		resourceID := fmt.Sprintf("resource-%d", i)
		var resourceType models.ResourceType
		
		switch i % 3 {
		case 0:
			resourceType = models.ResourceTypeEC2
		case 1:
			resourceType = models.ResourceTypeEBS
		case 2:
			resourceType = models.ResourceTypeRDS
		}
		
		resource := models.NewResource(resourceID, string(resourceType), "aws", "us-east-1", "123456789012")
		resource.Name = fmt.Sprintf("test-resource-%d", i)
		resource.State = models.ResourceStateRunning
		resource.Tags = map[string]string{
			"Environment": "test",
			"Index":       fmt.Sprintf("%d", i),
		}
		resource.Metadata = map[string]interface{}{
			"created_by": "bulk_test",
			"index":      i,
		}
		
		resources[i] = resource
	}
	
	return resources
}
