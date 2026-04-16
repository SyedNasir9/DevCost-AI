package rds

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/smithy-go"
	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/pkg/logger"
)

// DiscoveryService handles RDS resource discovery
type DiscoveryService struct {
	client *aws.Client
	logger *logger.Logger
	rds    *rds.Client
}

// NewDiscoveryService creates a new RDS discovery service
func NewDiscoveryService(awsClient *aws.Client) *DiscoveryService {
	return &DiscoveryService{
		client: awsClient,
		logger: awsClient.logger,
		rds:    awsClient.RDS,
	}
}

// DiscoverAllDBInstances discovers all RDS instances with pagination support
func (s *DiscoveryService) DiscoverAllDBInstances(ctx context.Context) ([]*models.RDSResource, error) {
	s.logger.Info("Starting RDS instance discovery")

	var allInstances []*models.RDSResource
	var marker *string

	page := 1
	for {
		s.logger.Debug("Fetching RDS instances page",
			zap.Int("page", page),
			zap.String("marker", func() string {
				if marker == nil {
					return "none"
				}
				return *marker
			}()),
		)

		// Describe instances with pagination
		input := &rds.DescribeDBInstancesInput{
			MaxRecords: aws.Int32(100), // Maximum allowed by AWS
		}

		if marker != nil {
			input.Marker = marker
		}

		result, err := s.rds.DescribeDBInstances(ctx, input)
		if err != nil {
			s.logger.Error("Failed to describe RDS instances", zap.Error(err))
			return nil, s.handleAWSError(err, "DescribeDBInstances")
		}

		// Process instances from this page
		pageInstances, err := s.processInstancePage(result.DBInstances)
		if err != nil {
			return nil, fmt.Errorf("failed to process RDS instance page: %w", err)
		}

		allInstances = append(allInstances, pageInstances...)

		s.logger.Info("Processed RDS instances page",
			zap.Int("page", page),
			zap.Int("instances_in_page", len(pageInstances)),
			zap.Int("total_instances", len(allInstances)),
		)

		// Check for next page
		marker = result.Marker
		if marker == nil {
			break
		}

		page++
	}

	s.logger.Info("RDS instance discovery completed",
		zap.Int("total_instances", len(allInstances)),
		zap.Int("pages_processed", page),
	)

	return allInstances, nil
}

// DiscoverDBInstancesByFilter discovers RDS instances with specific filters
func (s *DiscoveryService) DiscoverDBInstancesByFilter(ctx context.Context, filters []types.Filter) ([]*models.RDSResource, error) {
	s.logger.Info("Starting filtered RDS instance discovery",
		zap.Int("filter_count", len(filters)),
	)

	var allInstances []*models.RDSResource
	var marker *string

	page := 1
	for {
		input := &rds.DescribeDBInstancesInput{
			Filters:    filters,
			MaxRecords: aws.Int32(100),
		}

		if marker != nil {
			input.Marker = marker
		}

		result, err := s.rds.DescribeDBInstances(ctx, input)
		if err != nil {
			s.logger.Error("Failed to describe filtered RDS instances", zap.Error(err))
			return nil, s.handleAWSError(err, "DescribeDBInstances")
		}

		pageInstances, err := s.processInstancePage(result.DBInstances)
		if err != nil {
			return nil, fmt.Errorf("failed to process filtered RDS instance page: %w", err)
		}

		allInstances = append(allInstances, pageInstances...)

		marker = result.Marker
		if marker == nil {
			break
		}

		page++
	}

	s.logger.Info("Filtered RDS instance discovery completed",
		zap.Int("total_instances", len(allInstances)),
	)

	return allInstances, nil
}

// DiscoverAvailableDBInstances discovers only available RDS instances
func (s *DiscoveryService) DiscoverAvailableDBInstances(ctx context.Context) ([]*models.RDSResource, error) {
	filters := []types.Filter{
		{
			Name:   aws.String("engine"),
			Values: []string{"mysql", "postgres", "mariadb", "aurora-mysql", "aurora-postgresql"},
		},
	}

	return s.DiscoverDBInstancesByFilter(ctx, filters)
}

