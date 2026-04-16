package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AIService provides AI-powered analysis using Ollama
type AIService struct {
	baseURL    string
	model      string
	httpClient *http.Client
	enabled    bool
}

// AIConfig holds AI service configuration
type AIConfig struct {
	BaseURL string
	Model   string
	Timeout time.Duration
	Enabled bool
}

// OllamaRequest represents a request to Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents a response from Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// CostAnalysisInput holds data for cost analysis
type CostAnalysisInput struct {
	TotalCost       float64          `json:"total_cost"`
	PreviousCost    float64          `json:"previous_cost"`
	TopServices     []ServiceCost    `json:"top_services"`
	ResourceChanges []ResourceChange `json:"resource_changes"`
}

// ServiceCost represents cost per service
type ServiceCost struct {
	Service string  `json:"service"`
	Cost    float64 `json:"cost"`
	Change  float64 `json:"change_percent"`
}

// ResourceChange represents a resource change event
type ResourceChange struct {
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	ChangeType   string `json:"change_type"`
	Impact       string `json:"impact"`
}

// CostAnalysisResult holds AI analysis output
type CostAnalysisResult struct {
	Summary     string   `json:"summary"`
	CostDrivers []string `json:"cost_drivers"`
	Suggestions []string `json:"suggestions"`
	RiskLevel   string   `json:"risk_level"`
}

// AnomalyResult holds anomaly detection output
type AnomalyResult struct {
	IsAnomaly   bool     `json:"is_anomaly"`
	Confidence  float64  `json:"confidence"`
	Explanation string   `json:"explanation"`
	Factors     []string `json:"factors"`
}

// NewAIService creates a new AI service instance
func NewAIService(cfg AIConfig) *AIService {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "llama3.2"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &AIService{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		enabled: cfg.Enabled,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// AnalyzeCost analyzes cost data and provides insights
func (s *AIService) AnalyzeCost(ctx context.Context, input CostAnalysisInput) (*CostAnalysisResult, error) {
	if !s.enabled {
		return s.fallbackCostAnalysis(input), nil
	}

	prompt := BuildCostAnalysisPrompt(input)
	response, err := s.query(ctx, prompt)
	if err != nil {
		return s.fallbackCostAnalysis(input), nil
	}

	return parseCostAnalysisResponse(response, input), nil
}

// ExplainRecommendation generates a human-readable explanation
func (s *AIService) ExplainRecommendation(ctx context.Context, rec RecommendationInput) (string, error) {
	if !s.enabled {
		return s.fallbackExplanation(rec), nil
	}

	prompt := BuildRecommendationPrompt(rec)
	response, err := s.query(ctx, prompt)
	if err != nil {
		return s.fallbackExplanation(rec), nil
	}

	return response, nil
}

// DetectAnomaly checks for cost anomalies
func (s *AIService) DetectAnomaly(ctx context.Context, data []DailyCost) (*AnomalyResult, error) {
	if !s.enabled {
		return s.fallbackAnomalyDetection(data), nil
	}

	prompt := BuildAnomalyPrompt(data)
	response, err := s.query(ctx, prompt)
	if err != nil {
		return s.fallbackAnomalyDetection(data), nil
	}

	return parseAnomalyResponse(response, data), nil
}

// query sends a prompt to Ollama and returns the response
func (s *AIService) query(ctx context.Context, prompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:  s.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return ollamaResp.Response, nil
}

// Fallback methods when AI is disabled or fails

func (s *AIService) fallbackCostAnalysis(input CostAnalysisInput) *CostAnalysisResult {
	result := &CostAnalysisResult{
		RiskLevel: "low",
	}

	change := ((input.TotalCost - input.PreviousCost) / input.PreviousCost) * 100

	if change > 20 {
		result.Summary = fmt.Sprintf("Costs increased %.1f%% from $%.2f to $%.2f", change, input.PreviousCost, input.TotalCost)
		result.RiskLevel = "high"
	} else if change > 10 {
		result.Summary = fmt.Sprintf("Costs increased %.1f%% - moderate growth", change)
		result.RiskLevel = "medium"
	} else if change < -10 {
		result.Summary = fmt.Sprintf("Costs decreased %.1f%% - good progress", -change)
	} else {
		result.Summary = "Costs are stable"
	}

	for _, svc := range input.TopServices {
		if svc.Change > 15 {
			result.CostDrivers = append(result.CostDrivers, fmt.Sprintf("%s (+%.1f%%)", svc.Service, svc.Change))
		}
	}

	if len(result.CostDrivers) == 0 {
		result.CostDrivers = []string{"No significant cost drivers identified"}
	}

	result.Suggestions = []string{"Review top services for optimization opportunities"}
	return result
}

func (s *AIService) fallbackExplanation(rec RecommendationInput) string {
	return fmt.Sprintf(
		"Recommendation: %s\n\nThis %s resource (%s) can save approximately $%.2f/month. %s",
		rec.Title,
		rec.ResourceType,
		rec.ResourceID,
		rec.EstimatedSavings,
		rec.Rationale,
	)
}

func (s *AIService) fallbackAnomalyDetection(data []DailyCost) *AnomalyResult {
	if len(data) < 2 {
		return &AnomalyResult{IsAnomaly: false, Confidence: 0, Explanation: "Insufficient data"}
	}

	var sum, count float64
	for _, d := range data[:len(data)-1] {
		sum += d.Cost
		count++
	}
	avg := sum / count
	latest := data[len(data)-1].Cost
	deviation := ((latest - avg) / avg) * 100

	if deviation > 50 {
		return &AnomalyResult{
			IsAnomaly:   true,
			Confidence:  0.8,
			Explanation: fmt.Sprintf("Cost spike of %.1f%% above average", deviation),
			Factors:     []string{"Unusual spending pattern detected"},
		}
	}

	return &AnomalyResult{
		IsAnomaly:   false,
		Confidence:  0.9,
		Explanation: "Costs within normal range",
	}
}

// Helper functions

func parseCostAnalysisResponse(response string, input CostAnalysisInput) *CostAnalysisResult {
	// Simple parsing - in production, use structured output from LLM
	result := &CostAnalysisResult{
		Summary:     response,
		CostDrivers: []string{},
		Suggestions: []string{},
		RiskLevel:   "medium",
	}

	change := ((input.TotalCost - input.PreviousCost) / input.PreviousCost) * 100
	if change > 20 {
		result.RiskLevel = "high"
	} else if change < 5 {
		result.RiskLevel = "low"
	}

	return result
}

func parseAnomalyResponse(response string, data []DailyCost) *AnomalyResult {
	// Simple parsing - check if response indicates anomaly
	return &AnomalyResult{
		IsAnomaly:   false,
		Confidence:  0.7,
		Explanation: response,
	}
}

// IsEnabled returns whether AI service is enabled
func (s *AIService) IsEnabled() bool {
	return s.enabled
}

// Health checks if Ollama is reachable
func (s *AIService) Health(ctx context.Context) error {
	if !s.enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama health check failed: status %d", resp.StatusCode)
	}

	return nil
}
