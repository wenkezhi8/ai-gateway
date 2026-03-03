package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

type providerDeleteTestProvider struct {
	name    string
	models  []string
	enabled bool
}

func (p *providerDeleteTestProvider) Name() string { return p.name }

func (p *providerDeleteTestProvider) Chat(_ context.Context, _ *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{}, nil
}

func (p *providerDeleteTestProvider) StreamChat(_ context.Context, _ *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	ch := make(chan *provider.StreamChunk)
	close(ch)
	return ch, nil
}

func (p *providerDeleteTestProvider) Models() []string { return p.models }

func (p *providerDeleteTestProvider) ValidateKey(_ context.Context) bool { return true }

func (p *providerDeleteTestProvider) IsEnabled() bool { return p.enabled }

func (p *providerDeleteTestProvider) SetEnabled(enabled bool) { p.enabled = enabled }

func TestProviderHandler_DeleteProvider_ShouldCascadeAndPersist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := os.MkdirAll("configs", 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile("configs/config.json", []byte(`{"providers":[{"name":"openai","api_key":"sk-openai","base_url":"https://api.openai.com/v1","enabled":true,"models":["gpt-4o"]},{"name":"qwen","api_key":"sk-qwen","base_url":"https://dashscope.aliyuncs.com/compatible-mode/v1","enabled":true,"models":["qwen-max"]}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	registry := provider.NewRegistry()
	registry.Register("openai", &providerDeleteTestProvider{name: "openai", models: []string{"gpt-4o"}, enabled: true})
	registry.Register("qwen", &providerDeleteTestProvider{name: "qwen", models: []string{"qwen-max"}, enabled: true})

	manager := limiter.NewAccountManager(nil, nil)
	if addErr := manager.AddAccount(&limiter.AccountConfig{ID: "acc-openai", Name: "openai-account", Provider: "openai", ProviderType: "openai", Enabled: true}); addErr != nil {
		t.Fatalf("add openai account: %v", addErr)
	}
	if addErr := manager.AddAccount(&limiter.AccountConfig{ID: "acc-qwen", Name: "qwen-account", Provider: "qwen", ProviderType: "qwen", Enabled: true}); addErr != nil {
		t.Fatalf("add qwen account: %v", addErr)
	}

	smartRouter := routing.NewSmartRouter()
	smartRouter.UpdateModelScore("gpt-4o", &routing.ModelScore{Model: "gpt-4o", Provider: "openai", QualityScore: 90, SpeedScore: 80, CostScore: 70, Enabled: true})
	smartRouter.UpdateModelScore("qwen-max", &routing.ModelScore{Model: "qwen-max", Provider: "qwen", QualityScore: 80, SpeedScore: 85, CostScore: 88, Enabled: true})
	smartRouter.SetProviderDefault("openai", "gpt-4o")
	smartRouter.SetProviderDefault("qwen", "qwen-max")

	handler := NewProviderHandler(registry, manager, smartRouter, filepath.Join(tmpDir, "configs", "config.json"))

	engine := gin.New()
	engine.DELETE("/api/admin/providers/:id", handler.DeleteProvider)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/providers/openai", http.NoBody)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	if _, ok := registry.Get("openai"); ok {
		t.Fatalf("expected openai removed from registry")
	}
	if _, ok := registry.Get("qwen"); !ok {
		t.Fatalf("expected qwen to remain in registry")
	}

	accounts := manager.GetAllAccounts()
	for _, acc := range accounts {
		if acc.Provider == "openai" || acc.ProviderType == "openai" {
			t.Fatalf("expected openai accounts removed, found account=%s", acc.ID)
		}
	}

	defaults := smartRouter.GetProviderDefaults()
	if _, ok := defaults["openai"]; ok {
		t.Fatalf("expected openai default model removed")
	}

	for model, score := range smartRouter.GetAllModelScores() {
		if score != nil && score.Provider == "openai" {
			t.Fatalf("expected openai model scores removed, found model=%s", model)
		}
	}

	configRaw, err := os.ReadFile(filepath.Join(tmpDir, "configs", "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var configDoc map[string]any
	if err := json.Unmarshal(configRaw, &configDoc); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	providersAny, ok := configDoc["providers"].([]any)
	if !ok {
		t.Fatalf("expected providers array in config")
	}
	for _, item := range providersAny {
		providerObj, _ := item.(map[string]any)
		if providerObj["name"] == "openai" {
			t.Fatalf("expected openai removed from configs/config.json")
		}
	}

	accountsRaw, err := os.ReadFile(filepath.Join(tmpDir, "data", "accounts.json"))
	if err != nil {
		t.Fatalf("read persisted accounts: %v", err)
	}
	var persistedAccounts []map[string]any
	if err := json.Unmarshal(accountsRaw, &persistedAccounts); err != nil {
		t.Fatalf("unmarshal persisted accounts: %v", err)
	}
	for _, acc := range persistedAccounts {
		if acc["provider"] == "openai" || acc["provider_type"] == "openai" {
			t.Fatalf("expected no openai account in persisted accounts")
		}
	}
}
