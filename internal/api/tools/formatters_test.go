package tools

import (
	"fmt"
	"testing"
)

// TestToolFormatterRegistry tests the tool formatter registry
func TestToolFormatterRegistry(t *testing.T) {
	registry := NewToolFormatterRegistry()

	t.Run("Default Formatters Registered", func(t *testing.T) {
		supportedFormats := registry.GetSupportedFormats()
		if len(supportedFormats) == 0 {
			t.Error("Expected at least one formatter to be registered")
		}

		// Check that major formats are registered
		formats := make(map[ToolFormat]bool)
		for _, format := range supportedFormats {
			formats[format] = true
		}

		if !formats[ToolFormatOpenAI] {
			t.Error("OpenAI formatter not registered")
		}
		if !formats[ToolFormatAnthropic] {
			t.Error("Anthropic formatter not registered")
		}
		if !formats[ToolFormatGemini] {
			t.Error("Gemini formatter not registered")
		}
	})

	t.Run("Format for Provider", func(t *testing.T) {
		// Test known providers
		providers := map[string]ToolFormat{
			"openai":     ToolFormatOpenAI,
			"anthropic":  ToolFormatAnthropic,
			"gemini":     ToolFormatGemini,
			"qwen":       ToolFormatOpenAI,
			"openrouter": ToolFormatOpenAI,
			"cerebras":   ToolFormatOpenAI,
		}

		for provider, expectedFormat := range providers {
			formatter, err := registry.FormatForProvider(provider)
			if err != nil {
				t.Errorf("Failed to get formatter for %s: %v", provider, err)
				continue
			}

			if formatter.GetFormat() != expectedFormat {
				t.Errorf("Expected format %s for %s, got %s", expectedFormat, provider, formatter.GetFormat())
			}
		}

		// Test unknown provider (should default to OpenAI)
		formatter, err := registry.FormatForProvider("unknown")
		if err != nil {
			t.Errorf("Failed to get formatter for unknown provider: %v", err)
		} else if formatter.GetFormat() != ToolFormatOpenAI {
			t.Errorf("Expected default OpenAI format, got %s", formatter.GetFormat())
		}
	})
}

// TestToolFormatDetector tests the tool format detector
func TestToolFormatDetector(t *testing.T) {
	registry := NewToolFormatterRegistry()
	detector := NewToolFormatDetector(registry)

	t.Run("Detect Format", func(t *testing.T) {
		testCases := []struct {
			provider       string
			modelID        string
			expectedFormat ToolFormat
		}{
			{"openai", "", ToolFormatOpenAI},
			{"anthropic", "", ToolFormatAnthropic},
			{"gemini", "gemini-1.5-pro", ToolFormatGemini},
			{"qwen", "qwen-max", ToolFormatOpenAI},
			{"openrouter", "Mistral-7B", ToolFormatOpenAI},
			{"unknown", "", ToolFormatOpenAI}, // defaults to OpenAI
		}

		for _, tc := range testCases {
			format := detector.DetectFormat(tc.provider, tc.modelID)
			if format != tc.expectedFormat {
				t.Errorf("Expected format %s for %s/%s, got %s", tc.expectedFormat, tc.provider, tc.modelID, format)
			}
		}
	})

	t.Run("Is Format Supported", func(t *testing.T) {
		testCases := []struct {
			provider string
			modelID  string
			format   ToolFormat
			expected bool
		}{
			{"openai", "", ToolFormatOpenAI, true},
			{"openai", "", ToolFormatAnthropic, false},
			{"anthropic", "", ToolFormatAnthropic, true},
			{"anthropic", "", ToolFormatOpenAI, false},
			{"gemini", "", ToolFormatGemini, true},
			{"gemini", "", ToolFormatAnthropic, false}, // Gemini uses its own format
			{"qwen", "", ToolFormatOpenAI, true},
			{"qwen", "", ToolFormatAnthropic, true}, // Qwen supports multiple formats
		}

		for _, tc := range testCases {
			supported := detector.IsFormatSupported(tc.provider, tc.modelID, tc.format)
			if supported != tc.expected {
				t.Errorf("Expected %s to be %s for %s/%s, got %v",
					tc.format,
					map[bool]string{true: "supported", false: "not supported"}[tc.expected],
					tc.provider, tc.modelID, supported)
			}
		}
	})
}

