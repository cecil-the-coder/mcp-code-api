package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/auth"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/tools"
)

// GeminiProvider implements Provider interface for Google Gemini
type GeminiProvider struct {
	*provider.BaseProvider
	authConfig  *provider.OAuthConfig
	oauthAuth   *auth.GeminiOAuthAuthenticator
	toolManager tools.ToolFormatManager
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(config provider.ProviderConfig) *GeminiProvider {
	// Create token storage
	storage := auth.NewMemoryTokenStorage() // Use memory storage for now

	// Initialize OAuth authenticator
	oauthAuth := auth.NewGeminiOAuthAuthenticator(storage)

	// Initialize tool manager
	toolManager := tools.NewToolFormatManager()

	// Extract OAuth config
	authConfig := config.OAuthConfig
	if authConfig == nil {
		authConfig = &provider.OAuthConfig{
			// Default empty config - will be populated during authentication
		}
	}

	p := &GeminiProvider{
		BaseProvider: provider.NewBaseProvider("gemini", config, &http.Client{
			Timeout: 60 * time.Second, // TODO: Make configurable
		}, log.Default()),
		authConfig:  authConfig,
		oauthAuth:   oauthAuth,
		toolManager: toolManager,
	}

	// OAuth configuration will be set during authentication

	return p
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return "Gemini"
}

// Type returns the provider type
func (p *GeminiProvider) Type() provider.ProviderType {
	return provider.ProviderTypeGemini
}

// Description returns the provider description
func (p *GeminiProvider) Description() string {
	return "Google Gemini models with OAuth support"
}

// GetModels returns available Gemini models
func (p *GeminiProvider) GetModels(ctx context.Context) ([]provider.Model, error) {
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	// TODO: Implement actual Gemini models API call
	// For now, return static list
	models := []provider.Model{
		{ID: "gemini-2.0-flash-exp", Name: "Gemini 2.0 Flash Experimental", Provider: p.Type(), MaxTokens: 1048576, SupportsStreaming: true, SupportsToolCalling: true, Description: "Google's latest experimental flash model"},
		{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", Provider: p.Type(), MaxTokens: 30720, SupportsStreaming: true, SupportsToolCalling: true, Description: "Google's most capable model"},
		{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Provider: p.Type(), MaxTokens: 1048576, SupportsStreaming: true, SupportsToolCalling: true, Description: "Google's fastest model"},
		{ID: "gemini-1.0-pro", Name: "Gemini 1.0 Pro", Provider: p.Type(), MaxTokens: 512000, SupportsStreaming: true, SupportsToolCalling: true, Description: "Google's balanced model"},
		{ID: "gemini-1.0-flash", Name: "Gemini 1.0 Flash", Provider: p.Type(), MaxTokens: 1048576, SupportsStreaming: true, SupportsToolCalling: true, Description: "Google's flash model"},
		{ID: "gemini-pro", Name: "Gemini Pro", Provider: p.Type(), MaxTokens: 30720, SupportsStreaming: true, SupportsToolCalling: true, Description: "Google's previous generation model"},
	}

	return models, nil
}

// GetDefaultModel returns the default model
func (p *GeminiProvider) GetDefaultModel() string {
	config := p.GetConfig()
	if config.DefaultModel != "" {
		return config.DefaultModel
	}
	return "gemini-2.0-flash-exp" // Default to latest experimental model
}

// GenerateChatCompletion generates a chat completion
func (p *GeminiProvider) GenerateChatCompletion(
	ctx context.Context,
	options provider.GenerateOptions,
) (provider.ChatCompletionStream, error) {
	config := p.GetConfig()
	p.LogRequest("POST", config.BaseURL, map[string]string{
		"Content-Type":       "application/json",
		"Authorization":      "Bearer " + config.APIKey,
		"x-goog-api-version": "1",
	}, options)

	// TODO: Implement actual Gemini API call
	// For now, return a mock response
	return &MockStream{
		chunks: []provider.ChatCompletionChunk{
			{Content: "This is a mock Gemini response for " + options.Prompt, Done: true},
		},
	}, nil
}

// InvokeServerTool invokes a server tool
func (p *GeminiProvider) InvokeServerTool(
	ctx context.Context,
	toolName string,
	params interface{},
) (interface{}, error) {
	return nil, fmt.Errorf("tool invocation not yet implemented for Gemini provider")
}

// Authenticate handles authentication (API key and OAuth)
func (p *GeminiProvider) Authenticate(ctx context.Context, authConfig provider.AuthConfig) error {
	switch authConfig.Method {
	case provider.AuthMethodOAuth:
		if authConfig.OAuthConfig == nil {
			return fmt.Errorf("OAuth config is required for OAuth authentication")
		}

		// Configure OAuth authenticator
		// Convert provider.AuthConfig to auth.AuthConfig
		authConfigWithTypes := auth.AuthConfig{
			Method: auth.AuthMethod(authConfig.Method),
		}

		// Convert OAuth config if present
		if authConfig.OAuthConfig != nil {
			authConfigWithTypes.OAuthConfig = &auth.OAuthConfig{
				ClientID:     authConfig.OAuthConfig.ClientID,
				ClientSecret: authConfig.OAuthConfig.ClientSecret,
				AuthURL:      authConfig.OAuthConfig.AuthURL,
				TokenURL:     authConfig.OAuthConfig.TokenURL,
				RedirectURL:  authConfig.OAuthConfig.RedirectURL,
				AccessToken:  authConfig.OAuthConfig.AccessToken,
				RefreshToken: authConfig.OAuthConfig.RefreshToken,
				ExpiresAt:    authConfig.OAuthConfig.ExpiresAt,
				TokenType:    "Bearer", // Default token type
				Scopes:       authConfig.OAuthConfig.Scopes,
			}
		}

		authConfigWithTypes.APIKey = authConfig.APIKey

		// Attempt authentication with stored token or OAuth flow
		err := p.oauthAuth.Authenticate(ctx, authConfigWithTypes)
		if err != nil {
			return fmt.Errorf("OAuth authentication failed: %w", err)
		}

		// Update configuration
		newConfig := p.GetConfig()
		newConfig.OAuthConfig = authConfig.OAuthConfig
		if err := p.Configure(newConfig); err != nil {
			return fmt.Errorf("failed to update provider config: %w", err)
		}

		// Validate token
		if err := p.oauthAuth.ValidateToken(ctx); err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}

		return nil

	case provider.AuthMethodAPIKey:
		if authConfig.APIKey == "" {
			return fmt.Errorf("API key is required for API key authentication")
		}

		// Update configuration with API key
		newConfig := p.GetConfig()
		newConfig.APIKey = authConfig.APIKey
		newConfig.OAuthConfig = nil // Clear OAuth config when using API key
		return p.Configure(newConfig)

	default:
		return fmt.Errorf("Gemini supports OAuth and API key authentication, got: %s", authConfig.Method)
	}
}

// IsAuthenticated checks if the provider is authenticated
func (p *GeminiProvider) IsAuthenticated() bool {
	// Check OAuth authentication
	if p.oauthAuth != nil && p.oauthAuth.IsAuthenticated() {
		return true
	}

	// Check API key authentication
	config := p.GetConfig()
	return config.APIKey != ""
}

// Logout handles logout
func (p *GeminiProvider) Logout(ctx context.Context) error {
	// Logout OAuth if active
	if p.oauthAuth != nil && p.oauthAuth.IsAuthenticated() {
		if err := p.oauthAuth.Logout(ctx); err != nil {
			return fmt.Errorf("OAuth logout failed: %w", err)
		}
	}

	// Clear configuration
	newConfig := p.GetConfig()
	newConfig.APIKey = ""
	newConfig.OAuthConfig = nil
	return p.Configure(newConfig)
}

// Configure updates the provider configuration
func (p *GeminiProvider) Configure(config provider.ProviderConfig) error {
	return p.BaseProvider.Configure(config)
}

// GetConfig returns the current configuration
func (p *GeminiProvider) GetConfig() provider.ProviderConfig {
	config := p.BaseProvider.GetConfig()
	config.OAuthConfig = p.authConfig
	return config
}

// SupportsToolCalling returns whether the provider supports tool calling
func (p *GeminiProvider) SupportsToolCalling() bool {
	return true
}

// SupportsStreaming returns whether the provider supports streaming
func (p *GeminiProvider) SupportsStreaming() bool {
	return true
}

// SupportsResponsesAPI returns whether the provider supports Responses API
func (p *GeminiProvider) SupportsResponsesAPI() bool {
	return false // Gemini uses native format
}

// GetToolFormat returns the tool format used by this provider
func (p *GeminiProvider) GetToolFormat() provider.ToolFormat {
	return provider.ToolFormatAnthropic // Gemini uses Anthropic-style tool format
}

// HealthCheck performs a health check
func (p *GeminiProvider) HealthCheck(ctx context.Context) error {
	return p.BaseProvider.HealthCheck(ctx)
}

// GetMetrics returns provider metrics
func (p *GeminiProvider) GetMetrics() provider.ProviderMetrics {
	return p.BaseProvider.GetMetrics()
}

// OAuth flow methods

// StartOAuthFlow initiates the OAuth flow for Gemini
func (p *GeminiProvider) StartOAuthFlow(ctx context.Context, scopes []string) (string, error) {
	if p.oauthAuth == nil {
		return "", fmt.Errorf("OAuth authenticator not initialized")
	}
	return p.oauthAuth.StartOAuthFlow(ctx, scopes)
}

// HandleOAuthCallback handles the OAuth callback
func (p *GeminiProvider) HandleOAuthCallback(ctx context.Context, code, state string) error {
	if p.oauthAuth == nil {
		return fmt.Errorf("OAuth authenticator not initialized")
	}
	if err := p.oauthAuth.HandleCallback(ctx, code, state); err != nil {
		return fmt.Errorf("OAuth callback failed: %w", err)
	}

	// Update provider config with new token
	if tokenInfo, err := p.oauthAuth.GetTokenInfo(); err == nil {
		newConfig := p.GetConfig()
		newConfig.OAuthConfig.AccessToken = tokenInfo.AccessToken
		newConfig.OAuthConfig.RefreshToken = tokenInfo.RefreshToken
		newConfig.OAuthConfig.ExpiresAt = tokenInfo.ExpiresAt
		_ = p.Configure(newConfig)
	}

	return nil
}

// GetOAuthURL returns the OAuth URL for authentication
func (p *GeminiProvider) GetOAuthURL() string {
	return auth.GeminiOAuthConfig().OAuthURL
}

// GetTokenInfo returns current token information
func (p *GeminiProvider) GetTokenInfo() (*auth.TokenInfo, error) {
	if p.oauthAuth == nil {
		return nil, fmt.Errorf("OAuth authenticator not initialized")
	}
	return p.oauthAuth.GetTokenInfo()
}

// RefreshToken refreshes the OAuth token
func (p *GeminiProvider) RefreshToken(ctx context.Context) error {
	if p.oauthAuth == nil {
		return fmt.Errorf("OAuth authenticator not initialized")
	}
	return p.oauthAuth.RefreshToken(ctx)
}

// Tool formatting methods

// FormatTools formats tools for Gemini API
func (p *GeminiProvider) FormatTools(providerTools []provider.Tool) (interface{}, error) {
	if len(providerTools) == 0 {
		return nil, nil
	}

	// Convert provider tools to tool formatter tools
	formatterTools := make([]tools.Tool, len(providerTools))
	for i, tool := range providerTools {
		formatterTools[i] = tools.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}
	return p.toolManager.FormatTools("gemini", formatterTools)
}

// ParseToolCalls parses tool calls from Gemini response
func (p *GeminiProvider) ParseToolCalls(response interface{}) ([]provider.ToolCall, error) {
	toolCalls, err := p.toolManager.ParseToolCalls("gemini", response)
	if err != nil {
		return nil, err
	}

	// Convert tool formatter calls to provider calls
	providerCalls := make([]provider.ToolCall, len(toolCalls))
	for i, call := range toolCalls {
		providerCalls[i] = provider.ToolCall{
			ID:   call.ID,
			Type: "function",
			Function: provider.ToolCallFunction{
				Name:      call.Name,
				Arguments: p.serializeArguments(call.Arguments),
			},
			Metadata: call.Metadata,
		}
	}

	return providerCalls, nil
}

// Helper method to serialize arguments to JSON string
func (p *GeminiProvider) serializeArguments(args map[string]interface{}) string {
	if len(args) == 0 {
		return "{}"
	}

	if data, err := json.Marshal(args); err == nil {
		return string(data)
	}
	return "{}"
}

// MockStream implements ChatCompletionStream for testing
type MockStream struct {
	chunks []provider.ChatCompletionChunk
	index  int
}

func (ms *MockStream) Next() (provider.ChatCompletionChunk, error) {
	if ms.index >= len(ms.chunks) {
		return provider.ChatCompletionChunk{}, nil
	}
	chunk := ms.chunks[ms.index]
	ms.index++
	return chunk, nil
}

func (ms *MockStream) Close() error {
	ms.index = 0
	return nil
}
