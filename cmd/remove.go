package cmd

import (
	"fmt"

	"github.com/cecil-the-coder/mcp-code-api/internal/config/interactive"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove MCP configurations and clean up files",
	Long: `Remove and clean up MCP configurations for any IDE
or perform a complete system cleanup.

This wizard will:
- Detect configured IDE integrations
- Remove configuration files
- Clean up MCP server settings
- Provide options for selective removal
- Verify cleanup completion`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🧹 MCP Code API Cleanup Wizard")
		fmt.Println("===============================")
		fmt.Println()
		fmt.Println("This wizard will help you:")
		fmt.Println("  • Remove IDE configurations")
		fmt.Println("  • Clean up MCP server settings")
		fmt.Println("  • Remove configuration files")
		fmt.Println("  • Verify cleanup completion")
		fmt.Println()

		// Run the interactive removal
		wizard := interactive.NewRemovalWizard()
		if err := wizard.Run(); err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}

		fmt.Println("✅ Cleanup completed successfully!")
		fmt.Println()
		fmt.Println("All MCP configurations have been removed.")
		fmt.Println("You can run 'mcp-code-api config' to set up again anytime.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
