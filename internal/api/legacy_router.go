package api

import (
	"context"
	"fmt"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
)

// RouteAPICall routes an API call to the appropriate provider (legacy compatibility)
func RouteAPICall(ctx context.Context, cfg *config.Config, prompt string, contextFile string, outputFile string, language string, contextFiles []string) (string, error) {
	// Try providers in the preferred order
	preferredOrder := cfg.Providers.Order
	if len(preferredOrder) == 0 {
		// Default order if not specified
		preferredOrder = []string{"anthropic", "cerebras", "openrouter", "gemini"}
	}

	for _, providerName := range preferredOrder {
		// Skip if not enabled
		enabled := false
		for _, enabledProvider := range cfg.Providers.Enabled {
			if enabledProvider == providerName {
				enabled = true
				break
			}
		}
		if !enabled {
			continue
		}

		// Try each provider
		switch providerName {
		case "anthropic":
			if cfg.Providers.Anthropic != nil && cfg.Providers.Anthropic.APIKey != "" {
				client := NewAnthropicClient(*cfg.Providers.Anthropic)
				result, err := client.GenerateCode(ctx, prompt, "", outputFile, &language, contextFiles)
				if err == nil {
					return result, nil
				}
				// Continue to next provider on failure
			}
		case "cerebras":
			if cfg.Providers.Cerebras != nil && cfg.Providers.Cerebras.APIKey != "" {
				client := NewCerebrasClient(*cfg.Providers.Cerebras)
				result, err := client.GenerateCode(ctx, prompt, "", outputFile, &language, contextFiles)
				if err == nil {
					return result, nil
				}
				// Continue to next provider on failure
			}
		case "openrouter":
			if cfg.Providers.OpenRouter != nil && cfg.Providers.OpenRouter.APIKey != "" {
				client := NewOpenRouterClient(*cfg.Providers.OpenRouter)
				result, err := client.GenerateCode(ctx, prompt, "", outputFile, &language, contextFiles)
				if err == nil {
					return result, nil
				}
				// Continue to next provider on failure
			}
		case "gemini":
			// Gemini uses OAuth, skip for now
			// if cfg.Providers.Gemini != nil {
			//	client := NewGeminiClient(*cfg.Providers.Gemini)
			//	result, err := client.GenerateCode(ctx, prompt, "", outputFile, &language, contextFiles)
			//	if err == nil {
			//		return result, nil
			//	}
			// }
		}
	}

	// All providers failed
	return "", fmt.Errorf("all providers failed or no API keys configured")
}
