package pkg

import (
	"log/slog"
	"os"
	"strings"
)

var logger *slog.Logger

// init initializes the logger with proper configuration
func init() {
	// Get log level from environment variable, default to INFO
	logLevel := slog.LevelInfo
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			logLevel = slog.LevelDebug
		case "INFO":
			logLevel = slog.LevelInfo
		case "WARN":
			logLevel = slog.LevelWarn
		case "ERROR":
			logLevel = slog.LevelError
		}
	}

	// Set up structured logging with JSON format and proper level
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}

	// Use JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger = slog.New(handler)

	// Replace the default slog logger
	slog.SetDefault(logger)
}

// GetLogger returns the configured logger instance
func GetLogger() *slog.Logger {
	return logger
}

// SetLogLevel allows runtime configuration of log level
func SetLogLevel(level slog.Level) {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger = slog.New(handler)
	slog.SetDefault(logger)
}
