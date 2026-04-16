package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// SlackHandler handles Slack slash commands and webhooks
type SlackHandler struct {
	logger              *logger.Logger
	wasteService        *services.WasteDetectionService
	recommendationService *services.RecommendationService
	actionPipeline      *services.ActionPipeline
	simulationService *services.SimulationService
	aiService         *services.AIService
	aiValidator       *services.AISafetyValidator
	slackToken          string
}

// NewSlackHandler creates a new Slack handler
func NewSlackHandler(
	logger *logger.Logger,
	wasteService *services.WasteDetectionService,
	recommendationService *services.RecommendationService,
	actionPipeline *services.ActionPipeline,
	simulationService *services.SimulationService,
	aiService *services.AIService,
	slackToken string,
) *SlackHandler {
	return &SlackHandler{
		logger:              logger,
		wasteService:        wasteService,
		recommendationService: recommendationService,
		actionPipeline:      actionPipeline,
		simulationService:   simulationService,
		aiService:           aiService,
		aiValidator:         services.NewAISafetyValidator(),
		slackToken:          slackToken,
	}
}

// SlackCommandRequest represents a Slack slash command request
type SlackCommandRequest struct {
	Token       string `form:"token"`
	TeamID      string `form:"team_id"`
	TeamDomain  string `form:"team_domain"`
	ChannelID   string `form:"channel_id"`
	ChannelName string `form:"channel_name"`
	UserID      string `form:"user_id"`
	UserName    string `form:"user_name"`
	Command     string `form:"command"`
	Text        string `form:"text"`
	ResponseURL string `form:"response_url"`
	TriggerID   string `form:"trigger_id"`
}

// SlackResponse represents a Slack API response
type SlackResponse struct {
	ResponseType string `json:"response_type,omitempty"` // ephemeral, in_channel
	Text         string `json:"text,omitempty"`
	Blocks       []Block `json:"blocks,omitempty"`
}

// Block represents a Slack block
type Block struct {
	Type string      `json:"type"`
	Text *TextObject `json:"text,omitempty"`
	Fields []TextObject `json:"fields,omitempty"`
}

// TextObject represents Slack text
type TextObject struct {
	Type  string `json:"type"` // mrkdwn, plain_text
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// HandleSlackCommand handles incoming Slack slash commands
func (h *SlackHandler) HandleSlackCommand(c *gin.Context) {
	var req SlackCommandRequest
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Warn("Invalid Slack request", zap.Error(err))
		c.JSON(http.StatusBadRequest, SlackResponse{
			Text: "Invalid request",
		})
		return
	}

	// Verify token
	if req.Token != h.slackToken {
		h.logger.Warn("Invalid Slack token",
			zap.String("expected", h.slackToken),
			zap.String("received", req.Token),
		)
		c.JSON(http.StatusUnauthorized, SlackResponse{
			Text: "Unauthorized",
		})
		return
	}

	h.logger.Info("Received Slack command",
		zap.String("command", req.Command),
		zap.String("text", req.Text),
		zap.String("user", req.UserName),
		zap.String("channel", req.ChannelName),
	)

	// Route command
	ctx := context.Background()
	var response SlackResponse

	switch req.Command {
	case "/cost":
		response = h.handleCostCommand(ctx, req)
	default:
		response = SlackResponse{
			Text: fmt.Sprintf("Unknown command: %s", req.Command),
		}
	}

	c.JSON(http.StatusOK, response)
}

// handleCostCommand routes /cost subcommands
func (h *SlackHandler) handleCostCommand(ctx context.Context, req SlackCommandRequest) SlackResponse {
	args := strings.Fields(strings.ToLower(req.Text))
	if len(args) == 0 {
		return h.showHelp()
	}

	subcommand := args[0]

	switch subcommand {
	case "report":
		return h.handleCostReport(ctx)
	case "optimize":
		return h.handleCostOptimize(ctx)
	case "simulate":
		return h.handleCostSimulate(ctx)
	case "explain":
		return h.handleCostExplain(ctx, args[1:])
	case "why":
		return h.handleCostWhy(ctx)
	case "help":
		return h.showHelp()
	default:
		return SlackResponse{
			Text: fmt.Sprintf("Unknown subcommand: `%s`. Try `/cost help` for available commands.", subcommand),
		}
	}
}

