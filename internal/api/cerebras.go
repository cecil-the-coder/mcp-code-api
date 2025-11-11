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
	"time"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
)
// CerebrasClient handles Cerebras API interactions
type CerebrasClient struct {
	config     config.CerebrasConfig
	client     *http.Client
	keyManager *APIKeyManager
	lastUsage  *types.Usage
}
// NewCerebrasClient creates a new Cerebras client
func NewCerebrasClient(cfg config.CerebrasConfig) *CerebrasClient {
	return &CerebrasClient{
		config:     cfg,
		keyManager: NewAPIKeyManager("Cerebras", cfg.GetAllAPIKeys()),
		client: &http.Client{
			Timeout: 60 * time.Second, // Configurable timeout
		},
	}
}
// GenerateCode generates code using the Cerebras API with automatic failover
func (c *CerebrasClient) GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (*types.CodeGenerationResult, error) {
	if c.keyManager == nil {
		return nil, fmt.Errorf("no Cerebras API key configured")
	}
	// Determine language from file extension or explicit parameter
	detectedLanguage := utils.GetLanguageFromFile(outputFile, language)
	// Build the full prompt
	fullPrompt := c.buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage, contextFiles)
	// Prepare the request
	requestData := c.prepareRequest(fullPrompt, detectedLanguage)
	// Use failover to try multiple API keys if needed
	code, err := c.keyManager.ExecuteWithFailover(func(apiKey string) (string, error) {
		// Make the API call with this specific key
		response, err := c.makeAPICallWithKey(ctx, requestData, apiKey)
		if err != nil {
			return "", err
		}
		// Extract and clean the content
		content := response.Choices[0].Message.Content
		cleanedContent := utils.CleanCodeResponse(content)
		// Store usage information
		c.lastUsage = &types.Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		}
		logger.Debugf("Cerebras: Extracted token usage - Prompt: %d, Completion: %d, Total: %d",
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
		logger.Debugf("Cerebras: Returning result with usage - Total tokens: %d", result.Usage.TotalTokens)
	} else {
		logger.Warnf("Cerebras: Returning result with nil usage")
	}
	return result, nil
}
// buildFullPrompt builds the complete prompt including context and existing content
func (c *CerebrasClient) buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage string, contextFiles []string) string {
	var parts []string
	// Add context files if provided
	if len(contextFiles) > 0 {
		// Filter out the output file from context files to avoid duplication
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
	// Add additional context if provided
	if contextStr != "" {
		parts = append(parts, fmt.Sprintf("Context: %s", contextStr))
	}
	// Add existing file content if it exists
	if existingContent, err := utils.ReadFileContent(outputFile); err == nil && existingContent != "" {
		parts = append(parts, fmt.Sprintf("Existing file content:\n```%s\n%s\n```\n", detectedLanguage, existingContent))
	}
	// Add the main prompt
	parts = append(parts, fmt.Sprintf("Generate %s code for: %s", detectedLanguage, prompt))
	return strings.Join(parts, "\n\n")
}
// filterContextFiles filters out the output file from context files
func (c *CerebrasClient) filterContextFiles(contextFiles []string, outputFile string) []string {
	var filtered []string
	for _, file := range contextFiles {
		// Resolve paths for comparison
		contextAbs := filepath.Clean(file)
		outputAbs := filepath.Clean(outputFile)
		if contextAbs != outputAbs {
			filtered = append(filtered, file)
		}
	}
	return filtered
}
// prepareRequest prepares the API request payload
func (c *CerebrasClient) prepareRequest(fullPrompt, detectedLanguage string) CerebrasRequest {
	requestData := CerebrasRequest{
		Model: c.config.Model,
		Messages: []CerebrasMessage{
			{
				Role:    "system",
				Content: fmt.Sprintf("You are an expert programmer. Generate ONLY clean, functional code in %s with no explanations, comments about the code generation process, or markdown formatting. Include necessary imports and ensure the code is ready to run. When modifying existing files, preserve the structure and style while implementing the requested changes. Output raw code only. Never use markdown code blocks.", detectedLanguage),
			},
			{
				Role:    "user",
				Content: fullPrompt,
			},
		},
		Temperature: c.config.Temperature,
		Stream:      false,
	}
	// Add max_tokens if explicitly set
	if c.config.MaxTokens > 0 {
		requestData.MaxTokens = c.config.MaxTokens
	}
	return requestData
}
// makeAPICallWithKey makes the actual HTTP request to the Cerebras API with a specific API key
func (c *CerebrasClient) makeAPICallWithKey(ctx context.Context, requestData CerebrasRequest, apiKey string) (*CerebrasResponse, error) {
	// Serialize request
	jsonBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	// Create HTTP request
	url := c.config.BaseURL + config.CerebrasAPIEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", strconv.Itoa(len(jsonBody)))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	logger.Debugf("Making Cerebras API call to %s", url)
	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errorResponse CerebrasErrorResponse
		if parseErr := json.Unmarshal(body, &errorResponse); parseErr == nil {
			return nil, fmt.Errorf("Cerebras API error: %d - %s", resp.StatusCode, errorResponse.Error.Message)
		}
		return nil, fmt.Errorf("Cerebras API error: %d - %s", resp.StatusCode, string(body))
	}
	// Parse successful response
	var response CerebrasResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in API response")
	}
	return &response, nil
}
// CerebrasRequest represents the request payload for Cerebras API
type CerebrasRequest struct {
	Model       string            `json:"model"`
	Messages    []CerebrasMessage `json:"messages"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream"`
}
// CerebrasMessage represents a message in the conversation
type CerebrasMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
// CerebrasResponse represents the response from Cerebras API
type CerebrasResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []CerebrasChoice `json:"choices"`
	Usage   CerebrasUsage    `json:"usage"`
}
// CerebrasChoice represents a choice in the response
type CerebrasChoice struct {
	Index        int             `json:"index"`
	Message      CerebrasMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}
// CerebrasUsage represents token usage information
type CerebrasUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
// CerebrasErrorResponse represents an error response
type CerebrasErrorResponse struct {
	Error CerebrasError `json:"error"`
}
// CerebrasError represents an error in the response
type CerebrasError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}