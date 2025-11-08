package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AuthManagerImpl manages authentication for multiple providers
type AuthManagerImpl struct {
	authenticators map[string]Authenticator
	storage        TokenStorage
	mutex          sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(storage TokenStorage) *AuthManagerImpl {
	return &AuthManagerImpl{
		authenticators: make(map[string]Authenticator),
		storage:        storage,
	}
}

// RegisterAuthenticator registers an authenticator for a provider
func (am *AuthManagerImpl) RegisterAuthenticator(provider string, authenticator Authenticator) error {
	if provider == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if authenticator == nil {
		return fmt.Errorf("authenticator cannot be nil")
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.authenticators[provider] = authenticator
	return nil
}

// GetAuthenticator returns the authenticator for a provider
func (am *AuthManagerImpl) GetAuthenticator(provider string) (Authenticator, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	authenticator, exists := am.authenticators[provider]
	if !exists {
		return nil, &AuthError{
			Provider: provider,
			Code:     ErrCodeProviderUnavailable,
			Message:  "No authenticator registered for provider",
		}
	}

	return authenticator, nil
}

// Authenticate authenticates a provider
func (am *AuthManagerImpl) Authenticate(ctx context.Context, provider string, config AuthConfig) error {
	authenticator, err := am.GetAuthenticator(provider)
	if err != nil {
		return err
	}

	return authenticator.Authenticate(ctx, config)
}

// IsAuthenticated checks if a provider is authenticated
func (am *AuthManagerImpl) IsAuthenticated(provider string) bool {
	authenticator, err := am.GetAuthenticator(provider)
	if err != nil {
		return false
	}

	return authenticator.IsAuthenticated()
}

// Logout logs out a provider
func (am *AuthManagerImpl) Logout(ctx context.Context, provider string) error {
	authenticator, err := am.GetAuthenticator(provider)
	if err != nil {
		return err
	}

	return authenticator.Logout(ctx)
}

// RefreshAllTokens refreshes all tokens that need it
func (am *AuthManagerImpl) RefreshAllTokens(ctx context.Context) error {
	am.mutex.RLock()
	authenticators := make(map[string]Authenticator)
	for provider, auth := range am.authenticators {
		authenticators[provider] = auth
	}
	am.mutex.RUnlock()

	var errors []error
	for provider, authenticator := range authenticators {
		if authenticator.IsAuthenticated() {
			if err := authenticator.RefreshToken(ctx); err != nil {
				errors = append(errors, fmt.Errorf("failed to refresh %s: %w", provider, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("refresh errors: %v", errors)
	}

	return nil
}

// GetAuthenticatedProviders returns a list of authenticated providers
func (am *AuthManagerImpl) GetAuthenticatedProviders() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var authenticated []string
	for provider, authenticator := range am.authenticators {
		if authenticator.IsAuthenticated() {
			authenticated = append(authenticated, provider)
		}
	}

	return authenticated
}

// GetAuthStatus returns the authentication status for all providers
func (am *AuthManagerImpl) GetAuthStatus() map[string]*AuthState {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	status := make(map[string]*AuthState)
	for provider, authenticator := range am.authenticators {
		state := &AuthState{
			Provider: provider,
		}

		if auth := authenticator.IsAuthenticated(); auth {
			state.Authenticated = true
			state.Method = authenticator.GetAuthMethod()
			state.LastAuth = time.Now() // We don't track this per provider

			// Get token info if available
			if oauthAuth, ok := authenticator.(OAuthAuthenticator); ok {
				if tokenInfo, err := oauthAuth.GetTokenInfo(); err == nil {
					state.ExpiresAt = tokenInfo.ExpiresAt
					state.CanRefresh = tokenInfo.RefreshToken != ""
				}
			}
		}

		status[provider] = state
	}

	return status
}

// CleanupExpired removes expired tokens and cleans up authenticators
func (am *AuthManagerImpl) CleanupExpired() error {
	// Get stored tokens
	storedTokens, err := am.storage.ListTokens()
	if err != nil {
		return fmt.Errorf("failed to list stored tokens: %w", err)
	}

	var expired []string
	for _, provider := range storedTokens {
		if !am.storage.IsTokenValid(provider) {
			expired = append(expired, provider)
		}
	}

	// Remove expired tokens
	for _, provider := range expired {
		if err := am.storage.DeleteToken(provider); err != nil {
			return fmt.Errorf("failed to delete expired token for %s: %w", provider, err)
		}
	}

	return nil
}

// ForEachAuthenticated executes a function for each authenticated provider
func (am *AuthManagerImpl) ForEachAuthenticated(ctx context.Context, fn func(provider string, authenticator Authenticator) error) error {
	authenticated := am.GetAuthenticatedProviders()

	for _, provider := range authenticated {
		authenticator, err := am.GetAuthenticator(provider)
		if err != nil {
			continue
		}

		if err := fn(provider, authenticator); err != nil {
			return fmt.Errorf("error processing %s: %w", provider, err)
		}
	}

	return nil
}

// GetTokenInfo returns token information for a specific provider
func (am *AuthManagerImpl) GetTokenInfo(provider string) (*TokenInfo, error) {
	authenticator, err := am.GetAuthenticator(provider)
	if err != nil {
		return nil, err
	}

	if oauthAuth, ok := authenticator.(OAuthAuthenticator); ok {
		return oauthAuth.GetTokenInfo()
	}

	return nil, &AuthError{
		Provider: provider,
		Code:     ErrCodeInvalidConfig,
		Message:  "Provider does not support OAuth token info",
	}
}

// StartOAuthFlow starts OAuth flow for a provider
func (am *AuthManagerImpl) StartOAuthFlow(ctx context.Context, provider string, scopes []string) (string, error) {
	authenticator, err := am.GetAuthenticator(provider)
	if err != nil {
		return "", err
	}

	if oauthAuth, ok := authenticator.(OAuthAuthenticator); ok {
		return oauthAuth.StartOAuthFlow(ctx, scopes)
	}

	return "", &AuthError{
		Provider: provider,
		Code:     ErrCodeInvalidConfig,
		Message:  "Provider does not support OAuth flow",
	}
}

// HandleOAuthCallback handles OAuth callback for a provider
func (am *AuthManagerImpl) HandleOAuthCallback(ctx context.Context, provider string, code, state string) error {
	authenticator, err := am.GetAuthenticator(provider)
	if err != nil {
		return err
	}

	if oauthAuth, ok := authenticator.(OAuthAuthenticator); ok {
		return oauthAuth.HandleCallback(ctx, code, state)
	}

	return &AuthError{
		Provider: provider,
		Code:     ErrCodeInvalidConfig,
		Message:  "Provider does not support OAuth callback",
	}
}
