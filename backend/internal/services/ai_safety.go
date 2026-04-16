package services

import (
	"fmt"
	"regexp"
	"strings"
)

// AISafetyValidator validates AI outputs before returning to users
type AISafetyValidator struct {
	blockedPatterns []*regexp.Regexp
	blockedKeywords []string
	maxLength       int
}

// NewAISafetyValidator creates a new safety validator
func NewAISafetyValidator() *AISafetyValidator {
	return &AISafetyValidator{
		blockedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(delete|terminate|stop|remove|destroy)\s+(this|the|all)\s+`),
			regexp.MustCompile(`(?i)run\s+(this\s+)?command`),
			regexp.MustCompile(`(?i)execute\s+(the\s+)?following`),
			regexp.MustCompile(`(?i)aws\s+(ec2|rds|s3)\s+(delete|terminate|stop)`),
			regexp.MustCompile(`(?i)click\s+(the\s+)?button`),
			regexp.MustCompile(`(?i)(sudo|rm\s+-rf|DROP\s+TABLE)`),
		},
		blockedKeywords: []string{
			"execute now",
			"run immediately",
			"delete immediately",
			"terminate now",
			"click here",
			"follow these steps to delete",
		},
		maxLength: 2000,
	}
}

// ValidateOutput checks AI output for safety issues
func (v *AISafetyValidator) ValidateOutput(output string) ValidationResult {
	result := ValidationResult{
		IsValid:   true,
		Original:  output,
		Sanitized: output,
	}

	// Check length
	if len(output) > v.maxLength {
		result.Sanitized = output[:v.maxLength] + "..."
		result.Warnings = append(result.Warnings, "Output truncated due to length")
	}

	// Check for blocked patterns
	for _, pattern := range v.blockedPatterns {
		if pattern.MatchString(output) {
			result.IsValid = false
			result.Violations = append(result.Violations, "Contains action directive")
			result.Sanitized = pattern.ReplaceAllString(result.Sanitized, "[action removed]")
		}
	}

	// Check for blocked keywords
	lowerOutput := strings.ToLower(output)
	for _, keyword := range v.blockedKeywords {
		if strings.Contains(lowerOutput, keyword) {
			result.IsValid = false
			result.Violations = append(result.Violations, "Contains blocked keyword: "+keyword)
			result.Sanitized = strings.ReplaceAll(
				result.Sanitized,
				keyword,
				"[removed]",
			)
		}
	}

	// Check for potential hallucinated resource IDs
	if containsHallucinatedID(output) {
		result.Warnings = append(result.Warnings, "May contain generated resource identifiers")
	}

	return result
}

// ValidationResult holds the result of safety validation
type ValidationResult struct {
	IsValid    bool
	Original   string
	Sanitized  string
	Violations []string
	Warnings   []string
}

// SanitizeForSlack prepares AI output for Slack
func (v *AISafetyValidator) SanitizeForSlack(output string) string {
	result := v.ValidateOutput(output)

	// Remove any markdown that Slack doesn't support
	sanitized := result.Sanitized
	sanitized = strings.ReplaceAll(sanitized, "```", "`")

	// Limit length for Slack
	if len(sanitized) > 500 {
		sanitized = sanitized[:497] + "..."
	}

	return sanitized
}

// containsHallucinatedID checks for potentially fake resource IDs
func containsHallucinatedID(text string) bool {
	// Common patterns for AWS resource IDs that might be hallucinated
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`i-[a-f0-9]{17}`),                   // EC2 instance ID
		regexp.MustCompile(`vol-[a-f0-9]{17}`),                 // EBS volume ID
		regexp.MustCompile(`snap-[a-f0-9]{17}`),                // Snapshot ID
		regexp.MustCompile(`arn:aws:[a-z]+:[a-z0-9-]+:[0-9]+`), // ARN
	}

	for _, p := range patterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}

// AIResponsePolicy defines what AI can and cannot do
type AIResponsePolicy struct {
	AllowExplanations     bool
	AllowSuggestions      bool
	AllowActionDirectives bool // Should always be false
	RequireDisclaimer     bool
}

// DefaultPolicy returns the default safe policy
func DefaultPolicy() AIResponsePolicy {
	return AIResponsePolicy{
		AllowExplanations:     true,
		AllowSuggestions:      true,
		AllowActionDirectives: false, // Never allow AI to direct actions
		RequireDisclaimer:     true,
	}
}

// ApplyPolicy applies safety policy to AI output
func ApplyPolicy(output string, policy AIResponsePolicy) string {
	if !policy.AllowActionDirectives {
		validator := NewAISafetyValidator()
		result := validator.ValidateOutput(output)
		output = result.Sanitized
	}

	if policy.RequireDisclaimer && len(output) > 0 {
		output = output + "\n\n_AI-generated analysis. Verify before taking action._"
	}

	return output
}

// EnhancedRecommendation adds AI explanation to a recommendation
type EnhancedRecommendation struct {
	ResourceID       string  `json:"resource_id"`
	ResourceType     string  `json:"resource_type"`
	Title            string  `json:"title"`
	EstimatedSavings float64 `json:"estimated_savings"`

	// AI-generated fields
	Explanation      string `json:"explanation"`
	SavingsRationale string `json:"savings_rationale"`
	ImpactSummary    string `json:"impact_summary"`
	Confidence       string `json:"confidence"` // high, medium, low
}

// GenerateExplanation creates an explanation for a recommendation
func GenerateExplanation(ai *AIService, rec RecommendationInput, metrics map[string]float64) EnhancedRecommendation {
	enhanced := EnhancedRecommendation{
		ResourceID:       rec.ResourceID,
		ResourceType:     rec.ResourceType,
		Title:            rec.Title,
		EstimatedSavings: rec.EstimatedSavings,
		Confidence:       "high",
	}

	// Try static template first (faster, deterministic)
	recType := extractRecType(rec.Title)
	staticExplanation := GetStaticExplanation(recType, rec.ResourceType, rec.EstimatedSavings, metrics)

	if staticExplanation != "" {
		enhanced.Explanation = staticExplanation
		enhanced.SavingsRationale = generateSavingsRationale(rec.ResourceType, rec.EstimatedSavings)
		enhanced.ImpactSummary = generateImpactSummary(recType, rec.RiskLevel)
		return enhanced
	}

	// Fallback to generic explanation
	enhanced.Explanation = rec.Rationale
	enhanced.SavingsRationale = generateSavingsRationale(rec.ResourceType, rec.EstimatedSavings)
	enhanced.ImpactSummary = "Review recommended before implementation."
	enhanced.Confidence = "medium"

	return enhanced
}

func extractRecType(title string) string {
	title = strings.ToLower(title)
	if strings.Contains(title, "stop") {
		return "stop"
	}
	if strings.Contains(title, "delete") || strings.Contains(title, "remove") {
		return "delete"
	}
	if strings.Contains(title, "resize") || strings.Contains(title, "downsize") {
		return "resize"
	}
	return "other"
}

func generateSavingsRationale(resourceType string, savings float64) string {
	annualSavings := savings * 12
	return fmt.Sprintf("Monthly: $%.2f | Annual: $%.2f", savings, annualSavings)
}

func generateImpactSummary(recType string, riskLevel string) string {
	impacts := map[string]map[string]string{
		"stop": {
			"low":    "Safe to stop. No active workloads detected.",
			"medium": "Review active connections before stopping.",
			"high":   "Verify no critical processes running.",
		},
		"delete": {
			"low":    "Safe to delete. No dependencies found.",
			"medium": "Ensure backups exist before deletion.",
			"high":   "Manual review required before deletion.",
		},
		"resize": {
			"low":    "Resize during low-traffic window.",
			"medium": "Brief downtime expected during resize.",
			"high":   "Plan maintenance window for resize.",
		},
	}

	if typeImpacts, ok := impacts[recType]; ok {
		if impact, ok := typeImpacts[riskLevel]; ok {
			return impact
		}
	}

	return "Review impact before implementation."
}

// Disclaimer constants
const (
	AIDisclaimer    = "AI-generated analysis. Verify before taking action."
	SlackDisclaimer = "_Analysis by DevCost AI. Always verify recommendations._"
	APIDisclaimer   = "This analysis is AI-generated and should be reviewed by a human before any action is taken."
)
