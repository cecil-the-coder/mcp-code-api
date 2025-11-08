package tools

import (
	"encoding/json"
	"fmt"
)

// OpenAIToolFormatter implements ToolFormatter for OpenAI format
type OpenAIToolFormatter struct {
	format ToolFormat
}

// NewOpenAIToolFormatter creates a new OpenAI tool formatter
func NewOpenAIToolFormatter() *OpenAIToolFormatter {
	return &OpenAIToolFormatter{
		format: ToolFormatOpenAI,
	}
}

// FormatRequest formats tools for OpenAI API
func (f *OpenAIToolFormatter) FormatRequest(tools []Tool) (interface{}, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	// Convert to OpenAI format
	openaiTools := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		openaiTool, err := f.convertToolToOpenAI(tool)
		if err != nil {
			return nil, &FormatConversionError{
				FromFormat: f.format,
				ToFormat:   ToolFormatOpenAI,
				Message:    fmt.Sprintf("Failed to convert tool '%s'", tool.Name),
				Details:    err.Error(),
			}
		}
		openaiTools[i] = openaiTool
	}

	return map[string]interface{}{
		"type":      "function",
		"functions": openaiTools,
	}, nil
}

// ParseResponse parses tool calls from OpenAI API response
func (f *OpenAIToolFormatter) ParseResponse(response interface{}) ([]ToolCall, error) {
	// Convert response to map for easier handling
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, &FormatConversionError{
			FromFormat: ToolFormatOpenAI,
			ToFormat:   f.format,
			Message:    "Invalid response format - expected map",
		}
	}

	// Extract choices
	choices, ok := respMap["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return []ToolCall{}, nil
	}

	// Get first choice
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, &FormatConversionError{
			FromFormat: ToolFormatOpenAI,
			ToFormat:   f.format,
			Message:    "Invalid choice format",
		}
	}

	// Extract message
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return nil, &FormatConversionError{
			FromFormat: ToolFormatOpenAI,
			ToFormat:   f.format,
			Message:    "Invalid message format",
		}
	}

	// Extract tool calls
	toolCallsRaw, ok := message["tool_calls"].([]interface{})
	if !ok || len(toolCallsRaw) == 0 {
		return []ToolCall{}, nil
	}

	toolCalls := make([]ToolCall, len(toolCallsRaw))
	for i, toolCallRaw := range toolCallsRaw {
		toolCall, ok := toolCallRaw.(map[string]interface{})
		if !ok {
			return nil, &FormatConversionError{
				FromFormat: ToolFormatOpenAI,
				ToFormat:   f.format,
				Message:    fmt.Sprintf("Invalid tool call format at index %d", i),
			}
		}

		parsedCall, err := f.parseOpenAIToolCall(toolCall)
		if err != nil {
			return nil, err
		}
		toolCalls[i] = parsedCall
	}

	return toolCalls, nil
}

// GetContentType returns the content type for OpenAI format
func (f *OpenAIToolFormatter) GetContentType() string {
	return "application/json"
}

// GetFormat returns the tool format type
func (f *OpenAIToolFormatter) GetFormat() ToolFormat {
	return ToolFormatOpenAI
}

// SupportsStreaming returns whether this formatter supports streaming
func (f *OpenAIToolFormatter) SupportsStreaming() bool {
	return true
}

// ValidateTools validates the tool definitions for OpenAI format
func (f *OpenAIToolFormatter) ValidateTools(tools []Tool) error {
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

func (f *OpenAIToolFormatter) convertToolToOpenAI(tool Tool) (map[string]interface{}, error) {
	schema, err := f.convertSchemaToOpenAI(tool.InputSchema)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":        tool.Name,
		"description": tool.Description,
		"parameters":  schema,
		"metadata":    tool.Metadata,
	}, nil
}

func (f *OpenAIToolFormatter) convertSchemaToOpenAI(schema map[string]interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"type": "object",
	}

	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		result["properties"] = properties
	}

	if required, ok := schema["required"].([]interface{}); ok {
		result["required"] = required
	} else if required, ok := schema["required"].([]string); ok {
		interfaceArray := make([]interface{}, len(required))
		for i, r := range required {
			interfaceArray[i] = r
		}
		result["required"] = interfaceArray
	}

	if additionalProperties, ok := schema["additionalProperties"]; ok {
		result["additionalProperties"] = additionalProperties
	}

	return result, nil
}

func (f *OpenAIToolFormatter) parseOpenAIToolCall(toolCall map[string]interface{}) (ToolCall, error) {
	id, _ := toolCall["id"].(string)
	function, ok := toolCall["function"].(map[string]interface{})
	if !ok {
		return ToolCall{}, &FormatConversionError{
			FromFormat: ToolFormatOpenAI,
			ToFormat:   f.format,
			Message:    "Invalid function format in tool call",
		}
	}

	name, _ := function["name"].(string)
	argumentsRaw, _ := function["arguments"].(string)

	var arguments map[string]interface{}
	if argumentsRaw != "" {
		if err := json.Unmarshal([]byte(argumentsRaw), &arguments); err != nil {
			// If JSON parsing fails, keep as raw string
			arguments = map[string]interface{}{"raw": argumentsRaw}
		}
	}

	return ToolCall{
		ID:        id,
		Name:      name,
		Arguments: arguments,
		Metadata:  toolCall,
		Raw:       toolCall,
	}, nil
}

func (f *OpenAIToolFormatter) validateSchema(schema map[string]interface{}) error {
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

func (f *OpenAIToolFormatter) validatePropertySchema(propName string, schema map[string]interface{}) error {
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
