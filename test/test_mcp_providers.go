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
	"sort"
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

// OAuthConfig holds OAuth authentication configuration
type OAuthConfig struct {
	AccessToken  string `yaml:"access_token" json:"access_token"`
	RefreshToken string `yaml:"refresh_token" json:"refresh_token"`
	ExpiresAt    string `yaml:"expires_at" json:"expires_at"`
	TokenType    string `yaml:"token_type" json:"token_type"`
}

// ProviderConfig holds configuration for a single provider
type ProviderConfig struct {
	Name        string       `yaml:"name" json:"name"`
	DisplayName string       `yaml:"display_name,omitempty" json:"display_name,omitempty"` // Optional display name (defaults to Name)
	APIKey      string       `yaml:"api_key" json:"api_key"`
	APIKeys     []string     `yaml:"api_keys,omitempty" json:"api_keys,omitempty"` // Multiple API keys for load balancing
	OAuth       *OAuthConfig `yaml:"oauth,omitempty" json:"oauth,omitempty"`        // OAuth authentication
	Models      []string     `yaml:"models,omitempty" json:"models,omitempty"`      // Multiple models
	BaseURL     string       `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	Temperature float64      `yaml:"temperature" json:"temperature"`
	MaxTokens   int          `yaml:"max_tokens" json:"max_tokens"`
	SiteURL     string       `yaml:"site_url,omitempty" json:"site_url,omitempty"`
	SiteName    string       `yaml:"site_name,omitempty" json:"site_name,omitempty"`
	Concurrency int          `yaml:"concurrency,omitempty" json:"concurrency,omitempty"` // Max concurrent model tests (default: 1)
	IsLocal     bool         `yaml:"-" json:"is_local"`
	Enabled     bool         `yaml:"-" json:"enabled"`
}

// GetDisplayName returns the display name if set, otherwise returns the provider name
func (p *ProviderConfig) GetDisplayName() string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	return p.Name
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
	TTFT          time.Duration `json:"ttft"` // Time To First Token
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
	configFile    = flag.String("config", "~/.mcp-code-api/config.yaml", "Configuration file path")
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

		// If Models is empty, skip this provider
		if len(provider.Models) == 0 {
			provider.Enabled = false
		}

		// Default concurrency to 1 if not set or invalid
		if provider.Concurrency <= 0 {
			provider.Concurrency = 1
		}

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
	cmd := exec.Command("go", "run", "./main.go", "server", "--config", *configFile)

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

func (c *MCPClient) makeRequest(request MCPRequest) (*MCPResponse, time.Duration, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	request.ID = c.requestID
	c.requestID++

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	requestStart := time.Now()

	_, err = fmt.Fprintln(c.stdin, string(jsonData))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send request: %w", err)
	}

	err = c.stdin.Flush()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to flush stdin: %w", err)
	}

	// Read lines until we get a valid JSON response with timeout
	type scanResult struct {
		line  string
		valid bool
		ttft  time.Duration
		err   error
	}

	resultChan := make(chan scanResult, 1)

	go func() {
		var buffer strings.Builder
		maxLines := 50 // Allow more lines for multi-line JSON
		jsonStarted := false
		var firstDataTime time.Time

		for i := 0; i < maxLines; i++ {
			if c.stdout.Scan() {
				line := c.stdout.Text()

				// Track time to first data (TTFT)
				if firstDataTime.IsZero() && len(strings.TrimSpace(line)) > 0 {
					firstDataTime = time.Now()
				}

				// Check if this line starts JSON
				if strings.HasPrefix(strings.TrimSpace(line), "{") {
					jsonStarted = true
					buffer.Reset() // Start fresh
				}

				// If we're collecting JSON, add this line
				if jsonStarted {
					buffer.WriteString(line)

					// Try to parse accumulated JSON
					accumulated := buffer.String()
					var testJSON json.RawMessage
					if err := json.Unmarshal([]byte(accumulated), &testJSON); err == nil {
						// Valid JSON found!
						ttft := time.Duration(0)
						if !firstDataTime.IsZero() {
							ttft = firstDataTime.Sub(requestStart)
						}
						resultChan <- scanResult{line: accumulated, valid: true, ttft: ttft, err: nil}
						return
					}
					// Not valid yet, continue accumulating
				} else {
					// Skip non-JSON lines (like startup messages)
					if *verboseOutput {
						fmt.Printf("🔍 DEBUG: Skipping line: %s\n", strings.TrimSpace(line))
					}
				}
			} else {
				if err := c.stdout.Err(); err != nil {
					resultChan <- scanResult{line: "", valid: false, ttft: 0, err: err}
					return
				}
			}
		}
		resultChan <- scanResult{line: "", valid: false, ttft: 0, err: fmt.Errorf("no valid JSON response after %d lines", maxLines)}
	}()

	// Wait for response with timeout
	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, 0, result.err
		}
		if !result.valid {
			return nil, 0, fmt.Errorf("no valid JSON response received")
		}
		var response MCPResponse
		err = json.Unmarshal([]byte(result.line), &response)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		return &response, result.ttft, nil
	case <-time.After(30 * time.Second):
		return nil, 0, fmt.Errorf("timeout waiting for JSON response after 30 seconds")
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
	displayName := pt.config.GetDisplayName()
	fmt.Printf("🔍 DEBUG: Starting TestProvider for %s\n", displayName)
	var results []*ModelTestResult

	configured := pt.isConfigured()
	skipReason := pt.getSkipReason()
	fmt.Printf("🔍 DEBUG: Configured=%t, Reason=%s\n", configured, skipReason)

	if !configured {
		fmt.Printf("⚪ %s: %s\n", gray(displayName), skipReason)
		return []*ModelTestResult{
			{
				Provider: displayName,
				Reason:   skipReason,
				Skipped:  true,
			},
		}
	}

	if pt.config.IsLocal {
		fmt.Printf("🔍 DEBUG: IsLocal provider, checking service...\n")
		if !pt.isLocalServiceRunning() {
			fmt.Printf("⚪ %s: %s\n", gray(displayName), skipReason)
			return []*ModelTestResult{
				{
					Provider: displayName,
					Reason:   "Local service not running",
					Skipped:  true,
				},
			}
		}
	}

	fmt.Printf("🔍 DEBUG: About to test %d models for %s (concurrency: %d)\n",
		len(pt.config.Models), displayName, pt.config.Concurrency)

	// Test models with concurrency limit
	var wg sync.WaitGroup
	var resultsMutex sync.Mutex
	semaphore := make(chan struct{}, pt.config.Concurrency)

	for _, model := range pt.config.Models {
		wg.Add(1)
		go func(m string) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("🔍 DEBUG: Testing model %s for %s\n", m, displayName)
			result := pt.testModel(ctx, m)

			resultsMutex.Lock()
			results = append(results, result)
			resultsMutex.Unlock()

			fmt.Printf("🔍 DEBUG: Completed model %s, continuing...\n", m)
		}(model)
	}

	// Wait for all model tests to complete
	wg.Wait()

	return results
}

func (pt *ProviderTester) isConfigured() bool {
	if pt.config.IsLocal {
		return true
	}
	// Check both single APIKey, multiple APIKeys array, and OAuth
	hasKey := (pt.config.APIKey != "" && pt.config.APIKey != "your-api-key-here") ||
		len(pt.config.APIKeys) > 0 ||
		(pt.config.OAuth != nil && pt.config.OAuth.AccessToken != "")
	if *verboseOutput {
		if len(pt.config.APIKeys) > 0 {
			fmt.Printf("🔍 DEBUG: %s has %d API keys, Configured=%t\n", pt.config.Name, len(pt.config.APIKeys), hasKey)
		} else if pt.config.OAuth != nil {
			fmt.Printf("🔍 DEBUG: %s has OAuth, Configured=%t\n", pt.config.Name, hasKey)
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
	if pt.config.OAuth != nil && pt.config.OAuth.AccessToken != "" {
		return "OAuth configured"
	}
	if len(pt.config.APIKeys) > 0 {
		return "Multiple API keys configured"
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

func (pt *ProviderTester) testModel(ctx context.Context, model string) *ModelTestResult {
	displayName := pt.config.GetDisplayName()
	fmt.Printf("🔍 DEBUG: Starting testModel for %s[%s]\n", displayName, model)
	start := time.Now()
	result := &ModelTestResult{
		Provider:     displayName,
		Model:        model,
		Configured:   true,
		ResponseTime: 0,
	}

	// Create a dedicated client for this model test
	fmt.Printf("🔍 DEBUG: Creating MCP client for %s[%s]...\n", displayName, model)
	client, err := NewMCPClient()
	if err != nil {
		fmt.Printf("🔍 DEBUG: Failed to create MCP client: %s\n", err)
		result.Error = fmt.Sprintf("Failed to create MCP client: %s", err)
		result.Skipped = true
		result.Reason = fmt.Sprintf("Client creation failed: %s", err)
		return result
	}
	defer func() { _ = client.Stop() }()

	// Start the MCP server
	if err := client.Start(); err != nil {
		fmt.Printf("🔍 DEBUG: Failed to start MCP server: %s\n", err)
		result.Error = fmt.Sprintf("Failed to start MCP server: %s", err)
		result.Skipped = true
		result.Reason = fmt.Sprintf("Server start failed: %s", err)
		return result
	}

	// Give server time to initialize
	time.Sleep(2 * time.Second)

	fmt.Printf("🔍 DEBUG: Calling testInitialize for %s[%s]\n", displayName, model)
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

	fmt.Printf("🔍 DEBUG: Initialize passed for %s[%s]\n", displayName, model)
	if *verboseOutput {
		fmt.Printf("✅ %s[%s]: Initialization passed (%dms)\n", green(displayName), model, result.InitTime.Milliseconds())
	} else {
		fmt.Printf("✅ %s[%s]: Initialization passed\n", green(displayName), model)
	}

	fmt.Printf("🔍 DEBUG: Calling testTools for %s[%s]\n", displayName, model)
	if err := pt.testTools(ctx, client); err != nil {
		fmt.Printf("🔍 DEBUG: testTools failed: %s\n", err)
		result.Error = err.Error()
		result.Skipped = true
		result.Reason = fmt.Sprintf("Tools test failed: %s", err)
		result.ResponseTime = time.Since(start)
		return result
	}

	fmt.Printf("🔍 DEBUG: Tools passed for %s[%s]\n", displayName, model)
	fmt.Printf("🔍 DEBUG: Calling testWriteFile for %s[%s]\n", displayName, model)
	writeStart := time.Now()
	outputFile, generatedCode, ttft, err := pt.testWriteFileWithCapture(ctx, model, client)
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
	result.TTFT = ttft
	result.OutputFile = outputFile
	result.GeneratedCode = generatedCode

	// Calculate total response time after all tests complete
	result.ResponseTime = time.Since(start)

	fmt.Printf("🔍 DEBUG: All tests passed for %s[%s] - Write took %dms, Total: %dms\n",
		displayName, model, result.WriteTime.Milliseconds(), result.ResponseTime.Milliseconds())
	result.APITest = true
	result.ChatTest = true
	result.ToolsTest = true

	if *verboseOutput {
		fmt.Printf("✅ %s[%s]: All tests passed - Write: %dms, Total: %dms, Output: %s\n",
			green(displayName), model, result.WriteTime.Milliseconds(), result.ResponseTime.Milliseconds(), outputFile)
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

	resp, _, err := client.makeRequest(request)
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

	resp, _, err := client.makeRequest(request)
	if err != nil {
		return fmt.Errorf("tools list test failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("MCP error during tools list: %s", resp.Error.Message)
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		if tools, ok := result["tools"].([]interface{}); ok {
			if *verboseOutput {
				fmt.Printf("✅ %s: Tools test passed (%d tools available)\n", green(pt.config.GetDisplayName()), len(tools))
			}
			return nil
		}
	}

	return fmt.Errorf("no tools found in response")
}

func (pt *ProviderTester) testWriteFileWithCapture(ctx context.Context, model string, client *MCPClient) (string, string, time.Duration, error) {
	displayName := pt.config.GetDisplayName()
	// Create output file path
	timestamp := time.Now().Unix()
	// Sanitize model name for filename
	sanitizedModel := strings.ReplaceAll(model, "/", "_")
	sanitizedModel = strings.ReplaceAll(sanitizedModel, ":", "_")
	outputFile := fmt.Sprintf("/tmp/test_%s_%s_%d.txt", displayName, sanitizedModel, timestamp)

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "write",
			"arguments": map[string]interface{}{
				"file_path": outputFile,
				"prompt":    fmt.Sprintf("Test message to %s[%s]: %s", displayName, model, TEST_MESSAGE),
				"provider":  pt.config.Name, // Keep real provider name for API routing
				"model":     model,
			},
		},
	}

	resp, ttft, err := client.makeRequest(request)
	if err != nil {
		return outputFile, "", 0, fmt.Errorf("write file test failed: %w", err)
	}

	if resp.Error != nil {
		return outputFile, "", ttft, fmt.Errorf("MCP error during write file: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		return outputFile, "", ttft, fmt.Errorf("no result in write file response")
	}

	// Read the generated code from the file
	generatedCode := ""
	if content, err := os.ReadFile(outputFile); err == nil {
		generatedCode = string(content)
	} else {
		fmt.Printf("⚠️  Warning: Could not read generated file %s: %v\n", outputFile, err)
	}

	return outputFile, generatedCode, ttft, nil
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

	// Sort provider names for consistent output
	var providerNames []string
	for name := range results {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)

	for _, providerName := range providerNames {
		modelResults := results[providerName]
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
	fmt.Println(strings.Repeat("-", 110))
	fmt.Printf("%-50s %-8s %-8s %-10s %-10s %-10s %-12s\n",
		"Model", "API", "Tools", "Init(ms)", "TTFT(ms)", "Write(ms)", "Total(ms)")
	fmt.Println(strings.Repeat("-", 110))

	// Sort provider names for detailed results
	for _, providerName := range providerNames {
		modelResults := results[providerName]

		// Sort model results by model name for consistent output
		sort.Slice(modelResults, func(i, j int) bool {
			return modelResults[i].Model < modelResults[j].Model
		})

		fmt.Printf("\n%s (%d %s):\n", cyan(providerName), len(modelResults), pluralize("model", len(modelResults)))
		for _, modelResult := range modelResults {
			initTime := "N/A"
			ttftTime := "N/A"
			writeTime := "N/A"
			totalTime := "N/A"
			if modelResult.InitTime > 0 {
				initTime = fmt.Sprintf("%d", modelResult.InitTime.Milliseconds())
			}
			if modelResult.TTFT > 0 {
				ttftTime = fmt.Sprintf("%d", modelResult.TTFT.Milliseconds())
			}
			if modelResult.WriteTime > 0 {
				writeTime = fmt.Sprintf("%d", modelResult.WriteTime.Milliseconds())
			}
			if modelResult.ResponseTime > 0 {
				totalTime = fmt.Sprintf("%d", modelResult.ResponseTime.Milliseconds())
			}

			fmt.Printf("  %-48s %-8s %-8s %-10s %-10s %-10s %-12s\n",
				modelResult.Model,
				boolToEmoji(modelResult.APITest),
				boolToEmoji(modelResult.ToolsTest),
				initTime,
				ttftTime,
				writeTime,
				totalTime,
			)
		}
	}

	fmt.Println(strings.Repeat("-", 110))

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

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
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
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup

	fmt.Println("\n🧪 Running provider tests concurrently...")

	// Run each provider test concurrently
	for _, tester := range testers {
		wg.Add(1)
		go func(t *ProviderTester) {
			defer wg.Done()
			providerResults := t.TestProvider(ctx)
			if len(providerResults) > 0 {
				resultsMutex.Lock()
				results[t.config.GetDisplayName()] = providerResults
				resultsMutex.Unlock()
			}
		}(tester)
	}

	// Wait for all provider tests to complete
	wg.Wait()

	printResults(results)

	fmt.Printf("\n⏰ Test completed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}
