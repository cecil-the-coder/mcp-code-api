package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthAuthenticatorImpl implements OAuthAuthenticator
type OAuthAuthenticatorImpl struct {
	provider string
	config   *OAuthConfig
	storage  TokenStorage
	client   *http.Client
	state    string
	lastAuth time.Time
	isAuth   bool
}

// NewOAuthAuthenticator creates a new OAuth authenticator
func NewOAuthAuthenticator(provider string, storage TokenStorage) *OAuthAuthenticatorImpl {
	return &OAuthAuthenticatorImpl{
		provider: provider,
		storage:  storage,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Authenticate performs authentication with the given config
func (a *OAuthAuthenticatorImpl) Authenticate(ctx context.Context, config AuthConfig) error {
	if config.Method != AuthMethodOAuth {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeInvalidConfig,
			Message:  "OAuth authenticator only supports OAuth method",
		}
	}

	if config.OAuthConfig == nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeInvalidConfig,
			Message:  "OAuth config is required",
		}
	}

	// Store the config
	a.config = config.OAuthConfig

	// Check if we have a stored token
	if storedToken, err := a.storage.RetrieveToken(a.provider); err == nil && storedToken != nil {
		a.config = storedToken
		if !a.isTokenExpired(storedToken) {
			a.isAuth = true
			a.lastAuth = time.Now()
			return nil
		}
		// Token expired, try to refresh
		if err := a.RefreshToken(ctx); err != nil {
			// Refresh failed, need full OAuth flow
			return &AuthError{
				Provider: a.provider,
				Code:     ErrCodeTokenExpired,
				Message:  "Stored token expired and refresh failed",
				Details:  err.Error(),
			}
		}
	} else {
		// No stored token, need OAuth flow
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeOAuthFlowFailed,
			Message:  "OAuth flow required - call StartOAuthFlow",
		}
	}

	return nil
}

// IsAuthenticated checks if currently authenticated
func (a *OAuthAuthenticatorImpl) IsAuthenticated() bool {
	if !a.isAuth || a.config == nil {
		return false
	}
	return !a.isTokenExpired(a.config)
}

// GetToken returns the current authentication token
func (a *OAuthAuthenticatorImpl) GetToken() (string, error) {
	if !a.IsAuthenticated() {
		return "", &AuthError{
			Provider: a.provider,
			Code:     ErrCodeTokenExpired,
			Message:  "Not authenticated",
		}
	}
	return a.config.AccessToken, nil
}

// RefreshToken refreshes the authentication token
func (a *OAuthAuthenticatorImpl) RefreshToken(ctx context.Context) error {
	if a.config == nil || a.config.RefreshToken == "" {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeRefreshFailed,
			Message:  "No refresh token available",
		}
	}

	// Use token URL or refresh URL
	tokenURL := a.config.RefreshURL
	if tokenURL == "" {
		tokenURL = a.config.TokenURL
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", a.config.RefreshToken)
	data.Set("client_id", a.config.ClientID)
	data.Set("client_secret", a.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeNetworkError,
			Message:  "Failed to create refresh request",
			Details:  err.Error(),
		}
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeNetworkError,
			Message:  "Network error during token refresh",
			Details:  err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeRefreshFailed,
			Message:  fmt.Sprintf("Token refresh failed with status %d", resp.StatusCode),
		}
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeRefreshFailed,
			Message:  "Failed to parse refresh response",
			Details:  err.Error(),
		}
	}

	// Update config with new token
	a.config.AccessToken = tokenResp.AccessToken
	a.config.RefreshToken = tokenResp.RefreshToken
	if tokenResp.ExpiresIn > 0 {
		a.config.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	a.config.TokenType = tokenResp.TokenType

	// Store the updated token
	if err := a.storage.StoreToken(a.provider, a.config); err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeNetworkError,
			Message:  "Failed to store refreshed token",
			Details:  err.Error(),
		}
	}

	a.isAuth = true
	a.lastAuth = time.Now()
	return nil
}

// Logout clears authentication state
func (a *OAuthAuthenticatorImpl) Logout(ctx context.Context) error {
	if err := a.storage.DeleteToken(a.provider); err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeNetworkError,
			Message:  "Failed to delete stored token",
			Details:  err.Error(),
		}
	}

	a.config = nil
	a.isAuth = false
	a.lastAuth = time.Time{}
	return nil
}

// GetAuthMethod returns the authentication method type
func (a *OAuthAuthenticatorImpl) GetAuthMethod() AuthMethod {
	return AuthMethodOAuth
}

