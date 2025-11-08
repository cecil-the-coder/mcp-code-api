package tools

import (
	"fmt"
	"sync"
)

// ToolFormatterRegistryImpl implements ToolFormatterRegistry
type ToolFormatterRegistryImpl struct {
	formatters map[ToolFormat]ToolFormatter
	mutex      sync.RWMutex
}

// NewToolFormatterRegistry creates a new tool formatter registry
func NewToolFormatterRegistry() *ToolFormatterRegistryImpl {
	registry := &ToolFormatterRegistryImpl{
		formatters: make(map[ToolFormat]ToolFormatter),
	}

	// Register default formatters
	registry.registerDefaults()

	return registry
}

// RegisterFormatter registers a new formatter
func (r *ToolFormatterRegistryImpl) RegisterFormatter(format ToolFormat, formatter ToolFormatter) error {
	if format == "" {
		return fmt.Errorf("format cannot be empty")
	}
	if formatter == nil {
		return fmt.Errorf("formatter cannot be nil")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.formatters[format] = formatter
	return nil
}

// GetFormatter returns the formatter for a format
func (r *ToolFormatterRegistryImpl) GetFormatter(format ToolFormat) (ToolFormatter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	formatter, exists := r.formatters[format]
	if !exists {
		return nil, fmt.Errorf("no formatter registered for format: %s", format)
	}

	return formatter, nil
}

// GetSupportedFormats returns all supported formats
func (r *ToolFormatterRegistryImpl) GetSupportedFormats() []ToolFormat {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	formats := make([]ToolFormat, 0, len(r.formatters))
	for format := range r.formatters {
		formats = append(formats, format)
	}

	return formats
}

// FormatForProvider returns the best formatter for a provider type
func (r *ToolFormatterRegistryImpl) FormatForProvider(providerType string) (ToolFormatter, error) {
	var format ToolFormat

	switch providerType {
	case "openai":
		format = ToolFormatOpenAI
	case "anthropic":
		format = ToolFormatAnthropic
	case "gemini":
		format = ToolFormatGemini
	case "qwen":
		// Qwen can use OpenAI format or custom
		format = ToolFormatOpenAI
	case "openrouter":
		format = ToolFormatOpenAI
	case "cerebras":
		format = ToolFormatOpenAI
	default:
		// Default to OpenAI format
		format = ToolFormatOpenAI
	}

	return r.GetFormatter(format)
}

// registerDefaults registers the default tool formatters
func (r *ToolFormatterRegistryImpl) registerDefaults() {
	r.formatters[ToolFormatOpenAI] = NewOpenAIToolFormatter()
	r.formatters[ToolFormatAnthropic] = NewAnthropicToolFormatter()
	r.formatters[ToolFormatGemini] = NewGeminiToolFormatter()
}

// ToolFormatDetectorImpl implements ToolFormatDetector
type ToolFormatDetectorImpl struct {
	registry *ToolFormatterRegistryImpl
}

// NewToolFormatDetector creates a new tool format detector
func NewToolFormatDetector(registry *ToolFormatterRegistryImpl) *ToolFormatDetectorImpl {
	return &ToolFormatDetectorImpl{
		registry: registry,
	}
}

// DetectFormat determines the tool format for a provider
func (d *ToolFormatDetectorImpl) DetectFormat(providerType, modelID string) ToolFormat {
	switch providerType {
	case "openai":
		return ToolFormatOpenAI
	case "anthropic":
		return ToolFormatAnthropic
	case "gemini":
		return ToolFormatGemini
	case "qwen":
		// Check model-specific format preferences
		if isQwenOpenAICompatible(modelID) {
			return ToolFormatOpenAI
		}
		return ToolFormatOpenAI // Default to OpenAI
	case "openrouter":
		return ToolFormatOpenAI
	case "cerebras":
		return ToolFormatOpenAI
	default:
		// Default to OpenAI format
		return ToolFormatOpenAI
	}
}

// IsFormatSupported checks if a format is supported for a provider
func (d *ToolFormatDetectorImpl) IsFormatSupported(providerType, modelID string, format ToolFormat) bool {
	switch providerType {
	case "openai":
		return format == ToolFormatOpenAI
	case "anthropic":
		return format == ToolFormatAnthropic
	case "gemini":
		return format == ToolFormatGemini
	case "qwen":
		// Qwen supports multiple formats
		switch format {
		case ToolFormatOpenAI, ToolFormatAnthropic, ToolFormatGemini:
			return true
		default:
			return false
		}
	case "openrouter", "cerebras":
		// These are OpenAI compatible
		return format == ToolFormatOpenAI
	default:
		// Default to OpenAI format support
		return format == ToolFormatOpenAI
	}
}

// GetDefaultFormat returns the default format for a provider
func (d *ToolFormatDetectorImpl) GetDefaultFormat(providerType string) ToolFormat {
	return d.DetectFormat(providerType, "")
}

// Helper functions

func isQwenOpenAICompatible(modelID string) bool {
	// Most modern Qwen models are OpenAI compatible
	return true
}

// ToolCapabilitiesRegistry manages tool format capabilities
type ToolCapabilitiesRegistry struct {
	capabilities map[ToolFormat]*ToolFormatCapabilities
	mutex        sync.RWMutex
}

// NewToolCapabilitiesRegistry creates a new capabilities registry
func NewToolCapabilitiesRegistry() *ToolCapabilitiesRegistry {
	registry := &ToolCapabilitiesRegistry{
		capabilities: make(map[ToolFormat]*ToolFormatCapabilities),
	}

	// Register default capabilities
	registry.registerDefaults()

	return registry
}

// GetCapabilities returns capabilities for a format
func (r *ToolCapabilitiesRegistry) GetCapabilities(format ToolFormat) (*ToolFormatCapabilities, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	caps, exists := r.capabilities[format]
	if !exists {
		return nil, fmt.Errorf("no capabilities registered for format: %s", format)
	}

	return caps, nil
}

// registerDefaults registers default capabilities
func (r *ToolCapabilitiesRegistry) registerDefaults() {
	r.capabilities[ToolFormatOpenAI] = &ToolFormatCapabilities{
		Format:               ToolFormatOpenAI,
		SupportsStreaming:    true,
		SupportsComplexTypes: true,
		SupportsNested:       true,
		SupportsArrays:       true,
		SupportsEnums:        true,
		MaxToolCount:         128,
		MaxParameterCount:    256,
		SupportedProviders:   []string{"openai", "openrouter", "cerebras", "qwen"},
	}

	r.capabilities[ToolFormatAnthropic] = &ToolFormatCapabilities{
		Format:               ToolFormatAnthropic,
		SupportsStreaming:    true,
		SupportsComplexTypes: true,
		SupportsNested:       true,
		SupportsArrays:       true,
		SupportsEnums:        false,
		MaxToolCount:         32,
		MaxParameterCount:    64,
		SupportedProviders:   []string{"anthropic"},
	}

	r.capabilities[ToolFormatGemini] = &ToolFormatCapabilities{
		Format:               ToolFormatGemini,
		SupportsStreaming:    true,
		SupportsComplexTypes: true,
		SupportsNested:       true,
		SupportsArrays:       true,
		SupportsEnums:        true,
		MaxToolCount:         64,
		MaxParameterCount:    128,
		SupportedProviders:   []string{"gemini"},
	}
}

// IsFeatureSupported checks if a feature is supported by a format
func (r *ToolCapabilitiesRegistry) IsFeatureSupported(format ToolFormat, feature string) bool {
	caps, err := r.GetCapabilities(format)
	if err != nil {
		return false
	}

	switch feature {
	case "streaming":
		return caps.SupportsStreaming
	case "complex_types":
		return caps.SupportsComplexTypes
	case "nested":
		return caps.SupportsNested
	case "arrays":
		return caps.SupportsArrays
	case "enums":
		return caps.SupportsEnums
	default:
		return false
	}
}
