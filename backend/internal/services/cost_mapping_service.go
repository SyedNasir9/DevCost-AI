package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/pkg/logger"
)

// CostMappingService handles mapping cost data to resources
type CostMappingService struct {
	logger *logger.Logger
	
	// Service mappings
	serviceMappings map[string]models.ResourceType
	
	// Confidence thresholds
	minConfidenceThreshold float64
	
	// Matching strategies priority
	strategyPriority []MatchingStrategy
}

// MatchingStrategy defines the interface for cost-to-resource matching strategies
type MatchingStrategy interface {
	Name() string
	Match(cost *aws.CostData, resources []*models.Resource) (*MatchResult, error)
	Priority() int
}

// MatchResult represents the result of a matching attempt
type MatchResult struct {
	Resource        *models.Resource
	Confidence      float64
	Strategy        string
	MatchType       MatchType
	MatchDetails    map[string]interface{}
	PartialMatches  []PartialMatch
}

// MatchType represents the type of match achieved
type MatchType int

const (
	MatchTypeNone MatchType = iota
	MatchTypeExact
	MatchTypeHighConfidence
	MatchTypeMediumConfidence
	MatchTypeLowConfidence
	MatchTypePartial
	MatchTypeUnknown
)

// String returns string representation of match type
func (m MatchType) String() string {
	switch m {
	case MatchTypeExact:
		return "exact"
	case MatchTypeHighConfidence:
		return "high_confidence"
	case MatchTypeMediumConfidence:
		return "medium_confidence"
	case MatchTypeLowConfidence:
		return "low_confidence"
	case MatchTypePartial:
		return "partial"
	case MatchTypeUnknown:
		return "unknown"
	default:
		return "none"
	}
}

// PartialMatch represents a partial match candidate
type PartialMatch struct {
	Resource   *models.Resource
	Confidence float64
	Reason     string
}

// EnrichedCostData represents cost data enriched with resource information
type EnrichedCostData struct {
	*aws.CostData
	
	// Resource identification
	MatchedResourceID   string                  `json:"matched_resource_id,omitempty"`
	MatchedResourceUUID string                  `json:"matched_resource_uuid,omitempty"`
	Resource            *models.Resource        `json:"resource,omitempty"`
	
	// Matching information
	MatchType           string                  `json:"match_type"`
	MatchConfidence     float64                 `json:"match_confidence"`
	MatchStrategy       string                  `json:"match_strategy"`
	MatchDetails        map[string]interface{}  `json:"match_details,omitempty"`
	
	// Alternative candidates
	PartialMatches      []PartialMatchInfo      `json:"partial_matches,omitempty"`
	
	// Status
	IsMatched           bool                    `json:"is_matched"`
	IsUnmatched         bool                    `json:"is_unmatched"`
	UnmatchReason       string                  `json:"unmatch_reason,omitempty"`
}

// PartialMatchInfo represents simplified partial match information for output
type PartialMatchInfo struct {
	ResourceID   string  `json:"resource_id"`
	ResourceType string  `json:"resource_type"`
	ResourceName string  `json:"resource_name,omitempty"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason"`
}

// MappingResult represents the overall mapping operation result
type MappingResult struct {
	EnrichedCosts   []*EnrichedCostData     `json:"enriched_costs"`
	Summary         *MappingSummary         `json:"summary"`
	UnmatchedCosts  []*UnmatchedCostInfo    `json:"unmatched_costs,omitempty"`
}

// MappingSummary provides statistics about the mapping operation
type MappingSummary struct {
	TotalCosts          int     `json:"total_costs"`
	MatchedCosts        int     `json:"matched_costs"`
	UnmatchedCosts      int     `json:"unmatched_costs"`
	PartialMatches      int     `json:"partial_matches"`
	
	MatchTypeBreakdown  map[string]int  `json:"match_type_breakdown"`
	StrategyBreakdown   map[string]int  `json:"strategy_breakdown"`
	
	AverageConfidence   float64 `json:"average_confidence"`
	HighConfidenceRate  float64 `json:"high_confidence_rate"`
}

// UnmatchedCostInfo provides details about unmatched costs
type UnmatchedCostInfo struct {
	*aws.CostData
	Reason          string   `json:"reason"`
	AttemptedStrategies []string `json:"attempted_strategies,omitempty"`
	Suggestions     []string `json:"suggestions,omitempty"`
}

