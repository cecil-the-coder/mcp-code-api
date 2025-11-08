package interactive

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"gopkg.in/yaml.v3"
)

// Wizard handles interactive configuration
type Wizard struct {
	reader *bufio.Reader
	config *collectedConfig
}

// collectedConfig stores configuration values collected during the wizard
type collectedConfig struct {
	// Cerebras
	cerebrasAPIKey      string
	cerebrasModels      []string
	cerebrasTemperature string
	cerebrasMaxTokens   string

	// OpenRouter
	openrouterAPIKey   string
	openrouterModels   []string
	openrouterSiteURL  string
	openrouterSiteName string

	// OpenAI
	openaiAPIKey string
	openaiModels []string

	// Anthropic
	anthropicAPIKey string
	anthropicModels []string
	anthropicOAuth  *oauthTokenData

	// Gemini
	geminiAPIKey string
	geminiModels []string
	geminiOAuth  *oauthTokenData

	// Qwen
	qwenAPIKey string
	qwenModels []string
	qwenOAuth  *oauthTokenData
}

// oauthTokenData stores OAuth token information
type oauthTokenData struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    string
	TokenType    string
}

// NewWizard creates a new wizard instance
func NewWizard() *Wizard {
	return &Wizard{
		reader: bufio.NewReader(os.Stdin),
		config: &collectedConfig{},
	}
}

// Run runs the interactive configuration wizard
func Run() error {
	wizard := NewWizard()
	return wizard.run()
}

// run executes the wizard flow
func (w *Wizard) run() error {
	fmt.Println("\n╔════════════════════════════════════════╗")
	fmt.Println("║  MCP Code API Configuration Wizard    ║")
	fmt.Println("╚════════════════════════════════════════╝")

	// Step 1: Select providers to configure
	selectedProviders, err := w.selectProviders()
	if err != nil {
		return err
	}

	if len(selectedProviders) == 0 {
		fmt.Println("\n⚠️  No providers selected. At least one provider is required.")
		return fmt.Errorf("no providers configured")
	}

	// Step 2: Configure selected providers
	for _, provider := range selectedProviders {
		if err := w.configureProvider(provider); err != nil {
			logger.Errorf("Failed to configure %s: %v", provider, err)
		}
	}

	// Step 3: Test configuration
	if err := w.testConfiguration(); err != nil {
		return err
	}

	// Step 4: Save configuration to file
	configPath, err := w.saveConfiguration()
	if err != nil {
		logger.Errorf("Failed to save configuration: %v", err)
		fmt.Println("\n⚠️  Warning: Configuration was not saved to file.")
		fmt.Println("   You can manually set environment variables or create a config.yaml file.")
	} else {
		fmt.Printf("\n✅ Configuration saved to: %s\n", configPath)
	}

	fmt.Println("\n✅ Configuration complete!")
	fmt.Println("\n📝 Next steps:")
	if configPath != "" {
		fmt.Printf("   1. Start the MCP server: mcp-code-api server --config %s\n", configPath)
	} else {
		fmt.Println("   1. Start the MCP server: mcp-code-api server")
	}
	fmt.Println("   2. The server will automatically provide systemPrompt instructions to all MCP-compatible IDEs")
	fmt.Println("   3. Use the 'write' tool in your IDE for all code operations")

	return nil
}

// selectProviders presents a menu of providers and returns the user's selection
func (w *Wizard) selectProviders() ([]string, error) {
	fmt.Println("\n📋 Available AI Providers:")
	fmt.Println("   1. Cerebras - Fast inference with ZhipuAI GLM and other models")
	fmt.Println("   2. OpenRouter - Access to multiple models with fallback support")
	fmt.Println("   3. Anthropic Claude - Advanced reasoning with API key or OAuth")
	fmt.Println("   4. Google Gemini - Multimodal AI with API key or OAuth")
	fmt.Println("   5. Alibaba Qwen - Chinese language models with API key or OAuth")
	fmt.Println("   6. OpenAI - GPT models with API key")
	fmt.Println()
	fmt.Println("Select providers to configure:")
	fmt.Println("  • Enter numbers separated by commas (e.g., 1,3,4)")
	fmt.Println("  • Enter 'all' to configure all providers")
	fmt.Println("  • Press Enter to skip and configure later")
	fmt.Println()

	input := w.prompt("Your selection: ", true)
	if input == "" {
		return []string{}, nil
	}

	// Handle 'all' selection
	if strings.ToLower(strings.TrimSpace(input)) == "all" {
		return []string{"cerebras", "openrouter", "anthropic", "gemini", "qwen", "openai"}, nil
	}

	// Parse comma-separated numbers
	providerMap := map[int]string{
		1: "cerebras",
		2: "openrouter",
		3: "anthropic",
		4: "gemini",
		5: "qwen",
		6: "openai",
	}

	var selected []string
	seen := make(map[string]bool)

	numbers := strings.Split(input, ",")
	for _, numStr := range numbers {
		numStr = strings.TrimSpace(numStr)
		num, err := strconv.Atoi(numStr)
		if err != nil || num < 1 || num > 6 {
			fmt.Printf("⚠️  Invalid selection: %s (skipping)\n", numStr)
			continue
		}

		provider := providerMap[num]
		if !seen[provider] {
			selected = append(selected, provider)
			seen[provider] = true
		}
	}

	return selected, nil
}

