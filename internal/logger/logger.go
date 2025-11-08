package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	logFile    *os.File
	logMutex   sync.Mutex
	verbose    bool
	debug      bool
	onlyStderr bool
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// SetLogFile sets the log file for writing logs
func SetLogFile(filename string) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile != nil {
		logFile.Close()
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	logFile = file
	onlyStderr = false
	return nil
}

// SetVerbose enables verbose logging
func SetVerbose(v bool) {
	verbose = v
}

// SetDebug enables debug logging
func SetDebug(d bool) {
	debug = d
}

// SetStderrOnly sets logging to stderr only (no file output)
func SetStderrOnly() {
	logMutex.Lock()
	defer logMutex.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	onlyStderr = true
}

// Debug logs a debug message
func Debug(msg string) {
	logWithLevel(LogLevelDebug, msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	logWithLevel(LogLevelDebug, fmt.Sprintf(format, args...))
}

// Info logs an info message
func Info(msg string) {
	logWithLevel(LogLevelInfo, msg)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	logWithLevel(LogLevelInfo, fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func Warn(msg string) {
	logWithLevel(LogLevelWarn, msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	logWithLevel(LogLevelWarn, fmt.Sprintf(format, args...))
}

// Error logs an error message
func Error(msg string) {
	logWithLevel(LogLevelError, msg)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	logWithLevel(LogLevelError, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits the program
func Fatal(msg string) {
	logWithLevel(LogLevelFatal, msg)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits the program
func Fatalf(format string, args ...interface{}) {
	logWithLevel(LogLevelFatal, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// logWithLevel logs a message at the specified level
func logWithLevel(level LogLevel, msg string) {
	// Skip debug messages unless debug mode is enabled
	if level == LogLevelDebug && !debug {
		return
	}

	// Skip info messages unless verbose mode is enabled (except for important messages)
	if level == LogLevelInfo && !verbose && !shouldLogByDefault(msg) {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := levelString(level)
	logMessage := fmt.Sprintf("[%s] %s: %s", timestamp, levelStr, msg)

	// Always write to stderr
	fmt.Fprintf(os.Stderr, "%s\n", logMessage)

	// Write to file if configured and not in stderr-only mode
	if !onlyStderr && logFile != nil {
		logMutex.Lock()
		defer logMutex.Unlock()

		if logFile != nil {
			fmt.Fprintf(logFile, "%s\n", logMessage)
		}
	}
}

// levelString returns the string representation of a log level
func levelString(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// shouldLogByDefault determines if a message should be logged even without verbose mode
func shouldLogByDefault(msg string) bool {
	// Always log important system messages
	importantKeywords := []string{
		"SERVER STARTUP",
		"API key",
		"MCP Server",
		"ERROR",
		"FATAL",
		"Configuration",
		"Starting",
		"Shutting down",
	}

	msgUpper := strings.ToUpper(msg)
	for _, keyword := range importantKeywords {
		if strings.Contains(msgUpper, keyword) {
			return true
		}
	}

	return false
}

// Close closes the log file
func Close() {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// GetLogFile returns the current log file path
func GetLogFile() string {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile != nil {
		return logFile.Name()
	}
	return ""
}

// Flush flushes the log file
func Flush() {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile != nil {
		_ = logFile.Sync()
	}
}
