package api
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)
const (
	cloudcodeBaseURL           = "https://cloudcode-pa.googleapis.com/v1internal"
	standardGeminiBaseURL      = "https://generativelanguage.googleapis.com/v1beta"
	geminiDefaultModel         = "gemini-2.0-flash-exp"
)
// GeminiClient handles Gemini API interactions with OAuth authentication and token refresh
type GeminiClient struct {
	config             config.GeminiConfig
	client             *http.Client
	oauth2Config       *oauth2.Config
	oauth2Token        *oauth2.Token
	tokenMutex         sync.RWMutex
}
func NewGeminiClient(cfg config.GeminiConfig) *GeminiClient {
	client := &GeminiClient{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	if cfg.ClientID != "" && cfg.RefreshToken != "" {
		client.oauth2Config = client.createOAuth2Config()
		client.oauth2Token = client.createOAuth2Token()
		logger.Debugf("Gemini: OAuth token refresh enabled")
	}
	return client
}
func (c *GeminiClient) getEndpoint(model string) string {
	baseURL := c.getBaseURL()

	// Cloud Code API uses ":generateContent" format
	if baseURL == cloudcodeBaseURL || c.oauth2Token != nil {
		return ":generateContent"
	}

	// Standard Gemini API uses "models/{model}:generateContent" format
	return fmt.Sprintf("models/%s:generateContent", model)
}

func (c *GeminiClient) GenerateCode(ctx context.Context, prompt, contextStr, outputFile string, language *string, contextFiles []string) (*types.CodeGenerationResult, error) {
	detectedLanguage := utils.GetLanguageFromFile(outputFile, language)
	fullPrompt := c.buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage, contextFiles)
	model := c.config.Model
	if model == "" {
		model = geminiDefaultModel
	}
	endpoint := c.getEndpoint(model)
	reqBody := GenerateContentRequest{
		Contents: []Content{
			{
				Role: "user",
				Parts: []Part{
					{Text: fullPrompt},
				},
			},
		},
		GenerationConfig: &GenerationConfig{
			Temperature:     0.7,
			TopP:            0.95,
			TopK:            40,
			MaxOutputTokens: 8192,
		},
	}
	var requestBody interface{}

	// Cloud Code API requires onboarding and wrapper format
	if c.oauth2Token != nil {
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" && c.config.ProjectID != "" {
			projectID = c.config.ProjectID
		}

		// Always attempt onboarding if config doesn't have project ID (means we haven't onboarded yet)
		if c.config.ProjectID == "" {
			logger.Debugf("Gemini: Attempting onboarding (env project ID: %q)...", projectID)
			onboardedID, err := c.SetupUserProject(ctx)
			if err != nil {
				if projectID != "" {
					// We have env project ID, warn but continue
					logger.Warnf("Gemini: Onboarding failed, will use env project ID: %v", err)
				} else {
					return nil, fmt.Errorf("onboarding failed: %w", err)
				}
			} else {
				projectID = onboardedID
				// Persist to config so we don't onboard again
				if err := c.persistProjectID(projectID); err != nil {
					logger.Warnf("Failed to persist project ID: %v", err)
				}
				logger.Debugf("Gemini: Onboarding successful, project ID: %s", projectID)
			}
		} else {
			logger.Debugf("Gemini: Using project ID: %s", projectID)
		}

		// Cloud Code API uses wrapper format
		requestBody = CloudCodeRequestWrapper{
			Model:   model,
			Project: projectID,
			Request: reqBody,
		}
	} else {
		// Standard API uses request directly
		requestBody = reqBody
	}

	logger.Debugf("Gemini: Calling API with model %s", model)
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}
	resp, err := c.doRequest(ctx, "POST", endpoint, requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API error: %d - %s", resp.StatusCode, string(body))
	}

	var apiResp GenerateContentResponse
	if c.oauth2Token != nil {
		// Cloud Code API returns wrapped response
		var wrapperResp CloudCodeResponseWrapper
		if err := json.NewDecoder(resp.Body).Decode(&wrapperResp); err != nil {
			return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
		}
		apiResp = wrapperResp.Response
	} else {
		// Standard API returns response directly
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
		}
	}
	if len(apiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in Gemini response")
	}
	candidate := apiResp.Candidates[0]
	if candidate.FinishReason == "SAFETY" {
		return nil, fmt.Errorf("content was filtered due to safety concerns")
	}
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no parts in candidate content")
	}
	var fullText strings.Builder
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			fullText.WriteString(part.Text)
		}
	}
	result := fullText.String()
	if result == "" {
		return nil, fmt.Errorf("empty response from Gemini API")
	}
	cleanedCode := utils.CleanCodeResponse(result)
	var usage *types.Usage
	if apiResp.UsageMetadata != nil {
		usage = &types.Usage{
			PromptTokens:     apiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: apiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      apiResp.UsageMetadata.TotalTokenCount,
		}
		logger.Debugf("Gemini: Extracted token usage - Prompt: %d, Completion: %d, Total: %d",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	} else {
		logger.Warnf("Gemini: No usage metadata in response")
	}
	return &types.CodeGenerationResult{
		Code:  cleanedCode,
		Usage: usage,
	}, nil
}
func (c *GeminiClient) getBaseURL() string {
	// If user explicitly configured a base URL, use it
	if c.config.BaseURL != "" {
		return c.config.BaseURL
	}

	// OAuth users use Cloud Code API
	if c.oauth2Token != nil {
		return cloudcodeBaseURL
	}

	// API key users use standard Gemini API
	return standardGeminiBaseURL
}

