package api
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
)
// OpenRouterClient handles OpenRouter API interactions
type OpenRouterClient struct {
	config        config.OpenRouterConfig
	client        *http.Client
	keyManager    *APIKeyManager
	modelSelector *ModelSelector
	lastUsedModel string
	lastUsage     *types.Usage
	mutex         sync.RWMutex
}
// NewOpenRouterClient creates a new OpenRouter client
func NewOpenRouterClient(cfg config.OpenRouterConfig) *OpenRouterClient {
	models := cfg.Models
	if len(models) == 0 && cfg.Model != "" {
		models = []string{cfg.Model}
	}
	strategy := cfg.ModelStrategy
	if strategy == "" {
		strategy = "failover"
	}
	return &OpenRouterClient{
		config:        cfg,
		keyManager:    NewAPIKeyManager("OpenRouter", cfg.GetAllAPIKeys()),
		modelSelector: NewModelSelector(models, strategy),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}
// GenerateCode generates code using the OpenRouter API with automatic failover
func (c *OpenRouterClient) GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (*types.CodeGenerationResult, error) {
	if c.keyManager == nil {
		return nil, fmt.Errorf("no OpenRouter API key configured")
	}

	// Check rate limits before attempting API call
	rateLimits, err := c.GetRateLimits(ctx)
	if err != nil {
		logger.Warnf("OpenRouter: Failed to check rate limits (continuing anyway): %v", err)
	} else {
		// Check if we're on free tier with limited requests
		if rateLimits.IsFreeTier && rateLimits.LimitRemaining != nil && *rateLimits.LimitRemaining <= 0 {
			return nil, fmt.Errorf("OpenRouter: rate limit exceeded (free tier limit reached, remaining: %.2f)", *rateLimits.LimitRemaining)
		}
		// Check if we have credits remaining (for paid tiers)
		if rateLimits.Limit != nil && rateLimits.LimitRemaining != nil && *rateLimits.LimitRemaining <= 0 {
			return nil, fmt.Errorf("OpenRouter: credit limit exceeded (remaining: %.2f of %.2f)", *rateLimits.LimitRemaining, *rateLimits.Limit)
		}
		logger.Debugf("OpenRouter: Rate limit check passed - remaining: %v, free_tier: %v",
			rateLimits.LimitRemaining, rateLimits.IsFreeTier)
	}

	detectedLanguage := utils.GetLanguageFromFile(outputFile, language)
	fullPrompt := c.buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage, contextFiles)
	requestData, err := c.prepareRequest(fullPrompt, detectedLanguage)
	if err != nil {
		return nil, err
	}
	code, err := c.keyManager.ExecuteWithFailover(func(apiKey string) (string, error) {
		response, err := c.makeAPICallWithKey(ctx, requestData, apiKey)
		if err != nil {
			return "", err
		}
		content := response.Choices[0].Message.Content
		cleanedContent := utils.CleanCodeResponse(content)
		// Store usage information
		c.lastUsage = &types.Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		}
		logger.Debugf("OpenRouter: Extracted token usage - Prompt: %d, Completion: %d, Total: %d",
			c.lastUsage.PromptTokens, c.lastUsage.CompletionTokens, c.lastUsage.TotalTokens)
		return cleanedContent, nil
	})
	if err != nil {
		return nil, err
	}
	result := &types.CodeGenerationResult{
		Code:  code,
		Usage: c.lastUsage,
	}
	if result.Usage != nil {
		logger.Debugf("OpenRouter: Returning result with usage - Total tokens: %d", result.Usage.TotalTokens)
	} else {
		logger.Warnf("OpenRouter: Returning result with nil usage")
	}
	return result, nil
}
// buildFullPrompt builds the complete prompt including context and existing content
func (c *OpenRouterClient) buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage string, contextFiles []string) string {
	var parts []string
	if len(contextFiles) > 0 {
		filteredContextFiles := c.filterContextFiles(contextFiles, outputFile)
		if len(filteredContextFiles) > 0 {
			contextContent := "Context Files:\n"
			for _, contextFile := range filteredContextFiles {
				if content, err := utils.ReadFileContent(contextFile); err == nil && content != "" {
					contextLang := utils.GetLanguageFromFile(contextFile, nil)
					contextContent += fmt.Sprintf("\nFile: %s\n```%s\n%s\n```\n", contextFile, contextLang, content)
				} else {
					logger.Warnf("Could not read context file %s: %v", contextFile, err)
				}
			}
			parts = append(parts, contextContent)
		}
	}
	if contextStr != "" {
		parts = append(parts, fmt.Sprintf("Context: %s", contextStr))
	}
	if existingContent, err := utils.ReadFileContent(outputFile); err == nil && existingContent != "" {
		parts = append(parts, fmt.Sprintf("Existing file content:\n```%s\n%s\n```\n", detectedLanguage, existingContent))
	}
	parts = append(parts, fmt.Sprintf("Generate %s code for: %s", detectedLanguage, prompt))
	return strings.Join(parts, "\n\n")
}
// filterContextFiles filters out the output file from context files
func (c *OpenRouterClient) filterContextFiles(contextFiles []string, outputFile string) []string {
	var filtered []string
	for _, file := range contextFiles {
		contextAbs := filepath.Clean(file)
		outputAbs := filepath.Clean(outputFile)
		if contextAbs != outputAbs {
			filtered = append(filtered, file)
		}
	}
	return filtered
}
// prepareRequest prepares the API request payload
func (c *OpenRouterClient) prepareRequest(fullPrompt, detectedLanguage string) (OpenRouterRequest, error) {
	modelName, err := c.modelSelector.SelectModel()
	if err != nil {
		return OpenRouterRequest{}, fmt.Errorf("failed to select model: %w", err)
	}
	if c.config.FreeOnly && !strings.HasSuffix(modelName, ":free") {
		modelName = modelName + ":free"
		logger.Debugf("OpenRouter: free_only enabled, using model: %s", modelName)
	} else {
		logger.Debugf("OpenRouter: selected model: %s (strategy: %s)", modelName, c.config.ModelStrategy)
	}
	c.mutex.Lock()
	c.lastUsedModel = modelName
	c.mutex.Unlock()
	requestData := OpenRouterRequest{
		Model: modelName,
		Messages: []OpenRouterMessage{
			{
				Role:    "system",
				Content: fmt.Sprintf("You are an expert programmer. Generate ONLY clean, functional code in %s with no explanations, comments about the code generation process, or markdown formatting. Include necessary imports and ensure the code is ready to run. When modifying existing files, preserve the structure and style while implementing the requested changes. Output raw code only. Never use markdown code blocks.", detectedLanguage),
			},
			{
				Role:    "user",
				Content: fullPrompt,
			},
		},
		Stream: false,
	}
	requestData.HTTPReferer = c.config.SiteURL
	requestData.HTTPUserAgent = c.config.SiteName
	return requestData, nil
}
// makeAPICallWithKey makes the actual HTTP request to the OpenRouter API with a specific API key
func (c *OpenRouterClient) makeAPICallWithKey(ctx context.Context, requestData OpenRouterRequest, apiKey string) (*OpenRouterResponse, error) {
	jsonBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	url := c.config.BaseURL + config.OpenRouterAPIEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", strconv.Itoa(len(jsonBody)))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", c.config.SiteURL)
	req.Header.Set("X-Title", c.config.SiteName)
	logger.Debugf("Making OpenRouter API call to %s", url)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errorResponse OpenRouterErrorResponse
		if parseErr := json.Unmarshal(body, &errorResponse); parseErr == nil {
			return nil, fmt.Errorf("OpenRouter API error: %d - %s", resp.StatusCode, errorResponse.Error.Message)
		}
		return nil, fmt.Errorf("OpenRouter API error: %d - %s", resp.StatusCode, string(body))
	}
	var response OpenRouterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in API response")
	}
	// Debug log the usage information from API response
	logger.Debugf("OpenRouter API response usage: PromptTokens=%d, CompletionTokens=%d, TotalTokens=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	return &response, nil
}
// GetLastUsedModel returns the model name that was used in the last API call
func (c *OpenRouterClient) GetLastUsedModel() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lastUsedModel
}
// OpenRouterRequest represents the request payload for OpenRouter API
type OpenRouterRequest struct {
	Model          string               `json:"model"`
	Messages       []OpenRouterMessage  `json:"messages"`
	Stream         bool                 `json:"stream"`
	HTTPReferer    string               `json:"http_referer,omitempty"`
	HTTPUserAgent  string               `json:"x-title,omitempty"`
	Temperature    float64              `json:"temperature,omitempty"`
	MaxTokens      int                  `json:"max_tokens,omitempty"`
}

