package services

import (
	"regexp"
	"strings"

	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/pkg/logger"
)

// HeuristicMatchingStrategy matches cost data to resources using various heuristics
type HeuristicMatchingStrategy struct {
	logger          *logger.Logger
	serviceMappings map[string]models.ResourceType
	
	// Pattern matchers for extracting resource IDs from usage types
	resourceIDPatterns map[models.ResourceType]*regexp.Regexp
}

// NewHeuristicMatchingStrategy creates a new heuristic matching strategy
func NewHeuristicMatchingStrategy(
	logger *logger.Logger,
	serviceMappings map[string]models.ResourceType,
) *HeuristicMatchingStrategy {
	strategy := &HeuristicMatchingStrategy{
		logger:          logger,
		serviceMappings: serviceMappings,
		resourceIDPatterns: make(map[models.ResourceType]*regexp.Regexp),
	}
	
	// Compile resource ID patterns
	strategy.compilePatterns()
	
	return strategy
}

// Name returns the strategy name
func (s *HeuristicMatchingStrategy) Name() string {
	return "HeuristicMatching"
}

// Priority returns the strategy priority
func (s *HeuristicMatchingStrategy) Priority() int {
	return 3 // Between tag-based and service type
}

// Match attempts to match cost data using various heuristics
func (s *HeuristicMatchingStrategy) Match(cost *aws.CostData, resources []*models.Resource) (*MatchResult, error) {
	if cost == nil || len(resources) == 0 {
		return nil, nil
	}
	
	s.logger.Debug("Attempting heuristic matching",
		zap.String("service", cost.Service),
		zap.String("usage_type", cost.UsageType),
		zap.Int("candidate_count", len(resources)),
	)
	
	// Try multiple heuristic approaches
	
	// 1. Extract resource ID from usage type and match
	if result := s.matchByUsageTypeExtraction(cost, resources); result != nil {
		return result, nil
	}
	
	// 2. Match by name similarity in tags
	if result := s.matchByNameSimilarity(cost, resources); result != nil {
		return result, nil
	}
	
	// 3. Match by resource type + region + account combination
	if result := s.matchByContextualCombination(cost, resources); result != nil {
		return result, nil
	}
	
	// 4. Match by ARN patterns if available
	if result := s.matchByARNPattern(cost, resources); result != nil {
		return result, nil
	}
	
	return nil, nil
}

// matchByUsageTypeExtraction attempts to extract resource ID from usage type
func (s *HeuristicMatchingStrategy) matchByUsageTypeExtraction(
	cost *aws.CostData,
	resources []*models.Resource,
) *MatchResult {
	if cost.UsageType == "" {
		return nil
	}
	
	// Determine expected resource type from service
	expectedType, known := s.serviceMappings[cost.Service]
	if !known {
		return nil
	}
	
	// Get the appropriate pattern for this resource type
	pattern, hasPattern := s.resourceIDPatterns[expectedType]
	if !hasPattern {
		return nil
	}
	
	// Try to extract resource ID from usage type
	matches := pattern.FindStringSubmatch(cost.UsageType)
	if len(matches) < 2 {
		return nil
	}
	
	extractedID := matches[1]
	
	s.logger.Debug("Extracted resource ID from usage type",
		zap.String("extracted_id", extractedID),
		zap.String("usage_type", cost.UsageType),
	)
	
	// Search for matching resource
	for _, resource := range resources {
		if resource.ResourceID == extractedID {
			confidence := 0.85 // High confidence for extracted ID match
			
			// Boost confidence if region also matches
			if cost.Region != "" && resource.Region != "" {
				if strings.EqualFold(cost.Region, resource.Region) {
					confidence = 0.95
				}
			}
			
			s.logger.Debug("Usage type extraction match found",
				zap.String("resource_id", resource.ResourceID),
				zap.Float64("confidence", confidence),
			)
			
			return &MatchResult{
				Resource:   resource,
				Confidence: confidence,
				Strategy:   s.Name(),
				MatchType:  s.determineMatchType(confidence),
				MatchDetails: map[string]interface{}{
					"match_method":   "usage_type_extraction",
					"extracted_id":   extractedID,
					"usage_type":     cost.UsageType,
					"expected_type":  expectedType,
				},
			}
		}
	}
	
	return nil
}

