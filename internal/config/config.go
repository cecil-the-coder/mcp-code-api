package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Config holds all configuration for the MCP server
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Name        string        `mapstructure:"name"`
	Version     string        `mapstructure:"version"`
	Description string        `mapstructure:"description"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// ProvidersConfig holds provider configuration
type ProvidersConfig struct {
	Active        string              `mapstructure:"active"`
	Primary       string              `mapstructure:"primary"`
	Order         []string            `mapstructure:"preferred_order"`
	Enabled       []string            `mapstructure:"enabled"`
	OpenAI        *OpenAIConfig       `mapstructure:"openai"`
	Anthropic     *AnthropicConfig    `mapstructure:"anthropic"`
	Gemini        *GeminiConfig       `mapstructure:"gemini"`
	Qwen          *QwenConfig         `mapstructure:"qwen"`
	Synthetic     *SyntheticConfig    `mapstructure:"synthetic"`
	Cerebras      *CerebrasConfig     `mapstructure:"cerebras"`
	OpenRouter    *OpenRouterConfig   `mapstructure:"openrouter"`
	Racing        *RacingConfig       `mapstructure:"racing"`        // Virtual provider for racing
	RacingClever  *RacingConfig       `mapstructure:"racing-clever"` // Virtual provider for clever racing
	// Alias providers (built-in)
	Aliases map[string]ProviderConfig `mapstructure:"aliases"`
	// Custom providers (user-defined)
	Custom map[string]ProviderConfig `mapstructure:"custom"`
}

// ProviderConfig represents configuration for a specific provider
type ProviderConfig struct {
	Type           string                 `json:"type"`
	Name           string                 `json:"name"`
	BaseURL        string                 `json:"base_url,omitempty"`
	APIKey         string                 `json:"api_key,omitempty"`
	APIKeyEnv      string                 `json:"api_key_env,omitempty"`
	DefaultModel   string                 `json:"default_model,omitempty"`
	Description    string                 `json:"description,omitempty"`
	ProviderConfig map[string]interface{} `json:"provider_config,omitempty"`

	// OAuth configuration
	OAuthConfig *OAuthConfig `json:"oauth,omitempty"`

	// Tool calling
	ToolFormat           *string `json:"tool_format,omitempty"`
	SupportsToolCalling  bool    `json:"supports_tool_calling"`
	SupportsStreaming    bool    `json:"supports_streaming"`
	SupportsResponsesAPI bool    `json:"supports_responses_api"`

	// Rate limiting
	MaxRequestsPerMinute int `json:"max_requests_per_minute,omitempty"`
}

// OAuthConfig represents OAuth configuration
type OAuthConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	Scopes       []string `json:"scopes"`
	TokenURL     string   `json:"token_url"`
	AuthURL      string   `json:"auth_url"`
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey          string   `mapstructure:"api_key"`
	APIKeys         []string `mapstructure:"api_keys,omitempty"` // Multiple API keys for load balancing
	BaseURL         string   `mapstructure:"base_url,omitempty"`
	Model           string   `mapstructure:"model,omitempty"`
	UseResponsesAPI bool     `mapstructure:"use_responses_api,omitempty"`
}

// AnthropicConfig holds Anthropic-specific configuration
type AnthropicConfig struct {
	DisplayName string   `mapstructure:"display_name,omitempty"` // Optional display name for provider (e.g., "z.ai")
	APIKey      string   `mapstructure:"api_key"`
	APIKeys     []string `mapstructure:"api_keys,omitempty"` // Multiple API keys for load balancing
	BaseURL     string   `mapstructure:"base_url,omitempty"`
	Model       string   `mapstructure:"model,omitempty"`

	// OAuth configuration
	ClientID     string   `mapstructure:"client_id,omitempty"`
	ClientSecret string   `mapstructure:"client_secret,omitempty"`
	RedirectURI  string   `mapstructure:"redirect_uri,omitempty"`
	Scopes       []string `mapstructure:"scopes,omitempty"`
	TokenURL     string   `mapstructure:"token_url,omitempty"`
	AuthURL      string   `mapstructure:"auth_url,omitempty"`
}

// GeminiConfig holds Gemini-specific configuration
type GeminiConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url,omitempty"`
	Model   string `mapstructure:"model,omitempty"`

	// OAuth configuration
	ClientID     string   `mapstructure:"client_id,omitempty"`
	ClientSecret string   `mapstructure:"client_secret,omitempty"`
	RedirectURI  string   `mapstructure:"redirect_uri,omitempty"`
	Scopes       []string `mapstructure:"scopes,omitempty"`
	TokenURL     string   `mapstructure:"token_url,omitempty"`
	AuthURL      string   `mapstructure:"auth_url,omitempty"`

	// OAuth tokens (populated after authentication)
	AccessToken  string    `mapstructure:"access_token,omitempty"`
	RefreshToken string    `mapstructure:"refresh_token,omitempty"`
	TokenExpiry  time.Time `mapstructure:"token_expiry,omitempty"` // RFC3339 format
	DisplayName  string    `mapstructure:"display_name,omitempty"`

	// Cloud Code API project ID (free tier users get this from server during onboarding)
	ProjectID string `mapstructure:"project_id,omitempty"`
}

