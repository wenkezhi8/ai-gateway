package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/intent"
	"ai-gateway/internal/routing"
)

func TestApplyControlTTLBand(t *testing.T) {
	base := 30 * time.Minute
	signals := &routing.ControlSignals{TTLBand: "long"}

	if got := applyControlTTLBand(base, routing.ControlConfig{}, signals); got != base {
		t.Fatalf("expected base ttl when control disabled, got %v", got)
	}

	cfg := routing.ControlConfig{Enable: true, CacheWriteGateEnable: true}
	if got := applyControlTTLBand(base, cfg, signals); got != 7*24*time.Hour {
		t.Fatalf("expected long ttl mapping, got %v", got)
	}

	signals.TTLBand = "medium"
	if got := applyControlTTLBand(base, cfg, signals); got != 24*time.Hour {
		t.Fatalf("expected medium ttl mapping, got %v", got)
	}

	signals.TTLBand = "short"
	if got := applyControlTTLBand(base, cfg, signals); got != time.Hour {
		t.Fatalf("expected short ttl mapping, got %v", got)
	}

	ruleMatchedTTL := 2 * time.Hour
	signals.TTLBand = "long"
	if got := applyControlTTLBand(ruleMatchedTTL, cfg, signals); got != 7*24*time.Hour {
		t.Fatalf("expected control ttl to override matched rule ttl, got %v", got)
	}
}

func TestShouldAllowCacheWrite(t *testing.T) {
	allow := true
	deny := false

	cfg := routing.ControlConfig{Enable: true, CacheWriteGateEnable: true}
	if !shouldAllowCacheWrite(cfg, &routing.ControlSignals{Cacheable: &allow}) {
		t.Fatal("expected write allowed")
	}
	if shouldAllowCacheWrite(cfg, &routing.ControlSignals{Cacheable: &deny}) {
		t.Fatal("expected write denied")
	}

	if !shouldAllowCacheWrite(routing.ControlConfig{}, &routing.ControlSignals{Cacheable: &deny}) {
		t.Fatal("expected write allowed when control disabled")
	}
}

func TestApplyControlToolGate(t *testing.T) {
	req := &ChatCompletionRequest{
		Tools: []Tool{{Type: "function", Function: Function{Name: "lookup"}}},
		ToolChoice: map[string]interface{}{
			"type": "function",
		},
	}

	cfg := routing.ControlConfig{Enable: true, ToolGateEnable: true}
	assessment := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{ToolNeeded: boolPtr(false)}}

	applyControlToolGate(req, cfg, assessment)

	if len(req.Tools) != 0 {
		t.Fatalf("expected tools cleared, got %d", len(req.Tools))
	}
	if req.ToolChoice != nil {
		t.Fatal("expected tool choice cleared")
	}

	req2 := &ChatCompletionRequest{Tools: []Tool{{Type: "function", Function: Function{Name: "lookup"}}}}
	assessment2 := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{ToolNeeded: boolPtr(true)}}
	applyControlToolGate(req2, cfg, assessment2)
	if len(req2.Tools) == 0 {
		t.Fatal("expected tools preserved when tool_needed=true")
	}

	req3 := &ChatCompletionRequest{Tools: []Tool{{Type: "function", Function: Function{Name: "lookup"}}}}
	shadowCfg := routing.ControlConfig{Enable: true, ToolGateEnable: true, ShadowOnly: true}
	assessment3 := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{ToolNeeded: boolPtr(false)}}
	applyControlToolGate(req3, shadowCfg, assessment3)
	if len(req3.Tools) == 0 {
		t.Fatal("expected tools preserved in shadow mode")
	}

	req4 := &ChatCompletionRequest{DeepThink: true}
	assessment4 := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{RAGNeeded: boolPtr(false)}}
	applyControlToolGate(req4, cfg, assessment4)
	if req4.DeepThink {
		t.Fatal("expected deepThink disabled when rag_needed=false")
	}

	req5 := &ChatCompletionRequest{DeepThink: true}
	applyControlToolGate(req5, shadowCfg, assessment4)
	if !req5.DeepThink {
		t.Fatal("expected deepThink preserved in shadow mode")
	}
}

