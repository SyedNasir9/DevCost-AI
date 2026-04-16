package ec2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/pkg/logger"
)

// DiscoveryService handles EC2 resource discovery
type DiscoveryService struct {
	client *aws.Client
	logger *logger.Logger
	ec2    *ec2.Client
}

// NewDiscoveryService creates a new EC2 discovery service
func NewDiscoveryService(awsClient *aws.Client) *DiscoveryService {
	return &DiscoveryService{
		client: awsClient,
		logger: awsClient.logger,
		ec2:    awsClient.EC2,
	}
}

// DiscoverAllInstances discovers all EC2 instances with pagination support
func (s *DiscoveryService) DiscoverAllInstances(ctx context.Context) ([]*models.EC2Resource, error) {
	s.logger.Info("Starting EC2 instance discovery")

	var allInstances []*models.EC2Resource
	var nextToken *string

	page := 1
	for {
		s.logger.Debug("Fetching EC2 instances page",
			zap.Int("page", page),
			zap.String("next_token", func() string {
				if nextToken == nil {
					return "none"
				}
				return *nextToken
			}()),
		)

		// Describe instances with pagination
		input := &ec2.DescribeInstancesInput{
			MaxResults: aws.Int32(1000), // Maximum allowed by AWS
		}

		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.ec2.DescribeInstances(ctx, input)
		if err != nil {
			s.logger.Error("Failed to describe EC2 instances", zap.Error(err))
			return nil, s.handleAWSError(err, "DescribeInstances")
		}

		// Process instances from this page
		pageInstances, err := s.processInstancePage(result.Reservations)
		if err != nil {
			return nil, fmt.Errorf("failed to process instance page: %w", err)
		}

		allInstances = append(allInstances, pageInstances...)

		s.logger.Info("Processed EC2 instances page",
			zap.Int("page", page),
			zap.Int("instances_in_page", len(pageInstances)),
			zap.Int("total_instances", len(allInstances)),
		)

		// Check for next page
		nextToken = result.NextToken
		if nextToken == nil {
			break
		}

		page++
	}

	s.logger.Info("EC2 instance discovery completed",
		zap.Int("total_instances", len(allInstances)),
		zap.Int("pages_processed", page),
	)

	return allInstances, nil
}

// DiscoverInstancesByFilter discovers EC2 instances with specific filters
func (s *DiscoveryService) DiscoverInstancesByFilter(ctx context.Context, filters []types.Filter) ([]*models.EC2Resource, error) {
	s.logger.Info("Starting filtered EC2 instance discovery",
		zap.Int("filter_count", len(filters)),
	)

	var allInstances []*models.EC2Resource
	var nextToken *string

	page := 1
	for {
		input := &ec2.DescribeInstancesInput{
			Filters:    filters,
			MaxResults: aws.Int32(1000),
		}

		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.ec2.DescribeInstances(ctx, input)
		if err != nil {
			s.logger.Error("Failed to describe filtered EC2 instances", zap.Error(err))
			return nil, s.handleAWSError(err, "DescribeInstances")
		}

		pageInstances, err := s.processInstancePage(result.Reservations)
		if err != nil {
			return nil, fmt.Errorf("failed to process filtered instance page: %w", err)
		}

		allInstances = append(allInstances, pageInstances...)

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}

		page++
	}

	s.logger.Info("Filtered EC2 instance discovery completed",
		zap.Int("total_instances", len(allInstances)),
	)

	return allInstances, nil
}

// DiscoverRunningInstances discovers only running EC2 instances
func (s *DiscoveryService) DiscoverRunningInstances(ctx context.Context) ([]*models.EC2Resource, error) {
	filters := []types.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: []string{"running"},
		},
	}

	return s.DiscoverInstancesByFilter(ctx, filters)
}

// DiscoverInstancesByTag discovers EC2 instances by tag
func (s *DiscoveryService) DiscoverInstancesByTag(ctx context.Context, tagKey, tagValue string) ([]*models.EC2Resource, error) {
	filters := []types.Filter{
		{
			Name:   aws.String("tag:" + tagKey),
			Values: []string{tagValue},
		},
	}

	return s.DiscoverInstancesByFilter(ctx, filters)
}

