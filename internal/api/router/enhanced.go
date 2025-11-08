package router

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
	"github.com/cecil-the-coder/mcp-code-api/internal/config"
)

// EnhancedRouter handles routing to different AI providers with advanced features
type EnhancedRouter struct {
	config       *config.Config
	factory      *provider.DefaultProviderFactory
	providers    map[types.ProviderType]types.Provider
	healthStatus map[types.ProviderType]*HealthStatus
	metrics      RouterMetrics
	mutex        sync.RWMutex
	logger       *log.Logger
}

// HealthStatus represents the health status of a provider
type HealthStatus struct {
	IsHealthy    bool          `json:"is_healthy"`
	LastChecked  time.Time     `json:"last_checked"`
	ErrorMessage string        `json:"error_message,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
}

// RouterMetrics holds router performance metrics
type RouterMetrics struct {
	TotalRequests      int64 `json:"total_requests"`
	SuccessfulRequests int64 `json:"successful_requests"`
	FailedRequests     int64 `json:"failed_requests"`
	FallbackAttempts   int64 `json:"fallback_attempts"`
}

// NewEnhancedRouter creates a new enhanced router
func NewEnhancedRouter(config *config.Config, factory *provider.DefaultProviderFactory) *EnhancedRouter {
	return &EnhancedRouter{
		config:       config,
		factory:      factory,
		providers:    make(map[types.ProviderType]types.Provider),
		healthStatus: make(map[types.ProviderType]*HealthStatus),
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
	// Use default enabled providers (Phase 1)
	enabledProviders := []string{"openai", "anthropic", "gemini", "qwen", "cerebras", "openrouter"}

	// Initialize each provider with mock config
	for _, providerName := range enabledProviders {
		// Create mock config
		mockConfig := types.ProviderConfig{
			Type:                types.ProviderType(providerName),
			Name:                providerName,
			APIKey:              "test-key",
			DefaultModel:        "test-model",
			SupportsStreaming:   true,
			SupportsToolCalling: true,
		}

		// Create provider directly with factory
		providerType := types.ProviderType(providerName)
		provider, err := r.factory.CreateProvider(providerType, mockConfig)
		if err != nil {
			r.logger.Printf("Failed to create provider %s: %v", providerName, err)
			continue
		}

		// Store provider
		r.mutex.Lock()
		r.providers[providerType] = provider

		// Initialize health status
		r.healthStatus[providerType] = &HealthStatus{
			IsHealthy:    true,
			LastChecked:  time.Now(),
			ErrorMessage: "Initialized with mock config",
			ResponseTime: 0,
		}
		r.mutex.Unlock()

		r.logger.Printf("✅ Provider %s initialized successfully", providerName)
	}

	r.logger.Printf("Router initialized with %d providers", len(r.providers))
	return nil
}
