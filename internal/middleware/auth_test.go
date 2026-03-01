package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_Disabled(t *testing.T) {
	cfg := AuthConfig{
		Enabled: false,
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_Enabled_ValidBearer(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{
			"test-key": "user-123",
		},
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.String(http.StatusOK, userID)
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-123", w.Body.String())
}

func TestAuth_Enabled_ValidAPIKeyHeader(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{
			"test-key": "user-456",
		},
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.String(http.StatusOK, userID)
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-456", w.Body.String())
}

// TestAuth_Enabled_QueryParamNotSupported tests that query param auth is not supported for security.
func TestAuth_Enabled_QueryParamNotSupported(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{
			"test-key": "user-789",
		},
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.String(http.StatusOK, userID)
	})

	// Query parameter authentication is not supported for security reasons
	req := httptest.NewRequest("GET", "/test?api_key=test-key", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 401 because query param is not supported
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_Enabled_NoAPIKey(t *testing.T) {
	cfg := AuthConfig{
		Enabled:  true,
		Optional: false,
		APIKeys:  map[string]string{},
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_Enabled_OptionalNoAPIKey(t *testing.T) {
	cfg := AuthConfig{
		Enabled:  true,
		Optional: true,
		APIKeys:  map[string]string{},
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_Enabled_InvalidAPIKey(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{
			"valid-key": "user-123",
		},
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer invalid-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_GetUserID(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		userID := GetUserID(c)
		c.String(http.StatusOK, userID)
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "test-user", w.Body.String())
}

func TestAuth_GetUserID_NotSet(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.String(http.StatusOK, userID)
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "", w.Body.String())
}

func TestAuth_GetAPIKey(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.Set("api_key", "test-api-key")
		apiKey := GetAPIKey(c)
		c.String(http.StatusOK, apiKey)
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "test-api-key", w.Body.String())
}

func TestRequireAuth(t *testing.T) {
	router := gin.New()

	// Middleware that sets user_id
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "authenticated-user")
		c.Next()
	})
	router.Use(RequireAuth())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAuth_Unauthorized(t *testing.T) {
	router := gin.New()
	router.Use(RequireAuth())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_AuthorizationWithoutBearer(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{
			"plain-key": "user-123",
		},
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.String(http.StatusOK, userID)
	})

	// Authorization header without "Bearer " prefix
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "plain-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-123", w.Body.String())
}

func TestAuth_Priority(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{
			"bearer-key": "bearer-user",
			"header-key": "header-user",
		},
	}

	router := gin.New()
	router.Use(Auth(cfg))
	router.GET("/test", func(c *gin.Context) {
		apiKey := GetAPIKey(c)
		c.String(http.StatusOK, apiKey)
	})

	// Authorization header should take priority over X-API-Key
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer bearer-key")
	req.Header.Set("X-API-Key", "header-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "bearer-key", w.Body.String())
}
