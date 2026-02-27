package middleware

import (
	"ai-gateway/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var log = logger.WithField("component", "middleware")

// Logger returns a gin middleware for logging
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		log.WithFields(logrus.Fields{
			"status":     statusCode,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"latency":    latency,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("HTTP Request")
	}
}
