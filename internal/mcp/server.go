package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/provider"
	"github.com/cecil-the-coder/mcp-code-api/internal/api/router"
	"github.com/cecil-the-coder/mcp-code-api/internal/config"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
)

// Request represents an MCP request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents an MCP response
type Response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      interface{}    `json:"id"`
	Result  interface{}    `json:"result,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

// ErrorResponse represents an MCP error
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Content type is imported from types package
// Use types.Content directly

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Server represents an MCP server
type Server struct {
	config *config.Config
	router *router.EnhancedRouter
	reader *bufio.Reader
	writer *bufio.Writer
}

// NewServer creates a new MCP server instance
func NewServer(cfg *config.Config) *Server {
	// Create provider factory
	factory := provider.NewProviderFactory()
	provider.InitializeDefaultProviders(factory)

	// Create enhanced router
	enhancedRouter := router.NewEnhancedRouter(cfg, factory)

	s := &Server{
		config: cfg,
		router: enhancedRouter,
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}
	return s
}

// GetRouter returns the server's router (for metrics access)
func (s *Server) GetRouter() *router.EnhancedRouter {
	return s.router
}

// Start starts an MCP server
func (s *Server) Start(ctx context.Context) error {
	// Initialize router with providers
	if err := s.router.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}
	
	logger.Info("MCP Server entering message loop...")
	// Start message loop
	return s.messageLoop(ctx)
}

// messageLoop handles the main message loop for MCP communication
func (s *Server) messageLoop(ctx context.Context) error {
	logger.Debugf("Message loop started, waiting for requests...")
	decoder := json.NewDecoder(s.reader)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var request Request
			if err := decoder.Decode(&request); err != nil {
				if err == io.EOF {
					return nil
				}
				logger.Debugf("Failed to decode request: %v", err)
				return fmt.Errorf("failed to decode request: %w", err)
			}

			logger.Debugf("Received request: method=%s, id=%v", request.Method, request.ID)

			// Handle the request
			response, err := s.handleRequest(ctx, &request)
			if err != nil {
				logger.Debugf("Request handling failed: %v", err)
				// Send error response
				s.sendErrorResponse(&request, err)
				continue
			}

			// If no response (e.g., notification), skip sending
			if response == nil {
				logger.Debugf("No response needed for request (notification)")
				continue
			}

			logger.Debugf("Sending success response for request ID %v", request.ID)

			// Send the response
			if err := s.sendResponse(response); err != nil {
				logger.Debugf("Failed to send response: %v", err)
				return fmt.Errorf("failed to send response: %w", err)
			}

			logger.Debugf("Response sent successfully for request ID %v", request.ID)
		}
	}
}

// handleRequest handles different types of MCP requests
func (s *Server) handleRequest(ctx context.Context, request *Request) (*Response, error) {
	switch request.Method {
	case "initialize":
		return s.handleInitialize(ctx, request)
	case "initialized", "notifications/initialized":
		// Notification - no response needed
		logger.Debugf("Received initialized notification")
		return nil, nil
	case "tools/list":
		return s.handleListTools(ctx, request)
	case "tools/call":
		return s.handleCallTool(ctx, request)
	default:
		logger.Debugf("Unknown method received: %s", request.Method)
		return nil, fmt.Errorf("unknown method: %s", request.Method)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(ctx context.Context, request *Request) (*Response, error) {
	return &Response{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":        s.config.Server.Name,
				"version":     s.config.Server.Version,
				"description": s.config.Server.Description,
			},
			"instructions": buildSystemInstructions(),
		},
	}, nil
}

// handleListTools handles the tools/list request
func (s *Server) handleListTools(ctx context.Context, request *Request) (*Response, error) {
	tools := s.getTools()
	return &Response{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}, nil
}

// handleCallTool handles the tools/call request
func (s *Server) handleCallTool(ctx context.Context, request *Request) (*Response, error) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := s.unmarshalParams(request.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse tool call parameters: %w", err)
	}

	switch params.Name {
	case "write":
		return s.handleWriteTool(ctx, request, &params.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

// getTools returns a list of available tools
func (s *Server) getTools() []Tool {
	writeTool := Tool{
		Name: "write",
		Description: `🚨 USE THIS TOOL FOR AI-GENERATED CODE 🚨

⭐ WHEN TO USE THIS TOOL:
- Creating new files with AI-generated code → USE THIS TOOL
- Generating code for existing files → USE THIS TOOL
- Complex code modifications requiring AI assistance → USE THIS TOOL
- Any code generation task → USE THIS TOOL

⚠️  WHEN YOU CAN USE NATIVE TOOLS:
- Simple manual edits (typo fixes, single-line changes)
- Direct file operations you're performing yourself
- Reading files or searching code

This tool provides AI-powered code generation with:
- Multiple provider fallback (Cerebras, Anthropic, OpenRouter)
- Automatic syntax validation and error correction
- Smart diff generation
- Undo support

✨ FEATURES:
- Creates new files automatically
- Modifies existing files with smart diffs
- Shows visually enhanced git-style diffs with emoji indicators (✅ additions, ❌ removals, 🔍 changes)
- Supports context_files for better code understanding
- Handles all programming languages
- Automatic syntax validation (Python, Go, JavaScript, TypeScript)
- Auto-fix for Go code formatting
- CONTEXT SAVER: Use write_only: true to get minimal response and save 80-95% context
- UNDO SUPPORT: Automatically backs up files before modification - use restore_previous: true to undo

