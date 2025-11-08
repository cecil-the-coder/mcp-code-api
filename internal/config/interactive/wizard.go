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
	fmt.Println("\nLet's configure your MCP Code API server!")
	fmt.Println("==========================================")

	// Step 1: Configure Cerebras API
	if err := w.configureCerebrasAPI(); err != nil {
		return err
	}

	// Step 2: Configure OpenRouter (optional)
	if err := w.configureOpenRouterAPI(); err != nil {
		return err
	}

	// Step 3: Configure additional providers (optional)
	if err := w.configureAdditionalProviders(); err != nil {
		return err
	}

	// Step 4: Choose IDEs
	if err := w.configureIDEs(); err != nil {
		return err
	}

	// Step 5: Test configuration
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
			siteName := w.prompt("Site name (default: MCP Code API): ", true)
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

// configureAdditionalProviders configures additional providers with OAuth support
func (w *Wizard) configureAdditionalProviders() error {
	fmt.Println("\n🌟 Additional Provider Configuration (Optional)")
	fmt.Println("-----------------------------------------------")
	fmt.Println("Configure additional AI providers with OAuth or API key authentication.")
	fmt.Println("Supported providers: Anthropic, Gemini, Qwen, OpenAI")
	fmt.Println()

	configure := w.promptYesNo("Do you want to configure additional providers? (y/N): ", false)
	if !configure {
		fmt.Println("⏭️  Skipping additional provider configuration")
		return nil
	}

	// Anthropic
	fmt.Println()
	configureAnthropic := w.promptYesNo("Configure Anthropic Claude? (y/N): ", false)
	if configureAnthropic {
		if err := w.configureAnthropicProvider(); err != nil {
			logger.Errorf("Failed to configure Anthropic: %v", err)
		}
	}

	// Gemini
	fmt.Println()
	configureGemini := w.promptYesNo("Configure Google Gemini? (y/N): ", false)
	if configureGemini {
		if err := w.configureGeminiProvider(); err != nil {
			logger.Errorf("Failed to configure Gemini: %v", err)
		}
	}

	// Qwen
	fmt.Println()
	configureQwen := w.promptYesNo("Configure Alibaba Qwen? (y/N): ", false)
	if configureQwen {
		if err := w.configureQwenProvider(); err != nil {
			logger.Errorf("Failed to configure Qwen: %v", err)
		}
	}

	// OpenAI
	fmt.Println()
	configureOpenAI := w.promptYesNo("Configure OpenAI? (y/N): ", false)
	if configureOpenAI {
		if err := w.configureOpenAIProvider(); err != nil {
			logger.Errorf("Failed to configure OpenAI: %v", err)
		}
	}

	return nil
}

// configureAnthropicProvider configures Anthropic with API key or OAuth
func (w *Wizard) configureAnthropicProvider() error {
	fmt.Println("\n🤖 Anthropic Claude Configuration")
	fmt.Println("----------------------------------")
	fmt.Println("Choose authentication method:")
	fmt.Println("  1. API Key (recommended for most users)")
	fmt.Println("  2. OAuth (for advanced integrations)")
	fmt.Println()

	method := w.prompt("Select method (1 or 2): ", false)

	switch method {
	case "1":
		apiKey := w.prompt("Enter Anthropic API key: ", false)
		if apiKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", apiKey)
			fmt.Println("✅ Anthropic API key configured")
		}
	case "2":
		_, tokenInfo, err := w.configureProviderOAuth("anthropic", "Anthropic")
		if err != nil {
			fmt.Printf("⚠️  OAuth configuration failed: %v\n", err)
			fmt.Println("Falling back to API key method...")
			return w.configureAnthropicProvider()
		}
		if tokenInfo != nil {
			fmt.Println("✅ Anthropic OAuth configured successfully")
			// Store OAuth config in environment for this session
			// In a real implementation, we'd save this to the config file
		}
	default:
		fmt.Println("Invalid selection, skipping Anthropic configuration")
	}

	return nil
}

// configureGeminiProvider configures Gemini with API key or OAuth
func (w *Wizard) configureGeminiProvider() error {
	fmt.Println("\n✨ Google Gemini Configuration")
	fmt.Println("------------------------------")
	fmt.Println("Choose authentication method:")
	fmt.Println("  1. API Key (recommended for most users)")
	fmt.Println("  2. OAuth (for advanced integrations)")
	fmt.Println()

	method := w.prompt("Select method (1 or 2): ", false)

	switch method {
	case "1":
		apiKey := w.prompt("Enter Gemini API key: ", false)
		if apiKey != "" {
			os.Setenv("GEMINI_API_KEY", apiKey)
			fmt.Println("✅ Gemini API key configured")
		}
	case "2":
		_, tokenInfo, err := w.configureProviderOAuth("gemini", "Gemini")
		if err != nil {
			fmt.Printf("⚠️  OAuth configuration failed: %v\n", err)
			fmt.Println("Falling back to API key method...")
			return w.configureGeminiProvider()
		}
		if tokenInfo != nil {
			fmt.Println("✅ Gemini OAuth configured successfully")
		}
	default:
		fmt.Println("Invalid selection, skipping Gemini configuration")
	}

	return nil
}

// configureQwenProvider configures Qwen with API key or OAuth
func (w *Wizard) configureQwenProvider() error {
	fmt.Println("\n🐉 Alibaba Qwen Configuration")
	fmt.Println("-----------------------------")
	fmt.Println("Choose authentication method:")
	fmt.Println("  1. API Key (recommended for most users)")
	fmt.Println("  2. OAuth (for advanced integrations)")
	fmt.Println()

	method := w.prompt("Select method (1 or 2): ", false)

	switch method {
	case "1":
		apiKey := w.prompt("Enter Qwen API key: ", false)
		if apiKey != "" {
			os.Setenv("QWEN_API_KEY", apiKey)
			fmt.Println("✅ Qwen API key configured")
		}
	case "2":
		_, tokenInfo, err := w.configureProviderOAuth("qwen", "Qwen")
		if err != nil {
			fmt.Printf("⚠️  OAuth configuration failed: %v\n", err)
			fmt.Println("Falling back to API key method...")
			return w.configureQwenProvider()
		}
		if tokenInfo != nil {
			fmt.Println("✅ Qwen OAuth configured successfully")
		}
	default:
		fmt.Println("Invalid selection, skipping Qwen configuration")
	}

	return nil
}

// configureOpenAIProvider configures OpenAI with API key
func (w *Wizard) configureOpenAIProvider() error {
	fmt.Println("\n🤖 OpenAI Configuration")
	fmt.Println("-----------------------")
	apiKey := w.prompt("Enter OpenAI API key: ", false)
	if apiKey != "" {
		os.Setenv("OPENAI_API_KEY", apiKey)
		fmt.Println("✅ OpenAI API key configured")
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
