package tools

import (
	"fmt"
)

// AnthropicToolFormatter implements ToolFormatter for Anthropic format
type AnthropicToolFormatter struct {
	format ToolFormat
}

// NewAnthropicToolFormatter creates a new Anthropic tool formatter
func NewAnthropicToolFormatter() *AnthropicToolFormatter {
	return &AnthropicToolFormatter{
		format: ToolFormatAnthropic,
	}
}

// FormatRequest formats tools for Anthropic API
func (f *AnthropicToolFormatter) FormatRequest(tools []Tool) (interface{}, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	// Convert to Anthropic format
	anthropicTools := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		anthropicTool, err := f.convertToolToAnthropic(tool)
		if err != nil {
			return nil, &FormatConversionError{
				FromFormat: f.format,
				ToFormat:   ToolFormatAnthropic,
				Message:    fmt.Sprintf("Failed to convert tool '%s'", tool.Name),
				Details:    err.Error(),
			}
		}
		anthropicTools[i] = anthropicTool
	}

	return map[string]interface{}{
		"tools": anthropicTools,
		"tool_choice": map[string]interface{}{
			"type": "auto",
		},
	}, nil
}

// ParseResponse parses tool calls from Anthropic API response
func (f *AnthropicToolFormatter) ParseResponse(response interface{}) ([]ToolCall, error) {
	// Convert response to map for easier handling
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, &FormatConversionError{
			FromFormat: ToolFormatAnthropic,
			ToFormat:   f.format,
			Message:    "Invalid response format - expected map",
		}
	}

	// Extract content
	content, ok := respMap["content"].([]interface{})
	if !ok || len(content) == 0 {
		return []ToolCall{}, nil
	}

	var toolCalls []ToolCall
	for _, contentItem := range content {
		contentMap, ok := contentItem.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is a tool use content item
		if contentType, ok := contentMap["type"].(string); !ok || contentType != "tool_use" {
			continue
		}

		parsedCall, err := f.parseAnthropicToolCall(contentMap)
		if err != nil {
			return nil, err
		}
		toolCalls = append(toolCalls, parsedCall)
	}

	return toolCalls, nil
}

// GetContentType returns the content type for Anthropic format
func (f *AnthropicToolFormatter) GetContentType() string {
	return "application/json"
}

// GetFormat returns the tool format type
func (f *AnthropicToolFormatter) GetFormat() ToolFormat {
	return ToolFormatAnthropic
}

// SupportsStreaming returns whether this formatter supports streaming
func (f *AnthropicToolFormatter) SupportsStreaming() bool {
	return true
}

// ValidateTools validates the tool definitions for Anthropic format
func (f *AnthropicToolFormatter) ValidateTools(tools []Tool) error {
	for i, tool := range tools {
		if tool.Name == "" {
			return &ToolValidationError{
				ToolName: tool.Name,
				Format:   f.format,
				Message:  fmt.Sprintf("Tool at index %d has no name", i),
			}
		}

		if tool.Description == "" {
			return &ToolValidationError{
				ToolName: tool.Name,
				Format:   f.format,
				Message:  "Tool description is required",
			}
		}

		if tool.InputSchema == nil {
			return &ToolValidationError{
				ToolName: tool.Name,
				Format:   f.format,
				Message:  "Tool input schema is required",
			}
		}

		// Validate schema format
		if err := f.validateSchema(tool.InputSchema); err != nil {
			return &ToolValidationError{
				ToolName: tool.Name,
				Format:   f.format,
				Message:  "Invalid input schema",
				Details:  err.Error(),
			}
		}
	}
	return nil
}

// Helper methods

func (f *AnthropicToolFormatter) convertToolToAnthropic(tool Tool) (map[string]interface{}, error) {
	schema, err := f.convertSchemaToAnthropic(tool.InputSchema)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":         tool.Name,
		"description":  tool.Description,
		"input_schema": schema,
		"metadata":     tool.Metadata,
	}, nil
}

func (f *AnthropicToolFormatter) convertSchemaToAnthropic(schema map[string]interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"type": "object",
	}

	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		result["properties"] = properties
	}

	if required, ok := schema["required"].([]string); ok {
		result["required"] = required
	} else if required, ok := schema["required"].([]interface{}); ok {
		// Convert to []string
		strArray := make([]string, len(required))
		for i, r := range required {
			if s, ok := r.(string); ok {
				strArray[i] = s
			}
		}
		result["required"] = strArray
	}

	return result, nil
}

func (f *AnthropicToolFormatter) parseAnthropicToolCall(toolCall map[string]interface{}) (ToolCall, error) {
	id, _ := toolCall["id"].(string)
	name, _ := toolCall["name"].(string)
	input, _ := toolCall["input"].(map[string]interface{})

	if input == nil {
		input = make(map[string]interface{})
	}

	return ToolCall{
		ID:        id,
		Name:      name,
		Arguments: input,
		Metadata:  toolCall,
		Raw:       toolCall,
	}, nil
}

func (f *AnthropicToolFormatter) validateSchema(schema map[string]interface{}) error {
	if schemaType, ok := schema["type"].(string); ok && schemaType != "object" {
		return fmt.Errorf("schema type must be 'object', got '%s'", schemaType)
	}

	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for propName, propSchema := range properties {
			if propMap, ok := propSchema.(map[string]interface{}); ok {
				if err := f.validatePropertySchema(propName, propMap); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (f *AnthropicToolFormatter) validatePropertySchema(propName string, schema map[string]interface{}) error {
	if propType, ok := schema["type"].(string); ok {
		switch propType {
		case "string", "number", "integer", "boolean", "array", "object":
			// Valid types
		default:
			return fmt.Errorf("invalid property type '%s' for property '%s'", propType, propName)
		}
	}

	return nil
}
