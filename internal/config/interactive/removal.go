package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
)

// RemovalWizard handles interactive removal of configurations
type RemovalWizard struct {
	reader *bufio.Reader
}

// NewRemovalWizard creates a new removal wizard instance
func NewRemovalWizard() *RemovalWizard {
	return &RemovalWizard{
		reader: bufio.NewReader(os.Stdin),
	}
}

// promptYesNo prompts the user for a yes/no answer
func (w *RemovalWizard) promptYesNo(prompt string, defaultValue bool) bool {
	for {
		fmt.Print(prompt)
		input, err := w.reader.ReadString('\n')
		if err != nil {
			// Handle EOF gracefully
			return defaultValue
		}

		input = strings.ToLower(strings.TrimSpace(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no", "":
			return defaultValue
		default:
			fmt.Println("Please enter 'y' or 'n'.")
		}
	}
}

// Run executes the removal wizard
func (w *RemovalWizard) Run() error {
	fmt.Println("\n🧹 MCP Code API Configuration Cleanup")
	fmt.Println("======================================")

	// Check what configurations exist
	configs := w.findExistingConfigs()
	if len(configs) == 0 {
		fmt.Println("ℹ️  No MCP configurations found.")
		return nil
	}

	// Show found configurations
	fmt.Println("📁 Found configurations:")
	for _, config := range configs {
		fmt.Printf("  • %s\n", config)
	}
	fmt.Println()

	// Ask for confirmation
	confirmed := w.promptYesNo("Do you want to remove all MCP configurations? (y/N): ", false)
	if !confirmed {
		fmt.Println("❌ Cleanup cancelled.")
		return nil
	}

	// Remove configurations
	return w.removeConfigs(configs)
}

// findExistingConfigs finds all existing MCP configurations
func (w *RemovalWizard) findExistingConfigs() []string {
	var configs []string

	paths := config.GetIDEPaths()

	// Check various IDE configurations
	if paths.VSCode != "" {
		if config.FileExists(paths.VSCode) {
			configs = append(configs, "VS Code")
		}
	}

	if paths.Cursor != "" {
		if config.FileExists(paths.Cursor) {
			configs = append(configs, "Cursor")
		}
	}

	if paths.Cline != "" {
		if config.FileExists(paths.Cline) {
			configs = append(configs, "Cline")
		}
	}

	if paths.Claude != "" {
		if config.FileExists(paths.Claude) {
			configs = append(configs, "Claude")
		}
	}

	// Check for global config file
	home, err := os.UserHomeDir()
	if err == nil {
		configFile := fmt.Sprintf("%s/.cerebras-mcp.yaml", home)
		if config.FileExists(configFile) {
			configs = append(configs, "Global Config")
		}
	}

	return configs
}

// removeConfigs removes the specified configurations
func (w *RemovalWizard) removeConfigs(configs []string) error {
	fmt.Println("\n🗑️  Removing configurations...")

	paths := config.GetIDEPaths()
	home, _ := os.UserHomeDir()

	for _, config := range configs {
		switch config {
		case "VS Code":
			if paths.VSCode != "" {
				if err := os.Remove(paths.VSCode); err != nil {
					fmt.Printf("❌ Failed to remove VS Code config: %v\n", err)
				} else {
					fmt.Println("✅ Removed VS Code configuration")
				}
			}
		case "Cursor":
			if paths.Cursor != "" {
				if err := os.Remove(paths.Cursor); err != nil {
					fmt.Printf("❌ Failed to remove Cursor config: %v\n", err)
				} else {
					fmt.Println("✅ Removed Cursor configuration")
				}
			}
		case "Cline":
			if paths.Cline != "" {
				if err := os.RemoveAll(paths.Cline); err != nil {
					fmt.Printf("❌ Failed to remove Cline config: %v\n", err)
				} else {
					fmt.Println("✅ Removed Cline configuration")
				}
			}
		case "Claude":
			if paths.Claude != "" {
				if err := os.RemoveAll(paths.Claude); err != nil {
					fmt.Printf("❌ Failed to remove Claude config: %v\n", err)
				} else {
					fmt.Println("✅ Removed Claude configuration")
				}
			}
		case "Global Config":
			configFile := fmt.Sprintf("%s/.cerebras-mcp.yaml", home)
			if err := os.Remove(configFile); err != nil {
				fmt.Printf("❌ Failed to remove global config: %v\n", err)
			} else {
				fmt.Println("✅ Removed global configuration")
			}
		}
	}

	fmt.Println("\n🎉 Cleanup completed successfully!")
	fmt.Println("ℹ️  Note: You may need to restart your IDE for changes to take effect.")
	return nil
}
