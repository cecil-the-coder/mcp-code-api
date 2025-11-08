package validation

import (
	"path/filepath"
	"strings"
)

// Language represents a programming language
type Language string

const (
	LanguagePython     Language = "python"
	LanguageJavaScript Language = "javascript"
	LanguageTypeScript Language = "typescript"
	LanguageGo         Language = "go"
	LanguageRust       Language = "rust"
	LanguageJava       Language = "java"
	LanguageC          Language = "c"
	LanguageCPP        Language = "cpp"
	LanguageRuby       Language = "ruby"
	LanguagePHP        Language = "php"
	LanguageUnknown    Language = "unknown"
)

// DetectLanguage detects the programming language from file extension
func DetectLanguage(filePath string) Language {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]Language{
		".py":   LanguagePython,
		".js":   LanguageJavaScript,
		".jsx":  LanguageJavaScript,
		".mjs":  LanguageJavaScript,
		".cjs":  LanguageJavaScript,
		".ts":   LanguageTypeScript,
		".tsx":  LanguageTypeScript,
		".go":   LanguageGo,
		".rs":   LanguageRust,
		".java": LanguageJava,
		".c":    LanguageC,
		".h":    LanguageC,
		".cpp":  LanguageCPP,
		".cc":   LanguageCPP,
		".cxx":  LanguageCPP,
		".hpp":  LanguageCPP,
		".hxx":  LanguageCPP,
		".rb":   LanguageRuby,
		".php":  LanguagePHP,
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return LanguageUnknown
}

// String returns the string representation of the language
func (l Language) String() string {
	return string(l)
}

// GetValidator returns the appropriate validator for the language
func (l Language) GetValidator() Validator {
	switch l {
	case LanguagePython:
		return &PythonValidator{}
	case LanguageJavaScript:
		return &JavaScriptValidator{}
	case LanguageTypeScript:
		return &TypeScriptValidator{}
	case LanguageGo:
		return &GoValidator{}
	default:
		return &NoOpValidator{}
	}
}
