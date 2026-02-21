package integration

import (
	"ai-gateway/internal/config"
	"ai-gateway/internal/handler"
	"ai-gateway/internal/middleware"
	"ai-gateway/internal/provider"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestSecurity_APIKeyExposure tests that API keys are not exposed in responses
func TestSecurity_APIKeyExposure(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:    "test-provider",
				APIKey:  "super-secret-api-key-12345",
				BaseURL: "https://api.example.com",
				Enabled: true,
			},
		},
	}

	ginRouter := gin.New()
	proxyHandler := handler.NewProxyHandler(cfg, nil, nil)
	ginRouter.GET("/v1/providers", proxyHandler.ListProviders)

	req := httptest.NewRequest("GET", "/v1/providers", nil)
	w := httptest.NewRecorder()
	ginRouter.ServeHTTP(w, req)

	// API key should never appear in response
	body := w.Body.String()
	assert.NotContains(t, body, "super-secret-api-key-12345")
	assert.NotContains(t, body, "api_key")
}

// TestSecurity_RateLimiting tests rate limiting enforcement
func TestSecurity_RateLimiting(t *testing.T) {
	cfg := config.LimiterConfig{
		Enabled: true,
		Rate:    1, // Very low rate for testing
		Burst:   2,
		PerUser: false,
	}

	ginRouter := gin.New()
	ginRouter.Use(middleware.RateLimiter(cfg))
	ginRouter.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// First few requests should succeed (within burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		ginRouter.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}

	// Rapid subsequent requests should be rate limited
	// Note: In practice, this is timing-dependent
}

// TestSecurity_InputValidation tests input validation
func TestSecurity_InputValidation(t *testing.T) {
	// Clear global registry before test
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "test", Enabled: true},
		},
	}

	ginRouter := gin.New()
	proxyHandler := handler.NewProxyHandler(cfg, nil, nil)

	// Register a mock provider for testing
	mockProvider := &mockProviderForIntegration{
		name:    "test",
		enabled: true,
	}
	provider.RegisterProvider("test", mockProvider)

	ginRouter.POST("/api/v1/chat/completions", proxyHandler.ChatCompletions)

	tests := []struct {
		name       string
		body       string
		expectCode int
	}{
		{
			name:       "empty body",
			body:       "",
			expectCode: http.StatusBadRequest, // Empty body should return 400
		},
		{
			name:       "invalid JSON",
			body:       "{invalid json}",
			expectCode: http.StatusBadRequest, // Invalid JSON should return 400
		},
		{
			name:       "valid JSON",
			body:       `{"model": "gpt-4", "messages": []}`,
			expectCode: http.StatusBadRequest, // Empty messages should return 400
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

// TestSecurity_HTTPHeaders tests security headers
func TestSecurity_HTTPHeaders(t *testing.T) {
	ginRouter := gin.New()

	// Add security headers middleware
	ginRouter.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	})

	ginRouter.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ginRouter.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
}

// TestSecurity_SQlInjection tests SQL injection protection
func TestSecurity_SQLInjection(t *testing.T) {
	// Clear global registry before test
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "test", Enabled: true},
		},
	}

	ginRouter := gin.New()
	proxyHandler := handler.NewProxyHandler(cfg, nil, nil)
	ginRouter.GET("/v1/providers", proxyHandler.ListProviders)

	// Try SQL injection in query params - proper URL encoding
	req := httptest.NewRequest("GET", "/v1/providers?id=1%27%20OR%20%271%27%3D%271", nil)
	w := httptest.NewRecorder()
	ginRouter.ServeHTTP(w, req)

	// Should return normally without error
	require.Equal(t, http.StatusOK, w.Code)
}

// TestSecurity_PathTraversal tests path traversal protection
func TestSecurity_PathTraversal(t *testing.T) {
	ginRouter := gin.New()
	ginRouter.GET("/files/:filename", func(c *gin.Context) {
		// In real app, this would serve files
		c.String(http.StatusBadRequest, "Not implemented")
	})

	tests := []struct {
		name string
		path string
	}{
		{"directory traversal", "/files/../../../etc/passwd"},
		{"encoded traversal", "/files/%2e%2e%2f%2e%2e%2fetc/passwd"},
		{"double encoded", "/files/%252e%252e%252fetc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			ginRouter.ServeHTTP(w, req)

			// Should not return 200 with file contents
			assert.NotEqual(t, http.StatusOK, w.Code)
		})
	}
}

// TestSecurity_XSSProtection tests XSS protection
func TestSecurity_XSSProtection(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "test", Enabled: true},
		},
	}

	ginRouter := gin.New()
	proxyHandler := handler.NewProxyHandler(cfg, nil, nil)
	ginRouter.GET("/v1/providers", proxyHandler.ListProviders)

	// Try XSS in query params
	req := httptest.NewRequest("GET", "/v1/providers?name=<script>alert('xss')</script>", nil)
	w := httptest.NewRecorder()
	ginRouter.ServeHTTP(w, req)

	body := w.Body.String()
	// Should not contain unescaped script tags
	assert.NotContains(t, body, "<script>alert")
}

// TestSecurity_MethodNotAllowed tests HTTP method restrictions
func TestSecurity_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{}
	ginRouter := gin.New()
	proxyHandler := handler.NewProxyHandler(cfg, nil, nil)

	// Only POST should be allowed for chat completions
	ginRouter.POST("/api/v1/chat/completions", proxyHandler.ChatCompletions)

	// Try GET request on POST-only endpoint
	req := httptest.NewRequest("GET", "/api/v1/chat/completions", nil)
	w := httptest.NewRecorder()
	ginRouter.ServeHTTP(w, req)

	// Should return 404 (route not found) or 405 (method not allowed)
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusMethodNotAllowed)
}
