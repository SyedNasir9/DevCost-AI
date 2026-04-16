package ebs

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

// DiscoveryService handles EBS volume discovery
type DiscoveryService struct {
	client *aws.Client
	logger *logger.Logger
	ec2    *ec2.Client
}

// NewDiscoveryService creates a new EBS discovery service
func NewDiscoveryService(awsClient *aws.Client) *DiscoveryService {
	return &DiscoveryService{
		client: awsClient,
		logger: awsClient.logger,
		ec2:    awsClient.EC2,
	}
}

// DiscoverAllVolumes discovers all EBS volumes with pagination support
func (s *DiscoveryService) DiscoverAllVolumes(ctx context.Context) ([]*models.EBSResource, error) {
	s.logger.Info("Starting EBS volume discovery")

	var allVolumes []*models.EBSResource
	var nextToken *string

	page := 1
	for {
		s.logger.Debug("Fetching EBS volumes page",
			zap.Int("page", page),
			zap.String("next_token", func() string {
				if nextToken == nil {
					return "none"
				}
				return *nextToken
			}()),
		)

		// Describe volumes with pagination
		input := &ec2.DescribeVolumesInput{
			MaxResults: aws.Int32(1000), // Maximum allowed by AWS
		}

		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.ec2.DescribeVolumes(ctx, input)
		if err != nil {
			s.logger.Error("Failed to describe EBS volumes", zap.Error(err))
			return nil, s.handleAWSError(err, "DescribeVolumes")
		}

		// Process volumes from this page
		pageVolumes, err := s.processVolumePage(result.Volumes)
		if err != nil {
			return nil, fmt.Errorf("failed to process EBS volume page: %w", err)
		}

		allVolumes = append(allVolumes, pageVolumes...)

		s.logger.Info("Processed EBS volumes page",
			zap.Int("page", page),
			zap.Int("volumes_in_page", len(pageVolumes)),
			zap.Int("total_volumes", len(allVolumes)),
		)

		// Check for next page
		nextToken = result.NextToken
		if nextToken == nil {
			break
		}

		page++
	}

	s.logger.Info("EBS volume discovery completed",
		zap.Int("total_volumes", len(allVolumes)),
		zap.Int("pages_processed", page),
	)

	return allVolumes, nil
}

// DiscoverVolumesByFilter discovers EBS volumes with specific filters
func (s *DiscoveryService) DiscoverVolumesByFilter(ctx context.Context, filters []types.Filter) ([]*models.EBSResource, error) {
	s.logger.Info("Starting filtered EBS volume discovery",
		zap.Int("filter_count", len(filters)),
	)

	var allVolumes []*models.EBSResource
	var nextToken *string

	page := 1
	for {
		input := &ec2.DescribeVolumesInput{
			Filters:    filters,
			MaxResults: aws.Int32(1000),
		}

		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.ec2.DescribeVolumes(ctx, input)
		if err != nil {
			s.logger.Error("Failed to describe filtered EBS volumes", zap.Error(err))
			return nil, s.handleAWSError(err, "DescribeVolumes")
		}

		pageVolumes, err := s.processVolumePage(result.Volumes)
		if err != nil {
			return nil, fmt.Errorf("failed to process filtered EBS volume page: %w", err)
		}

		allVolumes = append(allVolumes, pageVolumes...)

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}

		page++
	}

	s.logger.Info("Filtered EBS volume discovery completed",
		zap.Int("total_volumes", len(allVolumes)),
	)

	return allVolumes, nil
}

// DiscoverAvailableVolumes discovers only available (not deleted) EBS volumes
func (s *DiscoveryService) DiscoverAvailableVolumes(ctx context.Context) ([]*models.EBSResource, error) {
	filters := []types.Filter{
		{
			Name:   aws.String("status"),
			Values: []string{"available", "in-use"},
		},
	}

	return s.DiscoverVolumesByFilter(ctx, filters)
}

// DiscoverUnattachedVolumes discovers only unattached EBS volumes
func (s *DiscoveryService) DiscoverUnattachedVolumes(ctx context.Context) ([]*models.EBSResource, error) {
	filters := []types.Filter{
		{
			Name:   aws.String("status"),
			Values: []string{"available"},
		},
	}

	return s.DiscoverVolumesByFilter(ctx, filters)
}

