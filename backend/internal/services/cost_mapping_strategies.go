package services

import (
	"strings"

	"go.uber.org/zap"

	"devcost-ai/internal/aws"
	"devcost-ai/internal/models"
	"devcost-ai/pkg/logger"
)

// ExactIDMatchingStrategy matches cost data to resources by exact resource ID
type ExactIDMatchingStrategy struct {
	logger *logger.Logger
}

// NewExactIDMatchingStrategy creates a new exact ID matching strategy
func NewExactIDMatchingStrategy(logger *logger.Logger) *ExactIDMatchingStrategy {
	return &ExactIDMatchingStrategy{
		logger: logger,
	}
}

// Name returns the strategy name
func (s *ExactIDMatchingStrategy) Name() string {
	return "ExactIDMatching"
}

// Priority returns the strategy priority (lower = higher priority)
func (s *ExactIDMatchingStrategy) Priority() int {
	return 1 // Highest priority
}

// Match attempts to match cost data to a resource by exact resource ID
func (s *ExactIDMatchingStrategy) Match(cost *aws.CostData, resources []*models.Resource) (*MatchResult, error) {
	if cost == nil {
		return nil, nil
	}
	
	// Check if cost data has a resource ID
	if cost.ResourceID == "" {
		s.logger.Debug("Cost data has no resource ID, skipping exact match",
			zap.String("service", cost.Service),
		)
		return nil, nil
	}
	
	s.logger.Debug("Attempting exact ID match",
		zap.String("cost_resource_id", cost.ResourceID),
		zap.String("service", cost.Service),
	)
	
	// Search for exact match in resources
	for _, resource := range resources {
		if resource.ResourceID == cost.ResourceID {
			s.logger.Debug("Exact ID match found",
				zap.String("resource_id", cost.ResourceID),
				zap.String("service", cost.Service),
			)
			
			return &MatchResult{
				Resource:     resource,
				Confidence:   1.0, // 100% confidence for exact match
				Strategy:     s.Name(),
				MatchType:    MatchTypeExact,
				MatchDetails: map[string]interface{}{
					"match_method": "exact_resource_id",
					"matched_field": "resource_id",
					"matched_value": cost.ResourceID,
				},
			}, nil
		}
	}
	
	// No exact match found
	s.logger.Debug("No exact ID match found",
		zap.String("cost_resource_id", cost.ResourceID),
	)
	
	return nil, nil
}

// TagBasedMatchingStrategy matches cost data to resources using tags
type TagBasedMatchingStrategy struct {
	logger *logger.Logger
	
	// Weight configuration for different tag types
	tagWeights map[string]float64
}

// NewTagBasedMatchingStrategy creates a new tag-based matching strategy
func NewTagBasedMatchingStrategy(logger *logger.Logger) *TagBasedMatchingStrategy {
	return &TagBasedMatchingStrategy{
		logger: logger,
		tagWeights: map[string]float64{
			"Name":          1.0,  // Exact name match is highly reliable
			"Environment":   0.3,  // Environment provides context
			"Project":       0.4,  // Project tag is moderately useful
			"Owner":         0.2,  // Owner provides weak signal
			"CostCenter":    0.5,  // Cost center is useful for allocation
			"Application":   0.6,  // Application tag is quite useful
			"Service":       0.5,  // Service tag is useful
			"aws:createdBy": 0.3,  // AWS created by tag
			"user:Name":     0.9,  // User-defined name is very reliable
		},
	}
}

// Name returns the strategy name
func (s *TagBasedMatchingStrategy) Name() string {
	return "TagBasedMatching"
}

// Priority returns the strategy priority
func (s *TagBasedMatchingStrategy) Priority() int {
	return 2
}

