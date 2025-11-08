package validation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

// JavaScriptValidator validates JavaScript code syntax
type JavaScriptValidator struct{}

// Validate checks JavaScript syntax using Node.js
func (v *JavaScriptValidator) Validate(code string, filePath string) (*ValidationResult, error) {
	// Check if node is available using tool cache
	toolCache := GetToolCache()
	if !toolCache.IsAvailable("node") {
		// No Node.js available, skip validation
		return &ValidationResult{Valid: true, Errors: nil}, nil
	}

	// Create a temporary file with the code
	tmpFile, err := os.CreateTemp("", "validate-*.js")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(code); err != nil {
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}

	tmpFile.Close()

	// Create context with timeout (5s should be plenty for single file validation)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run Node.js syntax check with timeout
	cmd := exec.CommandContext(ctx, "node", "--check", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("javascript validation timeout exceeded (5s)")
	}

	if err == nil {
		return &ValidationResult{Valid: true, Errors: nil}, nil
	}

	// Parse errors from output
	errors := v.parseErrors(string(output))
	return &ValidationResult{Valid: false, Errors: errors}, nil
}

// CanAutoFix returns false - we don't auto-fix JavaScript yet
func (v *JavaScriptValidator) CanAutoFix() bool {
	return false
}

// AutoFix is not implemented for JavaScript
func (v *JavaScriptValidator) AutoFix(code string) (string, error) {
	return "", fmt.Errorf("auto-fix not supported for JavaScript")
}

// parseErrors parses Node.js error messages
func (v *JavaScriptValidator) parseErrors(output string) []ValidationError {
	var errors []ValidationError

	// Node.js error format: file.js:line:col
	// Try to parse complete error with message
	re := regexp.MustCompile(`(\S+):(\d+):(\d+)\n([\s\S]*?)(?:SyntaxError|Error):\s*(.+?)(?:\n|$)`)
	matches := re.FindAllStringSubmatch(output, -1)

	if len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 5 {
				lineNum, _ := strconv.Atoi(match[2])
				colNum, _ := strconv.Atoi(match[3])
				message := match[5]
				errors = append(errors, ValidationError{
					Line:    lineNum,
					Column:  colNum,
					Message: message,
				})
			}
		}
	}

	// Fallback: try simpler pattern
	if len(errors) == 0 {
		re = regexp.MustCompile(`:(\d+):(\d+)`)
		matches = re.FindAllStringSubmatch(output, -1)

		if len(matches) > 0 {
			for _, match := range matches {
				if len(match) > 2 {
					lineNum, _ := strconv.Atoi(match[1])
					colNum, _ := strconv.Atoi(match[2])
					errors = append(errors, ValidationError{
						Line:    lineNum,
						Column:  colNum,
						Message: "Syntax error",
					})
				}
			}
		} else {
			// Generic error
			errors = append(errors, ValidationError{
				Line:    0,
				Message: output,
			})
		}
	}

	return errors
}

// TypeScriptValidator validates TypeScript code syntax
type TypeScriptValidator struct{}

// Validate checks TypeScript syntax using tsc
func (v *TypeScriptValidator) Validate(code string, filePath string) (*ValidationResult, error) {
	// Check if tsc is available using tool cache
	toolCache := GetToolCache()
	if !toolCache.IsAvailable("tsc") {
		// No TypeScript available, fall back to JavaScript validation
		jsValidator := &JavaScriptValidator{}
		return jsValidator.Validate(code, filePath)
	}

	// Create a temporary file with the code
	tmpFile, err := os.CreateTemp("", "validate-*.ts")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(code); err != nil {
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}

	tmpFile.Close()

	// Create context with timeout (10s for TypeScript since it can be slower)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run TypeScript syntax check with timeout
	cmd := exec.CommandContext(ctx, "tsc", "--noEmit", "--skipLibCheck", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("typescript validation timeout exceeded (10s)")
	}

	if err == nil {
		return &ValidationResult{Valid: true, Errors: nil}, nil
	}

	// Parse errors from output
	errors := v.parseErrors(string(output))
	return &ValidationResult{Valid: false, Errors: errors}, nil
}

// CanAutoFix returns false
func (v *TypeScriptValidator) CanAutoFix() bool {
	return false
}

// AutoFix is not implemented
func (v *TypeScriptValidator) AutoFix(code string) (string, error) {
	return "", fmt.Errorf("auto-fix not supported for TypeScript")
}

// parseErrors parses TypeScript error messages
func (v *TypeScriptValidator) parseErrors(output string) []ValidationError {
	var errors []ValidationError

	// TypeScript error format: file.ts(line,col): error TS####: message
	// Parse complete error with message
	re := regexp.MustCompile(`(\S+)\((\d+),(\d+)\):\s*error\s*\w*:\s*(.+)`)
	matches := re.FindAllStringSubmatch(output, -1)

	if len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 4 {
				lineNum, _ := strconv.Atoi(match[2])
				colNum, _ := strconv.Atoi(match[3])
				message := match[4]
				errors = append(errors, ValidationError{
					Line:    lineNum,
					Column:  colNum,
					Message: message,
				})
			}
		}
	}

	// Fallback: try simpler pattern
	if len(errors) == 0 {
		re = regexp.MustCompile(`\((\d+),(\d+)\):`)
		matches = re.FindAllStringSubmatch(output, -1)

		if len(matches) > 0 {
			for _, match := range matches {
				if len(match) > 2 {
					lineNum, _ := strconv.Atoi(match[1])
					colNum, _ := strconv.Atoi(match[2])
					errors = append(errors, ValidationError{
						Line:    lineNum,
						Column:  colNum,
						Message: "Type error",
					})
				}
			}
		} else {
			errors = append(errors, ValidationError{
				Line:    0,
				Message: output,
			})
		}
	}

	return errors
}
