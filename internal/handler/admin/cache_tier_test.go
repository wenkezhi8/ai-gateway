package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ai-gateway/internal/cache"

	"github.com/gin-gonic/gin"
)

type tierTestHotStore struct {
	exactDocs      map[string]*cache.VectorCacheDocument
	migrationDocs  []*cache.VectorCacheDocument
	memorySequence []float64
	memoryIdx      int
}

func (s *tierTestHotStore) EnsureIndex(_ context.Context) error  { return nil }
func (s *tierTestHotStore) RebuildIndex(_ context.Context) error { return nil }
func (s *tierTestHotStore) GetExact(_ context.Context, cacheKey string) (*cache.VectorCacheDocument, error) {
	if s.exactDocs == nil {
		return nil, nil
	}
	return s.exactDocs[cacheKey], nil
}
func (s *tierTestHotStore) VectorSearch(_ context.Context, _ string, _ []float64, _ int, _ float64) ([]cache.VectorSearchHit, error) {
	return nil, nil
}
func (s *tierTestHotStore) Upsert(_ context.Context, doc *cache.VectorCacheDocument) error {
	if doc != nil {
		if s.exactDocs == nil {
			s.exactDocs = map[string]*cache.VectorCacheDocument{}
		}
		cp := *doc
		s.exactDocs[doc.CacheKey] = &cp
	}
	return nil
}
func (s *tierTestHotStore) Delete(_ context.Context, cacheKey string) error {
	if s.exactDocs != nil {
		delete(s.exactDocs, cacheKey)
	}
	return nil
}
func (s *tierTestHotStore) TouchTTL(_ context.Context, _ string, _ int64) error {
	return nil
}
func (s *tierTestHotStore) Stats(_ context.Context) (cache.VectorStoreStats, error) {
	return cache.VectorStoreStats{Enabled: true, IndexName: "hot-test"}, nil
}
func (s *tierTestHotStore) MemoryUsagePercent(_ context.Context) (float64, error) {
	if len(s.memorySequence) == 0 {
		return 0, nil
	}
	idx := s.memoryIdx
	if idx >= len(s.memorySequence) {
		idx = len(s.memorySequence) - 1
	}
	s.memoryIdx++
	return s.memorySequence[idx], nil
}
func (s *tierTestHotStore) ListMigrationCandidates(_ context.Context, batchSize int) ([]*cache.VectorCacheDocument, error) {
	if len(s.migrationDocs) == 0 {
		return nil, nil
	}
	if batchSize <= 0 || batchSize >= len(s.migrationDocs) {
		return s.migrationDocs, nil
	}
	return s.migrationDocs[:batchSize], nil
}

type tierTestColdStore struct {
	docs map[string]*cache.VectorCacheDocument
}

func (s *tierTestColdStore) EnsureSchema(_ context.Context) error { return nil }
func (s *tierTestColdStore) Upsert(_ context.Context, doc *cache.VectorCacheDocument) error {
	if doc == nil {
		return nil
	}
	if s.docs == nil {
		s.docs = map[string]*cache.VectorCacheDocument{}
	}
	cp := *doc
	s.docs[doc.CacheKey] = &cp
	return nil
}
func (s *tierTestColdStore) VectorSearch(_ context.Context, _ string, _ []float64, _ int, _ float64) ([]cache.VectorSearchHit, error) {
	return nil, nil
}
func (s *tierTestColdStore) GetExact(_ context.Context, cacheKey string) (*cache.VectorCacheDocument, error) {
	if s.docs == nil {
		return nil, nil
	}
	return s.docs[cacheKey], nil
}
func (s *tierTestColdStore) Delete(_ context.Context, cacheKey string) error {
	if s.docs != nil {
		delete(s.docs, cacheKey)
	}
	return nil
}
func (s *tierTestColdStore) Stats(_ context.Context) (cache.ColdVectorStoreStats, error) {
	return cache.ColdVectorStoreStats{
		Backend:   cache.ColdVectorBackendSQLite,
		Available: true,
		Entries:   int64(len(s.docs)),
	}, nil
}

