package interactive

import (
	"context"
	"fmt"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/auth"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/oauth"
)

// ProviderOAuthConfig holds OAuth configuration for a provider
type ProviderOAuthConfig struct {
	Provider     string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	AuthURL      string
	TokenURL     string
	RefreshURL   string
}

// performOAuthFlow performs the full OAuth authentication flow
func (w *Wizard) performOAuthFlow(providerName string, config ProviderOAuthConfig) (*auth.TokenInfo, error) {
	fmt.Printf("\n🔐 Starting OAuth flow for %s...\n", providerName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Start callback server
	callbackPort := 8080
	server, err := oauth.NewCallbackServer(callbackPort)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer server.Stop()

	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	redirectURL := server.GetRedirectURL()
	fmt.Printf("📍 Callback server started at: %s\n", redirectURL)

	// Create storage and authenticator
	storage := auth.NewMemoryTokenStorage()
	var authenticator auth.OAuthAuthenticator

	switch providerName {
	case "anthropic":
		authenticator = auth.NewAnthropicOAuthAuthenticator(storage)
	case "gemini":
		authenticator = auth.NewOAuthAuthenticator("gemini", storage)
	case "qwen":
		authenticator = auth.NewOAuthAuthenticator("qwen", storage)
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", providerName)
	}

	// Configure OAuth
	oauthConfig := &auth.OAuthConfig{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       config.Scopes,
		AuthURL:      config.AuthURL,
		TokenURL:     config.TokenURL,
		RefreshURL:   config.RefreshURL,
	}

	authConfig := auth.AuthConfig{
		Method:      auth.AuthMethodOAuth,
		OAuthConfig: oauthConfig,
	}

	ctx := context.Background()
	if err := authenticator.Authenticate(ctx, authConfig); err != nil {
		// Authentication will fail because we don't have a token yet
		// This is expected, we'll get the auth URL next
		logger.Debugf("Initial auth failed (expected): %v", err)
	}

	// Start OAuth flow
	authURL, err := authenticator.StartOAuthFlow(ctx, config.Scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to start OAuth flow: %w", err)
	}

	fmt.Println("\n📱 Opening browser for authentication...")
	fmt.Printf("🌐 Auth URL: %s\n\n", authURL)

	// Try to open browser
	if err := oauth.OpenBrowser(authURL); err != nil {
		logger.Debugf("Failed to open browser automatically: %v", err)
		fmt.Println("⚠️  Could not open browser automatically.")
		fmt.Println("Please manually open the URL above in your browser.")
	}

	fmt.Println("⏳ Waiting for authentication callback...")
	fmt.Println("   (This will timeout in 5 minutes)")

	// Wait for callback
	result, err := server.WaitForCallback(5 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to receive callback: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("OAuth error: %s", result.Error)
	}

	fmt.Println("\n✅ Received authorization code!")
	fmt.Println("🔄 Exchanging code for access token...")

	// Handle callback to exchange code for token
	if err := authenticator.HandleCallback(ctx, result.Code, result.State); err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get token info
	tokenInfo, err := authenticator.GetTokenInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get token info: %w", err)
	}

	fmt.Println("\n✅ Authentication successful!")
	fmt.Printf("📅 Token expires: %s\n", tokenInfo.ExpiresAt.Format(time.RFC3339))

	return tokenInfo, nil
}

// configureProviderOAuth configures OAuth for a specific provider
func (w *Wizard) configureProviderOAuth(providerName, displayName string) (*ProviderOAuthConfig, *auth.TokenInfo, error) {
	fmt.Printf("\n🔐 %s OAuth Configuration\n", displayName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Get provider-specific OAuth config
	var defaultAuthURL, defaultTokenURL, defaultRefreshURL string
	var defaultScopes []string

	switch providerName {
	case "anthropic":
		defaultAuthURL = "https://api.anthropic.com/oauth/authorize"
		defaultTokenURL = "https://api.anthropic.com/oauth/token"
		defaultRefreshURL = "https://api.anthropic.com/oauth/refresh"
		defaultScopes = []string{"messages", "tools"}
		fmt.Println("Register your OAuth app at: https://console.anthropic.com/settings/oauth")
	case "gemini":
		defaultAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
		defaultTokenURL = "https://oauth2.googleapis.com/token"
		defaultScopes = []string{"https://www.googleapis.com/auth/generative-language"}
		fmt.Println("Register your OAuth app at: https://console.cloud.google.com/apis/credentials")
	case "qwen":
		defaultAuthURL = "https://dashscope.aliyuncs.com/oauth/authorize"
		defaultTokenURL = "https://dashscope.aliyuncs.com/oauth/token"
		defaultScopes = []string{"api"}
		fmt.Println("Register your OAuth app at: https://dashscope.console.aliyun.com/")
	default:
		return nil, nil, fmt.Errorf("OAuth not supported for %s", providerName)
	}

	fmt.Println()

	// Collect OAuth credentials
	clientID := w.prompt(fmt.Sprintf("Enter %s OAuth Client ID: ", displayName), false)
	if clientID == "" {
		return nil, nil, fmt.Errorf("client ID is required")
	}

	clientSecret := w.prompt(fmt.Sprintf("Enter %s OAuth Client Secret: ", displayName), false)
	if clientSecret == "" {
		return nil, nil, fmt.Errorf("client secret is required")
	}

	config := &ProviderOAuthConfig{
		Provider:     providerName,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       defaultScopes,
		AuthURL:      defaultAuthURL,
		TokenURL:     defaultTokenURL,
		RefreshURL:   defaultRefreshURL,
	}

	// Perform OAuth flow
	tokenInfo, err := w.performOAuthFlow(providerName, *config)
	if err != nil {
		return config, nil, err
	}

	return config, tokenInfo, nil
}

// getOAuthConfigForProvider returns OAuth config for a provider in the config format
func getOAuthConfigForProvider(oauthConfig *ProviderOAuthConfig, tokenInfo *auth.TokenInfo) *provider.OAuthConfig {
	if oauthConfig == nil {
		return nil
	}

	providerConfig := &provider.OAuthConfig{
		ClientID:     oauthConfig.ClientID,
		ClientSecret: oauthConfig.ClientSecret,
		RedirectURL:  "", // Will be set during flow
		Scopes:       oauthConfig.Scopes,
		AuthURL:      oauthConfig.AuthURL,
		TokenURL:     oauthConfig.TokenURL,
	}

	if tokenInfo != nil {
		providerConfig.AccessToken = tokenInfo.AccessToken
		providerConfig.RefreshToken = tokenInfo.RefreshToken
		providerConfig.ExpiresAt = tokenInfo.ExpiresAt
	}

	return providerConfig
}