🎯 USAGE GUIDE:
- Creating files with AI: Use 'write' with file_path + detailed prompt
- Generating code: Use 'write' with file_path + prompt + optional context_files
- Complex modifications: Use 'write' for AI assistance with code changes
- Save context: Add write_only: true to skip full diff (saves 80-95% tokens)
- Undo AI changes: Use restore_previous: true with file_path
- Manual edits: You can still use native Edit/Write tools for simple changes

💡 BEST PRACTICE: Prefer this tool for code generation tasks, especially new files. Use native tools only for trivial manual edits.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "REQUIRED: Absolute path to the file (e.g., '/Users/username/project/file.py'). This tool will create or modify the file at this location.",
				},
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "REQUIRED: A comprehensive plan dump that MUST include: 1) EXACT method signatures and parameters, 2) SPECIFIC database queries/SQL if needed, 3) DETAILED error handling requirements, 4) PRECISE integration points with context files, 5) EXACT constructor parameters and data flow, 6) SPECIFIC return types and data structures. Be extremely detailed - this is your blueprint for implementation.",
				},
				"context_files": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type":        "string",
						"description": "OPTIONAL: Array of file paths to include as context for the model. These files will be read and their content included to help understand the codebase structure and patterns.",
					},
				},
				"write_only": map[string]interface{}{
					"type":        "boolean",
					"description": "OPTIONAL: When true, returns a minimal success message instead of the full diff. This significantly reduces context usage in the conversation. Set to true when you don't need to see the changes. Default: false",
				},
				"validate": map[string]interface{}{
					"type":        "boolean",
					"description": "OPTIONAL: When true, validates code syntax before writing using language-specific validators (gofmt, node, python, tsc). Automatically enabled when write_only is true. If validation fails and auto-fix is available (e.g., gofmt for Go), attempts to fix automatically. Otherwise returns error message for the AI to fix. Default: false (true if write_only is true)",
				},
				"restore_previous": map[string]interface{}{
					"type":        "boolean",
					"description": "OPTIONAL: When true, restores the previous version of the file from the in-memory backup. The backup is created automatically each time a file is modified. This allows you to undo the last change made to a file. Note: Only works for files modified in the current session, and the backup is cleared after restore. When using this parameter, you only need to provide file_path (prompt is not required). Default: false",
				},
			},
			"required": []string{"file_path"},
		},
	}

	return []Tool{writeTool}
}

// sendResponse sends a response to the client
func (s *Server) sendResponse(response *Response) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if _, err := s.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	if _, err := s.writer.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return s.writer.Flush()
}

// sendErrorResponse sends an error response to the client
func (s *Server) sendErrorResponse(request *Request, err error) {
	// JSON-RPC 2.0 spec: If request ID is null/missing, don't send error response
	// (this indicates a notification or malformed request)
	if request.ID == nil {
		// Silently ignore - per JSON-RPC 2.0 spec section 5
		logger.Debugf("Skipping error response for request with nil ID: %v", err)
		return
	}

	logger.Debugf("Sending error response for request ID %v: %v", request.ID, err)

	errorResponse := &Response{
		JSONRPC: "2.0",
		ID:      request.ID,
		Error: &ErrorResponse{
			Code:    -1,
			Message: err.Error(),
		},
	}

	data, marshalErr := json.Marshal(errorResponse)
	if marshalErr != nil {
		logger.Debugf("Failed to marshal error response: %v", marshalErr)
		return
	}

	if _, writeErr := s.writer.Write(data); writeErr != nil {
		logger.Debugf("Failed to write error response: %v", writeErr)
		return
	}

	if _, writeErr := s.writer.Write([]byte("\n")); writeErr != nil {
		logger.Debugf("Failed to write newline after error response: %v", writeErr)
		return
	}

	if flushErr := s.writer.Flush(); flushErr != nil {
		logger.Debugf("Failed to flush error response: %v", flushErr)
	}
}

// unmarshalParams safely unmarshals parameters
func (s *Server) unmarshalParams(params interface{}, target interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}

	return nil
}

// buildSystemInstructions builds the system instructions for the MCP server
func buildSystemInstructions() string {
	return `🚨 AI CODE GENERATION TOOL AVAILABLE 🚨

This environment provides an MCP tool called 'write' for AI-powered code generation.

⭐ USE 'write' TOOL FOR:
- Creating new files with AI-generated code
- Generating complex code or entire functions/classes
- Code modifications that require AI assistance
- Any task where you need to generate substantial code

✨ TOOL FEATURES:
- Multi-provider fallback (Cerebras → Anthropic → OpenRouter)
- Automatic syntax validation and auto-fix
- Smart diff generation with emoji indicators (✅ additions, ❌ removals, 🔍 changes)
- Context-aware code generation using context_files
- Automatic file backups with undo support (restore_previous: true)
- Token-efficient mode (write_only: true saves 80-95% context)

🎯 USAGE EXAMPLES:
- New file: write(file_path="/path/file.go", prompt="Create a user service with CRUD operations")
- Edit file: write(file_path="/path/file.go", prompt="Add error handling to SaveUser method", context_files=[...])
- Undo change: write(file_path="/path/file.go", restore_previous=true)

⚠️  YOU CAN STILL USE NATIVE TOOLS FOR:
- Simple manual edits (fixing typos, changing single values)
- Reading or searching files
- Direct file operations you perform yourself

💡 BEST PRACTICE: Prefer the 'write' tool for code generation, especially for new files or complex changes. Reserve native Edit/Write tools for trivial manual modifications only.`
}