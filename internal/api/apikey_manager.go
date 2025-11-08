package api

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
)

// APIKeyManager manages multiple API keys with load balancing and failover
type APIKeyManager struct {
	providerName string
	keys         []string
	currentIndex uint32 // Atomic counter for round-robin
	keyHealth    map[string]*keyHealth
	mu           sync.RWMutex
}

// keyHealth tracks the health status of an individual API key
type keyHealth struct {
	failureCount int
	lastFailure  time.Time
	lastSuccess  time.Time
	isHealthy    bool
	backoffUntil time.Time
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(providerName string, keys []string) *APIKeyManager {
	if len(keys) == 0 {
		return nil
	}

	manager := &APIKeyManager{
		providerName: providerName,
		keys:         keys,
		currentIndex: 0,
		keyHealth:    make(map[string]*keyHealth),
	}

	// Initialize health tracking for all keys
	for _, key := range keys {
		manager.keyHealth[key] = &keyHealth{
			isHealthy:   true,
			lastSuccess: time.Now(),
		}
	}

	logger.Infof("APIKeyManager initialized for %s with %d key(s)", providerName, len(keys))
	return manager
}

// GetNextKey returns the next available API key using round-robin load balancing
// It skips keys that are currently in backoff due to failures
func (m *APIKeyManager) GetNextKey() (string, error) {
	if len(m.keys) == 0 {
		return "", fmt.Errorf("no API keys configured for %s", m.providerName)
	}

	// Single key fast path
	if len(m.keys) == 1 {
		m.mu.RLock()
		health := m.keyHealth[m.keys[0]]
		m.mu.RUnlock()

		if m.isKeyAvailable(m.keys[0], health) {
			return m.keys[0], nil
		}
		return "", fmt.Errorf("only API key for %s is unavailable (in backoff)", m.providerName)
	}

	// Try all keys in round-robin order
	startIndex := atomic.AddUint32(&m.currentIndex, 1) % uint32(len(m.keys))

	for i := 0; i < len(m.keys); i++ {
		index := (int(startIndex) + i) % len(m.keys)
		key := m.keys[index]

		m.mu.RLock()
		health := m.keyHealth[key]
		m.mu.RUnlock()

		if m.isKeyAvailable(key, health) {
			logger.Debugf("%s: Selected key #%d/%d", m.providerName, index+1, len(m.keys))
			return key, nil
		}
	}

	return "", fmt.Errorf("all %d API keys for %s are currently unavailable", len(m.keys), m.providerName)
}

// isKeyAvailable checks if a key is available (not in backoff)
func (m *APIKeyManager) isKeyAvailable(key string, health *keyHealth) bool {
	if health == nil {
		return true
	}

	// Check if key is in backoff period
	if time.Now().Before(health.backoffUntil) {
		return false
	}

	return true
}

// ReportSuccess reports that an API call succeeded with this key
func (m *APIKeyManager) ReportSuccess(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	health, exists := m.keyHealth[key]
	if !exists {
		return
	}

	health.lastSuccess = time.Now()
	health.failureCount = 0
	health.isHealthy = true
	health.backoffUntil = time.Time{} // Clear backoff

	logger.Debugf("%s: API key succeeded (healthy)", m.providerName)
}

// ReportFailure reports that an API call failed with this key
// Implements exponential backoff for repeatedly failing keys
func (m *APIKeyManager) ReportFailure(key string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	health, exists := m.keyHealth[key]
	if !exists {
		return
	}

	health.lastFailure = time.Now()
	health.failureCount++

	// Exponential backoff: 1s, 2s, 4s, 8s, max 60s
	backoffSeconds := 1 << uint(min(health.failureCount-1, 6))
	if backoffSeconds > 60 {
		backoffSeconds = 60
	}
	health.backoffUntil = time.Now().Add(time.Duration(backoffSeconds) * time.Second)

	// Mark as unhealthy after 3 consecutive failures
	if health.failureCount >= 3 {
		health.isHealthy = false
		logger.Warnf("%s: API key marked unhealthy after %d failures (backoff: %ds)",
			m.providerName, health.failureCount, backoffSeconds)
	} else {
		logger.Debugf("%s: API key failure %d (backoff: %ds)",
			m.providerName, health.failureCount, backoffSeconds)
	}
}

// ExecuteWithFailover attempts an operation with automatic failover to next key on failure
// The operation function should accept an API key and return (result, error)
func (m *APIKeyManager) ExecuteWithFailover(operation func(apiKey string) (string, error)) (string, error) {
	if len(m.keys) == 0 {
		return "", fmt.Errorf("no API keys configured for %s", m.providerName)
	}

	var lastErr error
	attemptsLimit := min(len(m.keys), 3) // Try up to 3 keys or all keys, whichever is less

	for attempt := 0; attempt < attemptsLimit; attempt++ {
		// Get next available key
		key, err := m.GetNextKey()
		if err != nil {
			// All keys exhausted or unavailable
			if lastErr != nil {
				return "", fmt.Errorf("%s: all API keys failed, last error: %w", m.providerName, lastErr)
			}
			return "", err
		}

		// Execute the operation
		result, err := operation(key)
		if err != nil {
			lastErr = err
			m.ReportFailure(key, err)
			logger.Warnf("%s: Attempt %d/%d failed with key, trying next key: %v",
				m.providerName, attempt+1, attemptsLimit, err)
			continue
		}

		// Success!
		m.ReportSuccess(key)
		if attempt > 0 {
			logger.Infof("%s: Request succeeded on attempt %d/%d (failover successful)",
				m.providerName, attempt+1, attemptsLimit)
		}
		return result, nil
	}

	// All attempts failed
	return "", fmt.Errorf("%s: all %d failover attempts failed, last error: %w",
		m.providerName, attemptsLimit, lastErr)
}

// GetStatus returns the current health status of all keys
func (m *APIKeyManager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]interface{})
	status["provider"] = m.providerName
	status["total_keys"] = len(m.keys)

	healthyCount := 0
	keyStatuses := make([]map[string]interface{}, 0, len(m.keys))

	for i, key := range m.keys {
		health := m.keyHealth[key]
		keyMasked := maskAPIKey(key)

		keyStatus := map[string]interface{}{
			"index":         i + 1,
			"key_masked":    keyMasked,
			"healthy":       health.isHealthy,
			"failure_count": health.failureCount,
			"last_success":  health.lastSuccess.Format(time.RFC3339),
		}

		if !health.lastFailure.IsZero() {
			keyStatus["last_failure"] = health.lastFailure.Format(time.RFC3339)
		}

		if time.Now().Before(health.backoffUntil) {
			keyStatus["in_backoff"] = true
			keyStatus["backoff_until"] = health.backoffUntil.Format(time.RFC3339)
		}

		if health.isHealthy {
			healthyCount++
		}

		keyStatuses = append(keyStatuses, keyStatus)
	}

	status["healthy_keys"] = healthyCount
	status["unhealthy_keys"] = len(m.keys) - healthyCount
	status["keys"] = keyStatuses

	return status
}

// maskAPIKey masks an API key for display (shows first 8 and last 4 characters)
func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
