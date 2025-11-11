package types

import (
	"context"
	"io"
	"net/http"
	timepkg "time"
)

// ProviderType represents the type of AI provider
type ProviderType string

const (
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeAnthropic  ProviderType = "anthropic"
	ProviderTypeGemini     ProviderType = "gemini"
	ProviderTypeQwen       ProviderType = "qwen"
	ProviderTypeCerebras   ProviderType = "cerebras"
	ProviderTypeOpenRouter ProviderType = "openrouter"
	ProviderTypeSynthetic  ProviderType = "synthetic"
	ProviderTypexAI        ProviderType = "xai"
	ProviderTypeFireworks  ProviderType = "fireworks"
	ProviderTypeDeepseek   ProviderType = "deepseek"
	ProviderTypeMistral    ProviderType = "mistral"
	ProviderTypeLMStudio   ProviderType = "lmstudio"
	ProviderTypeLlamaCpp   ProviderType = "llamacpp"
	ProviderTypeOllama     ProviderType = "ollama"
)

// AuthMethod represents the authentication method
type AuthMethod string

const (
	AuthMethodAPIKey      AuthMethod = "api_key"
	AuthMethodBearerToken AuthMethod = "bearer_token"
	AuthMethodOAuth       AuthMethod = "oauth"
	AuthMethodCustom      AuthMethod = "custom"
)

// ToolFormat represents the format used for tool calling
type ToolFormat string

const (
	ToolFormatOpenAI    ToolFormat = "openai"
	ToolFormatAnthropic ToolFormat = "anthropic"
	ToolFormatXML       ToolFormat = "xml"
	ToolFormatHermes    ToolFormat = "hermes"
	ToolFormatText      ToolFormat = "text"
)

// HealthStatus represents the health status of a provider
type HealthStatus struct {
	Healthy      bool         `json:"healthy"`
	LastChecked  timepkg.Time `json:"last_checked"`
	Message      string       `json:"message"`
	ResponseTime float64      `json:"response_time"`
	StatusCode   int          `json:"status_code"`
}

// ProviderInfo contains information about a provider
type ProviderInfo struct {
	Name           string       `json:"name"`
	Type           ProviderType `json:"type"`
	Description    string       `json:"description"`
	HealthStatus   HealthStatus `json:"health_status"`
	Models         []Model      `json:"models"`
	SupportedTools []string     `json:"supported_tools"`
	DefaultModel   string       `json:"default_model"`
}

// Model represents an AI model
type Model struct {
	ID                   string       `json:"id"`
	Name                 string       `json:"name"`
	Provider             ProviderType `json:"provider"`
	Description          string       `json:"description"`
	MaxTokens            int          `json:"max_tokens"`
	InputTokens          int          `json:"input_tokens"`
	OutputTokens         int          `json:"output_tokens"`
	SupportsStreaming    bool         `json:"supports_streaming"`
	SupportsToolCalling  bool         `json:"supports_tool_calling"`
	SupportsResponsesAPI bool         `json:"supports_responses_api"`
	Capabilities         []string     `json:"capabilities"`
	Pricing              Pricing      `json:"pricing"`
}

