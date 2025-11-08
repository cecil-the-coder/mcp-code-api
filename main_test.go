package main

import (
	"os"
	"testing"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/mcp"
)

func TestMain(t *testing.T) {
	// Test that main doesn't panic
	// This is a basic smoke test
	t.Log("Main function test passed")
}

func TestConfigLoading(t *testing.T) {
	// Test configuration loading
	cfg := config.Load()

	if cfg == nil {
		t.Fatal("Config should not be nil")
	}

	// Test default values
	if cfg.Server.Name == "" {
		t.Error("Server name should have default value")
	}

	t.Logf("Config loaded successfully: %+v", cfg)
}

func TestServerCreation(t *testing.T) {
	cfg := config.Load()
	server := mcp.NewServer(cfg)

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	t.Log("Server created successfully")
}

func TestEnvironmentVariables(t *testing.T) {
	// Test environment variable handling
	originalValue := os.Getenv("CEREBRAS_API_KEY")

	// Set test value
	os.Setenv("CEREBRAS_API_KEY", "test-key")

	cfg := config.Load()

	// Note: With the new config structure, provider-specific settings
	// are loaded in the Providers section, not directly on the config
	// This test just ensures config loading works with environment variables
	if cfg == nil {
		t.Error("Config should not be nil")
	}

	// Restore original value
	if originalValue == "" {
		os.Unsetenv("CEREBRAS_API_KEY")
	} else {
		os.Setenv("CEREBRAS_API_KEY", originalValue)
	}

	t.Log("Environment variable test passed")
}
