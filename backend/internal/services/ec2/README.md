# EC2 Resource Discovery Service

This module provides comprehensive EC2 resource discovery capabilities for DevCost AI, including pagination support, error handling, and mapping to internal resource models.

## 🚀 Features

- **Full Resource Discovery**: Fetch all EC2 instances with complete metadata
- **Pagination Support**: Handle large-scale deployments efficiently
- **Flexible Filtering**: Discover instances by state, tags, instance type, and custom filters
- **Error Handling**: Comprehensive AWS API error handling with retry logic
- **Resource Mapping**: Convert AWS instances to internal resource models
- **Production-Ready**: Structured logging, metrics, and monitoring support

## 📁 Module Structure

```
internal/services/ec2/
├── discovery.go        # Main discovery service implementation
├── discovery_test.go   # Comprehensive unit tests and benchmarks
├── examples.go         # Usage examples and integration patterns
└── README.md          # This documentation
```

## 🔧 Usage

### Basic Discovery

```go
import (
    "devcost-ai/internal/services/ec2"
    "devcost-ai/internal/aws"
)

// Create discovery service
discoveryService := ec2.NewDiscoveryService(awsClient)

// Discover all instances
instances, err := discoveryService.DiscoverAllInstances(context.Background())
if err != nil {
    log.Printf("Failed to discover instances: %v", err)
}

fmt.Printf("Found %d EC2 instances\n", len(instances))
```

### Filtered Discovery

```go
// Discover only running instances
runningInstances, err := discoveryService.DiscoverRunningInstances(context.Background())

// Discover instances by tag
tagInstances, err := discoveryService.DiscoverInstancesByTag(
    context.Background(),
    "Environment",
    "production",
)

// Discover instances with custom filters
filters := []types.Filter{
    {
        Name:   aws.String("instance-type"),
        Values: []string{"t3.micro", "t3.small"},
    },
    {
        Name:   aws.String("tag:Owner"),
        Values: []string{"team-a"},
    },
}
filteredInstances, err := discoveryService.DiscoverInstancesByFilter(context.Background(), filters)
```

### Resource Mapping

```go
// Convert to internal resource model
for _, instance := range instances {
    resource := instance.ToResource()
    
    fmt.Printf("Resource: %s (%s) [%s]\n",
        resource.Name,
        resource.InstanceType,
        resource.State,
    )
    
    // Access tags and metadata
    if environment, exists := resource.GetTag("Environment"); exists {
        fmt.Printf("Environment: %s\n", environment)
    }
}
```

## 📊 Resource Model

### EC2Resource Structure

```go
type EC2Resource struct {
    Resource
    InstanceType        string    `json:"instance_type"`
    AvailabilityZone    string    `json:"availability_zone"`
    SubnetID           string    `json:"subnet_id"`
    VpcID              string    `json:"vpc_id"`
    SecurityGroups     []string  `json:"security_groups"`
    KeyName            string    `json:"key_name"`
    LaunchTime         time.Time `json:"launch_time"`
    PublicIP           string    `json:"public_ip"`
    PrivateIP          string    `json:"private_ip"`
    Platform           string    `json:"platform"`
    Architecture       string    `json:"architecture"`
    Hypervisor         string    `json:"hypervisor"`
    VirtualizationType string   `json:"virtualization_type"`
    Lifecycle          string    `json:"lifecycle"`
    MonitoringState    string    `json:"monitoring_state"`
}
```

### Base Resource Model

```go
type Resource struct {
    ID           string                 `json:"id"`
    ResourceID   string                 `json:"resource_id"`
    ResourceType ResourceType           `json:"resource_type"`
    Provider     string                 `json:"provider"`
    Region       string                 `json:"region"`
    AccountID    string                 `json:"account_id"`
    Name         string                 `json:"name"`
    State        ResourceState          `json:"state"`
    InstanceType string                 `json:"instance_type"`
    Tags         map[string]string      `json:"tags"`
    Metadata     map[string]interface{} `json:"metadata"`
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
}
```

## 🔍 Discovery Methods

### DiscoverAllInstances

Discovers all EC2 instances with automatic pagination handling.

```go
instances, err := discoveryService.DiscoverAllInstances(context.Background())
```

