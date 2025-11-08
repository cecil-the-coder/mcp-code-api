package formatting

import (
	"fmt"
	"strings"

	"github.com/cecil-the-coder/mcp-code-api/internal/types"
	"github.com/cecil-the-coder/mcp-code-api/internal/utils"
)

// FormatEditResponse formats an edit response with visual diff
func FormatEditResponse(fileName, existingContent, newContent, filePath string) *types.Content {
	// Generate diff between existing and new content
	diff := generateDiff(existingContent, newContent)

	// Create formatted response
	response := fmt.Sprintf("🔝 File Modified: %s\n\n📁 Path: %s\n\n🔄 Changes Summary:\n%s\n\n💾 File has been updated successfully.\n\n⚠️  Important: Always use 'write' tool for any additional modifications.\n", fileName, filePath, diff)

	return &types.Content{
		Type: "text",
		Text: response,
	}
}

// FormatCreateResponse formats a create response
func FormatCreateResponse(fileName, content, filePath string) *types.Content {
	// Get language for syntax highlighting
	language := utils.GetLanguageFromFile(filePath, nil)

	// Create formatted response
	response := fmt.Sprintf("✨ File Created: %s\n\n📁 Path: %s\n\n🔤 Language: %s\n\n📄 Content Preview:\n%s\n\n💾 File has been created successfully.\n\n⚠️  Important: Always use 'write' tool for any additional modifications.\n", fileName, filePath, language, formatContentPreview(content))

	return &types.Content{
		Type: "text",
		Text: response,
	}
}

// FormatErrorResponse formats an error response
func FormatErrorResponse(err error) *types.Content {
	response := fmt.Sprintf("❌ Operation Failed\n\n🚨 Error: %v\n\n💡 Troubleshooting:\n• Check if file path is valid and accessible\n• Verify your API keys are properly configured\n• Ensure you have write permissions for the target directory\n• Try using a more specific prompt\n\n📞 If the problem persists, please check the debug log file.\n", err)

	return &types.Content{
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
		var oldLine, newLine string

		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine == newLine {
			// Lines are the same, skip
			continue
		}

		if i >= len(oldLines) {
			// Line was added
			diffBuilder.WriteString(fmt.Sprintf("✅ %s\n", newLine))
			additions++
		} else if i >= len(newLines) {
			// Line was removed
			diffBuilder.WriteString(fmt.Sprintf("❌ %s\n", oldLine))
			removals++
		} else {
			// Line was modified
			diffBuilder.WriteString(fmt.Sprintf("❌ %s\n", oldLine))
			diffBuilder.WriteString(fmt.Sprintf("✅ %s\n", newLine))
			modifications++
		}
	}

	// Add summary
	summary := fmt.Sprintf("📊 Changes:\n   • %d additions\n   • %d removals\n   • %d modifications", additions, removals, modifications)

	diff := diffBuilder.String()
	if diff == "" {
		return summary
	}

	return fmt.Sprintf("%s\n\n%s", summary, diff)
}

// formatContentPreview formats a content preview with syntax highlighting indication
func formatContentPreview(content string) string {
	lines := strings.Split(content, "\n")
	maxPreviewLines := 10

	if len(lines) <= maxPreviewLines {
		// Content is short enough to show completely
		return fmt.Sprintf("```\n%s\n```", content)
	}

	// Show preview with ellipsis
	previewLines := lines[:maxPreviewLines]
	preview := strings.Join(previewLines, "\n")

	return fmt.Sprintf("```%s\n%s\n...\n\n📏 Full content: %d lines total\n", "", preview, len(lines))
}

// FormatSuccessResponse formats a general success response
func FormatSuccessResponse(message string) *types.Content {
	response := fmt.Sprintf("✅ Success\n\n🎉 %s\n\n💡 Tip: Continue using the 'write' tool for all your code operations.\n", message)

	return &types.Content{
		Type: "text",
		Text: response,
	}
}

// FormatInfoResponse formats an informational response
func FormatInfoResponse(title, message string) *types.Content {
	response := fmt.Sprintf("ℹ️ %s\n\n%s\n", title, message)

	return &types.Content{
		Type: "text",
		Text: response,
	}
}

// FormatWarningResponse formats a warning response
func FormatWarningResponse(message string) *types.Content {
	response := fmt.Sprintf("⚠️ Warning\n\n%s\n\n💡 Please review and consider the above information.\n", message)

	return &types.Content{
		Type: "text",
		Text: response,
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