func (c *GeminiClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}
	baseURL := c.getBaseURL()
	url := fmt.Sprintf("%s/%s", baseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.oauth2Token != nil {
		logger.Debugf("Gemini: Using OAuth authentication with Cloud Code API (%s)", baseURL)
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.oauth2Token.TokenType, c.oauth2Token.AccessToken))
	} else if c.config.APIKey != "" {
		logger.Debugf("Gemini: Using API key authentication with standard API (%s)", baseURL)
		req.Header.Set("x-goog-api-key", c.config.APIKey)
	} else {
		return nil, fmt.Errorf("Gemini requires OAuth or API key authentication")
	}
	logger.Debugf("Gemini: Making API call to %s", url)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}
func (c *GeminiClient) createOAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.config.ClientID,
		ClientSecret: c.config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: c.config.RedirectURI,
		Scopes:      c.config.Scopes,
	}
}
func (c *GeminiClient) createOAuth2Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  c.config.AccessToken,
		RefreshToken: c.config.RefreshToken,
		Expiry:       c.config.TokenExpiry,
		TokenType:    "Bearer",
	}
}
func (c *GeminiClient) isTokenExpired() bool {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	if c.oauth2Token == nil {
		return true
	}
	return c.oauth2Token.Expiry.IsZero() || c.oauth2Token.Expiry.Before(time.Now().Add(5*time.Minute))
}
func (c *GeminiClient) ensureValidToken(ctx context.Context) error {
	logger.Debugf("Gemini: ensureValidToken called")
	if c.oauth2Config == nil || c.oauth2Token == nil {
		logger.Debugf("Gemini: OAuth not configured (config=%v, token=%v)", c.oauth2Config != nil, c.oauth2Token != nil)
		return nil
	}
	isExpired := c.isTokenExpired()
	logger.Debugf("Gemini: Token expired check: %v", isExpired)
	if !isExpired {
		return nil
	}
	logger.Debugf("Gemini: Acquiring token mutex lock")
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()
	logger.Debugf("Gemini: Token mutex lock acquired")
	if c.oauth2Token != nil && !c.oauth2Token.Expiry.IsZero() && !c.oauth2Token.Expiry.Before(time.Now().Add(5*time.Minute)) {
		logger.Debugf("Gemini: Token already refreshed by another goroutine")
		return nil
	}
	logger.Debugf("Gemini: Refreshing expired OAuth token")
	logger.Debugf("Gemini: Current refresh token: %s...", c.oauth2Token.RefreshToken[:10])
	tokenSource := c.oauth2Config.TokenSource(ctx, c.oauth2Token)
	logger.Debugf("Gemini: Created token source, calling Token()")
	newToken, err := tokenSource.Token()
	logger.Debugf("Gemini: Token() call completed, err=%v", err)
	if err != nil {
		return fmt.Errorf("failed to refresh OAuth token: %w", err)
	}
	c.oauth2Token = newToken
	c.config.AccessToken = newToken.AccessToken
	c.config.RefreshToken = newToken.RefreshToken
	c.config.TokenExpiry = newToken.Expiry
	logger.Debugf("Gemini: OAuth token refreshed successfully, new expiry: %s", newToken.Expiry.Format(time.RFC3339))
	if err := c.persistToken(); err != nil {
		logger.Warnf("Failed to persist updated token to config file: %v. Don't fail the request, token is valid in memory", err)
	}
	return nil
}
func (c *GeminiClient) persistToken() error {
	logger.Debugf("Gemini: Persisting token to config file")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(homeDir, ".mcp-code-api", "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}
	providers, ok := configMap["providers"].(map[string]interface{})
	if !ok {
		providers = make(map[string]interface{})
		configMap["providers"] = providers
	}
	gemini, ok := providers["gemini"].(map[string]interface{})
	if !ok {
		gemini = make(map[string]interface{})
		providers["gemini"] = gemini
	}
	gemini["access_token"] = c.config.AccessToken
	gemini["refresh_token"] = c.config.RefreshToken
	gemini["token_expiry"] = c.config.TokenExpiry.Format(time.RFC3339)
	updatedData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}
	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("failed to write updated config file: %w", err)
	}
	logger.Debugf("Gemini: Token persisted successfully to %s", configPath)
	return nil
}

