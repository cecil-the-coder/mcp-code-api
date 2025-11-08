package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start MCP server",
	Long: `Start the Model Context Protocol (MCP) server that provides
a single 'write' tool for all code operations.

The server will:
- Listen for MCP requests via stdio
- Route requests to Cerebras or OpenRouter APIs
- Handle automatic fallback between providers
- Provide visual diffs for code changes
- Log all operations for debugging`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logging
		logFile := viper.GetString("log-file")
		if logFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			logFile = fmt.Sprintf("%s/mcp-code-api-debug.log", home)
		}

		if err := logger.SetLogFile(logFile); err != nil {
			return fmt.Errorf("failed to set log file: %w", err)
		}

		logger.Info("=== SERVER STARTUP ===")
		logger.Info("Cerebras Code MCP Server (Go) starting...")
		logger.Infof("Log file location: %s", logFile)

		// Load configuration
		cfg := config.Load()

		// Check API keys availability
		if cfg.Providers.Cerebras == nil || cfg.Providers.Cerebras.APIKey == "" {
			fmt.Fprintf(os.Stderr, "No Cerebras API key found\n")
			fmt.Fprintf(os.Stderr, "Get your Cerebras API key at: https://cloud.cerebras.ai\n")
		} else {
			fmt.Fprintf(os.Stderr, "Cerebras API key found\n")
			logger.Info("Cerebras API key configured")
		}

		if cfg.Providers.OpenRouter == nil || cfg.Providers.OpenRouter.APIKey == "" {
			fmt.Fprintf(os.Stderr, "No OpenRouter API key found\n")
			fmt.Fprintf(os.Stderr, "Get your OpenRouter API key at: https://openrouter.ai/keys\n")
		} else {
			fmt.Fprintf(os.Stderr, "OpenRouter API key found (will be used as fallback)\n")
			logger.Info("OpenRouter API key configured")
		}

		cerebrasAvail := cfg.Providers.Cerebras != nil && cfg.Providers.Cerebras.APIKey != ""
		openrouterAvail := cfg.Providers.OpenRouter != nil && cfg.Providers.OpenRouter.APIKey != ""
		if !cerebrasAvail && !openrouterAvail {
			fmt.Fprintf(os.Stderr, "No API keys available. Server will not function properly.\n")
			return fmt.Errorf("no API keys configured")
		}

		fmt.Fprintf(os.Stderr, "Starting MCP server...\n")

		// Create and start MCP server
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			logger.Info("Received shutdown signal")
			cancel()
		}()

		// Start the MCP server
		server := mcp.NewServer(cfg)
		if err := server.Start(ctx); err != nil {
			return fmt.Errorf("failed to start MCP server: %w", err)
		}

		fmt.Fprintf(os.Stderr, "🚀 MCP Server connected and ready with AUTO-INSTRUCTION SYSTEM!\n")
		fmt.Fprintf(os.Stderr, "🚨 CRITICAL: Enhanced system_instructions will automatically enforce MCP tool usage\n")
		fmt.Fprintf(os.Stderr, "🔧 write: MANDATORY tool for ALL code operations (file creation, generation, edits)\n")
		fmt.Fprintf(os.Stderr, "✨ Models will automatically use write tool - no user instruction needed!\n")
		if cfg.Providers.Cerebras != nil && cfg.Providers.Cerebras.APIKey != "" {
			fmt.Fprintf(os.Stderr, "Primary: Cerebras API\n")
		}
		if cfg.Providers.OpenRouter != nil && cfg.Providers.OpenRouter.APIKey != "" {
			fmt.Fprintf(os.Stderr, "Fallback: OpenRouter API\n")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Server-specific flags
	serverCmd.Flags().String("log-file", "", "path to log file")
	_ = viper.BindPFlag("log_file", serverCmd.Flags().Lookup("log-file"))

	// Add usage examples
	serverCmd.SetUsageTemplate(serverCmd.UsageTemplate() + `
Examples:
  # Start server with default settings
  mcp-code-api server

  # Start server with debug logging
  mcp-code-api server --debug

  # Start server with custom log file
  mcp-code-api server --log-file /tmp/mcp.log

  # Set API keys via environment variables
  CEREBRAS_API_KEY=your_key mcp-code-api server
  OPENROUTER_API_KEY=your_key mcp-code-api server
`)
}
