package mcp

import (
	"fmt"
	"strings"
)

// NewEditResponse creates an edit response with visual diff
func NewEditResponse(fileName, existingContent, newContent, filePath string) *Content {
	diff := generateDiff(existingContent, newContent)

	response := fmt.Sprintf(`🔝 File Modified: %s

📁 Path: %s

🔄 Changes Summary:
%s

💾 File has been updated successfully.

⚠️  Important: Always use 'write' tool for any additional modifications.
`, fileName, filePath, diff)

	return &Content{
		Type: "text",
		Text: response,
	}
}

// NewCreateResponse creates a creation response
func NewCreateResponse(fileName, content, filePath string) *Content {
	language := "text"
	if fileName != "" {
		// Simple language detection based on extension
		if idx := len(fileName) - 1; idx >= 0 && fileName[idx] == 'g' && len(fileName) > 2 && fileName[idx-2:] == ".go" {
			language = "go"
		} else if idx := len(fileName) - 1; idx >= 0 && fileName[idx] == 's' && len(fileName) > 2 && fileName[idx-2:] == ".js" {
			language = "javascript"
		}
	}

	response := fmt.Sprintf(`📝 File Created: %s

📁 Path: %s

🔧 Language: %s

💾 File has been created successfully.

⚠️  Important: Always use 'write' tool for any additional modifications.
`, fileName, filePath, language)

	return &Content{
		Type: "text",
		Text: response,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(err error) *Content {
	response := fmt.Sprintf(`❌ Operation Failed

🚨 Error: %v

💡 Troubleshooting:
• Check if file path is valid and accessible
• Verify your API keys are properly configured
• Ensure you have write permissions for the target directory
• Try using a more specific prompt

📞 If the problem persists, please check the debug log file.
`, err)

	return &Content{
		Type: "text",
		Text: response,
	}
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(message string) *Content {
	response := fmt.Sprintf(`✅ Success

%s
`, message)

	return &Content{
		Type: "text",
		Text: response,
	}
}

// NewInfoResponse creates an info response
func NewInfoResponse(title, message string) *Content {
	response := fmt.Sprintf(`ℹ️  %s

%s
`, title, message)

	return &Content{
		Type: "text",
		Text: response,
	}
}

// NewWarningResponse creates a warning response
func NewWarningResponse(message string) *Content {
	response := fmt.Sprintf(`⚠️  Warning

%s
`, message)

	return &Content{
		Type: "text",
		Text: response,
	}
}

// generateDiff generates a simple visual diff between two text contents
func generateDiff(oldContent, newContent string) string {
	if oldContent == newContent {
		return "🔍 No changes detected"
	}

	// For simplicity, we'll use a basic diff approach
	// In a real implementation, you'd use a proper diff library
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var diffBuilder strings.Builder
	additions := 0
	removals := 0
	modifications := 0

	// Simple line-by-line comparison
	maxLines := max(len(oldLines), len(newLines))

	for i := 0; i < maxLines; i++ {
		oldLine := ""
		newLine := ""

		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine == newLine {
			continue
		}

		if oldLine == "" {
			diffBuilder.WriteString(fmt.Sprintf("+ %s\n", newLine))
			additions++
		} else if newLine == "" {
			diffBuilder.WriteString(fmt.Sprintf("- %s\n", oldLine))
			removals++
		} else {
			diffBuilder.WriteString(fmt.Sprintf("- %s\n", oldLine))
			diffBuilder.WriteString(fmt.Sprintf("+ %s\n", newLine))
			modifications++
		}
	}

	summary := fmt.Sprintf("Additions: %d, Removals: %d, Modifications: %d", additions, removals, modifications)

	if additions == 0 && removals == 0 && modifications == 0 {
		return "🔍 No changes detected"
	}

	return fmt.Sprintf("%s\n\n%s", diffBuilder.String(), summary)
}