// matchByNameSimilarity attempts to match by name similarity
func (s *HeuristicMatchingStrategy) matchByNameSimilarity(
	cost *aws.CostData,
	resources []*models.Resource,
) *MatchResult {
	// Extract potential name from cost data
	costName := s.extractNameFromCost(cost)
	if costName == "" {
		return nil
	}
	
	var bestMatch *models.Resource
	var bestScore float64
	var partialMatches []PartialMatch
	
	for _, resource := range resources {
		if resource.Name == "" {
			continue
		}
		
		// Calculate name similarity
		similarity := calculateStringSimilarity(costName, resource.Name)
		
		// Check if names are related (e.g., contain similar substrings)
		if similarity > bestScore {
			// Store previous best as partial match
			if bestMatch != nil && bestScore > 0.4 {
				partialMatches = append(partialMatches, PartialMatch{
					Resource:   bestMatch,
					Confidence: bestScore,
					Reason:     "name_similarity_candidate",
				})
			}
			
			bestMatch = resource
			bestScore = similarity
		} else if similarity > 0.5 {
			partialMatches = append(partialMatches, PartialMatch{
				Resource:   resource,
					Confidence: similarity,
					Reason:     "name_similarity_candidate",
			})
		}
	}
	
	if bestMatch == nil || bestScore < 0.7 {
		return nil
	}
	
	s.logger.Debug("Name similarity match found",
		zap.String("cost_name", costName),
		zap.String("resource_name", bestMatch.Name),
		zap.Float64("similarity", bestScore),
	)
	
	confidence := bestScore
	matchType := s.determineMatchType(confidence)
	
	return &MatchResult{
		Resource:       bestMatch,
		Confidence:     confidence,
		Strategy:       s.Name(),
		MatchType:      matchType,
		MatchDetails: map[string]interface{}{
			"match_method":   "name_similarity",
			"cost_name":    costName,
			"similarity":   bestScore,
		},
		PartialMatches: partialMatches,
	}
}

// matchByContextualCombination matches by combining multiple context factors
func (s *HeuristicMatchingStrategy) matchByContextualCombination(
	cost *aws.CostData,
	resources []*models.Resource,
) *MatchResult {
	expectedType, known := s.serviceMappings[cost.Service]
	if !known {
		return nil
	}
	
	// Build scoring system
	type ScoredResource struct {
		Resource   *models.Resource
		Score      float64
		MatchCount int
	}
	
	var scoredResources []ScoredResource
	
	for _, resource := range resources {
		if resource.ResourceType != expectedType {
			continue
		}
		
		score := 0.0
		matches := 0
		
		// Region match (high weight)
		if cost.Region != "" && resource.Region != "" {
			if strings.EqualFold(cost.Region, resource.Region) {
				score += 0.4
				matches++
			}
		}
		
		// Account match (high weight)
		if cost.AccountID != "" && resource.AccountID != "" {
			if resource.AccountID == cost.AccountID {
				score += 0.3
				matches++
			}
		}
		
		// State check - prefer running/active resources
		if resource.State == "running" || resource.State == "available" || resource.State == "in-use" {
			score += 0.2
			matches++
		}
		
		// Time alignment - check if resource existed during cost period
		if !resource.CreatedAt.IsZero() {
			costTime := cost.Timestamp
			if costTime.After(resource.CreatedAt) {
				score += 0.1
				matches++
			}
		}
		
		if matches >= 2 {
			scoredResources = append(scoredResources, ScoredResource{
				Resource:   resource,
				Score:      score,
				MatchCount: matches,
			})
		}
	}
	
	if len(scoredResources) == 0 {
		return nil
	}
	
	// Find best match
	var best ScoredResource
	for _, scored := range scoredResources {
		if scored.Score > best.Score {
			best = scored
		}
	}
	
	if best.Resource == nil {
		return nil
	}
	
	confidence := best.Score
	if len(scoredResources) == 1 {
		confidence += 0.1 // Bonus for unique match
	}
	
	// Build partial matches list
	var partialMatches []PartialMatch
	for _, scored := range scoredResources {
		if scored.Resource.ResourceID != best.Resource.ResourceID && scored.Score >= 0.5 {
			partialMatches = append(partialMatches, PartialMatch{
				Resource:   scored.Resource,
				Confidence: scored.Score,
				Reason:     "contextual_match_candidate",
			})
		}
	}
	
	s.logger.Debug("Contextual combination match found",
		zap.String("resource_id", best.Resource.ResourceID),
		zap.Float64("confidence", confidence),
		zap.Int("match_count", best.MatchCount),
	)
	
	return &MatchResult{
		Resource:       best.Resource,
		Confidence:     confidence,
		Strategy:       s.Name(),
		MatchType:      s.determineMatchType(confidence),
		MatchDetails: map[string]interface{}{
			"match_method":     "contextual_combination",
			"match_count":      best.MatchCount,
			"candidate_count": len(scoredResources),
			"region_match":     cost.Region == best.Resource.Region,
			"account_match":    cost.AccountID == best.Resource.AccountID,
		},
		PartialMatches: partialMatches,
	}
}

