package api

import (
	"context"
	"fmt"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
)

// RouteAPICall routes an API call to the appropriate provider (legacy compatibility)
func RouteAPICall(ctx context.Context, cfg *config.Config, prompt string, contextFile string, outputFile string, language string, contextFiles []string) (string, error) {
	// Try Cerebras first (primary provider)
	if cfg.Providers.Cerebras != nil && cfg.Providers.Cerebras.APIKey != "" {
		client := NewCerebrasClient(*cfg.Providers.Cerebras)
		result, err := client.GenerateCode(ctx, prompt, "", outputFile, &language, contextFiles)
		if err == nil {
			return result, nil
		}
		// Log Cerebras failure but continue to fallback
	}

	// Fallback to OpenRouter
	if cfg.Providers.OpenRouter != nil && cfg.Providers.OpenRouter.APIKey != "" {
		client := NewOpenRouterClient(*cfg.Providers.OpenRouter)
		result, err := client.GenerateCode(ctx, prompt, "", outputFile, &language, contextFiles)
		if err == nil {
			return result, nil
		}
	}

	// Both providers failed
	return "", fmt.Errorf("all providers failed or no API keys configured")
}
