package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSwitchClassifierModelAsync_CreateTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := newClassifierSwitchTaskStore(filepath.Join(t.TempDir(), "switch-task.db"))
	if err != nil {
		t.Fatalf("create switch task store failed: %v", err)
	}
	defer store.Close()

	handler := &RouterHandler{switchTaskStore: store}
	router := gin.New()
	router.POST("/api/admin/router/classifier/switch-async", handler.SwitchClassifierModelAsync)

	reqBody := map[string]string{"model": "qwen3:4b"}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request body failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/router/classifier/switch-async", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("success = false, want true")
	}

	if resp.Data.TaskID == "" {
		t.Fatalf("task_id is empty")
	}
}
