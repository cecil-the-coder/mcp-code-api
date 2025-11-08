package auth

import (
	"context"
	"fmt"
)

// QwenOAuthConfig returns the OAuth configuration for Qwen
func QwenOAuthConfig() *ProviderAuthConfig {
	return &ProviderAuthConfig{
		Provider:    "qwen",
		AuthMethod:  AuthMethodOAuth,
		DisplayName: "Alibaba Qwen",
		Description: "Authenticate with Alibaba Qwen API using OAuth 2.0 or Device Flow",
		OAuthURL:    "https://qwen.alibabacloud.com",
		RequiredScopes: []string{
			"chat",
			"tools",
		},
		OptionalScopes: []string{
			"models",
			"usage",
			"fine-tune",
		},
	}
}

// QwenOAuthAuthenticator implements OAuth for Qwen
type QwenOAuthAuthenticator struct {
	*OAuthAuthenticatorImpl
}

// NewQwenOAuthAuthenticator creates a new Qwen OAuth authenticator
func NewQwenOAuthAuthenticator(storage TokenStorage) *QwenOAuthAuthenticator {
	return &QwenOAuthAuthenticator{
		OAuthAuthenticatorImpl: NewOAuthAuthenticator("qwen", storage),
	}
}

// StartOAuthFlow starts the Qwen OAuth flow
func (q *QwenOAuthAuthenticator) StartOAuthFlow(ctx context.Context, scopes []string) (string, error) {
	// Ensure required scopes are included
	requiredScopes := []string{"chat", "tools"}
	allScopes := mergeScopes(requiredScopes, scopes)

	return q.OAuthAuthenticatorImpl.StartOAuthFlow(ctx, allScopes)
}

// HandleCallback handles the Qwen OAuth callback
func (q *QwenOAuthAuthenticator) HandleCallback(ctx context.Context, code, state string) error {
	return q.OAuthAuthenticatorImpl.HandleCallback(ctx, code, state)
}

// ValidateToken validates a Qwen token
func (q *QwenOAuthAuthenticator) ValidateToken(ctx context.Context) error {
	if !q.IsAuthenticated() {
		return &AuthError{
			Provider: q.provider,
			Code:     ErrCodeTokenExpired,
			Message:  "Not authenticated",
		}
	}

	tokenInfo, err := q.GetTokenInfo()
	if err != nil {
		return err
	}

	// Check for required scopes
	requiredScopes := []string{"chat", "tools"}
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
				Provider: q.provider,
				Code:     ErrCodeScopeInsufficient,
				Message:  fmt.Sprintf("Missing required scope: %s", required),
			}
		}
	}

	return nil
}

// GetQwenOAuthConfig returns the OAuth configuration for Qwen
func GetQwenOAuthConfig(clientID, clientSecret, redirectURL string) *OAuthConfig {
	return &OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://oauth.qwen.com/authorize",
		TokenURL:     "https://oauth.qwen.com/token",
		RefreshURL:   "https://oauth.qwen.com/refresh",
		Scopes: []string{
			"chat",
			"tools",
			"models",
			"usage",
		},
		TokenType: "Bearer",
	}
}

// GetQwenAPIKeyAuthConfig returns an API key auth config for Qwen
func GetQwenAPIKeyAuthConfig(apiKey string) AuthConfig {
	return AuthConfig{
		Method:  AuthMethodAPIKey,
		APIKey:  apiKey,
		BaseURL: "https://dashscope.aliyuncs.com",
	}
}

// GetQwenDeviceFlowConfig returns config for Qwen Device Flow
func GetQwenDeviceFlowConfig(clientID, clientSecret string) *OAuthConfig {
	return &OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://oauth.qwen.com/device/authorize",
		TokenURL:     "https://oauth.qwen.com/token",
		RefreshURL:   "https://oauth.qwen.com/refresh",
		Scopes: []string{
			"chat",
			"tools",
			"models",
		},
		TokenType: "Bearer",
	}
}

// QwenDeviceFlow implements device flow for Qwen
type QwenDeviceFlow struct {
	authenticator   *QwenOAuthAuthenticator
	deviceCode      string
	userCode        string
	verificationURL string
	expiresIn       int64
	interval        int64
}

// NewQwenDeviceFlow creates a new device flow client
func NewQwenDeviceFlow(authenticator *QwenOAuthAuthenticator) *QwenDeviceFlow {
	return &QwenDeviceFlow{
		authenticator: authenticator,
		interval:      5, // Poll every 5 seconds
	}
}

// StartDeviceFlow initiates the device flow
func (qdf *QwenDeviceFlow) StartDeviceFlow(ctx context.Context) (*DeviceFlowResponse, error) {
	// TODO: Implement device flow initiation
	// This would make a request to get device code and user code
	return &DeviceFlowResponse{
		DeviceCode:      qdf.deviceCode,
		UserCode:        qdf.userCode,
		VerificationURL: qdf.verificationURL,
		ExpiresIn:       qdf.expiresIn,
		Interval:        qdf.interval,
	}, nil
}

// PollForToken polls for token completion
func (qdf *QwenDeviceFlow) PollForToken(ctx context.Context) error {
	// TODO: Implement token polling
	return nil
}

// DeviceFlowResponse represents the response from device flow initiation
type DeviceFlowResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int64  `json:"expires_in"`
	Interval        int64  `json:"interval"`
}
