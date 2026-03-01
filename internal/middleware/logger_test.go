package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLogger_RecordsRequest(t *testing.T) {
	r := gin.New()
	r.Use(Logger())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("User-Agent", "test-agent")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_RecordsPostRequest(t *testing.T) {
	r := gin.New()
	r.Use(Logger())
	r.POST("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_RecordsErrorStatus(t *testing.T) {
	r := gin.New()
	r.Use(Logger())
	r.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "Error")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogger_MultipleRequests(t *testing.T) {
	r := gin.New()
	r.Use(Logger())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Make multiple requests
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", http.NoBody)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
