package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouterHandler_DeleteOllamaModel_ShouldRejectEmptyModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &RouterHandler{}
	r := gin.New()
	r.POST("/api/admin/router/ollama/delete", h.DeleteOllamaModel)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/router/ollama/delete", bytes.NewBufferString(`{"model":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