func TestBuildSemanticQueryCandidates(t *testing.T) {
	candidates := buildSemanticQueryCandidates(true, "norm", "sig", "prompt")
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0] != "prompt" || candidates[1] != "norm" {
		t.Fatalf("unexpected candidate order: %#v", candidates)
	}

	candidates = buildSemanticQueryCandidates(false, "norm", "sig", "sig")
	if len(candidates) != 1 || candidates[0] != "sig" {
		t.Fatalf("expected prompt fallback when prompt missing, got %#v", candidates)
	}

	candidates = buildSemanticQueryCandidates(false, "norm", "sig", "prompt")
	if len(candidates) != 1 || candidates[0] != "prompt" {
		t.Fatalf("expected prompt-only candidates, got %#v", candidates)
	}

	candidates = buildSemanticQueryCandidates(true, "", "", "")
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates for empty input, got %#v", candidates)
	}
}

func TestBuildSemanticCacheWriteQuery(t *testing.T) {
	if got := buildSemanticCacheWriteQuery(" prompt ", "norm", "sig", true); got != "prompt" {
		t.Fatalf("expected prompt-first write query, got %q", got)
	}

	if got := buildSemanticCacheWriteQuery("", " norm ", "sig", true); got != "norm" {
		t.Fatalf("expected normalized query fallback, got %q", got)
	}

	if got := buildSemanticCacheWriteQuery("", " norm ", "sig", false); got != "sig" {
		t.Fatalf("expected semantic signature fallback, got %q", got)
	}
}

func TestShouldAllowSemanticCache(t *testing.T) {
	if shouldAllowSemanticCache(routing.TaskTypeChat) {
		t.Fatal("expected chat task to skip semantic cache")
	}
	if shouldAllowSemanticCache(routing.TaskTypeCreative) {
		t.Fatal("expected creative task to skip semantic cache")
	}
	if shouldAllowSemanticCache(routing.TaskTypeUnknown) {
		t.Fatal("expected unknown task to skip semantic cache")
	}
	if !shouldAllowSemanticCache(routing.TaskTypeFact) {
		t.Fatal("expected fact task to allow semantic cache")
	}
}

func TestShouldBlockByRisk(t *testing.T) {
	cfg := routing.ControlConfig{Enable: true, RiskTagEnable: true, RiskBlockEnable: true}
	assessment := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{RiskLevel: "high"}}

	if !shouldBlockByRisk(cfg, assessment) {
		t.Fatal("expected high risk to be blocked")
	}

	cfg.ShadowOnly = true
	if shouldBlockByRisk(cfg, assessment) {
		t.Fatal("expected shadow mode not to block")
	}

	cfg.ShadowOnly = false
	assessment.ControlSignals.RiskLevel = "low"
	if shouldBlockByRisk(cfg, assessment) {
		t.Fatal("expected low risk not to be blocked")
	}
}

func TestApplyControlGenerationHints(t *testing.T) {
	temp := 0.3
	topP := 0.85
	maxTokens := 768

	req := &ChatCompletionRequest{}
	cfg := routing.ControlConfig{Enable: true, ParameterHintEnable: true}
	assessment := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{
		RecommendedTemperature: &temp,
		RecommendedTopP:        &topP,
		RecommendedMaxTokens:   &maxTokens,
	}}

	applyControlGenerationHints(req, cfg, assessment)
	if req.Temperature == nil || *req.Temperature != temp {
		t.Fatal("expected temperature hint applied")
	}
	if req.TopP == nil || *req.TopP != topP {
		t.Fatal("expected top_p hint applied")
	}
	if req.MaxTokens == nil || *req.MaxTokens != maxTokens {
		t.Fatal("expected max_tokens hint applied")
	}

	shadowReq := &ChatCompletionRequest{}
	shadowCfg := routing.ControlConfig{Enable: true, ParameterHintEnable: true, ShadowOnly: true}
	applyControlGenerationHints(shadowReq, shadowCfg, assessment)
	if shadowReq.Temperature != nil || shadowReq.TopP != nil || shadowReq.MaxTokens != nil {
		t.Fatal("expected no mutation in shadow mode")
	}
}

func TestBuildControlHeaders(t *testing.T) {
	cfg := routing.ControlConfig{Enable: true}
	assessment := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{
		ExperimentTag: "exp-a",
		DomainTag:     "coding",
	}}

	headers := buildControlHeaders(cfg, assessment)
	if headers["X-Control-Experiment"] != "exp-a" {
		t.Fatalf("unexpected experiment header: %#v", headers)
	}
	if headers["X-Control-Domain"] != "coding" {
		t.Fatalf("unexpected domain header: %#v", headers)
	}

	headers = buildControlHeaders(routing.ControlConfig{Enable: false}, assessment)
	if len(headers) != 0 {
		t.Fatalf("expected no headers when control disabled, got %#v", headers)
	}
}

