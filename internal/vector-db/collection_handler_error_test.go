package vectordb

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCollectionHandler_RespondServiceError_ShouldMapStatusCodes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{}))

	tests := []struct {
		name string
		err  error
		code int
	}{
		{name: "nil", err: nil, code: http.StatusInternalServerError},
		{name: "collection not found", err: ErrCollectionNotFound, code: http.StatusNotFound},
		{name: "collection exists", err: ErrCollectionExists, code: http.StatusConflict},
		{name: "import job not found", err: ErrImportJobNotFound, code: http.StatusNotFound},
		{name: "import retry exceeded", err: ErrImportJobRetryExceeded, code: http.StatusBadRequest},
		{name: "alert rule not found", err: ErrAlertRuleNotFound, code: http.StatusNotFound},
		{name: "api key not found", err: ErrVectorAPIKeyNotFound, code: http.StatusForbidden},
		{name: "backup not found", err: ErrBackupTaskNotFound, code: http.StatusNotFound},
		{name: "backend unavailable", err: ErrBackendUnavailable, code: http.StatusServiceUnavailable},
		{name: "text search unsupported", err: ErrTextSearchNotSupported, code: http.StatusBadRequest},
		{name: "dependency missing", err: errors.New("repository is required"), code: http.StatusInternalServerError},
		{name: "required message", err: fmt.Errorf("name is required") /*nolint:goerr113*/, code: http.StatusBadRequest},
		{name: "positive message", err: fmt.Errorf("id must be positive") /*nolint:goerr113*/, code: http.StatusBadRequest},
		{name: "allowed message", err: fmt.Errorf("index_type only allowed") /*nolint:goerr113*/, code: http.StatusBadRequest},
		{name: "default", err: errors.New("unexpected failure"), code: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			h.respondServiceError(c, tc.err)
			if w.Code != tc.code {
				t.Fatalf("status=%d, want %d", w.Code, tc.code)
			}
		})
	}
}

func TestCollectionHandler_HelperBranches_ShouldHandleNilAndFallbacks(t *testing.T) {
	t.Parallel()

	if h := NewCollectionHandler(nil); h == nil {
		t.Fatal("NewCollectionHandler(nil) should not return nil")
	}

	var nilHandler *CollectionHandler
	if svc := nilHandler.RBACService(); svc != nil {
		t.Fatalf("RBACService(nil handler) = %v, want nil", svc)
	}

	if got := parseIntDefault("bad", 7); got != 7 {
		t.Fatalf("parseIntDefault(invalid)=%d, want 7", got)
	}
}
