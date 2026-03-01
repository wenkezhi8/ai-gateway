package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newTestTraceDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE request_traces (
			id TEXT PRIMARY KEY,
			request_id TEXT NOT NULL,
			trace_id TEXT NOT NULL,
			span_id TEXT NOT NULL,
			parent_span_id TEXT,
			operation TEXT NOT NULL,
			status TEXT NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			attributes TEXT,
			events TEXT,
			user_id TEXT,
			method TEXT,
			path TEXT,
			model TEXT,
			provider TEXT,
			error TEXT,
			created_at TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	return db
}

func waitForTraceRow(t *testing.T, db *sql.DB, operation string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		err := db.QueryRow(`SELECT COUNT(1) FROM request_traces WHERE operation = ?`, operation).Scan(&count)
		if err == nil && count > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("trace row for operation %s not found in time", operation)
}

func TestRecordSimpleSpan_UsesProvidedDurationMs(t *testing.T) {
	db := newTestTraceDB(t)
	defer db.Close()

	recorder := NewSpanRecorder(db)
	ctx := SetRequestIDToContext(context.Background(), "req-1")

	recorder.RecordSimpleSpan(ctx, "cache.read-v2", map[string]interface{}{
		"duration_ms": int64(123),
		"model":       "ark-code-latest",
	})

	waitForTraceRow(t, db, "cache.read-v2")

	var duration int64
	err := db.QueryRow(`SELECT duration_ms FROM request_traces WHERE operation = ?`, "cache.read-v2").Scan(&duration)
	if err != nil {
		t.Fatalf("query duration: %v", err)
	}

	if duration != 123 {
		t.Fatalf("duration_ms = %d, want 123", duration)
	}
}

func TestRecordSpanWithResult_WritesResultAttribute(t *testing.T) {
	db := newTestTraceDB(t)
	defer db.Close()

	recorder := NewSpanRecorder(db)
	ctx := SetRequestIDToContext(context.Background(), "req-2")

	recorder.RecordSpanWithResult(ctx, "cache.read-exact", "hit", map[string]interface{}{
		"cache_key": "k1",
	})

	waitForTraceRow(t, db, "cache.read-exact")

	var attrsRaw string
	err := db.QueryRow(`SELECT attributes FROM request_traces WHERE operation = ?`, "cache.read-exact").Scan(&attrsRaw)
	if err != nil {
		t.Fatalf("query attrs: %v", err)
	}

	attrs := map[string]interface{}{}
	if err := json.Unmarshal([]byte(attrsRaw), &attrs); err != nil {
		t.Fatalf("unmarshal attrs: %v", err)
	}

	if attrs["result"] != "hit" {
		t.Fatalf("result attr = %v, want hit", attrs["result"])
	}
}

func TestExtractResponseTextPreview_OpenAIResponse(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"这是一个很长的命中答案"}}]}`)
	preview, full, truncated := ExtractResponseTextPreview(body, 5, 50)

	if preview != "这是一个很" {
		t.Fatalf("preview = %q", preview)
	}
	if full != "这是一个很长的命中答案" {
		t.Fatalf("full = %q", full)
	}
	if truncated {
		t.Fatalf("truncated should be false")
	}
}

func TestExtractResponseTextPreview_FallbackPlainBody(t *testing.T) {
	body := []byte("plain text body")
	preview, full, truncated := ExtractResponseTextPreview(body, 5, 8)

	if preview != "plain" {
		t.Fatalf("preview = %q", preview)
	}
	if full != "plain te" {
		t.Fatalf("full = %q", full)
	}
	if !truncated {
		t.Fatalf("truncated should be true")
	}
}
