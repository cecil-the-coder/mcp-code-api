package router

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
	"github.com/cecil-the-coder/mcp-code-api/internal/validation"
)

// EnhancedRouter handles routing to different AI providers with advanced features
type EnhancedRouter struct {
	config               *config.Config
	factory              *provider.DefaultProviderFactory
	providers            map[types.ProviderType]types.Provider
	healthStatus         map[types.ProviderType]*HealthStatus
	metrics              RouterMetrics
	providerMetrics      map[string]*ProviderMetricsTracker
	overallLatencyTracker *LatencyTracker // Track overall request latencies
	mutex                sync.RWMutex
	logger               *log.Logger
}

// HealthStatus represents the health status of a provider
type HealthStatus struct {
	IsHealthy    bool          `json:"IsHealthy"`
	LastChecked  time.Time     `json:"LastChecked"`
	ErrorMessage string        `json:"ErrorMessage,omitempty"`
	ResponseTime time.Duration `json:"ResponseTime"`
}

// RouterMetrics holds router performance metrics
type RouterMetrics struct {
	TotalRequests      int64 `json:"TotalRequests"`
	SuccessfulRequests int64 `json:"SuccessfulRequests"`
	FailedRequests     int64 `json:"FailedRequests"`
	FallbackAttempts   int64 `json:"FallbackAttempts"`
}

// ValidationWarningFunc is called to send validation warnings to the client
type ValidationWarningFunc func(providerName, message string)

// NewEnhancedRouter creates a new enhanced router
func NewEnhancedRouter(config *config.Config, factory *provider.DefaultProviderFactory) *EnhancedRouter {
	return &EnhancedRouter{
		config:               config,
		factory:              factory,
		providers:            make(map[types.ProviderType]types.Provider),
		healthStatus:         make(map[types.ProviderType]*HealthStatus),
		providerMetrics:      make(map[string]*ProviderMetricsTracker),
		overallLatencyTracker: NewLatencyTracker(1000), // Track last 1000 overall requests
		metrics: RouterMetrics{
			TotalRequests:      0,
			SuccessfulRequests: 0,
			FailedRequests:     0,
			FallbackAttempts:   0,
		},
		logger: log.Default(),
	}
}

// Initialize initializes the router with configured providers
func (r *EnhancedRouter) Initialize(ctx context.Context) error {
	// Only initialize providers that are enabled and have API keys configured
	for _, providerName := range r.config.Providers.Enabled {
		var apiKey string
		var model string

		// Get API key and model from config
		switch providerName {
		case "anthropic":
			if r.config.Providers.Anthropic != nil && r.config.Providers.Anthropic.APIKey != "" {
				apiKey = r.config.Providers.Anthropic.APIKey
				model = r.config.Providers.Anthropic.Model
			}
		case "cerebras":
			if r.config.Providers.Cerebras != nil {
				if r.config.Providers.Cerebras.APIKey != "" {
					apiKey = r.config.Providers.Cerebras.APIKey
				} else if len(r.config.Providers.Cerebras.APIKeys) > 0 {
					apiKey = r.config.Providers.Cerebras.APIKeys[0]
				}
				model = r.config.Providers.Cerebras.Model
			}
		case "openrouter":
			if r.config.Providers.OpenRouter != nil && r.config.Providers.OpenRouter.APIKey != "" {
				apiKey = r.config.Providers.OpenRouter.APIKey
				model = r.config.Providers.OpenRouter.Model
			}
		case "gemini":
			if r.config.Providers.Gemini != nil && (r.config.Providers.Gemini.APIKey != "" || r.config.Providers.Gemini.AccessToken != "") {
				// Support both API key and OAuth authentication
				apiKey = r.config.Providers.Gemini.APIKey
				if apiKey == "" {
					apiKey = "oauth" // Placeholder to indicate OAuth is configured
				}
				model = r.config.Providers.Gemini.Model
			}
		case "openai":
			if r.config.Providers.OpenAI != nil && r.config.Providers.OpenAI.APIKey != "" {
				apiKey = r.config.Providers.OpenAI.APIKey
				model = r.config.Providers.OpenAI.Model
			}
		case "qwen":
			if r.config.Providers.Qwen != nil && r.config.Providers.Qwen.APIKey != "" {
				apiKey = r.config.Providers.Qwen.APIKey
				model = r.config.Providers.Qwen.Model
			}
		}

		// Skip if no API key
		if apiKey == "" {
			continue
		}

		// Create provider config
		providerConfig := types.ProviderConfig{
			Type:                types.ProviderType(providerName),
			Name:                providerName,
			APIKey:              apiKey,
			DefaultModel:        model,
			SupportsStreaming:   true,
			SupportsToolCalling: true,
		}

		// Create provider
		providerType := types.ProviderType(providerName)
		provider, err := r.factory.CreateProvider(providerType, providerConfig)
		if err != nil {
			r.logger.Printf("Failed to create provider %s: %v", providerName, err)
			continue
		}

		// Store provider
		r.mutex.Lock()
		r.providers[providerType] = provider

		// Initialize health status (will be updated on first request)
		r.healthStatus[providerType] = &HealthStatus{
			IsHealthy:    true,
			LastChecked:  time.Now(),
			ErrorMessage: "",
			ResponseTime: 0,
		}
		r.mutex.Unlock()

		r.logger.Printf("✅ Provider %s initialized successfully", providerName)
	}

	r.logger.Printf("Router initialized with %d providers", len(r.providers))
	return nil
}

