package middlewares

import (
	"net/http"
	"sync"
	"time"
	"voting-system/internal/api/models"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	visitors map[string]*Visitor
	mutex    sync.RWMutex
	rate     int
	burst    int
	cleanup  time.Duration
}

type Visitor struct {
	limiter  *time.Ticker
	lastSeen time.Time
	count    int
	window   time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		burst:    burst,
		cleanup:  time.Minute * 10,
	}

	// Start cleanup goroutine
	go rl.cleanupExpiredVisitors()
	return rl
}

// RateLimit middleware implements rate limiting
func RateLimit() gin.HandlerFunc {
	limiter := NewRateLimiter(100, 200) // 100 requests per minute, burst of 200

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, models.BaseResponse{
				Success: false,
				Error: &models.ErrorInfo{
					Code:    "RATE_LIMIT_EXCEEDED",
					Message: "Rate limit exceeded. Please try again later.",
				},
				Timestamp: time.Now().Unix(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	visitor, exists := rl.visitors[ip]

	if !exists {
		rl.visitors[ip] = &Visitor{
			lastSeen: now,
			count:    1,
			window:   now,
		}
		return true
	}

	visitor.lastSeen = now

	// Reset counter if window has passed
	if now.Sub(visitor.window) >= time.Minute {
		visitor.count = 1
		visitor.window = now
		return true
	}

	if visitor.count >= rl.rate {
		return false
	}

	visitor.count++
	return true
}

func (rl *RateLimiter) cleanupExpiredVisitors() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for ip, visitor := range rl.visitors {
			if now.Sub(visitor.lastSeen) > rl.cleanup {
				delete(rl.visitors, ip)
			}
		}
		rl.mutex.Unlock()
	}
}
