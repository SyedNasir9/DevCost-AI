package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"go.uber.org/zap"
)

// EC2Service provides EC2-specific operations
type EC2Service struct {
	client *Client
	logger *logger.Logger
}

// NewEC2Service creates a new EC2 service
func NewEC2Service(client *Client) *EC2Service {
	return &EC2Service{
		client: client,
		logger: client.logger,
	}
}

// ListInstances lists all EC2 instances in the region
func (s *EC2Service) ListInstances(ctx context.Context) ([]types.Instance, error) {
	s.logger.Info("Listing EC2 instances")

	var instances []types.Instance
	result, err := s.client.EC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		s.logger.Error("Failed to describe EC2 instances", zap.Error(err))
		return nil, fmt.Errorf("failed to describe EC2 instances: %w", err)
	}

	for _, reservation := range result.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	s.logger.Info("Retrieved EC2 instances", zap.Int("count", len(instances)))
	return instances, nil
}

// GetInstanceMetrics retrieves CloudWatch metrics for an EC2 instance
func (s *EC2Service) GetInstanceMetrics(ctx context.Context, instanceID string, startTime, endTime time.Time) ([]types.Datapoint, error) {
	s.logger.Info("Getting EC2 instance metrics",
		zap.String("instance_id", instanceID),
		zap.Time("start_time", startTime),
		zap.Time("end_time", endTime),
	)

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("CPUUtilization"),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(instanceID),
			},
		},
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		Period:    aws.Int32(300), // 5-minute intervals
		Statistics: []types.Statistic{types.StatisticAverage},
	}

	result, err := s.client.CloudWatch.GetMetricStatistics(ctx, input)
	if err != nil {
		s.logger.Error("Failed to get EC2 metrics", zap.Error(err))
		return nil, fmt.Errorf("failed to get EC2 metrics: %w", err)
	}

	s.logger.Info("Retrieved EC2 metrics", zap.Int("datapoints", len(result.Datapoints)))
	return result.Datapoints, nil
}

// RDSService provides RDS-specific operations
type RDSService struct {
	client *Client
	logger *logger.Logger
}

// NewRDSService creates a new RDS service
func NewRDSService(client *Client) *RDSService {
	return &RDSService{
		client: client,
		logger: client.logger,
	}
}

// ListDBInstances lists all RDS instances in the region
func (s *RDSService) ListDBInstances(ctx context.Context) ([]types.DBInstance, error) {
	s.logger.Info("Listing RDS instances")

	var instances []types.DBInstance
	result, err := s.client.RDS.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		s.logger.Error("Failed to describe RDS instances", zap.Error(err))
		return nil, fmt.Errorf("failed to describe RDS instances: %w", err)
	}

	instances = result.DBInstances

	s.logger.Info("Retrieved RDS instances", zap.Int("count", len(instances)))
	return instances, nil
}

// GetDBInstanceMetrics retrieves CloudWatch metrics for an RDS instance
func (s *RDSService) GetDBInstanceMetrics(ctx context.Context, dbInstanceID string, startTime, endTime time.Time) (map[string][]types.Datapoint, error) {
	s.logger.Info("Getting RDS instance metrics",
		zap.String("db_instance_id", dbInstanceID),
		zap.Time("start_time", startTime),
		zap.Time("end_time", endTime),
	)

	metrics := map[string][]types.Datapoint{}
	metricNames := []string{"CPUUtilization", "DatabaseConnections", "FreeStorageSpace"}

	for _, metricName := range metricNames {
		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/RDS"),
			MetricName: aws.String(metricName),
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("DBInstanceIdentifier"),
					Value: aws.String(dbInstanceID),
				},
			},
			StartTime: aws.Time(startTime),
			EndTime:   aws.Time(endTime),
			Period:    aws.Int32(300), // 5-minute intervals
			Statistics: []types.Statistic{types.StatisticAverage},
		}

		result, err := s.client.CloudWatch.GetMetricStatistics(ctx, input)
		if err != nil {
			s.logger.Error("Failed to get RDS metrics",
				zap.String("metric", metricName),
				zap.Error(err),
			)
			continue
		}

		metrics[metricName] = result.Datapoints
	}

	s.logger.Info("Retrieved RDS metrics", zap.Int("metric_count", len(metrics)))
	return metrics, nil
}

// CostService provides cost analysis operations
type CostService struct {
	client *Client
	logger *logger.Logger
}

// NewCostService creates a new Cost service
func NewCostService(client *Client) *CostService {
	return &CostService{
		client: client,
		logger: client.logger,
	}
}

