package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-gateway/internal/routing"
)

func TestOllamaService_ResolveStartCommand_AutoFallbackToCLI(t *testing.T) {
	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	svc.goos = goosDarwin
	svc.config.StartupMode = StartupModeAuto
	svc.appInstalledFn = func() bool { return false }
	svc.commandExistsFn = func(name string) bool { return name == "ollama" }

	mode, command, err := svc.resolveStartCommand()
	if err != nil {
		t.Fatalf("resolveStartCommand returned error: %v", err)
	}
	if mode != StartupModeCLI {
		t.Fatalf("expected mode %q, got %q", StartupModeCLI, mode)
	}
	if !strings.Contains(command, "ollama serve") {
		t.Fatalf("expected CLI command to contain ollama serve, got %q", command)
	}
}

func TestOllamaService_Start_ManualModeReturnsError(t *testing.T) {
	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	svc.config.StartupMode = StartupModeManual
	svc.checkRunningFn = func(context.Context, *routing.ClassifierConfig) (bool, []string, string) {
		return false, nil, "not running"
	}

	_, err := svc.Start(context.Background(), &routing.ClassifierConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var startErr *StartError
	if !errors.As(err, &startErr) {
		t.Fatalf("expected StartError, got %T", err)
	}
	if startErr.Code != "manual_mode" {
		t.Fatalf("expected error code manual_mode, got %q", startErr.Code)
	}
}

func TestOllamaService_CheckAndAutoRestart_WhenUnhealthy(t *testing.T) {
	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	svc.config.StartupMode = StartupModeCLI
	svc.config.Monitoring.Enabled = true
	svc.config.Monitoring.AutoRestart = true
	svc.config.Monitoring.MaxRestartAttempts = 3
	svc.config.Monitoring.RestartCooldownSeconds = 0
	svc.config.StartupTimeoutSeconds = 1

	running := false
	svc.checkRunningFn = func(context.Context, *routing.ClassifierConfig) (bool, []string, string) {
		if running {
			return true, nil, "ok"
		}
		return false, nil, "not running"
	}
	svc.commandExistsFn = func(name string) bool { return name == "ollama" }
	svc.runShellFn = func(time.Duration, string) (string, error) {
		running = true
		return "started", nil
	}
	svc.sleepFn = func(time.Duration) {}

	err := svc.CheckAndAutoRestart(context.Background(), &routing.ClassifierConfig{})
	if err != nil {
		t.Fatalf("CheckAndAutoRestart returned error: %v", err)
	}
	if !running {
		t.Fatal("expected service to become running after auto restart")
	}
	status := svc.GetMonitorStatus()
	if status.RestartAttempts != 1 {
		t.Fatalf("expected restart attempts to be 1, got %d", status.RestartAttempts)
	}
}

func TestOllamaService_PreloadModels_ShouldKeepAliveForIntentAndEmbedding(t *testing.T) {
	t.Parallel()

	chatKeepAlive := ""
	embedKeepAlive := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/chat":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if v, ok := body["keep_alive"].(string); ok {
				chatKeepAlive = v
			}
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"message":{"content":"ok"}}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		case "/api/embed":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if v, ok := body["keep_alive"].(string); ok {
				embedKeepAlive = v
			}
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	results := svc.PreloadModels(context.Background(), []PreloadTarget{
		{Model: "qwen2.5:0.5b-instruct", Kind: PreloadTargetIntent, BaseURL: server.URL},
		{Model: "nomic-embed-text", Kind: PreloadTargetEmbedding, BaseURL: server.URL, EmbeddingEndpointMode: "embed"},
	}, 180)

	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	if results[0].Status != "success" {
		t.Fatalf("intent preload status = %q, want success", results[0].Status)
	}
	if results[1].Status != "success" {
		t.Fatalf("embedding preload status = %q, want success", results[1].Status)
	}
	if chatKeepAlive != "-1m" {
		t.Fatalf("chat keep_alive = %q, want -1m", chatKeepAlive)
	}
	if embedKeepAlive != "-1m" {
		t.Fatalf("embed keep_alive = %q, want -1m", embedKeepAlive)
	}
}

func TestOllamaService_PreloadModels_ShouldSupportEmbeddingsEndpoint(t *testing.T) {
	t.Parallel()

	calledEmbeddings := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/embeddings":
			calledEmbeddings = true
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"embedding":[0.1,0.2]}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	results := svc.PreloadModels(context.Background(), []PreloadTarget{
		{Model: "nomic-embed-text", Kind: PreloadTargetEmbedding, BaseURL: server.URL, EmbeddingEndpointMode: "embeddings"},
	}, 180)

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].Status != "success" {
		t.Fatalf("embedding preload status = %q, want success", results[0].Status)
	}
	if !calledEmbeddings {
		t.Fatal("expected /api/embeddings endpoint to be called")
	}
}

func TestOllamaService_PreloadModels_ShouldTimeoutPerModel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"message":{"content":"ok"}}`)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	results := svc.PreloadModels(context.Background(), []PreloadTarget{
		{Model: "qwen2.5:0.5b-instruct", Kind: PreloadTargetIntent, BaseURL: server.URL},
	}, 1)

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].Status != "failed" {
		t.Fatalf("status = %q, want failed", results[0].Status)
	}
	if !strings.Contains(results[0].Error, "deadline") && !strings.Contains(results[0].Error, "timeout") {
		t.Fatalf("error = %q, want timeout related", results[0].Error)
	}
}
