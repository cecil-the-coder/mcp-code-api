package auth

import (
	"context"
	"fmt"

	"github.com/cecil-the-coder/mcp-code-api/internal/oauth"
)

// Official Google OAuth credentials from llxprt-code project
// Source: https://github.com/google/llxprt-code
var (
	GeminiOAuthClientID     = oauth.GeminiOAuth.ClientID
	GeminiOAuthClientSecret = oauth.GeminiOAuth.ClientSecret
)

// GeminiOAuthConfig returns the OAuth configuration for Google Gemini
func GeminiOAuthConfig() *ProviderAuthConfig {
	return &ProviderAuthConfig{
		Provider:    "gemini",
		AuthMethod:  AuthMethodOAuth,
		DisplayName: "Google Gemini",
		Description: "Authenticate with Google Gemini API using Google OAuth 2.0",
		OAuthURL:    "https://console.cloud.google.com",
		RequiredScopes: []string{
			"https://www.googleapis.com/auth/generative-language",
		},
		OptionalScopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}
}

// GeminiOAuthAuthenticator implements OAuth for Google Gemini
type GeminiOAuthAuthenticator struct {
	*OAuthAuthenticatorImpl
}

// NewGeminiOAuthAuthenticator creates a new Gemini OAuth authenticator
func NewGeminiOAuthAuthenticator(storage TokenStorage) *GeminiOAuthAuthenticator {
	return &GeminiOAuthAuthenticator{
		OAuthAuthenticatorImpl: NewOAuthAuthenticator("gemini", storage),
	}
}

// StartOAuthFlow starts the Google OAuth flow
func (g *GeminiOAuthAuthenticator) StartOAuthFlow(ctx context.Context, scopes []string) (string, error) {
	// Ensure required scopes are included
	requiredScopes := []string{"https://www.googleapis.com/auth/generative-language"}
	allScopes := mergeScopes(requiredScopes, scopes)

	return g.OAuthAuthenticatorImpl.StartOAuthFlow(ctx, allScopes)
}

// HandleCallback handles the Google OAuth callback
func (g *GeminiOAuthAuthenticator) HandleCallback(ctx context.Context, code, state string) error {
	return g.OAuthAuthenticatorImpl.HandleCallback(ctx, code, state)
}

// ValidateToken validates a Google Gemini token
func (g *GeminiOAuthAuthenticator) ValidateToken(ctx context.Context) error {
	if !g.IsAuthenticated() {
		return &AuthError{
			Provider: g.provider,
			Code:     ErrCodeTokenExpired,
			Message:  "Not authenticated",
		}
	}

	tokenInfo, err := g.GetTokenInfo()
	if err != nil {
		return err
	}

	// Check for required scopes
	requiredScopes := []string{"https://www.googleapis.com/auth/generative-language"}
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
				Provider: g.provider,
				Code:     ErrCodeScopeInsufficient,
				Message:  fmt.Sprintf("Missing required scope: %s", required),
			}
		}
	}

	return nil
}

// GetGeminiOAuthConfig returns the OAuth configuration for Google Gemini
func GetGeminiOAuthConfig(clientID, clientSecret, redirectURL string) *OAuthConfig {
	return &OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		RefreshURL:   "https://oauth2.googleapis.com/token",
		Scopes: []string{
			"https://www.googleapis.com/auth/generative-language",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		TokenType: "Bearer",
	}
}

// GetGeminiAPIKeyAuthConfig returns an API key auth config for Gemini
func GetGeminiAPIKeyAuthConfig(apiKey string) AuthConfig {
	return AuthConfig{
		Method:  AuthMethodAPIKey,
		APIKey:  apiKey,
		BaseURL: "https://generativelanguage.googleapis.com",
	}
}

// GetAIGeminiOAuthConfig returns OAuth config for AI Studio (alternative)
func GetAIGeminiOAuthConfig(clientID, clientSecret, redirectURL string) *OAuthConfig {
	return &OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://aistudio.google.com/oauth/authorize",
		TokenURL:     "https://aistudio.google.com/oauth/token",
		RefreshURL:   "https://aistudio.google.com/oauth/refresh",
		Scopes:       []string{"https://www.googleapis.com/auth/generative-language"},
		TokenType:    "Bearer",
	}
}
