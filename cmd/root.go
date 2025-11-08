package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mcp-code-api",
	Short: "MCP Code API - Multi-Provider Code Generation Server",
	Long: `MCP Code API v` + version + ` - Multi-Provider Code Generation Server

A high-performance Model Context Protocol (MCP) server supporting multiple AI
providers (Cerebras, OpenRouter, OpenAI, Anthropic, Gemini, and more). Designed
for planning with Claude Code, Cline, or Cursor while leveraging fast providers
like Cerebras for code generation to maximize speed and avoid API limits.

Features:
- Multi-provider support with smart routing and fallback
- Enhanced visual diffs for Claude Code
- IDE support for Claude Code, Cursor, Cline, VS Code
- Interactive configuration wizard
- OAuth authentication (in development)
- Load balancing across multiple API keys

Built with Go for maximum performance and reliability.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default searches: ./config.yaml, ~/.mcp-code-api/config.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("debug", false, "debug mode with detailed logging")

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}

		// Try to find config.yaml in common locations
		configLocations := []string{
			"./config.yaml",                      // Current directory
			home + "/.mcp-code-api/config.yaml",  // User config directory
		}

		configFound := false
		for _, configPath := range configLocations {
			if _, err := os.Stat(configPath); err == nil {
				viper.SetConfigFile(configPath)
				configFound = true
				break
			}
		}

		// If no config file found, set default search paths
		if !configFound {
			viper.AddConfigPath(".")
			viper.AddConfigPath(home + "/.mcp-code-api")
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName("config")
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}