**Features:**
- Automatic pagination (1000 instances per page)
- Comprehensive error handling
- Structured logging with progress tracking
- Memory-efficient processing

### DiscoverRunningInstances

Discovers only instances in running state.

```go
runningInstances, err := discoveryService.DiscoverRunningInstances(context.Background())
```

### DiscoverInstancesByTag

Discovers instances with specific tag values.

```go
instances, err := discoveryService.DiscoverInstancesByTag(
    context.Background(),
    "Environment",
    "production",
)
```

### DiscoverInstancesByFilter

Discovers instances using custom AWS filters.

```go
filters := []types.Filter{
    {
        Name:   aws.String("instance-state-name"),
        Values: []string{"running", "stopped"},
    },
    {
        Name:   aws.String("tag:Owner"),
        Values: []string{"team-a", "team-b"},
    },
}
instances, err := discoveryService.DiscoverInstancesByFilter(context.Background(), filters)
```

### DiscoverInstanceByID

Discovers a specific instance by its ID.

```go
instance, err := discoveryService.DiscoverInstanceByID(context.Background(), "i-1234567890abcdef0")
```

## 🛡️ Error Handling

### AWS API Error Categories

```go
// Authorization errors
"UnauthorizedOperation" → Insufficient permissions

// Resource errors
"InvalidInstanceID.NotFound" → Instance not found

// Rate limiting
"RequestLimitExceeded" → Too many requests

// Service issues
"ServiceUnavailable" → AWS service temporarily unavailable
```

### Retry Logic

```go
// Automatic retry for transient errors
instances, err := discoveryService.DiscoverInstancesWithRetry(context.Background(), 3)
```

**Retryable Errors:**
- RequestLimitExceeded
- ServiceUnavailable
- InternalError
- Throttling

**Non-Retryable Errors:**
- UnauthorizedOperation
- InvalidInstanceID.NotFound
- InvalidParameter

## 📈 Performance & Scalability

### Pagination Handling

The service automatically handles pagination for large deployments:

```go
// Each page can contain up to 1000 instances
// Automatic next token handling
// Progress logging for monitoring
```

### Memory Efficiency

```go
// Process instances page by page
// Avoid loading all instances into memory at once
// Graceful handling of partial failures
```

### Benchmark Performance

```bash
# Run benchmarks
go test ./internal/services/ec2 -bench=.

# Example results:
# BenchmarkMapTags-8         	1000000	       1200 ns/op
# BenchmarkMapInstanceToResource-8   	 100000	      15000 ns/op
```

## 🧪 Testing

### Unit Tests

```bash
# Run all tests
go test ./internal/services/ec2 -v

# Run with coverage
go test ./internal/services/ec2 -cover

# Run benchmarks
go test ./internal/services/ec2 -bench=.
```

### Test Coverage

- **Service Creation**: Client initialization and configuration
- **Resource Mapping**: AWS instance to internal model conversion
- **Tag Processing**: Tag extraction and mapping
- **Error Handling**: AWS API error categorization
- **Pagination**: Page processing and token handling
- **Retry Logic**: Transient error retry behavior

### Mock Testing

For testing without AWS credentials, use the provided test utilities:

```go
// Create mock service
service := &DiscoveryService{
    client: &aws.Client{
        Region: "us-east-1",
        logger: logger.NewLogger(zaptest.NewLogger(t)),
    },
}

// Test mapping logic
instance := types.Instance{
    InstanceId:   aws.String("i-1234567890abcdef0"),
    InstanceType: aws.String("t3.micro"),
    State: &types.InstanceState{
        Name: types.InstanceStateNameRunning,
    },
}

resource, err := service.mapInstanceToResource(instance)
```

## 🔧 Integration Examples

### Background Service

```go
func StartDiscoveryService(ctx context.Context, awsClient *aws.Client) {
    discoveryService := ec2.NewDiscoveryService(awsClient)
    
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            instances, err := discoveryService.DiscoverInstancesWithRetry(ctx, 3)
            if err != nil {
                log.Printf("Discovery failed: %v", err)
                continue
            }
            
            // Process instances and store in database
            processInstances(instances)
        }
    }
}
```

### API Integration