// Match attempts to match cost data to a resource using tags
func (s *TagBasedMatchingStrategy) Match(cost *aws.CostData, resources []*models.Resource) (*MatchResult, error) {
	if cost == nil || len(resources) == 0 {
		return nil, nil
	}
	
	// Extract tags from cost metadata
	costTags := s.extractTagsFromCost(cost)
	if len(costTags) == 0 {
		s.logger.Debug("No tags found in cost data, skipping tag-based match",
			zap.String("service", cost.Service),
		)
		return nil, nil
	}
	
	s.logger.Debug("Attempting tag-based match",
		zap.String("service", cost.Service),
		zap.Int("cost_tags_count", len(costTags)),
	)
	
	var bestMatch *models.Resource
	var bestScore float64
	var matchDetails map[string]interface{}
	var partialMatches []PartialMatch
	
	// Compare cost tags with each resource's tags
	for _, resource := range resources {
		if len(resource.Tags) == 0 {
			continue
		}
		
		score, matchedTags := s.calculateTagMatchScore(costTags, resource.Tags)
		
		if score > 0 {
			// Check region match for additional confidence
			if cost.Region != "" && resource.Region != "" {
				if strings.EqualFold(cost.Region, resource.Region) {
					score += 0.2 // Bonus for region match
				}
			}
			
			if score > bestScore {
				// Previous best becomes partial match
				if bestMatch != nil && bestScore >= 0.3 {
					partialMatches = append(partialMatches, PartialMatch{
						Resource:   bestMatch,
						Confidence: bestScore,
						Reason:     "tag_match_candidate",
					})
				}
				
				bestMatch = resource
				bestScore = score
				matchDetails = map[string]interface{}{
					"match_method":   "tag_based",
					"matched_tags":   matchedTags,
					"score":          score,
					"cost_tags":      costTags,
					"resource_tags":  resource.Tags,
				}
			} else if score >= 0.3 {
				// Store as partial match
				partialMatches = append(partialMatches, PartialMatch{
					Resource:   resource,
					Confidence: score,
					Reason:     "tag_match_candidate",
				})
			}
		}
	}
	
	// Determine match type based on score
	if bestMatch == nil {
		return nil, nil
	}
	
	matchType := s.determineMatchType(bestScore)
	
	s.logger.Debug("Tag-based match result",
		zap.String("resource_id", bestMatch.ResourceID),
		zap.Float64("confidence", bestScore),
		zap.String("match_type", matchType.String()),
	)
	
	return &MatchResult{
		Resource:       bestMatch,
		Confidence:     bestScore,
		Strategy:       s.Name(),
		MatchType:      matchType,
		MatchDetails:   matchDetails,
		PartialMatches: partialMatches,
	}, nil
}

// extractTagsFromCost extracts tag information from cost data
func (s *TagBasedMatchingStrategy) extractTagsFromCost(cost *aws.CostData) map[string]string {
	tags := make(map[string]string)
	
	// If cost data has direct tags, use them
	// Note: AWS Cost Explorer sometimes provides tags in the response
	
	// Extract from metadata if available
	// This is a placeholder - actual implementation would parse
	// the Cost Explorer response format
	
	return tags
}

// calculateTagMatchScore calculates a match score based on tag similarity
func (s *TagBasedMatchingStrategy) calculateTagMatchScore(
	costTags map[string]string,
	resourceTags map[string]string,
) (float64, map[string]string) {
	if len(costTags) == 0 || len(resourceTags) == 0 {
		return 0, nil
	}
	
	var totalScore float64
	var maxPossibleScore float64
	matchedTags := make(map[string]string)
	
	for costKey, costValue := range costTags {
		weight := s.getTagWeight(costKey)
		maxPossibleScore += weight
		
		// Check for exact match in resource tags
		if resourceValue, exists := resourceTags[costKey]; exists {
			if strings.EqualFold(costValue, resourceValue) {
				totalScore += weight
				matchedTags[costKey] = costValue
			} else {
				// Partial match - values are different
				similarity := calculateStringSimilarity(costValue, resourceValue)
				totalScore += weight * similarity * 0.5
			}
		}
	}
	
	// Normalize score
	if maxPossibleScore > 0 {
		normalizedScore := totalScore / maxPossibleScore
		return normalizedScore, matchedTags
	}
	
	return 0, nil
}

// getTagWeight returns the weight for a given tag key
func (s *TagBasedMatchingStrategy) getTagWeight(tagKey string) float64 {
	if weight, exists := s.tagWeights[tagKey]; exists {
		return weight
	}
	
	// Default weight for unknown tags
	return 0.1
}

// determineMatchType determines the match type based on confidence score
func (s *TagBasedMatchingStrategy) determineMatchType(score float64) MatchType {
	switch {
	case score >= 0.9:
		return MatchTypeHighConfidence
	case score >= 0.7:
		return MatchTypeMediumConfidence
	case score >= 0.5:
		return MatchTypeLowConfidence
	default:
		return MatchTypePartial
	}
}

// ServiceTypeMatchingStrategy matches cost data to resources by service type
type ServiceTypeMatchingStrategy struct {
	logger          *logger.Logger
	serviceMappings map[string]models.ResourceType
}

// NewServiceTypeMatchingStrategy creates a new service type matching strategy
func NewServiceTypeMatchingStrategy(
	logger *logger.Logger,
	serviceMappings map[string]models.ResourceType,
) *ServiceTypeMatchingStrategy {
	return &ServiceTypeMatchingStrategy{
		logger:          logger,
		serviceMappings: serviceMappings,
	}
}

// Name returns the strategy name
func (s *ServiceTypeMatchingStrategy) Name() string {
	return "ServiceTypeMatching"
}

