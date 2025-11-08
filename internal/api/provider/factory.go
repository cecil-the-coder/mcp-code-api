package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/aliasinit"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
)

// DefaultProviderFactory is the default factory implementation
type DefaultProviderFactory struct {
	providers map[types.ProviderType]func(types.ProviderConfig) types.Provider
	mutex     sync.RWMutex
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *DefaultProviderFactory {
	return &DefaultProviderFactory{
		providers: make(map[types.ProviderType]func(types.ProviderConfig) types.Provider),
		mutex:     sync.RWMutex{},
	}
}

// RegisterProvider registers a new provider type
func (f *DefaultProviderFactory) RegisterProvider(providerType types.ProviderType, factoryFunc func(types.ProviderConfig) types.Provider) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.providers[providerType] = factoryFunc
}

// CreateProvider creates a provider instance
func (f *DefaultProviderFactory) CreateProvider(providerType types.ProviderType, config types.ProviderConfig) (types.Provider, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	factoryFunc, exists := f.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider type %s not registered", providerType)
	}

	return factoryFunc(config), nil
}

// GetSupportedProviders returns all supported provider types
func (f *DefaultProviderFactory) GetSupportedProviders() []types.ProviderType {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	var providerTypes []types.ProviderType
	for providerType := range f.providers {
		providerTypes = append(providerTypes, providerType)
	}

	return providerTypes
}

// InitializeDefaultProviders registers all default providers
func InitializeDefaultProviders(factory *DefaultProviderFactory) {
	// Register core providers (will be implemented in subsequent phases)
	// These are placeholders for now
	factory.RegisterProvider(types.ProviderTypeOpenAI, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "openai", providerType: types.ProviderTypeOpenAI, config: config}
	})
	factory.RegisterProvider(types.ProviderTypeAnthropic, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "anthropic", providerType: types.ProviderTypeAnthropic, config: config}
	})
	factory.RegisterProvider(types.ProviderTypeGemini, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "gemini", providerType: types.ProviderTypeGemini, config: config}
	})
	factory.RegisterProvider(types.ProviderTypeQwen, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "qwen", providerType: types.ProviderTypeQwen, config: config}
	})
	factory.RegisterProvider(types.ProviderTypeCerebras, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "cerebras", providerType: types.ProviderTypeCerebras, config: config}
	})

	// Register OpenRouter provider
	factory.RegisterProvider(types.ProviderTypeOpenRouter, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "openrouter", providerType: types.ProviderTypeOpenRouter, config: config}
	})

	// Register alias providers using registry package to break circular import (Phase 3 implementation)
	aliasinit.RegisterAliasProviders(factory)

	// Register local model providers (Phase 4 implementation)
	factory.RegisterProvider(types.ProviderTypeLMStudio, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "lmstudio", providerType: types.ProviderTypeLMStudio, config: config}
	})
	factory.RegisterProvider(types.ProviderTypeLlamaCpp, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "llamacpp", providerType: types.ProviderTypeLlamaCpp, config: config}
	})
	factory.RegisterProvider(types.ProviderTypeOllama, func(config types.ProviderConfig) types.Provider {
		return &SimpleProviderStub{name: "ollama", providerType: types.ProviderTypeOllama, config: config}
	})
}

// SimpleProviderStub implements types.Provider interface
type SimpleProviderStub struct {
	name         string
	providerType types.ProviderType
	config       types.ProviderConfig
}

