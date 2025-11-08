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

// GoValidator validates Go code syntax
type GoValidator struct{}

// Validate checks Go syntax using gofmt
func (v *GoValidator) Validate(code string, filePath string) (*ValidationResult, error) {
	// Check if gofmt is available using tool cache
	toolCache := GetToolCache()
	if !toolCache.IsAvailable("gofmt") {
		// No Go available, skip validation
		return &ValidationResult{Valid: true, Errors: nil}, nil
	}

	// Create a temporary file with the code
	tmpFile, err := os.CreateTemp("", "validate-*.go")
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

	// Run gofmt to check syntax with timeout
	cmd := exec.CommandContext(ctx, "gofmt", "-e", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("go validation timeout exceeded (5s)")
	}

	// gofmt returns error if there are syntax errors
	if err != nil {
		errors := v.parseErrors(string(output))
		return &ValidationResult{Valid: false, Errors: errors}, nil
	}

	return &ValidationResult{Valid: true, Errors: nil}, nil
}

// CanAutoFix returns true - gofmt can auto-format Go code
func (v *GoValidator) CanAutoFix() bool {
	return true
}

// AutoFix uses gofmt to format Go code
func (v *GoValidator) AutoFix(code string) (string, error) {
	// Create context with timeout (5s should be plenty for single file formatting)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gofmt")

	// Write code to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		_, _ = stdin.Write([]byte(code))
	}()

	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("gofmt timeout exceeded (5s)")
	}

	if err != nil {
		return "", fmt.Errorf("gofmt failed: %w", err)
	}

	return string(output), nil
}

// parseErrors parses Go error messages
func (v *GoValidator) parseErrors(output string) []ValidationError {
	var errors []ValidationError

	// Go error format: file.go:line:col: error message
	// Parse complete line with error message
	re := regexp.MustCompile(`(\S+):(\d+):(\d+):\s*(.+)`)
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

	// Fallback: try simpler pattern if complex one didn't work
	if len(errors) == 0 {
		re = regexp.MustCompile(`:(\d+):(\d+):`)
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
			// Generic error with no line/col
			errors = append(errors, ValidationError{
				Line:    0,
				Message: output,
			})
		}
	}

	return errors
}
