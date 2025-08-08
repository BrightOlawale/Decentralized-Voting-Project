package middlewares

import (
	"net/http"
	"strings"
	"time"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/models"

	"github.com/gin-gonic/gin"
)

// AuthRequired middleware validates JWT tokens
func AuthRequired(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, models.BaseResponse{
				Success: false,
				Error: &models.ErrorInfo{
					Code:    models.ErrCodeUnauthorized,
					Message: "Authorization token required",
				},
				Timestamp: time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// Validate token using the AuthService
		claims, err := services.AuthService().ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.BaseResponse{
				Success: false,
				Error: &models.ErrorInfo{
					Code:    models.ErrCodeInvalidToken,
					Message: "Invalid or expired token: " + err.Error(),
				},
				Timestamp: time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// Set user context from validated claims
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Set("user_permissions", claims.Permissions)
		c.Set("session_id", claims.SessionID)

		c.Next()
	}
}

// AdminRequired middleware ensures user has admin role
func AdminRequired(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists || userRole != "admin" {
			c.JSON(http.StatusForbidden, models.BaseResponse{
				Success: false,
				Error: &models.ErrorInfo{
					Code:    models.ErrCodeForbidden,
					Message: "Admin access required",
				},
				Timestamp: time.Now().Unix(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// PermissionRequired middleware checks for specific permissions
func PermissionRequired(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, exists := c.Get("user_permissions")
		if !exists {
			c.JSON(http.StatusForbidden, models.BaseResponse{
				Success: false,
				Error: &models.ErrorInfo{
					Code:    models.ErrCodeForbidden,
					Message: "Permission required: " + permission,
				},
				Timestamp: time.Now().Unix(),
			})
			c.Abort()
			return
		}

		userPerms, ok := permissions.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.BaseResponse{
				Success: false,
				Error: &models.ErrorInfo{
					Code:    models.ErrCodeInternalError,
					Message: "Invalid permission format",
				},
				Timestamp: time.Now().Unix(),
			})
			c.Abort()
			return
		}

		hasPermission := false
		for _, perm := range userPerms {
			if perm == permission || perm == "*" {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, models.BaseResponse{
				Success: false,
				Error: &models.ErrorInfo{
					Code:    models.ErrCodeForbidden,
					Message: "Insufficient permissions",
				},
				Timestamp: time.Now().Unix(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// WebAuth middleware for web interface authentication
func WebAuth(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for session cookie
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/web/login")
			c.Abort()
			return
		}

		// TODO: Implement session validation through interface
		// sessionData, err := services.Session.Get(sessionID)
		// For now, we'll skip validation to avoid import cycle
		_ = sessionID

		// Set user context (placeholder)
		c.Set("user_id", "placeholder")
		c.Set("user_role", "user")
		c.Next()
	}
}

// WSAuthRequired middleware for WebSocket authentication
func WSAuthRequired(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required for WebSocket"})
			c.Abort()
			return
		}

		// Validate token using the AuthService
		claims, err := services.AuthService().ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// extractToken extracts JWT token from Authorization header
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