// Interface implementations for SimpleProviderStub
func (p *SimpleProviderStub) Name() string             { return p.name }
func (p *SimpleProviderStub) Type() types.ProviderType { return p.providerType }
func (p *SimpleProviderStub) Description() string      { return fmt.Sprintf("%s provider", p.name) }
func (p *SimpleProviderStub) GetModels(ctx context.Context) ([]types.Model, error) {
	return []types.Model{}, nil
}
func (p *SimpleProviderStub) GetDefaultModel() string         { return "default-model" }
func (p *SimpleProviderStub) SupportsToolCalling() bool       { return true }
func (p *SimpleProviderStub) SupportsStreaming() bool         { return true }
func (p *SimpleProviderStub) SupportsResponsesAPI() bool      { return false }
func (p *SimpleProviderStub) GetToolFormat() types.ToolFormat { return types.ToolFormatOpenAI }
func (p *SimpleProviderStub) Authenticate(ctx context.Context, authConfig types.AuthConfig) error {
	return nil
}
func (p *SimpleProviderStub) IsAuthenticated() bool            { return true }
func (p *SimpleProviderStub) Logout(ctx context.Context) error { return nil }
func (p *SimpleProviderStub) Configure(config types.ProviderConfig) error {
	p.config = config
	return nil
}
func (p *SimpleProviderStub) GetConfig() types.ProviderConfig { return p.config }
func (p *SimpleProviderStub) GenerateChatCompletion(ctx context.Context, options types.GenerateOptions) (types.ChatCompletionStream, error) {
	return &FactoryMockStream{}, nil
}
func (p *SimpleProviderStub) InvokeServerTool(ctx context.Context, toolName string, params interface{}) (interface{}, error) {
	return nil, fmt.Errorf("tool calling not implemented")
}
func (p *SimpleProviderStub) HealthCheck(ctx context.Context) error { return nil }
func (p *SimpleProviderStub) GetMetrics() types.ProviderMetrics {
	return types.ProviderMetrics{RequestCount: 0, SuccessCount: 0, ErrorCount: 0}
}

// FactoryMockStream implements types.ChatCompletionStream
type FactoryMockStream struct{}

func (m *FactoryMockStream) Next() (types.ChatCompletionChunk, error) {
	return types.ChatCompletionChunk{Done: true}, nil
}
func (m *FactoryMockStream) Close() error {
	return nil
}

// ValidateProviderConfig validates a provider configuration
func ValidateProviderConfig(config types.ProviderConfig) error {
	if config.Type == "" {
		return fmt.Errorf("provider type is required")
	}
	if config.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if config.APIKey == "" && config.OAuthConfig == nil {
		return fmt.Errorf("either api_key or oauth configuration is required")
	}
	return nil
}

// CreateProviderFromConfig creates a provider from configuration map
func CreateProviderFromConfig(factory *DefaultProviderFactory, configMap map[string]interface{}) (types.Provider, error) {
	// Extract basic configuration
	providerType, ok := configMap["type"].(string)
	if !ok {
		return nil, fmt.Errorf("provider type is required")
	}

	name, ok := configMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("provider name is required")
	}

	// Build provider config
	config := types.ProviderConfig{
		Type:                 types.ProviderType(providerType),
		Name:                 name,
		APIKey:               getString(configMap, "api_key"),
		BaseURL:              getString(configMap, "base_url"),
		DefaultModel:         getString(configMap, "default_model"),
		Description:          getString(configMap, "description"),
		SupportsStreaming:    getBool(configMap, "supports_streaming"),
		SupportsToolCalling:  getBool(configMap, "supports_tool_calling"),
		SupportsResponsesAPI: getBool(configMap, "supports_responses_api"),
	}

	// Handle OAuth configuration
	if oauthConfig, ok := configMap["oauth"].(map[string]interface{}); ok {
		config.OAuthConfig = &types.OAuthConfig{
			ClientID:     getString(oauthConfig, "client_id"),
			ClientSecret: getString(oauthConfig, "client_secret"),
			RedirectURL:  getString(oauthConfig, "redirect_url"),
			Scopes:       getStringSlice(oauthConfig, "scopes"),
		}
	}

	// Create provider using factory
	return factory.CreateProvider(types.ProviderType(providerType), config)
}

// Helper functions for config parsing
func getString(configMap map[string]interface{}, key string) string {
	if val, ok := configMap[key].(string); ok {
		return val
	}
	return ""
}

func getBool(configMap map[string]interface{}, key string) bool {
	if val, ok := configMap[key].(bool); ok {
		return val
	}
	return false
}

func getStringSlice(configMap map[string]interface{}, key string) []string {
	if val, ok := configMap[key].([]interface{}); ok {
		var result []string
		for _, v := range val {
			if str, ok := v.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
