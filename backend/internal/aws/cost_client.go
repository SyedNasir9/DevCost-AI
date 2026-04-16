package aws

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"go.uber.org/zap"

	"devcost-ai/pkg/logger"
)

// CostClient represents the AWS Cost Explorer client
type CostClient struct {
	client       *costexplorer.Client
	logger       *logger.Logger
	maxRetries   int
	rateLimiter  *RateLimiter
}

// CostData represents the structured cost data output
type CostData struct {
	ResourceID   string    `json:"resource_id"`
	Service      string    `json:"service"`
	CostAmount   float64   `json:"cost_amount"`
	Currency     string    `json:"currency"`
	Timestamp    time.Time `json:"timestamp"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	UsageType    string    `json:"usage_type,omitempty"`
	Region       string    `json:"region,omitempty"`
	AccountID    string    `json:"account_id,omitempty"`
}

// CostQueryOptions represents options for cost queries
type CostQueryOptions struct {
	TimeRange      TimeRange
	Granularity    types.Granularity
	GroupBy        []types.GroupDefinition
	Filter         *types.Expression
	Metrics        []string
	MaxResults     int32
	NextPageToken  *string
}

// TimeRange represents the time range for cost queries
type TimeRange struct {
	Start string
	End   string
}

// RateLimiter implements simple rate limiting for API calls
type RateLimiter struct {
	requestsPerSecond float64
	lastRequest       time.Time
	minInterval       time.Duration
}

// NewCostClient creates a new AWS Cost Explorer client
func NewCostClient(client *costexplorer.Client, logger *logger.Logger) *CostClient {
	return &CostClient{
		client:      client,
		logger:      logger,
		maxRetries:  3,
		rateLimiter: NewRateLimiter(10), // 10 requests per second
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64) *RateLimiter {
	return &RateLimiter{
		requestsPerSecond: requestsPerSecond,
		minInterval:       time.Duration(float64(time.Second) / requestsPerSecond),
		lastRequest:       time.Time{},
	}
}

// Wait waits if necessary to respect the rate limit
func (r *RateLimiter) Wait() {
	if !r.lastRequest.IsZero() {
		elapsed := time.Since(r.lastRequest)
		if elapsed < r.minInterval {
			time.Sleep(r.minInterval - elapsed)
		}
	}
	r.lastRequest = time.Now()
}

// GetLast24HoursCosts fetches cost data for the last 24 hours
func (c *CostClient) GetLast24HoursCosts(ctx context.Context) ([]*CostData, error) {
	c.logger.Info("Fetching cost data for last 24 hours")

	now := time.Now().UTC()
	startTime := now.Add(-24 * time.Hour)

	timeRange := TimeRange{
		Start: startTime.Format("2006-01-02"),
		End:   now.Format("2006-01-02"),
	}

	return c.GetCostsByTimeRange(ctx, timeRange)
}

// GetLast7DaysCosts fetches cost data for the last 7 days
func (c *CostClient) GetLast7DaysCosts(ctx context.Context) ([]*CostData, error) {
	c.logger.Info("Fetching cost data for last 7 days")

	now := time.Now().UTC()
	startTime := now.Add(-7 * 24 * time.Hour)

	timeRange := TimeRange{
		Start: startTime.Format("2006-01-02"),
		End:   now.Format("2006-01-02"),
	}

	return c.GetCostsByTimeRange(ctx, timeRange)
}

// GetCostsByTimeRange fetches cost data for a specific time range
func (c *CostClient) GetCostsByTimeRange(ctx context.Context, timeRange TimeRange) ([]*CostData, error) {
	c.logger.Info("Fetching cost data by time range",
		zap.String("start_date", timeRange.Start),
		zap.String("end_date", timeRange.End),
	)

	// Validate time range
	if err := c.validateTimeRange(timeRange); err != nil {
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	// Build query options with SERVICE and RESOURCE_ID grouping
	options := CostQueryOptions{
		TimeRange:   timeRange,
		Granularity: types.GranularityDaily,
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String("SERVICE"),
			},
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String("RESOURCE_ID"),
			},
		},
		Metrics: []string{"BlendedCost", "UsageQuantity"},
		MaxResults: 100,
	}

	return c.fetchCostsWithPagination(ctx, options)
}

// GetCostsByService fetches cost data grouped by service
func (c *CostClient) GetCostsByService(ctx context.Context, timeRange TimeRange) ([]*CostData, error) {
	c.logger.Info("Fetching cost data grouped by service",
		zap.String("start_date", timeRange.Start),
		zap.String("end_date", timeRange.End),
	)

	if err := c.validateTimeRange(timeRange); err != nil {
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	options := CostQueryOptions{
		TimeRange:   timeRange,
		Granularity: types.GranularityDaily,
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String("SERVICE"),
			},
		},
		Metrics: []string{"BlendedCost", "UsageQuantity"},
		MaxResults: 100,
	}

	return c.fetchCostsWithPagination(ctx, options)
}

// GetCostsByResourceID fetches cost data grouped by resource ID
func (c *CostClient) GetCostsByResourceID(ctx context.Context, timeRange TimeRange) ([]*CostData, error) {
	c.logger.Info("Fetching cost data grouped by resource ID",
		zap.String("start_date", timeRange.Start),
		zap.String("end_date", timeRange.End),
	)

	if err := c.validateTimeRange(timeRange); err != nil {
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	options := CostQueryOptions{
		TimeRange:   timeRange,
		Granularity: types.GranularityDaily,
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String("RESOURCE_ID"),
			},
		},
		Metrics: []string{"BlendedCost", "UsageQuantity"},
		MaxResults: 100,
	}

	return c.fetchCostsWithPagination(ctx, options)
}

// fetchCostsWithPagination fetches cost data with automatic pagination handling
func (c *CostClient) fetchCostsWithPagination(ctx context.Context, options CostQueryOptions) ([]*CostData, error) {
	var allCosts []*CostData
	var nextToken *string
	pageCount := 0

	for {
		pageCount++
		c.logger.Debug("Fetching cost data page",
			zap.Int("page", pageCount),
		)

		// Apply rate limiting
		c.rateLimiter.Wait()

		// Build the request
		input := &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(options.TimeRange.Start),
				End:   aws.String(options.TimeRange.End),
			},
			Granularity: options.Granularity,
			GroupBy:     options.GroupBy,
			Metrics:     options.Metrics,
			MaxResults:  aws.Int32(options.MaxResults),
		}

		if nextToken != nil {
			input.NextPageToken = nextToken
		}

		if options.Filter != nil {
			input.Filter = options.Filter
		}

		// Execute the request with retry logic
		result, err := c.executeWithRetry(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch cost data: %w", err)
		}

		// Process results
		costs := c.processCostResults(result, options.TimeRange)
		allCosts = append(allCosts, costs...)

		c.logger.Debug("Fetched cost data page",
			zap.Int("page", pageCount),
			zap.Int("records_in_page", len(costs)),
			zap.Int("total_records", len(allCosts)),
		)

		// Check for pagination
		if result.NextPageToken == nil || *result.NextPageToken == "" {
			break
		}

		nextToken = result.NextPageToken

		// Safety check for maximum pages (prevent infinite loops)
		if pageCount >= 1000 {
			c.logger.Warn("Reached maximum page count, stopping pagination",
				zap.Int("max_pages", 1000),
			)
			break
		}
	}

	c.logger.Info("Cost data fetch completed",
		zap.Int("total_pages", pageCount),
		zap.Int("total_records", len(allCosts)),
	)

	return allCosts, nil
}

// executeWithRetry executes the API call with retry logic
func (c *CostClient) executeWithRetry(ctx context.Context, input *costexplorer.GetCostAndUsageInput) (*costexplorer.GetCostAndUsageOutput, error) {
	var result *costexplorer.GetCostAndUsageOutput
	var err error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Warn("Retrying cost data fetch",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", c.maxRetries),
			)
			
			// Exponential backoff
			backoffDuration := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		result, err = c.client.GetCostAndUsage(ctx, input)
		if err == nil {
			return result, nil
		}

		// Check if error is retryable
		if !c.isRetryableError(err) {
			c.logger.Error("Non-retryable error occurred",
				zap.Error(err),
			)
			return nil, err
		}

		c.logger.Warn("Retryable error occurred",
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.maxRetries, err)
}

// processCostResults processes the API response into structured cost data
func (c *CostClient) processCostResults(result *costexplorer.GetCostAndUsageOutput, timeRange TimeRange) []*CostData {
	var costs []*CostData

	if result == nil || result.ResultsByTime == nil {
		return costs
	}

	for _, timeResult := range result.ResultsByTime {
		if timeResult.TimePeriod == nil || timeResult.Groups == nil {
			continue
		}

		periodStart := aws.ToString(timeResult.TimePeriod.Start)
		periodEnd := aws.ToString(timeResult.TimePeriod.End)

		for _, group := range timeResult.Groups {
			if group.Keys == nil || group.Metrics == nil {
				continue
			}

			costData := &CostData{
				StartDate: periodStart,
				EndDate:   periodEnd,
				Currency:  "USD", // Default currency
			}

			// Extract group keys (SERVICE, RESOURCE_ID, etc.)
			for i, key := range group.Keys {
				if i == 0 {
					costData.Service = key
				} else if i == 1 {
					costData.ResourceID = key
				}
			}

			// Extract metrics
			if blendedCost, ok := group.Metrics["BlendedCost"]; ok {
				if blendedCost.Amount != nil {
					amount, err := strconv.ParseFloat(aws.ToString(blendedCost.Amount), 64)
					if err == nil {
						costData.CostAmount = amount
					}
				}
				if blendedCost.Unit != nil {
					costData.Currency = aws.ToString(blendedCost.Unit)
				}
			}

			if usageQuantity, ok := group.Metrics["UsageQuantity"]; ok {
				if usageQuantity.Amount != nil {
					// Usage quantity stored in metadata for reference
					if costData.UsageType == "" {
						costData.UsageType = aws.ToString(usageQuantity.Amount)
					}
				}
			}

			// Parse timestamp from period start
			if timestamp, err := time.Parse("2006-01-02", periodStart); err == nil {
				costData.Timestamp = timestamp
			}

			// Only include records with valid cost amounts
			if costData.CostAmount > 0 || costData.Service != "" {
				costs = append(costs, costData)
			}
		}
	}

	return costs
}

// isRetryableError checks if an error is retryable
func (c *CostClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	
	// AWS Cost Explorer specific retryable errors
	retryableErrors := []string{
		"RequestTimeout",
		"Throttling",
		"Rate exceeded",
		"InternalServerError",
		"ServiceUnavailable",
		"connection reset",
		"connection refused",
		"timeout",
		"Temporary",
	}

	for _, retryable := range retryableErrors {
		if containsString(errStr, retryable) {
			return true
		}
	}

	return false
}

// validateTimeRange validates the time range for cost queries
func (c *CostClient) validateTimeRange(timeRange TimeRange) error {
	if timeRange.Start == "" || timeRange.End == "" {
		return fmt.Errorf("start and end dates are required")
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", timeRange.Start)
	if err != nil {
		return fmt.Errorf("invalid start date format (expected YYYY-MM-DD): %w", err)
	}

	endDate, err := time.Parse("2006-01-02", timeRange.End)
	if err != nil {
		return fmt.Errorf("invalid end date format (expected YYYY-MM-DD): %w", err)
	}

	// Validate date range
	if startDate.After(endDate) {
		return fmt.Errorf("start date cannot be after end date")
	}

	// AWS Cost Explorer limitation: maximum 12 months
	maxRange := 365 * 24 * time.Hour // 1 year
	if endDate.Sub(startDate) > maxRange {
		return fmt.Errorf("time range cannot exceed 1 year")
	}

	// AWS Cost Explorer limitation: cannot query future dates
	now := time.Now().UTC()
	if endDate.After(now.Add(24 * time.Hour)) {
		return fmt.Errorf("end date cannot be in the future")
	}

	return nil
}

// GetCostSummary provides a summary of costs by service
func (c *CostClient) GetCostSummary(ctx context.Context, timeRange TimeRange) (map[string]float64, error) {
	c.logger.Info("Fetching cost summary",
		zap.String("start_date", timeRange.Start),
		zap.String("end_date", timeRange.End),
	)

	costs, err := c.GetCostsByService(ctx, timeRange)
	if err != nil {
		return nil, err
	}

	summary := make(map[string]float64)
	for _, cost := range costs {
		summary[cost.Service] += cost.CostAmount
	}

	return summary, nil
}

// GetResourceCostBreakdown provides cost breakdown by resource ID
func (c *CostClient) GetResourceCostBreakdown(ctx context.Context, timeRange TimeRange) (map[string]*ResourceCostInfo, error) {
	c.logger.Info("Fetching resource cost breakdown",
		zap.String("start_date", timeRange.Start),
		zap.String("end_date", timeRange.End),
	)

	costs, err := c.GetCostsByResourceID(ctx, timeRange)
	if err != nil {
		return nil, err
	}

	breakdown := make(map[string]*ResourceCostInfo)
	for _, cost := range costs {
		if _, exists := breakdown[cost.ResourceID]; !exists {
			breakdown[cost.ResourceID] = &ResourceCostInfo{
				ResourceID: cost.ResourceID,
				Service:    cost.Service,
				Currency:   cost.Currency,
			}
		}
		breakdown[cost.ResourceID].TotalCost += cost.CostAmount
		breakdown[cost.ResourceID].CostEntries++
	}

	return breakdown, nil
}

// ResourceCostInfo represents cost information for a specific resource
type ResourceCostInfo struct {
	ResourceID   string  `json:"resource_id"`
	Service      string  `json:"service"`
	TotalCost    float64 `json:"total_cost"`
	Currency     string  `json:"currency"`
	CostEntries  int     `json:"cost_entries"`
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

// containsSubstring checks if s contains substr (case-insensitive)
func containsSubstring(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
