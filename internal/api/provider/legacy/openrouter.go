package legacy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
)

// OpenRouterProvider is a legacy adapter for existing OpenRouter client
type OpenRouterProvider struct {
	*provider.BaseProvider
	// Use existing client for now
	client OpenRouterClientInterface
}

// OpenRouterClientInterface defines the interface for the existing OpenRouter client
type OpenRouterClientInterface interface {
	GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (string, error)
}

// NewOpenRouterProvider creates a new OpenRouter provider from legacy client
func NewOpenRouterProvider(config provider.ProviderConfig, existingClient OpenRouterClientInterface) *OpenRouterProvider {
	return &OpenRouterProvider{
		BaseProvider: provider.NewBaseProvider("openrouter", config, &http.Client{
			Timeout: 60 * time.Second,
		}, log.Default()),
		client: existingClient,
	}
}

// Name returns the provider name
func (p *OpenRouterProvider) Name() string {
	return "OpenRouter"
}

// Type returns the provider type
func (p *OpenRouterProvider) Type() provider.ProviderType {
	return provider.ProviderTypeOpenRouter
}

// Description returns the provider description
func (p *OpenRouterProvider) Description() string {
	return "OpenRouter - Access to multiple AI models via unified API"
}

// GetModels returns available OpenRouter models
func (p *OpenRouterProvider) GetModels(ctx context.Context) ([]provider.Model, error) {
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	// TODO: Implement actual models API call
	// For now, return static list of popular OpenRouter models
	models := []provider.Model{
		{ID: "qwen/qwen3-coder", Name: "Qwen3 Coder", Provider: p.Type(), MaxTokens: 8192, SupportsStreaming: true, SupportsToolCalling: true, Description: "Qwen's coding-focused model"},
		{ID: "openai/gpt-4o", Name: "GPT-4o", Provider: p.Type(), MaxTokens: 128000, SupportsStreaming: true, SupportsToolCalling: true, Description: "OpenAI's latest model"},
		{ID: "anthropic/claude-3-5-sonnet", Name: "Claude 3.5 Sonnet", Provider: p.Type(), MaxTokens: 200000, SupportsStreaming: true, SupportsToolCalling: true, Description: "Anthropic's capable model"},
		{ID: "google/gemini-pro", Name: "Gemini Pro", Provider: p.Type(), MaxTokens: 30720, SupportsStreaming: true, SupportsToolCalling: true, Description: "Google's balanced model"},
		{ID: "meta-llama/llama-3-70b-instruct", Name: "Llama 3 70B", Provider: p.Type(), MaxTokens: 8192, SupportsStreaming: true, SupportsToolCalling: true, Description: "Meta's large model"},
	}

	return models, nil
}

// GetDefaultModel returns the default model
func (p *OpenRouterProvider) GetDefaultModel() string {
	config := p.GetConfig()
	if config.DefaultModel != "" {
		return config.DefaultModel
	}
	return "qwen/qwen3-coder" // Default to coding model
}

// GenerateChatCompletion generates a chat completion using the legacy client
func (p *OpenRouterProvider) GenerateChatCompletion(
	ctx context.Context,
	options provider.GenerateOptions,
) (provider.ChatCompletionStream, error) {
	config := p.GetConfig()
	p.LogRequest("POST", config.BaseURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + config.APIKey,
		"HTTP-Referer":  "https://github.com/cecil-the-coder/mcp-code-api",
		"X-Title":       "MCP Code API",
	}, options)

	// Use legacy client
	result, err := p.client.GenerateCode(
		ctx,
		options.Prompt,
		options.Context,
		options.OutputFile,
		options.Language,
		options.ContextFiles,
	)
	if err != nil {
		return nil, fmt.Errorf("legacy OpenRouter client failed: %w", err)
	}

	// Return result as a simple stream implementation
	return NewMockStream([]provider.ChatCompletionChunk{
		{Content: result, Done: true},
	}), nil
}

// InvokeServerTool invokes a server tool
func (p *OpenRouterProvider) InvokeServerTool(
	ctx context.Context,
	toolName string,
	params interface{},
) (interface{}, error) {
	return nil, fmt.Errorf("tool invocation not yet implemented for OpenRouter provider")
}

// Authenticate handles API key authentication
func (p *OpenRouterProvider) Authenticate(ctx context.Context, authConfig provider.AuthConfig) error {
	if authConfig.Method != provider.AuthMethodAPIKey {
		return fmt.Errorf("OpenRouter only supports API key authentication")
	}

	newConfig := p.GetConfig()
	newConfig.APIKey = authConfig.APIKey
	newConfig.BaseURL = authConfig.BaseURL
	newConfig.DefaultModel = authConfig.DefaultModel

	return p.Configure(newConfig)
}

// IsAuthenticated checks if the provider is authenticated
func (p *OpenRouterProvider) IsAuthenticated() bool {
	config := p.GetConfig()
	return config.APIKey != ""
}

// Logout handles logout (clears API key)
func (p *OpenRouterProvider) Logout(ctx context.Context) error {
	newConfig := p.GetConfig()
	newConfig.APIKey = ""
	return p.Configure(newConfig)
}

// Configure updates the provider configuration
func (p *OpenRouterProvider) Configure(config provider.ProviderConfig) error {
	return p.BaseProvider.Configure(config)
}

// GetConfig returns the current configuration
func (p *OpenRouterProvider) GetConfig() provider.ProviderConfig {
	return p.BaseProvider.GetConfig()
}

// SupportsToolCalling returns whether the provider supports tool calling
func (p *OpenRouterProvider) SupportsToolCalling() bool {
	return true // OpenRouter supports tool calling through various models
}

// SupportsStreaming returns whether the provider supports streaming
func (p *OpenRouterProvider) SupportsStreaming() bool {
	return true // OpenRouter supports streaming
}

// SupportsResponsesAPI returns whether the provider supports Responses API
func (p *OpenRouterProvider) SupportsResponsesAPI() bool {
	return false // OpenRouter uses standard format
}

// GetToolFormat returns the tool format used by this provider
func (p *OpenRouterProvider) GetToolFormat() provider.ToolFormat {
	return provider.ToolFormatOpenAI // OpenRouter uses OpenAI-compatible format
}

// HealthCheck performs a health check
func (p *OpenRouterProvider) HealthCheck(ctx context.Context) error {
	return p.BaseProvider.HealthCheck(ctx)
}

// GetMetrics returns provider metrics
func (p *OpenRouterProvider) GetMetrics() provider.ProviderMetrics {
	return p.BaseProvider.GetMetrics()
}
