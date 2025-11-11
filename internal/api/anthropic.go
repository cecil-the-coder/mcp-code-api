package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
)

// AnthropicClient handles Anthropic API interactions
type AnthropicClient struct {
	config     config.AnthropicConfig
	client     *http.Client
	keyManager *APIKeyManager
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(cfg config.AnthropicConfig) *AnthropicClient {
	// Get all API keys (single key or multiple keys)
	keys := []string{}
	if cfg.APIKey != "" {
		keys = append(keys, cfg.APIKey)
	}
	if len(cfg.APIKeys) > 0 {
		keys = append(keys, cfg.APIKeys...)
	}

	return &AnthropicClient{
		config:     cfg,
		keyManager: NewAPIKeyManager("Anthropic", keys),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateCode generates code using the Anthropic API with automatic failover
func (c *AnthropicClient) GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (string, error) {
	if c.keyManager == nil {
		return "", fmt.Errorf("no Anthropic API key configured")
	}

	// Determine language from file extension or explicit parameter
	detectedLanguage := utils.GetLanguageFromFile(outputFile, language)

	// Build the full prompt
	fullPrompt := c.buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage, contextFiles)

	// Prepare the request
	requestData := c.prepareRequest(fullPrompt, detectedLanguage)

	// Use failover to try multiple API keys if needed
	return c.keyManager.ExecuteWithFailover(func(apiKey string) (string, error) {
		// Make the API call with this specific key
		response, err := c.makeAPICallWithKey(ctx, requestData, apiKey)
		if err != nil {
			return "", err
		}

		// Extract and clean the content
		if len(response.Content) == 0 {
			return "", fmt.Errorf("no content in API response")
		}
		content := response.Content[0].Text
		cleanedContent := utils.CleanCodeResponse(content)

		return cleanedContent, nil
	})
}

// buildFullPrompt builds the complete prompt including context and existing content
func (c *AnthropicClient) buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage string, contextFiles []string) string {
	var parts []string

	// Add context files if provided
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
func (c *AnthropicClient) filterContextFiles(contextFiles []string, outputFile string) []string {
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
func (c *AnthropicClient) prepareRequest(fullPrompt, detectedLanguage string) AnthropicRequest {
	model := c.config.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022" // Default model
	}

	return AnthropicRequest{
		Model:     model,
		MaxTokens: 4096,
		System:    fmt.Sprintf("You are an expert programmer. Generate ONLY clean, functional code in %s with no explanations, comments about the code generation process, or markdown formatting. Include necessary imports and ensure the code is ready to run. When modifying existing files, preserve the structure and style while implementing the requested changes. Output raw code only. Never use markdown code blocks.", detectedLanguage),
		Messages: []AnthropicMessage{
			{
				Role:    "user",
				Content: fullPrompt,
			},
		},
	}
}

// makeAPICallWithKey makes the actual HTTP request to the Anthropic API with a specific API key
func (c *AnthropicClient) makeAPICallWithKey(ctx context.Context, requestData AnthropicRequest, apiKey string) (*AnthropicResponse, error) {
	// Serialize request
	jsonBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	url := baseURL + "/v1/messages"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	logger.Debugf("Making Anthropic API call to %s", url)

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
		var errorResponse AnthropicErrorResponse
		if parseErr := json.Unmarshal(body, &errorResponse); parseErr == nil {
			return nil, fmt.Errorf("Anthropic API error: %d - %s", resp.StatusCode, errorResponse.Error.Message)
		}
		return nil, fmt.Errorf("Anthropic API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse successful response
	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(response.Content) == 0 {
		return nil, fmt.Errorf("no content in API response")
	}

	return &response, nil
}

// AnthropicRequest represents the request payload for Anthropic API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []AnthropicMessage `json:"messages"`
}

// AnthropicMessage represents a message in the conversation
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents the response from Anthropic API
type AnthropicResponse struct {
	ID      string                   `json:"id"`
	Type    string                   `json:"type"`
	Role    string                   `json:"role"`
	Content []AnthropicContentBlock  `json:"content"`
	Model   string                   `json:"model"`
	Usage   AnthropicUsage           `json:"usage"`
}

// AnthropicContentBlock represents a content block in the response
type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicUsage represents token usage information
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicErrorResponse represents an error response
type AnthropicErrorResponse struct {
	Type  string          `json:"type"`
	Error AnthropicError  `json:"error"`
}

// AnthropicError represents an error in the response
type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