// configureProvider routes to the appropriate provider configuration function
func (w *Wizard) configureProvider(provider string) error {
	switch provider {
	case "cerebras":
		return w.configureCerebrasAPI()
	case "openrouter":
		return w.configureOpenRouterAPI()
	case "anthropic":
		return w.configureAnthropicProvider()
	case "gemini":
		return w.configureGeminiProvider()
	case "qwen":
		return w.configureQwenProvider()
	case "openai":
		return w.configureOpenAIProvider()
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
}

// configureCerebrasAPI configures the Cerebras API key and settings
func (w *Wizard) configureCerebrasAPI() error {
	fmt.Println("\n🔧 Cerebras API Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Get your API key at: https://cloud.cerebras.ai")
	fmt.Println()

	apiKey := w.prompt("Enter your Cerebras API key: ", false)
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Store and set environment variable
	w.config.cerebrasAPIKey = apiKey
	os.Setenv("CEREBRAS_API_KEY", apiKey)
	fmt.Println("✅ Cerebras API key configured")

	// Ask about models (support multiple)
	fmt.Println()
	fmt.Println("Model Configuration:")
	fmt.Println("  • Enter one or more models separated by commas")
	fmt.Println("  • Example: zai-glm-4.6,llama3.1-70b")
	modelsInput := w.prompt("Models (default: zai-glm-4.6, press Enter for default): ", true)
	if modelsInput != "" {
		w.config.cerebrasModels = parseModelList(modelsInput)
		os.Setenv("CEREBRAS_MODEL", w.config.cerebrasModels[0]) // Set first model as env var
	} else {
		w.config.cerebrasModels = []string{"zai-glm-4.6"}
	}

	// Ask about temperature
	temp := w.prompt("Temperature (0.0-1.0, default: 0.6, press Enter for default): ", true)
	if temp != "" {
		if tempFloat, err := strconv.ParseFloat(temp, 64); err == nil && tempFloat >= 0.0 && tempFloat <= 1.0 {
			w.config.cerebrasTemperature = temp
			os.Setenv("CEREBRAS_TEMPERATURE", fmt.Sprintf("%.1f", tempFloat))
		} else {
			fmt.Println("⚠️  Invalid temperature, using default (0.6)")
		}
	}

	// Ask about max tokens
	maxTokens := w.prompt("Max tokens (default: unlimited, press Enter for default): ", true)
	if maxTokens != "" {
		if tokens, err := strconv.Atoi(maxTokens); err == nil && tokens > 0 {
			w.config.cerebrasMaxTokens = maxTokens
			os.Setenv("CEREBRAS_MAX_TOKENS", strconv.Itoa(tokens))
		} else {
			fmt.Println("⚠️  Invalid max tokens, using default (unlimited)")
		}
	}

	return nil
}

// configureOpenRouterAPI configures the OpenRouter API key and settings
func (w *Wizard) configureOpenRouterAPI() error {
	fmt.Println("\n🔄 OpenRouter API Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("OpenRouter provides access to multiple models and can serve as a fallback.")
	fmt.Println("Get your API key at: https://openrouter.ai/keys")
	fmt.Println()

	apiKey := w.prompt("Enter your OpenRouter API key: ", false)
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Store and set environment variable
	w.config.openrouterAPIKey = apiKey
	os.Setenv("OPENROUTER_API_KEY", apiKey)
	fmt.Println("✅ OpenRouter API key configured")

	// Ask about models
	fmt.Println()
	fmt.Println("Model Configuration:")
	fmt.Println("  • Enter one or more models separated by commas")
	fmt.Println("  • Example: qwen/qwen3-coder,anthropic/claude-3.5-sonnet")
	fmt.Println("  • See models at: https://openrouter.ai/models")
	modelsInput := w.prompt("Models (default: qwen/qwen3-coder, press Enter for default): ", true)
	if modelsInput != "" {
		w.config.openrouterModels = parseModelList(modelsInput)
	} else {
		w.config.openrouterModels = []string{"qwen/qwen3-coder"}
	}

	// Ask about site URL
	siteURL := w.prompt("Site URL (default: https://github.com/cecil-the-coder/mcp-code-api, press Enter for default): ", true)
	if siteURL == "" {
		siteURL = "https://github.com/cecil-the-coder/mcp-code-api"
	}
	w.config.openrouterSiteURL = siteURL
	os.Setenv("OPENROUTER_SITE_URL", siteURL)

	// Ask about site name
	siteName := w.prompt("Site name (default: MCP Code API, press Enter for default): ", true)
	if siteName != "" {
		w.config.openrouterSiteName = siteName
		os.Setenv("OPENROUTER_SITE_NAME", siteName)
	}

	return nil
}

// configureAnthropicProvider configures Anthropic with API key or OAuth
func (w *Wizard) configureAnthropicProvider() error {
	fmt.Println("\n🤖 Anthropic Claude Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Choose authentication method:")
	fmt.Println("  1. API Key (recommended for most users)")
	fmt.Println("  2. OAuth (automatic browser-based login)")
	fmt.Println()

	method := w.prompt("Select method (1 or 2): ", false)

	switch method {
	case "1":
		fmt.Println("\nGet your API key at: https://console.anthropic.com/settings/keys")
		apiKey := w.prompt("Enter Anthropic API key: ", false)
		if apiKey == "" {
			return fmt.Errorf("API key is required")
		}
		w.config.anthropicAPIKey = apiKey
		os.Setenv("ANTHROPIC_API_KEY", apiKey)
		fmt.Println("✅ Anthropic API key configured")
	case "2":
		_, tokenInfo, err := w.configureProviderOAuth("anthropic", "Anthropic")
		if err != nil {
			return fmt.Errorf("OAuth configuration failed: %w", err)
		}
		if tokenInfo != nil {
			w.config.anthropicOAuth = &oauthTokenData{
				AccessToken:  tokenInfo.AccessToken,
				RefreshToken: tokenInfo.RefreshToken,
				ExpiresAt:    tokenInfo.ExpiresAt.Format(time.RFC3339),
				TokenType:    tokenInfo.TokenType,
			}
			fmt.Println("✅ Anthropic OAuth configured successfully")
		}
	default:
		return fmt.Errorf("invalid selection: %s", method)
	}

	// Ask about models
	fmt.Println()
	fmt.Println("Model Configuration:")
	fmt.Println("  • Enter one or more models separated by commas")
	fmt.Println("  • Example: claude-3-5-sonnet-20241022,claude-3-opus-20240229")
	modelsInput := w.prompt("Models (default: claude-3-5-sonnet-20241022, press Enter for default): ", true)
	if modelsInput != "" {
		w.config.anthropicModels = parseModelList(modelsInput)
	} else {
		w.config.anthropicModels = []string{"claude-3-5-sonnet-20241022"}
	}

	return nil
}

// configureGeminiProvider configures Gemini with API key or OAuth
func (w *Wizard) configureGeminiProvider() error {
	fmt.Println("\n✨ Google Gemini Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Choose authentication method:")
	fmt.Println("  1. API Key (recommended for most users)")
	fmt.Println("  2. OAuth (automatic browser-based Google login)")
	fmt.Println()

	method := w.prompt("Select method (1 or 2): ", false)

	switch method {
	case "1":
		fmt.Println("\nGet your API key at: https://makersuite.google.com/app/apikey")
		apiKey := w.prompt("Enter Gemini API key: ", false)
		if apiKey == "" {
			return fmt.Errorf("API key is required")
		}
		w.config.geminiAPIKey = apiKey
		os.Setenv("GEMINI_API_KEY", apiKey)
		fmt.Println("✅ Gemini API key configured")
	case "2":
		_, tokenInfo, err := w.configureProviderOAuth("gemini", "Gemini")
		if err != nil {
			return fmt.Errorf("OAuth configuration failed: %w", err)
		}
		if tokenInfo != nil {
			w.config.geminiOAuth = &oauthTokenData{
				AccessToken:  tokenInfo.AccessToken,
				RefreshToken: tokenInfo.RefreshToken,
				ExpiresAt:    tokenInfo.ExpiresAt.Format(time.RFC3339),
				TokenType:    tokenInfo.TokenType,
			}
			fmt.Println("✅ Gemini OAuth configured successfully")
		}
	default:
		return fmt.Errorf("invalid selection: %s", method)
	}

	// Ask about models
	fmt.Println()
	fmt.Println("Model Configuration:")
	fmt.Println("  • Enter one or more models separated by commas")
	fmt.Println("  • Example: gemini-1.5-pro,gemini-1.5-flash")
	modelsInput := w.prompt("Models (default: gemini-1.5-pro, press Enter for default): ", true)
	if modelsInput != "" {
		w.config.geminiModels = parseModelList(modelsInput)
	} else {
		w.config.geminiModels = []string{"gemini-1.5-pro"}
	}

	return nil
}

// configureQwenProvider configures Qwen with API key or OAuth
func (w *Wizard) configureQwenProvider() error {
	fmt.Println("\n🐉 Alibaba Qwen Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Choose authentication method:")
	fmt.Println("  1. API Key (recommended for most users)")
	fmt.Println("  2. OAuth (automatic browser-based login)")
	fmt.Println()

	method := w.prompt("Select method (1 or 2): ", false)

	switch method {
	case "1":
		fmt.Println("\nGet your API key at: https://dashscope.console.aliyun.com/")
		apiKey := w.prompt("Enter Qwen API key: ", false)
		if apiKey == "" {
			return fmt.Errorf("API key is required")
		}
		w.config.qwenAPIKey = apiKey
		os.Setenv("QWEN_API_KEY", apiKey)
		fmt.Println("✅ Qwen API key configured")
	case "2":
		_, tokenInfo, err := w.configureProviderOAuth("qwen", "Qwen")
		if err != nil {
			return fmt.Errorf("OAuth configuration failed: %w", err)
		}
		if tokenInfo != nil {
			w.config.qwenOAuth = &oauthTokenData{
				AccessToken:  tokenInfo.AccessToken,
				RefreshToken: tokenInfo.RefreshToken,
				ExpiresAt:    tokenInfo.ExpiresAt.Format(time.RFC3339),
				TokenType:    tokenInfo.TokenType,
			}
			fmt.Println("✅ Qwen OAuth configured successfully")
		}
	default:
		return fmt.Errorf("invalid selection: %s", method)
	}

	// Ask about models
	fmt.Println()
	fmt.Println("Model Configuration:")
	fmt.Println("  • Enter one or more models separated by commas")
	fmt.Println("  • Example: qwen-max,qwen-turbo")
	modelsInput := w.prompt("Models (default: qwen-max, press Enter for default): ", true)
	if modelsInput != "" {
		w.config.qwenModels = parseModelList(modelsInput)
	} else {
		w.config.qwenModels = []string{"qwen-max"}
	}

	return nil
}

// configureOpenAIProvider configures OpenAI with API key
func (w *Wizard) configureOpenAIProvider() error {
	fmt.Println("\n🤖 OpenAI Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Get your API key at: https://platform.openai.com/api-keys")
	fmt.Println()

	apiKey := w.prompt("Enter OpenAI API key: ", false)
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	w.config.openaiAPIKey = apiKey
	os.Setenv("OPENAI_API_KEY", apiKey)
	fmt.Println("✅ OpenAI API key configured")

	// Ask about models
	fmt.Println()
	fmt.Println("Model Configuration:")
	fmt.Println("  • Enter one or more models separated by commas")
	fmt.Println("  • Example: gpt-4o,gpt-4-turbo,gpt-3.5-turbo")
	modelsInput := w.prompt("Models (default: gpt-4o, press Enter for default): ", true)
	if modelsInput != "" {
		w.config.openaiModels = parseModelList(modelsInput)
	} else {
		w.config.openaiModels = []string{"gpt-4o"}
	}

	return nil
}

// testConfiguration tests the API connections
func (w *Wizard) testConfiguration() error {
	fmt.Println("\n🧪 Testing Configuration")
	fmt.Println("-------------------------")

	// Check if any provider is configured (API key or OAuth)
	hasAnyProvider := w.config.cerebrasAPIKey != "" ||
		w.config.openrouterAPIKey != "" ||
		w.config.openaiAPIKey != "" ||
		w.config.anthropicAPIKey != "" ||
		w.config.anthropicOAuth != nil ||
		w.config.geminiAPIKey != "" ||
		w.config.geminiOAuth != nil ||
		w.config.qwenAPIKey != "" ||
		w.config.qwenOAuth != nil

	if !hasAnyProvider {
		return fmt.Errorf("no providers configured")
	}

	// Show summary
	fmt.Println("\n📋 Configuration Summary:")
	if w.config.cerebrasAPIKey != "" {
		fmt.Println("✅ Cerebras API configured")
	}
	if w.config.openrouterAPIKey != "" {
		fmt.Println("✅ OpenRouter API configured")
	}
	if w.config.openaiAPIKey != "" {
		fmt.Println("✅ OpenAI API configured")
	}
	if w.config.anthropicAPIKey != "" {
		fmt.Println("✅ Anthropic API Key configured")
	}
	if w.config.anthropicOAuth != nil {
		fmt.Printf("✅ Anthropic OAuth configured (expires: %s)\n", w.config.anthropicOAuth.ExpiresAt)
	}
	if w.config.geminiAPIKey != "" {
		fmt.Println("✅ Gemini API Key configured")
	}
	if w.config.geminiOAuth != nil {
		fmt.Printf("✅ Gemini OAuth configured (expires: %s)\n", w.config.geminiOAuth.ExpiresAt)
	}
	if w.config.qwenAPIKey != "" {
		fmt.Println("✅ Qwen API Key configured")
	}
	if w.config.qwenOAuth != nil {
		fmt.Printf("✅ Qwen OAuth configured (expires: %s)\n", w.config.qwenOAuth.ExpiresAt)
	}

	return nil
}

// parseModelList parses a comma-separated list of models
func parseModelList(input string) []string {
	if input == "" {
		return nil
	}

	var models []string
	parts := strings.Split(input, ",")
	for _, part := range parts {
		model := strings.TrimSpace(part)
		if model != "" {
			models = append(models, model)
		}
	}
	return models
}

// writeModelsYAML writes models array to YAML format
func writeModelsYAML(sb *strings.Builder, models []string, indent string) {
	if len(models) == 0 {
		return
	}

	if len(models) == 1 {
		// Single model: use simple format
		sb.WriteString(fmt.Sprintf("%smodel: \"%s\"\n", indent, models[0]))
	} else {
		// Multiple models: use array format
		sb.WriteString(fmt.Sprintf("%smodels:\n", indent))
		for _, model := range models {
			sb.WriteString(fmt.Sprintf("%s  - \"%s\"\n", indent, model))
		}
	}
}

// modelsToInterface converts model slice to interface{} for YAML marshal
func modelsToInterface(models []string) interface{} {
	if len(models) == 0 {
		return nil
	}
	if len(models) == 1 {
		return models[0]
	}
	// Convert to []interface{} for YAML marshaler
	result := make([]interface{}, len(models))
	for i, m := range models {
		result[i] = m
	}
	return result
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

// saveConfiguration prompts for config file location and saves the configuration
func (w *Wizard) saveConfiguration() (string, error) {
	fmt.Println("\n💾 Save Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Choose where to save your configuration:")
	fmt.Println("  1. config.yaml (current directory)")
	fmt.Println("  2. ~/.mcp-code-api/config.yaml (user config directory)")
	fmt.Println("  3. Custom path")
	fmt.Println("  4. Skip (don't save)")
	fmt.Println()

	choice := w.prompt("Select option (1-4): ", false)

	var configPath string
	switch choice {
	case "1":
		configPath = "config.yaml"
	case "2":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(homeDir, ".mcp-code-api")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "config.yaml")
	case "3":
		configPath = w.prompt("Enter full path to config file: ", false)
		if configPath == "" {
			return "", fmt.Errorf("path is required")
		}
		// Expand ~ to home directory
		if strings.HasPrefix(configPath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}
			configPath = filepath.Join(homeDir, configPath[2:])
		}
		// Create parent directory if it doesn't exist
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create config directory: %w", err)
		}
	case "4":
		fmt.Println("Skipping configuration save.")
		return "", nil
	default:
		return "", fmt.Errorf("invalid choice: %s", choice)
	}

	// Check if config file already exists
	var yamlContent string
	if _, err := os.Stat(configPath); err == nil {
		// File exists - ask if they want to merge or replace
		fmt.Printf("\n⚠️  Configuration file already exists: %s\n", configPath)
		fmt.Println("How would you like to proceed?")
		fmt.Println("  1. Merge (add/update providers, keep existing ones)")
		fmt.Println("  2. Replace (overwrite entire file)")
		fmt.Println("  3. Cancel")
		fmt.Println()

		mergeChoice := w.prompt("Select option (1-3): ", false)
		switch mergeChoice {
		case "1":
			// Merge with existing config
			merged, err := w.mergeWithExistingConfig(configPath)
			if err != nil {
				return "", fmt.Errorf("failed to merge with existing config: %w", err)
			}
			yamlContent = merged
			fmt.Println("✅ Configuration will be merged with existing file")
		case "2":
			// Replace - generate fresh YAML
			yamlContent = w.generateYAML()
			fmt.Println("✅ Configuration will replace existing file")
		case "3":
			fmt.Println("Configuration save cancelled.")
			return "", nil
		default:
			return "", fmt.Errorf("invalid choice: %s", mergeChoice)
		}
	} else {
		// File doesn't exist - generate fresh YAML
		yamlContent = w.generateYAML()
	}

	// Write to file
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return configPath, nil
}

