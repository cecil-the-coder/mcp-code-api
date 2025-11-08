package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileTokenStorage implements TokenStorage using encrypted files
type FileTokenStorage struct {
	storageDir string
	gcm        cipher.AEAD
}

// NewFileTokenStorage creates a new file-based token storage
func NewFileTokenStorage(storageDir string, encryptionKey string) (*FileTokenStorage, error) {
	// Ensure storage directory exists
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Derive encryption key
	key := sha256.Sum256([]byte(encryptionKey))

	// Create AES cipher
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &FileTokenStorage{
		storageDir: storageDir,
		gcm:        gcm,
	}, nil
}

// StoreToken stores an OAuth token encrypted on disk
func (fts *FileTokenStorage) StoreToken(provider string, token *OAuthConfig) error {
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	// Add timestamp to token
	if token.ExpiresAt.IsZero() && token.AccessToken != "" {
		// Set default expiration to 1 hour from now if not specified
		token.ExpiresAt = time.Now().Add(1 * time.Hour)
	}

	// Serialize token to JSON
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Encrypt data
	encrypted, err := fts.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Write to file
	filename := filepath.Join(fts.storageDir, fts.sanitizeFilename(provider)+".token")
	if err := os.WriteFile(filename, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// RetrieveToken retrieves and decrypts an OAuth token from disk
func (fts *FileTokenStorage) RetrieveToken(provider string) (*OAuthConfig, error) {
	filename := filepath.Join(fts.storageDir, fts.sanitizeFilename(provider)+".token")

	encrypted, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("token not found for provider: %s", provider)
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Decrypt data
	data, err := fts.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Deserialize token
	var token OAuthConfig
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	// Check if token is expired
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		// Token is expired, delete it and return error
		_ = fts.DeleteToken(provider)
		return nil, fmt.Errorf("token expired for provider: %s", provider)
	}

	return &token, nil
}

// DeleteToken removes a stored token
func (fts *FileTokenStorage) DeleteToken(provider string) error {
	filename := filepath.Join(fts.storageDir, fts.sanitizeFilename(provider)+".token")
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}
	return nil
}

// ListTokens returns a list of all stored provider tokens
func (fts *FileTokenStorage) ListTokens() ([]string, error) {
	files, err := os.ReadDir(fts.storageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var providers []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".token" {
			provider := file.Name()[:len(file.Name())-6] // Remove ".token" extension
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

// IsTokenValid checks if a token exists and is not expired
func (fts *FileTokenStorage) IsTokenValid(provider string) bool {
	token, err := fts.RetrieveToken(provider)
	if err != nil {
		return false
	}
	return token != nil && token.AccessToken != ""
}

// Helper methods

func (fts *FileTokenStorage) encrypt(data []byte) ([]byte, error) {
	// Generate nonce
	nonce := make([]byte, fts.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	encrypted := fts.gcm.Seal(nonce, nonce, data, nil)
	return encrypted, nil
}

func (fts *FileTokenStorage) decrypt(encrypted []byte) ([]byte, error) {
	// Extract nonce
	nonceSize := fts.gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	// Decrypt data
	data, err := fts.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return data, nil
}

func (fts *FileTokenStorage) sanitizeFilename(provider string) string {
	// Simple sanitization - replace invalid characters with underscores
	invalid := []string{"/", "\\", ":", "*", "?", `"`, "<", ">", "|"}
	result := provider
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// MemoryTokenStorage implements TokenStorage in memory for testing or temporary use
type MemoryTokenStorage struct {
	tokens map[string]*OAuthConfig
}

// NewMemoryTokenStorage creates a new memory-based token storage
func NewMemoryTokenStorage() *MemoryTokenStorage {
	return &MemoryTokenStorage{
		tokens: make(map[string]*OAuthConfig),
	}
}

// StoreToken stores an OAuth token in memory
func (mts *MemoryTokenStorage) StoreToken(provider string, token *OAuthConfig) error {
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	// Create a copy to avoid external mutation
	tokenCopy := *token
	mts.tokens[provider] = &tokenCopy
	return nil
}

// RetrieveToken retrieves an OAuth token from memory
func (mts *MemoryTokenStorage) RetrieveToken(provider string) (*OAuthConfig, error) {
	token, exists := mts.tokens[provider]
	if !exists || token == nil {
		return nil, fmt.Errorf("token not found for provider: %s", provider)
	}

	// Check if token is expired
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		// Token is expired, delete it
		delete(mts.tokens, provider)
		return nil, fmt.Errorf("token expired for provider: %s", provider)
	}

	// Return a copy to avoid external mutation
	tokenCopy := *token
	return &tokenCopy, nil
}

// DeleteToken removes a stored token from memory
func (mts *MemoryTokenStorage) DeleteToken(provider string) error {
	delete(mts.tokens, provider)
	return nil
}

// ListTokens returns a list of all stored provider tokens in memory
func (mts *MemoryTokenStorage) ListTokens() ([]string, error) {
	var providers []string
	for provider := range mts.tokens {
		providers = append(providers, provider)
	}
	return providers, nil
}

// IsTokenValid checks if a token exists and is not expired in memory
func (mts *MemoryTokenStorage) IsTokenValid(provider string) bool {
	token, err := mts.RetrieveToken(provider)
	if err != nil {
		return false
	}
	return token != nil && token.AccessToken != ""
}
