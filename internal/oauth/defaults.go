package oauth

// DefaultOAuthConfig holds preconfigured OAuth credentials for providers
// These credentials are embedded in the application following the OAuth 2.0
// "installed application" pattern, as described in Google's OAuth documentation:
// https://developers.google.com/identity/protocols/oauth2#installed
//
// For desktop/CLI applications, it's acceptable to embed OAuth client credentials
// in the source code because:
// 1. Users authenticate with their own accounts (get their own tokens)
// 2. PKCE (Proof Key for Code Exchange) protects against authorization code interception
// 3. Client secrets are not treated as secret for public clients
//
// Note: It's ok to save this in git because this is an installed application
// as described here: https://developers.google.com/identity/protocols/oauth2#installed
// "The process results in a client ID and, in some cases, a client secret,
// which you embed in the source code of your application. (In this context,
// the client secret is obviously not treated as a secret.)"
//
// To register OAuth apps and get credentials:
// - Anthropic: https://console.anthropic.com/settings/oauth
// - Gemini: https://console.cloud.google.com/apis/credentials
// - Qwen: https://dashscope.console.aliyun.com/

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
	// Uses the official Claude Code CLI OAuth client ID
	// This is a public client (PKCE-protected, no client secret)
	// Source: https://github.com/anthropics/claude-code
	AnthropicOAuth = OAuthProviderConfig{
		ClientID:     "9d1c250a-e61b-44d9-88ed-5944d1962f5e", // Official Claude Code CLI client ID
		ClientSecret: "",                                     // Anthropic uses PKCE, no client secret needed
		AuthURL:      "https://claude.ai/oauth/authorize",
		TokenURL:     "https://console.anthropic.com/v1/oauth/token",
		RefreshURL:   "https://console.anthropic.com/v1/oauth/token",
		Scopes:       []string{"org:create_api_key", "user:profile", "user:inference"},
	}

	// GeminiOAuth holds Google Gemini OAuth configuration
	// Uses the official Gemini CLI OAuth client ID
	// This is a public client as per Google's OAuth 2.0 "installed application" pattern
	// Source: https://github.com/google-gemini/gemini-cli (llxprt-code)
	// Note: Client secret is public for desktop apps (see https://developers.google.com/identity/protocols/oauth2#installed)
	GeminiOAuth = OAuthProviderConfig{
		ClientID:     "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com", // Official Gemini CLI client ID
		ClientSecret: "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",                                      // Public client secret (from llxprt-code)
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
	// Uses the client ID from Qwen Code implementation
	// This is a public client for device flow authentication
	// Source: Qwen Code documentation
	QwenOAuth = OAuthProviderConfig{
		ClientID:     "f0304373b74a44d2b584a3fb70ca9e56", // Qwen Code client ID
		ClientSecret: "",                                  // Qwen uses device flow, no client secret needed
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
