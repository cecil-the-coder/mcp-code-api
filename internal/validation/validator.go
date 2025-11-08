package validation

import (
	"fmt"
)

// ValidationResult represents the result of syntax validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ValidationError represents a syntax error
type ValidationError struct {
	Line    int
	Column  int
	Message string
}

// Validator defines the interface for language validators
type Validator interface {
	// Validate checks if the code is syntactically correct
	Validate(code string, filePath string) (*ValidationResult, error)

	// CanAutoFix returns true if this validator can automatically fix syntax errors
	CanAutoFix() bool

	// AutoFix attempts to automatically fix syntax errors
	AutoFix(code string) (string, error)
}

// FormatValidationErrors formats validation errors into a user-friendly message
func FormatValidationErrors(errors []ValidationError, language Language) string {
	if len(errors) == 0 {
		return ""
	}

	msg := fmt.Sprintf("❌ Syntax validation failed for %s:\n\n", language)
	for i, err := range errors {
		if i >= 5 { // Limit to 5 errors
			msg += fmt.Sprintf("... and %d more errors\n", len(errors)-5)
			break
		}
		if err.Line > 0 {
			msg += fmt.Sprintf("  Line %d", err.Line)
			if err.Column > 0 {
				msg += fmt.Sprintf(", Column %d", err.Column)
			}
			msg += fmt.Sprintf(": %s\n", err.Message)
		} else {
			msg += fmt.Sprintf("  %s\n", err.Message)
		}
	}

	msg += "\n🔧 Please fix these syntax errors and try again."
	return msg
}
