package middleware

import (
	"ai-gateway/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestIPRateLimiter_New(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Limit(100), 200)
	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.ips)
}

func TestIPRateLimiter_GetLimiter_NewIP(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Limit(100), 200)

	l1 := limiter.GetLimiter("192.168.1.1")
	assert.NotNil(t, l1)

	// Same IP should return same limiter
	l2 := limiter.GetLimiter("192.168.1.1")
	assert.Equal(t, l1, l2)

	// Different IP - verify it returns a functional limiter
	l3 := limiter.GetLimiter("192.168.1.2")
	assert.NotNil(t, l3)
	// Both limiters should be functional
	assert.True(t, l1.Allow())
	assert.True(t, l3.Allow())
}

func TestIPRateLimiter_GetLimiter_Concurrent(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Limit(100), 200)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			ip := string(rune('0' + id))
			for j := 0; j < 100; j++ {
				_ = limiter.GetLimiter(ip)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have created limiters for each unique IP
	limiter.mu.Lock()
	count := len(limiter.ips)
	limiter.mu.Unlock()

	assert.Equal(t, 10, count)
}

func TestRateLimiter_Middleware_Allowed(t *testing.T) {
	// Reset global limiter
	globalLimiter = nil

	cfg := config.LimiterConfig{
		Enabled: true,
		Rate:    100,
		Burst:   200,
		PerUser: false,
	}

	router := gin.New()
	router.Use(RateLimiter(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Middleware_PerUser(t *testing.T) {
	// Reset global limiter
	globalLimiter = nil

	cfg := config.LimiterConfig{
		Enabled: true,
		Rate:    100,
		Burst:   200,
		PerUser: true,
	}

	router := gin.New()
	router.Use(RateLimiter(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Request with API key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Middleware_UsesClientIP(t *testing.T) {
	// Reset global limiter
	globalLimiter = nil

	cfg := config.LimiterConfig{
		Enabled: true,
		Rate:    100,
		Burst:   200,
		PerUser: true,
	}

	router := gin.New()
	router.Use(RateLimiter(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Request without API key - should use ClientIP
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Disabled(t *testing.T) {
	globalLimiter = nil

	cfg := config.LimiterConfig{
		Enabled: false,
	}

	router := gin.New()
	// Even when disabled, the middleware still works but creates a default limiter
	router.Use(RateLimiter(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still work
	require.Equal(t, http.StatusOK, w.Code)
}