func TestIntentThreshold(t *testing.T) {
	h := &ProxyHandler{}
	settings := cache.CacheSettings{
		SimilarityThreshold: 0.92,
		VectorThresholds: map[string]float64{
			"calc": 0.97,
		},
	}
	if got := h.intentThreshold("calc", settings); got != 0.97 {
		t.Fatalf("expected intent-specific threshold, got %v", got)
	}
	if got := h.intentThreshold("qa", settings); got != 0.92 {
		t.Fatalf("expected fallback threshold, got %v", got)
	}
}

func TestIntentTTLSeconds(t *testing.T) {
	h := &ProxyHandler{
		config: &config.Config{
			VectorCache: config.VectorCacheConfig{
				TTLSeconds: map[string]int64{
					"calc": 2592000,
				},
			},
		},
	}
	result := &intent.EmbeddingResult{Intent: "calc"}
	if got := h.intentTTLSeconds(result); got != 2592000 {
		t.Fatalf("expected configured ttl, got %d", got)
	}
}

func TestProcessCacheV2Write_SkipUnknownIntent(t *testing.T) {
	store := &mockVectorStoreForProxy{}
	h := &ProxyHandler{
		config:      &config.Config{},
		vectorStore: store,
	}
	resp := ChatCompletionResponse{
		Choices: []Choice{
			{
				Message: &ChatMessage{Role: "assistant", Content: "ok"},
			},
		},
	}
	h.processCacheV2Write(context.Background(), &intent.EmbeddingResult{
		Intent:        "unknown",
		StandardKey:   "intent:unknown",
		Embedding:     []float64{0.1},
		EngineVersion: "v1",
	}, "openai", "gpt-4o-mini", routing.TaskTypeUnknown, resp)

	if store.upsertCalled {
		t.Fatal("expected unknown intent to skip vector cache write")
	}
}

type proxyTierHotStore struct {
	mu         sync.RWMutex
	upsertDocs []*cache.VectorCacheDocument
}

func (s *proxyTierHotStore) EnsureIndex(_ context.Context) error  { return nil }
func (s *proxyTierHotStore) RebuildIndex(_ context.Context) error { return nil }
func (s *proxyTierHotStore) GetExact(_ context.Context, _ string) (*cache.VectorCacheDocument, error) {
	return nil, nil
}
func (s *proxyTierHotStore) VectorSearch(_ context.Context, _ string, _ []float64, _ int, _ float64) ([]cache.VectorSearchHit, error) {
	return nil, nil
}
func (s *proxyTierHotStore) Upsert(_ context.Context, doc *cache.VectorCacheDocument) error {
	if doc != nil {
		cp := *doc
		s.mu.Lock()
		s.upsertDocs = append(s.upsertDocs, &cp)
		s.mu.Unlock()
	}
	return nil
}
func (s *proxyTierHotStore) Delete(_ context.Context, _ string) error { return nil }
func (s *proxyTierHotStore) TouchTTL(_ context.Context, _ string, _ int64) error {
	return nil
}
func (s *proxyTierHotStore) Stats(_ context.Context) (cache.VectorStoreStats, error) {
	return cache.VectorStoreStats{Enabled: true}, nil
}
func (s *proxyTierHotStore) MemoryUsagePercent(_ context.Context) (float64, error) {
	return 0, nil
}
func (s *proxyTierHotStore) ListMigrationCandidates(_ context.Context, _ int) ([]*cache.VectorCacheDocument, error) {
	return nil, nil
}

func (s *proxyTierHotStore) UpsertCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.upsertDocs)
}

type proxyTierColdStore struct {
	doc  *cache.VectorCacheDocument
	hits []cache.VectorSearchHit
}

func (s *proxyTierColdStore) EnsureSchema(_ context.Context) error { return nil }
func (s *proxyTierColdStore) Upsert(_ context.Context, _ *cache.VectorCacheDocument) error {
	return nil
}
func (s *proxyTierColdStore) VectorSearch(_ context.Context, _ string, _ []float64, _ int, _ float64) ([]cache.VectorSearchHit, error) {
	return s.hits, nil
}
func (s *proxyTierColdStore) GetExact(_ context.Context, cacheKey string) (*cache.VectorCacheDocument, error) {
	if s.doc != nil && s.doc.CacheKey == cacheKey {
		return s.doc, nil
	}
	return nil, nil
}
func (s *proxyTierColdStore) Delete(_ context.Context, _ string) error { return nil }
func (s *proxyTierColdStore) Stats(_ context.Context) (cache.ColdVectorStoreStats, error) {
	return cache.ColdVectorStoreStats{Backend: cache.ColdVectorBackendSQLite, Available: true}, nil
}

