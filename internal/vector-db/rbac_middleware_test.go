package vectordb

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRBACMiddleware_WhenForbidden_ShouldReturn403(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo := &mockRepo{}
	rbacService := NewRBACService(repo)
	middleware := NewRBACMiddleware(rbacService)

	searchHandler := NewSearchHandler(NewServiceWithDeps(repo, &mockSearchBackend{}))
	api := r.Group("/api/v1")
	RegisterVectorSearchRoutesWithRBAC(api, searchHandler, middleware.Middleware())

	body := []byte(`{"top_k": 2, "vector": [0.2, 0.4]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}