// NewCostMappingService creates a new cost mapping service
func NewCostMappingService(logger *logger.Logger) *CostMappingService {
	service := &CostMappingService{
		logger:                 logger,
		serviceMappings:        createDefaultServiceMappings(),
		minConfidenceThreshold: 0.6,
	}
	
	// Initialize strategy priority
	service.strategyPriority = []MatchingStrategy{
		NewExactIDMatchingStrategy(logger),
		NewTagBasedMatchingStrategy(logger),
		NewHeuristicMatchingStrategy(logger, service.serviceMappings),
		NewServiceTypeMatchingStrategy(logger, service.serviceMappings),
	}
	
	return service
}

// MapCostToResources maps cost data to resources using multiple strategies
func (s *CostMappingService) MapCostToResources(
	ctx context.Context,
	costData []*aws.CostData,
	resources []*models.Resource,
) (*MappingResult, error) {
	s.logger.Info("Starting cost-to-resource mapping",
		zap.Int("cost_count", len(costData)),
		zap.Int("resource_count", len(resources)),
	)
	
	if len(costData) == 0 {
		return &MappingResult{
			EnrichedCosts: []*EnrichedCostData{},
			Summary:       &MappingSummary{TotalCosts: 0},
		}, nil
	}
	
	// Build resource indexes for efficient lookup
	resourceIndexes := s.buildResourceIndexes(resources)
	
	var enrichedCosts []*EnrichedCostData
	var unmatchedCosts []*UnmatchedCostInfo
	
	// Process each cost entry
	for _, cost := range costData {
		enriched := s.mapSingleCost(cost, resources, resourceIndexes)
		enrichedCosts = append(enrichedCosts, enriched)
		
		if enriched.IsUnmatched {
			unmatchedCosts = append(unmatchedCosts, &UnmatchedCostInfo{
				CostData:        cost,
				Reason:          enriched.UnmatchReason,
				AttemptedStrategies: s.getAttemptedStrategies(enriched),
			})
		}
	}
	
	// Generate summary
	summary := s.generateMappingSummary(enrichedCosts)
	
	s.logger.Info("Cost mapping completed",
		zap.Int("total", summary.TotalCosts),
		zap.Int("matched", summary.MatchedCosts),
		zap.Int("unmatched", summary.UnmatchedCosts),
		zap.Float64("average_confidence", summary.AverageConfidence),
	)
	
	return &MappingResult{
		EnrichedCosts:  enrichedCosts,
		Summary:        summary,
		UnmatchedCosts: unmatchedCosts,
	}, nil
}

// mapSingleCost attempts to map a single cost entry to a resource
func (s *CostMappingService) mapSingleCost(
	cost *aws.CostData,
	resources []*models.Resource,
	indexes *ResourceIndexes,
) *EnrichedCostData {
	enriched := &EnrichedCostData{
		CostData:    cost,
		MatchDetails: make(map[string]interface{}),
		IsMatched:   false,
		IsUnmatched: false,
	}
	
	// Try each strategy in priority order
	for _, strategy := range s.strategyPriority {
		result, err := strategy.Match(cost, resources)
		if err != nil {
			s.logger.Warn("Matching strategy failed",
				zap.String("strategy", strategy.Name()),
				zap.Error(err),
			)
			continue
		}
		
		if result != nil && result.Resource != nil {
			// Found a match
			enriched.MatchedResourceID = result.Resource.ResourceID
			enriched.MatchedResourceUUID = result.Resource.ID.String()
			enriched.Resource = result.Resource
			enriched.MatchConfidence = result.Confidence
			enriched.MatchStrategy = result.Strategy
			enriched.MatchType = result.MatchType.String()
			enriched.MatchDetails = result.MatchDetails
			enriched.IsMatched = result.Confidence >= s.minConfidenceThreshold
			enriched.IsUnmatched = !enriched.IsMatched
			
			// Store partial matches if any
			if len(result.PartialMatches) > 0 {
				enriched.PartialMatches = s.convertPartialMatches(result.PartialMatches)
			}
			
			if enriched.IsMatched {
				return enriched
			}
		}
	}
	
	// No match found or confidence too low
	enriched.IsMatched = false
	enriched.IsUnmatched = true
	enriched.UnmatchReason = s.determineUnmatchReason(cost, enriched)
	enriched.MatchType = MatchTypeUnknown.String()
	
	return enriched
}