func TestProcessCacheV2Read_HotMissColdHit_ShouldPromoteHot(t *testing.T) {
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"embeddings": [][]float64{
				{0.1, 0.2},
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer ollamaServer.Close()

	hot := &proxyTierHotStore{}
	coldDoc := &cache.VectorCacheDocument{
		CacheKey: "intent:qa:key=cache-cold-hit",
		Intent:   "qa",
		Vector:   []float64{0.1, 0.2},
		Response: map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "这是冷层命中的缓存回答",
					},
				},
			},
		},
		TTLSec: 3600,
	}
	responseRaw, err := json.Marshal(coldDoc.Response)
	if err != nil {
		t.Fatalf("marshal cold response: %v", err)
	}
	cold := &proxyTierColdStore{
		doc: coldDoc,
		hits: []cache.VectorSearchHit{
			{
				CacheKey:   coldDoc.CacheKey,
				Intent:     "qa",
				Similarity: 0.96,
				Response:   responseRaw,
			},
		},
	}
	tiered := cache.NewTieredVectorStore(hot, map[string]cache.ColdVectorStore{
		cache.ColdVectorBackendSQLite: cold,
	}, cache.TieredVectorStoreConfig{
		ColdVectorEnabled:             true,
		ColdVectorQueryEnabled:        true,
		ColdVectorBackend:             cache.ColdVectorBackendSQLite,
		ColdVectorSimilarityThreshold: 0.95,
	})

	h := &ProxyHandler{
		vectorStore:    tiered,
		textNormalizer: cache.NewTextNormalizer(),
	}

	settings := cache.DefaultCacheSettings()
	settings.VectorEnabled = true
	settings.VectorPipelineEnabled = true
	settings.VectorEmbeddingProvider = "ollama"
	settings.VectorOllamaBaseURL = ollamaServer.URL
	settings.VectorOllamaEmbeddingModel = "nomic-embed-text"
	settings.VectorOllamaEmbeddingDimension = 2
	settings.VectorOllamaEndpointMode = cache.OllamaEndpointModeAuto

	intentResult, payload, hit, layer, key := h.processCacheV2Read(context.Background(), "什么是缓存", "什么是缓存", "qa", settings)
	if intentResult == nil {
		t.Fatal("expected intent result")
	}
	if !hit {
		t.Fatal("expected cold tier hit")
	}
	if layer != "vector-semantic" {
		t.Fatalf("expected vector-semantic layer, got %s", layer)
	}
	if key != coldDoc.CacheKey {
		t.Fatalf("expected hit key %s, got %s", coldDoc.CacheKey, key)
	}
	if len(payload) == 0 {
		t.Fatal("expected cached payload from cold hit")
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if hot.UpsertCount() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if hot.UpsertCount() == 0 {
		t.Fatal("expected cold hit to promote document back to hot tier")
	}
}

type mockVectorStoreForProxy struct {
	upsertCalled bool
}

func (m *mockVectorStoreForProxy) EnsureIndex(_ context.Context) error  { return nil }
func (m *mockVectorStoreForProxy) RebuildIndex(_ context.Context) error { return nil }
func (m *mockVectorStoreForProxy) GetExact(_ context.Context, _ string) (*cache.VectorCacheDocument, error) {
	return nil, nil
}
func (m *mockVectorStoreForProxy) VectorSearch(_ context.Context, _ string, _ []float64, _ int, _ float64) ([]cache.VectorSearchHit, error) {
	return nil, nil
}
func (m *mockVectorStoreForProxy) Upsert(_ context.Context, _ *cache.VectorCacheDocument) error {
	m.upsertCalled = true
	return nil
}
func (m *mockVectorStoreForProxy) Delete(_ context.Context, _ string) error { return nil }
func (m *mockVectorStoreForProxy) TouchTTL(_ context.Context, _ string, _ int64) error {
	return nil
}
func (m *mockVectorStoreForProxy) Stats(_ context.Context) (cache.VectorStoreStats, error) {
	return cache.VectorStoreStats{}, nil
}

func boolPtr(v bool) *bool {
	return &v
}
