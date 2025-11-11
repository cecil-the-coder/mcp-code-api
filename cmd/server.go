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
	"github.com/cecil-the-coder/mcp-code-api/internal/metrics"
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
		logger.Info("MCP Code API server starting...")
		logger.Infof("Log file location: %s", logFile)

		// Load configuration
		cfg := config.Load()

		// Apply logging configuration from config file
		logger.SetDebug(cfg.Logging.Debug)
		logger.SetVerbose(cfg.Logging.Verbose)
		logger.Debugf("Debug logging enabled: %v", cfg.Logging.Debug)
		logger.Debugf("Verbose logging enabled: %v", cfg.Logging.Verbose)

		// Log config details now that debug/verbose are enabled
		logger.Debugf("Preferred provider order: %v", cfg.Providers.Order)
		logger.Debugf("Enabled providers: %v", cfg.Providers.Enabled)

		// Check API keys availability (log to file only, not stderr)
		if cfg.Providers.Cerebras == nil || cfg.Providers.Cerebras.APIKey == "" {
			logger.Info("No Cerebras API key found")
		} else {
			logger.Info("Cerebras API key configured")
		}

		if cfg.Providers.OpenRouter == nil || cfg.Providers.OpenRouter.APIKey == "" {
			logger.Info("No OpenRouter API key found")
		} else {
			logger.Info("OpenRouter API key configured")
		}

		cerebrasAvail := cfg.Providers.Cerebras != nil && cfg.Providers.Cerebras.APIKey != ""
		openrouterAvail := cfg.Providers.OpenRouter != nil && cfg.Providers.OpenRouter.APIKey != ""
		geminiAvail := cfg.Providers.Gemini != nil && (cfg.Providers.Gemini.APIKey != "" || cfg.Providers.Gemini.AccessToken != "")
		if !cerebrasAvail && !openrouterAvail && !geminiAvail {
			logger.Error("No API keys available")
			return fmt.Errorf("no API keys configured")
		}

		logger.Info("Starting MCP server...")

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
		logger.Info("MCP Server starting...")

		// Create shared metrics store
		metricsStore, err := metrics.NewSharedMetricsStore()
		if err != nil {
			logger.Warnf("Failed to create shared metrics store: %v", err)
		} else {
			// Start periodic metrics updates
			metricsStore.Start(server.GetRouter())
			defer metricsStore.Stop()
		}

		// Start metrics server if enabled
		var metricsServer *metrics.MetricsServer
		if cfg.Metrics.Enabled && metricsStore != nil {
			port := cfg.Metrics.Port
			if viper.IsSet("metrics_port") && viper.GetInt("metrics_port") != 0 {
				port = viper.GetInt("metrics_port")
			}

			metricsServer = metrics.NewMetricsServer(metricsStore, cfg.Metrics.Host, port)
			if err := metricsServer.Start(); err != nil {
				logger.Warnf("Failed to start metrics server: %v", err)
			} else {
				logger.Infof("Metrics server started on http://%s:%d", cfg.Metrics.Host, port)
				defer func() {
					logger.Info("Shutting down metrics server...")
					if err := metricsServer.Stop(); err != nil {
						logger.Warnf("Error stopping metrics server: %v", err)
					}
				}()
			}
		}

		if err := server.Start(ctx); err != nil {
			return fmt.Errorf("failed to start MCP server: %w", err)
		}

		logger.Info("MCP Server shut down gracefully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Server-specific flags
	serverCmd.Flags().String("log-file", "", "path to log file")
	_ = viper.BindPFlag("log_file", serverCmd.Flags().Lookup("log-file"))

	serverCmd.Flags().Int("metrics-port", 0, "port for metrics HTTP server (0 = use config default)")
	_ = viper.BindPFlag("metrics_port", serverCmd.Flags().Lookup("metrics-port"))

	// Add usage examples
	serverCmd.SetUsageTemplate(serverCmd.UsageTemplate() + `
Examples:
  # Start server with default settings
  mcp-code-api server

  # Start server with debug logging
  mcp-code-api server --debug

  # Start server with custom log file
  mcp-code-api server --log-file /tmp/mcp.log

  # Start server with custom metrics port
  mcp-code-api server --metrics-port 9090

  # Set API keys via environment variables
  CEREBRAS_API_KEY=your_key mcp-code-api server
  OPENROUTER_API_KEY=your_key mcp-code-api server
`)
}