// mergeWithExistingConfig loads existing config and merges new provider configurations
func (w *Wizard) mergeWithExistingConfig(configPath string) (string, error) {
	// Read existing config file
	existingData, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read existing config: %w", err)
	}

	// Parse existing YAML into a map
	var existingConfig map[string]interface{}
	if err := yaml.Unmarshal(existingData, &existingConfig); err != nil {
		return "", fmt.Errorf("failed to parse existing config: %w", err)
	}

	// Ensure providers section exists
	providers, ok := existingConfig["providers"].(map[string]interface{})
	if !ok {
		providers = make(map[string]interface{})
		existingConfig["providers"] = providers
	}

	// Helper function to update/add provider config
	updateProvider := func(providerName string, config map[string]interface{}) {
		providers[providerName] = config
	}

	// Helper to add to list if not present
	addToList := func(list []interface{}, item string) []interface{} {
		for _, v := range list {
			if v == item {
				return list // Already in list
			}
		}
		return append(list, item)
	}

	// Get or create preferred_order and enabled lists
	preferredOrder, _ := providers["preferred_order"].([]interface{})
	if preferredOrder == nil {
		preferredOrder = []interface{}{}
	}
	enabled, _ := providers["enabled"].([]interface{})
	if enabled == nil {
		enabled = []interface{}{}
	}

	// Merge Cerebras configuration
	if w.config.cerebrasAPIKey != "" {
		cerebrasConfig := map[string]interface{}{
			"api_key":  w.config.cerebrasAPIKey,
			"base_url": "https://api.cerebras.ai",
		}
		if len(w.config.cerebrasModels) > 0 {
			if len(w.config.cerebrasModels) == 1 {
				cerebrasConfig["model"] = w.config.cerebrasModels[0]
			} else {
				cerebrasConfig["models"] = modelsToInterface(w.config.cerebrasModels)
			}
		} else {
			cerebrasConfig["model"] = "zai-glm-4.6"
		}
		if w.config.cerebrasTemperature != "" {
			tempFloat, _ := strconv.ParseFloat(w.config.cerebrasTemperature, 64)
			cerebrasConfig["temperature"] = tempFloat
		} else {
			cerebrasConfig["temperature"] = 0.6
		}
		if w.config.cerebrasMaxTokens != "" {
			tokens, _ := strconv.Atoi(w.config.cerebrasMaxTokens)
			cerebrasConfig["max_tokens"] = tokens
		}
		updateProvider("cerebras", cerebrasConfig)
		preferredOrder = addToList(preferredOrder, "cerebras")
		enabled = addToList(enabled, "cerebras")
	}

	// Merge OpenRouter configuration
	if w.config.openrouterAPIKey != "" {
		openrouterConfig := map[string]interface{}{
			"api_key":  w.config.openrouterAPIKey,
			"base_url": "https://openrouter.ai/api",
		}
		if len(w.config.openrouterModels) > 0 {
			if len(w.config.openrouterModels) == 1 {
				openrouterConfig["model"] = w.config.openrouterModels[0]
			} else {
				openrouterConfig["models"] = modelsToInterface(w.config.openrouterModels)
			}
		} else {
			openrouterConfig["model"] = "qwen/qwen3-coder"
		}
		if w.config.openrouterSiteURL != "" {
			openrouterConfig["site_url"] = w.config.openrouterSiteURL
		} else {
			openrouterConfig["site_url"] = "https://github.com/cecil-the-coder/mcp-code-api"
		}
		if w.config.openrouterSiteName != "" {
			openrouterConfig["site_name"] = w.config.openrouterSiteName
		} else {
			openrouterConfig["site_name"] = "MCP Code API"
		}
		updateProvider("openrouter", openrouterConfig)
		preferredOrder = addToList(preferredOrder, "openrouter")
		enabled = addToList(enabled, "openrouter")
	}

	// Merge OpenAI configuration
	if w.config.openaiAPIKey != "" {
		openaiConfig := map[string]interface{}{
			"api_key":           w.config.openaiAPIKey,
			"base_url":          "https://api.openai.com/v1",
			"use_responses_api": false,
		}
		if len(w.config.openaiModels) > 0 {
			if len(w.config.openaiModels) == 1 {
				openaiConfig["model"] = w.config.openaiModels[0]
			} else {
				openaiConfig["models"] = modelsToInterface(w.config.openaiModels)
			}
		} else {
			openaiConfig["model"] = "gpt-4o"
		}
		updateProvider("openai", openaiConfig)
		preferredOrder = addToList(preferredOrder, "openai")
		enabled = addToList(enabled, "openai")
	}

	// Merge Anthropic configuration
	if w.config.anthropicAPIKey != "" || w.config.anthropicOAuth != nil {
		anthropicConfig := map[string]interface{}{
			"base_url": "https://api.anthropic.com",
		}
		if w.config.anthropicAPIKey != "" {
			anthropicConfig["api_key"] = w.config.anthropicAPIKey
		}
		if w.config.anthropicOAuth != nil {
			anthropicConfig["oauth"] = map[string]interface{}{
				"access_token":  w.config.anthropicOAuth.AccessToken,
				"refresh_token": w.config.anthropicOAuth.RefreshToken,
				"expires_at":    w.config.anthropicOAuth.ExpiresAt,
				"token_type":    w.config.anthropicOAuth.TokenType,
			}
		}
		if len(w.config.anthropicModels) > 0 {
			if len(w.config.anthropicModels) == 1 {
				anthropicConfig["model"] = w.config.anthropicModels[0]
			} else {
				anthropicConfig["models"] = modelsToInterface(w.config.anthropicModels)
			}
		} else {
			anthropicConfig["model"] = "claude-3-5-sonnet-20241022"
		}
		updateProvider("anthropic", anthropicConfig)
		preferredOrder = addToList(preferredOrder, "anthropic")
		enabled = addToList(enabled, "anthropic")
	}

	// Merge Gemini configuration
	if w.config.geminiAPIKey != "" || w.config.geminiOAuth != nil {
		geminiConfig := map[string]interface{}{
			"base_url": "https://generativelanguage.googleapis.com",
		}
		if w.config.geminiAPIKey != "" {
			geminiConfig["api_key"] = w.config.geminiAPIKey
		}
		if w.config.geminiOAuth != nil {
			geminiConfig["oauth"] = map[string]interface{}{
				"access_token":  w.config.geminiOAuth.AccessToken,
				"refresh_token": w.config.geminiOAuth.RefreshToken,
				"expires_at":    w.config.geminiOAuth.ExpiresAt,
				"token_type":    w.config.geminiOAuth.TokenType,
			}
		}
		if len(w.config.geminiModels) > 0 {
			if len(w.config.geminiModels) == 1 {
				geminiConfig["model"] = w.config.geminiModels[0]
			} else {
				geminiConfig["models"] = modelsToInterface(w.config.geminiModels)
			}
		} else {
			geminiConfig["model"] = "gemini-1.5-pro"
		}
		updateProvider("gemini", geminiConfig)
		preferredOrder = addToList(preferredOrder, "gemini")
		enabled = addToList(enabled, "gemini")
	}

	// Merge Qwen configuration
	if w.config.qwenAPIKey != "" || w.config.qwenOAuth != nil {
		qwenConfig := map[string]interface{}{
			"base_url": "https://dashscope.aliyuncs.com/api/v1",
		}
		if w.config.qwenAPIKey != "" {
			qwenConfig["api_key"] = w.config.qwenAPIKey
		}
		if w.config.qwenOAuth != nil {
			qwenConfig["oauth"] = map[string]interface{}{
				"access_token":  w.config.qwenOAuth.AccessToken,
				"refresh_token": w.config.qwenOAuth.RefreshToken,
				"expires_at":    w.config.qwenOAuth.ExpiresAt,
				"token_type":    w.config.qwenOAuth.TokenType,
			}
		}
		if len(w.config.qwenModels) > 0 {
			if len(w.config.qwenModels) == 1 {
				qwenConfig["model"] = w.config.qwenModels[0]
			} else {
				qwenConfig["models"] = modelsToInterface(w.config.qwenModels)
			}
		} else {
			qwenConfig["model"] = "qwen-max"
		}
		updateProvider("qwen", qwenConfig)
		preferredOrder = addToList(preferredOrder, "qwen")
		enabled = addToList(enabled, "qwen")
	}

	// Update lists
	providers["preferred_order"] = preferredOrder
	providers["enabled"] = enabled

	// Marshal back to YAML
	mergedData, err := yaml.Marshal(existingConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged config: %w", err)
	}

	return string(mergedData), nil
}

