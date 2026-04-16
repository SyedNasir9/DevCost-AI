package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"go.uber.org/zap"

	"devcost-ai/internal/config"
	"devcost-ai/pkg/logger"
)

// Client represents the AWS client with service-specific clients
type Client struct {
	config      aws.Config
	logger      *logger.Logger
	region      string
	credentials *Credentials

	// Service clients
	EC2           *ec2.Client
	RDS           *rds.Client
	EKS           *eks.Client
	CloudWatch    *cloudwatch.Client
	CostExplorer  *costexplorer.Client
}

// Credentials holds AWS credential information
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	SessionToken    string // Optional for temporary credentials
}

// Config holds AWS client configuration
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Profile         string // Optional AWS profile
	Endpoint        string // Optional custom endpoint (for testing)
	MaxRetries      int
	Timeout         time.Duration
}

// NewClient creates a new AWS client with the provided configuration
func NewClient(cfg *config.AWSConfig, log *logger.Logger) (*Client, error) {
	log.Info("Initializing AWS client",
		zap.String("region", cfg.Region),
		zap.String("access_key_id", maskAccessKey(cfg.AccessKeyID)),
	)

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		log.Error("Invalid AWS configuration", zap.Error(err))
		return nil, fmt.Errorf("invalid AWS configuration: %w", err)
	}

	// Create AWS configuration
	awsConfig, err := createAWSConfig(cfg, log)
	if err != nil {
		log.Error("Failed to create AWS configuration", zap.Error(err))
		return nil, fmt.Errorf("failed to create AWS configuration: %w", err)
	}

	// Create client instance
	client := &Client{
		config: *awsConfig,
		logger: log,
		region: cfg.Region,
		credentials: &Credentials{
			AccessKeyID:     cfg.AccessKeyID,
			SecretAccessKey: cfg.SecretAccessKey,
			Region:          cfg.Region,
			SessionToken:    cfg.SessionToken,
		},
	}

	// Initialize service clients
	if err := client.initializeServiceClients(); err != nil {
		log.Error("Failed to initialize service clients", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize service clients: %w", err)
	}

	// Test AWS connectivity
	if err := client.testConnectivity(context.Background()); err != nil {
		log.Error("AWS connectivity test failed", zap.Error(err))
		return nil, fmt.Errorf("AWS connectivity test failed: %w", err)
	}

	log.Info("AWS client initialized successfully",
		zap.String("region", cfg.Region),
	)

	return client, nil
}

// validateConfig validates the AWS configuration
func validateConfig(cfg *config.AWSConfig) error {
	if cfg.Region == "" {
		return fmt.Errorf("AWS region is required")
	}

	// Check if we have credentials (either environment variables or IAM role)
	if cfg.AccessKeyID == "" && cfg.SecretAccessKey == "" {
		// No explicit credentials, will use IAM role or instance profile
		return nil
	}

	if cfg.AccessKeyID == "" {
		return fmt.Errorf("AWS access key ID is required")
	}

	if cfg.SecretAccessKey == "" {
		return fmt.Errorf("AWS secret access key is required")
	}

	return nil
}

// createAWSConfig creates the AWS SDK configuration
func createAWSConfig(cfg *config.AWSConfig, log *logger.Logger) (*aws.Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load default configuration
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load default AWS config: %w", err)
	}

	// Override with explicit credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		log.Info("Using explicit AWS credentials")
		awsCfg.Credentials = aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretAccessKey,
				SessionToken:    cfg.SessionToken,
				Source:          "EnvironmentVariables",
			}, nil
		})
	} else {
		log.Info("Using default AWS credential chain (IAM role, instance profile, or environment)")
	}

	// Set custom endpoint if provided (useful for testing)
	if cfg.Endpoint != "" {
		log.Info("Using custom AWS endpoint", zap.String("endpoint", cfg.Endpoint))
		// Note: This would require custom endpoint resolvers for each service
	}

	return &awsCfg, nil
}