// GenerateCodeWithValidation generates code with validation retry and provider failover
func (r *EnhancedRouter) GenerateCodeWithValidation(
	ctx context.Context,
	prompt string,
	filePath string,
	contextFiles []string,
	validateCode bool,
	warningCallback ValidationWarningFunc,
) (string, error) {
	const maxRetriesPerProvider = 2

	// Update total requests counter
	r.mutex.Lock()
	r.metrics.TotalRequests++
	r.mutex.Unlock()

	// Try providers in the preferred order
	preferredOrder := r.config.Providers.Order
	if len(preferredOrder) == 0 {
		// Default order if not specified
		preferredOrder = []string{"anthropic", "cerebras", "openrouter", "gemini"}
	}

	logger.Debugf("=== ENHANCED ROUTER DEBUG ===")
	logger.Debugf("Preferred order: %s", strings.Join(preferredOrder, ", "))
	logger.Debugf("Enabled providers: %s", strings.Join(r.config.Providers.Enabled, ", "))
	logger.Debugf("Validation enabled: %v", validateCode)

	for _, providerName := range preferredOrder {
		// Skip if not enabled
		enabled := false
		for _, enabledProvider := range r.config.Providers.Enabled {
			if enabledProvider == providerName {
				enabled = true
				break
			}
		}
		if !enabled {
			logger.Debugf("Skipping %s (not enabled)", providerName)
			continue
		}

		logger.Debugf("Trying provider: %s", providerName)

		// Try this provider with retry logic
		result, err := r.tryProviderWithRetry(ctx, providerName, prompt, filePath, contextFiles, validateCode, maxRetriesPerProvider, warningCallback)
		if err == nil {
			logger.Debugf("%s: Success!", providerName)
			r.mutex.Lock()
			r.metrics.SuccessfulRequests++
			r.mutex.Unlock()
			return result, nil
		}

		logger.Debugf("%s: Failed after retries: %v", providerName, err)

		// Mark fallback attempt
		r.mutex.Lock()
		r.metrics.FallbackAttempts++
		r.mutex.Unlock()
	}

	// All providers failed
	r.mutex.Lock()
	r.metrics.FailedRequests++
	r.mutex.Unlock()
	return "", fmt.Errorf("all providers failed or no API keys configured")
}