// TestToolFormatting tests actual tool formatting functionality
func TestToolFormatting(t *testing.T) {
	manager := NewToolFormatManager()

	// Create test tools
	testTools := []Tool{
		{
			Name:        "weather_tool",
			Description: "Get weather information for a location",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The location to get weather for",
					},
					"units": map[string]interface{}{
						"type":        "string",
						"enum":        []interface{}{"celsius", "fahrenheit"},
						"description": "Temperature units",
					},
				},
				"required": []string{"location"},
			},
		},
		{
			Name:        "calculator",
			Description: "Perform mathematical calculations",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "Mathematical expression to evaluate",
					},
				},
				"required": []string{"expression"},
			},
		},
	}

	t.Run("Format Tools for OpenAI", func(t *testing.T) {
		formatted, err := manager.FormatTools("openai", testTools)
		if err != nil {
			t.Fatalf("Failed to format tools for OpenAI: %v", err)
		}

		if formatted == nil {
			t.Error("Formatted tools should not be nil")
		}

		formatMap, ok := formatted.(map[string]interface{})
		if !ok {
			t.Error("Formatted result should be a map")
			return
		}

		// Check OpenAI-specific structure
		if formatMap["type"] != "function" {
			t.Errorf("Expected type 'function', got %v", formatMap["type"])
		}

		// functions can be either []interface{} or []map[string]interface{}
		var functions []interface{}
		switch v := formatMap["functions"].(type) {
		case []interface{}:
			functions = v
		case []map[string]interface{}:
			functions = make([]interface{}, len(v))
			for i, item := range v {
				functions[i] = item
			}
		default:
			t.Errorf("Expected functions array, got type %T: %v", formatMap["functions"], formatMap["functions"])
			return
		}

		if len(functions) != len(testTools) {
			t.Errorf("Expected %d functions, got %d", len(testTools), len(functions))
		}
	})

	t.Run("Format Tools for Anthropic", func(t *testing.T) {
		formatted, err := manager.FormatTools("anthropic", testTools)
		if err != nil {
			t.Fatalf("Failed to format tools for Anthropic: %v", err)
		}

		formatMap, ok := formatted.(map[string]interface{})
		if !ok {
			t.Error("Formatted result should be a map")
			return
		}

		// Check Anthropic-specific structure
		// tools can be either []interface{} or []map[string]interface{}
		var toolsArray []interface{}
		switch v := formatMap["tools"].(type) {
		case []interface{}:
			toolsArray = v
		case []map[string]interface{}:
			toolsArray = make([]interface{}, len(v))
			for i, item := range v {
				toolsArray[i] = item
			}
		default:
			t.Errorf("Expected tools array, got type %T: %v", formatMap["tools"], formatMap["tools"])
			return
		}

		if len(toolsArray) != len(testTools) {
			t.Errorf("Expected %d tools, got %d", len(testTools), len(toolsArray))
		}

		// Check tool_choice
		toolChoice, ok := formatMap["tool_choice"].(map[string]interface{})
		if !ok {
			t.Error("Expected tool_choice map")
			return
		}

		if toolChoice["type"] != "auto" {
			t.Errorf("Expected tool_choice type 'auto', got %v", toolChoice["type"])
		}
	})

	t.Run("Format Tools for Gemini", func(t *testing.T) {
		formatted, err := manager.FormatTools("gemini", testTools)
		if err != nil {
			t.Fatalf("Failed to format tools for Gemini: %v", err)
		}

		formatMap, ok := formatted.(map[string]interface{})
		if !ok {
			t.Error("Formatted result should be a map")
			return
		}

		// Check Gemini-specific structure
		toolsContainer, ok := formatMap["tools"].(map[string]interface{})
		if !ok {
			t.Error("Expected tools container map")
			return
		}

		// function_declarations can be either []interface{} or []map[string]interface{}
		var functions []interface{}
		switch v := toolsContainer["function_declarations"].(type) {
		case []interface{}:
			functions = v
		case []map[string]interface{}:
			functions = make([]interface{}, len(v))
			for i, item := range v {
				functions[i] = item
			}
		default:
			t.Errorf("Expected function_declarations array, got type %T: %v", toolsContainer["function_declarations"], toolsContainer["function_declarations"])
			return
		}

		if len(functions) != len(testTools) {
			t.Errorf("Expected %d function declarations, got %d", len(testTools), len(functions))
		}
	})

	t.Run("Format Empty Tools", func(t *testing.T) {
		formatted, err := manager.FormatTools("openai", []Tool{})
		if err != nil {
			t.Errorf("Failed to format empty tools: %v", err)
		}
		if formatted != nil {
			t.Error("Empty tools should return nil")
		}
	})
}