```go
// HTTP handler for instance listing
func ListInstancesHandler(c *gin.Context) {
    discoveryService := ec2.NewDiscoveryService(getAWSClient())
    
    instances, err := discoveryService.DiscoverAllInstances(c.Request.Context())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // Convert to API response format
    response := make([]map[string]interface{}, len(instances))
    for i, instance := range instances {
        response[i] = map[string]interface{}{
            "id":           instance.ResourceID,
            "name":         instance.Name,
            "type":         instance.InstanceType,
            "state":        instance.State,
            "region":       instance.Region,
            "availability_zone": instance.AvailabilityZone,
            "tags":         instance.Tags,
        }
    }
    
    c.JSON(200, response)
}
```

## 📊 Monitoring & Observability

### Structured Logging

```go
// Discovery progress
logger.Info("Fetching EC2 instances page",
    zap.Int("page", page),
    zap.String("next_token", nextToken),
)

// Completion summary
logger.Info("EC2 instance discovery completed",
    zap.Int("total_instances", len(allInstances)),
    zap.Int("pages_processed", page),
    zap.Duration("duration", time.Since(start)),
)
```

### Metrics Collection

```go
// Track discovery metrics
discoveryMetrics := map[string]int{
    "total_instances": len(instances),
    "running_instances": runningCount,
    "stopped_instances": stoppedCount,
    "pages_processed": pageCount,
    "api_calls": apiCallCount,
}
```

## 🚨 Best Practices

### 1. Resource Management

```go
// Always close AWS client
defer awsClient.Close()

// Use context for timeout and cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

### 2. Error Handling

```go
// Use retry logic for transient errors
instances, err := discoveryService.DiscoverInstancesWithRetry(ctx, 3)

// Handle specific error types
var apiErr smithy.APIError
if errors.As(err, &apiErr) {
    switch apiErr.ErrorCode() {
    case "UnauthorizedOperation":
        // Handle permission errors
    case "RequestLimitExceeded":
        // Handle rate limiting
    }
}
```

### 3. Performance Optimization

```go
// Use filtered discovery when possible
filters := []types.Filter{
    {Name: aws.String("instance-state-name"), Values: []string{"running"}},
}
instances, err := discoveryService.DiscoverInstancesByFilter(ctx, filters)

// Process instances in batches
batchSize := 100
for i := 0; i < len(instances); i += batchSize {
    end := i + batchSize
    if end > len(instances) {
        end = len(instances)
    }
    
    batch := instances[i:end]
    processBatch(batch)
}
```

### 4. Security

```go
// Use IAM roles instead of access keys when possible
awsConfig := &config.AWSConfig{
    Region: "us-east-1",
    // No AccessKeyID or SecretAccessKey - will use IAM role
}

// Implement least privilege access
// Required permissions:
// - ec2:DescribeInstances
// - ec2:DescribeTags
// - ec2:DescribeRegions
```

## 🔗 Related Services

- **AWS Client**: `internal/aws/client.go`
- **Resource Models**: `internal/models/resource.go`
- **Database Integration**: `internal/db/database.go`
- **Metrics Collection**: `internal/aws/services.go`

## 📚 Additional Resources

- [AWS EC2 API Documentation](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/)
- [AWS SDK for Go v2](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2)
- [EC2 Instance Types](https://aws.amazon.com/ec2/instance-types/)
- [AWS Cost Explorer](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/cost-explorer.html)

## 🆘 Troubleshooting

### Common Issues

1. **Permission Errors**
   ```bash
   # Check IAM permissions
   aws iam simulate-principal-policy \
     --policy-source-arn arn:aws:iam::123456789012:user/devcost-ai \
     --action-names ec2:DescribeInstances \
     --resource-arns arn:aws:ec2:us-east-1:123456789012:instance/*
   ```

2. **Rate Limiting**
   ```go
   // Implement exponential backoff
   discoveryService.DiscoverInstancesWithRetry(ctx, 5)
   ```

3. **Large Deployments**
   ```go
   // Use filtered discovery to reduce API calls
   filters := []types.Filter{
     {Name: aws.String("instance-state-name"), Values: []string{"running"}},
   }
   ```

### Debug Mode

Enable debug logging for troubleshooting:

```go
// Enable AWS SDK debug logging
awsConfig, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),
    config.WithLogLevel(aws.LogDebug),
)
```

## 📄 License

This module is part of the DevCost AI project and follows the same licensing terms.
