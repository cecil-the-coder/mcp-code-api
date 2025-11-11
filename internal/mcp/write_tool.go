package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cecil-the-coder/mcp-code-api/internal/api"
	"github.com/cecil-the-coder/mcp-code-api/internal/formatting"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
	"github.com/cecil-the-coder/mcp-code-api/internal/validation"
)

// handleWriteTool handles the write tool request
func (s *Server) handleWriteTool(ctx context.Context, request *Request, arguments *map[string]interface{}) (*Response, error) {
	// Get IDE identification from environment variable
	ideSource := os.Getenv("CEREBRAS_MCP_IDE")
	if ideSource == "" {
		ideSource = "unknown"
	}

	logger.Debug("=== MCP REQUEST DEBUG ===")
	logger.Debugf("IDE Source: %s", ideSource)
	logger.Debug("Tool called: write")
	logger.Debugf("Arguments: %s", toString(arguments))
	logger.Debug("========================")

	// Extract arguments
	filePath, err := extractStringArg(arguments, "file_path")
	if err != nil {
		return nil, fmt.Errorf("file_path is required: %w", err)
	}

	prompt, err := extractStringArg(arguments, "prompt")
	if err != nil {
		return nil, fmt.Errorf("prompt is required: %w", err)
	}

	contextFiles, err := extractStringSliceArg(arguments, "context_files")
	if err != nil {
		return nil, fmt.Errorf("context_files must be an array of strings: %w", err)
	}

	// Check for write_only flag to reduce context usage
	writeOnly := extractBoolArg(arguments, "write_only")

	// Check for validate flag - defaults to true if write_only is true
	validate := extractBoolArg(arguments, "validate")
	if !validate && writeOnly {
		// If validate wasn't explicitly set and write_only is true, enable validation
		if _, exists := (*arguments)["validate"]; !exists {
			validate = true
		}
	}

	// Check if file exists to determine operation type
	existingContent, err := utils.ReadFileContent(filePath)
	isEdit := err == nil && existingContent != ""

	logger.Debug("=== FILE OPERATION DEBUG ===")
	logger.Debugf("File path: %s", filePath)
	logger.Debugf("File exists: %v", isEdit)
	logger.Debugf("Existing content length: %d", len(existingContent))
	logger.Debug("============================")

	// Route API call to appropriate provider to generate/modify code with context files
	result, err := api.RouteAPICall(ctx, s.config, prompt, "", filePath, "", contextFiles)
	if err != nil {
		return s.createErrorResponse(request, err)
	}

	// Clean the AI response to remove markdown formatting
	cleanResult := utils.CleanCodeResponse(result)

	// Validate code syntax if requested
	if validate {
		language := validation.DetectLanguage(filePath)

		logger.Debug("=== VALIDATION DEBUG ===")
		logger.Debugf("Language detected: %s", language)
		logger.Debugf("Validation enabled: %v", validate)
		logger.Debug("========================")

		if language != validation.LanguageUnknown {
			validator := language.GetValidator()
			validationResult, err := validator.Validate(cleanResult, filePath)
			if err != nil {
				logger.Debugf("Validation error: %v", err)
				return s.createErrorResponse(request, fmt.Errorf("validation error: %w", err))
			}

			if !validationResult.Valid {
				logger.Debugf("Validation failed with %d errors", len(validationResult.Errors))

				// Try auto-fix if available
				if validator.CanAutoFix() {
					logger.Debug("Attempting auto-fix...")
					fixedCode, err := validator.AutoFix(cleanResult)
					if err == nil {
						logger.Debug("Auto-fix successful")
						cleanResult = fixedCode

						// Validate the fixed code
						validationResult, err = validator.Validate(cleanResult, filePath)
						if err != nil || !validationResult.Valid {
							logger.Debug("Auto-fix validation failed")
							errorMsg := validation.FormatValidationErrors(validationResult.Errors, language)
							return s.createValidationErrorResponse(request, errorMsg)
						}
					} else {
						logger.Debugf("Auto-fix failed: %v", err)
						errorMsg := validation.FormatValidationErrors(validationResult.Errors, language)
						return s.createValidationErrorResponse(request, errorMsg)
					}
				} else {
					// No auto-fix available, return error to AI
					errorMsg := validation.FormatValidationErrors(validationResult.Errors, language)
					return s.createValidationErrorResponse(request, errorMsg)
				}
			} else {
				logger.Debug("Validation passed")
			}
		} else {
			logger.Debug("Validation skipped for unknown language")
		}
	}

	// Write the cleaned result to the file
	if err := utils.WriteFileContent(filePath, cleanResult); err != nil {
		return s.createErrorResponse(request, fmt.Errorf("failed to write file: %w", err))
	}

	// If write_only is enabled, return minimal response to save context
	if writeOnly {
		fileName := filepath.Base(filePath)
		operation := "created"
		if isEdit {
			operation = "updated"
		}

		lineCount := strings.Count(cleanResult, "\n") + 1
		responseContent := []Content{{
			Type: "text",
			Text: fmt.Sprintf("✅ Successfully %s: %s\n📝 File: %s\n💾 Lines: %d\n\n(Full diff omitted to save context - use write_only: false to see changes)",
				operation, fileName, filePath, lineCount),
		}}

		logger.Debug("=== MCP RESPONSE DEBUG (WRITE_ONLY MODE) ===")
		logger.Debugf("IDE Source: %s", ideSource)
		logger.Debug("Response type: Minimal success message")
		logger.Debugf("Operation: %s", operation)
		logger.Debug("===========================================")

		return &Response{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result: map[string]interface{}{
				"content": responseContent,
			},
		}, nil
	}

	// Format the response based on operation type (normal mode with full diff)
	var responseContent []Content
	fileName := filepath.Base(filePath)

	if isEdit && existingContent != "" {
		// Clean the existing content too for consistent comparison
		cleanExistingContent := utils.CleanCodeResponse(existingContent)
		editResponse := formatting.FormatEditResponse(fileName, cleanExistingContent, cleanResult, filePath)
		if editResponse != nil {
			responseContent = append(responseContent, *editResponse)
		}
	} else if !isEdit {
		createResponse := formatting.FormatCreateResponse(fileName, cleanResult, filePath)
		responseContent = append(responseContent, *createResponse)
	}

	response := &Response{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"content": responseContent,
		},
	}

	// Log the full response for debugging
	logger.Debug("=== MCP RESPONSE DEBUG ===")
	logger.Debugf("IDE Source: %s", ideSource)
	logger.Debug("Response type: Standard text diff")
	logger.Debugf("Number of content items: %d", len(responseContent))
	logger.Debugf("Response structure: %s", toString(response.Result))
	logger.Debug("=========================")

	return response, nil
}