// TestToolValidation tests tool validation functionality
func TestToolValidation(t *testing.T) {
	manager := NewToolFormatManager()

	t.Run("Valid Tools", func(t *testing.T) {
		validTools := []Tool{
			{
				Name:        "test_tool",
				Description: "A valid test tool",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param": map[string]interface{}{
							"type":        "string",
							"description": "A parameter",
						},
					},
					"required": []string{"param"},
				},
			},
		}

		err := manager.ValidateForProvider("openai", validTools)
		if err != nil {
			t.Errorf("Valid tools should not fail validation: %v", err)
		}
	})

	t.Run("Invalid Tools - Missing Name", func(t *testing.T) {
		invalidTools := []Tool{
			{
				Name:        "",
				Description: "Tool with no name",
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		}

		err := manager.ValidateForProvider("openai", invalidTools)
		if err == nil {
			t.Error("Expected validation error for tool with no name")
		}
	})

	t.Run("Invalid Tools - Missing Description", func(t *testing.T) {
		invalidTools := []Tool{
			{
				Name:        "test_tool",
				Description: "",
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		}

		err := manager.ValidateForProvider("anthropic", invalidTools)
		if err == nil {
			t.Error("Expected validation error for tool with no description")
		}
	})

	t.Run("Invalid Tools - Missing Schema", func(t *testing.T) {
		invalidTools := []Tool{
			{
				Name:        "test_tool",
				Description: "Tool with no schema",
				InputSchema: nil,
			},
		}

		err := manager.ValidateForProvider("gemini", invalidTools)
		if err == nil {
			t.Error("Expected validation error for tool with no input schema")
		}
	})
}

// TestToolCapabilities tests tool format capabilities
func TestToolCapabilities(t *testing.T) {
	manager := NewToolFormatManager()

	t.Run("Get Capabilities", func(t *testing.T) {
		formats := []ToolFormat{ToolFormatOpenAI, ToolFormatAnthropic, ToolFormatGemini}

		for _, format := range formats {
			caps, err := manager.GetCapabilities(format)
			if err != nil {
				t.Errorf("Failed to get capabilities for %s: %v", format, err)
				continue
			}

			if caps.Format != format {
				t.Errorf("Expected format %s, got %s", format, caps.Format)
			}

			if caps.MaxToolCount <= 0 {
				t.Errorf("Expected positive max tool count for %s, got %d", format, caps.MaxToolCount)
			}

			if caps.MaxParameterCount <= 0 {
				t.Errorf("Expected positive max parameter count for %s, got %d", format, caps.MaxParameterCount)
			}

			if len(caps.SupportedProviders) == 0 {
				t.Errorf("Expected at least one supported provider for %s", format)
			}
		}
	})

	t.Run("Get Unknown Capabilities", func(t *testing.T) {
		_, err := manager.GetCapabilities("unknown_format")
		if err == nil {
			t.Error("Expected error for unknown format")
		}
	})
}

// BenchmarkToolFormatting benchmarks tool formatting operations
func BenchmarkToolFormatting(b *testing.B) {
	manager := NewToolFormatManager()
	tools := make([]Tool, 10)

	for i := 0; i < 10; i++ {
		tools[i] = Tool{
			Name:        fmt.Sprintf("tool_%d", i),
			Description: fmt.Sprintf("Test tool number %d", i),
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param": map[string]interface{}{
						"type":        "string",
						"description": fmt.Sprintf("Parameter for tool %d", i),
					},
				},
				"required": []string{"param"},
			},
		}
	}

	providers := []string{"openai", "anthropic", "gemini"}

	for _, provider := range providers {
		b.Run(provider, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = manager.FormatTools(provider, tools)
			}
		})
	}
}
