package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
)

func TestTraceHandler_GetTraces_RequestLevelPagination_ShouldReturnDistinctTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTraceTestDB(t)
	handler := NewTraceHandler(db)
	r := gin.New()
	r.GET("/admin/traces", handler.GetTraces)

	base := time.Now().UTC().Add(-10 * time.Minute)
	insertTraceSpan(t, db, "req-a", "http.entry", "success", "GET", 20, base, nil)
	insertTraceSpan(t, db, "req-a", "cache.read-exact", "success", "GET", 12, base.Add(1*time.Second), map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-a", "http.response", "success", "GET", 180, base.Add(2*time.Second), nil)

	insertTraceSpan(t, db, "req-b", "http.entry", "success", "POST", 25, base.Add(3*time.Second), nil)
	insertTraceSpan(t, db, "req-b", "provider.chat", "success", "POST", 300, base.Add(4*time.Second), nil)
	insertTraceSpan(t, db, "req-b", "http.response", "success", "POST", 360, base.Add(5*time.Second), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/traces?limit=1&offset=0", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Total   int  `json:"total"`
		Data    []struct {
			RequestID string `json:"request_id"`
			StepCount int    `json:"step_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("success=false")
	}
	if resp.Total != 2 {
		t.Fatalf("total=%d want=2", resp.Total)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len(data)=%d want=1", len(resp.Data))
	}
	if resp.Data[0].RequestID == "" {
		t.Fatalf("request_id empty")
	}
	if resp.Data[0].StepCount <= 0 {
		t.Fatalf("step_count=%d want>0", resp.Data[0].StepCount)
	}
}

func TestTraceHandler_GetTraces_AnswerSource_ShouldFollowPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTraceTestDB(t)
	handler := NewTraceHandler(db)
	r := gin.New()
	r.GET("/admin/traces", handler.GetTraces)

	base := time.Now().UTC().Add(-20 * time.Minute)

	insertTraceSpan(t, db, "req-v2", "cache.read-v2", "success", "GET", 10, base, map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-v2", "http.response", "success", "GET", 120, base.Add(1*time.Second), map[string]any{"cache_layer": "v2"})

	insertTraceSpan(t, db, "req-sem", "cache.read-semantic", "success", "GET", 10, base.Add(2*time.Second), map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-sem", "cache.read-exact", "success", "GET", 8, base.Add(3*time.Second), map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-sem", "http.response", "success", "GET", 118, base.Add(4*time.Second), map[string]any{"cache_layer": "semantic"})

	insertTraceSpan(t, db, "req-exact-raw", "cache.read-exact", "success", "GET", 8, base.Add(5*time.Second), map[string]any{"result": "hit", "cache_layer": "exact_raw"})
	insertTraceSpan(t, db, "req-exact-raw", "http.response", "success", "GET", 108, base.Add(6*time.Second), map[string]any{"cache_layer": "exact_raw"})

	insertTraceSpan(t, db, "req-exact-prompt", "cache.read-exact", "success", "GET", 8, base.Add(7*time.Second), map[string]any{"result": "hit", "cache_layer": "exact_prompt"})
	insertTraceSpan(t, db, "req-exact-prompt", "http.response", "success", "GET", 109, base.Add(8*time.Second), map[string]any{"cache_layer": "exact_prompt"})

	insertTraceSpan(t, db, "req-legacy-exact", "cache.read-exact", "success", "GET", 8, base.Add(9*time.Second), map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-legacy-exact", "http.response", "success", "GET", 110, base.Add(10*time.Second), map[string]any{"cache_layer": "exact"})

	insertTraceSpan(t, db, "req-legacy-vector", "cache.read-v2", "success", "GET", 11, base.Add(11*time.Second), map[string]any{"result": "hit", "layer": "vector-semantic"})
	insertTraceSpan(t, db, "req-legacy-vector", "http.response", "success", "GET", 111, base.Add(12*time.Second), map[string]any{"cache_layer": "vector-semantic"})

	insertTraceSpan(t, db, "req-provider", "provider.chat", "success", "GET", 210, base.Add(13*time.Second), nil)

	insertTraceSpan(t, db, "req-task-classifier", "classifier.assess", "success", "GET", 12, base.Add(14*time.Second), map[string]any{"task_type": "analysis"})
	insertTraceSpan(t, db, "req-task-classifier", "provider.chat", "success", "GET", 220, base.Add(15*time.Second), nil)

	insertTraceSpan(t, db, "req-task-v2", "cache.read-v2", "success", "GET", 18, base.Add(16*time.Second), map[string]any{"result": "hit", "task_type": "chat"})

	insertTraceSpan(t, db, "req-unknown", "provider.chat", "error", "GET", 90, base.Add(17*time.Second), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/traces?limit=20&offset=0", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    []struct {
			RequestID    string `json:"request_id"`
			AnswerSource string `json:"answer_source"`
			TaskType     string `json:"task_type"`
			Model        string `json:"model"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("success=false")
	}

	sourceByRequest := map[string]string{}
	taskTypeByRequest := map[string]string{}
	for _, row := range resp.Data {
		sourceByRequest[row.RequestID] = row.AnswerSource
		taskTypeByRequest[row.RequestID] = row.TaskType
	}

	assertTraceSource(t, sourceByRequest, "req-v2", "v2")
	assertTraceSource(t, sourceByRequest, "req-sem", "semantic")
	assertTraceSource(t, sourceByRequest, "req-exact-raw", "exact_raw")
	assertTraceSource(t, sourceByRequest, "req-exact-prompt", "exact_prompt")
	assertTraceSource(t, sourceByRequest, "req-legacy-exact", "exact_prompt")
	assertTraceSource(t, sourceByRequest, "req-legacy-vector", "v2")
	assertTraceSource(t, sourceByRequest, "req-provider", "provider_chat")
	assertTraceSource(t, sourceByRequest, "req-unknown", "provider_chat")

	assertTraceTaskType(t, taskTypeByRequest, "req-task-classifier", "analysis")
	assertTraceTaskType(t, taskTypeByRequest, "req-task-v2", "chat")

	for _, row := range resp.Data {
		if row.Model != "gpt-4o-mini" {
			t.Fatalf("request %s model=%s want=gpt-4o-mini", row.RequestID, row.Model)
		}
	}
}

func TestTraceHandler_ClearTraces_ShouldDeleteAllAndReturnDeleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTraceTestDB(t)
	handler := NewTraceHandler(db)
	r := gin.New()
	r.GET("/admin/traces", handler.GetTraces)
	r.DELETE("/admin/traces", handler.ClearTraces)

	base := time.Now().UTC().Add(-5 * time.Minute)
	insertTraceSpan(t, db, "req-clear-a", "http.entry", "success", "GET", 5, base, nil)
	insertTraceSpan(t, db, "req-clear-b", "provider.chat", "success", "POST", 18, base.Add(1*time.Second), nil)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/admin/traces", http.NoBody)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteW.Code, deleteW.Body.String())
	}

	var deleteResp struct {
		Success bool `json:"success"`
		Data    struct {
			Deleted int64 `json:"deleted"`
		} `json:"data"`
	}
	if err := json.Unmarshal(deleteW.Body.Bytes(), &deleteResp); err != nil {
		t.Fatalf("decode delete response failed: %v", err)
	}

	if !deleteResp.Success {
		t.Fatalf("delete success=false")
	}
	if deleteResp.Data.Deleted != 2 {
		t.Fatalf("deleted=%d want=2", deleteResp.Data.Deleted)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/traces?limit=20&offset=0", http.NoBody)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listW.Code, listW.Body.String())
	}

	var listResp struct {
		Success bool          `json:"success"`
		Total   int           `json:"total"`
		Data    []interface{} `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}

	if !listResp.Success {
		t.Fatalf("list success=false")
	}
	if listResp.Total != 0 {
		t.Fatalf("total=%d want=0", listResp.Total)
	}
	if len(listResp.Data) != 0 {
		t.Fatalf("len(data)=%d want=0", len(listResp.Data))
	}
}

func newTraceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	store, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "trace-handler-test.db"))
	if err != nil {
		t.Fatalf("new sqlite storage failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store.GetDB()
}

func insertTraceSpan(
	t *testing.T,
	db *sql.DB,
	requestID, operation, status, method string,
	durationMs int64,
	createdAt time.Time,
	attrs map[string]any,
) {
	t.Helper()

	attrBytes := []byte("{}")
	if attrs != nil {
		b, err := json.Marshal(attrs)
		if err != nil {
			t.Fatalf("marshal attrs failed: %v", err)
		}
		attrBytes = b
	}

	ts := createdAt.Format(time.RFC3339Nano)
	_, err := db.Exec(`
		INSERT INTO request_traces (
			id, request_id, trace_id, span_id, parent_span_id, operation, status,
			start_time, end_time, duration_ms, attributes, events,
			user_id, method, path, model, provider, error, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		fmt.Sprintf("%s-%s-%d", requestID, operation, createdAt.UnixNano()),
		requestID,
		fmt.Sprintf("trace-%s", requestID),
		fmt.Sprintf("span-%s-%d", operation, createdAt.UnixNano()),
		"",
		operation,
		status,
		ts,
		ts,
		durationMs,
		string(attrBytes),
		"{}",
		"",
		method,
		"/v1/chat",
		"gpt-4o-mini",
		"openai",
		"",
		ts,
	)
	if err != nil {
		t.Fatalf("insert trace span failed: %v", err)
	}
}

func assertTraceSource(t *testing.T, sourceByRequest map[string]string, requestID, want string) {
	t.Helper()
	got, ok := sourceByRequest[requestID]
	if !ok {
		t.Fatalf("request %s missing in response", requestID)
	}
	if got != want {
		t.Fatalf("request %s answer_source=%s want=%s", requestID, got, want)
	}
}

func assertTraceTaskType(t *testing.T, taskTypeByRequest map[string]string, requestID, want string) {
	t.Helper()
	got, ok := taskTypeByRequest[requestID]
	if !ok {
		t.Fatalf("request %s missing in response", requestID)
	}
	if got != want {
		t.Fatalf("request %s task_type=%s want=%s", requestID, got, want)
	}
}
