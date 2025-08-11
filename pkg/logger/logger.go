package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"ai-government-consultant/internal/config"

	"github.com/gin-gonic/gin"
)

// Logger interface defines logging methods
type Logger interface {
	Debug(message string, fields map[string]interface{})
	Info(message string, fields map[string]interface{})
	Warn(message string, fields map[string]interface{})
	Error(message string, err error, fields map[string]interface{})
	Fatal(message string, err error, fields map[string]interface{})
}

// logger implements the Logger interface
type logger struct {
	level  string
	format string
	output *log.Logger
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Error     string                 `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// New creates a new logger instance
func New(cfg config.LoggingConfig) Logger {
	var output *log.Logger

	switch cfg.Output {
	case "stdout":
		output = log.New(os.Stdout, "", 0)
	case "stderr":
		output = log.New(os.Stderr, "", 0)
	default:
		// For file output, create or append to file
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		output = log.New(file, "", 0)
	}

	return &logger{
		level:  cfg.Level,
		format: cfg.Format,
		output: output,
	}
}

// Debug logs debug level messages
func (l *logger) Debug(message string, fields map[string]interface{}) {
	if l.shouldLog("debug") {
		l.log("DEBUG", message, nil, fields)
	}
}

// Info logs info level messages
func (l *logger) Info(message string, fields map[string]interface{}) {
	if l.shouldLog("info") {
		l.log("INFO", message, nil, fields)
	}
}

// Warn logs warning level messages
func (l *logger) Warn(message string, fields map[string]interface{}) {
	if l.shouldLog("warn") {
		l.log("WARN", message, nil, fields)
	}
}

// Error logs error level messages
func (l *logger) Error(message string, err error, fields map[string]interface{}) {
	if l.shouldLog("error") {
		l.log("ERROR", message, err, fields)
	}
}

// Fatal logs fatal level messages and exits
func (l *logger) Fatal(message string, err error, fields map[string]interface{}) {
	l.log("FATAL", message, err, fields)
	os.Exit(1)
}

// log handles the actual logging
func (l *logger) log(level, message string, err error, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	var output string
	if l.format == "json" {
		jsonBytes, _ := json.Marshal(entry)
		output = string(jsonBytes)
	} else {
		// Simple text format
		output = fmt.Sprintf("[%s] %s: %s", entry.Timestamp, level, message)
		if err != nil {
			output += fmt.Sprintf(" error=%s", err.Error())
		}
		if fields != nil {
			fieldsJSON, _ := json.Marshal(fields)
			output += fmt.Sprintf(" fields=%s", string(fieldsJSON))
		}
	}

	l.output.Println(output)
}

// shouldLog determines if a message should be logged based on level
func (l *logger) shouldLog(level string) bool {
	levels := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
		"fatal": 4,
	}

	currentLevel, exists := levels[l.level]
	if !exists {
		currentLevel = 1 // default to info
	}

	messageLevel, exists := levels[level]
	if !exists {
		return false
	}

	return messageLevel >= currentLevel
}

// GinMiddleware returns a Gin middleware for logging HTTP requests
func GinMiddleware(logger Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build log fields
		fields := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency":    latency.String(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		if raw != "" {
			fields["query"] = raw
		}

		// Log based on status code
		status := c.Writer.Status()
		message := fmt.Sprintf("%s %s", c.Request.Method, path)

		switch {
		case status >= 500:
			logger.Error(message, nil, fields)
		case status >= 400:
			logger.Warn(message, fields)
		default:
			logger.Info(message, fields)
		}
	}
}