// StartOAuthFlow initiates the OAuth flow and returns the auth URL
func (a *OAuthAuthenticatorImpl) StartOAuthFlow(ctx context.Context, scopes []string) (string, error) {
	if a.config == nil {
		return "", &AuthError{
			Provider: a.provider,
			Code:     ErrCodeInvalidConfig,
			Message:  "OAuth config not set",
		}
	}

	// Generate random state
	state, err := a.generateRandomState()
	if err != nil {
		return "", &AuthError{
			Provider: a.provider,
			Code:     ErrCodeOAuthFlowFailed,
			Message:  "Failed to generate OAuth state",
			Details:  err.Error(),
		}
	}
	a.state = state

	// Build authorization URL
	authURL, err := url.Parse(a.config.AuthURL)
	if err != nil {
		return "", &AuthError{
			Provider: a.provider,
			Code:     ErrCodeInvalidConfig,
			Message:  "Invalid auth URL",
			Details:  err.Error(),
		}
	}

	// Add OAuth parameters
	params := authURL.Query()
	params.Set("response_type", "code")
	params.Set("client_id", a.config.ClientID)
	params.Set("redirect_uri", a.config.RedirectURL)
	params.Set("scope", strings.Join(scopes, " "))
	params.Set("state", state)
	if a.config.TokenType != "" {
		params.Set("token_type", a.config.TokenType)
	}

	authURL.RawQuery = params.Encode()
	return authURL.String(), nil
}

// HandleCallback processes the OAuth callback
func (a *OAuthAuthenticatorImpl) HandleCallback(ctx context.Context, code, state string) error {
	if a.state != state {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeOAuthFlowFailed,
			Message:  "Invalid OAuth state",
		}
	}

	if a.config == nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeInvalidConfig,
			Message:  "OAuth config not set",
		}
	}

	// Exchange authorization code for access token
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", a.config.RedirectURL)
	data.Set("client_id", a.config.ClientID)
	data.Set("client_secret", a.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", a.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeNetworkError,
			Message:  "Failed to create token request",
			Details:  err.Error(),
		}
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeNetworkError,
			Message:  "Network error during token exchange",
			Details:  err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeOAuthFlowFailed,
			Message:  fmt.Sprintf("Token exchange failed with status %d", resp.StatusCode),
		}
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeOAuthFlowFailed,
			Message:  "Failed to parse token response",
			Details:  err.Error(),
		}
	}

	// Update config with received token
	a.config.AccessToken = tokenResp.AccessToken
	a.config.RefreshToken = tokenResp.RefreshToken
	if tokenResp.ExpiresIn > 0 {
		a.config.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	a.config.TokenType = tokenResp.TokenType

	// Store the token
	if err := a.storage.StoreToken(a.provider, a.config); err != nil {
		return &AuthError{
			Provider: a.provider,
			Code:     ErrCodeNetworkError,
			Message:  "Failed to store token",
			Details:  err.Error(),
		}
	}

	a.isAuth = true
	a.lastAuth = time.Now()
	return nil
}

// IsOAuthEnabled checks if OAuth is properly configured
func (a *OAuthAuthenticatorImpl) IsOAuthEnabled() bool {
	return a.config != nil &&
		a.config.ClientID != "" &&
		a.config.ClientSecret != "" &&
		a.config.AuthURL != "" &&
		a.config.TokenURL != ""
}

// GetTokenInfo returns detailed token information
func (a *OAuthAuthenticatorImpl) GetTokenInfo() (*TokenInfo, error) {
	if !a.IsAuthenticated() {
		return nil, &AuthError{
			Provider: a.provider,
			Code:     ErrCodeTokenExpired,
			Message:  "Not authenticated",
		}
	}

	expiresIn := int64(0)
	if !a.config.ExpiresAt.IsZero() {
		expiresIn = int64(time.Until(a.config.ExpiresAt).Seconds())
		if expiresIn < 0 {
			expiresIn = 0
		}
	}

	return &TokenInfo{
		AccessToken:  a.config.AccessToken,
		RefreshToken: a.config.RefreshToken,
		TokenType:    a.config.TokenType,
		ExpiresAt:    a.config.ExpiresAt,
		Scopes:       a.config.Scopes,
		IsExpired:    a.isTokenExpired(a.config),
		ExpiresIn:    expiresIn,
	}, nil
}

// Helper methods

func (a *OAuthAuthenticatorImpl) isTokenExpired(config *OAuthConfig) bool {
	if config.ExpiresAt.IsZero() {
		return false // No expiration set, assume not expired
	}
	return time.Now().After(config.ExpiresAt.Add(-5 * time.Minute)) // Refresh 5 minutes early
}

func (a *OAuthAuthenticatorImpl) generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// OAuthTokenResponse represents the token response from OAuth servers
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}
