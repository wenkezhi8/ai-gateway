package admin

import (
	"ai-gateway/internal/cache"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type tierTestHotStore struct {
	exactDocs      map[string]*cache.VectorCacheDocument
	migrationDocs  []*cache.VectorCacheDocument
	memorySequence []float64
	memoryIdx      int
}

func (s *tierTestHotStore) EnsureIndex(ctx context.Context) error  { return nil }
func (s *tierTestHotStore) RebuildIndex(ctx context.Context) error { return nil }
func (s *tierTestHotStore) GetExact(ctx context.Context, cacheKey string) (*cache.VectorCacheDocument, error) {
	if s.exactDocs == nil {
		return nil, nil
	}
	return s.exactDocs[cacheKey], nil
}
func (s *tierTestHotStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]cache.VectorSearchHit, error) {
	return nil, nil
}
func (s *tierTestHotStore) Upsert(ctx context.Context, doc *cache.VectorCacheDocument) error {
	if doc != nil {
		if s.exactDocs == nil {
			s.exactDocs = map[string]*cache.VectorCacheDocument{}
		}
		cp := *doc
		s.exactDocs[doc.CacheKey] = &cp
	}
	return nil
}
func (s *tierTestHotStore) Delete(ctx context.Context, cacheKey string) error {
	if s.exactDocs != nil {
		delete(s.exactDocs, cacheKey)
	}
	return nil
}
func (s *tierTestHotStore) TouchTTL(ctx context.Context, cacheKey string, ttlSec int64) error {
	return nil
}
func (s *tierTestHotStore) Stats(ctx context.Context) (cache.VectorStoreStats, error) {
	return cache.VectorStoreStats{Enabled: true, IndexName: "hot-test"}, nil
}
func (s *tierTestHotStore) MemoryUsagePercent(ctx context.Context) (float64, error) {
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
func (s *tierTestHotStore) ListMigrationCandidates(ctx context.Context, batchSize int) ([]*cache.VectorCacheDocument, error) {
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

func (s *tierTestColdStore) EnsureSchema(ctx context.Context) error { return nil }
func (s *tierTestColdStore) Upsert(ctx context.Context, doc *cache.VectorCacheDocument) error {
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
func (s *tierTestColdStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]cache.VectorSearchHit, error) {
	return nil, nil
}
func (s *tierTestColdStore) GetExact(ctx context.Context, cacheKey string) (*cache.VectorCacheDocument, error) {
	if s.docs == nil {
		return nil, nil
	}
	return s.docs[cacheKey], nil
}
func (s *tierTestColdStore) Delete(ctx context.Context, cacheKey string) error {
	if s.docs != nil {
		delete(s.docs, cacheKey)
	}
	return nil
}
func (s *tierTestColdStore) Stats(ctx context.Context) (cache.ColdVectorStoreStats, error) {
	return cache.ColdVectorStoreStats{
		Backend:   cache.ColdVectorBackendSQLite,
		Available: true,
		Entries:   int64(len(s.docs)),
	}, nil
}

func TestCacheHandler_UpdateCacheConfig_ShouldPersistColdTierFields(t *testing.T) {
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
	router.PUT("/api/admin/cache/config", handler.UpdateCacheConfig)

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
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/cache/config", bytes.NewReader(raw))
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

func TestCacheHandler_VectorTierEndpoints_ShouldWork(t *testing.T) {
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
	router.GET("/api/admin/cache/vector/tier/stats", handler.GetVectorTierStats)
	router.POST("/api/admin/cache/vector/tier/migrate", handler.TriggerVectorTierMigrate)
	router.POST("/api/admin/cache/vector/tier/promote", handler.PromoteVectorTierEntry)

	statsReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/vector/tier/stats", nil)
	statsW := httptest.NewRecorder()
	router.ServeHTTP(statsW, statsReq)
	if statsW.Code != http.StatusOK {
		t.Fatalf("expected stats 200, got %d body=%s", statsW.Code, statsW.Body.String())
	}

	migrateReq := httptest.NewRequest(http.MethodPost, "/api/admin/cache/vector/tier/migrate", nil)
	migrateW := httptest.NewRecorder()
	router.ServeHTTP(migrateW, migrateReq)
	if migrateW.Code != http.StatusOK {
		t.Fatalf("expected migrate 200, got %d body=%s", migrateW.Code, migrateW.Body.String())
	}

	promoteBody := []byte(`{"cache_key":"intent:qa:key=1"}`)
	promoteReq := httptest.NewRequest(http.MethodPost, "/api/admin/cache/vector/tier/promote", bytes.NewReader(promoteBody))
	promoteReq.Header.Set("Content-Type", "application/json")
	promoteW := httptest.NewRecorder()
	router.ServeHTTP(promoteW, promoteReq)
	if promoteW.Code != http.StatusOK {
		t.Fatalf("expected promote 200, got %d body=%s", promoteW.Code, promoteW.Body.String())
	}
}

func TestCacheHandler_UpdateCacheConfig_ShouldPersistColdFieldsToConfigFile(t *testing.T) {
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
	router.PUT("/api/admin/cache/config", handler.UpdateCacheConfig)

	body := map[string]any{
		"cold_vector_enabled":               true,
		"cold_vector_query_enabled":         false,
		"cold_vector_backend":               "qdrant",
		"cold_vector_similarity_threshold":  0.9,
		"cold_vector_top_k":                 3,
		"hot_memory_high_watermark_percent": 79,
		"hot_memory_relief_percent":         68,
		"hot_to_cold_batch_size":            256,
		"hot_to_cold_interval_seconds":      22,
		"cold_vector_qdrant_url":            "http://127.0.0.1:6333",
		"cold_vector_qdrant_collection":     "tier_v22",
		"cold_vector_qdrant_timeout_ms":     1800,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/cache/config", bytes.NewReader(raw))
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

	if enabled, _ := vectorCache["cold_vector_enabled"].(bool); !enabled {
		t.Fatalf("expected cold_vector_enabled=true in persisted config, got %#v", vectorCache["cold_vector_enabled"])
	}
	if backend, _ := vectorCache["cold_vector_backend"].(string); backend != "qdrant" {
		t.Fatalf("expected backend=qdrant in persisted config, got %#v", vectorCache["cold_vector_backend"])
	}
	if topK, ok := vectorCache["cold_vector_top_k"].(float64); !ok || int(topK) != 3 {
		t.Fatalf("expected cold_vector_top_k=3 in persisted config, got %#v", vectorCache["cold_vector_top_k"])
	}
}
