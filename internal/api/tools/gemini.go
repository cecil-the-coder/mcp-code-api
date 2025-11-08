package tools

import (
	"fmt"
)

// GeminiToolFormatter implements ToolFormatter for Gemini format (similar to Anthropic)
type GeminiToolFormatter struct {
	format ToolFormat
}

// NewGeminiToolFormatter creates a new Gemini tool formatter
func NewGeminiToolFormatter() *GeminiToolFormatter {
	return &GeminiToolFormatter{
		format: ToolFormatGemini,
	}
}

// FormatRequest formats tools for Gemini API
func (f *GeminiToolFormatter) FormatRequest(tools []Tool) (interface{}, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	// Convert to Gemini format (similar to Anthropic)
	geminiTools := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		geminiTool, err := f.convertToolToGemini(tool)
		if err != nil {
			return nil, &FormatConversionError{
				FromFormat: f.format,
				ToFormat:   ToolFormatGemini,
				Message:    fmt.Sprintf("Failed to convert tool '%s'", tool.Name),
				Details:    err.Error(),
			}
		}
		geminiTools[i] = geminiTool
	}

	return map[string]interface{}{
		"tools": map[string]interface{}{
			"function_declarations": geminiTools,
		},
	}, nil
}

// ParseResponse parses tool calls from Gemini API response
func (f *GeminiToolFormatter) ParseResponse(response interface{}) ([]ToolCall, error) {
	// Convert response to map for easier handling
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, &FormatConversionError{
			FromFormat: ToolFormatGemini,
			ToFormat:   f.format,
			Message:    "Invalid response format - expected map",
		}
	}

	// Extract candidates
	candidates, ok := respMap["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return []ToolCall{}, nil
	}

	// Get first candidate
	candidate, ok := candidates[0].(map[string]interface{})
	if !ok {
		return nil, &FormatConversionError{
			FromFormat: ToolFormatGemini,
			ToFormat:   f.format,
			Message:    "Invalid candidate format",
		}
	}

	// Extract content
	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return []ToolCall{}, nil
	}

	// Extract parts
	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return []ToolCall{}, nil
	}

	var toolCalls []ToolCall
	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is a function call part
		if _, ok := partMap["functionCall"]; !ok {
			continue
		}

		parsedCall, err := f.parseGeminiToolCall(partMap)
		if err != nil {
			return nil, err
		}
		toolCalls = append(toolCalls, parsedCall)
	}

	return toolCalls, nil
}

// GetContentType returns the content type for Gemini format
func (f *GeminiToolFormatter) GetContentType() string {
	return "application/json"
}

// GetFormat returns the tool format type
func (f *GeminiToolFormatter) GetFormat() ToolFormat {
	return ToolFormatGemini
}

// SupportsStreaming returns whether this formatter supports streaming
func (f *GeminiToolFormatter) SupportsStreaming() bool {
	return true
}

// ValidateTools validates the tool definitions for Gemini format
func (f *GeminiToolFormatter) ValidateTools(tools []Tool) error {
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

func (f *GeminiToolFormatter) convertToolToGemini(tool Tool) (map[string]interface{}, error) {
	schema, err := f.convertSchemaToGemini(tool.InputSchema)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":        tool.Name,
		"description": tool.Description,
		"parameters":  schema,
	}, nil
}

func (f *GeminiToolFormatter) convertSchemaToGemini(schema map[string]interface{}) (map[string]interface{}, error) {
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

func (f *GeminiToolFormatter) parseGeminiToolCall(toolCall map[string]interface{}) (ToolCall, error) {
	functionCall, ok := toolCall["functionCall"].(map[string]interface{})
	if !ok {
		return ToolCall{}, &FormatConversionError{
			FromFormat: ToolFormatGemini,
			ToFormat:   f.format,
			Message:    "Invalid functionCall format in Gemini tool call",
		}
	}

	name, _ := functionCall["name"].(string)
	args, _ := functionCall["args"].(map[string]interface{})

	if args == nil {
		args = make(map[string]interface{})
	}

	return ToolCall{
		ID:        "", // Gemini doesn't use IDs for tool calls
		Name:      name,
		Arguments: args,
		Metadata:  toolCall,
		Raw:       toolCall,
	}, nil
}

func (f *GeminiToolFormatter) validateSchema(schema map[string]interface{}) error {
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

func (f *GeminiToolFormatter) validatePropertySchema(propName string, schema map[string]interface{}) error {
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