func TestCacheHandler_VectorTierConfigEndpoints_ShouldGetAndUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	handler := NewCacheHandler(manager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	initial := []byte(`{
  "server":{"port":"8566","mode":"debug"},
  "vector_cache":{"enabled":true,"dimension":1024,"query_timeout_ms":1200}
}`)
	if err := os.WriteFile(configPath, initial, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	originPath := os.Getenv("CONFIG_PATH")
	t.Cleanup(func() {
		_ = os.Setenv("CONFIG_PATH", originPath)
	})
	_ = os.Setenv("CONFIG_PATH", configPath)

	router := gin.New()
	router.GET("/api/admin/router/vector/tier/config", handler.GetVectorTierConfig)
	router.PUT("/api/admin/router/vector/tier/config", handler.UpdateVectorTierConfig)

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/router/vector/tier/config", nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected GET 200, got %d body=%s", getW.Code, getW.Body.String())
	}

	body := map[string]any{
		"cold_vector_enabled":               true,
		"cold_vector_query_enabled":         false,
		"cold_vector_backend":               "sqlite",
		"cold_vector_dual_write_enabled":    true,
		"cold_vector_similarity_threshold":  0.91,
		"cold_vector_top_k":                 2,
		"hot_memory_high_watermark_percent": 78,
		"hot_memory_relief_percent":         66,
		"hot_to_cold_batch_size":            320,
		"hot_to_cold_interval_seconds":      45,
		"cold_vector_qdrant_url":            "http://127.0.0.1:6333",
		"cold_vector_qdrant_api_key":        "test-key",
		"cold_vector_qdrant_collection":     "cache_v22",
		"cold_vector_qdrant_timeout_ms":     1900,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/admin/router/vector/tier/config", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	settings := manager.GetSettings()
	if !settings.ColdVectorEnabled {
		t.Fatal("expected cold vector enabled")
	}
	if settings.ColdVectorQueryEnabled {
		t.Fatal("expected cold vector query disabled")
	}
	if settings.ColdVectorBackend != cache.ColdVectorBackendSQLite {
		t.Fatalf("expected sqlite backend, got %s", settings.ColdVectorBackend)
	}
	if !settings.ColdVectorDualWriteEnabled {
		t.Fatal("expected cold dual-write enabled")
	}
	if settings.HotToColdBatchSize != 320 {
		t.Fatalf("expected batch size 320, got %d", settings.HotToColdBatchSize)
	}
}

func TestCacheHandler_VectorTierRouterEndpoints_ShouldWork(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	handler := NewCacheHandler(manager)

	hot := &tierTestHotStore{
		memorySequence: []float64{80, 80, 62},
		migrationDocs: []*cache.VectorCacheDocument{
			{
				CacheKey: "intent:qa:key=1",
				Intent:   "qa",
				Vector:   []float64{0.1, 0.2},
				Response: map[string]any{"answer": "test"},
				TTLSec:   3600,
			},
		},
	}
	cold := &tierTestColdStore{}
	tiered := cache.NewTieredVectorStore(hot, map[string]cache.ColdVectorStore{
		cache.ColdVectorBackendSQLite: cold,
	}, cache.TieredVectorStoreConfig{
		ColdVectorEnabled:             true,
		ColdVectorQueryEnabled:        true,
		ColdVectorBackend:             cache.ColdVectorBackendSQLite,
		HotMemoryHighWatermarkPercent: 75,
		HotMemoryReliefPercent:        65,
		HotToColdBatchSize:            1,
		HotToColdInterval:             30 * time.Second,
	})
	manager.SetTieredVectorStore(tiered)

	router := gin.New()
	router.GET("/api/admin/router/vector/tier/stats", handler.GetVectorTierStats)
	router.POST("/api/admin/router/vector/tier/migrate", handler.TriggerVectorTierMigrate)
	router.POST("/api/admin/router/vector/tier/promote", handler.PromoteVectorTierEntry)

	statsReq := httptest.NewRequest(http.MethodGet, "/api/admin/router/vector/tier/stats", http.NoBody)
	statsW := httptest.NewRecorder()
	router.ServeHTTP(statsW, statsReq)
	if statsW.Code != http.StatusOK {
		t.Fatalf("expected stats 200, got %d body=%s", statsW.Code, statsW.Body.String())
	}

	migrateReq := httptest.NewRequest(http.MethodPost, "/api/admin/router/vector/tier/migrate", http.NoBody)
	migrateW := httptest.NewRecorder()
	router.ServeHTTP(migrateW, migrateReq)
	if migrateW.Code != http.StatusOK {
		t.Fatalf("expected migrate 200, got %d body=%s", migrateW.Code, migrateW.Body.String())
	}

	promoteBody := []byte(`{"cache_key":"intent:qa:key=1"}`)
	promoteReq := httptest.NewRequest(http.MethodPost, "/api/admin/router/vector/tier/promote", bytes.NewReader(promoteBody))
	promoteReq.Header.Set("Content-Type", "application/json")
	promoteW := httptest.NewRecorder()
	router.ServeHTTP(promoteW, promoteReq)
	if promoteW.Code != http.StatusOK {
		t.Fatalf("expected promote 200, got %d body=%s", promoteW.Code, promoteW.Body.String())
	}
}