// tryProviderWithRetry tries a single provider with validation retry logic
func (r *EnhancedRouter) tryProviderWithRetry(
	ctx context.Context,
	providerName string,
	originalPrompt string,
	filePath string,
	contextFiles []string,
	validateCode bool,
	maxRetries int,
	warningCallback ValidationWarningFunc,
) (string, error) {
	currentPrompt := originalPrompt

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Debugf("%s: Retry attempt %d/%d", providerName, attempt, maxRetries)
			if warningCallback != nil {
				warningCallback(providerName, fmt.Sprintf("⚠️ Validation failed, retrying with %s (attempt %d/%d)...", providerName, attempt+1, maxRetries+1))
			}
		}

		// Call the provider
		result, err := r.callProvider(ctx, providerName, currentPrompt, filePath, contextFiles)
		if err != nil {
			// Provider call failed (API error, network error, etc.)
			logger.Debugf("%s: API call failed: %v", providerName, err)
			return "", err
		}

		// Clean the result
		cleanResult := utils.CleanCodeResponse(result)

		// Validate if requested
		if validateCode && filePath != "" {
			language := validation.DetectLanguage(filePath)

			if language != validation.LanguageUnknown {
				validator := language.GetValidator()
				validationResult, err := validator.Validate(cleanResult, filePath)

				if err != nil {
					logger.Debugf("%s: Validation error: %v", providerName, err)

					// On last attempt, return error
					if attempt >= maxRetries {
						return "", fmt.Errorf("validation error after %d retries: %w", maxRetries, err)
					}

					// Retry with error feedback
					currentPrompt = fmt.Sprintf("%s\n\n🚨 PREVIOUS ATTEMPT FAILED WITH ERROR:\n%v\n\nPlease fix the code to resolve this error.", originalPrompt, err)
					continue
				}

				if !validationResult.Valid {
					logger.Debugf("%s: Validation failed with %d errors", providerName, len(validationResult.Errors))

					// Try auto-fix
					if validator.CanAutoFix() {
						logger.Debugf("%s: Attempting auto-fix...", providerName)
						if warningCallback != nil {
							warningCallback(providerName, fmt.Sprintf("⚠️ Invalid %s response, attempting auto-fix...", providerName))
						}

						fixedCode, err := validator.AutoFix(cleanResult)
						if err == nil {
							// Validate fixed code
							validationResult, err = validator.Validate(fixedCode, filePath)
							if err == nil && validationResult.Valid {
								logger.Debugf("%s: Auto-fix successful", providerName)
								if warningCallback != nil {
									warningCallback(providerName, fmt.Sprintf("✅ Auto-fix successful for %s response", providerName))
								}
                                return fixedCode, nil
							}
						}
						logger.Debugf("%s: Auto-fix failed", providerName)
					}

					// On last attempt, return error
					if attempt >= maxRetries {
						errorMsg := validation.FormatValidationErrors(validationResult.Errors, language)
						return "", fmt.Errorf("validation failed after %d retries:\n%s", maxRetries, errorMsg)
					}

					// Retry with validation feedback
					errorMsg := validation.FormatValidationErrors(validationResult.Errors, language)
					currentPrompt = fmt.Sprintf("%s\n\n🚨 PREVIOUS ATTEMPT FAILED VALIDATION:\n%s\n\nPlease fix the code to resolve these validation errors.", originalPrompt, errorMsg)
					continue
				}

				// Validation passed
				logger.Debugf("%s: Validation passed", providerName)
				return cleanResult, nil
			}
		}

		// No validation or validation not applicable
		return cleanResult, nil
	}

	return "", fmt.Errorf("max retries exceeded")
}