// DiscoverVolumeByID discovers a specific EBS volume by ID
func (s *DiscoveryService) DiscoverVolumeByID(ctx context.Context, volumeID string) (*models.EBSResource, error) {
	s.logger.Info("Discovering specific EBS volume",
		zap.String("volume_id", volumeID),
	)

	input := &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	}

	result, err := s.ec2.DescribeVolumes(ctx, input)
	if err != nil {
		s.logger.Error("Failed to describe EBS volume",
			zap.String("volume_id", volumeID),
			zap.Error(err),
		)
		return nil, s.handleAWSError(err, "DescribeVolumes")
	}

	if len(result.Volumes) == 0 {
		return nil, fmt.Errorf("EBS volume %s not found", volumeID)
	}

	volume := result.Volumes[0]
	ebsResource, err := s.mapVolumeToResource(volume)
	if err != nil {
		return nil, fmt.Errorf("failed to map EBS volume %s to resource: %w", volumeID, err)
	}

	s.logger.Info("Successfully discovered EBS volume",
		zap.String("volume_id", volumeID),
		zap.String("size", fmt.Sprintf("%d GB", ebsResource.Size)),
		zap.String("state", string(ebsResource.State)),
		zap.Bool("attached", ebsResource.IsAttached),
	)

	return ebsResource, nil
}

// processVolumePage processes a single page of EBS volumes
func (s *DiscoveryService) processVolumePage(volumes []types.Volume) ([]*models.EBSResource, error) {
	var ebsVolumes []*models.EBSResource

	for _, volume := range volumes {
		ebsResource, err := s.mapVolumeToResource(volume)
		if err != nil {
			s.logger.Error("Failed to map EBS volume to resource",
				zap.String("volume_id", aws.ToString(volume.VolumeId)),
				zap.Error(err),
			)
			continue // Skip this volume but continue processing others
		}

		ebsVolumes = append(ebsVolumes, ebsResource)
	}

	return ebsVolumes, nil
}

