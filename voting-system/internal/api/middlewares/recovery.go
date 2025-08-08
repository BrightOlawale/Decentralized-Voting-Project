package middlewares

import (
	"net/http"
	"time"
	"voting-system/internal/api/models"

	"github.com/gin-gonic/gin"
)

// Recovery middleware recovers from panics
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.JSON(http.StatusInternalServerError, models.BaseResponse{
				Success: false,
				Error: &models.ErrorInfo{
					Code:    models.ErrCodeInternalError,
					Message: "Internal server error",
					Details: err,
				},
				Timestamp: time.Now().Unix(),
			})
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