// callProvider calls a specific provider to generate code
func (r *EnhancedRouter) callProvider(ctx context.Context, providerName, prompt, filePath string, contextFiles []string) (string, error) {
	// Ensure provider metrics tracker exists
	r.mutex.Lock()
	if r.providerMetrics[providerName] == nil {
		r.providerMetrics[providerName] = NewProviderMetricsTracker(providerName)
	}
	tracker := r.providerMetrics[providerName]
	r.mutex.Unlock()

	// Start timing
	startTime := time.Now()
	language := ""
	var result string
	var err error
	var modelUsed string
	var tokenUsage *types.Usage

	switch providerName {
	case "anthropic":
		if r.config.Providers.Anthropic != nil && r.config.Providers.Anthropic.APIKey != "" {
			logger.Debugf("Anthropic: API key found, attempting call")
			client := api.NewAnthropicClient(*r.config.Providers.Anthropic)
			cgResult, err := client.GenerateCode(ctx, prompt, "", filePath, &language, contextFiles)
			if err == nil {
				result = cgResult.Code
				tokenUsage = cgResult.Usage
			}
			modelUsed = r.config.Providers.Anthropic.Model
		} else {
			err = fmt.Errorf("anthropic: no config or API key")
		}

	case "cerebras":
		if r.config.Providers.Cerebras != nil && (r.config.Providers.Cerebras.APIKey != "" || len(r.config.Providers.Cerebras.APIKeys) > 0) {
			logger.Debugf("Cerebras: API key found, attempting call")
			client := api.NewCerebrasClient(*r.config.Providers.Cerebras)
			cgResult, err := client.GenerateCode(ctx, prompt, "", filePath, &language, contextFiles)
			if err == nil {
				result = cgResult.Code
				tokenUsage = cgResult.Usage
			}
			modelUsed = r.config.Providers.Cerebras.Model
		} else {
			err = fmt.Errorf("cerebras: no config or API key")
		}

	case "openrouter":
		if r.config.Providers.OpenRouter != nil && r.config.Providers.OpenRouter.APIKey != "" {
			logger.Debugf("OpenRouter: API key found, attempting call")
			client := api.NewOpenRouterClient(*r.config.Providers.OpenRouter)
			cgResult, err := client.GenerateCode(ctx, prompt, "", filePath, &language, contextFiles)
			if err == nil {
				result = cgResult.Code
				tokenUsage = cgResult.Usage
			}
			modelUsed = client.GetLastUsedModel()
		} else {
			err = fmt.Errorf("openrouter: no config or API key")
		}

	case "racing":
		if r.config.Providers.Racing != nil && len(r.config.Providers.Racing.Models) > 0 {
			logger.Debugf("Racing: Starting model race with %d models", len(r.config.Providers.Racing.Models))
			racingProvider := api.NewRacingProvider(r.config.Providers.Racing, r.config)
			cgResult, err := racingProvider.GenerateCode(ctx, prompt, "", filePath, &language, contextFiles)
			if err == nil {
				result = cgResult.Code
				tokenUsage = cgResult.Usage
			}
			winner := racingProvider.GetLastWinner()
			if winner != "" {
				modelUsed = winner
			} else {
				modelUsed = "racing"
			}
		} else {
			err = fmt.Errorf("racing: no models configured")
		}

	case "racing-clever":
		if r.config.Providers.RacingClever != nil && len(r.config.Providers.RacingClever.Models) > 0 {
			logger.Debugf("Racing-Clever: Starting model race with %d models", len(r.config.Providers.RacingClever.Models))
			racingProvider := api.NewRacingProvider(r.config.Providers.RacingClever, r.config)
			cgResult, err := racingProvider.GenerateCode(ctx, prompt, "", filePath, &language, contextFiles)
			if err == nil {
				result = cgResult.Code
				tokenUsage = cgResult.Usage
			}
			winner := racingProvider.GetLastWinner()
			if winner != "" {
				modelUsed = winner
			} else {
				modelUsed = "racing-clever"
			}
		} else {
			err = fmt.Errorf("racing-clever: no models configured")
		}

	case "gemini":
		if r.config.Providers.Gemini != nil && (r.config.Providers.Gemini.APIKey != "" || r.config.Providers.Gemini.AccessToken != "") {
			logger.Debugf("Gemini: Calling API (OAuth: %v)", r.config.Providers.Gemini.AccessToken != "")
			client := api.NewGeminiClient(*r.config.Providers.Gemini)
			cgResult, err := client.GenerateCode(ctx, prompt, "", filePath, &language, contextFiles)
			if err == nil {
				result = cgResult.Code
				tokenUsage = cgResult.Usage
			}
			modelUsed = r.config.Providers.Gemini.Model
		} else {
			err = fmt.Errorf("gemini: no config or API key/OAuth")
		}

	default:
		err = fmt.Errorf("unknown provider: %s", providerName)
	}

	// Record timing and update metrics
	latency := time.Since(startTime)
	success := err == nil

	// Debug logging for token usage
	if tokenUsage != nil {
		logger.Debugf("Router: Provider %s returned tokenUsage - Total: %d", providerName, tokenUsage.TotalTokens)
	} else {
		logger.Warnf("Router: Provider %s returned nil tokenUsage", providerName)
	}

	// Update provider-level metrics
	tracker.RecordRequest(success, latency, tokenUsage)

	// Update overall latency tracking (for successful requests only)
	if success {
		r.overallLatencyTracker.Add(latency)
	}

	// Update model-level metrics (for multi-model providers)
	if success && modelUsed != "" {
		modelKey := fmt.Sprintf("%s:%s", providerName, modelUsed)
		r.mutex.Lock()
		if r.providerMetrics[modelKey] == nil {
			r.providerMetrics[modelKey] = NewModelMetricsTracker(providerName, modelUsed)
		}
		modelTracker := r.providerMetrics[modelKey]
		r.mutex.Unlock()

		if tokenUsage != nil {
			logger.Debugf("Router: Recording model metrics for %s with tokenUsage - Total: %d", modelKey, tokenUsage.TotalTokens)
		} else {
			logger.Warnf("Router: Recording model metrics for %s with nil tokenUsage", modelKey)
		}
		modelTracker.RecordRequest(success, latency, tokenUsage)
		logger.Debugf("Recorded metrics for model: %s (key: %s)", modelUsed, modelKey)
	}

	// Update health status
	r.mutex.Lock()
	providerType := types.ProviderType(providerName)
	if r.healthStatus[providerType] == nil {
		r.healthStatus[providerType] = &HealthStatus{}
	}
	r.healthStatus[providerType].IsHealthy = success
	r.healthStatus[providerType].LastChecked = time.Now()
	r.healthStatus[providerType].ResponseTime = latency
	if err != nil {
		r.healthStatus[providerType].ErrorMessage = err.Error()
	} else {
		r.healthStatus[providerType].ErrorMessage = ""
	}
	r.mutex.Unlock()

	return result, err
}

