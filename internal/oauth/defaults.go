package oauth

import "os"

// DefaultOAuthConfig holds preconfigured OAuth credentials for providers
// These credentials should be configured via environment variables or config file
// following the OAuth 2.0 "installed application" pattern.
//
// For desktop/CLI applications:
// 1. Users authenticate with their own accounts (get their own tokens)
// 2. PKCE (Proof Key for Code Exchange) protects against authorization code interception
// 3. Client secrets are not treated as secret for public clients
//
// To register OAuth apps and get credentials:
// - Anthropic: https://console.anthropic.com/settings/oauth
// - Gemini: https://console.cloud.google.com/apis/credentials
// - Qwen: https://dashscope.console.aliyun.com/
//
// Environment variables:
// - ANTHROPIC_OAUTH_CLIENT_ID
// - GEMINI_OAUTH_CLIENT_ID and GEMINI_OAUTH_CLIENT_SECRET
// - QWEN_OAUTH_CLIENT_ID

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	RefreshURL   string
	Scopes       []string
}

var (
	// AnthropicOAuth holds Anthropic Claude OAuth configuration
	// Set ANTHROPIC_OAUTH_CLIENT_ID environment variable
	// This is a public client (PKCE-protected, no client secret)
	AnthropicOAuth = OAuthProviderConfig{
		ClientID:     os.Getenv("ANTHROPIC_OAUTH_CLIENT_ID"),
		ClientSecret: "",
		AuthURL:      "https://claude.ai/oauth/authorize",
		TokenURL:     "https://console.anthropic.com/v1/oauth/token",
		RefreshURL:   "https://console.anthropic.com/v1/oauth/token",
		Scopes:       []string{"org:create_api_key", "user:profile", "user:inference"},
	}

	// GeminiOAuth holds Google Gemini OAuth configuration
	// Set GEMINI_OAUTH_CLIENT_ID and GEMINI_OAUTH_CLIENT_SECRET environment variables
	// This is a public client as per Google's OAuth 2.0 "installed application" pattern
	GeminiOAuth = OAuthProviderConfig{
		ClientID:     os.Getenv("GEMINI_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GEMINI_OAUTH_CLIENT_SECRET"),
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		RefreshURL:   "https://oauth2.googleapis.com/token",
		Scopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	// QwenOAuth holds Alibaba Qwen OAuth configuration
	// Set QWEN_OAUTH_CLIENT_ID environment variable
	// This is a public client for device flow authentication
	QwenOAuth = OAuthProviderConfig{
		ClientID:     os.Getenv("QWEN_OAUTH_CLIENT_ID"),
		ClientSecret: "",
		AuthURL:      "https://chat.qwen.ai/api/v1/oauth2/device/code",
		TokenURL:     "https://chat.qwen.ai/api/v1/oauth2/token",
		RefreshURL:   "https://chat.qwen.ai/api/v1/oauth2/token",
		Scopes:       []string{"openid", "profile", "email", "model.completion"},
	}
)

// GetProviderConfig returns the OAuth configuration for a provider
func GetProviderConfig(provider string) (*OAuthProviderConfig, bool) {
	switch provider {
	case "anthropic":
		return &AnthropicOAuth, true
	case "gemini":
		return &GeminiOAuth, true
	case "qwen":
		return &QwenOAuth, true
	default:
		return nil, false
	}
}

// IsConfigured checks if a provider has OAuth credentials configured
func IsConfigured(provider string) bool {
	config, ok := GetProviderConfig(provider)
	if !ok {
		return false
	}
	// All providers now have official client IDs configured
	return config.ClientID != ""
}
