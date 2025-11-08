package qwen

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
)

// QwenProvider implements Provider interface for Qwen (Tongji AI)
type QwenProvider struct {
	*provider.BaseProvider
	authConfig *provider.OAuthConfig
}

// NewQwenProvider creates a new Qwen provider
func NewQwenProvider(config provider.ProviderConfig) *QwenProvider {
	// Extract OAuth config
	authConfig := config.OAuthConfig
	if authConfig == nil {
		authConfig = &provider.OAuthConfig{
			// Default to API key for now
			// TODO: Implement Qwen OAuth flow
		}
	}

	return &QwenProvider{
		BaseProvider: provider.NewBaseProvider("qwen", config, &http.Client{
			Timeout: 60 * time.Second, // TODO: Make configurable
		}, log.Default()),
		authConfig: authConfig,
	}
}

// Name returns the provider name
func (p *QwenProvider) Name() string {
	return "Qwen"
}

// Type returns the provider type
func (p *QwenProvider) Type() provider.ProviderType {
	return provider.ProviderTypeQwen
}

// Description returns the provider description
func (p *QwenProvider) Description() string {
	return "Qwen (Tongji AI) models with OAuth support"
}

// GetModels returns available Qwen models
func (p *QwenProvider) GetModels(ctx context.Context) ([]provider.Model, error) {
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	// TODO: Implement actual Qwen models API call
	// For now, return static list
	models := []provider.Model{
		{ID: "qwen-max", Name: "Qwen Max", Provider: p.Type(), MaxTokens: 8192, SupportsStreaming: true, SupportsToolCalling: true, Description: "Qwen's most capable model"},
		{ID: "qwen-plus", Name: "Qwen Plus", Provider: p.Type(), MaxTokens: 32768, SupportsStreaming: true, SupportsToolCalling: true, Description: "Qwen's balanced model"},
		{ID: "qwen-turbo", Name: "Qwen Turbo", Provider: p.Type(), MaxTokens: 6144, SupportsStreaming: true, SupportsToolCalling: true, Description: "Qwen's fastest model"},
		{ID: "qwen", Name: "Qwen 7B Chat", Provider: p.Type(), MaxTokens: 6144, SupportsStreaming: true, SupportsToolCalling: true, Description: "Qwen's capable model"},
	}
	return models, nil
}

// GetDefaultModel returns the default model
func (p *QwenProvider) GetDefaultModel() string {
	config := p.GetConfig()
	if config.DefaultModel != "" {
		return config.DefaultModel
	}
	return "qwen-max" // Default to latest capable model
}

// GenerateChatCompletion generates a chat completion
func (p *QwenProvider) GenerateChatCompletion(
	ctx context.Context,
	options provider.GenerateOptions,
) (provider.ChatCompletionStream, error) {
	config := p.GetConfig()
	p.LogRequest("POST", config.BaseURL+"/v1/chat/completions", map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + config.APIKey,
	}, options)

	// TODO: Implement actual Qwen API call
	// For now, return a mock response
	return &MockStream{
		chunks: []provider.ChatCompletionChunk{
			{Content: "This is a mock Qwen response for " + options.Prompt, Done: true},
		},
	}, nil
}

// InvokeServerTool invokes a server tool
func (p *QwenProvider) InvokeServerTool(
	ctx context.Context,
	toolName string,
	params interface{},
) (interface{}, error) {
	return nil, fmt.Errorf("tool invocation not yet implemented for Qwen provider")
}

// Authenticate handles authentication (API key and OAuth)
func (p *QwenProvider) Authenticate(ctx context.Context, authConfig provider.AuthConfig) error {
	if authConfig.Method == provider.AuthMethodAPIKey {
		// Handle API key authentication
		if authConfig.APIKey == "" {
			return fmt.Errorf("API key is required")
		}

		newConfig := p.GetConfig()
		newConfig.APIKey = authConfig.APIKey
		return p.Configure(newConfig)
	}

	// Handle OAuth authentication
	if authConfig.Method == provider.AuthMethodOAuth {
		// TODO: Implement Qwen OAuth flow
		if p.authConfig == nil || p.authConfig.ClientID == "" || p.authConfig.ClientSecret == "" {
			return fmt.Errorf("OAuth config is required")
		}

		// For now, just validate config
		return fmt.Errorf("Qwen OAuth not yet implemented")
	}

	return fmt.Errorf("unknown authentication method: %s", authConfig.Method)
}

// IsAuthenticated checks if the provider is authenticated
func (p *QwenProvider) IsAuthenticated() bool {
	config := p.GetConfig()
	if config.APIKey != "" {
		return true
	}

	if p.authConfig != nil && p.authConfig.ClientID != "" {
		// TODO: Implement token validation
		return true // Temporary
	}
	return false
}

// Logout handles logout (clears API key and OAuth token)
func (p *QwenProvider) Logout(ctx context.Context) error {
	newConfig := p.GetConfig()
	newConfig.APIKey = ""
	newConfig.OAuthConfig = nil
	return p.Configure(newConfig)
}

// Configure updates the provider configuration
func (p *QwenProvider) Configure(config provider.ProviderConfig) error {
	return p.BaseProvider.Configure(config)
}

// GetConfig returns the current configuration
func (p *QwenProvider) GetConfig() provider.ProviderConfig {
	config := p.BaseProvider.GetConfig()
	config.OAuthConfig = p.authConfig
	return config
}

// SupportsToolCalling returns whether the provider supports tool calling
func (p *QwenProvider) SupportsToolCalling() bool {
	return true
}

// SupportsStreaming returns whether the provider supports streaming
func (p *QwenProvider) SupportsStreaming() bool {
	return true
}

// SupportsResponsesAPI returns whether the provider supports Responses API
func (p *QwenProvider) SupportsResponsesAPI() bool {
	return false
}

// GetToolFormat returns the tool format used by this provider
func (p *QwenProvider) GetToolFormat() provider.ToolFormat {
	// Qwen uses OpenAI-compatible tool format
	return provider.ToolFormatOpenAI
}

// HealthCheck performs a health check
func (p *QwenProvider) HealthCheck(ctx context.Context) error {
	return p.BaseProvider.HealthCheck(ctx)
}

// GetMetrics returns provider metrics
func (p *QwenProvider) GetMetrics() provider.ProviderMetrics {
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