// DiscoverInstanceByID discovers a specific EC2 instance by ID
func (s *DiscoveryService) DiscoverInstanceByID(ctx context.Context, instanceID string) (*models.EC2Resource, error) {
	s.logger.Info("Discovering specific EC2 instance",
		zap.String("instance_id", instanceID),
	)

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	result, err := s.ec2.DescribeInstances(ctx, input)
	if err != nil {
		s.logger.Error("Failed to describe EC2 instance",
			zap.String("instance_id", instanceID),
			zap.Error(err),
		)
		return nil, s.handleAWSError(err, "DescribeInstances")
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	instance := result.Reservations[0].Instances[0]
	ec2Resource, err := s.mapInstanceToResource(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to map instance %s to resource: %w", instanceID, err)
	}

	s.logger.Info("Successfully discovered EC2 instance",
		zap.String("instance_id", instanceID),
		zap.String("instance_type", ec2Resource.InstanceType),
		zap.String("state", string(ec2Resource.State)),
	)

	return ec2Resource, nil
}

// processInstancePage processes a single page of EC2 instances
func (s *DiscoveryService) processInstancePage(reservations []types.Reservation) ([]*models.EC2Resource, error) {
	var instances []*models.EC2Resource

	for _, reservation := range reservations {
		for _, instance := range reservation.Instances {
			ec2Resource, err := s.mapInstanceToResource(instance)
			if err != nil {
				s.logger.Error("Failed to map instance to resource",
					zap.String("instance_id", aws.ToString(instance.InstanceId)),
					zap.Error(err),
				)
				continue // Skip this instance but continue processing others
			}

			instances = append(instances, ec2Resource)
		}
	}

	return instances, nil
}

// mapInstanceToResource maps an AWS EC2 instance to our internal model
func (s *DiscoveryService) mapInstanceToResource(instance types.Instance) (*models.EC2Resource, error) {
	instanceID := aws.ToString(instance.InstanceId)
	instanceType := aws.ToString(instance.InstanceType)
	region := s.client.GetRegion()
	accountID := s.getAccountID(instance)

	// Create base EC2 resource
	ec2Resource := models.NewEC2Resource(instanceID, instanceType, region, accountID)

	// Map basic fields
	ec2Resource.State = models.ResourceState(aws.ToString(instance.State.Name))
	ec2Resource.LaunchTime = aws.ToTime(instance.LaunchTime)

	// Map network information
	if instance.Placement != nil {
		ec2Resource.AvailabilityZone = aws.ToString(instance.Placement.AvailabilityZone)
	}

	if instance.SubnetId != nil {
		ec2Resource.SubnetID = aws.ToString(instance.SubnetId)
	}

	if instance.VpcId != nil {
		ec2Resource.VpcID = aws.ToString(instance.VpcId)
	}

	// Map IP addresses
	if instance.PublicIpAddress != nil {
		ec2Resource.PublicIP = aws.ToString(instance.PublicIpAddress)
	}

	if instance.PrivateIpAddress != nil {
		ec2Resource.PrivateIP = aws.ToString(instance.PrivateIpAddress)
	}

	// Map security groups
	for _, sg := range instance.SecurityGroups {
		if sg.GroupId != nil {
			ec2Resource.SecurityGroups = append(ec2Resource.SecurityGroups, aws.ToString(sg.GroupId))
		}
	}

	// Map additional attributes
	if instance.KeyName != nil {
		ec2Resource.KeyName = aws.ToString(instance.KeyName)
	}

	if instance.Platform != nil {
		ec2Resource.Platform = string(instance.Platform)
	}

	if instance.Architecture != nil {
		ec2Resource.Architecture = string(instance.Architecture)
	}

	if instance.Hypervisor != nil {
		ec2Resource.Hypervisor = string(instance.Hypervisor)
	}

	if instance.VirtualizationType != nil {
		ec2Resource.VirtualizationType = string(instance.VirtualizationType)
	}

	if instance.InstanceLifecycle != nil {
		ec2Resource.Lifecycle = string(instance.InstanceLifecycle)
	}

	if instance.Monitoring != nil {
		ec2Resource.MonitoringState = string(instance.Monitoring.State)
	}

	// Map tags
	ec2Resource.Tags = s.mapTags(instance.Tags)

	// Set name from tags or use instance ID
	if name, exists := ec2Resource.GetTag("Name"); exists && name != "" {
		ec2Resource.Name = name
	} else {
		ec2Resource.Name = instanceID
	}

	s.logger.Debug("Mapped EC2 instance to resource",
		zap.String("instance_id", instanceID),
		zap.String("name", ec2Resource.Name),
		zap.String("state", string(ec2Resource.State)),
		zap.Int("tag_count", len(ec2Resource.Tags)),
	)

	return ec2Resource, nil
}

// mapTags converts AWS tags to a map[string]string
func (s *DiscoveryService) mapTags(tags []types.Tag) map[string]string {
	result := make(map[string]string)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return result
}

// getAccountID extracts account ID from instance ARN or other sources
func (s *DiscoveryService) getAccountID(instance types.Instance) string {
	// Try to get account ID from instance ARN if available
	if instance.IamInstanceProfile != nil && instance.IamInstanceProfile.Arn != nil {
		arn := aws.ToString(instance.IamInstanceProfile.Arn)
		parts := strings.Split(arn, ":")
		if len(parts) >= 5 {
			return parts[4] // Account ID is the 5th part in ARN
		}
	}

	// Fallback: this would typically come from AWS STS or configuration
	// For now, we'll return a placeholder that should be set from the AWS client
	return "unknown"
}

// handleAWSError handles AWS API errors with proper categorization
func (s *DiscoveryService) handleAWSError(err error, operation string) error {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "UnauthorizedOperation":
			s.logger.Error("AWS authorization failed",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
				zap.String("message", apiErr.ErrorMessage()),
			)
			return fmt.Errorf("insufficient permissions for %s: %s", operation, apiErr.ErrorMessage())

		case "InvalidInstanceID.NotFound":
			s.logger.Error("EC2 instance not found",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
				zap.String("message", apiErr.ErrorMessage()),
			)
			return fmt.Errorf("instance not found: %s", apiErr.ErrorMessage())

		case "RequestLimitExceeded":
			s.logger.Error("AWS request limit exceeded",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
			)
			return fmt.Errorf("AWS request limit exceeded, please retry later")

		case "ServiceUnavailable":
			s.logger.Error("AWS service unavailable",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
			)
			return fmt.Errorf("AWS service temporarily unavailable")

		default:
			s.logger.Error("AWS API error",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
				zap.String("message", apiErr.ErrorMessage()),
			)
			return fmt.Errorf("AWS API error %s: %s", apiErr.ErrorCode(), apiErr.ErrorMessage())
		}
	}

	// Non-API errors
	s.logger.Error("Non-API error occurred",
		zap.String("operation", operation),
		zap.Error(err),
	)
	return fmt.Errorf("error in %s: %w", operation, err)
}