// initializeServiceClients initializes all AWS service clients
func (c *Client) initializeServiceClients() error {
	c.logger.Info("Initializing AWS service clients")

	// Initialize EC2 client
	c.EC2 = ec2.NewFromConfig(c.config)
	c.logger.Debug("EC2 client initialized")

	// Initialize RDS client
	c.RDS = rds.NewFromConfig(c.config)
	c.logger.Debug("RDS client initialized")

	// Initialize EKS client
	c.EKS = eks.NewFromConfig(c.config)
	c.logger.Debug("EKS client initialized")

	// Initialize CloudWatch client
	c.CloudWatch = cloudwatch.NewFromConfig(c.config)
	c.logger.Debug("CloudWatch client initialized")

	// Initialize Cost Explorer client
	c.CostExplorer = costexplorer.NewFromConfig(c.config)
	c.logger.Debug("Cost Explorer client initialized")

	return nil
}

// testConnectivity tests AWS connectivity by making a simple API call
func (c *Client) testConnectivity(ctx context.Context) error {
	c.logger.Info("Testing AWS connectivity")

	// Test with EC2 DescribeRegions (simple, low-privilege API call)
	_, err := c.EC2.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		RegionNames: []string{c.region},
	})
	if err != nil {
		return fmt.Errorf("failed to test AWS connectivity: %w", err)
	}

	c.logger.Info("AWS connectivity test passed")
	return nil
}

// GetRegion returns the configured AWS region
func (c *Client) GetRegion() string {
	return c.region
}

// GetCredentials returns the AWS credentials (masked for security)
func (c *Client) GetCredentials() *Credentials {
	return &Credentials{
		AccessKeyID:     maskAccessKey(c.credentials.AccessKeyID),
		SecretAccessKey: "***masked***",
		Region:          c.credentials.Region,
		SessionToken:    maskAccessKey(c.credentials.SessionToken),
	}
}

// GetConfig returns the AWS configuration
func (c *Client) GetConfig() aws.Config {
	return c.config
}

// Close performs cleanup operations
func (c *Client) Close() error {
	c.logger.Info("Closing AWS client")
	// AWS SDK v2 doesn't require explicit cleanup for most clients
	return nil
}

// Health performs a health check on AWS services
func (c *Client) Health(ctx context.Context) error {
	c.logger.Debug("Performing AWS health check")

	// Test connectivity to multiple services
	services := map[string]func(ctx context.Context) error{
		"EC2": func(ctx context.Context) error {
			_, err := c.EC2.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
				RegionNames: []string{c.region},
			})
			return err
		},
		"CloudWatch": func(ctx context.Context) error {
			_, err := c.CloudWatch.ListMetrics(ctx, &cloudwatch.ListMetricsInput{
				Namespace: aws.String("AWS/EC2"),
			})
			return err
		},
	}

	// Test each service
	for serviceName, healthCheck := range services {
		if err := healthCheck(ctx); err != nil {
			c.logger.Error("AWS service health check failed",
				zap.String("service", serviceName),
				zap.Error(err),
			)
			return fmt.Errorf("AWS %s health check failed: %w", serviceName, err)
		}
	}

	c.logger.Debug("All AWS services healthy")
	return nil
}

// maskAccessKey masks sensitive parts of access keys for logging
func maskAccessKey(key string) string {
	if len(key) <= 8 {
		return "***masked***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}

// NewClientFromEnvironment creates an AWS client from environment variables
func NewClientFromEnvironment(log *logger.Logger) (*Client, error) {
	cfg := &config.AWSConfig{
		Region:          log.GetConfig().AWS.Region,
		AccessKeyID:     log.GetConfig().AWS.AccessKeyID,
		SecretAccessKey: log.GetConfig().AWS.SecretAccessKey,
		SessionToken:    log.GetConfig().AWS.SessionToken,
	}

	return NewClient(cfg, log)
}
