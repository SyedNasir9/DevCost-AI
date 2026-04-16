package aws

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"go.uber.org/zap/zaptest"

	"devcost-ai/internal/config"
	"devcost-ai/pkg/logger"
)

// ExampleUsage demonstrates how to use the AWS client
func ExampleUsage() {
	// Create logger for the example
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	// Create AWS configuration
	awsConfig := &config.AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "your-access-key-id",
		SecretAccessKey: "your-secret-access-key",
		SessionToken:    "", // Optional for temporary credentials
	}

	// Initialize AWS client
	client, err := NewClient(awsConfig, logger)
	if err != nil {
		log.Fatalf("Failed to create AWS client: %v", err)
	}
	defer client.Close()

	// Example 1: Collect all resources
	fmt.Println("=== Collecting AWS Resources ===")
	collector := NewResourceCollector(client)
	resources, err := collector.CollectResources(context.Background())
	if err != nil {
		log.Printf("Failed to collect resources: %v", err)
	} else {
		fmt.Printf("Found %d resources:\n", len(resources))
		for _, resource := range resources {
			fmt.Printf("- %s (%s): %s [%s]\n", resource.Name, resource.Type, resource.ID, resource.State)
		}
	}

	// Example 2: Get EC2 instances with metrics
	fmt.Println("\n=== EC2 Instances and Metrics ===")
	ec2Service := NewEC2Service(client)
	instances, err := ec2Service.ListInstances(context.Background())
	if err != nil {
		log.Printf("Failed to list EC2 instances: %v", err)
	} else {
		for _, instance := range instances {
			instanceID := aws.ToString(instance.InstanceId)
			fmt.Printf("Instance: %s (%s)\n", instanceID, string(instance.State.Name))

			// Get CPU metrics for the last hour
			endTime := time.Now()
			startTime := endTime.Add(-1 * time.Hour)
			
			metrics, err := ec2Service.GetInstanceMetrics(context.Background(), instanceID, startTime, endTime)
			if err != nil {
				log.Printf("Failed to get metrics for %s: %v", instanceID, err)
			} else if len(metrics) > 0 {
				latestMetric := metrics[0] // CloudWatch returns in reverse chronological order
				avgCPU := aws.ToFloat64(latestMetric.Average)
				fmt.Printf("  Average CPU (last hour): %.2f%%\n", avgCPU)
			}
		}
	}

	// Example 3: Get RDS instances and their metrics
	fmt.Println("\n=== RDS Instances and Metrics ===")
	rdsService := NewRDSService(client)
	rdsInstances, err := rdsService.ListDBInstances(context.Background())
	if err != nil {
		log.Printf("Failed to list RDS instances: %v", err)
	} else {
		for _, instance := range rdsInstances {
			instanceID := aws.ToString(instance.DBInstanceIdentifier)
			fmt.Printf("RDS Instance: %s (%s)\n", instanceID, string(instance.DBInstanceStatus))

			// Get metrics for the last hour
			endTime := time.Now()
			startTime := endTime.Add(-1 * time.Hour)
			
			metrics, err := rdsService.GetDBInstanceMetrics(context.Background(), instanceID, startTime, endTime)
			if err != nil {
				log.Printf("Failed to get RDS metrics for %s: %v", instanceID, err)
			} else {
				for metricName, datapoints := range metrics {
					if len(datapoints) > 0 {
						latestMetric := datapoints[0]
						value := aws.ToFloat64(latestMetric.Average)
						fmt.Printf("  %s: %.2f\n", metricName, value)
					}
				}
			}
		}
	}

	// Example 4: Get cost analysis
	fmt.Println("\n=== Cost Analysis ===")
	costService := NewCostService(client)
	
	// Get costs for the last month
	endTime := time.Now()
	startTime := endTime.AddDate(0, -1, 0) // 1 month ago
	
	serviceCosts, err := costService.GetServiceCosts(context.Background(), startTime, endTime)
	if err != nil {
		log.Printf("Failed to get service costs: %v", err)
	} else {
		fmt.Printf("Service costs for the last month:\n")
		totalCost := 0.0
		for service, cost := range serviceCosts {
			fmt.Printf("  %s: $%.2f\n", service, cost)
			totalCost += cost
		}
		fmt.Printf("Total: $%.2f\n", totalCost)
	}

	// Example 5: Detailed cost and usage data
	fmt.Println("\n=== Detailed Cost and Usage ===")
	costData, err := costService.GetCostAndUsage(
		context.Background(),
		startTime,
		endTime,
		types.GranularityMonthly,
	)
	if err != nil {
		log.Printf("Failed to get detailed cost data: %v", err)
	} else {
		for _, timePeriod := range costData.ResultsByTime {
			period := aws.ToString(timePeriod.TimePeriod.Start)
			fmt.Printf("Period: %s\n", period)
			
			for _, group := range timePeriod.Groups {
				var service, region string
				for _, dimension := range group.Keys {
					key := aws.ToString(dimension.Key)
					value := aws.ToString(dimension.Value)
					if key == "SERVICE" {
						service = value
					} else if key == "REGION" {
						region = value
					}
				}
				
				if blendedCost, ok := group.Metrics["BlendedCost"]; ok && blendedCost.Amount != nil {
					cost := aws.ToFloat64(blendedCost.Amount)
					fmt.Printf("  %s (%s): $%.2f\n", service, region, cost)
				}
			}
		}
	}

	// Example 6: Health check
	fmt.Println("\n=== AWS Health Check ===")
	err = client.Health(context.Background())
	if err != nil {
		log.Printf("AWS health check failed: %v", err)
	} else {
		fmt.Println("All AWS services are healthy")
	}
}

