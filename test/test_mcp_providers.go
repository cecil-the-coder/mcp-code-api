package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

// =============================================
// CONFIGURATION STRUCTURES
// =============================================

// Config represents the full configuration file
type Config struct {
	Providers map[string]ProviderConfig `yaml:"providers"`
	Server    ServerConfig              `yaml:"server"`
	Logging   LoggingConfig             `yaml:"logging"`
}

// ProviderConfig holds configuration for a single provider
type ProviderConfig struct {
	Name        string   `yaml:"name" json:"name"`
	APIKey      string   `yaml:"api_key" json:"api_key"`
	APIKeys     []string `yaml:"api_keys,omitempty" json:"api_keys,omitempty"` // Multiple API keys for load balancing
	Models      []string `yaml:"models" json:"models"`
	BaseURL     string   `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	Temperature float64  `yaml:"temperature" json:"temperature"`
	MaxTokens   int      `yaml:"max_tokens" json:"max_tokens"`
	SiteURL     string   `yaml:"site_url,omitempty" json:"site_url,omitempty"`
	SiteName    string   `yaml:"site_name,omitempty" json:"site_name,omitempty"`
	IsLocal     bool     `yaml:"-" json:"is_local"`
	Enabled     bool     `yaml:"-" json:"enabled"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
	Timeout     string `yaml:"timeout"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level   string `yaml:"level"`
	Verbose bool   `yaml:"verbose"`
	Debug   bool   `yaml:"debug"`
}

// ModelTestResult holds test results for a specific model
type ModelTestResult struct {
	Provider      string        `json:"provider"`
	Model         string        `json:"model"`
	Configured    bool          `json:"configured"`
	APITest       bool          `json:"api_test"`
	ChatTest      bool          `json:"chat_test"`
	ToolsTest     bool          `json:"tools_test"`
	ResponseTime  time.Duration `json:"response_time"`
	WriteTime     time.Duration `json:"write_time"`
	InitTime      time.Duration `json:"init_time"`
	Error         string        `json:"error,omitempty"`
	ModelInfo     interface{}   `json:"model_info,omitempty"`
	Skipped       bool          `json:"skipped"`
	Reason        string        `json:"reason,omitempty"`
	GeneratedCode string        `json:"generated_code,omitempty"`
	OutputFile    string        `json:"output_file,omitempty"`
}

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Command-line flags
var (
	configFile    = flag.String("config", "test-config.yaml", "Configuration file path")
	verboseOutput = flag.Bool("verbose", false, "Show verbose test output")
	showHelp      = flag.Bool("help", false, "Show usage information")
)

// Global configuration
var globalConfig Config
var providers map[string]*ProviderConfig

// =============================================
// CONFIGURATION LOADING
// =============================================

func loadConfig(configPath string) (*Config, error) {
	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Process environment variable substitutions
	for name, provider := range config.Providers {
		// Substitute environment variables in API keys
		if strings.HasPrefix(provider.APIKey, "${") && strings.HasSuffix(provider.APIKey, "}") {
			envVar := strings.TrimPrefix(strings.TrimSuffix(provider.APIKey, "}"), "${")
			provider.APIKey = os.Getenv(envVar)
		}

		// Set provider name from map key
		provider.Name = name
		provider.Enabled = true

		// Mark local providers
		localProviders := map[string]bool{
			"lmstudio": true,
			"llamacpp": true,
			"ollama":   true,
		}
		provider.IsLocal = localProviders[name]

		// Update the map
		config.Providers[name] = provider
	}

	return &config, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if usr, err := user.Current(); err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	}
	return path
}

// =============================================
// COLOR SETUP
// =============================================

var (
	green = color.New(color.FgGreen).SprintFunc()
	cyan  = color.New(color.FgCyan).SprintFunc()
	gray  = color.New(color.FgHiBlack).SprintFunc()
)

// =============================================
// HELPER FUNCTIONS
// =============================================

// MCPClient represents a client that communicates with MCP server via stdio
type MCPClient struct {
	cmd       *exec.Cmd
	stdin     *bufio.Writer
	stdout    *bufio.Scanner
	stderr    *bufio.Scanner
	requestID int
	mutex     sync.Mutex
}

func NewMCPClient() (*MCPClient, error) {
	cmd := exec.Command("go", "run", "main.go", "server", "--config", *configFile)

	// Build environment variables from config
	var envVars []string
	for name, provider := range providers {
		if !provider.IsLocal {
			// Support both single APIKey and multiple APIKeys
			if provider.APIKey != "" {
				varName := strings.ToUpper(name) + "_API_KEY"
				envVars = append(envVars, fmt.Sprintf("%s=%s", varName, provider.APIKey))
			}
			// If multiple keys, set the first one as default env var for backward compatibility
			if len(provider.APIKeys) > 0 {
				varName := strings.ToUpper(name) + "_API_KEY"
				envVars = append(envVars, fmt.Sprintf("%s=%s", varName, provider.APIKeys[0]))
			}
		}
	}

	cmd.Env = append(os.Environ(), envVars...)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	client := &MCPClient{
		cmd:       cmd,
		stdin:     bufio.NewWriter(stdinPipe),
		stdout:    bufio.NewScanner(stdoutPipe),
		stderr:    bufio.NewScanner(stderrPipe),
		requestID: 1,
	}

	return client, nil
}

func (c *MCPClient) Start() error {
	return c.cmd.Start()
}

func (c *MCPClient) Stop() error {
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Signal(syscall.SIGTERM)
		return c.cmd.Wait()
	}
	return nil
}

func (c *MCPClient) makeRequest(request MCPRequest) (*MCPResponse, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	request.ID = c.requestID
	c.requestID++

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = fmt.Fprintln(c.stdin, string(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	err = c.stdin.Flush()
	if err != nil {
		return nil, fmt.Errorf("failed to flush stdin: %w", err)
	}

	// Read lines until we get a valid JSON response with timeout
	type scanResult struct {
		line  string
		valid bool
		err   error
	}

	resultChan := make(chan scanResult, 1)

	go func() {
		maxLines := 20 // prevent infinite loop
		for i := 0; i < maxLines; i++ {
			if c.stdout.Scan() {
				line := c.stdout.Text()
				// Check if this line looks like JSON
				if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
					resultChan <- scanResult{line: line, valid: true, err: nil}
					return
				}
				// Skip instruction lines and other non-JSON content
				if *verboseOutput {
					fmt.Printf("🔍 DEBUG: Skipping line: %s\n", strings.TrimSpace(line))
				}
			} else {
				if err := c.stdout.Err(); err != nil {
					resultChan <- scanResult{line: "", valid: false, err: err}
					return
				}
			}
		}
		resultChan <- scanResult{line: "", valid: false, err: fmt.Errorf("no valid JSON response after %d lines", maxLines)}
	}()

	// Wait for response with timeout
	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		if !result.valid {
			return nil, fmt.Errorf("no valid JSON response received")
		}
		var response MCPResponse
		err = json.Unmarshal([]byte(result.line), &response)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		return &response, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for JSON response after 10 seconds")
	}
}

// =============================================
// TESTING FUNCTIONS
// =============================================

type ProviderTester struct {
	config *ProviderConfig
}

func NewProviderTester(config *ProviderConfig) *ProviderTester {
	return &ProviderTester{
		config: config,
	}
}

func (pt *ProviderTester) TestProvider(ctx context.Context) []*ModelTestResult {
	fmt.Printf("🔍 DEBUG: Starting TestProvider for %s\n", pt.config.Name)
	var results []*ModelTestResult

	configured := pt.isConfigured()
	skipReason := pt.getSkipReason()
	fmt.Printf("🔍 DEBUG: Configured=%t, Reason=%s\n", configured, skipReason)

	if !configured {
		fmt.Printf("⚪ %s: %s\n", gray(pt.config.Name), skipReason)
		return []*ModelTestResult{
			{
				Provider: pt.config.Name,
				Reason:   skipReason,
				Skipped:  true,
			},
		}
	}

	if pt.config.IsLocal {
		fmt.Printf("🔍 DEBUG: IsLocal provider, checking service...\n")
		if !pt.isLocalServiceRunning() {
			fmt.Printf("⚪ %s: %s\n", gray(pt.config.Name), skipReason)
			return []*ModelTestResult{
				{
					Provider: pt.config.Name,
					Reason:   "Local service not running",
					Skipped:  true,
				},
			}
		}
	}

	fmt.Printf("🔍 DEBUG: Creating MCP client for %s...\n", pt.config.Name)
	client, err := NewMCPClient()
	if err != nil {
		fmt.Printf("🔍 DEBUG: Failed to create MCP client: %s\n", err)
		return []*ModelTestResult{
			{
				Provider: pt.config.Name,
				Error:    fmt.Sprintf("Failed to create MCP client: %s", err),
				Skipped:  true,
			},
		}
	}

	fmt.Printf("🔍 DEBUG: MCP client created successfully for %s\n", pt.config.Name)
	defer func() { _ = client.Stop() }()

	// Start the MCP server
	if err := client.Start(); err != nil {
		fmt.Printf("🔍 DEBUG: Failed to start MCP server: %s\n", err)
		return []*ModelTestResult{
			{
				Provider: pt.config.Name,
				Error:    fmt.Sprintf("Failed to start MCP server: %s", err),
				Skipped:  true,
			},
		}
	}
	fmt.Printf("🔍 DEBUG: MCP server started successfully for %s\n", pt.config.Name)

	// Give server time to initialize properly
	time.Sleep(2 * time.Second)
	fmt.Printf("🔍 DEBUG: About to test %d models for %s\n", len(pt.config.Models), pt.config.Name)

	for _, model := range pt.config.Models {
		fmt.Printf("🔍 DEBUG: Testing model %s for %s\n", model, pt.config.Name)
		result := pt.testModel(ctx, model, client)
		results = append(results, result)
		time.Sleep(1 * time.Second)
		fmt.Printf("🔍 DEBUG: Completed model %s, continuing...\n", model)
	}

	return results
}

func (pt *ProviderTester) isConfigured() bool {
	if pt.config.IsLocal {
		return true
	}
	// Check both single APIKey and multiple APIKeys array
	hasKey := (pt.config.APIKey != "" && pt.config.APIKey != "your-api-key-here") || len(pt.config.APIKeys) > 0
	if *verboseOutput {
		if len(pt.config.APIKeys) > 0 {
			fmt.Printf("🔍 DEBUG: %s has %d API keys, Configured=%t\n", pt.config.Name, len(pt.config.APIKeys), hasKey)
		} else {
			fmt.Printf("🔍 DEBUG: %s APIKey='%s', Configured=%t\n", pt.config.Name, pt.config.APIKey, hasKey)
		}
	}
	return hasKey
}

func (pt *ProviderTester) getSkipReason() string {
	if pt.config.IsLocal {
		if !pt.isLocalServiceRunning() {
			switch pt.config.Name {
			case "lmstudio":
				return "LM Studio not running on localhost:1234"
			case "llamacpp":
				return "llama.cpp not running on localhost:8080"
			case "ollama":
				return "Ollama not running on localhost:11434"
			default:
				return "Local service not running"
			}
		}
		return "Local provider available"
	}
	if pt.config.APIKey == "" || pt.config.APIKey == "your-api-key-here" {
		return "Skipped (missing API key)"
	}
	return "API key available"
}

func (pt *ProviderTester) isLocalServiceRunning() bool {
	if !pt.config.IsLocal || pt.config.BaseURL == "" {
		return false
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(pt.config.BaseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		endpoints := []string{"/v1", "/models", "/"}
		for _, endpoint := range endpoints {
			resp, err := client.Get(pt.config.BaseURL + endpoint)
			if err == nil && resp.StatusCode == 200 {
				return true
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
		return false
	}

	return true
}

func (pt *ProviderTester) testModel(ctx context.Context, model string, client *MCPClient) *ModelTestResult {
	fmt.Printf("🔍 DEBUG: Starting testModel for %s[%s]\n", pt.config.Name, model)
	start := time.Now()
	result := &ModelTestResult{
		Provider:     pt.config.Name,
		Model:        model,
		Configured:   true,
		ResponseTime: 0,
	}

	fmt.Printf("🔍 DEBUG: Calling testInitialize for %s[%s]\n", pt.config.Name, model)
	initStart := time.Now()
	if err := pt.testInitialize(ctx, client); err != nil {
		fmt.Printf("🔍 DEBUG: testInitialize failed: %s\n", err)
		result.Error = err.Error()
		result.Skipped = true
		result.Reason = fmt.Sprintf("Initialize failed: %s", err)
		result.ResponseTime = time.Since(start)
		result.InitTime = time.Since(initStart)
		return result
	}
	result.InitTime = time.Since(initStart)

	fmt.Printf("🔍 DEBUG: Initialize passed for %s[%s]\n", pt.config.Name, model)
	if *verboseOutput {
		fmt.Printf("✅ %s[%s]: Initialization passed (%dms)\n", green(pt.config.Name), model, result.InitTime.Milliseconds())
	} else {
		fmt.Printf("✅ %s[%s]: Initialization passed\n", green(pt.config.Name), model)
	}

	fmt.Printf("🔍 DEBUG: Calling testTools for %s[%s]\n", pt.config.Name, model)
	if err := pt.testTools(ctx, client); err != nil {
		fmt.Printf("🔍 DEBUG: testTools failed: %s\n", err)
		result.Error = err.Error()
		result.Skipped = true
		result.Reason = fmt.Sprintf("Tools test failed: %s", err)
		result.ResponseTime = time.Since(start)
		return result
	}

	fmt.Printf("🔍 DEBUG: Tools passed for %s[%s]\n", pt.config.Name, model)
	fmt.Printf("🔍 DEBUG: Calling testWriteFile for %s[%s]\n", pt.config.Name, model)
	writeStart := time.Now()
	outputFile, generatedCode, err := pt.testWriteFileWithCapture(ctx, model, client)
	if err != nil {
		fmt.Printf("🔍 DEBUG: testWriteFile failed: %s\n", err)
		result.Error = err.Error()
		result.Skipped = true
		result.Reason = fmt.Sprintf("Write file test failed: %s", err)
		result.ResponseTime = time.Since(start)
		result.WriteTime = time.Since(writeStart)
		return result
	}
	result.WriteTime = time.Since(writeStart)
	result.OutputFile = outputFile
	result.GeneratedCode = generatedCode

	// Calculate total response time after all tests complete
	result.ResponseTime = time.Since(start)

	fmt.Printf("🔍 DEBUG: All tests passed for %s[%s] - Write took %dms, Total: %dms\n",
		pt.config.Name, model, result.WriteTime.Milliseconds(), result.ResponseTime.Milliseconds())
	result.APITest = true
	result.ChatTest = true
	result.ToolsTest = true

	if *verboseOutput {
		fmt.Printf("✅ %s[%s]: All tests passed - Write: %dms, Total: %dms, Output: %s\n",
			green(pt.config.Name), model, result.WriteTime.Milliseconds(), result.ResponseTime.Milliseconds(), outputFile)
	}

	return result
}

func (pt *ProviderTester) testInitialize(ctx context.Context, client *MCPClient) error {
	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "mcp-test-client",
				"version": "1.0.0",
			},
		},
	}

	resp, err := client.makeRequest(request)
	if err != nil {
		return fmt.Errorf("initialize test failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("MCP error during initialize: %s", resp.Error.Message)
	}

	return nil
}

func (pt *ProviderTester) testTools(ctx context.Context, client *MCPClient) error {
	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	resp, err := client.makeRequest(request)
	if err != nil {
		return fmt.Errorf("tools list test failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("MCP error during tools list: %s", resp.Error.Message)
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		if tools, ok := result["tools"].([]interface{}); ok {
			if *verboseOutput {
				fmt.Printf("✅ %s: Tools test passed (%d tools available)\n", green(pt.config.Name), len(tools))
			}
			return nil
		}
	}

	return fmt.Errorf("no tools found in response")
}

func (pt *ProviderTester) testWriteFileWithCapture(ctx context.Context, model string, client *MCPClient) (string, string, error) {
	// Create output file path
	timestamp := time.Now().Unix()
	// Sanitize model name for filename
	sanitizedModel := strings.ReplaceAll(model, "/", "_")
	sanitizedModel = strings.ReplaceAll(sanitizedModel, ":", "_")
	outputFile := fmt.Sprintf("/tmp/test_%s_%s_%d.txt", pt.config.Name, sanitizedModel, timestamp)

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "write",
			"arguments": map[string]interface{}{
				"file_path": outputFile,
				"prompt":    fmt.Sprintf("Test message to %s[%s]: %s", pt.config.Name, model, TEST_MESSAGE),
				"provider":  pt.config.Name,
				"model":     model,
			},
		},
	}

	resp, err := client.makeRequest(request)
	if err != nil {
		return outputFile, "", fmt.Errorf("write file test failed: %w", err)
	}

	if resp.Error != nil {
		return outputFile, "", fmt.Errorf("MCP error during write file: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		return outputFile, "", fmt.Errorf("no result in write file response")
	}

	// Read the generated code from the file
	generatedCode := ""
	if content, err := os.ReadFile(outputFile); err == nil {
		generatedCode = string(content)
	} else {
		fmt.Printf("⚠️  Warning: Could not read generated file %s: %v\n", outputFile, err)
	}

	return outputFile, generatedCode, nil
}

// =============================================
// MAIN EXECUTION
// =============================================

// Configuration
const (
	TEST_MESSAGE = `Write a complete, production-ready Go function that implements a concurrent HTTP rate limiter using the token bucket algorithm. The function should:
1. Accept requests per second as a parameter
2. Handle concurrent requests safely using mutexes
3. Return whether the request is allowed or should be rate limited
4. Include comprehensive error handling
5. Add detailed comments explaining the algorithm
6. Include example usage in comments

Make this a real, working implementation with proper Go idioms and best practices.`
)

func printResults(results map[string][]*ModelTestResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 MULTI-MODEL TEST RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	var totalProviders, totalModels, configuredProviders, testedModels int
	var totalTests, apiPassed, chatPassed, toolsPassed int

	for providerName, modelResults := range results {
		totalProviders++
		configured := false
		providerTested := false

		for _, modelResult := range modelResults {
			totalModels++
			if modelResult.Configured && !modelResult.Skipped {
				configured = true
				testedModels++
				if modelResult.APITest {
					apiPassed++
				}
				if modelResult.ChatTest {
					chatPassed++
				}
				if modelResult.ToolsTest {
					toolsPassed++
				}
				totalTests++
				providerTested = true
			}
		}

		if configured {
			configuredProviders++
		}

		if providerTested {
			fmt.Printf("✅ %s: Configured & Tested\n", green(providerName))
		} else {
			fmt.Printf("⚪ %s: %s\n", gray(providerName), modelResults[0].Reason)
		}
	}

	fmt.Printf("📋 Total Providers: %d\n", totalProviders)
	fmt.Printf("🧪 Total Models: %d\n", totalModels)
	fmt.Printf("🔧 Configured Providers: %d\n", configuredProviders)
	fmt.Printf("🎯 Models Tested: %d\n", testedModels)
	fmt.Printf("🌐 API Tests Passed: %d/%d\n", apiPassed, totalTests)
	fmt.Printf("💬 Chat Tests Passed: %d/%d\n", chatPassed, totalTests)
	fmt.Printf("🔧 Tools Tests Passed: %d/%d\n", toolsPassed, totalTests)

	fmt.Println("\n📈 Detailed Results:")
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("%-15s %-35s %-8s %-8s %-10s %-10s %-12s\n",
		"Provider", "Model", "API", "Tools", "Init(ms)", "Write(ms)", "Total(ms)")
	fmt.Println(strings.Repeat("-", 100))

	for providerName, modelResults := range results {
		fmt.Printf("%s (%d models):\n", cyan(providerName), len(modelResults))
		for _, modelResult := range modelResults {
			if modelResult.Skipped {
				fmt.Printf("  ⚪ %-35s: %s\n", modelResult.Model, modelResult.Reason)
			} else {
				initTime := "N/A"
				writeTime := "N/A"
				totalTime := "N/A"
				if modelResult.InitTime > 0 {
					initTime = fmt.Sprintf("%d", modelResult.InitTime.Milliseconds())
				}
				if modelResult.WriteTime > 0 {
					writeTime = fmt.Sprintf("%d", modelResult.WriteTime.Milliseconds())
				}
				if modelResult.ResponseTime > 0 {
					totalTime = fmt.Sprintf("%d", modelResult.ResponseTime.Milliseconds())
				}

				fmt.Printf("  %-35s %-8s %-8s %-10s %-10s %-12s\n",
					modelResult.Model,
					boolToEmoji(modelResult.APITest),
					boolToEmoji(modelResult.ToolsTest),
					initTime,
					writeTime,
					totalTime,
				)

				// Show output file if available
				if modelResult.OutputFile != "" && *verboseOutput {
					codePreview := modelResult.GeneratedCode
					if len(codePreview) > 100 {
						codePreview = codePreview[:100] + "..."
					}
					fmt.Printf("     📄 Output: %s (%.1f KB)\n", modelResult.OutputFile, float64(len(modelResult.GeneratedCode))/1024)
					if codePreview != "" {
						fmt.Printf("     📝 Preview: %s\n", strings.TrimSpace(codePreview))
					}
				}
			}
		}
	}

	fmt.Println(strings.Repeat("-", 80))

	fmt.Println("\n🎯 Recommendations:")
	if totalModels == 0 {
		fmt.Println("- No models configured! Update config file with models to test.")
	} else if configuredProviders == 0 {
		fmt.Println("- Set your API keys as environment variables")
	} else if apiPassed < totalTests {
		fmt.Println("- Check API keys and server connectivity for failed providers")
	} else if chatPassed < totalTests {
		fmt.Println("- Some providers may have issues with file operations")
	} else {
		fmt.Println("- All configured models are working! 🎉")
	}
}

func boolToEmoji(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}

func printUsage() {
	fmt.Println("🚀 MCP Multi-Model Provider Test Suite")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ./test_mcp_providers [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --config string    Configuration file path (default: test-config.yaml)")
	fmt.Println("  --verbose         Show verbose test output")
	fmt.Println("  --help, -h       Show this help message")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Models are configured in the YAML config file under 'providers' section.")
	fmt.Println("  Each provider can have multiple models defined in the 'models' list.")
	fmt.Println("  API keys are read from environment variables (see config file).")
	fmt.Println()
	fmt.Println("Example config structure:")
	fmt.Println("  providers:")
	fmt.Println("    openai:")
	fmt.Println("      api_key: \"${OPENAI_API_KEY}\"")
	fmt.Println("      models:")
	fmt.Println("        - \"gpt-4o-mini\"")
	fmt.Println("        - \"gpt-4o\"")
}

func main() {
	flag.Parse()
	if *showHelp {
		printUsage()
		return
	}

	// Load configuration from file
	configPath := expandPath(*configFile)
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %s\n", err)
		os.Exit(1)
	}

	globalConfig = *config
	providers = make(map[string]*ProviderConfig)
	for name, provider := range globalConfig.Providers {
		// Create a copy to avoid pointer issues
		providerCopy := provider
		providers[name] = &providerCopy
	}

	fmt.Println("🚀 Starting MCP Multi-Model Provider Tests")
	fmt.Printf("📝 Using config file: %s\n", configPath)
	fmt.Printf("📝 Testing via stdio communication\n")
	fmt.Printf("⏰ Started at: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	// Create testers for all enabled providers
	var testers []*ProviderTester
	for _, config := range providers {
		if config.Enabled {
			testers = append(testers, NewProviderTester(config))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Received interrupt signal, shutting down gracefully...")
		cancel()
		os.Exit(0)
	}()

	results := make(map[string][]*ModelTestResult)

	fmt.Println("\n🧪 Running provider tests...")

	for _, tester := range testers {
		providerResults := tester.TestProvider(ctx)
		if len(providerResults) > 0 {
			results[tester.config.Name] = providerResults
		}
		time.Sleep(1 * time.Second)
	}

	printResults(results)

	fmt.Printf("\n⏰ Test completed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}