// buildResourceIndexes creates indexes for efficient resource lookup
func (s *CostMappingService) buildResourceIndexes(resources []*models.Resource) *ResourceIndexes {
	indexes := &ResourceIndexes{
		ByID:           make(map[string]*models.Resource),
		ByResourceID:   make(map[string]*models.Resource),
		ByType:         make(map[models.ResourceType][]*models.Resource),
		ByRegion:       make(map[string][]*models.Resource),
		ByTag:          make(map[string]map[string][]*models.Resource),
	}
	
	for _, resource := range resources {
		// Index by ID
		indexes.ByID[resource.ID.String()] = resource
		
		// Index by resource ID
		if resource.ResourceID != "" {
			indexes.ByResourceID[resource.ResourceID] = resource
		}
		
		// Index by type
		indexes.ByType[resource.ResourceType] = append(indexes.ByType[resource.ResourceType], resource)
		
		// Index by region
		if resource.Region != "" {
			indexes.ByRegion[resource.Region] = append(indexes.ByRegion[resource.Region], resource)
		}
		
		// Index by tags
		for key, value := range resource.Tags {
			if indexes.ByTag[key] == nil {
				indexes.ByTag[key] = make(map[string][]*models.Resource)
			}
			indexes.ByTag[key][value] = append(indexes.ByTag[key][value], resource)
		}
	}
	
	return indexes
}

// generateMappingSummary creates a summary of the mapping operation
func (s *CostMappingService) generateMappingSummary(enriched []*EnrichedCostData) *MappingSummary {
	summary := &MappingSummary{
		TotalCosts:         len(enriched),
		MatchTypeBreakdown: make(map[string]int),
		StrategyBreakdown:  make(map[string]int),
	}
	
	var totalConfidence float64
	highConfidenceCount := 0
	
	for _, e := range enriched {
		if e.IsMatched {
			summary.MatchedCosts++
			totalConfidence += e.MatchConfidence
			
			if e.MatchConfidence >= 0.8 {
				highConfidenceCount++
			}
		} else {
			summary.UnmatchedCosts++
		}
		
		if len(e.PartialMatches) > 0 {
			summary.PartialMatches++
		}
		
		// Track match types
		summary.MatchTypeBreakdown[e.MatchType]++
		
		// Track strategies
		if e.MatchStrategy != "" {
			summary.StrategyBreakdown[e.MatchStrategy]++
		}
	}
	
	// Calculate averages
	if summary.MatchedCosts > 0 {
		summary.AverageConfidence = totalConfidence / float64(summary.MatchedCosts)
		summary.HighConfidenceRate = float64(highConfidenceCount) / float64(summary.MatchedCosts)
	}
	
	return summary
}

// determineUnmatchReason determines why a cost couldn't be matched
func (s *CostMappingService) determineUnmatchReason(cost *aws.CostData, enriched *EnrichedCostData) string {
	// Check if resource_id is present in cost data
	if cost.ResourceID != "" {
		return "resource_id_not_found_in_inventory"
	}
	
	// Check if service is known
	if cost.Service != "" {
		_, known := s.serviceMappings[cost.Service]
		if !known {
			return "unknown_service_type"
		}
	}
	
	// Check if we have partial matches
	if len(enriched.PartialMatches) > 0 {
		return "low_confidence_match_rejected"
	}
	
	return "no_matching_resource_found"
}

// convertPartialMatches converts internal partial matches to output format
func (s *CostMappingService) convertPartialMatches(matches []PartialMatch) []PartialMatchInfo {
	var result []PartialMatchInfo
	
	for _, match := range matches {
		if match.Resource == nil {
			continue
		}
		
		result = append(result, PartialMatchInfo{
			ResourceID:   match.Resource.ResourceID,
			ResourceType: string(match.Resource.ResourceType),
			ResourceName: match.Resource.Name,
			Confidence:   match.Confidence,
			Reason:       match.Reason,
		})
	}
	
	return result
}

// getAttemptedStrategies returns the list of strategies that were attempted
func (s *CostMappingService) getAttemptedStrategies(enriched *EnrichedCostData) []string {
	var strategies []string
	
	for _, strategy := range s.strategyPriority {
		strategies = append(strategies, strategy.Name())
	}
	
	return strategies
}

