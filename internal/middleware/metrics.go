package middleware

import (
	"ai-gateway/internal/metrics"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware returns a Gin middleware that records Prometheus metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if metrics.GetMetrics() == nil {
			c.Next()
			return
		}

		m := metrics.GetMetrics()
		start := time.Now()
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Track active connections
		m.IncActiveConnections(endpoint)
		defer m.DecActiveConnections(endpoint)

		// Process request
		c.Next()

		// Record metrics after request completes
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method

		m.RecordRequest(method, endpoint, statusCode, duration)
	}
}
