package aliasinit

import (
	"github.com/cecil-the-coder/mcp-code-api/internal/api/aliasprovider"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
)

// ProviderRegistry represents an interface for registering providers
type ProviderRegistry interface {
	RegisterProvider(providerType types.ProviderType, providerFunc func(types.ProviderConfig) types.Provider)
}

// RegisterAliasProviders registers all alias providers with the given factory
// This function breaks the circular import by having the init package handle the registration
func RegisterAliasProviders(factory ProviderRegistry) {
	// Register xAI alias providers
	factory.RegisterProvider(types.ProviderTypexAI, func(config types.ProviderConfig) types.Provider {
		return aliasprovider.NewXAIProvider(config)
	})

	// Register Fireworks alias providers
	factory.RegisterProvider(types.ProviderTypeFireworks, func(config types.ProviderConfig) types.Provider {
		return aliasprovider.NewFireworksProvider(config)
	})

	// Register Deepseek alias providers
	factory.RegisterProvider(types.ProviderTypeDeepseek, func(config types.ProviderConfig) types.Provider {
		return aliasprovider.NewDeepseekProvider(config)
	})

	// Register Mistral alias providers
	factory.RegisterProvider(types.ProviderTypeMistral, func(config types.ProviderConfig) types.Provider {
		return aliasprovider.NewMistralProvider(config)
	})
}

// GetAvailableAliasProviders returns a list of available alias provider types
func GetAvailableAliasProviders() []types.ProviderType {
	return []types.ProviderType{
		types.ProviderTypexAI,
		types.ProviderTypeFireworks,
		types.ProviderTypeDeepseek,
		types.ProviderTypeMistral,
	}
}

// IsAliasProvider checks if the given provider type is an alias provider
func IsAliasProvider(providerType types.ProviderType) bool {
	for _, aliasType := range GetAvailableAliasProviders() {
		if aliasType == providerType {
			return true
		}
	}
	return false
}
