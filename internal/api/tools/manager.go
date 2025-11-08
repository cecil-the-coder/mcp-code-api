package tools

import (
	"fmt"
)

// ToolFormatManagerImpl implements ToolFormatManager
type ToolFormatManagerImpl struct {
	registry     *ToolFormatterRegistryImpl
	detector     *ToolFormatDetectorImpl
	capabilities *ToolCapabilitiesRegistry
}

// NewToolFormatManager creates a new tool format manager
func NewToolFormatManager() *ToolFormatManagerImpl {
	registry := NewToolFormatterRegistry()
	detector := NewToolFormatDetector(registry)
	capabilities := NewToolCapabilitiesRegistry()

	return &ToolFormatManagerImpl{
		registry:     registry,
		detector:     detector,
		capabilities: capabilities,
	}
}

// FormatTools formats tools for a specific provider
func (m *ToolFormatManagerImpl) FormatTools(providerType string, tools []Tool) (interface{}, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	// Get the formatter for this provider
	formatter, err := m.registry.FormatForProvider(providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get formatter for provider %s: %w", providerType, err)
	}

	// Validate tools for this format
	if err := formatter.ValidateTools(tools); err != nil {
		return nil, fmt.Errorf("tool validation failed for provider %s: %w", providerType, err)
	}

	// Format the tools
	return formatter.FormatRequest(tools)
}

// ParseToolCalls parses tool calls from a provider response
func (m *ToolFormatManagerImpl) ParseToolCalls(providerType string, response interface{}) ([]ToolCall, error) {
	if response == nil {
		return []ToolCall{}, nil
	}

	// Get the formatter for this provider
	formatter, err := m.registry.FormatForProvider(providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get formatter for provider %s: %w", providerType, err)
	}

	// Parse the response
	return formatter.ParseResponse(response)
}

