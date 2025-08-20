// Package pkg provides utility functions for logging.
package pkg

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// GetLogger returns the configured logger instance
func GetLogger(levelStr string) *slog.Logger {
	logLevel := slog.LevelDebug
	final := strings.ToUpper(levelStr)
	switch final {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	}

	fmt.Println("final logLevel", logLevel)
	// Set up structured logging with JSON format and proper level
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}

	// Use JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	// Replace the default slog logger
	return logger
}
