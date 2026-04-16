package services

import (
	"fmt"
	"strings"
)

// RecommendationInput holds data for recommendation explanation
type RecommendationInput struct {
	ResourceID       string  `json:"resource_id"`
	ResourceType     string  `json:"resource_type"`
	Title            string  `json:"title"`
	Rationale        string  `json:"rationale"`
	EstimatedSavings float64 `json:"estimated_savings"`
	CurrentState     string  `json:"current_state"`
	ProposedState    string  `json:"proposed_state"`
	RiskLevel        string  `json:"risk_level"`
}

// DailyCost represents daily cost data point
type DailyCost struct {
	Date string  `json:"date"`
	Cost float64 `json:"cost"`
}

// BuildCostAnalysisPrompt creates a structured prompt for cost analysis
func BuildCostAnalysisPrompt(input CostAnalysisInput) string {
	var sb strings.Builder

	sb.WriteString("Analyze this cloud cost data and provide insights.\n\n")
	sb.WriteString("RULES:\n")
	sb.WriteString("- Be concise (max 3 sentences per section)\n")
	sb.WriteString("- Focus on actionable insights\n")
	sb.WriteString("- Do NOT suggest specific actions to execute\n\n")

	sb.WriteString("DATA:\n")
	sb.WriteString(fmt.Sprintf("Current Period Cost: $%.2f\n", input.TotalCost))
	sb.WriteString(fmt.Sprintf("Previous Period Cost: $%.2f\n", input.PreviousCost))

	change := ((input.TotalCost - input.PreviousCost) / input.PreviousCost) * 100
	sb.WriteString(fmt.Sprintf("Change: %.1f%%\n\n", change))

	if len(input.TopServices) > 0 {
		sb.WriteString("TOP SERVICES BY COST:\n")
		for i, svc := range input.TopServices {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s: $%.2f (%.1f%% change)\n", svc.Service, svc.Cost, svc.Change))
		}
		sb.WriteString("\n")
	}

	if len(input.ResourceChanges) > 0 {
		sb.WriteString("RECENT CHANGES:\n")
		for i, rc := range input.ResourceChanges {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s %s: %s\n", rc.ChangeType, rc.ResourceType, rc.Impact))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("RESPOND WITH:\n")
	sb.WriteString("1. Summary (1-2 sentences)\n")
	sb.WriteString("2. Main cost drivers (bullet points)\n")
	sb.WriteString("3. Areas to investigate (bullet points)\n")

	return sb.String()
}

// BuildRecommendationPrompt creates a prompt for explaining a recommendation
func BuildRecommendationPrompt(rec RecommendationInput) string {
	var sb strings.Builder

	sb.WriteString("Explain this cloud cost optimization recommendation in plain English.\n\n")
	sb.WriteString("RULES:\n")
	sb.WriteString("- Write for a non-technical audience\n")
	sb.WriteString("- Be specific about the savings\n")
	sb.WriteString("- Explain WHY this saves money\n")
	sb.WriteString("- Do NOT instruct to execute any action\n")
	sb.WriteString("- Maximum 4 sentences\n\n")

	sb.WriteString("RECOMMENDATION:\n")
	sb.WriteString(fmt.Sprintf("Resource: %s (%s)\n", rec.ResourceID, rec.ResourceType))
	sb.WriteString(fmt.Sprintf("Action: %s\n", rec.Title))
	sb.WriteString(fmt.Sprintf("Reason: %s\n", rec.Rationale))
	sb.WriteString(fmt.Sprintf("Estimated Savings: $%.2f/month\n", rec.EstimatedSavings))
	sb.WriteString(fmt.Sprintf("Risk Level: %s\n", rec.RiskLevel))

	if rec.CurrentState != "" {
		sb.WriteString(fmt.Sprintf("Current State: %s\n", rec.CurrentState))
	}
	if rec.ProposedState != "" {
		sb.WriteString(fmt.Sprintf("Proposed State: %s\n", rec.ProposedState))
	}

	sb.WriteString("\nExplain this recommendation:")

	return sb.String()
}

// BuildAnomalyPrompt creates a prompt for anomaly detection
func BuildAnomalyPrompt(data []DailyCost) string {
	var sb strings.Builder

	sb.WriteString("Analyze this cost time series for anomalies.\n\n")
	sb.WriteString("RULES:\n")
	sb.WriteString("- Identify unusual spikes or drops\n")
	sb.WriteString("- Explain what might cause the anomaly\n")
	sb.WriteString("- Do NOT suggest executing any actions\n")
	sb.WriteString("- Be concise (max 3 sentences)\n\n")

	sb.WriteString("DAILY COSTS (last 14 days):\n")
	start := 0
	if len(data) > 14 {
		start = len(data) - 14
	}
	for _, d := range data[start:] {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", d.Date, d.Cost))
	}

	// Calculate stats
	var sum float64
	for _, d := range data {
		sum += d.Cost
	}
	avg := sum / float64(len(data))
	latest := data[len(data)-1].Cost
	deviation := ((latest - avg) / avg) * 100

	sb.WriteString(fmt.Sprintf("\nAverage: $%.2f\n", avg))
	sb.WriteString(fmt.Sprintf("Latest: $%.2f (%.1f%% from avg)\n", latest, deviation))

	sb.WriteString("\nIs this an anomaly? Explain briefly:")

	return sb.String()
}

// BuildWasteExplanationPrompt creates a prompt for explaining waste
func BuildWasteExplanationPrompt(wasteType string, resourceType string, reason string, savings float64) string {
	var sb strings.Builder

	sb.WriteString("Explain this cloud resource waste in simple terms.\n\n")
	sb.WriteString("RULES:\n")
	sb.WriteString("- Explain why this is considered waste\n")
	sb.WriteString("- Be specific about the impact\n")
	sb.WriteString("- Maximum 2 sentences\n")
	sb.WriteString("- Do NOT instruct to delete or modify anything\n\n")

	sb.WriteString("WASTE DETECTED:\n")
	sb.WriteString(fmt.Sprintf("Type: %s\n", wasteType))
	sb.WriteString(fmt.Sprintf("Resource: %s\n", resourceType))
	sb.WriteString(fmt.Sprintf("Reason: %s\n", reason))
	sb.WriteString(fmt.Sprintf("Potential Savings: $%.2f/month\n", savings))

	sb.WriteString("\nExplain this waste:")

	return sb.String()
}

// BuildSlackSummaryPrompt creates a prompt for Slack cost summary
func BuildSlackSummaryPrompt(totalCost float64, wastePercent float64, topWaste []string) string {
	var sb strings.Builder

	sb.WriteString("Create a brief Slack message about cloud costs.\n\n")
	sb.WriteString("RULES:\n")
	sb.WriteString("- Use Slack markdown formatting\n")
	sb.WriteString("- Be concise (max 100 words)\n")
	sb.WriteString("- Include emoji sparingly\n")
	sb.WriteString("- Do NOT suggest specific actions to take\n\n")

	sb.WriteString("DATA:\n")
	sb.WriteString(fmt.Sprintf("Total Monthly Cost: $%.2f\n", totalCost))
	sb.WriteString(fmt.Sprintf("Waste Percentage: %.1f%%\n", wastePercent))

	if len(topWaste) > 0 {
		sb.WriteString("Top Waste Areas:\n")
		for _, w := range topWaste {
			sb.WriteString(fmt.Sprintf("- %s\n", w))
		}
	}

	sb.WriteString("\nCreate a brief summary:")

	return sb.String()
}

// Pre-built explanation templates for common scenarios (no AI needed)

// GetStaticExplanation returns a template-based explanation
func GetStaticExplanation(recType string, resourceType string, savings float64, metrics map[string]float64) string {
	templates := map[string]map[string]string{
		"stop": {
			"ec2": "This EC2 instance can be stopped because CPU usage is %.1f%% (below 5%% threshold). Stopping saves $%.2f/month while preserving data.",
			"rds": "This RDS instance shows minimal query activity. Stopping during off-hours saves $%.2f/month.",
		},
		"delete": {
			"ebs":      "This EBS volume has been unattached for 30+ days. Deleting saves $%.2f/month with no impact.",
			"snapshot": "This snapshot is older than retention policy. Deleting saves $%.2f/month.",
		},
		"resize": {
			"ec2": "This EC2 instance uses only %.1f%% of CPU. Downsizing to a smaller type saves $%.2f/month.",
			"rds": "This RDS instance is oversized for current workload. Right-sizing saves $%.2f/month.",
		},
	}

	if typeTemplates, ok := templates[recType]; ok {
		if template, ok := typeTemplates[resourceType]; ok {
			if cpu, hasCPU := metrics["cpu"]; hasCPU {
				return fmt.Sprintf(template, cpu, savings)
			}
			return fmt.Sprintf(template, savings)
		}
	}

	return fmt.Sprintf("This optimization saves approximately $%.2f per month.", savings)
}