// createDefaultServiceMappings creates the default AWS service to resource type mappings
func createDefaultServiceMappings() map[string]models.ResourceType {
	return map[string]models.ResourceType{
		// Compute
		"Amazon EC2":              models.ResourceTypeEC2,
		"AWS Lambda":              models.ResourceTypeLambda,
		"Amazon ECS":              models.ResourceTypeECS,
		"Amazon EKS":              models.ResourceTypeEKS,
		"AWS Fargate":             models.ResourceTypeFargate,
		"AWS Batch":               models.ResourceTypeBatch,
		"AWS Elastic Beanstalk":   models.ResourceTypeElasticBeanstalk,
		
		// Storage
		"Amazon S3":               models.ResourceTypeS3,
		"Amazon EBS":              models.ResourceTypeEBS,
		"Amazon EFS":              models.ResourceTypeEFS,
		"Amazon FSx":              models.ResourceTypeFSx,
		"AWS Storage Gateway":     models.ResourceTypeStorageGateway,
		
		// Database
		"Amazon RDS":              models.ResourceTypeRDS,		"Amazon DynamoDB":         models.ResourceTypeDynamoDB,
		"Amazon ElastiCache":      models.ResourceTypeElastiCache,
		"Amazon Neptune":          models.ResourceTypeNeptune,
		"Amazon DocumentDB":       models.ResourceTypeDocumentDB,
		"Amazon Keyspaces":        models.ResourceTypeKeyspaces,
		"Amazon QLDB":             models.ResourceTypeQLDB,
		"Amazon Timestream":       models.ResourceTypeTimestream,
		
		// Networking
		"Amazon VPC":              models.ResourceTypeVPC,
		"Amazon CloudFront":       models.ResourceTypeCloudFront,
		"Amazon Route 53":         models.ResourceTypeRoute53,
		"AWS Direct Connect":      models.ResourceTypeDirectConnect,
		"AWS App Mesh":            models.ResourceTypeAppMesh,
		"AWS Cloud Map":           models.ResourceTypeCloudMap,
		"AWS Transit Gateway":     models.ResourceTypeTransitGateway,
		"AWS PrivateLink":         models.ResourceTypePrivateLink,
		"AWS AppSync":             models.ResourceTypeAppSync,
		
		// Security
		"AWS KMS":                 models.ResourceTypeKMS,
		"AWS WAF":                 models.ResourceTypeWAF,
		"AWS Shield":              models.ResourceTypeShield,
		"AWS Secrets Manager":     models.ResourceTypeSecretsManager,
		"AWS Certificate Manager": models.ResourceTypeACM,
		
		// Analytics
		"Amazon Athena":           models.ResourceTypeAthena,
		"Amazon EMR":              models.ResourceTypeEMR,
		"Amazon Redshift":         models.ResourceTypeRedshift,
		"AWS Glue":                models.ResourceTypeGlue,
		"Amazon Kinesis":          models.ResourceTypeKinesis,
		"Amazon MSK":              models.ResourceTypeMSK,
		"AWS Data Pipeline":       models.ResourceTypeDataPipeline,
		
		// Machine Learning
		"Amazon SageMaker":        models.ResourceTypeSageMaker,
		"Amazon Rekognition":      models.ResourceTypeRekognition,
		"Amazon Comprehend":       models.ResourceTypeComprehend,
		"Amazon Translate":        models.ResourceTypeTranslate,
		"Amazon Polly":            models.ResourceTypePolly,
		"Amazon Lex":              models.ResourceTypeLex,
		
		// Management
		"Amazon CloudWatch":       models.ResourceTypeCloudWatch,
		"AWS CloudFormation":      models.ResourceTypeCloudFormation,
		"AWS CloudTrail":          models.ResourceTypeCloudTrail,
		"AWS Config":              models.ResourceTypeConfig,
		"AWS Systems Manager":     models.ResourceTypeSystemsManager,
		"AWS Trusted Advisor":     models.ResourceTypeTrustedAdvisor,
		
		// Application Integration
		"Amazon SNS":              models.ResourceTypeSNS,
		"Amazon SQS":              models.ResourceTypeSQS,
		"Amazon EventBridge":      models.ResourceTypeEventBridge,
		"AWS Step Functions":      models.ResourceTypeStepFunctions,
		"Amazon MQ":               models.ResourceTypeMQ,
		"AWS AppFlow":             models.ResourceTypeAppFlow,
		
		// Developer Tools
		"AWS CodeCommit":          models.ResourceTypeCodeCommit,
		"AWS CodeBuild":           models.ResourceTypeCodeBuild,
		"AWS CodeDeploy":          models.ResourceTypeCodeDeploy,
		"AWS CodePipeline":        models.ResourceTypeCodePipeline,
		"AWS CodeArtifact":        models.ResourceTypeCodeArtifact,
		"AWS CodeStar":            models.ResourceTypeCodeStar,
		"AWS X-Ray":               models.ResourceTypeXRay,
		
		// Containers
		"Amazon ECR":              models.ResourceTypeECR,
		"AWS App2Container":       models.ResourceTypeApp2Container,
		
		// Other
		"Amazon SES":              models.ResourceTypeSES,
		"Amazon WorkSpaces":       models.ResourceTypeWorkSpaces,
		"Amazon Connect":          models.ResourceTypeConnect,
		"AWS IoT":                 models.ResourceTypeIoT,
		"AWS GameLift":            models.ResourceTypeGameLift,
		"Amazon Gamelift":         models.ResourceTypeGameLift,
	}
}

