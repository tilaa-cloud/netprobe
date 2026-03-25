package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// LogLevel defines the verbosity level for logging
type LogLevel int

const (
	LevelSilent LogLevel = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
)

var (
	currentLevel = LevelInfo // Default to Info
	mu           sync.RWMutex
)

// SetLevel sets the global log level
func SetLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

// SetLevelFromString sets the log level from a string (useful for env vars)
// Valid values: "silent", "error", "warn", "info", "debug", "trace"
func SetLevelFromString(levelStr string) {
	levelStr = strings.ToLower(strings.TrimSpace(levelStr))
	level := LevelInfo // default
	switch levelStr {
	case "silent":
		level = LevelSilent
	case "error":
		level = LevelError
	case "warn":
		level = LevelWarn
	case "info":
		level = LevelInfo
	case "debug":
		level = LevelDebug
	case "trace":
		level = LevelTrace
	}
	SetLevel(level)
}

// GetLevel returns the current log level
func GetLevel() LogLevel {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// shouldLog checks if the given level should be logged
func shouldLog(level LogLevel) bool {
	mu.RLock()
	defer mu.RUnlock()
	return level <= currentLevel
}

// Trace logs at trace level (most verbose)
func Trace(format string, args ...interface{}) {
	if shouldLog(LevelTrace) {
		log.Printf("[TRACE] "+format, args...)
	}
}

// Debug logs at debug level
func Debug(format string, args ...interface{}) {
	if shouldLog(LevelDebug) {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs at info level (default)
func Info(format string, args ...interface{}) {
	if shouldLog(LevelInfo) {
		log.Printf(format, args...)
	}
}

// Warn logs at warn level
func Warn(format string, args ...interface{}) {
	if shouldLog(LevelWarn) {
		log.Printf("[WARN] "+format, args...)
	}
}

// Error logs at error level
func Error(format string, args ...interface{}) {
	if shouldLog(LevelError) {
		log.Printf("[ERROR] "+format, args...)
	}
}

// Fatal logs a fatal error and exits
func Fatal(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

// Printf is a direct printf that respects the Info level
func Printf(format string, args ...interface{}) {
	if shouldLog(LevelInfo) {
		log.Printf(format, args...)
	}
}

// Println is a direct println that respects the Info level
func Println(format string) {
	if shouldLog(LevelInfo) {
		log.Println(format)
	}
}

// InitFromEnv initializes the logger from environment variables
// LOG_LEVEL env var can be set to: silent, error, warn, info, debug, trace
func InitFromEnv() {
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		SetLevelFromString(envLevel)
	}
}

// String returns the string representation of a LogLevel
func (l LogLevel) String() string {
	switch l {
	case LevelSilent:
		return "silent"
	case LevelError:
		return "error"
	case LevelWarn:
		return "warn"
	case LevelInfo:
		return "info"
	case LevelDebug:
		return "debug"
	case LevelTrace:
		return "trace"
	default:
		return fmt.Sprintf("unknown(%d)", l)
	}
}
