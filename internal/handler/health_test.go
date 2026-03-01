package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthHandler_CheckReady_Ready(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(func() error { return nil })
	r := gin.New()
	r.GET("/ready", h.CheckReady)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthHandler_CheckReady_NotReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(func() error { return errors.New("cache unavailable") })
	r := gin.New()
	r.GET("/ready", h.CheckReady)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
