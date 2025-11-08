package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
)

// Wizard handles interactive configuration
type Wizard struct {
	reader *bufio.Reader
}

// NewWizard creates a new wizard instance
func NewWizard() *Wizard {
	return &Wizard{
		reader: bufio.NewReader(os.Stdin),
	}
}

// Run runs the interactive configuration wizard
func Run() error {
	wizard := NewWizard()
	return wizard.run()
}

// run executes the wizard flow
func (w *Wizard) run() error {
	fmt.Println("\nLet's configure your Cerebras MCP Server!")
	fmt.Println("======================================")

	// Step 1: Configure Cerebras API
	if err := w.configureCerebrasAPI(); err != nil {
		return err
	}

	// Step 2: Configure OpenRouter (optional)
	if err := w.configureOpenRouterAPI(); err != nil {
		return err
	}

	// Step 3: Choose IDEs
	if err := w.configureIDEs(); err != nil {
		return err
	}

	// Step 4: Test configuration
	if err := w.testConfiguration(); err != nil {
		return err
	}

	return nil
}

// configureCerebrasAPI configures the Cerebras API key and settings
func (w *Wizard) configureCerebrasAPI() error {
	fmt.Println("\n🔧 Cerebras API Configuration")
	fmt.Println("------------------------------")
	fmt.Println("Get your API key at: https://cloud.cerebras.ai")
	fmt.Println()

	apiKey := w.prompt("Enter your Cerebras API key (or press Enter to skip): ", true)
	if apiKey != "" {
		// Set environment variable
		os.Setenv("CEREBRAS_API_KEY", apiKey)
		fmt.Println("✅ Cerebras API key configured")

		// Ask about model
		model := w.prompt("Model (default: zai-glm-4.6): ", true)
		if model != "" {
			os.Setenv("CEREBRAS_MODEL", model)
		}

		// Ask about temperature
		temp := w.prompt("Temperature (0.0-1.0, default: 0.1): ", true)
		if temp != "" {
			if tempFloat, err := strconv.ParseFloat(temp, 64); err == nil && tempFloat >= 0.0 && tempFloat <= 1.0 {
				os.Setenv("CEREBRAS_TEMPERATURE", fmt.Sprintf("%.1f", tempFloat))
			} else {
				fmt.Println("⚠️  Invalid temperature, using default (0.1)")
			}
		}

		// Ask about max tokens
		maxTokens := w.prompt("Max tokens (optional, default: unlimited): ", true)
		if maxTokens != "" {
			if tokens, err := strconv.Atoi(maxTokens); err == nil && tokens > 0 {
				os.Setenv("CEREBRAS_MAX_TOKENS", strconv.Itoa(tokens))
			} else {
				fmt.Println("⚠️  Invalid max tokens, using default (unlimited)")
			}
		}
	} else {
		fmt.Println("⏭️  Skipping Cerebras API configuration")
	}

	return nil
}

// configureOpenRouterAPI configures the OpenRouter API key and settings
func (w *Wizard) configureOpenRouterAPI() error {
	fmt.Println("\n🔄 OpenRouter API Configuration (Optional)")
	fmt.Println("-----------------------------------------")
	fmt.Println("OpenRouter can serve as a fallback when Cerebras hits rate limits.")
	fmt.Println("Get your API key at: https://openrouter.ai/keys")
	fmt.Println()

	useOpenRouter := w.promptYesNo("Do you want to configure OpenRouter as a fallback? (y/N): ", false)
	if useOpenRouter {
		apiKey := w.prompt("Enter your OpenRouter API key: ", false)
		if apiKey != "" {
			// Set environment variable
			os.Setenv("OPENROUTER_API_KEY", apiKey)
			fmt.Println("✅ OpenRouter API key configured")

			// Ask about site URL
			siteURL := w.prompt("Site URL (default: https://github.com/cerebras/cerebras-code-mcp): ", true)
			if siteURL != "" {
				os.Setenv("OPENROUTER_SITE_URL", siteURL)
			}

			// Ask about site name
			siteName := w.prompt("Site name (default: Cerebras MCP): ", true)
			if siteName != "" {
				os.Setenv("OPENROUTER_SITE_NAME", siteName)
			}
		} else {
			fmt.Println("⚠️  OpenRouter API key is required")
			return w.configureOpenRouterAPI() // Retry
		}
	} else {
		fmt.Println("⏭️  Skipping OpenRouter API configuration")
	}

	return nil
}