// handleCostReport generates a cost report
func (h *SlackHandler) handleCostReport(ctx context.Context) SlackResponse {
	h.logger.Info("Generating cost report for Slack")

	// Get waste detection results
	wasteResult, err := h.wasteService.DetectWaste(ctx)
	if err != nil {
		h.logger.Error("Failed to detect waste", zap.Error(err))
		return SlackResponse{
			Text: ":x: Failed to generate report. Please try again later.",
		}
	}

	// Get recommendations
	recommendations, err := h.recommendationService.GetActiveRecommendations(ctx)
	if err != nil {
		h.logger.Error("Failed to get recommendations", zap.Error(err))
		return SlackResponse{
			Text: ":x: Failed to generate report. Please try again later.",
		}
	}

	// Calculate totals
	var totalSavings float64
	var criticalCount, highCount, mediumCount int
	for _, rec := range recommendations {
		totalSavings += rec.EstimatedSavings
		switch rec.Priority {
		case services.RecommendationPriorityCritical:
			criticalCount++
		case services.RecommendationPriorityHigh:
			highCount++
		case services.RecommendationPriorityMedium:
			mediumCount++
		}
	}

	// Format report
	text := fmt.Sprintf(`
*📊 DevCost AI Report*

*Waste Detected:*
• %d idle EC2 instances
• %d unattached EBS volumes
• %d underutilized RDS instances

*Recommendations:*
• Total: %d
• Critical: %d :red_circle:
• High: %d :orange_circle:
• Medium: %d :yellow_circle:

*💰 Potential Monthly Savings:* $%.2f
*💰 Potential Annual Savings:* $%.2f

Use `/cost simulate` to preview actions.
Use `/cost optimize` to execute recommendations.
`,
		countByType(wasteResult.WasteResources, "EC2"),
		countByType(wasteResult.WasteResources, "EBS"),
		countByType(wasteResult.WasteResources, "RDS"),
		len(recommendations),
		criticalCount,
		highCount,
		mediumCount,
		totalSavings,
		totalSavings*12,
	)

	return SlackResponse{
		ResponseType: "in_channel",
		Text:         text,
	}
}

// handleCostOptimize executes optimization
func (h *SlackHandler) handleCostOptimize(ctx context.Context) SlackResponse {
	h.logger.Info("Starting cost optimization from Slack")

	// First run simulation to show what will happen
	summary, err := h.simulationService.GetQuickSummary(ctx)
	if err != nil {
		h.logger.Error("Failed to get simulation summary", zap.Error(err))
		return SlackResponse{
			Text: ":x: Failed to start optimization. Please try again later.",
		}
	}

	if summary.ReadyToExecute == 0 {
		return SlackResponse{
			ResponseType: "ephemeral",
			Text:         ":information_source: No recommendations ready for execution. Run `/cost report` to see current status.",
		}
	}

	// Run the action pipeline
	result, err := h.actionPipeline.Run(ctx)
	if err != nil {
		h.logger.Error("Action pipeline failed", zap.Error(err))
		return SlackResponse{
			Text: ":x: Optimization failed. Please check logs or contact support.",
		}
	}

	// Format response
	var text string
	if result.FailedCount == 0 {
		text = fmt.Sprintf(`
✅ *Optimization Complete*

*Actions Executed:* %d
*Failed:* %d
*Duration:* %s

*💰 Monthly Savings:* $%.2f
*💰 Annual Savings:* $%.2f

All approved recommendations have been executed successfully.
`,
			result.ExecutedCount,
			result.FailedCount,
			result.Duration.String(),
			result.TotalEstimatedSavings,
			result.TotalEstimatedSavings*12,
		)
	} else {
		text = fmt.Sprintf(`
⚠️ *Optimization Partially Complete*

*Actions Executed:* %d
*Failed:* %d
*Duration:* %s

*💰 Monthly Savings:* $%.2f
*💰 Annual Savings:* $%.2f

Some actions failed. Check `/cost report` for details.
`,
			result.ExecutedCount,
			result.FailedCount,
			result.Duration.String(),
			result.TotalEstimatedSavings,
			result.TotalEstimatedSavings*12,
		)
	}

	return SlackResponse{
		ResponseType: "in_channel",
		Text:         text,
	}
}

