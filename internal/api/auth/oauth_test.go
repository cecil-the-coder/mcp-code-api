package auth

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestOAuthAuthenticatorImplementsInterface ensures OAuthAuthenticatorImpl implements the interface
func TestOAuthAuthenticatorImplementsInterface(t *testing.T) {
	var _ OAuthAuthenticator = (*OAuthAuthenticatorImpl)(nil)
	var _ Authenticator = (*OAuthAuthenticatorImpl)(nil)
}

// TestTokenStorageImplementsInterface ensures storage implementations implement the interface
func TestTokenStorageImplementsInterface(t *testing.T) {
	var _ TokenStorage = (*FileTokenStorage)(nil)
	var _ TokenStorage = (*MemoryTokenStorage)(nil)
}

// TestMemoryTokenStorageOperations tests memory token storage operations
func TestMemoryTokenStorageOperations(t *testing.T) {
	storage := NewMemoryTokenStorage()

	t.Run("Store and Retrieve Token", func(t *testing.T) {
		token := &OAuthConfig{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
			TokenType:    "Bearer",
		}

		// Store token
		err := storage.StoreToken("test-provider", token)
		if err != nil {
			t.Fatalf("Failed to store token: %v", err)
		}

		// Retrieve token
		retrieved, err := storage.RetrieveToken("test-provider")
		if err != nil {
			t.Fatalf("Failed to retrieve token: %v", err)
		}

		if retrieved.ClientID != token.ClientID {
			t.Errorf("Expected client ID %s, got %s", token.ClientID, retrieved.ClientID)
		}
		if retrieved.AccessToken != token.AccessToken {
			t.Errorf("Expected access token %s, got %s", token.AccessToken, retrieved.AccessToken)
		}
	})

	t.Run("Token Not Found", func(t *testing.T) {
		_, err := storage.RetrieveToken("nonexistent-provider")
		if err == nil {
			t.Error("Expected error for nonexistent token, got nil")
		}
	})

	t.Run("List Tokens", func(t *testing.T) {
		// Store another token
		_ = storage.StoreToken("test-provider-2", &OAuthConfig{
			ClientID:    "test-client-2",
			AccessToken: "test-access-token-2",
		})

		providers, err := storage.ListTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		if len(providers) != 2 {
			t.Errorf("Expected 2 providers, got %d", len(providers))
		}
	})

	t.Run("Delete Token", func(t *testing.T) {
		err := storage.DeleteToken("test-provider")
		if err != nil {
			t.Fatalf("Failed to delete token: %v", err)
		}

		_, err = storage.RetrieveToken("test-provider")
		if err == nil {
			t.Error("Expected error after deleting token, got nil")
		}
	})

	t.Run("Is Token Valid", func(t *testing.T) {
		valid := storage.IsTokenValid("test-provider-2")
		if !valid {
			t.Error("Expected token to be valid")
		}

		invalid := storage.IsTokenValid("test-provider")
		if invalid {
			t.Error("Expected deleted token to be invalid")
		}
	})
}

// TestOAuthAuthenticatorBasicFunctionality tests basic OAuth authenticator functionality
func TestOAuthAuthenticatorBasicFunctionality(t *testing.T) {
	storage := NewMemoryTokenStorage()
	auth := NewOAuthAuthenticator("test-provider", storage)

	t.Run("Initial State", func(t *testing.T) {
		if auth.IsAuthenticated() {
			t.Error("Expected not authenticated initially")
		}

		if auth.GetAuthMethod() != AuthMethodOAuth {
			t.Errorf("Expected auth method %s, got %s", AuthMethodOAuth, auth.GetAuthMethod())
		}
	})

	t.Run("Invalid Auth Method", func(t *testing.T) {
		config := AuthConfig{
			Method: AuthMethodAPIKey,
			APIKey: "test-key",
		}

		err := auth.Authenticate(context.Background(), config)
		if err == nil {
			t.Error("Expected error for invalid auth method, got nil")
		}
	})

	t.Run("Missing OAuth Config", func(t *testing.T) {
		config := AuthConfig{
			Method: AuthMethodOAuth,
		}

		err := auth.Authenticate(context.Background(), config)
		if err == nil {
			t.Error("Expected error for missing OAuth config, got nil")
		}
	})

	t.Run("Is OAuth Enabled", func(t *testing.T) {
		// Initially not enabled
		if auth.IsOAuthEnabled() {
			t.Error("Expected OAuth not to be enabled initially")
		}

		// Configure with basic settings
		oauthConfig := &OAuthConfig{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			AuthURL:      "https://example.com/oauth/auth",
			TokenURL:     "https://example.com/oauth/token",
		}

		ctx := context.Background()
		config := AuthConfig{
			Method:      AuthMethodOAuth,
			OAuthConfig: oauthConfig,
		}

		err := auth.Authenticate(ctx, config)
		if err != nil {
			t.Logf("Authentication failed as expected (no token): %v", err)
		}

		// Should now be enabled even though not authenticated
		if !auth.IsOAuthEnabled() {
			t.Error("Expected OAuth to be enabled after configuration")
		}
	})
}