// configureIDEs configures IDE integrations
func (w *Wizard) configureIDEs() error {
	fmt.Println("\n💻 IDE Integration Configuration")
	fmt.Println("---------------------------------")

	// List supported IDEs
	ides := config.GetAllIDEs()
	fmt.Println("Supported IDEs:")
	for i, ide := range ides {
		fmt.Printf("  %d. %s\n", i+1, ide.String())
	}
	fmt.Println()

	selected := w.prompt("Enter IDE numbers to configure (comma-separated, or 'all'): ", false)
	if strings.ToLower(selected) == "all" {
		for _, ide := range ides {
			if err := w.configureIDE(ide); err != nil {
				logger.Errorf("Failed to configure %s: %v", ide.String(), err)
			}
		}
	} else {
		numbers := strings.Split(selected, ",")
		for _, numStr := range numbers {
			if num, err := strconv.Atoi(strings.TrimSpace(numStr)); err == nil && num > 0 && num <= len(ides) {
				ide := ides[num-1]
				if err := w.configureIDE(ide); err != nil {
					logger.Errorf("Failed to configure %s: %v", ide.String(), err)
				}
			}
		}
	}

	return nil
}

// configureIDE configures a specific IDE
func (w *Wizard) configureIDE(ide config.IDE) error {
	fmt.Printf("\nConfiguring %s...\n", ide.String())

	switch ide {
	case config.IDECline:
		return configureCline()
	case config.IDECursor:
		return configureCursor()
	case config.IDEVSCode:
		return configureVSCode()
	case config.IDEClaude:
		return configureClaude()
	default:
		return fmt.Errorf("unsupported IDE: %s", ide.String())
	}
}

// testConfiguration tests the API connections
func (w *Wizard) testConfiguration() error {
	fmt.Println("\n🧪 Testing Configuration")
	fmt.Println("-------------------------")

	// Test configuration loading
	cfg := config.Load()
	if !cfg.HasAnyAPIKey() {
		return fmt.Errorf("no API keys configured")
	}

	// Show summary
	fmt.Println("\n📋 Configuration Summary:")
	if cfg.Providers.Cerebras != nil && cfg.Providers.Cerebras.APIKey != "" {
		fmt.Printf("✅ Cerebras API: %s\n", cfg.Providers.Cerebras.Model)
	}
	if cfg.Providers.OpenRouter != nil && cfg.Providers.OpenRouter.APIKey != "" {
		fmt.Printf("✅ OpenRouter API: %s\n", cfg.Providers.OpenRouter.Model)
	}

	primary := cfg.GetPrimaryProvider()
	fallback := cfg.GetFallbackProvider(primary)
	fmt.Printf("Primary: %s\n", primary)
	if fallback != "" {
		fmt.Printf("Fallback: %s\n", fallback)
	}

	return nil
}

// prompt prompts the user for input
func (w *Wizard) prompt(prompt string, allowEmpty bool) string {
	for {
		fmt.Print(prompt)
		input, err := w.reader.ReadString('\n')
		if err != nil {
			// Debug: print the error
			fmt.Fprintf(os.Stderr, "DEBUG: ReadString error: %v\n", err)
			// Handle EOF or other input errors gracefully - just return
			return ""
		}

		input = strings.TrimSpace(input)
		if input != "" || allowEmpty {
			return input
		}

		fmt.Println("This field is required. Please enter a value.")
	}
}

// promptYesNo prompts the user for a yes/no answer
func (w *Wizard) promptYesNo(prompt string, defaultValue bool) bool {
	for {
		fmt.Print(prompt)
		input, err := w.reader.ReadString('\n')
		if err != nil {
			// Handle EOF or other input errors gracefully - just return default
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