// generateYAML generates YAML configuration from collected values
func (w *Wizard) generateYAML() string {
	var sb strings.Builder

	sb.WriteString("# MCP Code API Configuration\n")
	sb.WriteString("# Generated by interactive wizard\n\n")

	sb.WriteString("server:\n")
	sb.WriteString("  name: \"mcp-code-api\"\n")
	sb.WriteString("  version: \"1.0.0\"\n")
	sb.WriteString("  description: \"MCP Code API - Multi-Provider Code Generation Server\"\n")
	sb.WriteString("  timeout: \"60s\"\n\n")

	sb.WriteString("providers:\n")

	// Cerebras configuration
	if w.config.cerebrasAPIKey != "" {
		sb.WriteString("  cerebras:\n")
		sb.WriteString(fmt.Sprintf("    api_key: \"%s\"\n", w.config.cerebrasAPIKey))
		if len(w.config.cerebrasModels) > 0 {
			writeModelsYAML(&sb, w.config.cerebrasModels, "    ")
		} else {
			sb.WriteString("    model: \"zai-glm-4.6\"\n")
		}
		if w.config.cerebrasTemperature != "" {
			sb.WriteString(fmt.Sprintf("    temperature: %s\n", w.config.cerebrasTemperature))
		} else {
			sb.WriteString("    temperature: 0.6\n")
		}
		if w.config.cerebrasMaxTokens != "" {
			sb.WriteString(fmt.Sprintf("    max_tokens: %s\n", w.config.cerebrasMaxTokens))
		}
		sb.WriteString("    base_url: \"https://api.cerebras.ai\"\n\n")
	}

	// OpenRouter configuration
	if w.config.openrouterAPIKey != "" {
		sb.WriteString("  openrouter:\n")
		sb.WriteString(fmt.Sprintf("    api_key: \"%s\"\n", w.config.openrouterAPIKey))
		if len(w.config.openrouterModels) > 0 {
			writeModelsYAML(&sb, w.config.openrouterModels, "    ")
		} else {
			sb.WriteString("    model: \"qwen/qwen3-coder\"\n")
		}
		if w.config.openrouterSiteURL != "" {
			sb.WriteString(fmt.Sprintf("    site_url: \"%s\"\n", w.config.openrouterSiteURL))
		} else {
			sb.WriteString("    site_url: \"https://github.com/cecil-the-coder/mcp-code-api\"\n")
		}
		if w.config.openrouterSiteName != "" {
			sb.WriteString(fmt.Sprintf("    site_name: \"%s\"\n", w.config.openrouterSiteName))
		} else {
			sb.WriteString("    site_name: \"MCP Code API\"\n")
		}
		sb.WriteString("    base_url: \"https://openrouter.ai/api\"\n\n")
	}

	// OpenAI configuration
	if w.config.openaiAPIKey != "" {
		sb.WriteString("  openai:\n")
		sb.WriteString(fmt.Sprintf("    api_key: \"%s\"\n", w.config.openaiAPIKey))
		if len(w.config.openaiModels) > 0 {
			writeModelsYAML(&sb, w.config.openaiModels, "    ")
		} else {
			sb.WriteString("    model: \"gpt-4o\"\n")
		}
		sb.WriteString("    base_url: \"https://api.openai.com/v1\"\n")
		sb.WriteString("    use_responses_api: false\n\n")
	}

	// Anthropic configuration
	if w.config.anthropicAPIKey != "" || w.config.anthropicOAuth != nil {
		sb.WriteString("  anthropic:\n")
		if w.config.anthropicAPIKey != "" {
			sb.WriteString(fmt.Sprintf("    api_key: \"%s\"\n", w.config.anthropicAPIKey))
		}
		if w.config.anthropicOAuth != nil {
			sb.WriteString("    oauth:\n")
			sb.WriteString(fmt.Sprintf("      access_token: \"%s\"\n", w.config.anthropicOAuth.AccessToken))
			sb.WriteString(fmt.Sprintf("      refresh_token: \"%s\"\n", w.config.anthropicOAuth.RefreshToken))
			sb.WriteString(fmt.Sprintf("      expires_at: \"%s\"\n", w.config.anthropicOAuth.ExpiresAt))
			sb.WriteString(fmt.Sprintf("      token_type: \"%s\"\n", w.config.anthropicOAuth.TokenType))
		}
		if len(w.config.anthropicModels) > 0 {
			writeModelsYAML(&sb, w.config.anthropicModels, "    ")
		} else {
			sb.WriteString("    model: \"claude-3-5-sonnet-20241022\"\n")
		}
		sb.WriteString("    base_url: \"https://api.anthropic.com\"\n\n")
	}

	// Gemini configuration
	if w.config.geminiAPIKey != "" || w.config.geminiOAuth != nil {
		sb.WriteString("  gemini:\n")
		if w.config.geminiAPIKey != "" {
			sb.WriteString(fmt.Sprintf("    api_key: \"%s\"\n", w.config.geminiAPIKey))
		}
		if w.config.geminiOAuth != nil {
			sb.WriteString("    oauth:\n")
			sb.WriteString(fmt.Sprintf("      access_token: \"%s\"\n", w.config.geminiOAuth.AccessToken))
			sb.WriteString(fmt.Sprintf("      refresh_token: \"%s\"\n", w.config.geminiOAuth.RefreshToken))
			sb.WriteString(fmt.Sprintf("      expires_at: \"%s\"\n", w.config.geminiOAuth.ExpiresAt))
			sb.WriteString(fmt.Sprintf("      token_type: \"%s\"\n", w.config.geminiOAuth.TokenType))
		}
		if len(w.config.geminiModels) > 0 {
			writeModelsYAML(&sb, w.config.geminiModels, "    ")
		} else {
			sb.WriteString("    model: \"gemini-1.5-pro\"\n")
		}
		sb.WriteString("    base_url: \"https://generativelanguage.googleapis.com\"\n\n")
	}

	// Qwen configuration
	if w.config.qwenAPIKey != "" || w.config.qwenOAuth != nil {
		sb.WriteString("  qwen:\n")
		if w.config.qwenAPIKey != "" {
			sb.WriteString(fmt.Sprintf("    api_key: \"%s\"\n", w.config.qwenAPIKey))
		}
		if w.config.qwenOAuth != nil {
			sb.WriteString("    oauth:\n")
			sb.WriteString(fmt.Sprintf("      access_token: \"%s\"\n", w.config.qwenOAuth.AccessToken))
			sb.WriteString(fmt.Sprintf("      refresh_token: \"%s\"\n", w.config.qwenOAuth.RefreshToken))
			sb.WriteString(fmt.Sprintf("      expires_at: \"%s\"\n", w.config.qwenOAuth.ExpiresAt))
			sb.WriteString(fmt.Sprintf("      token_type: \"%s\"\n", w.config.qwenOAuth.TokenType))
		}
		if len(w.config.qwenModels) > 0 {
			writeModelsYAML(&sb, w.config.qwenModels, "    ")
		} else {
			sb.WriteString("    model: \"qwen-max\"\n")
		}
		sb.WriteString("    base_url: \"https://dashscope.aliyuncs.com/api/v1\"\n\n")
	}

	// Provider ordering
	sb.WriteString("  preferred_order:\n")
	if w.config.cerebrasAPIKey != "" {
		sb.WriteString("    - cerebras\n")
	}
	if w.config.openrouterAPIKey != "" {
		sb.WriteString("    - openrouter\n")
	}
	if w.config.openaiAPIKey != "" {
		sb.WriteString("    - openai\n")
	}
	if w.config.anthropicAPIKey != "" || w.config.anthropicOAuth != nil {
		sb.WriteString("    - anthropic\n")
	}
	if w.config.geminiAPIKey != "" || w.config.geminiOAuth != nil {
		sb.WriteString("    - gemini\n")
	}
	if w.config.qwenAPIKey != "" || w.config.qwenOAuth != nil {
		sb.WriteString("    - qwen\n")
	}
	sb.WriteString("\n")

	// Enabled providers
	sb.WriteString("  enabled:\n")
	if w.config.cerebrasAPIKey != "" {
		sb.WriteString("    - cerebras\n")
	}
	if w.config.openrouterAPIKey != "" {
		sb.WriteString("    - openrouter\n")
	}
	if w.config.openaiAPIKey != "" {
		sb.WriteString("    - openai\n")
	}
	if w.config.anthropicAPIKey != "" || w.config.anthropicOAuth != nil {
		sb.WriteString("    - anthropic\n")
	}
	if w.config.geminiAPIKey != "" || w.config.geminiOAuth != nil {
		sb.WriteString("    - gemini\n")
	}
	if w.config.qwenAPIKey != "" || w.config.qwenOAuth != nil {
		sb.WriteString("    - qwen\n")
	}
	sb.WriteString("\n")

	// Logging configuration
	sb.WriteString("logging:\n")
	sb.WriteString("  level: \"info\"\n")
	sb.WriteString("  verbose: false\n")
	sb.WriteString("  debug: false\n")

	return sb.String()
}
