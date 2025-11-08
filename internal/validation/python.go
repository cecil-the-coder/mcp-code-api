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

// PythonValidator validates Python code syntax
type PythonValidator struct{}

// Validate checks Python syntax using py_compile
func (v *PythonValidator) Validate(code string, filePath string) (*ValidationResult, error) {
	// Create a temporary file with the code
	tmpFile, err := os.CreateTemp("", "validate-*.py")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(code); err != nil {
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}

	tmpFile.Close()

	// Use tool cache to check availability
	toolCache := GetToolCache()

	// Try python3 first, fall back to python
	pythonCmd := "python3"
	if !toolCache.IsAvailable("python3") {
		pythonCmd = "python"
		if !toolCache.IsAvailable("python") {
			// No Python available, skip validation
			return &ValidationResult{Valid: true, Errors: nil}, nil
		}
	}

	// Create context with timeout (5s should be plenty for single file validation)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run Python syntax check with timeout
	cmd := exec.CommandContext(ctx, pythonCmd, "-m", "py_compile", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("python validation timeout exceeded (5s)")
	}

	if err == nil {
		return &ValidationResult{Valid: true, Errors: nil}, nil
	}

	// Parse errors from output
	errors := v.parseErrors(string(output), tmpFile.Name())
	return &ValidationResult{Valid: false, Errors: errors}, nil
}

// CanAutoFix returns false - we don't auto-fix Python yet
func (v *PythonValidator) CanAutoFix() bool {
	return false
}

// AutoFix is not implemented for Python
func (v *PythonValidator) AutoFix(code string) (string, error) {
	return "", fmt.Errorf("auto-fix not supported for Python")
}

// parseErrors parses Python error messages
func (v *PythonValidator) parseErrors(output string, tmpPath string) []ValidationError {
	var errors []ValidationError

	// Python error format:
	// File "path", line X
	//     code line
	//     ^
	// SyntaxError: message
	// Try to extract the SyntaxError message
	re := regexp.MustCompile(`File ".*?", line (\d+)[\s\S]*?(SyntaxError|IndentationError|TabError):\s*(.+?)(?:\n|$)`)
	matches := re.FindAllStringSubmatch(output, -1)

	if len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 3 {
				lineNum, _ := strconv.Atoi(match[1])
				errorType := match[2]
				message := match[3]
				errors = append(errors, ValidationError{
					Line:    lineNum,
					Message: fmt.Sprintf("%s: %s", errorType, message),
				})
			}
		}
	}

	// Fallback: try simpler pattern if complex one didn't work
	if len(errors) == 0 {
		re = regexp.MustCompile(`line (\d+)`)
		matches = re.FindAllStringSubmatch(output, -1)

		if len(matches) > 0 {
			// Use the full output as message for all errors (better than nothing)
			for _, match := range matches {
				if len(match) > 1 {
					lineNum, _ := strconv.Atoi(match[1])
					errors = append(errors, ValidationError{
						Line:    lineNum,
						Message: "Syntax error (see full output for details)",
					})
				}
			}
		} else {
			// Generic error with no line number
			errors = append(errors, ValidationError{
				Line:    0,
				Message: output,
			})
		}
	}

	return errors
}

// NoOpValidator is a no-op validator for unsupported languages
type NoOpValidator struct{}

func (v *NoOpValidator) Validate(code string, filePath string) (*ValidationResult, error) {
	return &ValidationResult{Valid: true, Errors: nil}, nil
}

func (v *NoOpValidator) CanAutoFix() bool {
	return false
}

func (v *NoOpValidator) AutoFix(code string) (string, error) {
	return "", fmt.Errorf("validation not supported for this language")
}
