package openai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	*provider.BaseProvider
	useResponsesAPI bool
	apiKey          string
	baseURL         string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config provider.ProviderConfig) *OpenAIProvider {
	// Create HTTP client
	client := &http.Client{
		Timeout: 60 * time.Second, // TODO: Make configurable
	}

	// Extract configuration
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = config.APIKeyEnv // Try environment variable
	}
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1" // Default
	}

	return &OpenAIProvider{
		BaseProvider:    provider.NewBaseProvider("openai", config, client, log.Default()),
		useResponsesAPI: config.SupportsResponsesAPI,
		apiKey:          apiKey,
		baseURL:         baseURL,
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

// Type returns the provider type
func (p *OpenAIProvider) Type() provider.ProviderType {
	return provider.ProviderTypeOpenAI
}

// Description returns the provider description
func (p *OpenAIProvider) Description() string {
	return "OpenAI - GPT models with native API access"
}

// GetModels returns available models
func (p *OpenAIProvider) GetModels(ctx context.Context) ([]provider.Model, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("no OpenAI API key configured")
	}

	// TODO: Implement actual models API call
	// For now, return static list
	models := []provider.Model{
		{ID: "gpt-4o", Name: "GPT-4o", Provider: p.Type(), MaxTokens: 128000, SupportsStreaming: true, SupportsToolCalling: true, Description: "OpenAI's latest reasoning model"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: p.Type(), MaxTokens: 128000, SupportsStreaming: true, SupportsToolCalling: true, Description: "OpenAI's compact reasoning model"},
		{ID: "gpt-4", Name: "GPT-4", Provider: p.Type(), MaxTokens: 8192, SupportsStreaming: true, SupportsToolCalling: true, Description: "OpenAI's previous generation model"},
		{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: p.Type(), MaxTokens: 4096, SupportsStreaming: true, SupportsToolCalling: true, Description: "OpenAI's balanced model"},
		{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: p.Type(), MaxTokens: 4096, SupportsStreaming: true, SupportsToolCalling: true, Description: "OpenAI's fast, cost-effective model"},
		{ID: "gpt-3.5", Name: "GPT-3.5", Provider: p.Type(), MaxTokens: 4096, SupportsStreaming: false, SupportsToolCalling: true, Description: "OpenAI's capable model"},
	}

	return models, nil
}

// GetDefaultModel returns the default model
func (p *OpenAIProvider) GetDefaultModel() string {
	config := p.GetConfig()
	if config.DefaultModel != "" {
		return config.DefaultModel
	}
	return "gpt-4o" // Default to latest model
}

// GenerateChatCompletion generates a chat completion
func (p *OpenAIProvider) GenerateChatCompletion(
	ctx context.Context,
	options provider.GenerateOptions,
) (provider.ChatCompletionStream, error) {
	config := p.GetConfig()
	p.LogRequest("POST", config.BaseURL+"/chat/completions", map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + config.APIKey,
	}, options)

	// TODO: Implement actual API call
	// For now, return a mock response
	return &MockStream{
		chunks: []provider.ChatCompletionChunk{
			{Content: "This is a mock OpenAI response for " + options.Prompt, Done: true},
		},
	}, nil
}

// InvokeServerTool invokes a server tool (placeholder)
func (p *OpenAIProvider) InvokeServerTool(
	ctx context.Context,
	toolName string,
	params interface{},
) (interface{}, error) {
	return nil, fmt.Errorf("tool invocation not yet implemented for OpenAI provider")
}

// Authenticate handles API key authentication
func (p *OpenAIProvider) Authenticate(ctx context.Context, authConfig provider.AuthConfig) error {
	if authConfig.Method != provider.AuthMethodAPIKey {
		return fmt.Errorf("OpenAI only supports API key authentication")
	}

	// Update configuration
	newConfig := p.GetConfig()
	newConfig.APIKey = authConfig.APIKey
	newConfig.BaseURL = authConfig.BaseURL
	newConfig.DefaultModel = authConfig.DefaultModel

	return p.Configure(newConfig)
}

// IsAuthenticated checks if the provider is authenticated
func (p *OpenAIProvider) IsAuthenticated() bool {
	return p.apiKey != ""
}

// Logout handles logout (clears API key)
func (p *OpenAIProvider) Logout(ctx context.Context) error {
	newConfig := p.GetConfig()
	newConfig.APIKey = ""
	return p.Configure(newConfig)
}

// Configure updates the provider configuration
func (p *OpenAIProvider) Configure(config provider.ProviderConfig) error {
	// Validate configuration
	if config.Type != provider.ProviderTypeOpenAI {
		return fmt.Errorf("invalid provider type for OpenAI: %s", config.Type)
	}
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for OpenAI provider")
	}

	return p.BaseProvider.Configure(config)
}

// GetConfig returns the current configuration
func (p *OpenAIProvider) GetConfig() provider.ProviderConfig {
	return p.BaseProvider.GetConfig()
}

// SupportsToolCalling returns whether the provider supports tool calling
func (p *OpenAIProvider) SupportsToolCalling() bool {
	return true
}

// SupportsStreaming returns whether the provider supports streaming
func (p *OpenAIProvider) SupportsStreaming() bool {
	return true
}

// SupportsResponsesAPI returns whether the provider supports Responses API
func (p *OpenAIProvider) SupportsResponsesAPI() bool {
	return p.useResponsesAPI
}

// GetToolFormat returns the tool format used by this provider
func (p *OpenAIProvider) GetToolFormat() provider.ToolFormat {
	return provider.ToolFormatOpenAI
}

// HealthCheck performs a health check
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	return p.BaseProvider.HealthCheck(ctx)
}

// GetMetrics returns provider metrics
func (p *OpenAIProvider) GetMetrics() provider.ProviderMetrics {
	return p.BaseProvider.GetMetrics()
}

// MockStream implements ChatCompletionStream for testing
type MockStream struct {
	chunks []provider.ChatCompletionChunk
	index  int
}

func (ms *MockStream) Next() (provider.ChatCompletionChunk, error) {
	if ms.index >= len(ms.chunks) {
		return provider.ChatCompletionChunk{}, nil
	}
	chunk := ms.chunks[ms.index]
	ms.index++
	return chunk, nil
}

func (ms *MockStream) Close() error {
	ms.index = 0
	return nil
}
