package auth

import (
	"context"
	"fmt"
	"time"
)

// Authenticator defines the interface for authentication methods
type Authenticator interface {
	// Authenticate performs authentication with the given config
	Authenticate(ctx context.Context, config AuthConfig) error

	// IsAuthenticated checks if currently authenticated
	IsAuthenticated() bool

	// GetToken returns the current authentication token
	GetToken() (string, error)

	// RefreshToken refreshes the authentication token if needed
	RefreshToken(ctx context.Context) error

	// Logout clears authentication state
	Logout(ctx context.Context) error

	// GetAuthMethod returns the authentication method type
	GetAuthMethod() AuthMethod
}

// OAuthAuthenticator extends Authenticator for OAuth-specific functionality
type OAuthAuthenticator interface {
	Authenticator

	// StartOAuthFlow initiates the OAuth flow and returns the auth URL
	StartOAuthFlow(ctx context.Context, scopes []string) (string, error)

	// HandleCallback processes the OAuth callback
	HandleCallback(ctx context.Context, code, state string) error

	// IsOAuthEnabled checks if OAuth is properly configured
	IsOAuthEnabled() bool

	// GetTokenInfo returns detailed token information
	GetTokenInfo() (*TokenInfo, error)
}

// AuthMethod represents the authentication method
type AuthMethod string

const (
	AuthMethodAPIKey      AuthMethod = "api_key"
	AuthMethodOAuth       AuthMethod = "oauth"
	AuthMethodBearerToken AuthMethod = "bearer_token"
	AuthMethodCustom      AuthMethod = "custom"
)

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Method      AuthMethod   `json:"method"`
	APIKey      string       `json:"api_key,omitempty"`
	BaseURL     string       `json:"base_url,omitempty"`
	OAuthConfig *OAuthConfig `json:"oauth,omitempty"`
}

// OAuthConfig represents OAuth configuration
type OAuthConfig struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	RedirectURL  string    `json:"redirect_url,omitempty"`
	Scopes       []string  `json:"scopes"`
	AuthURL      string    `json:"auth_url"`
	TokenURL     string    `json:"token_url"`
	RefreshURL   string    `json:"refresh_url,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	State        string    `json:"state,omitempty"`
}

// TokenInfo represents information about an authentication token
type TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scopes       []string  `json:"scopes"`
	IsExpired    bool      `json:"is_expired"`
	ExpiresIn    int64     `json:"expires_in"`
}

// TokenStorage defines the interface for token persistence
type TokenStorage interface {
	StoreToken(provider string, token *OAuthConfig) error
	RetrieveToken(provider string) (*OAuthConfig, error)
	DeleteToken(provider string) error
	ListTokens() ([]string, error)
	IsTokenValid(provider string) bool
}

// AuthManager manages authentication for multiple providers
type AuthManager interface {
	// RegisterAuthenticator registers an authenticator for a provider
	RegisterAuthenticator(provider string, authenticator Authenticator) error

	// GetAuthenticator returns the authenticator for a provider
	GetAuthenticator(provider string) (Authenticator, error)

	// Authenticate authenticates a provider
	Authenticate(ctx context.Context, provider string, config AuthConfig) error

	// IsAuthenticated checks if a provider is authenticated
	IsAuthenticated(provider string) bool

	// Logout logs out a provider
	Logout(ctx context.Context, provider string) error

	// RefreshAllTokens refreshes all tokens that need it
	RefreshAllTokens(ctx context.Context) error

	// GetAuthenticatedProviders returns a list of authenticated providers
	GetAuthenticatedProviders() []string
}

// ProviderAuthConfig represents provider-specific authentication configuration
type ProviderAuthConfig struct {
	Provider       string     `json:"provider"`
	AuthMethod     AuthMethod `json:"auth_method"`
	DisplayName    string     `json:"display_name"`
	Description    string     `json:"description"`
	OAuthURL       string     `json:"oauth_url,omitempty"`
	RequiredScopes []string   `json:"required_scopes,omitempty"`
	OptionalScopes []string   `json:"optional_scopes,omitempty"`
}

// AuthError represents authentication-related errors
type AuthError struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Details  string `json:"details,omitempty"`
}

func (e *AuthError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s authentication error: %s (%s) - %s", e.Provider, e.Message, e.Code, e.Details)
	}
	return fmt.Sprintf("%s authentication error: %s (%s)", e.Provider, e.Message, e.Code)
}

// Common error codes
const (
	ErrCodeInvalidCredentials  = "invalid_credentials"
	ErrCodeTokenExpired        = "token_expired"
	ErrCodeRefreshFailed       = "refresh_failed"
	ErrCodeOAuthFlowFailed     = "oauth_flow_failed"
	ErrCodeInvalidConfig       = "invalid_config"
	ErrCodeNetworkError        = "network_error"
	ErrCodeProviderUnavailable = "provider_unavailable"
	ErrCodeScopeInsufficient   = "scope_insufficient"
)

// AuthState represents the current authentication state
type AuthState struct {
	Provider      string     `json:"provider"`
	Authenticated bool       `json:"authenticated"`
	Method        AuthMethod `json:"method"`
	LastAuth      time.Time  `json:"last_auth"`
	ExpiresAt     time.Time  `json:"expires_at,omitempty"`
	CanRefresh    bool       `json:"can_refresh"`
}