// DiscoverDBInstanceByID discovers a specific RDS instance by ID
func (s *DiscoveryService) DiscoverDBInstanceByID(ctx context.Context, dbInstanceID string) (*models.RDSResource, error) {
	s.logger.Info("Discovering specific RDS instance",
		zap.String("db_instance_id", dbInstanceID),
	)

	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstanceID),
	}

	result, err := s.rds.DescribeDBInstances(ctx, input)
	if err != nil {
		s.logger.Error("Failed to describe RDS instance",
			zap.String("db_instance_id", dbInstanceID),
			zap.Error(err),
		)
		return nil, s.handleAWSError(err, "DescribeDBInstances")
	}

	if len(result.DBInstances) == 0 {
		return nil, fmt.Errorf("RDS instance %s not found", dbInstanceID)
	}

	instance := result.DBInstances[0]
	rdsResource, err := s.mapInstanceToResource(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to map RDS instance %s to resource: %w", dbInstanceID, err)
	}

	s.logger.Info("Successfully discovered RDS instance",
		zap.String("db_instance_id", dbInstanceID),
		zap.String("engine", rdsResource.Engine),
		zap.String("status", string(rdsResource.Status)),
	)

	return rdsResource, nil
}

// processInstancePage processes a single page of RDS instances
func (s *DiscoveryService) processInstancePage(instances []types.DBInstance) ([]*models.RDSResource, error) {
	var rdsInstances []*models.RDSResource

	for _, instance := range instances {
		rdsResource, err := s.mapInstanceToResource(instance)
		if err != nil {
			s.logger.Error("Failed to map RDS instance to resource",
				zap.String("db_instance_id", aws.ToString(instance.DBInstanceIdentifier)),
				zap.Error(err),
			)
			continue // Skip this instance but continue processing others
		}

		rdsInstances = append(rdsInstances, rdsResource)
	}

	return rdsInstances, nil
}

