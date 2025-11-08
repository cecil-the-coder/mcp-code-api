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

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
)

// OpenRouterClient handles OpenRouter API interactions
type OpenRouterClient struct {
	config     config.OpenRouterConfig
	client     *http.Client
	keyManager *APIKeyManager
}

// NewOpenRouterClient creates a new OpenRouter client
func NewOpenRouterClient(cfg config.OpenRouterConfig) *OpenRouterClient {
	return &OpenRouterClient{
		config:     cfg,
		keyManager: NewAPIKeyManager("OpenRouter", cfg.GetAllAPIKeys()),
		client: &http.Client{
			Timeout: 60 * time.Second, // Configurable timeout
		},
	}
}

// GenerateCode generates code using the OpenRouter API with automatic failover
func (c *OpenRouterClient) GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (string, error) {
	if c.keyManager == nil {
		return "", fmt.Errorf("no OpenRouter API key configured")
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
		content := response.Choices[0].Message.Content
		cleanedContent := utils.CleanCodeResponse(content)

		return cleanedContent, nil
	})
}

// buildFullPrompt builds the complete prompt including context and existing content
func (c *OpenRouterClient) buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage string, contextFiles []string) string {
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
func (c *OpenRouterClient) filterContextFiles(contextFiles []string, outputFile string) []string {
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
func (c *OpenRouterClient) prepareRequest(fullPrompt, detectedLanguage string) OpenRouterRequest {
	requestData := OpenRouterRequest{
		Model: c.config.Model,
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

	// Add OpenRouter specific headers
	requestData.HTTPReferer = c.config.SiteURL
	requestData.HTTPUserAgent = c.config.SiteName

	return requestData
}

// makeAPICallWithKey makes the actual HTTP request to the OpenRouter API with a specific API key
func (c *OpenRouterClient) makeAPICallWithKey(ctx context.Context, requestData OpenRouterRequest, apiKey string) (*OpenRouterResponse, error) {
	// Serialize request
	jsonBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := c.config.BaseURL + config.OpenRouterAPIEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", strconv.Itoa(len(jsonBody)))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", c.config.SiteURL)
	req.Header.Set("X-Title", c.config.SiteName)

	logger.Debugf("Making OpenRouter API call to %s", url)

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
		var errorResponse OpenRouterErrorResponse
		if parseErr := json.Unmarshal(body, &errorResponse); parseErr == nil {
			return nil, fmt.Errorf("OpenRouter API error: %d - %s", resp.StatusCode, errorResponse.Error.Message)
		}
		return nil, fmt.Errorf("OpenRouter API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse successful response
	var response OpenRouterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in API response")
	}

	return &response, nil
}

// OpenRouterRequest represents the request payload for OpenRouter API
type OpenRouterRequest struct {
	Model         string              `json:"model"`
	Messages      []OpenRouterMessage `json:"messages"`
	Temperature   float64             `json:"temperature"`
	MaxTokens     int                 `json:"max_tokens,omitempty"`
	Stream        bool                `json:"stream"`
	HTTPReferer   string              `json:"http_referer,omitempty"`
	HTTPUserAgent string              `json:"http_user_agent,omitempty"`
}

// OpenRouterMessage represents a message in the conversation
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response from OpenRouter API
type OpenRouterResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenRouterChoice `json:"choices"`
	Usage   OpenRouterUsage    `json:"usage"`
}

// OpenRouterChoice represents a choice in the response
type OpenRouterChoice struct {
	Index        int               `json:"index"`
	Message      OpenRouterMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
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
	Code    string `json:"code"`
}