// ResourceIndexes provides indexed access to resources
type ResourceIndexes struct {
	ByID         map[string]*models.Resource
	ByResourceID map[string]*models.Resource
	ByType       map[models.ResourceType][]*models.Resource
	ByRegion     map[string][]*models.Resource
	ByTag        map[string]map[string][]*models.Resource
}

// SetConfidenceThreshold sets the minimum confidence threshold for matches
func (s *CostMappingService) SetConfidenceThreshold(threshold float64) {
	if threshold >= 0.0 && threshold <= 1.0 {
		s.minConfidenceThreshold = threshold
	}
}

// RegisterStrategy adds a new matching strategy to the service
func (s *CostMappingService) RegisterStrategy(strategy MatchingStrategy) {
	s.strategyPriority = append(s.strategyPriority, strategy)
	
	// Re-sort by priority
	for i := 0; i < len(s.strategyPriority)-1; i++ {
		for j := i + 1; j < len(s.strategyPriority); j++ {
			if s.strategyPriority[j].Priority() < s.strategyPriority[i].Priority() {
				s.strategyPriority[i], s.strategyPriority[j] = s.strategyPriority[j], s.strategyPriority[i]
			}
		}
	}
}

// GetServiceMapping returns the resource type for a given AWS service
func (s *CostMappingService) GetServiceMapping(service string) (models.ResourceType, bool) {
	resourceType, ok := s.serviceMappings[service]
	return resourceType, ok
}

// AddServiceMapping adds or updates a service to resource type mapping
func (s *CostMappingService) AddServiceMapping(service string, resourceType models.ResourceType) {
	s.serviceMappings[service] = resourceType
}

// Helper function to normalize service names
func normalizeServiceName(service string) string {
	// Remove extra whitespace
	service = strings.TrimSpace(service)
	
	// Handle common variations
	normalizations := map[string]string{
		"EC2": "Amazon EC2",
		"RDS": "Amazon RDS",
		"S3":  "Amazon S3",
		"Lambda": "AWS Lambda",
		"EBS": "Amazon EBS",
		"VPC": "Amazon VPC",
		"CloudWatch": "Amazon CloudWatch",
		"CloudFront": "Amazon CloudFront",
		"Route53": "Amazon Route 53",
		"DynamoDB": "Amazon DynamoDB",
	}
	
	if normalized, ok := normalizations[service]; ok {
		return normalized
	}
	
	return service
}

// Helper function to extract resource ID from usage type
func extractResourceIDFromUsageType(usageType string) string {
	// Common patterns:
	// - i-1234567890abcdef0 (EC2 instances)
	// - vol-1234567890abcdef0 (EBS volumes)
	// - db-1234567890abcdef0 (RDS instances)
	// - snap-1234567890abcdef0 (Snapshots)
	
	patterns := []string{
		`\b(i-[a-f0-9]{17})\b`,                    // EC2 instances
		`\b(vol-[a-f0-9]{17})\b`,                  // EBS volumes
		`\b(db-[a-f0-9]{17})\b`,                   // RDS instances
		`\b(snap-[a-f0-9]{17})\b`,                 // Snapshots
		`\b(ami-[a-f0-9]{17})\b`,                  // AMIs
		`\b(arn:aws:[^:]+:[^:]+:[^:]+:instance/[a-zA-Z0-9-]+)\b`, // ARN-based IDs
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(usageType)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

// Helper function to calculate similarity between two strings (0.0 to 1.0)
func calculateStringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	
	// Normalize strings
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))
	
	if s1 == s2 {
		return 1.0
	}
	
	// Simple contains check
	if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
		return 0.8
	}
	
	// TODO: Implement more sophisticated similarity (e.g., Levenshtein distance)
	return 0.0
}

// Helper function to match region codes
func matchRegion(region1, region2 string) bool {
	if region1 == "" || region2 == "" {
		return true // Empty regions match any
	}
	
	return strings.EqualFold(region1, region2)
}
