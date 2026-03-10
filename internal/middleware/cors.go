//nolint:godot // Legacy comments are kept terse in this file.
package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a gin middleware for CORS handling
func CORS() gin.HandlerFunc {
	allowedOrigins, allowAllOrigins := loadAllowedOrigins()

	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			if allowAllOrigins {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
			} else if isAllowedOrigin(origin, allowedOrigins) {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
			} else {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
				return
			}
		} else if allowAllOrigins {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func loadAllowedOrigins() (map[string]struct{}, bool) {
	envValue := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if envValue == "" || envValue == "*" {
		return nil, true
	}

	allowedOrigins := make(map[string]struct{})
	for _, item := range strings.Split(envValue, ",") {
		origin := strings.TrimSpace(item)
		if origin == "" {
			continue
		}
		if origin == "*" {
			return nil, true
		}
		allowedOrigins[origin] = struct{}{}
	}

	if len(allowedOrigins) == 0 {
		return nil, true
	}
	return allowedOrigins, false
}

func isAllowedOrigin(origin string, allowed map[string]struct{}) bool {
	_, ok := allowed[origin]
	return ok
}