// extractStringArg extracts a string argument from the arguments map
func extractStringArg(arguments *map[string]interface{}, key string) (string, error) {
	if arguments == nil {
		return "", fmt.Errorf("arguments map is nil")
	}

	value, exists := (*arguments)[key]
	if !exists {
		return "", fmt.Errorf("missing required argument: %s", key)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("argument %s must be a string, got %T", key, value)
	}

	return strValue, nil
}

// extractStringSliceArg extracts a string slice argument from the arguments map
func extractStringSliceArg(arguments *map[string]interface{}, key string) ([]string, error) {
	if arguments == nil {
		return nil, fmt.Errorf("arguments map is nil")
	}

	value, exists := (*arguments)[key]
	if !exists {
		// Optional argument, return empty slice if not present
		return []string{}, nil
	}

	switch v := value.(type) {
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			strItem, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("context_files[%d] must be a string, got %T", i, item)
			}
			result[i] = strItem
		}
		return result, nil
	case []string:
		return v, nil
	case nil:
		return []string{}, nil
	default:
		return nil, fmt.Errorf("argument %s must be an array of strings, got %T", key, value)
	}
}

// extractBoolArg extracts a boolean argument from the arguments map
func extractBoolArg(arguments *map[string]interface{}, key string) bool {
	if arguments == nil {
		return false
	}

	value, exists := (*arguments)[key]
	if !exists {
		return false
	}

	boolValue, ok := value.(bool)
	if !ok {
		return false
	}

	return boolValue
}

// createErrorResponse creates an error response
func (s *Server) createErrorResponse(request *Request, err error) (*Response, error) {
	// Get IDE identification from environment variable (in case of error)
	ideSource := os.Getenv("CEREBRAS_MCP_IDE")
	if ideSource == "" {
		ideSource = "unknown"
	}

	logger.Debug("=== MCP ERROR DEBUG ===")
	logger.Debugf("IDE Source: %s", ideSource)
	logger.Debugf("Error occurred: %v", err)
	logger.Debug("=======================")

	// Return a standard text error if something goes wrong
	return &Response{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"content": []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error in mcp-code-api server: %v", err),
			}},
		},
	}, nil
}

// createValidationErrorResponse creates a validation error response
func (s *Server) createValidationErrorResponse(request *Request, errorMsg string) (*Response, error) {
	ideSource := os.Getenv("CEREBRAS_MCP_IDE")
	if ideSource == "" {
		ideSource = "unknown"
	}

	logger.Debug("=== VALIDATION ERROR DEBUG ===")
	logger.Debugf("IDE Source: %s", ideSource)
	logger.Debugf("Validation failed: %s", errorMsg)
	logger.Debug("==============================")

	// Return validation error to AI so it can fix the code
	return &Response{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"content": []Content{{
				Type: "text",
				Text: errorMsg,
			}},
		},
	}, nil
}

// toString converts any value to a string representation
func toString(v interface{}) string {
	if v == nil {
		return "null"
	}

	if data, err := json.MarshalIndent(v, "", "  "); err == nil {
		return string(data)
	}

	return fmt.Sprintf("%v", v)
}
