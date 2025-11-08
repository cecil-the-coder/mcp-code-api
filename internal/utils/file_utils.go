package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadFileContent reads the content of a file
func ReadFileContent(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty string
			return "", nil
		}
		return "", err
	}

	return string(content), nil
}

// WriteFileContent writes content to a file
func WriteFileContent(filePath, content string) error {
	if filePath == "" {
		return nil
	}

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// GetLanguageFromFile determines the programming language from a file path
func GetLanguageFromFile(filePath string, language *string) string {
	// If language is explicitly provided, use it
	if language != nil && *language != "" {
		return *language
	}

	// Determine language from file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".h":
		return "c"
	case ".hpp":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".rs":
		return "rust"
	case ".sh":
		return "bash"
	case ".zsh":
		return "zsh"
	case ".fish":
		return "fish"
	case ".ps1":
		return "powershell"
	case ".bat", ".cmd":
		return "batch"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".less":
		return "less"
	case ".xml":
		return "xml"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sql":
		return "sql"
	case ".dockerfile":
		return "dockerfile"
	case ".makefile":
		return "makefile"
	case ".md":
		return "markdown"
	case ".txt":
		return "text"
	case ".ini":
		return "ini"
	case ".conf":
		return "config"
	case ".gitignore", ".gitattributes":
		return "git"
	case ".env":
		return "env"
	case ".log":
		return "log"
	default:
		// Default to text for unknown extensions
		return "text"
	}
}

// FileExists checks if a file exists
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	return err == nil && !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	return err == nil && info.IsDir()
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(dirPath string) error {
	if dirPath == "" {
		return nil
	}
	return os.MkdirAll(dirPath, 0755)
}

// IsAbsolutePath checks if a path is absolute
func IsAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

// NormalizePath normalizes a file path
func NormalizePath(path string) string {
	return filepath.Clean(path)
}

// GetRelativePath gets the relative path from base to target
func GetRelativePath(base, target string) (string, error) {
	return filepath.Rel(base, target)
}

// GetFileExtension returns the file extension (without dot)
func GetFileExtension(filePath string) string {
	return strings.TrimPrefix(filepath.Ext(filePath), ".")
}

// GetFileName returns the filename without extension
func GetFileName(filePath string) string {
	return strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
}

// GetFileNameWithExtension returns the filename with extension
func GetFileNameWithExtension(filePath string) string {
	return filepath.Base(filePath)
}

// GetDirPath returns the directory path of a file
func GetDirPath(filePath string) string {
	return filepath.Dir(filePath)
}

// JoinPaths joins multiple path elements
func JoinPaths(elements ...string) string {
	return filepath.Join(elements...)
}

// CleanCodeResponse removes markdown formatting from AI responses
func CleanCodeResponse(response string) string {
	// Remove markdown code blocks
	lines := strings.Split(response, "\n")
	var cleanLines []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip code block markers
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		// Skip language identifiers at the start of code blocks
		if !inCodeBlock && (strings.HasPrefix(trimmed, "```python") ||
			strings.HasPrefix(trimmed, "```javascript") ||
			strings.HasPrefix(trimmed, "```typescript") ||
			strings.HasPrefix(trimmed, "```go") ||
			strings.HasPrefix(trimmed, "```java") ||
			strings.HasPrefix(trimmed, "```cpp") ||
			strings.HasPrefix(trimmed, "```c") ||
			strings.HasPrefix(trimmed, "```ruby") ||
			strings.HasPrefix(trimmed, "```php") ||
			strings.HasPrefix(trimmed, "```swift") ||
			strings.HasPrefix(trimmed, "```kotlin") ||
			strings.HasPrefix(trimmed, "```rust") ||
			strings.HasPrefix(trimmed, "```bash") ||
			strings.HasPrefix(trimmed, "```sh") ||
			strings.HasPrefix(trimmed, "```sql") ||
			strings.HasPrefix(trimmed, "```html") ||
			strings.HasPrefix(trimmed, "```css") ||
			strings.HasPrefix(trimmed, "```json") ||
			strings.HasPrefix(trimmed, "```yaml") ||
			strings.HasPrefix(trimmed, "```xml") ||
			strings.HasPrefix(trimmed, "```dockerfile") ||
			strings.HasPrefix(trimmed, "```makefile") ||
			strings.HasPrefix(trimmed, "```markdown") ||
			strings.HasPrefix(trimmed, "```text")) {
			continue
		}

		// Add the line if we're not in a code block that we're skipping
		if !inCodeBlock || trimmed != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	cleaned := strings.Join(cleanLines, "\n")

	// Remove leading and trailing whitespace
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}