// GetInstanceCount returns the total count of EC2 instances
func (s *DiscoveryService) GetInstanceCount(ctx context.Context) (int, error) {
	s.logger.Info("Getting EC2 instance count")

	input := &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(5), // We only need the first page to get the count
	}

	result, err := s.ec2.DescribeInstances(ctx, input)
	if err != nil {
		s.logger.Error("Failed to get EC2 instance count", zap.Error(err))
		return 0, s.handleAWSError(err, "DescribeInstances")
	}

	// Count instances in the first page
	count := 0
	for _, reservation := range result.Reservations {
		count += len(reservation.Instances)
	}

	// If there are more pages, we need to count all instances
	if result.NextToken != nil {
		s.logger.Info("Multiple pages detected, counting all instances")
		
		var nextToken = result.NextToken
		for {
			input := &ec2.DescribeInstancesInput{
				NextToken:  nextToken,
				MaxResults: aws.Int32(1000),
			}

			result, err := s.ec2.DescribeInstances(ctx, input)
			if err != nil {
				s.logger.Error("Failed to get EC2 instance count page", zap.Error(err))
				return count, s.handleAWSError(err, "DescribeInstances")
			}

			for _, reservation := range result.Reservations {
				count += len(reservation.Instances)
			}

			nextToken = result.NextToken
			if nextToken == nil {
				break
			}
		}
	}

	s.logger.Info("EC2 instance count retrieved", zap.Int("count", count))
	return count, nil
}

// DiscoverInstancesWithRetry discovers instances with retry logic for transient errors
func (s *DiscoveryService) DiscoverInstancesWithRetry(ctx context.Context, maxRetries int) ([]*models.EC2Resource, error) {
	var instances []*models.EC2Resource
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Info("Retrying EC2 instance discovery",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
			)

			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		instances, lastErr = s.DiscoverAllInstances(ctx)
		if lastErr == nil {
			return instances, nil
		}

		// Check if error is retryable
		if !s.isRetryableError(lastErr) {
			break
		}

		s.logger.Warn("EC2 instance discovery failed, will retry",
			zap.Int("attempt", attempt),
			zap.Error(lastErr),
		)
	}

	return nil, fmt.Errorf("failed to discover EC2 instances after %d attempts: %w", maxRetries+1, lastErr)
}

// isRetryableError checks if an error is retryable
func (s *DiscoveryService) isRetryableError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "RequestLimitExceeded", "ServiceUnavailable", "InternalError", "Throttling":
			return true
		default:
			return false
		}
	}
	return false
}
