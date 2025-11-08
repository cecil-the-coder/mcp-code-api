package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cecil-the-coder/mcp-code-api/internal/api"
	"github.com/cecil-the-coder/mcp-code-api/internal/formatting"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
)

// handleWriteTool handles the write tool request
func (s *Server) handleWriteTool(ctx context.Context, arguments *map[string]interface{}) (*Response, error) {
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
		return s.createErrorResponse(err)
	}

	// Clean the AI response to remove markdown formatting
	cleanResult := utils.CleanCodeResponse(result)

	// Write the cleaned result to the file
	if err := utils.WriteFileContent(filePath, cleanResult); err != nil {
		return s.createErrorResponse(fmt.Errorf("failed to write file: %w", err))
	}

	// Format the response based on operation type
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

// createErrorResponse creates an error response
func (s *Server) createErrorResponse(err error) (*Response, error) {
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
		Result: map[string]interface{}{
			"content": []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error in mcp-code-api server: %v", err),
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