// mapVolumeToResource maps an AWS EBS volume to our internal model
func (s *DiscoveryService) mapVolumeToResource(volume types.Volume) (*models.EBSResource, error) {
	volumeID := aws.ToString(volume.VolumeId)
	size := aws.ToInt32(volume.Size)
	region := s.client.GetRegion()
	accountID := s.getAccountID(volume)

	// Create base EBS resource
	ebsResource := models.NewEBSResource(volumeID, size, region, accountID)

	// Map basic fields
	ebsResource.State = models.ResourceState(aws.ToString(volume.State))
	ebsResource.VolumeType = aws.ToString(volume.VolumeType)
	ebsResource.Iops = aws.ToInt32(volume.Iops)
	ebsResource.Throughput = aws.ToInt32(volume.Throughput)
	ebsResource.Encrypted = volume.Encrypted
	ebsResource.AvailabilityZone = aws.ToString(volume.AvailabilityZone)
	ebsResource.CreateTime = aws.ToTime(volume.CreateTime)

	// Map attachment information
	if len(volume.Attachments) > 0 {
		// Use the first attachment (volumes can only be attached to one instance at a time)
		attachment := volume.Attachments[0]
		ebsResource.IsAttached = true
		ebsResource.Attachment = models.Attachment{
			InstanceID: aws.ToString(attachment.InstanceId),
			Device:     aws.ToString(attachment.Device),
			State:      string(attachment.State),
			AttachTime: aws.ToTime(attachment.AttachTime),
		}
	} else {
		ebsResource.IsAttached = false
	}

	// Map snapshot information
	if volume.SnapshotId != nil {
		ebsResource.SnapshotID = aws.ToString(volume.SnapshotId)
	}

	// Map tags
	ebsResource.Tags = s.mapTags(volume.Tags)

	// Set name from tags or use volume ID
	if name, exists := ebsResource.GetTag("Name"); exists && name != "" {
		ebsResource.Name = name
	} else {
		ebsResource.Name = volumeID
	}

	s.logger.Debug("Mapped EBS volume to resource",
		zap.String("volume_id", volumeID),
		zap.String("name", ebsResource.Name),
		zap.Int32("size", ebsResource.Size),
		zap.String("state", string(ebsResource.State)),
		zap.Bool("attached", ebsResource.IsAttached),
		zap.Int("tag_count", len(ebsResource.Tags)),
	)

	return ebsResource, nil
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

// getAccountID extracts account ID from volume ARN or other sources
func (s *DiscoveryService) getAccountID(volume types.Volume) string {
	// Try to get account ID from volume ARN if available
	if volume.Arn != nil {
		arn := aws.ToString(volume.Arn)
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

		case "InvalidVolume.NotFound":
			s.logger.Error("EBS volume not found",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
				zap.String("message", apiErr.ErrorMessage()),
			)
			return fmt.Errorf("EBS volume not found: %s", apiErr.ErrorMessage())

		case "InvalidVolumeID.NotFound":
			s.logger.Error("EBS volume ID not found",
				zap.String("operation", operation),
				zap.String("error_code", apiErr.ErrorCode()),
				zap.String("message", apiErr.ErrorMessage()),
			)
			return fmt.Errorf("EBS volume not found: %s", apiErr.ErrorMessage())

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

// GetVolumeCount returns the total count of EBS volumes
func (s *DiscoveryService) GetVolumeCount(ctx context.Context) (int, error) {
	s.logger.Info("Getting EBS volume count")

	input := &ec2.DescribeVolumesInput{
		MaxResults: aws.Int32(5), // We only need the first page to get the count
	}

	result, err := s.ec2.DescribeVolumes(ctx, input)
	if err != nil {
		s.logger.Error("Failed to get EBS volume count", zap.Error(err))
		return 0, s.handleAWSError(err, "DescribeVolumes")
	}

	// Count volumes in the first page
	count := len(result.Volumes)

	// If there are more pages, we need to count all volumes
	if result.NextToken != nil {
		s.logger.Info("Multiple pages detected, counting all volumes")
		
		var nextToken = result.NextToken
		for {
			input := &ec2.DescribeVolumesInput{
				NextToken:  nextToken,
				MaxResults: aws.Int32(1000),
			}

			result, err := s.ec2.DescribeVolumes(ctx, input)
			if err != nil {
				s.logger.Error("Failed to get EBS volume count page", zap.Error(err))
				return count, s.handleAWSError(err, "DescribeVolumes")
			}

			count += len(result.Volumes)

			nextToken = result.NextToken
			if nextToken == nil {
				break
			}
		}
	}

	s.logger.Info("EBS volume count retrieved", zap.Int("count", count))
	return count, nil
}

// DiscoverVolumesWithRetry discovers volumes with retry logic for transient errors
func (s *DiscoveryService) DiscoverVolumesWithRetry(ctx context.Context, maxRetries int) ([]*models.EBSResource, error) {
	var volumes []*models.EBSResource
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			s.logger.Info("Retrying EBS volume discovery",
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

		volumes, lastErr = s.DiscoverAllVolumes(ctx)
		if lastErr == nil {
			return volumes, nil
		}

		// Check if error is retryable
		if !s.isRetryableError(lastErr) {
			break
		}

		s.logger.Warn("EBS volume discovery failed, will retry",
			zap.Int("attempt", attempt),
			zap.Error(lastErr),
		)
	}

	return nil, fmt.Errorf("failed to discover EBS volumes after %d attempts: %w", maxRetries+1, lastErr)
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

// GetVolumeStatistics provides statistics about EBS volumes
func (s *DiscoveryService) GetVolumeStatistics(ctx context.Context) (*VolumeStatistics, error) {
	s.logger.Info("Getting EBS volume statistics")

	volumes, err := s.DiscoverAllVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get volumes for statistics: %w", err)
	}

	stats := &VolumeStatistics{
		TotalVolumes:     len(volumes),
		AttachedVolumes:  0,
		UnattachedVolumes: 0,
		TotalSizeGB:      0,
		VolumesByType:   make(map[string]int),
		VolumesByAZ:     make(map[string]int),
		VolumesByState:  make(map[string]int),
		EncryptedVolumes: 0,
	}

	for _, volume := range volumes {
		// Attachment status
		if volume.IsAttached {
			stats.AttachedVolumes++
		} else {
			stats.UnattachedVolumes++
		}

		// Size
		stats.TotalSizeGB += int(volume.Size)

		// Volume type
		stats.VolumesByType[volume.VolumeType]++

		// Availability zone
		stats.VolumesByAZ[volume.AvailabilityZone]++

		// State
		stats.VolumesByAZ[string(volume.State)]++

		// Encryption
		if volume.Encrypted {
			stats.EncryptedVolumes++
		}
	}

	s.logger.Info("EBS volume statistics completed",
		zap.Int("total_volumes", stats.TotalVolumes),
		zap.Int("attached_volumes", stats.AttachedVolumes),
		zap.Int("unattached_volumes", stats.UnattachedVolumes),
		zap.Int("total_size_gb", stats.TotalSizeGB),
		zap.Int("encrypted_volumes", stats.EncryptedVolumes),
	)

	return stats, nil
}

// VolumeStatistics represents EBS volume statistics
type VolumeStatistics struct {
	TotalVolumes      int            `json:"total_volumes"`
	AttachedVolumes   int            `json:"attached_volumes"`
	UnattachedVolumes int            `json:"unattached_volumes"`
	TotalSizeGB       int            `json:"total_size_gb"`
	VolumesByType     map[string]int `json:"volumes_by_type"`
	VolumesByAZ       map[string]int `json:"volumes_by_az"`
	VolumesByState    map[string]int `json:"volumes_by_state"`
	EncryptedVolumes  int            `json:"encrypted_volumes"`
}