// OpenRouterMessage represents a message in the conversation
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response from OpenRouter API
type OpenRouterResponse struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []OpenRouterChoice  `json:"choices"`
	Usage   OpenRouterUsage     `json:"usage"`
}

// OpenRouterChoice represents a choice in the response
type OpenRouterChoice struct {
	Index        int                 `json:"index"`
	Message      OpenRouterMessage   `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

// OpenRouterUsage represents token usage information
type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenRouterErrorResponse represents an error response
type OpenRouterErrorResponse struct {
	Error OpenRouterError `json:"error"`
}

// OpenRouterError represents an error in the response
type OpenRouterError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

// OpenRouterRateLimits represents rate limit information from the /api/v1/key endpoint
type OpenRouterRateLimits struct {
	Limit          *float64 `json:"limit"`           // Credit limit (can be null)
	LimitReset     string   `json:"limit_reset"`     // Reset type for credits
	LimitRemaining *float64 `json:"limit_remaining"` // Remaining credits
	Usage          float64  `json:"usage"`           // Total credits used
	IsFreeTier     bool     `json:"is_free_tier"`    // Whether account is on free tier
	RateLimit      struct {
		RequestsPerMinute int `json:"requests_per_minute"`
		RequestsPerDay    int `json:"requests_per_day"`
	} `json:"rate_limit,omitempty"` // Rate limit information
}

// GetRateLimits queries the OpenRouter API for current rate limit information
func (c *OpenRouterClient) GetRateLimits(ctx context.Context) (*OpenRouterRateLimits, error) {
	if c.keyManager == nil {
		return nil, fmt.Errorf("no OpenRouter API key configured")
	}

	// Get the current API key
	apiKey := c.keyManager.GetCurrentKey()
	if apiKey == "" {
		return nil, fmt.Errorf("no valid API key available")
	}

	// Build the request URL
	url := c.config.BaseURL + "/v1/key"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", c.config.SiteURL)
	req.Header.Set("X-Title", c.config.SiteName)

	logger.Debugf("Querying OpenRouter rate limits at %s", url)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter rate limits API error: %d - %s", resp.StatusCode, string(body))
	}

	var rateLimits OpenRouterRateLimits
	if err := json.Unmarshal(body, &rateLimits); err != nil {
		return nil, fmt.Errorf("failed to parse rate limits response: %w", err)
	}

	logger.Debugf("OpenRouter rate limits: usage=%.2f, remaining=%v, limit=%v, free_tier=%v",
		rateLimits.Usage,
		rateLimits.LimitRemaining,
		rateLimits.Limit,
		rateLimits.IsFreeTier)

	return &rateLimits, nil
}