// handleCostSimulate runs a simulation
func (h *SlackHandler) handleCostSimulate(ctx context.Context) SlackResponse {
	h.logger.Info("Running cost simulation from Slack")

	// Run simulation
	result, err := h.simulationService.SimulateAll(ctx, nil)
	if err != nil {
		h.logger.Error("Simulation failed", zap.Error(err))
		return SlackResponse{
			Text: ":x: Failed to run simulation. Please try again later.",
		}
	}

	// Format simulation results
	text := fmt.Sprintf(`
🔮 *Cost Optimization Simulation*

*Recommendations Analyzed:* %d
*Ready to Execute:* %d :white_check_mark:
*Requires Approval:* %d :hourglass:
*Rejected:* %d :x:

*💰 Potential Monthly Savings:* $%.2f
*💰 Potential Annual Savings:* $%.2f

*Resources Affected:* %d
*High Risk:* %d :warning:
*Medium Risk:* %d
*Low Risk:* %d

*Estimated Implementation Time:* %s

Run `/cost optimize` to execute ready recommendations.
`,
		result.TotalRecommendations,
		result.ApprovedCount,
		result.PendingCount,
		result.RejectedCount,
		result.TotalEstimatedSavings,
		result.TotalEstimatedSavings*12,
		result.ResourcesAffected,
		result.HighRiskCount,
		result.MediumRiskCount,
		result.LowRiskCount,
		result.EstimatedImplementationTime,
	)

	return SlackResponse{
		ResponseType: "in_channel",
		Text:         text,
	}
}

// showHelp displays help information
func (h *SlackHandler) showHelp() SlackResponse {
	text := `
*🤖 DevCost AI - Slack Commands*

*/cost report*
Show current waste detection and recommendations summary.

*/cost simulate*
Preview what actions would be taken and potential savings.

*/cost optimize*
Execute approved recommendations to optimize costs.

*/cost explain [resource-id]*
Get AI explanation for a specific recommendation.

*/cost why*
AI-generated explanation of current cost drivers and waste.

*/cost help*
Show this help message.

All commands provide cost savings estimates and risk assessment.
`

	return SlackResponse{
		ResponseType: "ephemeral",
		Text:         text,
	}
}

// handleCostExplain provides AI explanation for a recommendation
func (h *SlackHandler) handleCostExplain(ctx context.Context, args []string) SlackResponse {
	if len(args) == 0 {
		return SlackResponse{
			ResponseType: "ephemeral",
			Text:         ":information_source: Usage: `/cost explain <resource-id>`\nExample: `/cost explain i-1234567890abcdef0`",
		}
	}

	resourceID := args[0]
	h.logger.Info("Generating AI explanation for resource", zap.String("resource_id", resourceID))

	// Get recommendation for this resource
	recommendations, err := h.recommendationService.GetActiveRecommendations(ctx)
	if err != nil {
		return SlackResponse{
			Text: ":x: Failed to fetch recommendations.",
		}
	}

	var targetRec *services.Recommendation
	for _, rec := range recommendations {
		if rec.ResourceID == resourceID {
			targetRec = rec
			break
		}
	}

	if targetRec == nil {
		return SlackResponse{
			ResponseType: "ephemeral",
			Text:         fmt.Sprintf(":question: No recommendation found for resource `%s`", resourceID),
		}
	}

	// Generate explanation
	input := services.RecommendationInput{
		ResourceID:       targetRec.ResourceID,
		ResourceType:     targetRec.ResourceType,
		Title:            targetRec.Title,
		Rationale:        targetRec.Rationale,
		EstimatedSavings: targetRec.EstimatedSavings,
		RiskLevel:        string(targetRec.RiskLevel),
	}

	enhanced := services.GenerateExplanation(h.aiService, input, nil)

	// Sanitize for Slack
	explanation := h.aiValidator.SanitizeForSlack(enhanced.Explanation)

	text := fmt.Sprintf(`
*🔍 Recommendation Explanation*

*Resource:* `+"`%s`"+` (%s)
*Action:* %s

*Explanation:*
%s

*💰 Savings:* %s
*Impact:* %s

%s
`,
		targetRec.ResourceID,
		targetRec.ResourceType,
		targetRec.Title,
		explanation,
		enhanced.SavingsRationale,
		enhanced.ImpactSummary,
		services.SlackDisclaimer,
	)

	return SlackResponse{
		ResponseType: "in_channel",
		Text:         text,
	}
}

