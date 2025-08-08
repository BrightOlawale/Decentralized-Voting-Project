package middlewares

import (
	"fmt"
	"math/rand"
	"time"
	"voting-system/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestLogging middleware logs HTTP requests
func RequestLogging(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Generate request ID
		requestID := generateRequestID()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Writer.Status()

		// Create log entry
		logData := map[string]interface{}{
			"request_id":  requestID,
			"method":      c.Request.Method,
			"path":        path,
			"query":       raw,
			"status_code": status,
			"latency_ms":  latency.Milliseconds(),
			"client_ip":   c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"user_id":     c.GetString("user_id"),
		}

		// Log based on status code
		if status >= 500 {
			log.WithFields(logData).Error("HTTP request completed with server error")
		} else if status >= 400 {
			log.WithFields(logData).Warning("HTTP request completed with client error")
		} else {
			log.WithFields(logData).Info("HTTP request completed")
		}
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
