package interactive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
)

// configureCline configures Cline IDE integration
func configureCline() error {
	paths := config.GetIDEPaths()
	rulesDir := paths.Cline

	if rulesDir == "" {
		return fmt.Errorf("could not determine Cline rules directory")
	}

	// Create rules directory if it doesn't exist
	if err := os.MkdirAll(rulesDir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create Cline rules directory: %w", err)
	}

	// Get MCP rules
	rules := config.GetMCPRules()
	// Create rules file
	rulesFile := filepath.Join(rulesDir, config.DefaultRulesFile)
	if err := os.WriteFile(rulesFile, []byte(rules.Comment), 0644); err != nil {
		return fmt.Errorf("failed to create Cline rules file: %w", err)
	}

	// Check for existing user rules file
	userRulesFile := filepath.Join(rulesDir, config.DefaultUserRulesFile)
	if config.FileExists(userRulesFile) {
		// Update existing file with MCP rules
		content, err := os.ReadFile(userRulesFile)
		if err != nil {
			return fmt.Errorf("failed to read existing user rules file: %w", err)
		}

		// In a real implementation, you would merge with existing content
		updatedContent := rules.Comment + "\n\n" + strings.TrimSpace(string(content))
		if err := os.WriteFile(userRulesFile, []byte(updatedContent), 0644); err != nil {
			return fmt.Errorf("failed to update user rules file: %w", err)
		}
	} else {
		// Create new user rules file
		if err := os.WriteFile(userRulesFile, []byte(rules.Comment), 0644); err != nil {
			return fmt.Errorf("failed to create user rules file: %w", err)
		}
	}

	fmt.Printf("✅ Cline configured successfully\n")
	fmt.Printf("📁 Rules file: %s\n", rulesFile)
	fmt.Printf("📝 User rules file: %s\n", userRulesFile)
	fmt.Println()
	fmt.Println("Instructions for Cline:")
	fmt.Println("1. Restart Cline")
	fmt.Println("2. The 'write' tool will be available for all code operations")
	fmt.Println("3. Use it for file creation, editing, and generation")

	return nil
}

// configureCursor configures Cursor IDE integration
func configureCursor() error {
	paths := config.GetIDEPaths()
	rulesDir := paths.Cursor

	if rulesDir == "" {
		return fmt.Errorf("could not determine Cursor config directory")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(rulesDir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create Cursor config directory: %w", err)
	}

	// Get MCP rules
	rules := config.GetMCPRules()
	// Create user rules file
	userRulesFile := filepath.Join(rulesDir, config.DefaultUserRulesFile)
	if err := os.WriteFile(userRulesFile, []byte(rules.Comment), 0644); err != nil {
		return fmt.Errorf("failed to create Cursor rules file: %w", err)
	}

	fmt.Printf("✅ Cursor configured successfully\n")
	fmt.Printf("📝 Rules file: %s\n", userRulesFile)
	fmt.Println()
	fmt.Println("Instructions for Cursor:")
	fmt.Println("1. Copy the following to Cursor → Settings → Developer → User Rules:")
	fmt.Println()
	fmt.Println(rules.Comment)
	fmt.Println()
	fmt.Println("2. Save and restart Cursor")
	fmt.Println("3. The 'write' tool will be available for all code operations")
	fmt.Println("4. Use it for file creation, editing, and generation")

	return nil
}

// configureVSCode configures VS Code integration
func configureVSCode() error {
	paths := config.GetIDEPaths()
	extensionsFile := paths.VSCode

	if extensionsFile == "" {
		return fmt.Errorf("could not determine VS Code extensions file")
	}

	// Create config directory if it doesn't exist
	dir := filepath.Dir(extensionsFile)
	if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create VS Code config directory: %w", err)
	}

	// Read existing extensions.json or create new one
	var content string
	if config.FileExists(extensionsFile) {
		existingContent, err := os.ReadFile(extensionsFile)
		if err != nil {
			return fmt.Errorf("failed to read existing extensions file: %w", err)
		}

		// Parse existing JSON and add MCP server
		var extensions map[string]interface{}
		if err := json.Unmarshal([]byte(existingContent), &extensions); err != nil {
			return fmt.Errorf("failed to parse existing extensions file: %w", err)
		}
		extensions["mcpServers"] = map[string]interface{}{
			"mcp-code-api": map[string]interface{}{
				"command": "mcp-code-api",
				"args":    []string{"server"},
			},
		}
		contentBytes, err := json.MarshalIndent(extensions, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal extensions file: %w", err)
		}
		content = string(contentBytes)
	} else {
		// Create new extensions.json with MCP server
		content = config.VSCodeExtensionTemplate
	}

	if err := os.WriteFile(extensionsFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create VS Code extensions file: %w", err)
	}

	fmt.Printf("✅ VS Code configured successfully\n")
	fmt.Printf("📁 Extensions file: %s\n", extensionsFile)
	fmt.Println()
	fmt.Println("Instructions for VS Code:")
	fmt.Println("1. Install an MCP extension for VS Code")
	fmt.Println("2. Restart VS Code")
	fmt.Println("3. The 'write' tool will be available via MCP extension")

	return nil
}

// configureClaude configures Claude Code integration
func configureClaude() error {
	paths := config.GetIDEPaths()
	configDir := paths.Claude
	if configDir == "" {
		return fmt.Errorf("could not determine Claude config directory")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create Claude config directory: %w", err)
	}

	// Create MCP config file
	mcpConfigFile := filepath.Join(configDir, config.DefaultMCPConfigFile)
	if err := os.WriteFile(mcpConfigFile, []byte(config.ClaudeConfigTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create Claude MCP config file: %w", err)
	}

	fmt.Printf("✅ Claude Code configured successfully\n")
	fmt.Printf("📁 Config file: %s\n", mcpConfigFile)
	fmt.Println()
	fmt.Println("Instructions for Claude Code:")
	fmt.Println("1. Restart Claude Code")
	fmt.Println("2. The 'write' tool will be available for all code operations")
	fmt.Println("3. Use it for file creation, editing, and generation")

	return nil
}
