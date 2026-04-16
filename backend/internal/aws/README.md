# DevCost AI AWS Client

This module provides a production-grade AWS client implementation for DevCost AI, using AWS SDK for Go v2 with comprehensive error handling, logging, and modular service support.

## 🚀 Features

- **Multi-Service Support**: EC2, RDS, EKS, CloudWatch, Cost Explorer
- **Environment-Based Configuration**: Support for environment variables and IAM roles
- **Production-Ready Error Handling**: Comprehensive error handling with structured logging
- **Modular Design**: Service-specific clients for clean architecture
- **Health Monitoring**: Built-in health checks for AWS services
- **Resource Collection**: Unified resource discovery across services
- **Cost Analysis**: Integrated cost and usage data retrieval
- **Security**: Credential masking and secure configuration handling

## 📁 Module Structure

```
internal/aws/
├── client.go          # Main AWS client and configuration
├── services.go        # Service-specific implementations
├── examples.go        # Usage examples and integration patterns
├── client_test.go     # Unit tests and benchmarks
└── README.md          # This documentation
```

## 🔧 Configuration

### Environment Variables

```bash
# Required
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key-id
AWS_SECRET_ACCESS_KEY=your-secret-access-key

# Optional
AWS_SESSION_TOKEN=your-session-token  # For temporary credentials
AWS_PROFILE=default                    # AWS profile name
```

### Configuration Structure

```go
type AWSConfig struct {
    Region          string
    AccessKeyID     string
    SecretAccessKey string
    SessionToken    string // Optional
}
```

## 📖 Usage Examples

### Basic Client Initialization

```go
import (
    "devcost-ai/internal/aws"
    "devcost-ai/internal/config"
    "devcost-ai/pkg/logger"
)

// Create configuration
cfg := &config.AWSConfig{
    Region:          "us-east-1",
    AccessKeyID:     "your-access-key-id",
    SecretAccessKey: "your-secret-access-key",
}

// Create logger
logger := logger.NewLogger(zap.NewProduction())

// Initialize AWS client
client, err := aws.NewClient(cfg, logger)
if err != nil {
    log.Fatalf("Failed to create AWS client: %v", err)
}
defer client.Close()
```

### Environment-Based Initialization

```go
// Uses environment variables automatically
client, err := aws.NewClientFromEnvironment(logger)
if err != nil {
    log.Fatalf("Failed to create AWS client: %v", err)
}
```

### Service-Specific Operations

#### EC2 Operations

```go
// Create EC2 service
ec2Service := aws.NewEC2Service(client)

// List all instances
instances, err := ec2Service.ListInstances(context.Background())
if err != nil {
    log.Printf("Failed to list instances: %v", err)
}

// Get instance metrics
endTime := time.Now()
startTime := endTime.Add(-1 * time.Hour)
metrics, err := ec2Service.GetInstanceMetrics(
    context.Background(),
    "i-1234567890abcdef0",
    startTime,
    endTime,
)
```

#### RDS Operations

```go
// Create RDS service
rdsService := aws.NewRDSService(client)

// List all RDS instances
instances, err := rdsService.ListDBInstances(context.Background())
if err != nil {
    log.Printf("Failed to list RDS instances: %v", err)
}

// Get RDS metrics
metrics, err := rdsService.GetDBInstanceMetrics(
    context.Background(),
    "my-database-instance",
    startTime,
    endTime,
)
```

#### Cost Analysis

```go
// Create Cost service
costService := aws.NewCostService(client)

// Get service costs
serviceCosts, err := costService.GetServiceCosts(
    context.Background(),
    time.Now().AddDate(0, -1, 0), // 1 month ago
    time.Now(),
)
if err != nil {
    log.Printf("Failed to get service costs: %v", err)
}

for service, cost := range serviceCosts {
    fmt.Printf("%s: $%.2f\n", service, cost)
}
```

### Resource Collection

```go
// Create resource collector
collector := aws.NewResourceCollector(client)

// Collect all resources
resources, err := collector.CollectResources(context.Background())
if err != nil {
    log.Printf("Failed to collect resources: %v", err)
}

for _, resource := range resources {
    fmt.Printf("%s %s (%s): %s\n",
        resource.Type,
        resource.Name,
        resource.ID,
        resource.State,
    )
}
```

## 🔍 Service Details

### EC2 Service

- **ListInstances**: Retrieve all EC2 instances
- **GetInstanceMetrics**: Get CloudWatch metrics (CPU, network, etc.)
- **Instance State Tracking**: Monitor instance lifecycle

### RDS Service

- **ListDBInstances**: Retrieve all RDS instances
- **GetDBInstanceMetrics**: Get performance metrics
- **Multi-Metric Support**: CPU, connections, storage

### Cost Service

- **GetCostAndUsage**: Detailed cost and usage data
- **GetServiceCosts**: Costs broken down by service
- **Flexible Granularity**: Monthly, daily, hourly support

## 🛡️ Security Features

### Credential Management

```go
// Credentials are automatically masked in logs
creds := client.GetCredentials()
fmt.Printf("Access Key: %s\n", creds.AccessKeyID) // Shows: AKIA***PLE
```