// Pricing contains pricing information for a model
type Pricing struct {
	InputTokenPrice  float64 `json:"input_token_price"`
	OutputTokenPrice float64 `json:"output_token_price"`
	Unit             string  `json:"unit"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CodeGenerationResult represents the result of code generation including token usage
type CodeGenerationResult struct {
	Code  string  `json:"code"`
	Usage *Usage  `json:"usage,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role       string                 `json:"role"`
	Content    string                 `json:"content"`
	ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool call
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function ToolCallFunction       `json:"function"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCallFunction represents a tool call function
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool represents an available tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ChatCompletionStream represents a streaming response
type ChatCompletionStream interface {
	Next() (ChatCompletionChunk, error)
	Close() error
}

// ChatCompletionChunk represents a chunk of a streaming response
type ChatCompletionChunk struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
	Done    bool         `json:"done"`
	Content string       `json:"content"`
	Error   string       `json:"error"`
}

// ChatChoice represents a choice in a chat completion
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
	Delta        ChatMessage `json:"delta"`
}

// GenerateOptions represents options for generating content
type GenerateOptions struct {
	Prompt         string                 `json:"prompt"`
	Context        string                 `json:"context"`
	OutputFile     string                 `json:"output_file"`
	Language       *string                `json:"language"`
	ContextFiles   []string               `json:"context_files"`
	Messages       []ChatMessage          `json:"messages"`
	MaxTokens      int                    `json:"max_tokens,omitempty"`
	Temperature    float64                `json:"temperature,omitempty"`
	Stop           []string               `json:"stop,omitempty"`
	Stream         bool                   `json:"stream"`
	Tools          []Tool                 `json:"tools,omitempty"`
	ResponseFormat string                 `json:"response_format,omitempty"`
	Timeout        timepkg.Duration       `json:"timeout,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderMetrics represents metrics for a provider
type ProviderMetrics struct {
	RequestCount    int64            `json:"request_count"`
	SuccessCount    int64            `json:"success_count"`
	ErrorCount      int64            `json:"error_count"`
	TotalLatency    timepkg.Duration `json:"total_latency"`
	AverageLatency  timepkg.Duration `json:"average_latency"`
	LastRequestTime timepkg.Time     `json:"last_request_time"`
	LastSuccessTime timepkg.Time     `json:"last_success_time"`
	LastErrorTime   timepkg.Time     `json:"last_error_time"`
	LastError       string           `json:"last_error"`
	TokensUsed      int64            `json:"tokens_used"`
	HealthStatus    HealthStatus     `json:"health_status"`
}

// ProviderConfig represents configuration for a specific provider
type ProviderConfig struct {
	Type           ProviderType           `json:"type"`
	Name           string                 `json:"name"`
	BaseURL        string                 `json:"base_url,omitempty"`
	APIKey         string                 `json:"api_key,omitempty"`
	APIKeyEnv      string                 `json:"api_key_env,omitempty"`
	DefaultModel   string                 `json:"default_model,omitempty"`
	Description    string                 `json:"description,omitempty"`
	ProviderConfig map[string]interface{} `json:"provider_config,omitempty"`

	// OAuth configuration
	OAuthConfig *OAuthConfig `json:"oauth,omitempty"`

	// Feature flags
	SupportsStreaming    bool `json:"supports_streaming"`
	SupportsToolCalling  bool `json:"supports_tool_calling"`
	SupportsResponsesAPI bool `json:"supports_responses_api"`

	// Limits and timeouts
	MaxTokens int              `json:"max_tokens,omitempty"`
	Timeout   timepkg.Duration `json:"timeout,omitempty"`

	// Tool format
	ToolFormat ToolFormat `json:"tool_format,omitempty"`
}

// OAuthConfig represents OAuth configuration
type OAuthConfig struct {
	ClientID     string       `json:"client_id"`
	ClientSecret string       `json:"client_secret"`
	RedirectURL  string       `json:"redirect_url,omitempty"`
	Scopes       []string     `json:"scopes"`
	AuthURL      string       `json:"auth_url,omitempty"`
	TokenURL     string       `json:"token_url,omitempty"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	AccessToken  string       `json:"access_token,omitempty"`
	ExpiresAt    timepkg.Time `json:"expires_at,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Method       AuthMethod   `json:"method"`
	APIKey       string       `json:"api_key,omitempty"`
	BaseURL      string       `json:"base_url,omitempty"`
	DefaultModel string       `json:"default_model,omitempty"`
	OAuthConfig  *OAuthConfig `json:"oauth,omitempty"`
}

// TokenStorage represents a token storage interface
type TokenStorage interface {
	StoreToken(key string, token *OAuthConfig) error
	RetrieveToken(key string) (*OAuthConfig, error)
	DeleteToken(key string) error
	ListTokens() ([]string, error)
}

// Options represents configuration options for a provider
type Options interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) timepkg.Duration
	GetStringSlice(key string) []string
}

// Router represents a router for provider selection
type Router interface {
	SelectProvider(prompt string, options interface{}) (Provider, error)
	GetAvailableProviders() []ProviderInfo
	GetProvider(name string) (Provider, error)
	SetPreference(providerName string) error
}

// ProviderRegistry represents a registry of providers
type ProviderRegistry interface {
	Register(provider Provider) error
	Unregister(name string) error
	Get(name string) (Provider, error)
	List() []Provider
	ListByType(providerType ProviderType) []Provider
	GetAvailable() []Provider
	GetHealthy() []Provider
}

// Logger represents a logger interface
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// HTTPClient represents an HTTP client interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
	Post(url string, bodyType string, body io.Reader) (*http.Response, error)
}

// Provider represents an AI provider - this will be moved to types
type Provider interface {
	// Basic provider information
	Name() string
	Type() ProviderType
	Description() string

	// Model management
	GetModels(ctx context.Context) ([]Model, error)
	GetDefaultModel() string

	// Authentication
	Authenticate(ctx context.Context, authConfig AuthConfig) error
	IsAuthenticated() bool
	Logout(ctx context.Context) error

	// Configuration
	Configure(config ProviderConfig) error
	GetConfig() ProviderConfig

	// Core capabilities
	GenerateChatCompletion(ctx context.Context, options GenerateOptions) (ChatCompletionStream, error)
	InvokeServerTool(ctx context.Context, toolName string, params interface{}) (interface{}, error)

	// Tool format support
	SupportsToolCalling() bool
	SupportsStreaming() bool
	SupportsResponsesAPI() bool
	GetToolFormat() ToolFormat

	// Health and metrics
	HealthCheck(ctx context.Context) error
	GetMetrics() ProviderMetrics
}
