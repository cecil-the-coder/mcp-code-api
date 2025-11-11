package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
)

// GeminiClient handles Gemini API interactions
type GeminiClient struct {
	config     config.GeminiConfig
	client     *http.Client
	keyManager *APIKeyManager
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(cfg config.GeminiConfig) *GeminiClient {
	// Note: Gemini primarily uses OAuth, but we'll check for API key too
	keys := []string{}
	if cfg.APIKey != "" {
		keys = append(keys, cfg.APIKey)
	}

	return &GeminiClient{
		config:     cfg,
		keyManager: NewAPIKeyManager("Gemini", keys),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateCode generates code using the Gemini API
func (c *GeminiClient) GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (string, error) {
	// Gemini uses OAuth tokens, not API keys
	// For now, return an error indicating OAuth is required
	return "", fmt.Errorf("Gemini requires OAuth authentication - please configure OAuth tokens")
}

// buildFullPrompt builds the complete prompt including context and existing content
func (c *GeminiClient) buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage string, contextFiles []string) string {
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
func (c *GeminiClient) filterContextFiles(contextFiles []string, outputFile string) []string {
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
