package vectordb

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type VectorSearchRateLimiter struct {
	maxRequests int
	window      time.Duration

	mu       sync.Mutex
	counters map[string]*rateCounter
}

type rateCounter struct {
	count   int
	resetAt time.Time
}

func NewVectorSearchRateLimiter(maxRequests int, window time.Duration) *VectorSearchRateLimiter {
	if maxRequests <= 0 {
		maxRequests = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &VectorSearchRateLimiter{
		maxRequests: maxRequests,
		window:      window,
		counters:    make(map[string]*rateCounter),
	}
}

func (l *VectorSearchRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil {
			c.Next()
			return
		}

		key := l.requestKey(c)
		allowed, retryAfter := l.allowNow(key)
		if allowed {
			c.Next()
			return
		}

		c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
		c.JSON(http.StatusTooManyRequests, gin.H{"success": false, "error": "rate limit exceeded"})
		c.Abort()
	}
}

func (l *VectorSearchRateLimiter) requestKey(c *gin.Context) string {
	apiKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
	if apiKey == "" {
		authorization := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			apiKey = strings.TrimSpace(authorization[len("Bearer "):])
		}
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.ClientIP())
	}
	collection := strings.TrimSpace(c.Param("name"))
	if collection == "" {
		collection = "unknown"
	}
	return apiKey + ":" + collection
}

func (l *VectorSearchRateLimiter) allowNow(key string) (allowed bool, retryAfter int64) {
	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	counter, ok := l.counters[key]
	if !ok || now.After(counter.resetAt) {
		l.counters[key] = &rateCounter{count: 1, resetAt: now.Add(l.window)}
		return true, 0
	}
	if counter.count >= l.maxRequests {
		retryAfter := int64(time.Until(counter.resetAt).Seconds())
		if retryAfter <= 0 {
			retryAfter = 1
		}
		return false, retryAfter
	}

	counter.count++
	return true, 0
}