// Priority returns the strategy priority
func (s *ServiceTypeMatchingStrategy) Priority() int {
	return 4 // Lower priority than exact and tag-based
}

// Match attempts to match cost data to a resource by service type
func (s *ServiceTypeMatchingStrategy) Match(cost *aws.CostData, resources []*models.Resource) (*MatchResult, error) {
	if cost == nil || cost.Service == "" {
		return nil, nil
	}
	
	// Get expected resource type for this service
	expectedType, known := s.serviceMappings[cost.Service]
	if !known {
		s.logger.Debug("Unknown service type, cannot match by service",
			zap.String("service", cost.Service),
		)
		return nil, nil
	}
	
	s.logger.Debug("Attempting service type match",
		zap.String("service", cost.Service),
		zap.String("expected_type", string(expectedType)),
	)
	
	// Filter resources by expected type
	var matchingResources []*models.Resource
	for _, resource := range resources {
		if resource.ResourceType == expectedType {
			matchingResources = append(matchingResources, resource)
		}
	}
	
	if len(matchingResources) == 0 {
		s.logger.Debug("No resources of expected type found",
			zap.String("service", cost.Service),
			zap.String("expected_type", string(expectedType)),
		)
		return nil, nil
	}
	
	// If only one resource matches, it's likely the right one
	if len(matchingResources) == 1 {
		resource := matchingResources[0]
		
		// Check for region match to increase confidence
		confidence := 0.6 // Base confidence for type-only match
		matchDetails := map[string]interface{}{
			"match_method":    "service_type_only",
			"service":         cost.Service,
			"expected_type":   expectedType,
			"unique_match":    true,
		}
		
		if cost.Region != "" && resource.Region != "" {
			if strings.EqualFold(cost.Region, resource.Region) {
				confidence = 0.8
				matchDetails["region_match"] = true
			}
		}
		
		s.logger.Debug("Service type match found (unique)",
			zap.String("resource_id", resource.ResourceID),
			zap.Float64("confidence", confidence),
		)
		
		return &MatchResult{
			Resource:     resource,
			Confidence:   confidence,
			Strategy:     s.Name(),
			MatchType:    s.determineMatchType(confidence),
			MatchDetails: matchDetails,
		}, nil
	}
	
	// Multiple resources match the type, need more criteria
	// Try to narrow down using region
	if cost.Region != "" {
		var regionMatches []*models.Resource
		for _, resource := range matchingResources {
			if strings.EqualFold(resource.Region, cost.Region) {
				regionMatches = append(regionMatches, resource)
			}
		}
		
		if len(regionMatches) == 1 {
			// Single resource matches both type and region
			resource := regionMatches[0]
			
			s.logger.Debug("Service type + region match found (unique)",
				zap.String("resource_id", resource.ResourceID),
				zap.Float64("confidence", 0.75),
			)
			
			return &MatchResult{
				Resource:   resource,
				Confidence: 0.75,
				Strategy:   s.Name(),
				MatchType:  MatchTypeMediumConfidence,
				MatchDetails: map[string]interface{}{
					"match_method":  "service_type_and_region",
					"service":       cost.Service,
					"region":        cost.Region,
					"unique_match":  true,
				},
			}, nil
		}
	}
	
	// Still multiple matches, return the first one with partial match info
	var partialMatches []PartialMatch
	for i, resource := range matchingResources {
		if i >= 5 {
			break // Only track top 5 partial matches
		}
		
		score := 0.5
		if cost.Region != "" && resource.Region != "" {
			if strings.EqualFold(cost.Region, resource.Region) {
				score = 0.65
			}
		}
		
		partialMatches = append(partialMatches, PartialMatch{
			Resource:   resource,
			Confidence: score,
			Reason:     "service_type_match",
		})
	}
	
	s.logger.Debug("Multiple service type matches, returning best candidates",
		zap.Int("candidate_count", len(partialMatches)),
	)
	
	return &MatchResult{
		Resource:       matchingResources[0],
		Confidence:     0.5,
		Strategy:       s.Name(),
		MatchType:      MatchTypePartial,
		MatchDetails: map[string]interface{}{
			"match_method":      "service_type_ambiguous",
			"service":           cost.Service,
			"matching_count":    len(matchingResources),
			"requires_review":   true,
		},
		PartialMatches: partialMatches,
	}, nil
}

// determineMatchType determines the match type based on confidence
func (s *ServiceTypeMatchingStrategy) determineMatchType(confidence float64) MatchType {
	switch {
	case confidence >= 0.8:
		return MatchTypeHighConfidence
	case confidence >= 0.6:
		return MatchTypeMediumConfidence
	default:
		return MatchTypeLowConfidence
	}
}
