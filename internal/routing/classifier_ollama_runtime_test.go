package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaChatRequest_KeepAliveDisabledUnload(t *testing.T) {
	t.Helper()

	var keepAlive string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req struct {
			KeepAlive string `json:"keep_alive"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		keepAlive = req.KeepAlive
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"{}"}}`))
	}))
	defer server.Close()

	classifier := NewOllamaTaskClassifier(ClassifierConfig{
		Provider:    "ollama",
		BaseURL:     server.URL,
		ActiveModel: "qwen3:4b",
		TimeoutMs:   2000,
	})

	_, _, err := classifier.chat(context.Background(), "qwen3:4b", "test")
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if keepAlive != "-1" {
		t.Fatalf("keep_alive = %q, want %q", keepAlive, "-1")
	}
}

func TestListOllamaRunningModels_FromPS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ps" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:4b"},{"model":"qwen2.5:0.5b-instruct"}]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	models, err := ListOllamaRunningModels(ctx, server.URL, time.Second)
	if err != nil {
		t.Fatalf("ListOllamaRunningModels failed: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("running models len = %d, want 2", len(models))
	}
	if models[0] != "qwen2.5:0.5b-instruct" || models[1] != "qwen3:4b" {
		t.Fatalf("unexpected running models: %#v", models)
	}
}

func TestListOllamaRunningModelDetails_FromPS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ps" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:4b","size_vram":2147483648},{"model":"qwen2.5:0.5b-instruct","size_vram":536870912}]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	details, err := ListOllamaRunningModelDetails(ctx, server.URL, time.Second)
	if err != nil {
		t.Fatalf("ListOllamaRunningModelDetails failed: %v", err)
	}
	if len(details) != 2 {
		t.Fatalf("running model details len = %d, want 2", len(details))
	}
	if details[0].Name != "qwen2.5:0.5b-instruct" || details[0].SizeVRAM != 536870912 {
		t.Fatalf("unexpected first detail: %#v", details[0])
	}
	if details[1].Name != "qwen3:4b" || details[1].SizeVRAM != 2147483648 {
		t.Fatalf("unexpected second detail: %#v", details[1])
	}
}
