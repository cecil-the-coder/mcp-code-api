package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cecil-the-coder/mcp-code-api/internal/formatting"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
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

	// Check for restore_previous flag to undo last write
	restorePrevious := extractBoolArg(arguments, "restore_previous")
	if restorePrevious {
		return s.handleRestorePrevious(request, filePath)
	}

	// Check if file exists to determine operation type
	existingContent, err := utils.ReadFileContent(filePath)
	isEdit := err == nil && existingContent != ""

	// Store backup of existing content before modification
	if isEdit && existingContent != "" {
		globalBackupStore.StoreBackup(filePath, existingContent)
		logger.Debugf("Stored backup for file: %s (%d bytes)", filePath, len(existingContent))
	}

	logger.Debug("=== FILE OPERATION DEBUG ===")
	logger.Debugf("File path: %s", filePath)
	logger.Debugf("File exists: %v", isEdit)
	logger.Debugf("Existing content length: %d", len(existingContent))
	logger.Debugf("Validation enabled: %v", validate)
	logger.Debug("============================")

	// Collect validation warnings
	var warnings []string
	var warningsMutex sync.Mutex

	warningCallback := func(providerName, message string) {
		warningsMutex.Lock()
		defer warningsMutex.Unlock()
		warnings = append(warnings, message)
		logger.Infof("[VALIDATION] %s", message)
	}

	// Route API call to appropriate provider with validation retry and failover
	result, err := s.router.GenerateCodeWithValidation(ctx, prompt, filePath, contextFiles, validate, warningCallback)
	if err != nil {
		// Check if we have warnings to include
		var errorMsg string
		if len(warnings) > 0 {
			errorMsg = fmt.Sprintf("%s\n\nValidation warnings:\n%s", err.Error(), strings.Join(warnings, "\n"))
		} else {
			errorMsg = err.Error()
		}
		return s.createErrorResponse(request, fmt.Errorf("%s", errorMsg))
	}

	// Write the result to the file
	if err := utils.WriteFileContent(filePath, result); err != nil {
		return s.createErrorResponse(request, fmt.Errorf("failed to write file: %w", err))
	}

	// If write_only is enabled, return minimal response to save context
	if writeOnly {
		fileName := filepath.Base(filePath)
		operation := "created"
		if isEdit {
			operation = "updated"
		}

		lineCount := strings.Count(result, "\n") + 1

		// Build response text
		responseText := fmt.Sprintf("✅ Successfully %s: %s\n📝 File: %s\n💾 Lines: %d",
			operation, fileName, filePath, lineCount)

		// Add warnings if any
		if len(warnings) > 0 {
			responseText += "\n\n⚠️ Validation warnings:\n" + strings.Join(warnings, "\n")
		}

		responseText += "\n\n(Full diff omitted to save context - use write_only: false to see changes)"

		responseContent := []Content{{
			Type: "text",
			Text: responseText,
		}}

		logger.Debug("=== MCP RESPONSE DEBUG (WRITE_ONLY MODE) ===")
		logger.Debugf("IDE Source: %s", ideSource)
		logger.Debug("Response type: Minimal success message")
		logger.Debugf("Operation: %s", operation)
		logger.Debugf("Warnings count: %d", len(warnings))
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

	// Add warnings as first content item if any
	if len(warnings) > 0 {
		warningText := "⚠️ **Validation Warnings:**\n\n" + strings.Join(warnings, "\n")
		responseContent = append(responseContent, Content{
			Type: "text",
			Text: warningText,
		})
	}

	if isEdit && existingContent != "" {
		// Clean the existing content too for consistent comparison
		cleanExistingContent := utils.CleanCodeResponse(existingContent)
		editResponse := formatting.FormatEditResponse(fileName, cleanExistingContent, result, filePath)
		if editResponse != nil {
			responseContent = append(responseContent, *editResponse)
		}
	} else if !isEdit {
		createResponse := formatting.FormatCreateResponse(fileName, result, filePath)
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
	logger.Debugf("Warnings count: %d", len(warnings))
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

// handleRestorePrevious restores the previous version of a file from backup
func (s *Server) handleRestorePrevious(request *Request, filePath string) (*Response, error) {
	logger.Debugf("Attempting to restore previous version of: %s", filePath)

	// Check if backup exists
	if !globalBackupStore.HasBackup(filePath) {
		return s.createErrorResponse(request, fmt.Errorf("no backup found for file: %s\nBackup is only available for files that were modified in this session.", filePath))
	}

	// Get backed up content
	backupContent, err := globalBackupStore.GetBackup(filePath)
	if err != nil {
		return s.createErrorResponse(request, fmt.Errorf("failed to get backup: %w", err))
	}

	// Write backup content to file
	if err := utils.WriteFileContent(filePath, backupContent); err != nil {
		return s.createErrorResponse(request, fmt.Errorf("failed to restore file: %w", err))
	}

	// Clear the backup after successful restore
	globalBackupStore.ClearBackup(filePath)

	fileName := filepath.Base(filePath)
	responseText := fmt.Sprintf("✅ Successfully restored previous version of: %s\n📁 File: %s\n💾 Restored %d bytes\n\n⚠️  The backup has been cleared - you cannot undo this restore.",
		fileName, filePath, len(backupContent))

	logger.Infof("Restored previous version of: %s", filePath)

	return &Response{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"content": []Content{{
				Type: "text",
				Text: responseText,
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
