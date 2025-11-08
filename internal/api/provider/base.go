package provider

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	name    string
	config  ProviderConfig
	client  *http.Client
	logger  *log.Logger
	mutex   sync.RWMutex
	metrics ProviderMetrics
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string, config ProviderConfig, client *http.Client, logger *log.Logger) *BaseProvider {
	return &BaseProvider{
		name:   name,
		config: config,
		client: client,
		logger: logger,
		metrics: ProviderMetrics{
			RequestCount: 0,
			SuccessCount: 0,
			ErrorCount:   0,
			TokensUsed:   0,
		},
	}
}

// Name returns the provider name
func (p *BaseProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *BaseProvider) Type() ProviderType {
	return ProviderType(p.config.Type)
}

// Description returns the provider description
func (p *BaseProvider) Description() string {
	return "Base provider implementation"
}

// Configure config stub
func (p *BaseProvider) Configure(config ProviderConfig) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	oldType := p.config.Type
	p.config = config
	p.logger.Printf("Provider %s type changed from %s to %s", p.name, oldType, config.Type)
	return nil
}

// UpdateConfig updates provider configuration
func (p *BaseProvider) UpdateConfig(config ProviderConfig) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	oldType := p.config.Type
	p.config = config
	if p.logger != nil {
		p.logger.Printf("Provider %s updated from %s to %s", p.name, oldType, config.Type)
	}
}

// GetConfig stub
func (p *BaseProvider) GetConfig() ProviderConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.config
}

// GetModels returns available models
func (p *BaseProvider) GetModels(ctx context.Context) ([]Model, error) {
	return []Model{}, nil
}

// GetDefaultModel returns default model
func (p *BaseProvider) GetDefaultModel() string {
	if p.config.DefaultModel != "" {
		return p.config.DefaultModel
	}
	return "default-model"
}

// Authenticate provider
func (p *BaseProvider) Authenticate(ctx context.Context, authConfig AuthConfig) error {
	if authConfig.APIKey != "" {
		p.config.APIKey = authConfig.APIKey
		return nil
	}
	return fmt.Errorf("authentication not implemented")
}

// IsAuthenticated checks if provider is authenticated
func (p *BaseProvider) IsAuthenticated() bool {
	return p.config.APIKey != ""
}

// Logout provider
func (p *BaseProvider) Logout(ctx context.Context) error {
	return nil
}

// SupportsToolCalling returns whether the provider supports tool calling
func (p *BaseProvider) SupportsToolCalling() bool {
	return p.config.SupportsToolCalling
}

// SupportsStreaming returns whether the provider supports streaming
func (p *BaseProvider) SupportsStreaming() bool {
	return p.config.SupportsStreaming
}

// SupportsResponsesAPI returns whether the provider supports Responses API
func (p *BaseProvider) SupportsResponsesAPI() bool {
	return p.config.SupportsResponsesAPI
}

// InvokeServerTool stub
func (p *BaseProvider) InvokeServerTool(ctx context.Context, toolName string, params interface{}) (interface{}, error) {
	return nil, fmt.Errorf("tool invocation not implemented")
}

// GenerateChatCompletion stub
func (p *BaseProvider) GenerateChatCompletion(ctx context.Context, options GenerateOptions) (ChatCompletionStream, error) {
	return &MockStream{
		chunks: []ChatCompletionChunk{
			{Content: "Mock response from " + p.name, Done: true},
		},
	}, nil
}

// GetToolFormat returns tool format
func (p *BaseProvider) GetToolFormat() ToolFormat {
	return ToolFormatOpenAI // Default
}

// HealthCheck performs a health check
func (p *BaseProvider) HealthCheck(ctx context.Context) error {
	return nil // Default to healthy
}

// GetMetrics returns provider metrics
func (p *BaseProvider) GetMetrics() ProviderMetrics {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.metrics
}

// LogRequest logs an HTTP request
func (p *BaseProvider) LogRequest(method, url string, headers map[string]string, body interface{}) {
	p.logger.Printf("Provider %s - %s %s", p.name, method, url)
	for key, value := range headers {
		p.logger.Printf("  Header: %s: %s", key, value)
	}
	if body != nil {
		p.logger.Printf("  Body: %+v", body)
	}
}

// LogResponse logs detailed response information
func (p *BaseProvider) LogResponse(resp *http.Response, duration time.Duration) {
	p.logger.Printf("Provider %s response in %v - Status: %d", p.name, duration, resp.StatusCode)
}

// MockStream implementation
type MockStream struct {
	chunks []ChatCompletionChunk
	index  int
}

func (ms *MockStream) Next() (ChatCompletionChunk, error) {
	if ms.index >= len(ms.chunks) {
		return ChatCompletionChunk{}, nil
	}
	chunk := ms.chunks[ms.index]
	ms.index++
	return chunk, nil
}

func (ms *MockStream) Close() error {
	ms.index = 0
	return nil
}

// BaseProviderStub wraps BaseProvider to implement Provider interface
type BaseProviderStub struct {
	*BaseProvider
}

func NewBaseProviderStub(name string, config ProviderConfig, client *http.Client, logger *log.Logger) *BaseProviderStub {
	base := NewBaseProvider(name, config, client, logger)
	return &BaseProviderStub{BaseProvider: base}
}

// Implement interface methods
func (b *BaseProviderStub) Name() string        { return b.name }
func (b *BaseProviderStub) Type() ProviderType  { return ProviderType(b.config.Type) }
func (b *BaseProviderStub) Description() string { return "Base provider stub" }

// Pass through to base provider
func (b *BaseProviderStub) GetModels(ctx context.Context) ([]Model, error) {
	return b.BaseProvider.GetModels(ctx)
}

func (b *BaseProviderStub) GetDefaultModel() string {
	return b.BaseProvider.GetDefaultModel()
}

func (b *BaseProviderStub) Authenticate(ctx context.Context, authConfig AuthConfig) error {
	return b.BaseProvider.Authenticate(ctx, authConfig)
}

func (b *BaseProviderStub) IsAuthenticated() bool {
	return b.BaseProvider.IsAuthenticated()
}

func (b *BaseProviderStub) Logout(ctx context.Context) error {
	return b.BaseProvider.Logout(ctx)
}

func (b *BaseProviderStub) Configure(config ProviderConfig) error {
	return b.BaseProvider.Configure(config)
}

func (b *BaseProviderStub) GetConfig() ProviderConfig {
	return b.BaseProvider.GetConfig()
}

func (b *BaseProviderStub) GenerateChatCompletion(ctx context.Context, options GenerateOptions) (ChatCompletionStream, error) {
	return b.BaseProvider.GenerateChatCompletion(ctx, options)
}

func (b *BaseProviderStub) InvokeServerTool(ctx context.Context, toolName string, params interface{}) (interface{}, error) {
	return b.BaseProvider.InvokeServerTool(ctx, toolName, params)
}

func (b *BaseProviderStub) SupportsToolCalling() bool {
	return b.BaseProvider.SupportsToolCalling()
}

func (b *BaseProviderStub) SupportsStreaming() bool {
	return b.BaseProvider.SupportsStreaming()
}

func (b *BaseProviderStub) SupportsResponsesAPI() bool {
	return b.BaseProvider.SupportsResponsesAPI()
}

func (b *BaseProviderStub) GetToolFormat() ToolFormat {
	return b.BaseProvider.GetToolFormat()
}

func (b *BaseProviderStub) HealthCheck(ctx context.Context) error {
	return b.BaseProvider.HealthCheck(ctx)
}

func (b *BaseProviderStub) GetMetrics() ProviderMetrics {
	return b.BaseProvider.GetMetrics()
}