func (c *GeminiClient) persistProjectID(projectID string) error {
	logger.Debugf("Gemini: Persisting project ID to config file")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(homeDir, ".mcp-code-api", "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}
	providers, ok := configMap["providers"].(map[string]interface{})
	if !ok {
		providers = make(map[string]interface{})
		configMap["providers"] = providers
	}
	gemini, ok := providers["gemini"].(map[string]interface{})
	if !ok {
		gemini = make(map[string]interface{})
		providers["gemini"] = gemini
	}
	gemini["project_id"] = projectID
	c.config.ProjectID = projectID
	updatedData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}
	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("failed to write updated config file: %w", err)
	}
	logger.Debugf("Gemini: Project ID persisted successfully to %s", configPath)
	return nil
}
// buildFullPrompt builds the complete prompt including context and existing content
func (c *GeminiClient) buildFullPrompt(prompt, contextStr, outputFile, detectedLanguage string, contextFiles []string) string {
	var parts []string
	// Add context files if provided
	if len(contextFiles) > 0 {
		filteredContextFiles := c.filterContextFiles(contextFiles, outputFile)
		if len(filteredContextFiles) > 0 {
			contextContent := "Context Files:\n"
			for _, contextFile := range filteredContextFiles {
				if content, err := utils.ReadFileContent(contextFile); err == nil && content != "" {
					contextLang := utils.GetLanguageFromFile(contextFile, nil)
					contextContent += fmt.Sprintf("\nFile: %s\n```%s\n%s\n```\n", contextFile, contextLang, content)
				} else {
					logger.Warnf("Could not read context file %s: %v", contextFile, err)
				}
			}
			parts = append(parts, contextContent)
		}
	}
	// Add additional context if provided
	if contextStr != "" {
		parts = append(parts, fmt.Sprintf("Context: %s", contextStr))
	}
	// Add existing file content if it exists
	if existingContent, err := utils.ReadFileContent(outputFile); err == nil && existingContent != "" {
		parts = append(parts, fmt.Sprintf("Existing file content:\n```%s\n%s\n```\n", detectedLanguage, existingContent))
	}
	// Add the prompt
	parts = append(parts, fmt.Sprintf("Generate %s code for: %s", detectedLanguage, prompt))
	return strings.Join(parts, "\n\n")
}
// filterContextFiles filters out the output file from context files
func (c *GeminiClient) filterContextFiles(contextFiles []string, outputFile string) []string {
	var filtered []string
	for _, file := range contextFiles {
		contextAbs := filepath.Clean(file)
		outputAbs := filepath.Clean(outputFile)
		if contextAbs != outputAbs {
			filtered = append(filtered, file)
		}
	}
	return filtered
}
// Request/Response types for Gemini API
type GenerateContentRequest struct {
	Contents         []Content         `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}
type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}
type Part struct {
	Text string `json:"text"`
}
type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}
type GenerateContentResponse struct {
	Candidates    []Candidate    `json:"candidates"`
	UsageMetadata *UsageMetadata `json:"usageMetadata,omitempty"`
}
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason,omitempty"`
}
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}
// CloudCode request/response wrappers
type CloudCodeRequestWrapper struct {
	Model        string                 `json:"model"`
	Project      string                 `json:"project,omitempty"`
	UserPromptID string                 `json:"user_prompt_id,omitempty"`
	Request      GenerateContentRequest `json:"request"`
}
type CloudCodeResponseWrapper struct {
	Response GenerateContentResponse `json:"response"`
}