// TestAuthManagerOperations tests the auth manager functionality
func TestAuthManagerOperations(t *testing.T) {
	storage := NewMemoryTokenStorage()
	manager := NewAuthManager(storage)

	t.Run("Register Authenticator for invalid input", func(t *testing.T) {
		err := manager.RegisterAuthenticator("", nil)
		if err == nil {
			t.Error("Expected error for empty provider name")
		}

		err = manager.RegisterAuthenticator("test", nil)
		if err == nil {
			t.Error("Expected error for nil authenticator")
		}
	})

	t.Run("Register and Get Authenticator", func(t *testing.T) {
		auth := NewOAuthAuthenticator("test-provider", storage)

		err := manager.RegisterAuthenticator("test-provider", auth)
		if err != nil {
			t.Fatalf("Failed to register authenticator: %v", err)
		}

		retrieved, err := manager.GetAuthenticator("test-provider")
		if err != nil {
			t.Fatalf("Failed to get authenticator: %v", err)
		}

		if retrieved.GetAuthMethod() != AuthMethodOAuth {
			t.Errorf("Expected OAuth authenticator, got %s", retrieved.GetAuthMethod())
		}
	})

	t.Run("Get Nonexistent Authenticator", func(t *testing.T) {
		_, err := manager.GetAuthenticator("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent authenticator")
		}
	})

	t.Run("Authentication Status", func(t *testing.T) {
		// Initially not authenticated
		if manager.IsAuthenticated("test-provider") {
			t.Error("Expected provider not to be authenticated initially")
		}

		// Get status
		status := manager.GetAuthStatus()
		if len(status) != 1 {
			t.Errorf("Expected 1 status entry, got %d", len(status))
		}

		if status["test-provider"].Authenticated {
			t.Error("Expected authenticated status to be false")
		}
	})
}

// TestProviderAuthConfigs tests provider-specific auth configurations
func TestProviderAuthConfigs(t *testing.T) {
	t.Run("Anthropic Config", func(t *testing.T) {
		config := AnthropicOAuthConfig()
		if config.Provider != "anthropic" {
			t.Errorf("Expected provider 'anthropic', got '%s'", config.Provider)
		}
		if config.AuthMethod != AuthMethodOAuth {
			t.Errorf("Expected auth method %s, got %s", AuthMethodOAuth, config.AuthMethod)
		}
		if len(config.RequiredScopes) == 0 {
			t.Error("Expected required scopes to be set")
		}
	})

	t.Run("Gemini Config", func(t *testing.T) {
		config := GeminiOAuthConfig()
		if config.Provider != "gemini" {
			t.Errorf("Expected provider 'gemini', got '%s'", config.Provider)
		}
		if config.AuthMethod != AuthMethodOAuth {
			t.Errorf("Expected auth method %s, got %s", AuthMethodOAuth, config.AuthMethod)
		}
		if len(config.RequiredScopes) == 0 {
			t.Error("Expected required scopes to be set")
		}
	})

	t.Run("Qwen Config", func(t *testing.T) {
		config := QwenOAuthConfig()
		if config.Provider != "qwen" {
			t.Errorf("Expected provider 'qwen', got '%s'", config.Provider)
		}
		if config.AuthMethod != AuthMethodOAuth {
			t.Errorf("Expected auth method %s, got %s", AuthMethodOAuth, config.AuthMethod)
		}
		if len(config.RequiredScopes) == 0 {
			t.Error("Expected required scopes to be set")
		}
	})
}

// TestOAuthConfigsValidation tests OAuth config creation and validation
func TestOAuthConfigsValidation(t *testing.T) {
	t.Run("Anthropic OAuth Config", func(t *testing.T) {
		config := GetAnthropicOAuthConfig("client-id", "client-secret", "http://localhost/callback")

		if config.ClientID != "client-id" {
			t.Errorf("Expected client ID 'client-id', got '%s'", config.ClientID)
		}
		if config.ClientSecret != "client-secret" {
			t.Errorf("Expected client secret 'client-secret', got '%s'", config.ClientSecret)
		}
		if config.RedirectURL != "http://localhost/callback" {
			t.Errorf("Expected redirect URL 'http://localhost/callback', got '%s'", config.RedirectURL)
		}
		if config.AuthURL == "" {
			t.Error("Expected auth URL to be set")
		}
		if config.TokenURL == "" {
			t.Error("Expected token URL to be set")
		}
		if len(config.Scopes) == 0 {
			t.Error("Expected scopes to be set")
		}
	})

	t.Run("Anthropic API Key Config", func(t *testing.T) {
		config := GetAnthropicAPIKeyAuthConfig("sk-ant-test-key")

		if config.Method != AuthMethodAPIKey {
			t.Errorf("Expected method %s, got %s", AuthMethodAPIKey, config.Method)
		}
		if config.APIKey != "sk-ant-test-key" {
			t.Errorf("Expected API key 'sk-ant-test-key', got '%s'", config.APIKey)
		}
		if config.BaseURL == "" {
			t.Error("Expected base URL to be set")
		}
	})
}

// BenchmarkMemoryTokenStorage benchmarks memory token storage operations
func BenchmarkMemoryTokenStorage(b *testing.B) {
	storage := NewMemoryTokenStorage()
	token := &OAuthConfig{
		ClientID:    "bench-client",
		AccessToken: "bench-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	b.Run("Store", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = storage.StoreToken(fmt.Sprintf("provider-%d", i), token)
		}
	})

	b.Run("Retrieve", func(b *testing.B) {
		_ = storage.StoreToken("bench-provider", token)
		for i := 0; i < b.N; i++ {
			_, _ = storage.RetrieveToken("bench-provider")
		}
	})
}