// GetCostAndUsage retrieves cost and usage data for the specified time period
func (s *CostService) GetCostAndUsage(ctx context.Context, start, end time.Time, granularity types.Granularity) (*costexplorer.GetCostAndUsageOutput, error) {
	s.logger.Info("Getting cost and usage data",
		zap.Time("start", start),
		zap.Time("end", end),
		zap.String("granularity", string(granularity)),
	)

	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(start.Format("2006-01-02")),
			End:   aws.String(end.Format("2006-01-02")),
		},
		Granularity: granularity,
		Metrics: []string{"BlendedCost", "UnblendedCost", "UsageQuantity"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String("SERVICE"),
			},
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String("REGION"),
			},
		},
	}

	result, err := s.client.CostExplorer.GetCostAndUsage(ctx, input)
	if err != nil {
		s.logger.Error("Failed to get cost and usage data", zap.Error(err))
		return nil, fmt.Errorf("failed to get cost and usage data: %w", err)
	}

	s.logger.Info("Retrieved cost and usage data",
		zap.Int("result_count", len(result.ResultsByTime)),
	)

	return result, nil
}

// GetServiceCosts retrieves costs broken down by service
func (s *CostService) GetServiceCosts(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	s.logger.Info("Getting service costs")

	result, err := s.GetCostAndUsage(ctx, start, end, types.GranularityMonthly)
	if err != nil {
		return nil, err
	}

	serviceCosts := make(map[string]float64)

	for _, timePeriod := range result.ResultsByTime {
		for _, group := range timePeriod.Groups {
			// Extract service name from dimensions
			var serviceName string
			for _, dimension := range group.Keys {
				if aws.ToString(dimension.Key) == "SERVICE" {
					serviceName = aws.ToString(dimension.Value)
					break
				}
			}

			// Extract cost amount
			if blendedCost, ok := group.Metrics["BlendedCost"]; ok && blendedCost.Amount != nil {
				cost := aws.ToFloat64(blendedCost.Amount)
				serviceCosts[serviceName] += cost
			}
		}
	}

	s.logger.Info("Retrieved service costs", zap.Int("service_count", len(serviceCosts)))
	return serviceCosts, nil
}

// Resource represents a generic AWS resource
type Resource struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	Region       string            `json:"region"`
	State        string            `json:"state"`
	Tags         map[string]string `json:"tags"`
	CostLastMonth float64           `json:"cost_last_month"`
	LastUpdated  time.Time         `json:"last_updated"`
}

// ResourceCollector collects resources from multiple AWS services
type ResourceCollector struct {
	client      *Client
	logger      *logger.Logger
	ec2Service  *EC2Service
	rdsService  *RDSService
	costService *CostService
}

// NewResourceCollector creates a new resource collector
func NewResourceCollector(client *Client) *ResourceCollector {
	return &ResourceCollector{
		client:      client,
		logger:      client.logger,
		ec2Service:  NewEC2Service(client),
		rdsService:  NewRDSService(client),
		costService: NewCostService(client),
	}
}

// CollectResources collects all resources from configured AWS services
func (c *ResourceCollector) CollectResources(ctx context.Context) ([]Resource, error) {
	c.logger.Info("Collecting AWS resources")

	var resources []Resource

	// Collect EC2 instances
	ec2Instances, err := c.ec2Service.ListInstances(ctx)
	if err != nil {
		c.logger.Error("Failed to collect EC2 instances", zap.Error(err))
	} else {
		for _, instance := range ec2Instances {
			resource := Resource{
				ID:     aws.ToString(instance.InstanceId),
				Type:   "EC2",
				Name:   getInstanceName(instance.Tags),
				Region: c.client.GetRegion(),
				State:  string(instance.State.Name),
				Tags:   tagsToMap(instance.Tags),
			}
			resources = append(resources, resource)
		}
	}

	// Collect RDS instances
	rdsInstances, err := c.rdsService.ListDBInstances(ctx)
	if err != nil {
		c.logger.Error("Failed to collect RDS instances", zap.Error(err))
	} else {
		for _, instance := range rdsInstances {
			resource := Resource{
				ID:     aws.ToString(instance.DBInstanceIdentifier),
				Type:   "RDS",
				Name:   aws.ToString(instance.DBInstanceIdentifier),
				Region: c.client.GetRegion(),
				State:  string(instance.DBInstanceStatus),
				Tags:   tagsToMap(instance.Tags),
			}
			resources = append(resources, resource)
		}
	}

	c.logger.Info("Collected AWS resources", zap.Int("total_count", len(resources)))
	return resources, nil
}

// Helper functions

func getInstanceName(tags []types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func tagsToMap(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}