// ConvertFormat converts tools from one format to another
func (m *ToolFormatManagerImpl) ConvertFormat(fromFormat, toFormat ToolFormat, tools interface{}) (interface{}, error) {
	// Get source formatter
	sourceFormatter, err := m.registry.GetFormatter(fromFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to get source formatter: %w", err)
	}

	// Get target formatter
	targetFormatter, err := m.registry.GetFormatter(toFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to get target formatter: %w", err)
	}

	// Convert tools to generic format first
	var genericTools []Tool

	switch v := tools.(type) {
	case []Tool:
		genericTools = v
	case []interface{}:
		for _, tool := range v {
			if toolMap, ok := tool.(map[string]interface{}); ok {
				// Parse from source format
				toolCalls, err := sourceFormatter.ParseResponse(map[string]interface{}{
					"tools": []map[string]interface{}{toolMap},
				})
				if err != nil {
					return nil, fmt.Errorf("failed to parse tool from source format: %w", err)
				}
				// Convert to Tool - this is a simplified conversion
				// In practice, you'd need a more sophisticated conversion
				if len(toolCalls) > 0 {
					genericTools = append(genericTools, Tool{
						Name:        toolCalls[0].Name,
						Description: "converted tool",
						InputSchema: toolCalls[0].Arguments,
					})
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported tools type: %T", tools)
	}

	// Format with target formatter
	return targetFormatter.FormatRequest(genericTools)
}

// GetCapabilities returns capabilities for a format
func (m *ToolFormatManagerImpl) GetCapabilities(format ToolFormat) (*ToolFormatCapabilities, error) {
	return m.capabilities.GetCapabilities(format)
}

// ValidateForProvider validates tools for a specific provider
func (m *ToolFormatManagerImpl) ValidateForProvider(providerType string, tools []Tool) error {
	if len(tools) == 0 {
		return nil
	}

	// Get the formatter for this provider
	formatter, err := m.registry.FormatForProvider(providerType)
	if err != nil {
		return fmt.Errorf("failed to get formatter for provider %s: %w", providerType, err)
	}

	// Get capabilities for this format
	capabilities, err := m.GetCapabilities(formatter.GetFormat())
	if err != nil {
		return fmt.Errorf("failed to get capabilities: %w", err)
	}

	// Check tool count
	if len(tools) > capabilities.MaxToolCount {
		return fmt.Errorf("too many tools (%d), max allowed: %d", len(tools), capabilities.MaxToolCount)
	}

	// Validate each tool
	for _, tool := range tools {
		// Check parameter count
		paramCount := countParameters(tool.InputSchema)
		if paramCount > capabilities.MaxParameterCount {
			return fmt.Errorf("tool '%s' has too many parameters (%d), max allowed: %d", tool.Name, paramCount, capabilities.MaxParameterCount)
		}

		// Validate format-specific constraints
		if err := validateToolForFormat(tool, formatter.GetFormat(), capabilities); err != nil {
			return fmt.Errorf("tool '%s' validation failed: %w", tool.Name, err)
		}
	}

	return formatter.ValidateTools(tools)
}

// RegisterFormatter registers a new formatter
func (m *ToolFormatManagerImpl) RegisterFormatter(format ToolFormat, formatter ToolFormatter) error {
	return m.registry.RegisterFormatter(format, formatter)
}

// GetFormatter returns the formatter for a format
func (m *ToolFormatManagerImpl) GetFormatter(format ToolFormat) (ToolFormatter, error) {
	return m.registry.GetFormatter(format)
}

// GetSupportedFormats returns all supported formats
func (m *ToolFormatManagerImpl) GetSupportedFormats() []ToolFormat {
	return m.registry.GetSupportedFormats()
}

// DetectFormat detects the format for a provider
func (m *ToolFormatManagerImpl) DetectFormat(providerType, modelID string) ToolFormat {
	return m.detector.DetectFormat(providerType, modelID)
}

// IsFormatSupported checks if a format is supported
func (m *ToolFormatManagerImpl) IsFormatSupported(providerType, modelID string, format ToolFormat) bool {
	return m.detector.IsFormatSupported(providerType, modelID, format)
}

// Helper functions

func countParameters(schema map[string]interface{}) int {
	count := 0
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for _, prop := range properties {
			count++
			// Recursively count nested parameters
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propType, ok := propMap["type"].(string); ok && propType == "object" {
					if nestedProps, ok := propMap["properties"].(map[string]interface{}); ok {
						count += countParameters(map[string]interface{}{"properties": nestedProps})
					}
				}
			}
		}
	}
	return count
}

func validateToolForFormat(tool Tool, format ToolFormat, capabilities *ToolFormatCapabilities) error {
	switch format {
	case ToolFormatAnthropic:
		// Anthropic-specific validations
		if len(tool.Name) > 64 {
			return fmt.Errorf("tool name too long for Anthropic (max 64 chars)")
		}
		if len(tool.Description) > 512 {
			return fmt.Errorf("tool description too long for Anthropic (max 512 chars)")
		}
	case ToolFormatOpenAI:
		// OpenAI-specific validations
		if len(tool.Name) > 128 {
			return fmt.Errorf("tool name too long for OpenAI (max 128 chars)")
		}
	case ToolFormatGemini:
		// Gemini-specific validations
		if len(tool.Name) > 64 {
			return fmt.Errorf("tool name too long for Gemini (max 64 chars)")
		}
	}

	// Check for unsupported features
	if !capabilities.SupportsArrays {
		if hasArrayType(tool.InputSchema) {
			return fmt.Errorf("tool '%s' uses arrays which are not supported by %s format", tool.Name, format)
		}
	}

	if !capabilities.SupportsEnums {
		if hasEnumType(tool.InputSchema) {
			return fmt.Errorf("tool '%s' uses enums which are not supported by %s format", tool.Name, format)
		}
	}

	return nil
}

func hasArrayType(schema map[string]interface{}) bool {
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for _, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propType, ok := propMap["type"].(string); ok && propType == "array" {
					return true
				}
				// Check nested properties
				if propType, ok := propMap["type"].(string); ok && propType == "object" {
					if nestedProps, ok := propMap["properties"].(map[string]interface{}); ok {
						if hasArrayType(map[string]interface{}{"properties": nestedProps}) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func hasEnumType(schema map[string]interface{}) bool {
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for _, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if _, ok := propMap["enum"]; ok {
					return true
				}
				// Check nested properties
				if propType, ok := propMap["type"].(string); ok && propType == "object" {
					if nestedProps, ok := propMap["properties"].(map[string]interface{}); ok {
						if hasEnumType(map[string]interface{}{"properties": nestedProps}) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
