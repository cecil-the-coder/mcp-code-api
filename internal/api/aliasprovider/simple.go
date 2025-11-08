package aliasprovider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
)

// SimpleProvider represents a minimal provider implementation
type SimpleProvider struct {
	name         string
	providerType types.ProviderType
	config       types.ProviderConfig
	client       *http.Client
}

// NewSimpleProvider creates a simple provider for alias testing
func NewSimpleProvider(name string, providerType types.ProviderType, config types.ProviderConfig) *SimpleProvider {
	return &SimpleProvider{
		name:         name,
		providerType: providerType,
		config:       config,
		client:       &http.Client{},
	}
}

// Name returns provider name
func (p *SimpleProvider) Name() string {
	return p.name
}

// Type returns provider type
func (p *SimpleProvider) Type() types.ProviderType {
	return p.providerType
}

// Description returns provider description
func (p *SimpleProvider) Description() string {
	return fmt.Sprintf("%s alias provider", p.name)
}

// GetModels returns available models
func (p *SimpleProvider) GetModels(ctx context.Context) ([]types.Model, error) {
	// Return default model for each provider type
	switch p.providerType {
	case types.ProviderTypexAI:
		return []types.Model{
			{ID: "grok-beta", Name: "Grok Beta", Provider: types.ProviderTypexAI},
			{ID: "grok-2-latest", Name: "Grok 2", Provider: types.ProviderTypexAI},
		}, nil
	case types.ProviderTypeFireworks:
		return []types.Model{
			{ID: "llama-v3p1-8b-instruct", Name: "Llama 3.1 8B Instruct", Provider: types.ProviderTypeFireworks},
			{ID: "llama-v3p1-70b-instruct", Name: "Llama 3.1 70B Instruct", Provider: types.ProviderTypeFireworks},
		}, nil
	case types.ProviderTypeDeepseek:
		return []types.Model{
			{ID: "deepseek-chat", Name: "Deepseek Chat", Provider: types.ProviderTypeDeepseek},
			{ID: "deepseek-coder", Name: "Deepseek Coder", Provider: types.ProviderTypeDeepseek},
		}, nil
	case types.ProviderTypeMistral:
		return []types.Model{
			{ID: "mistral-small-latest", Name: "Mistral Small", Provider: types.ProviderTypeMistral},
			{ID: "mistral-large-latest", Name: "Mistral Large", Provider: types.ProviderTypeMistral},
		}, nil
	default:
		return []types.Model{}, nil
	}
}

// GetDefaultModel returns default model
func (p *SimpleProvider) GetDefaultModel() string {
	switch p.providerType {
	case types.ProviderTypexAI:
		return "grok-beta"
	case types.ProviderTypeFireworks:
		return "llama-v3p1-8b-instruct"
	case types.ProviderTypeDeepseek:
		return "deepseek-chat"
	case types.ProviderTypeMistral:
		return "mistral-small-latest"
	default:
		return "default-model"
	}
}

// Support methods
func (p *SimpleProvider) SupportsToolCalling() bool {
	return true
}

func (p *SimpleProvider) SupportsStreaming() bool {
	return true
}

func (p *SimpleProvider) SupportsResponsesAPI() bool {
	return false
}

func (p *SimpleProvider) GetToolFormat() types.ToolFormat {
	return types.ToolFormatOpenAI
}

// Stub implementations for remaining interface methods
func (p *SimpleProvider) Authenticate(ctx context.Context, authConfig types.AuthConfig) error {
	return nil
}

func (p *SimpleProvider) IsAuthenticated() bool {
	return true
}

func (p *SimpleProvider) Logout(ctx context.Context) error {
	return nil
}

func (p *SimpleProvider) Configure(config types.ProviderConfig) error {
	p.config = config
	return nil
}

func (p *SimpleProvider) GetConfig() types.ProviderConfig {
	return p.config
}

func (p *SimpleProvider) GenerateChatCompletion(ctx context.Context, options types.GenerateOptions) (types.ChatCompletionStream, error) {
	return &MockStream{}, nil
}

func (p *SimpleProvider) InvokeServerTool(ctx context.Context, toolName string, params interface{}) (interface{}, error) {
	return nil, fmt.Errorf("tool calling not implemented for alias providers")
}

func (p *SimpleProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (p *SimpleProvider) GetMetrics() types.ProviderMetrics {
	return types.ProviderMetrics{
		RequestCount: 0,
		SuccessCount: 0,
		ErrorCount:   0,
	}
}

// MockStream implements ChatCompletionStream interface
type MockStream struct{}

func (m *MockStream) Next() (types.ChatCompletionChunk, error) {
	return types.ChatCompletionChunk{Done: true}, nil
}

func (m *MockStream) Close() error {
	return nil
}

// Factory functions for each alias provider type
func NewXAIProvider(config types.ProviderConfig) types.Provider {
	return NewSimpleProvider("xai", types.ProviderTypexAI, config)
}

func NewFireworksProvider(config types.ProviderConfig) types.Provider {
	return NewSimpleProvider("fireworks", types.ProviderTypeFireworks, config)
}

func NewDeepseekProvider(config types.ProviderConfig) types.Provider {
	return NewSimpleProvider("deepseek", types.ProviderTypeDeepseek, config)
}

func NewMistralProvider(config types.ProviderConfig) types.Provider {
	return NewSimpleProvider("mistral", types.ProviderTypeMistral, config)
}