// matchByARNPattern attempts to match by ARN patterns
func (s *HeuristicMatchingStrategy) matchByARNPattern(
	cost *aws.CostData,
	resources []*models.Resource,
) *MatchResult {
	// This would extract ARN from cost data if available
	// For now, this is a placeholder for future implementation
	
	// ARNs typically follow the pattern:
	// arn:aws:service:region:account-id:resource-type/resource-id
	
	return nil
}

// compilePatterns compiles regex patterns for resource ID extraction
func (s *HeuristicMatchingStrategy) compilePatterns() {
	// EC2 instance patterns
	s.resourceIDPatterns[models.ResourceTypeEC2] = regexp.MustCompile(
		`\b(i-[a-f0-9]{17})\b`,
	)
	
	// EBS volume patterns
	s.resourceIDPatterns[models.ResourceTypeEBS] = regexp.MustCompile(
		`\b(vol-[a-f0-9]{17})\b`,
	)
	
	// RDS instance patterns
	s.resourceIDPatterns[models.ResourceTypeRDS] = regexp.MustCompile(
		`\b(db-[a-zA-Z0-9]{26}|[a-zA-Z0-9-]+)\b`, // db- prefix or identifier
	)
	
	// S3 bucket patterns
	s.resourceIDPatterns[models.ResourceTypeS3] = regexp.MustCompile(
		`\b([a-z0-9][a-z0-9-]{1,61}[a-z0-9])\b`, // S3 bucket naming rules
	)
	
	// Lambda function patterns
	s.resourceIDPatterns[models.ResourceTypeLambda] = regexp.MustCompile(
		`\b([a-zA-Z0-9-_]{1,140})\b`, // Lambda function naming
	)
	
	// Snapshot patterns
	s.resourceIDPatterns["snapshot"] = regexp.MustCompile(
		`\b(snap-[a-f0-9]{17})\b`,
	)
	
	// AMI patterns
	s.resourceIDPatterns["ami"] = regexp.MustCompile(
		`\b(ami-[a-f0-9]{17})\b`,
	)
	
	// VPC patterns
	s.resourceIDPatterns[models.ResourceTypeVPC] = regexp.MustCompile(
		`\b(vpc-[a-f0-9]{17})\b`,
	)
	
	// Subnet patterns
	s.resourceIDPatterns[models.ResourceTypeSubnet] = regexp.MustCompile(
		`\b(subnet-[a-f0-9]{17})\b`,
	)
	
	// Security group patterns
	s.resourceIDPatterns["security_group"] = regexp.MustCompile(
		`\b(sg-[a-f0-9]{17})\b`,
	)
}

// extractNameFromCost extracts a potential name from cost data
func (s *HeuristicMatchingStrategy) extractNameFromCost(cost *aws.CostData) string {
	// Try usage type first
	if cost.UsageType != "" {
		// Remove common prefixes
		name := cost.UsageType
		prefixes := []string{
			"BoxUsage:",
			"SpotUsage:",
			"DedicatedUsage:",
			"ReservedInstanceUsage:",
			"Fargate-vCPU-Hours:",
			"Fargate-GB-Hours:",
		}
		
		for _, prefix := range prefixes {
			name = strings.TrimPrefix(name, prefix)
		}
		
		if name != cost.UsageType {
			return name
		}
	}
	
	// Try to extract from metadata if available
	// This is a placeholder for actual implementation
	
	return ""
}

// determineMatchType determines the match type based on confidence
func (s *HeuristicMatchingStrategy) determineMatchType(confidence float64) MatchType {
	switch {
	case confidence >= 0.9:
		return MatchTypeHighConfidence
	case confidence >= 0.7:
		return MatchTypeMediumConfidence
	case confidence >= 0.5:
		return MatchTypeLowConfidence
	default:
		return MatchTypePartial
	}
}

// GetResourceIDPatterns returns the compiled patterns for testing
func (s *HeuristicMatchingStrategy) GetResourceIDPatterns() map[models.ResourceType]*regexp.Regexp {
	return s.resourceIDPatterns
}

// AddResourceIDPattern adds a custom pattern for a resource type
func (s *HeuristicMatchingStrategy) AddResourceIDPattern(
	resourceType models.ResourceType,
	pattern string,
) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	
	s.resourceIDPatterns[resourceType] = re
	return nil
}