### IAM Role Support

```go
// No explicit credentials needed - uses IAM role
cfg := &config.AWSConfig{
    Region: "us-east-1",
    // No AccessKeyID or SecretAccessKey
}

client, err := aws.NewClient(cfg, logger)
```

## 🏥 Health Monitoring

```go
// Check AWS service health
err := client.Health(context.Background())
if err != nil {
    log.Printf("AWS health check failed: %v", err)
} else {
    log.Println("All AWS services healthy")
}
```

## 📊 Error Handling

The client provides comprehensive error handling:

```go
// Example: Handle specific AWS errors
instances, err := ec2Service.ListInstances(context.Background())
if err != nil {
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.ErrorCode() {
        case "UnauthorizedOperation":
            log.Println("Insufficient permissions")
        case "InvalidInstanceID.NotFound":
            log.Println("Instance not found")
        default:
            log.Printf("AWS API error: %v", err)
        }
    }
    return
}
```

## 🧪 Testing

### Unit Tests

```bash
# Run all AWS client tests
go test ./internal/aws -v

# Run with coverage
go test ./internal/aws -cover

# Run benchmarks
go test ./internal/aws -bench=.
```

### Test Examples

```go
// See examples.go for comprehensive usage examples
func TestExampleUsage() {
    // Run the example usage
    aws.ExampleUsage()
}
```

## 🔧 Integration with Main Application

### In main.go

```go
// Initialize AWS client
awsClient, err := aws.NewClient(&cfg.AWS, logger)
if err != nil {
    logger.Fatal("Failed to initialize AWS client", zap.Error(err))
}
defer awsClient.Close()

// Initialize services
ec2Service := aws.NewEC2Service(awsClient)
rdsService := aws.NewRDSService(awsClient)
costService := aws.NewCostService(awsClient)
resourceCollector := aws.NewResourceCollector(awsClient)

// Start periodic resource collection
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        resources, err := resourceCollector.CollectResources(ctx)
        if err != nil {
            logger.Error("Failed to collect resources", zap.Error(err))
            continue
        }
        
        // Process resources and generate recommendations
        processResources(resources)
    }
}()
```

## 📈 Performance Considerations

### Connection Pooling

The AWS SDK v2 uses connection pooling automatically. The client is designed to be reused:

```go
// ✅ Good: Reuse client
client, _ := aws.NewClient(cfg, logger)
defer client.Close()

for i := 0; i < 100; i++ {
    instances, _ := client.EC2Service.ListInstances(ctx)
    // Process instances
}

// ❌ Bad: Creating new client for each request
for i := 0; i < 100; i++ {
    client, _ := aws.NewClient(cfg, logger)
    instances, _ := client.EC2Service.ListInstances(ctx)
    client.Close()
}
```

### Rate Limiting

The SDK handles rate limiting automatically, but you should implement backoff for bulk operations:

```go
// Implement exponential backoff for bulk operations
for _, instanceID := range instanceIDs {
    metrics, err := ec2Service.GetInstanceMetrics(ctx, instanceID, start, end)
    if err != nil {
        if isRateLimitError(err) {
            time.Sleep(time.Second * time.Duration(backoff))
            backoff *= 2
            continue
        }
        log.Printf("Failed to get metrics for %s: %v", instanceID, err)
    }
    backoff = 1 // Reset backoff
}
```

## 🚨 Best Practices

1. **Use IAM Roles**: Prefer IAM roles over access keys when possible
2. **Least Privilege**: Grant only necessary permissions
3. **Credential Rotation**: Regularly rotate access keys
4. **Region Specificity**: Configure specific regions for better performance
5. **Error Handling**: Always handle AWS API errors gracefully
6. **Logging**: Use structured logging for better observability
7. **Resource Cleanup**: Always close clients when done
8. **Context Usage**: Use context for timeout and cancellation

## 🔗 Related Services

- **Database**: PostgreSQL with pgx driver
- **Logging**: Zap structured logging
- **Configuration**: Environment-based configuration
- **Migration**: golang-migrate for schema management

## 📚 Additional Resources

- [AWS SDK for Go v2 Documentation](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2)
- [AWS Cost Explorer API](https://docs.aws.amazon.com/cost-management/latest/user-guide/ce-api.html)
- [CloudWatch Metrics](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/working_with_metrics.html)
- [EC2 Instance Types](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html)

## 🆘 Troubleshooting

### Common Issues

1. **Credential Errors**
   ```bash
   # Check environment variables
   env | grep AWS_
   
   # Verify credentials
   aws sts get-caller-identity
   ```

2. **Region Issues**
   ```bash
   # Set default region
   aws configure set region us-east-1
   ```

3. **Permission Errors**
   ```bash
   # Check IAM permissions
   aws iam list-attached-user-policies --user-name $USER
   ```

### Debug Mode

Enable debug logging for troubleshooting:

```go
// Enable AWS SDK debug logging
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),
    config.WithLogLevel(aws.LogDebug),
)
```

## 📄 License

This module is part of the DevCost AI project and follows the same licensing terms.