// handleCostWhy provides AI analysis of cost drivers
func (h *SlackHandler) handleCostWhy(ctx context.Context) SlackResponse {
	h.logger.Info("Generating AI cost analysis for Slack")

	// Get waste data
	wasteResult, err := h.wasteService.DetectWaste(ctx)
	if err != nil {
		return SlackResponse{
			Text: ":x: Failed to analyze costs.",
		}
	}

	// Get recommendations
	recommendations, err := h.recommendationService.GetActiveRecommendations(ctx)
	if err != nil {
		return SlackResponse{
			Text: ":x: Failed to analyze costs.",
		}
	}

	// Calculate totals
	var totalSavings float64
	var topServices []services.ServiceCost
	serviceMap := make(map[string]float64)

	for _, rec := range recommendations {
		totalSavings += rec.EstimatedSavings
		serviceMap[rec.ResourceType] += rec.EstimatedSavings
	}

	for svc, cost := range serviceMap {
		topServices = append(topServices, services.ServiceCost{
			Service: svc,
			Cost:    cost,
		})
	}

	// Build AI analysis input
	input := services.CostAnalysisInput{
		TotalCost:    totalSavings, // Using savings as proxy for waste
		PreviousCost: totalSavings * 0.9, // Assume 10% increase for demo
		TopServices:  topServices,
	}

	// Get AI analysis
	result, err := h.aiService.AnalyzeCost(ctx, input)
	if err != nil {
		result = &services.CostAnalysisResult{
			Summary:     "Analysis unavailable. Please try again.",
			CostDrivers: []string{},
			RiskLevel:   "unknown",
		}
	}

	// Sanitize output
	summary := h.aiValidator.SanitizeForSlack(result.Summary)

	// Format cost drivers
	driversText := ""
	for _, driver := range result.CostDrivers {
		driversText += fmt.Sprintf("• %s\n", driver)
	}
	if driversText == "" {
		driversText = "• No significant cost drivers identified\n"
	}

	// Format top waste sources
	wasteText := ""
	wasteCount := 0
	for _, w := range wasteResult.WasteResources {
		if wasteCount >= 3 {
			break
		}
		wasteText += fmt.Sprintf("• %s: %s ($%.2f/mo)\n", w.ResourceType, w.Reason, w.EstimatedSavings)
		wasteCount++
	}
	if wasteText == "" {
		wasteText = "• No waste detected\n"
	}

	text := fmt.Sprintf(`
*🤖 AI Cost Analysis*

*Summary:*
%s

*Cost Drivers:*
%s
*Top Waste Sources:*
%s
*Risk Level:* %s

*💰 Total Potential Savings:* $%.2f/month

%s
`,
		summary,
		driversText,
		wasteText,
		result.RiskLevel,
		totalSavings,
		services.SlackDisclaimer,
	)

	return SlackResponse{
		ResponseType: "in_channel",
		Text:         text,
	}
}

// HandleSlackWebhook handles incoming Slack webhooks (for async responses)
func (h *SlackHandler) HandleSlackWebhook(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Warn("Invalid webhook payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	h.logger.Debug("Received Slack webhook", zap.Any("payload", payload))

	// Process webhook asynchronously
	go h.processWebhook(payload)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// processWebhook processes webhook payload asynchronously
func (h *SlackHandler) processWebhook(payload map[string]interface{}) {
	// Implementation for async processing
	// e.g., handle interactive buttons, follow-up messages
	time.Sleep(100 * time.Millisecond) // Simulate processing
}

// countByType counts waste resources by type
func countByType(resources []*services.WasteResult, resourceType string) int {
	count := 0
	for _, r := range resources {
		if r.ResourceType == resourceType {
			count++
		}
	}
	return count
}
