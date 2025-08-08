package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps logrus with additional functionality
type Logger struct {
	*logrus.Logger
	fields logrus.Fields
}

// NewLogger creates a new logger instance
func NewLogger(level, logFile string) *Logger {
	log := logrus.New()
	
	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)

	// Set formatter
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     false,
	})

	// Set output
	if logFile != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Failed to create log directory: %v\n", err)
		} else {
			// Use lumberjack for log rotation
			fileLogger := &lumberjack.Logger{
				Filename:   logFile,
				MaxSize:    100, // MB
				MaxBackups: 3,
				MaxAge:     28, // days
				Compress:   true,
			}

			// Write to both file and stdout
			multiWriter := io.MultiWriter(os.Stdout, fileLogger)
			log.SetOutput(multiWriter)
		}
	}

	return &Logger{
		Logger: log,
		fields: make(logrus.Fields),
	}
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		Logger: l.Logger,
		fields: newFields,
	}
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		Logger: l.Logger,
		fields: newFields,
	}
}

// WithComponent adds a component field to the logger
func (l *Logger) WithComponent(component string) *Logger {
	return l.WithField("component", component)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	entry := l.Logger.WithFields(l.fields)
	if len(args) > 0 {
		entry.Debugf(msg, args...)
	} else {
		entry.Debug(msg)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	entry := l.Logger.WithFields(l.fields)
	if len(args) > 0 {
		// Handle key-value pairs
		if len(args)%2 == 0 {
			fields := make(logrus.Fields)
			for i := 0; i < len(args); i += 2 {
				if key, ok := args[i].(string); ok {
					fields[key] = args[i+1]
				}
			}
			entry.WithFields(fields).Info(msg)
		} else {
			entry.Infof(msg, args...)
		}
	} else {
		entry.Info(msg)
	}
}

// Warning logs a warning message
func (l *Logger) Warning(msg string, args ...interface{}) {
	entry := l.Logger.WithFields(l.fields)
	if len(args) > 0 {
		if len(args)%2 == 0 {
			fields := make(logrus.Fields)
			for i := 0; i < len(args); i += 2 {
				if key, ok := args[i].(string); ok {
					fields[key] = args[i+1]
				}
			}
			entry.WithFields(fields).Warning(msg)
		} else {
			entry.Warningf(msg, args...)
		}
	} else {
		entry.Warning(msg)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	entry := l.Logger.WithFields(l.fields)
	if len(args) > 0 {
		if len(args)%2 == 0 {
			fields := make(logrus.Fields)
			for i := 0; i < len(args); i += 2 {
				if key, ok := args[i].(string); ok {
					fields[key] = args[i+1]
				}
			}
			entry.WithFields(fields).Error(msg)
		} else {
			entry.Errorf(msg, args...)
		}
	} else {
		entry.Error(msg)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	entry := l.Logger.WithFields(l.fields)
	if len(args) > 0 {
		entry.Fatalf(msg, args...)
	} else {
		entry.Fatal(msg)
	}
}

// Writer returns an io.Writer for the logger
func (l *Logger) Writer() io.Writer {
	return l.Logger.Writer()
}

// GinLogger returns a Gin middleware for logging HTTP requests
func (l *Logger) GinLogger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[%s] \"%s %s %s %d %s \"%s\" %s\"\n",
				param.TimeStamp.Format("2006-01-02 15:04:05"),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
				param.ClientIP,
			)
		},
		Output:    l.Writer(),
		SkipPaths: []string{"/health", "/metrics"},
	})
}

// HTTPLogger logs HTTP request details
func (l *Logger) HTTPLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status
		status := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Build log entry
		entry := l.WithFields(map[string]interface{}{
			"method":      c.Request.Method,
			"path":        path,
			"query":       raw,
			"status_code": status,
			"latency":     latency,
			"client_ip":   clientIP,
			"user_agent":  c.Request.UserAgent(),
			"request_id":  c.GetString("request_id"),
		})

		// Log based on status code
		if status >= 500 {
			entry.Error("HTTP request completed with server error")
		} else if status >= 400 {
			entry.Warning("HTTP request completed with client error")
		} else {
			entry.Info("HTTP request completed successfully")
		}
	}
}

