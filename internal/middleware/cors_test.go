package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORS_DefaultHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestCORS_OptionsRequest(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.POST("/api", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api", http.NoBody)
	r.ServeHTTP(w, req)

	// OPTIONS should return 204
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCORS_PostRequest(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.POST("/api", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_AllowedHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	allowedHeaders := w.Header().Get("Access-Control-Allow-Headers")

	// Check all expected headers are present
	expectedHeaders := []string{
		"Content-Type",
		"Authorization",
		"X-API-Key",
		"X-Requested-With",
		"Cache-Control",
	}

	for _, header := range expectedHeaders {
		assert.Contains(t, allowedHeaders, header, "Expected header %s to be allowed", header)
	}
}

func TestCORS_AllowedMethods(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	allowedMethods := w.Header().Get("Access-Control-Allow-Methods")

	// Check all expected methods are present
	expectedMethods := []string{
		"POST",
		"GET",
		"PUT",
		"DELETE",
		"OPTIONS",
	}

	for _, method := range expectedMethods {
		assert.Contains(t, allowedMethods, method, "Expected method %s to be allowed", method)
	}
}
