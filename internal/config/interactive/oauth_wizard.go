package interactive

import (
	"context"
	"fmt"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/auth"
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

// performOAuthFlow performs the full OAuth authentication flow with PKCE
func (w *Wizard) performOAuthFlow(providerName string, config ProviderOAuthConfig) (*auth.TokenInfo, error) {
	fmt.Printf("\n🔐 Starting OAuth flow for %s...\n", providerName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Generate PKCE parameters for enhanced security
	pkceParams, err := oauth.GeneratePKCEParams()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE parameters: %w", err)
	}
	fmt.Println("🔒 PKCE protection enabled")

	// Start callback server - try a range of ports like llxprt-code does
	// Port range: 8080-8110 (31 ports to try)
	server, err := oauth.NewCallbackServerWithPortRange(8080, 8110)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer func() { _ = server.Stop() }()

	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	redirectURL := server.GetRedirectURL()
	fmt.Printf("📍 Callback server started at: %s\n", redirectURL)

	// Create memory-based storage (tokens will be saved to config.yaml instead)
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

	// Configure OAuth with PKCE parameters
	oauthConfig := &auth.OAuthConfig{
		ClientID:      config.ClientID,
		ClientSecret:  config.ClientSecret,
		RedirectURL:   redirectURL,
		Scopes:        config.Scopes,
		AuthURL:       config.AuthURL,
		TokenURL:      config.TokenURL,
		RefreshURL:    config.RefreshURL,
		CodeChallenge: pkceParams.CodeChallenge,
		CodeVerifier:  pkceParams.CodeVerifier,
		State:         pkceParams.State,
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

	// Validate state parameter to prevent CSRF attacks
	if !oauth.ValidateState(pkceParams.State, result.State) {
		return nil, fmt.Errorf("state validation failed: possible CSRF attack")
	}
	fmt.Println("✅ State validated (CSRF protection)")

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

	// Get preconfigured OAuth settings from defaults
	oauthDefaults, ok := oauth.GetProviderConfig(providerName)
	if !ok {
		return nil, nil, fmt.Errorf("OAuth not supported for %s", providerName)
	}

	// Show information about the OAuth flow
	switch providerName {
	case "anthropic":
		fmt.Println("Using official Claude Code CLI OAuth credentials")
		fmt.Println("You'll authenticate with your Anthropic account in the browser")
	case "gemini":
		fmt.Println("Using official Gemini CLI OAuth credentials")
		fmt.Println("You'll authenticate with your Google account in the browser")
	case "qwen":
		fmt.Println("Using Qwen Code OAuth credentials")
		fmt.Println("You'll authenticate with your Qwen account in the browser")
	}

	fmt.Println()

	config := &ProviderOAuthConfig{
		Provider:     providerName,
		ClientID:     oauthDefaults.ClientID,
		ClientSecret: oauthDefaults.ClientSecret,
		Scopes:       oauthDefaults.Scopes,
		AuthURL:      oauthDefaults.AuthURL,
		TokenURL:     oauthDefaults.TokenURL,
		RefreshURL:   oauthDefaults.RefreshURL,
	}

	// Perform OAuth flow
	tokenInfo, err := w.performOAuthFlow(providerName, *config)
	if err != nil {
		return config, nil, err
	}

	return config, tokenInfo, nil
}