// SecurityLogger logs security-related events
func (l *Logger) SecurityLogger(event, userID, details string) {
	l.WithFields(map[string]interface{}{
		"event_type": "security",
		"event":      event,
		"user_id":    userID,
		"details":    details,
		"timestamp":  time.Now().Unix(),
	}).Warning("Security event logged")
}

// AuditLogger logs audit events
func (l *Logger) AuditLogger(action, userID, resource, details string) {
	l.WithFields(map[string]interface{}{
		"event_type": "audit",
		"action":     action,
		"user_id":    userID,
		"resource":   resource,
		"details":    details,
		"timestamp":  time.Now().Unix(),
	}).Info("Audit event logged")
}

// VotingLogger logs voting-specific events
func (l *Logger) VotingLogger(event, voterHash, pollingUnit, details string) {
	l.WithFields(map[string]interface{}{
		"event_type":     "voting",
		"event":          event,
		"voter_hash":     voterHash,
		"polling_unit":   pollingUnit,
		"details":        details,
		"timestamp":      time.Now().Unix(),
	}).Info("Voting event logged")
}

// BlockchainLogger logs blockchain-related events
func (l *Logger) BlockchainLogger(event, txHash, blockNumber, details string) {
	l.WithFields(map[string]interface{}{
		"event_type":   "blockchain",
		"event":        event,
		"tx_hash":      txHash,
		"block_number": blockNumber,
		"details":      details,
		"timestamp":    time.Now().Unix(),
	}).Info("Blockchain event logged")
}

// SystemLogger logs system events
func (l *Logger) SystemLogger(event, component, details string, level logrus.Level) {
	entry := l.WithFields(map[string]interface{}{
		"event_type": "system",
		"event":      event,
		"component":  component,
		"details":    details,
		"timestamp":  time.Now().Unix(),
	})

	switch level {
	case logrus.DebugLevel:
		entry.Debug("System event logged")
	case logrus.InfoLevel:
		entry.Info("System event logged")
	case logrus.WarnLevel:
		entry.Warning("System event logged")
	case logrus.ErrorLevel:
		entry.Error("System event logged")
	case logrus.FatalLevel:
		entry.Fatal("System event logged")
	default:
		entry.Info("System event logged")
	}
}

// PerformanceLogger logs performance metrics
func (l *Logger) PerformanceLogger(operation string, duration time.Duration, success bool) {
	l.WithFields(map[string]interface{}{
		"event_type": "performance",
		"operation":  operation,
		"duration":   duration.Milliseconds(),
		"success":    success,
		"timestamp":  time.Now().Unix(),
	}).Info("Performance event logged")
}

// StructuredError logs a structured error with context
func (l *Logger) StructuredError(err error, context map[string]interface{}) {
	fields := map[string]interface{}{
		"error":     err.Error(),
		"timestamp": time.Now().Unix(),
	}
	
	// Merge context fields
	for k, v := range context {
		fields[k] = v
	}
	
	l.WithFields(fields).Error("Structured error logged")
}

// RequestLogger creates a middleware that logs request details with context
func (l *Logger) RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := generateRequestID()
		c.Set("request_id", requestID)
		c.Set("logger", l.WithField("request_id", requestID))

		// Log request start
		l.WithFields(map[string]interface{}{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("Request started")

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		// Log request completion
		l.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"status_code": c.Writer.Status(),
			"latency_ms":  latency.Milliseconds(),
		}).Info("Request completed")
	}
}

// GetLoggerFromContext retrieves the logger from Gin context
func GetLoggerFromContext(c *gin.Context) *Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*Logger); ok {
			return l
		}
	}
	// Return a default logger if not found
	return NewLogger("info", "")
}

// SetLogLevel dynamically sets the log level
func (l *Logger) SetLogLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	l.Logger.SetLevel(logLevel)
	return nil
}

// SetFormatter sets the log formatter
func (l *Logger) SetFormatter(format string) {
	switch format {
	case "json":
		l.Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	case "text":
		l.Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	default:
		l.Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewStructuredLogger creates a logger specifically for structured logging
func NewStructuredLogger(level, logFile string) *Logger {
	logger := NewLogger(level, logFile)
	logger.SetFormatter("json")
	return logger
}

// Close closes any open log files
func (l *Logger) Close() error {
	// If using lumberjack, it doesn't need explicit closing
	// This method is here for interface compatibility
	return nil
}