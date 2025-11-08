package legacy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
)

// CerebrasProvider is a legacy adapter for existing Cerebras client
type CerebrasProvider struct {
	*provider.BaseProvider
	// Use existing client for now
	client CerebrasClientInterface
}

// CerebrasClientInterface defines the interface for the existing Cerebras client
type CerebrasClientInterface interface {
	GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (string, error)
}

// NewCerebrasProvider creates a new Cerebras provider from legacy client
func NewCerebrasProvider(config provider.ProviderConfig, existingClient CerebrasClientInterface) *CerebrasProvider {
	return &CerebrasProvider{
		BaseProvider: provider.NewBaseProvider("cerebras", config, &http.Client{
			Timeout: 60 * time.Second,
		}, log.Default()),
		client: existingClient,
	}
}

// Name returns the provider name
func (p *CerebrasProvider) Name() string {
	return "Cerebras"
}

// Type returns the provider type
func (p *CerebrasProvider) Type() provider.ProviderType {
	return provider.ProviderTypeCerebras
}

// Description returns the provider description
func (p *CerebrasProvider) Description() string {
	return "Cerebras - Fast inference for AI models"
}

// GetModels returns available Cerebras models
func (p *CerebrasProvider) GetModels(ctx context.Context) ([]provider.Model, error) {
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	// TODO: Implement actual models API call
	// For now, return static list based on common Cerebras models
	models := []provider.Model{
		{ID: "zai-glm-4.6", Name: "GLM-4.6", Provider: p.Type(), MaxTokens: 8192, SupportsStreaming: true, SupportsToolCalling: false, Description: "Cerebras' most capable model"},
		{ID: "llama3.1-8b", Name: "Llama 3.1 8B", Provider: p.Type(), MaxTokens: 8192, SupportsStreaming: true, SupportsToolCalling: false, Description: "Cerebras' efficient model"},
		{ID: "llama3.1-70b", Name: "Llama 3.1 70B", Provider: p.Type(), MaxTokens: 8192, SupportsStreaming: true, SupportsToolCalling: false, Description: "Cerebras' large model"},
	}

	return models, nil
}

// GetDefaultModel returns the default model
func (p *CerebrasProvider) GetDefaultModel() string {
	config := p.GetConfig()
	if config.DefaultModel != "" {
		return config.DefaultModel
	}
	return "zai-glm-4.6" // Default Cerebras model
}

// GenerateChatCompletion generates a chat completion using the legacy client
func (p *CerebrasProvider) GenerateChatCompletion(
	ctx context.Context,
	options provider.GenerateOptions,
) (provider.ChatCompletionStream, error) {
	config := p.GetConfig()
	p.LogRequest("POST", config.BaseURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + config.APIKey,
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
		return nil, fmt.Errorf("legacy Cerebras client failed: %w", err)
	}

	// Return result as a simple stream implementation
	return NewMockStream([]provider.ChatCompletionChunk{
		{Content: result, Done: true},
	}), nil
}

// InvokeServerTool invokes a server tool
func (p *CerebrasProvider) InvokeServerTool(
	ctx context.Context,
	toolName string,
	params interface{},
) (interface{}, error) {
	return nil, fmt.Errorf("tool invocation not yet implemented for Cerebras provider")
}

// Authenticate handles API key authentication
func (p *CerebrasProvider) Authenticate(ctx context.Context, authConfig provider.AuthConfig) error {
	if authConfig.Method != provider.AuthMethodAPIKey {
		return fmt.Errorf("Cerebras only supports API key authentication")
	}

	newConfig := p.GetConfig()
	newConfig.APIKey = authConfig.APIKey
	newConfig.BaseURL = authConfig.BaseURL
	newConfig.DefaultModel = authConfig.DefaultModel

	return p.Configure(newConfig)
}

// IsAuthenticated checks if the provider is authenticated
func (p *CerebrasProvider) IsAuthenticated() bool {
	config := p.GetConfig()
	return config.APIKey != ""
}

// Logout handles logout (clears API key)
func (p *CerebrasProvider) Logout(ctx context.Context) error {
	newConfig := p.GetConfig()
	newConfig.APIKey = ""
	return p.Configure(newConfig)
}

// Configure updates the provider configuration
func (p *CerebrasProvider) Configure(config provider.ProviderConfig) error {
	return p.BaseProvider.Configure(config)
}

// GetConfig returns the current configuration
func (p *CerebrasProvider) GetConfig() provider.ProviderConfig {
	return p.BaseProvider.GetConfig()
}

// SupportsToolCalling returns whether the provider supports tool calling
func (p *CerebrasProvider) SupportsToolCalling() bool {
	return false // Cerebras doesn't currently support tool calling
}

// SupportsStreaming returns whether the provider supports streaming
func (p *CerebrasProvider) SupportsStreaming() bool {
	return true // Cerebras supports streaming
}

// SupportsResponsesAPI returns whether the provider supports Responses API
func (p *CerebrasProvider) SupportsResponsesAPI() bool {
	return false // Cerebras uses standard format
}

// GetToolFormat returns the tool format used by this provider
func (p *CerebrasProvider) GetToolFormat() provider.ToolFormat {
	return provider.ToolFormatOpenAI // Default format
}

// HealthCheck performs a health check
func (p *CerebrasProvider) HealthCheck(ctx context.Context) error {
	return p.BaseProvider.HealthCheck(ctx)
}

// GetMetrics returns provider metrics
func (p *CerebrasProvider) GetMetrics() provider.ProviderMetrics {
	return p.BaseProvider.GetMetrics()
}
