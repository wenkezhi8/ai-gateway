package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled  bool
	APIKeys  map[string]string // API Key -> User ID mapping
	Optional bool              // If true, allow requests without API key
}

// unauthorized returns a 401 response
func unauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":    "unauthorized",
			"message": message,
		},
	})
}

// Auth returns a gin middleware for API key authentication
func Auth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		// Get API key from header
		apiKey := extractAPIKey(c)
		if apiKey == "" {
			if cfg.Optional {
				c.Next()
				return
			}
			unauthorized(c, "API key is required")
			return
		}

		// Validate API key
		userID, valid := cfg.APIKeys[apiKey]
		if !valid {
			unauthorized(c, "Invalid API key")
			return
		}

		// Store user info in context
		c.Set("api_key", apiKey)
		c.Set("user_id", userID)
		c.Next()
	}
}

// extractAPIKey extracts API key from request
// Note: URL query parameter support has been removed for security reasons
// (API keys in URLs can be logged in access logs and browser history)
func extractAPIKey(c *gin.Context) string {
	// Check Authorization header (Bearer token)
	auth := c.GetHeader("Authorization")
	if auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}

	// Check X-API-Key header
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// URL query parameter support removed for security
	// API keys should only be passed via headers to prevent
	// exposure in logs, browser history, and referrer headers

	return ""
}

// GetUserID gets user ID from context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return ""
}

// GetAPIKey gets API key from context
func GetAPIKey(c *gin.Context) string {
	if apiKey, exists := c.Get("api_key"); exists {
		return apiKey.(string)
	}
	return ""
}

// RequireAuth is a middleware that requires authentication
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "unauthorized",
					"message": "Authentication required",
				},
			})
			return
		}
		c.Next()
	}
}
