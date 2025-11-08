package validation

import (
	"os/exec"
	"sync"
)

// Initialize the tool cache on package import
func init() {
	// Prewarm the cache in the background
	go globalToolCache.PrewarmCache()
}

// ToolCache caches the availability of external validation tools
type ToolCache struct {
	available map[string]bool
	mu        sync.RWMutex
}

// Global tool cache instance
var globalToolCache = &ToolCache{
	available: make(map[string]bool),
}

// GetToolCache returns the global tool cache instance
func GetToolCache() *ToolCache {
	return globalToolCache
}

// IsAvailable checks if a tool is available on the system
// Results are cached to avoid repeated exec.LookPath calls
func (c *ToolCache) IsAvailable(tool string) bool {
	// Try to read from cache first
	c.mu.RLock()
	if available, ok := c.available[tool]; ok {
		c.mu.RUnlock()
		return available
	}
	c.mu.RUnlock()

	// Check tool availability
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if available, ok := c.available[tool]; ok {
		return available
	}

	// Check if tool exists in PATH
	_, err := exec.LookPath(tool)
	available := err == nil

	c.available[tool] = available
	return available
}

// Clear clears the cache (useful for testing)
func (c *ToolCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.available = make(map[string]bool)
}

// PrewarmCache checks availability of common validation tools
// Call this at startup to populate the cache
func (c *ToolCache) PrewarmCache() {
	tools := []string{
		// Python
		"python3",
		"python",
		"pylint",
		"ruff",

		// JavaScript/TypeScript
		"node",
		"eslint",
		"tsc",

		// Go
		"gofmt",
		"golangci-lint",
		"staticcheck",

		// Other
		"rustc",
		"cargo",
		"javac",
		"clang",
		"gcc",
	}

	// Check all tools in parallel
	var wg sync.WaitGroup
	for _, tool := range tools {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			c.IsAvailable(t)
		}(tool)
	}
	wg.Wait()
}