func TestCacheHandler_VectorTierRoutes_ShouldOnlyExistUnderRouterGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	handler := NewCacheHandler(manager)

	router := gin.New()
	adminGroup := router.Group("/api/admin")
	routerGroup := adminGroup.Group("/router")
	routerGroup.GET("/vector/tier/stats", handler.GetVectorTierStats)

	// 新路径可用（即便 tier store 未初始化，也应由 handler 返回 200 + enabled=false）
	newReq := httptest.NewRequest(http.MethodGet, "/api/admin/router/vector/tier/stats", nil)
	newW := httptest.NewRecorder()
	router.ServeHTTP(newW, newReq)
	if newW.Code != http.StatusOK {
		t.Fatalf("expected new router path status 200, got %d body=%s", newW.Code, newW.Body.String())
	}

	// 旧路径未注册，应为 404
	oldReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/vector/tier/stats", nil)
	oldW := httptest.NewRecorder()
	router.ServeHTTP(oldW, oldReq)
	if oldW.Code != http.StatusNotFound {
		t.Fatalf("expected old cache path status 404, got %d body=%s", oldW.Code, oldW.Body.String())
	}
}

func TestCacheHandler_VectorTierConfig_ShouldPersistToConfigFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	handler := NewCacheHandler(manager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	initial := []byte(`{
  "server":{"port":"8566","mode":"debug"},
  "vector_cache":{
    "enabled": true,
    "index_name":"idx_ai_cache_v2",
    "key_prefix":"ai:v2:cache:",
    "dimension":1024,
    "query_timeout_ms":1200
  }
}`)
	if err := os.WriteFile(configPath, initial, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	originPath := os.Getenv("CONFIG_PATH")
	t.Cleanup(func() {
		_ = os.Setenv("CONFIG_PATH", originPath)
	})
	_ = os.Setenv("CONFIG_PATH", configPath)

	router := gin.New()
	router.PUT("/api/admin/router/vector/tier/config", handler.UpdateVectorTierConfig)

	body := map[string]any{
		"vector_pipeline_enabled":            true,
		"vector_standard_key_version":        "v2",
		"vector_embedding_provider":          "ollama",
		"vector_ollama_base_url":             "http://127.0.0.1:11434",
		"vector_ollama_embedding_model":      "nomic-embed-text",
		"vector_ollama_embedding_dimension":  1024,
		"vector_ollama_embedding_timeout_ms": 1600,
		"vector_ollama_endpoint_mode":        "auto",
		"vector_writeback_enabled":           true,
		"cold_vector_enabled":                true,
		"cold_vector_query_enabled":          false,
		"cold_vector_backend":                "qdrant",
		"cold_vector_similarity_threshold":   0.9,
		"cold_vector_top_k":                  3,
		"hot_memory_high_watermark_percent":  79,
		"hot_memory_relief_percent":          68,
		"hot_to_cold_batch_size":             256,
		"hot_to_cold_interval_seconds":       22,
		"cold_vector_qdrant_url":             "http://127.0.0.1:6333",
		"cold_vector_qdrant_collection":      "tier_v22",
		"cold_vector_qdrant_timeout_ms":      1800,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/admin/router/vector/tier/config", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	savedRaw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	var saved map[string]any
	if err := json.Unmarshal(savedRaw, &saved); err != nil {
		t.Fatalf("decode saved config: %v", err)
	}
	vectorCache, ok := saved["vector_cache"].(map[string]any)
	if !ok {
		t.Fatalf("expected vector_cache object in saved config, got %#v", saved["vector_cache"])
	}

	enabled, ok := vectorCache["cold_vector_enabled"].(bool)
	if !ok || !enabled {
		t.Fatalf("expected cold_vector_enabled=true in persisted config, got %#v", vectorCache["cold_vector_enabled"])
	}
	backend, ok := vectorCache["cold_vector_backend"].(string)
	if !ok || backend != "qdrant" {
		t.Fatalf("expected backend=qdrant in persisted config, got %#v", vectorCache["cold_vector_backend"])
	}
	baseURL, ok := vectorCache["ollama_base_url"].(string)
	if !ok || baseURL != "http://127.0.0.1:11434" {
		t.Fatalf("expected ollama_base_url persisted, got %#v", vectorCache["ollama_base_url"])
	}
	model, ok := vectorCache["ollama_embedding_model"].(string)
	if !ok || model != "nomic-embed-text" {
		t.Fatalf("expected ollama_embedding_model persisted, got %#v", vectorCache["ollama_embedding_model"])
	}
	endpointMode, ok := vectorCache["ollama_endpoint_mode"].(string)
	if !ok || endpointMode != "auto" {
		t.Fatalf("expected ollama_endpoint_mode persisted, got %#v", vectorCache["ollama_endpoint_mode"])
	}
	writebackEnabled, ok := vectorCache["writeback_enabled"].(bool)
	if !ok || !writebackEnabled {
		t.Fatalf("expected writeback_enabled=true in persisted config, got %#v", vectorCache["writeback_enabled"])
	}
	if topK, ok := vectorCache["cold_vector_top_k"].(float64); !ok || int(topK) != 3 {
		t.Fatalf("expected cold_vector_top_k=3 in persisted config, got %#v", vectorCache["cold_vector_top_k"])
	}
}

type vectorPipelineTestStore struct {
	hits []cache.VectorSearchHit
}

func (s *vectorPipelineTestStore) EnsureIndex(_ context.Context) error  { return nil }
func (s *vectorPipelineTestStore) RebuildIndex(_ context.Context) error { return nil }
func (s *vectorPipelineTestStore) GetExact(_ context.Context, _ string) (*cache.VectorCacheDocument, error) {
	return nil, nil
}
func (s *vectorPipelineTestStore) VectorSearch(_ context.Context, _ string, _ []float64, _ int, _ float64) ([]cache.VectorSearchHit, error) {
	return s.hits, nil
}
func (s *vectorPipelineTestStore) Upsert(_ context.Context, _ *cache.VectorCacheDocument) error {
	return nil
}
func (s *vectorPipelineTestStore) Delete(_ context.Context, _ string) error { return nil }
func (s *vectorPipelineTestStore) TouchTTL(_ context.Context, _ string, _ int64) error {
	return nil
}
func (s *vectorPipelineTestStore) Stats(_ context.Context) (cache.VectorStoreStats, error) {
	return cache.VectorStoreStats{Enabled: true, Dimension: 3, IndexName: "idx_ai_cache_v2"}, nil
}

func TestCacheHandler_VectorPipelineEndpoints_ShouldWork(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/embed":
			if err := json.NewEncoder(w).Encode(map[string]any{
				"embeddings": [][]float64{{0.1, 0.2, 0.3}},
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer ollamaServer.Close()

	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	settings := manager.GetSettings()
	settings.VectorEnabled = true
	settings.VectorPipelineEnabled = true
	settings.VectorEmbeddingProvider = "ollama"
	settings.VectorOllamaBaseURL = ollamaServer.URL
	settings.VectorOllamaEmbeddingModel = "nomic-embed-text"
	settings.VectorOllamaEmbeddingDimension = 3
	settings.VectorOllamaEmbeddingTimeoutMs = 500
	settings.VectorOllamaEndpointMode = cache.OllamaEndpointModeAuto
	settings.VectorWritebackEnabled = true
	manager.UpdateSettings(settings)

	store := &vectorPipelineTestStore{
		hits: []cache.VectorSearchHit{
			{CacheKey: "intent:qa:query_hash=abc", Intent: "qa", Similarity: 0.96, Score: 0.04},
		},
	}
	manager.SetVectorStore(store)
	handler := NewCacheHandler(manager)

	router := gin.New()
	router.GET("/api/admin/cache/vector/pipeline/health", handler.GetVectorPipelineHealth)
	router.POST("/api/admin/cache/vector/pipeline/test", handler.TestVectorPipeline)

	healthReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/vector/pipeline/health", http.NoBody)
	healthW := httptest.NewRecorder()
	router.ServeHTTP(healthW, healthReq)
	if healthW.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d body=%s", healthW.Code, healthW.Body.String())
	}

	testBody := []byte(`{"query":"向量检索测试","task_type":"qa","top_k":3,"min_similarity":0.9}`)
	testReq := httptest.NewRequest(http.MethodPost, "/api/admin/cache/vector/pipeline/test", bytes.NewReader(testBody))
	testReq.Header.Set("Content-Type", "application/json")
	testW := httptest.NewRecorder()
	router.ServeHTTP(testW, testReq)
	if testW.Code != http.StatusOK {
		t.Fatalf("expected test 200, got %d body=%s", testW.Code, testW.Body.String())
	}
}
