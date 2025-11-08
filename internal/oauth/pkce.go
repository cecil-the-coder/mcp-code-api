package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCEParams holds PKCE parameters for OAuth flow
// PKCE (Proof Key for Code Exchange) prevents authorization code interception attacks
// Defined in RFC 7636: https://tools.ietf.org/html/rfc7636
type PKCEParams struct {
	// CodeVerifier is a high-entropy cryptographic random string (43-128 characters)
	CodeVerifier string

	// CodeChallenge is the Base64URL-encoded SHA256 hash of the code verifier
	CodeChallenge string

	// State is a random string for CSRF protection
	State string
}

// GeneratePKCEParams generates PKCE parameters for OAuth authorization code flow
// This implements the S256 (SHA-256) code challenge method
func GeneratePKCEParams() (*PKCEParams, error) {
	// Generate code verifier (43-128 characters, we use 32 bytes = 43 base64url chars)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate code challenge using SHA256
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Generate state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	return &PKCEParams{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		State:         state,
	}, nil
}

// ValidateState validates that the state parameter matches the expected value
// This prevents CSRF attacks
func ValidateState(expected, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}
	return expected == actual
}
