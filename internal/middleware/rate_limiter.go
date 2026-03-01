//nolint:godot // Legacy comments are kept terse in this file.
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"ai-gateway/internal/config"
)

// IPRateLimiter tracks rate limiters per IP
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter creates a new IP rate limiter
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   r,
		b:   b,
	}
}

// GetLimiter returns the rate limiter for a given IP
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

// CleanupOldLimiters removes limiters that haven't been used recently
func (i *IPRateLimiter) CleanupOldLimiters() {
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C {
			i.mu.Lock()
			// Simple cleanup - in production, track last access time
			if len(i.ips) > 10000 {
				i.ips = make(map[string]*rate.Limiter)
			}
			i.mu.Unlock()
		}
	}()
}

var globalLimiter *IPRateLimiter

// RateLimiter returns a gin middleware for rate limiting
func RateLimiter(cfg config.LimiterConfig) gin.HandlerFunc {
	// If rate limiting is disabled, return a no-op middleware
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	if globalLimiter == nil {
		globalLimiter = NewIPRateLimiter(
			rate.Limit(cfg.Rate),
			cfg.Burst,
		)
		globalLimiter.CleanupOldLimiters()
	}

	return func(c *gin.Context) {
		var limiter *rate.Limiter

		if cfg.PerUser {
			// Use API key or IP for rate limiting
			apiKey := c.GetHeader("X-API-Key")
			if apiKey != "" {
				limiter = globalLimiter.GetLimiter(apiKey)
			} else {
				limiter = globalLimiter.GetLimiter(c.ClientIP())
			}
		} else {
			limiter = globalLimiter.GetLimiter("global")
		}

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
