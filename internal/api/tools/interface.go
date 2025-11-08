package tools

import (
	"fmt"
)

// ToolFormatter defines the interface for tool formatting conversion
type ToolFormatter interface {
	// FormatRequest formats tools for the specific provider API
	FormatRequest(tools []Tool) (interface{}, error)

	// ParseResponse parses tool calls from the provider's response
	ParseResponse(response interface{}) ([]ToolCall, error)

	// GetContentType returns the content type for this formatter
	GetContentType() string

	// GetFormat returns the tool format type
	GetFormat() ToolFormat

	// SupportsStreaming returns whether this formatter supports streaming
	SupportsStreaming() bool

	// ValidateTools validates the tool definitions for this format
	ValidateTools(tools []Tool) error
}

// ToolFormat represents the different tool formats supported
type ToolFormat string

const (
	ToolFormatOpenAI    ToolFormat = "openai"
	ToolFormatAnthropic ToolFormat = "anthropic"
	ToolFormatXML       ToolFormat = "xml"
	ToolFormatHermes    ToolFormat = "hermes"
	ToolFormatText      ToolFormat = "text"
	ToolFormatGemini    ToolFormat = "gemini"
)

// Tool represents a tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool call from a provider response
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Raw       interface{}            `json:"raw,omitempty"`
}

// ToolParameter represents a tool parameter definition
type ToolParameter struct {
	Type        string                   `json:"type"`
	Description string                   `json:"description,omitempty"`
	Required    bool                     `json:"required,omitempty"`
	Default     interface{}              `json:"default,omitempty"`
	Enum        []interface{}            `json:"enum,omitempty"`
	Properties  map[string]ToolParameter `json:"properties,omitempty"`
	Items       *ToolParameter           `json:"items,omitempty"`
}

// ToolFormatterRegistry manages tool formatters
type ToolFormatterRegistry interface {
	// RegisterFormatter registers a new formatter
	RegisterFormatter(format ToolFormat, formatter ToolFormatter) error

	// GetFormatter returns the formatter for a format
	GetFormatter(format ToolFormat) (ToolFormatter, error)

	// GetSupportedFormats returns all supported formats
	GetSupportedFormats() []ToolFormat

	// FormatForProvider returns the best formatter for a provider type
	FormatForProvider(providerType string) (ToolFormatter, error)
}

// ToolFormatDetector detects the appropriate tool format for a provider
type ToolFormatDetector interface {
	// DetectFormat determines the tool format for a provider
	DetectFormat(providerType, modelID string) ToolFormat

	// IsFormatSupported checks if a format is supported for a provider
	IsFormatSupported(providerType, modelID string, format ToolFormat) bool

	// GetDefaultFormat returns the default format for a provider
	GetDefaultFormat(providerType string) ToolFormat
}

// FormatConversionError represents errors in tool format conversion
type FormatConversionError struct {
	FromFormat ToolFormat `json:"from_format"`
	ToFormat   ToolFormat `json:"to_format"`
	Message    string     `json:"message"`
	Details    string     `json:"details,omitempty"`
}

func (e *FormatConversionError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("Tool format conversion error (%s -> %s): %s - %s", e.FromFormat, e.ToFormat, e.Message, e.Details)
	}
	return fmt.Sprintf("Tool format conversion error (%s -> %s): %s", e.FromFormat, e.ToFormat, e.Message)
}

// ToolValidationError represents tool validation errors
type ToolValidationError struct {
	ToolName string     `json:"tool_name"`
	Format   ToolFormat `json:"format"`
	Message  string     `json:"message"`
	Details  string     `json:"details,omitempty"`
}

func (e *ToolValidationError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("Tool validation error for '%s' (%s format): %s - %s", e.ToolName, e.Format, e.Message, e.Details)
	}
	return fmt.Sprintf("Tool validation error for '%s' (%s format): %s", e.ToolName, e.Format, e.Message)
}

// ToolFormatCapabilities represents capabilities of a tool format
type ToolFormatCapabilities struct {
	Format               ToolFormat `json:"format"`
	SupportsStreaming    bool       `json:"supports_streaming"`
	SupportsComplexTypes bool       `json:"supports_complex_types"`
	SupportsNested       bool       `json:"supports_nested"`
	SupportsArrays       bool       `json:"supports_arrays"`
	SupportsEnums        bool       `json:"supports_enums"`
	MaxToolCount         int        `json:"max_tool_count"`
	MaxParameterCount    int        `json:"max_parameter_count"`
	SupportedProviders   []string   `json:"supported_providers"`
}

// ToolFormatManager manages tool formatting across different providers
type ToolFormatManager interface {
	// FormatTools formats tools for a specific provider
	FormatTools(providerType string, tools []Tool) (interface{}, error)

	// ParseToolCalls parses tool calls from a provider response
	ParseToolCalls(providerType string, response interface{}) ([]ToolCall, error)

	// ConvertFormat converts tools from one format to another
	ConvertFormat(fromFormat, toFormat ToolFormat, tools interface{}) (interface{}, error)

	// GetCapabilities returns capabilities for a format
	GetCapabilities(format ToolFormat) (*ToolFormatCapabilities, error)

	// ValidateForProvider validates tools for a specific provider
	ValidateForProvider(providerType string, tools []Tool) error
}

// ToolSchema represents a JSON schema for tool parameters
type ToolSchema struct {
	Type        string                 `json:"type"`
	Properties  map[string]*ToolSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Description string                 `json:"description,omitempty"`
	Items       *ToolSchema            `json:"items,omitempty"`
	Enum        []interface{}          `json:"enum,omitempty"`
	Default     interface{}            `json:"default,omitempty"`
	MinItems    *int                   `json:"min_items,omitempty"`
	MaxItems    *int                   `json:"max_items,omitempty"`
	MinLength   *int                   `json:"min_length,omitempty"`
	MaxLength   *int                   `json:"max_length,omitempty"`
	Pattern     string                 `json:"pattern,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Minimum     *float64               `json:"minimum,omitempty"`
	Maximum     *float64               `json:"maximum,omitempty"`
}
