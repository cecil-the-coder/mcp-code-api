package cmd

import (
	"fmt"

	"github.com/cecil-the-coder/mcp-code-api/internal/config/interactive"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Run the interactive configuration wizard",
	Long: `Run the interactive setup wizard to configure the MCP server
for your preferred IDE (Claude Code, Cursor, Cline, VS Code).

The wizard will:
- Guide you through API key setup
- Configure IDE integrations
- Set up automatic fallback providers
- Test API connections
- Generate necessary configuration files

Supported IDEs:
- Claude Code
- Cursor
- Cline
- VS Code (Copilot)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🧙 MCP Code API Configuration Wizard")
		fmt.Println("=====================================")
		fmt.Println()
		fmt.Println("This wizard will help you configure:")
		fmt.Println("  • API keys for multiple providers")
		fmt.Println("  • IDE integrations")
		fmt.Println("  • Automatic fallback settings")
		fmt.Println("  • Testing and validation")
		fmt.Println()

		// Run the interactive configuration
		if err := interactive.Run(); err != nil {
			return fmt.Errorf("configuration failed: %w", err)
		}

		fmt.Println("✅ Configuration completed successfully!")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Start the server: mcp-code-api server")
		fmt.Println("  2. Configure your IDE to use the MCP server")
		fmt.Println("  3. Start using the 'write' tool for code operations")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