// GenerateCode routes an API call to the appropriate provider (legacy method without validation)
func (r *EnhancedRouter) GenerateCode(ctx context.Context, prompt, contextFile, outputFile, language string, contextFiles []string) (string, error) {
	// Use the new validation method with validation disabled
	return r.GenerateCodeWithValidation(ctx, prompt, outputFile, contextFiles, false, nil)
}

// GetMetrics returns a copy of the current router metrics (thread-safe)
func (r *EnhancedRouter) GetMetrics() RouterMetrics {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return RouterMetrics{
		TotalRequests:      r.metrics.TotalRequests,
		SuccessfulRequests: r.metrics.SuccessfulRequests,
		FailedRequests:     r.metrics.FailedRequests,
		FallbackAttempts:   r.metrics.FallbackAttempts,
	}
}

// GetHealthStatus returns a copy of the health status for all providers (thread-safe)
func (r *EnhancedRouter) GetHealthStatus() map[string]*HealthStatus {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*HealthStatus)
	for providerType, status := range r.healthStatus {
		result[string(providerType)] = &HealthStatus{
			IsHealthy:    status.IsHealthy,
			LastChecked:  status.LastChecked,
			ErrorMessage: status.ErrorMessage,
			ResponseTime: status.ResponseTime,
		}
	}

	return result
}

// GetProviderMetrics returns detailed metrics for all providers (thread-safe)
// Returns all enabled providers, even if they haven't been used yet
func (r *EnhancedRouter) GetProviderMetrics() map[string]ProviderMetrics {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]ProviderMetrics)

	// First, add all enabled providers (even if not used yet)
	for _, providerName := range r.config.Providers.Enabled {
		// Check if provider has an API key configured (or is a virtual provider)
		hasAPIKey := false
		switch providerName {
		case "anthropic":
			hasAPIKey = r.config.Providers.Anthropic != nil && r.config.Providers.Anthropic.APIKey != ""
		case "cerebras":
			hasAPIKey = r.config.Providers.Cerebras != nil && (r.config.Providers.Cerebras.APIKey != "" || len(r.config.Providers.Cerebras.APIKeys) > 0)
		case "openrouter":
			hasAPIKey = r.config.Providers.OpenRouter != nil && r.config.Providers.OpenRouter.APIKey != ""
		case "gemini":
			hasAPIKey = r.config.Providers.Gemini != nil && (r.config.Providers.Gemini.APIKey != "" || r.config.Providers.Gemini.AccessToken != "")
		case "openai":
			hasAPIKey = r.config.Providers.OpenAI != nil && r.config.Providers.OpenAI.APIKey != ""
		case "qwen":
			hasAPIKey = r.config.Providers.Qwen != nil && r.config.Providers.Qwen.APIKey != ""
		case "racing":
			// Virtual provider - check if models are configured
			hasAPIKey = r.config.Providers.Racing != nil && len(r.config.Providers.Racing.Models) > 0
		case "racing-clever":
			// Virtual provider - check if models are configured
			hasAPIKey = r.config.Providers.RacingClever != nil && len(r.config.Providers.RacingClever.Models) > 0
		}

		if !hasAPIKey {
			continue
		}

		// If provider has been used, get its metrics
		if tracker, exists := r.providerMetrics[providerName]; exists {
			result[providerName] = tracker.GetMetrics()
		} else {
			// Provider not used yet - create empty metrics
			result[providerName] = ProviderMetrics{
				Name:     providerName,
				IsModel:  false,
			}
		}
	}

	// Add any additional model-level metrics (for multi-model providers)
	for key, tracker := range r.providerMetrics {
		if _, exists := result[key]; !exists {
			// This is a model-level metric (key format: "provider:model")
			result[key] = tracker.GetMetrics()
		}
	}

	return result
}

// OverallLatencyMetrics represents overall latency percentiles
type OverallLatencyMetrics struct {
	MinLatency time.Duration `json:"MinLatency"`
	P50Latency time.Duration `json:"P50Latency"`
	P95Latency time.Duration `json:"P95Latency"`
	P99Latency time.Duration `json:"P99Latency"`
	MaxLatency time.Duration `json:"MaxLatency"`
}

// GetOverallLatencyMetrics returns overall latency percentiles for all requests (thread-safe)
func (r *EnhancedRouter) GetOverallLatencyMetrics() OverallLatencyMetrics {
	min, p50, p95, p99, max := r.overallLatencyTracker.GetPercentiles()
	return OverallLatencyMetrics{
		MinLatency: min,
		P50Latency: p50,
		P95Latency: p95,
		P99Latency: p99,
		MaxLatency: max,
	}
}