// QwenConfig holds Qwen-specific configuration
type QwenConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url,omitempty"`
	Model   string `mapstructure:"model,omitempty"`

	// OAuth configuration
	ClientID     string   `mapstructure:"client_id,omitempty"`
	ClientSecret string   `mapstructure:"client_secret,omitempty"`
	RedirectURI  string   `mapstructure:"redirect_uri,omitempty"`
	Scopes       []string `mapstructure:"scopes,omitempty"`
	TokenURL     string   `mapstructure:"token_url,omitempty"`
	AuthURL      string   `mapstructure:"auth_url,omitempty"`
}

// SyntheticConfig holds Synthetic (Hugging Face) configuration
type SyntheticConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url,omitempty"`
	Model   string `mapstructure:"model,omitempty"`
}

// CerebrasConfig holds Cerebras API configuration
type CerebrasConfig struct {
	DisplayName string   `mapstructure:"display_name,omitempty"` // Optional display name for provider
	APIKey      string   `mapstructure:"api_key"`
	APIKeys     []string `mapstructure:"api_keys,omitempty"` // Multiple API keys for load balancing
	Model       string   `mapstructure:"model"`
	MaxTokens   int      `mapstructure:"max_tokens"`
	Temperature float64  `mapstructure:"temperature"`
	BaseURL     string   `mapstructure:"base_url"`
}

// OpenRouterConfig holds OpenRouter API configuration
type OpenRouterConfig struct {
	APIKey        string   `mapstructure:"api_key"`
	APIKeys       []string `mapstructure:"api_keys,omitempty"`       // Multiple API keys for load balancing
	Model         string   `mapstructure:"model,omitempty"`          // Single model (fallback if models list empty)
	Models        []string `mapstructure:"models,omitempty"`         // List of models to use
	ModelStrategy string   `mapstructure:"model_strategy,omitempty"` // Strategy: "failover", "round-robin", "random"
	FreeOnly      bool     `mapstructure:"free_only,omitempty"`      // If true, automatically append :free suffix to model names
	SiteURL       string   `mapstructure:"site_url,omitempty"`
	SiteName      string   `mapstructure:"site_name,omitempty"`
	BaseURL       string   `mapstructure:"base_url,omitempty"`
}

// RacingConfig holds configuration for racing virtual providers
type RacingConfig struct {
	Models          []string `mapstructure:"models"`                     // Provider:model strings (e.g., "openrouter:deepseek/deepseek-chat-v3.1:free")
	NumRacers       int      `mapstructure:"num_racers,omitempty"`       // How many models to race (0 = race all)
	GracePeriodMS   int      `mapstructure:"grace_period_ms,omitempty"`  // Milliseconds to wait after first win for performance profiling
	SlownessThreshold float64 `mapstructure:"slowness_threshold,omitempty"` // Multiplier for slowness detection (default 2.5)
	EnableStatePersistence bool `mapstructure:"enable_state_persistence,omitempty"` // Save model performance to disk
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	TokenStore     TokenStoreConfig               `mapstructure:"token_store,omitempty"`
	OAuthProviders map[string]OAuthProviderConfig `mapstructure:"oauth_providers,omitempty"`
}

