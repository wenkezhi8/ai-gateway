package intent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIntentClient_Infer_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/intent-embed" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"intent":"calc","slots":{"expr":"1+1"},"standard_key":"intent:calc:expr=1+1","embedding":[0.1,0.2],"embedding_dim":2,"confidence":0.99,"engine_version":"v1"}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		Enabled:           true,
		BaseURL:           srv.URL,
		Timeout:           2 * time.Second,
		Language:          "zh-CN",
		ExpectedDimension: 2,
	})

	resp, err := client.Infer(context.Background(), "帮我算1+1", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Intent != "calc" {
		t.Fatalf("expected calc intent, got %q", resp.Intent)
	}
}

func TestIntentClient_Infer_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"intent":`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		Enabled: true,
		BaseURL: srv.URL,
		Timeout: 2 * time.Second,
	})

	if _, err := client.Infer(context.Background(), "hello", ""); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestIntentClient_Infer_DimensionMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"intent":"qa","slots":{},"standard_key":"intent:qa","embedding":[0.1,0.2,0.3],"embedding_dim":3,"confidence":0.9,"engine_version":"v1"}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		Enabled:           true,
		BaseURL:           srv.URL,
		Timeout:           2 * time.Second,
		ExpectedDimension: 2,
	})

	if _, err := client.Infer(context.Background(), "hello", ""); err == nil {
		t.Fatal("expected dimension mismatch error")
	}
}

func TestIntentClient_Infer_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"intent":"qa","slots":{},"standard_key":"intent:qa","embedding":[0.1],"embedding_dim":1,"confidence":0.9,"engine_version":"v1"}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		Enabled: true,
		BaseURL: srv.URL,
		Timeout: 30 * time.Millisecond,
	})

	if _, err := client.Infer(context.Background(), "hello", ""); err == nil {
		t.Fatal("expected timeout error")
	}
}