// ExampleIntegration demonstrates integration with the main application
func ExampleIntegration() {
	// This would typically be called from your main application setup
	fmt.Println("=== AWS Client Integration Example ===")

	// Load configuration (in real app, this would come from environment/config file)
	cfg := &config.AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "your-access-key-id",
		SecretAccessKey: "your-secret-access-key",
	}

	// Create logger
	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	// Initialize AWS client
	awsClient, err := NewClient(cfg, logger)
	if err != nil {
		log.Printf("Failed to initialize AWS client: %v", err)
		return
	}
	defer awsClient.Close()

	// Create service instances
	ec2Service := NewEC2Service(awsClient)
	rdsService := NewRDSService(awsClient)
	costService := NewCostService(awsClient)
	resourceCollector := NewResourceCollector(awsClient)

	// Example: Periodic resource collection
	fmt.Println("Starting periodic resource collection...")
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	ctx := context.Background()
	for range ticker.C {
		resources, err := resourceCollector.CollectResources(ctx)
		if err != nil {
			log.Printf("Failed to collect resources: %v", err)
			continue
		}

		// Process resources (e.g., store in database, generate recommendations)
		for _, resource := range resources {
			log.Printf("Resource: %s %s (%s)", resource.Type, resource.Name, resource.State)
			
			// Get metrics for running instances
			if resource.Type == "EC2" && resource.State == "running" {
				endTime := time.Now()
				startTime := endTime.Add(-1 * time.Hour)
				
				metrics, err := ec2Service.GetInstanceMetrics(ctx, resource.ID, startTime, endTime)
				if err != nil {
					log.Printf("Failed to get metrics for %s: %v", resource.ID, err)
					continue
				}
				
				if len(metrics) > 0 {
					avgCPU := aws.ToFloat64(metrics[0].Average)
					log.Printf("  CPU Utilization: %.2f%%", avgCPU)
					
					// Generate cost optimization recommendations
					if avgCPU < 10 {
						log.Printf("  Recommendation: Consider stopping or right-sizing instance %s (low utilization)", resource.ID)
					}
				}
			}
		}
	}
}

// ExampleErrorHandling demonstrates proper error handling patterns
func ExampleErrorHandling() {
	fmt.Println("=== AWS Error Handling Examples ===")

	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	cfg := &config.AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "invalid-key",
		SecretAccessKey: "invalid-secret",
	}

	// Example 1: Invalid credentials
	client, err := NewClient(cfg, logger)
	if err != nil {
		log.Printf("Expected error with invalid credentials: %v", err)
	}

	// Example 2: Valid client but invalid operations
	if client != nil {
		ec2Service := NewEC2Service(client)
		
		// Try to get metrics for non-existent instance
		endTime := time.Now()
		startTime := endTime.Add(-1 * time.Hour)
		
		_, err = ec2Service.GetInstanceMetrics(context.Background(), "i-nonexistent", startTime, endTime)
		if err != nil {
			log.Printf("Expected error for non-existent instance: %v", err)
		}
	}
}

// ExampleConfiguration shows different ways to configure the AWS client
func ExampleConfiguration() {
	fmt.Println("=== AWS Configuration Examples ===")

	zapLogger := zaptest.NewLogger(nil)
	logger := logger.NewLogger(zapLogger)

	// Example 1: Environment-based configuration
	fmt.Println("1. Environment-based configuration:")
	envClient, err := NewClientFromEnvironment(logger)
	if err != nil {
		log.Printf("Environment client error: %v", err)
	} else {
		fmt.Printf("  Region: %s\n", envClient.GetRegion())
		creds := envClient.GetCredentials()
		fmt.Printf("  Access Key: %s\n", creds.AccessKeyID)
		envClient.Close()
	}

	// Example 2: Explicit configuration
	fmt.Println("2. Explicit configuration:")
	explicitConfig := &config.AWSConfig{
		Region:          "us-west-2",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	explicitClient, err := NewClient(explicitConfig, logger)
	if err != nil {
		log.Printf("Explicit client error: %v", err)
	} else {
		fmt.Printf("  Region: %s\n", explicitClient.GetRegion())
		explicitClient.Close()
	}

	// Example 3: IAM role-based configuration (no explicit credentials)
	fmt.Println("3. IAM role-based configuration:")
	iamConfig := &config.AWSConfig{
		Region: "us-east-1",
		// No AccessKeyID or SecretAccessKey - will use IAM role
	}

	iamClient, err := NewClient(iamConfig, logger)
	if err != nil {
		log.Printf("IAM role client error: %v", err)
	} else {
		fmt.Printf("  Region: %s\n", iamClient.GetRegion())
		fmt.Printf("  Using IAM role or instance profile\n")
		iamClient.Close()
	}
}