// mapInstanceToResource maps an AWS RDS instance to our internal model
func (s *DiscoveryService) mapInstanceToResource(instance types.DBInstance) (*models.RDSResource, error) {
	dbInstanceID := aws.ToString(instance.DBInstanceIdentifier)
	instanceClass := aws.ToString(instance.DBInstanceClass)
	region := s.client.GetRegion()
	accountID := s.getAccountID(instance)

	// Create base RDS resource
	rdsResource := models.NewRDSResource(dbInstanceID, instanceClass, region, accountID)

	// Map basic fields
	rdsResource.Status = models.ResourceState(aws.ToString(instance.DBInstanceStatus))
	rdsResource.Engine = aws.ToString(instance.Engine)
	rdsResource.EngineVersion = aws.ToString(instance.EngineVersion)
	rdsResource.LicenseModel = aws.ToString(instance.LicenseModel)
	rdsResource.MultiAZ = instance.MultiAZ
	rdsResource.PubliclyAccessible = instance.PubliclyAccessible
	rdsResource.StorageType = aws.ToString(instance.StorageType)
	rdsResource.StorageEncrypted = instance.StorageEncrypted
	rdsResource.AllocatedStorage = aws.ToInt32(instance.AllocatedStorage)
	rdsResource.CreatedAt = aws.ToTime(instance.InstanceCreateTime)

	// Map endpoint information
	if instance.Endpoint != nil {
		rdsResource.Endpoint = models.Endpoint{
			Address: aws.ToString(instance.Endpoint.Address),
			Port:    aws.ToInt32(instance.Endpoint.Port),
			HostedZoneId: aws.ToString(instance.Endpoint.HostedZoneId),
		}
	}

	// Map availability zone information
	if instance.AvailabilityZone != nil {
		rdsResource.AvailabilityZone = aws.ToString(instance.AvailabilityZone)
	}

	// Map subnet group information
	if instance.DBSubnetGroup != nil {
		rdsResource.SubnetGroupName = aws.ToString(instance.DBSubnetGroup.DBSubnetGroupName)
		rdsResource.SubnetGroupVpcID = aws.ToString(instance.DBSubnetGroup.VpcId)
	}

	// Map security group information
	for _, sg := range instance.VpcSecurityGroups {
		if sg.VpcSecurityGroupId != nil {
			rdsResource.VpcSecurityGroups = append(rdsResource.VpcSecurityGroups, aws.ToString(sg.VpcSecurityGroupId))
		}
	}

	// Map parameter group information
	if instance.DBParameterGroups != nil && len(instance.DBParameterGroups) > 0 {
		rdsResource.ParameterGroupName = aws.ToString(instance.DBParameterGroups[0].DBParameterGroupName)
	}

	// Map option group information
	if instance.OptionGroupMemberships != nil && len(instance.OptionGroupMemberships) > 0 {
		rdsResource.OptionGroupName = aws.ToString(instance.OptionGroupMemberships[0].OptionGroupName)
	}

	// Map backup and maintenance information
	if instance.BackupRetentionPeriod != nil {
		rdsResource.BackupRetentionPeriod = aws.ToInt32(instance.BackupRetentionPeriod)
	}

	if instance.PreferredBackupWindow != nil {
		rdsResource.PreferredBackupWindow = aws.ToString(instance.PreferredBackupWindow)
	}

	if instance.PreferredMaintenanceWindow != nil {
		rdsResource.PreferredMaintenanceWindow = aws.ToString(instance.PreferredMaintenanceWindow)
	}

	rdsResource.AutoMinorVersionUpgrade = instance.AutoMinorVersionUpgrade

	// Map tags
	rdsResource.Tags = s.mapTags(instance.Tags)

	// Set name from tags or use instance identifier
	if name, exists := rdsResource.GetTag("Name"); exists && name != "" {
		rdsResource.Name = name
	} else {
		rdsResource.Name = dbInstanceID
	}

	s.logger.Debug("Mapped RDS instance to resource",
		zap.String("db_instance_id", dbInstanceID),
		zap.String("name", rdsResource.Name),
		zap.String("engine", rdsResource.Engine),
		zap.String("status", string(rdsResource.Status)),
		zap.Int("tag_count", len(rdsResource.Tags)),
	)

	return rdsResource, nil
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
func (s *DiscoveryService) getAccountID(instance types.DBInstance) string {
	// Try to get account ID from instance ARN if available
	if instance.DBInstanceArn != nil {
		arn := aws.ToString(instance.DBInstanceArn)
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

		case "DBInstanceNotFound":
			s.logger.Error("RDS instance not found",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
				zap.String("message", apiErr.ErrorMessage()),
			)
			return fmt.Errorf("RDS instance not found: %s", apiErr.ErrorMessage())

		case "InvalidParameterValue":
			s.logger.Error("Invalid parameter value",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
				zap.String("message", apiErr.ErrorMessage()),
			)
			return fmt.Errorf("invalid parameter value: %s", apiErr.ErrorMessage())

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

// GetDBInstanceCount returns the total count of RDS instances
func (s *DiscoveryService) GetDBInstanceCount(ctx context.Context) (int, error) {
	s.logger.Info("Getting RDS instance count")

	input := &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int32(20), // We only need the first page to get the count
	}

	result, err := s.rds.DescribeDBInstances(ctx, input)
	if err != nil {
		s.logger.Error("Failed to get RDS instance count", zap.Error(err))
		return 0, s.handleAWSError(err, "DescribeDBInstances")
	}

	// Count instances in the first page
	count := len(result.DBInstances)

	// If there are more pages, we need to count all instances
	if result.Marker != nil {
		s.logger.Info("Multiple pages detected, counting all instances")
		
		var marker = result.Marker
		for {
			input := &rds.DescribeDBInstancesInput{
				Marker:     marker,
				MaxRecords: aws.Int32(100),
			}

			result, err := s.rds.DescribeDBInstances(ctx, input)
			if err != nil {
				s.logger.Error("Failed to get RDS instance count page", zap.Error(err))
				return count, s.handleAWSError(err, "DescribeDBInstances")
			}

			count += len(result.DBInstances)

			marker = result.Marker
			if marker == nil {
				break
			}
		}
	}

	s.logger.Info("RDS instance count retrieved", zap.Int("count", count))
	return count, nil
}

// DiscoverDBInstancesWithRetry discovers instances with retry logic for transient errors
func (s *DiscoveryService) DiscoverDBInstancesWithRetry(ctx context.Context, maxRetries int) ([]*models.RDSResource, error) {
	var instances []*models.RDSResource
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Info("Retrying RDS instance discovery",
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

		instances, lastErr = s.DiscoverAllDBInstances(ctx)
		if lastErr == nil {
			return instances, nil
		}

		// Check if error is retryable
		if !s.isRetryableError(lastErr) {
			break
		}

		s.logger.Warn("RDS instance discovery failed, will retry",
			zap.Int("attempt", attempt),
			zap.Error(lastErr),
		)
	}

	return nil, fmt.Errorf("failed to discover RDS instances after %d attempts: %w", maxRetries+1, lastErr)
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
