package auth

import (
	"context"
	"fmt"
)

// AnthropicOAuthConfig returns the OAuth configuration for Anthropic
func AnthropicOAuthConfig() *ProviderAuthConfig {
	return &ProviderAuthConfig{
		Provider:    "anthropic",
		AuthMethod:  AuthMethodOAuth,
		DisplayName: "Anthropic Claude",
		Description: "Authenticate with Anthropic Claude API using OAuth",
		OAuthURL:    "https://console.anthropic.com",
		RequiredScopes: []string{
			"messages",
			"tools",
		},
		OptionalScopes: []string{
			"models",
			"usage",
		},
	}
}

// AnthropicOAuthAuthenticator implements OAuth for Anthropic
type AnthropicOAuthAuthenticator struct {
	*OAuthAuthenticatorImpl
}

// NewAnthropicOAuthAuthenticator creates a new Anthropic OAuth authenticator
func NewAnthropicOAuthAuthenticator(storage TokenStorage) *AnthropicOAuthAuthenticator {
	return &AnthropicOAuthAuthenticator{
		OAuthAuthenticatorImpl: NewOAuthAuthenticator("anthropic", storage),
	}
}

// StartOAuthFlow starts the Anthropic OAuth flow
func (a *AnthropicOAuthAuthenticator) StartOAuthFlow(ctx context.Context, scopes []string) (string, error) {
	// Ensure required scopes are included
	requiredScopes := []string{"messages", "tools"}
	allScopes := mergeScopes(requiredScopes, scopes)

	return a.OAuthAuthenticatorImpl.StartOAuthFlow(ctx, allScopes)
}

// HandleCallback handles the Anthropic OAuth callback
func (a *AnthropicOAuthAuthenticator) HandleCallback(ctx context.Context, code, state string) error {
	return a.OAuthAuthenticatorImpl.HandleCallback(ctx, code, state)
}

// ValidateToken validates an Anthropic token by checking its format and scope
func (a *AnthropicOAuthAuthenticator) ValidateToken(ctx context.Context) error {
	if !a.IsAuthenticated() {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeTokenExpired,
			Message:  "Not authenticated",
		}
	}

	tokenInfo, err := a.GetTokenInfo()
	if err != nil {
		return err
	}

	// Check for required scopes
	requiredScopes := []string{"messages", "tools"}
	for _, required := range requiredScopes {
		found := false
		for _, scope := range tokenInfo.Scopes {
			if scope == required {
				found = true
				break
			}
		}
		if !found {
			return &AuthError{
				Provider: a.provider,
				Code:     ErrCodeScopeInsufficient,
				Message:  fmt.Sprintf("Missing required scope: %s", required),
			}
		}
	}

	return nil
}

// GetOAuthConfig returns the OAuth configuration for Anthropic
func GetAnthropicOAuthConfig(clientID, clientSecret, redirectURL string) *OAuthConfig {
	return &OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://api.anthropic.com/oauth/authorize",
		TokenURL:     "https://api.anthropic.com/oauth/token",
		RefreshURL:   "https://api.anthropic.com/oauth/refresh",
		Scopes:       []string{"messages", "tools"},
		TokenType:    "Bearer",
	}
}

// GetAnthropicAPIKeyAuthConfig returns an API key auth config for Anthropic
func GetAnthropicAPIKeyAuthConfig(apiKey string) AuthConfig {
	return AuthConfig{
		Method:  AuthMethodAPIKey,
		APIKey:  apiKey,
		BaseURL: "https://api.anthropic.com",
	}
}

// Helper function to merge scopes
func mergeScopes(required, optional []string) []string {
	scopeSet := make(map[string]bool)
	for _, scope := range required {
		scopeSet[scope] = true
	}
	for _, scope := range optional {
		scopeSet[scope] = true
	}

	var result []string
	for scope := range scopeSet {
		result = append(result, scope)
	}
	return result
}