// OAuthProviderConfig holds OAuth configuration for a specific provider
type OAuthProviderConfig struct {
	ClientID        string   `mapstructure:"client_id,omitempty"`
	ClientSecret    string   `mapstructure:"client_secret,omitempty"`
	RedirectURI     string   `mapstructure:"redirect_uri,omitempty"`
	Scopes          []string `mapstructure:"scopes,omitempty"`
	TokenURL        string   `mapstructure:"token_url,omitempty"`
	AuthURL         string   `mapstructure:"auth_url,omitempty"`
	RefreshTokenURL string   `mapstructure:"refresh_token_url,omitempty"`
}

// TokenStoreConfig holds token storage configuration
type TokenStoreConfig struct {
	Type          string `mapstructure:"type,omitempty"`
	Path          string `mapstructure:"path,omitempty"`
	EncryptionKey string `mapstructure:"encryption_key,omitempty"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	File    string `mapstructure:"file,omitempty"`
	Verbose bool   `mapstructure:"verbose"`
	Debug   bool   `mapstructure:"debug"`
}

// MetricsConfig holds metrics/monitoring configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
}

// Load loads configuration from environment variables and config files
func Load() *Config {
	// Set defaults
	viper.SetDefault("server.name", "mcp-code-api")
	viper.SetDefault("server.version", "1.0.0")
	viper.SetDefault("server.description", "MCP Code API - Multi-Provider Code Generation Server")
	viper.SetDefault("server.timeout", "60s")

	// Provider defaults
	viper.SetDefault("providers.active", "")
	viper.SetDefault("providers.primary", "")
	viper.SetDefault("providers.preferred_order", "openai,anthropic,gemini,qwen,cerebras,openrouter")
	viper.SetDefault("providers.enabled", "openai,anthropic,gemini,qwen,cerebras,openrouter")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.verbose", false)
	viper.SetDefault("logging.debug", false)

	// Metrics defaults
	viper.SetDefault("metrics.enabled", false)
	viper.SetDefault("metrics.port", 8080)
	viper.SetDefault("metrics.host", "localhost")

	// OpenAI defaults
	viper.SetDefault("providers.openai.api_key", "")
	viper.SetDefault("providers.openai.base_url", "https://api.openai.com/v1")
	viper.SetDefault("providers.openai.use_responses_api", "false")
	viper.SetDefault("providers.openai.model", "gpt-4o")

	// Anthropic defaults
	viper.SetDefault("providers.anthropic.api_key", "")
	viper.SetDefault("providers.anthropic.base_url", "https://api.anthropic.com")
	viper.SetDefault("providers.anthropic.model", "claude-3-5-sonnet-20241022")

	// Gemini defaults
	viper.SetDefault("providers.gemini.api_key", "")
	viper.SetDefault("providers.gemini.base_url", "") // Auto-detect based on auth method
	viper.SetDefault("providers.gemini.model", "gemini-2.0-flash-exp")

	// Qwen defaults
	viper.SetDefault("providers.qwen.api_key", "")
	viper.SetDefault("providers.qwen.base_url", "https://dashscope.aliyuncs.com/api/v1")
	viper.SetDefault("providers.qwen.model", "qwen-max")

	// Cerebras defaults (legacy support)
	viper.SetDefault("providers.cerebras.api_key", "")
	viper.SetDefault("providers.cerebras.base_url", "https://api.cerebras.ai")
	viper.SetDefault("providers.cerebras.model", "zai-glm-4.6")
	viper.SetDefault("providers.cerebras.temperature", 0.6)

	// OpenRouter defaults (legacy support)
	viper.SetDefault("providers.openrouter.api_key", "")
	viper.SetDefault("providers.openrouter.site_url", "https://github.com/cecil-the-coder/mcp-code-api")
	viper.SetDefault("providers.openrouter.site_name", "MCP Code API")
	viper.SetDefault("providers.openrouter.base_url", "https://openrouter.ai/api")
	viper.SetDefault("providers.openrouter.model", "qwen/qwen3-coder")
	viper.SetDefault("providers.openrouter.model_strategy", "failover") // Default: failover
	viper.SetDefault("providers.openrouter.free_only", false)

	// Racing defaults
	viper.SetDefault("providers.racing.num_racers", 0) // 0 = race all models
	viper.SetDefault("providers.racing.grace_period_ms", 500)
	viper.SetDefault("providers.racing.slowness_threshold", 2.5)
	viper.SetDefault("providers.racing.enable_state_persistence", false)

	// Racing-Clever defaults
	viper.SetDefault("providers.racing-clever.num_racers", 0) // 0 = race all models
	viper.SetDefault("providers.racing-clever.grace_period_ms", 500)
	viper.SetDefault("providers.racing-clever.slowness_threshold", 2.5)
	viper.SetDefault("providers.racing-clever.enable_state_persistence", false)

	// Auth defaults
	viper.SetDefault("auth.token_store.type", "file")
	viper.SetDefault("auth.token_store.path", "~/.mcp-code-api/tokens")
	viper.SetDefault("auth.token_store.encryption_key", "mcp-code-api-token-key")

	// Configure config file location
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config paths (viper doesn't expand $HOME, so do it manually)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(homeDir + "/.mcp-code-api")
	}
	viper.AddConfigPath(".")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found or error reading - use defaults
		// This is not a fatal error, just continue with defaults
		logger.Warnf("Failed to read config file: %v - using defaults", err)
	} else {
		logger.Infof("Successfully loaded config from: %s", viper.ConfigFileUsed())
	}

	// Configure environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CEREBRAS_MCP")

	// Legacy environment variable support for backward compatibility
	bindLegacyEnv("providers.openai.api_key", "OPENAI_API_KEY")
	bindLegacyEnv("providers.anthropic.api_key", "ANTHROPIC_API_KEY")
	bindLegacyEnv("providers.anthropic.api_key", "ANTHROPIC_AUTH_TOKEN") // Alternative token name (e.g., z.ai)
	bindLegacyEnv("providers.anthropic.base_url", "ANTHROPIC_BASE_URL") // Support custom base URLs
	bindLegacyEnv("providers.gemini.api_key", "GEMINI_API_KEY")
	bindLegacyEnv("providers.qwen.api_key", "QWEN_API_KEY")
	bindLegacyEnv("providers.cerebras.api_key", "CEREBRAS_API_KEY")
	bindLegacyEnv("providers.openrouter.api_key", "OPENROUTER_API_KEY")
	bindLegacyEnv("providers.openai.base_url", "OPENAI_BASE_URL") // Support OpenAI-compatible endpoints
	bindLegacyEnv("providers.gemini.base_url", "GEMINI_BASE_URL")
	bindLegacyEnv("providers.qwen.base_url", "QWEN_BASE_URL")
	bindLegacyEnv("providers.cerebras.base_url", "CEREBRAS_BASE_URL")
	bindLegacyEnv("providers.cerebras.model", "CEREBRAS_MODEL")
	bindLegacyEnv("providers.cerebras.max_tokens", "CEREBRAS_MAX_TOKENS")
	bindLegacyEnv("providers.cerebras.temperature", "CEREBRAS_TEMPERATURE")
	bindLegacyEnv("providers.openrouter.site_url", "OPENROUTER_SITE_URL")
	bindLegacyEnv("providers.openrouter.site_name", "OPENROUTER_SITE_NAME")
	bindLegacyEnv("providers.openrouter.base_url", "OPENROUTER_BASE_URL")

	var cfg Config

	// Configure unmarshal with custom decode hooks for time.Time
	// Compose with default hooks to preserve standard conversions
	err = viper.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			func(f, t reflect.Type, data interface{}) (interface{}, error) {
				// Handle string to time.Time conversion for RFC3339 timestamps
				if f.Kind() == reflect.String && t == reflect.TypeOf(time.Time{}) {
					return time.Parse(time.RFC3339, data.(string))
				}
				return data, nil
			},
		),
	))

	if err != nil {
		// Return default config if unmarshal fails
		logger.Warnf("Failed to unmarshal config: %v", err)
		return &Config{}
	}

	return &cfg
}

// bindLegacyEnv binds legacy environment variables to new config paths
func bindLegacyEnv(key, envVar string) {
	if value := os.Getenv(envVar); value != "" {
		if key == "providers.cerebras.max_tokens" || key == "providers.openrouter.max_tokens" {
			if intValue, err := strconv.Atoi(value); err == nil {
				viper.Set(key, intValue)
			}
		} else if key == "providers.cerebras.temperature" || key == "providers.openrouter.temperature" {
			if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
				viper.Set(key, floatValue)
			}
		} else {
			viper.Set(key, value)
		}
	}
}

// GetLogLevel returns appropriate log level
func (c *Config) GetLogLevel() string {
	if c.Logging.Debug {
		return "debug"
	}
	if c.Logging.Verbose {
		return "verbose"
	}
	return c.Logging.Level
}

// GetActiveProvider returns the currently active provider
func (c *Config) GetActiveProvider() string {
	return c.Providers.Active
}

// SetActiveProvider sets the active provider
func (c *Config) SetActiveProvider(provider string) error {
	c.Providers.Active = provider
	viper.Set("providers.active", provider)
	return nil
}

// GetProviderConfig returns configuration for a specific provider
func (c *Config) GetProviderConfig(providerType string) (*ProviderConfig, error) {
	switch providerType {
	case "openai":
		return &ProviderConfig{
			Type:                 "openai",
			Name:                 "OpenAI",
			BaseURL:              c.Providers.OpenAI.BaseURL,
			APIKey:               c.Providers.OpenAI.APIKey,
			DefaultModel:         c.Providers.OpenAI.Model,
			SupportsStreaming:    true,
			SupportsToolCalling:  true,
			SupportsResponsesAPI: c.Providers.OpenAI.UseResponsesAPI,
		}, nil
	case "anthropic":
		return &ProviderConfig{
			Type:                 "anthropic",
			Name:                 "Anthropic",
			BaseURL:              c.Providers.Anthropic.BaseURL,
			APIKey:               c.Providers.Anthropic.APIKey,
			DefaultModel:         c.Providers.Anthropic.Model,
			SupportsStreaming:    true,
			SupportsToolCalling:  true,
			SupportsResponsesAPI: false,
			OAuthConfig: &OAuthConfig{
				ClientID:     c.Providers.Anthropic.ClientID,
				ClientSecret: c.Providers.Anthropic.ClientSecret,
				RedirectURI:  c.Providers.Anthropic.RedirectURI,
				Scopes:       c.Providers.Anthropic.Scopes,
				TokenURL:     c.Providers.Anthropic.TokenURL,
				AuthURL:      c.Providers.Anthropic.AuthURL,
			},
		}, nil
	case "gemini":
		return &ProviderConfig{
			Type:                 "gemini",
			Name:                 "Gemini",
			BaseURL:              c.Providers.Gemini.BaseURL,
			APIKey:               c.Providers.Gemini.APIKey,
			DefaultModel:         c.Providers.Gemini.Model,
			SupportsStreaming:    true,
			SupportsToolCalling:  true,
			SupportsResponsesAPI: false,
			OAuthConfig: &OAuthConfig{
				ClientID:     c.Providers.Gemini.ClientID,
				ClientSecret: c.Providers.Gemini.ClientSecret,
				RedirectURI:  c.Providers.Gemini.RedirectURI,
				Scopes:       c.Providers.Gemini.Scopes,
				TokenURL:     c.Providers.Gemini.TokenURL,
				AuthURL:      c.Providers.Gemini.AuthURL,
			},
		}, nil
	case "qwen":
		return &ProviderConfig{
			Type:                 "qwen",
			Name:                 "Qwen",
			BaseURL:              c.Providers.Qwen.BaseURL,
			APIKey:               c.Providers.Qwen.APIKey,
			DefaultModel:         c.Providers.Qwen.Model,
			SupportsStreaming:    true,
			SupportsToolCalling:  true,
			SupportsResponsesAPI: false,
			OAuthConfig: &OAuthConfig{
				ClientID:     c.Providers.Qwen.ClientID,
				ClientSecret: c.Providers.Qwen.ClientSecret,
				RedirectURI:  c.Providers.Qwen.RedirectURI,
				Scopes:       c.Providers.Qwen.Scopes,
				TokenURL:     c.Providers.Qwen.TokenURL,
				AuthURL:      c.Providers.Qwen.AuthURL,
			},
		}, nil
	case "cerebras":
		return &ProviderConfig{
			Type:                 "cerebras",
			Name:                 "Cerebras",
			BaseURL:              c.Providers.Cerebras.BaseURL,
			APIKey:               c.Providers.Cerebras.APIKey,
			DefaultModel:         c.Providers.Cerebras.Model,
			SupportsStreaming:    true,
			SupportsToolCalling:  false,
			SupportsResponsesAPI: false,
		}, nil
	case "openrouter":
		return &ProviderConfig{
			Type:                 "openrouter",
			Name:                 "OpenRouter",
			BaseURL:              c.Providers.OpenRouter.BaseURL,
			APIKey:               c.Providers.OpenRouter.APIKey,
			DefaultModel:         c.Providers.OpenRouter.Model,
			SupportsStreaming:    true,
			SupportsToolCalling:  false,
			SupportsResponsesAPI: false,
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerType)
	}
}

// GetEnabledProviders returns all enabled providers
func (c *Config) GetEnabledProviders() []string {
	if c.Providers.Enabled != nil {
		return c.Providers.Enabled
	}
	return c.Providers.Order
}

// HasAnyAPIKey returns true if at least one provider has an API key configured
func (c *Config) HasAnyAPIKey() bool {
	return (c.Providers.OpenAI != nil && c.Providers.OpenAI.APIKey != "") ||
		(c.Providers.Anthropic != nil && c.Providers.Anthropic.APIKey != "") ||
		(c.Providers.Gemini != nil && c.Providers.Gemini.APIKey != "") ||
		(c.Providers.Qwen != nil && c.Providers.Qwen.APIKey != "") ||
		(c.Providers.Cerebras != nil && c.Providers.Cerebras.APIKey != "") ||
		(c.Providers.OpenRouter != nil && c.Providers.OpenRouter.APIKey != "")
}

// GetDefaultOrder returns the default provider preference order
func (c *Config) GetDefaultOrder() []string {
	if c.Providers.Order != nil {
		return c.Providers.Order
	}
	return []string{"openai", "anthropic", "gemini", "qwen", "cerebras", "openrouter"}
}

// Legacy methods for backward compatibility

// GetPrimaryProvider returns primary API provider (legacy support)
func (c *Config) GetPrimaryProvider() string {
	// Check in order of preference
	if c.Providers.OpenAI != nil && c.Providers.OpenAI.APIKey != "" {
		return "openai"
	}
	if c.Providers.Anthropic != nil && c.Providers.Anthropic.APIKey != "" {
		return "anthropic"
	}
	if c.Providers.Gemini != nil && c.Providers.Gemini.APIKey != "" {
		return "gemini"
	}
	if c.Providers.Qwen != nil && c.Providers.Qwen.APIKey != "" {
		return "qwen"
	}
	if c.Providers.Cerebras != nil && c.Providers.Cerebras.APIKey != "" {
		return "cerebras"
	}
	if c.Providers.OpenRouter != nil && c.Providers.OpenRouter.APIKey != "" {
		return "openrouter"
	}
	return ""
}

// GetFallbackProvider returns fallback API provider (legacy support)
func (c *Config) GetFallbackProvider(primary string) string {
	return ""
}

// GetAllAPIKeys returns all API keys (single + multiple) for a provider
// Prioritizes the APIKeys array if set, otherwise falls back to APIKey
func (c *CerebrasConfig) GetAllAPIKeys() []string {
	if len(c.APIKeys) > 0 {
		return c.APIKeys
	}
	if c.APIKey != "" {
		return []string{c.APIKey}
	}
	return nil
}

// GetAllAPIKeys returns all API keys for OpenRouter
func (c *OpenRouterConfig) GetAllAPIKeys() []string {
	if len(c.APIKeys) > 0 {
		return c.APIKeys
	}
	if c.APIKey != "" {
		return []string{c.APIKey}
	}
	return nil
}

// GetAllAPIKeys returns all API keys for OpenAI
func (c *OpenAIConfig) GetAllAPIKeys() []string {
	if len(c.APIKeys) > 0 {
		return c.APIKeys
	}
	if c.APIKey != "" {
		return []string{c.APIKey}
	}
	return nil
}

// GetAllAPIKeys returns all API keys for Anthropic
func (c *AnthropicConfig) GetAllAPIKeys() []string {
	if len(c.APIKeys) > 0 {
		return c.APIKeys
	}
	if c.APIKey != "" {
		return []string{c.APIKey}
	}
	return nil
}