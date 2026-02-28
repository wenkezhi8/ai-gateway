package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TieredVectorStoreConfig controls hot/cold vector tiering behavior.
type TieredVectorStoreConfig struct {
	ColdVectorEnabled             bool
	ColdVectorQueryEnabled        bool
	ColdVectorBackend             string
	ColdVectorDualWriteEnabled    bool
	ColdVectorSimilarityThreshold float64
	ColdVectorTopK                int

	HotMemoryHighWatermarkPercent float64
	HotMemoryReliefPercent        float64
	HotToColdBatchSize            int
	HotToColdInterval             time.Duration
	HotToColdMaxBatchesPerRound   int
}

// TierMigrationResult captures one migration round result.
type TierMigrationResult struct {
	Triggered                bool    `json:"triggered"`
	BeforeMemoryUsagePercent float64 `json:"before_memory_usage_percent"`
	AfterMemoryUsagePercent  float64 `json:"after_memory_usage_percent"`
	BatchRuns                int     `json:"batch_runs"`
	MigratedCount            int     `json:"migrated_count"`
	FailedCount              int     `json:"failed_count"`
}

// TieredVectorStoreStats captures runtime status for admin APIs.
type TieredVectorStoreStats struct {
	Enabled                       bool                            `json:"enabled"`
	ColdVectorEnabled             bool                            `json:"cold_vector_enabled"`
	ColdVectorQueryEnabled        bool                            `json:"cold_vector_query_enabled"`
	ColdVectorBackend             string                          `json:"cold_vector_backend"`
	ColdVectorDualWriteEnabled    bool                            `json:"cold_vector_dual_write_enabled"`
	ColdVectorSimilarityThreshold float64                         `json:"cold_vector_similarity_threshold"`
	ColdVectorTopK                int                             `json:"cold_vector_top_k"`
	HotMemoryUsagePercent         float64                         `json:"hot_memory_usage_percent"`
	HotMemoryHighWatermarkPercent float64                         `json:"hot_memory_high_watermark_percent"`
	HotMemoryReliefPercent        float64                         `json:"hot_memory_relief_percent"`
	HotToColdBatchSize            int                             `json:"hot_to_cold_batch_size"`
	HotToColdIntervalSeconds      int                             `json:"hot_to_cold_interval_seconds"`
	HotToColdMaxBatchesPerRound   int                             `json:"hot_to_cold_max_batches_per_round"`
	MigrationRuns                 int64                           `json:"migration_runs"`
	MigrationMoved                int64                           `json:"migration_moved"`
	MigrationFailed               int64                           `json:"migration_failed"`
	PromoteSuccess                int64                           `json:"promote_success"`
	PromoteFailed                 int64                           `json:"promote_failed"`
	ColdBackends                  map[string]ColdVectorStoreStats `json:"cold_backends"`
}

type tieredCounters struct {
	migrationRuns   int64
	migrationMoved  int64
	migrationFailed int64
	promoteSuccess  int64
	promoteFailed   int64
}

// TieredVectorStore orchestrates hot/cold vector caches.
type TieredVectorStore struct {
	hot    VectorCacheStore
	hotOps HotVectorTierControl

	mu         sync.RWMutex
	coldStores map[string]ColdVectorStore
	cfg        TieredVectorStoreConfig
	counters   tieredCounters
	lastErrors map[string]string

	workerOnce sync.Once
}

// DefaultTieredVectorStoreConfig returns safe defaults for production.
func DefaultTieredVectorStoreConfig() TieredVectorStoreConfig {
	return TieredVectorStoreConfig{
		ColdVectorEnabled:             false,
		ColdVectorQueryEnabled:        true,
		ColdVectorBackend:             ColdVectorBackendSQLite,
		ColdVectorDualWriteEnabled:    false,
		ColdVectorSimilarityThreshold: 0.92,
		ColdVectorTopK:                1,
		HotMemoryHighWatermarkPercent: 75,
		HotMemoryReliefPercent:        65,
		HotToColdBatchSize:            500,
		HotToColdInterval:             30 * time.Second,
		HotToColdMaxBatchesPerRound:   8,
	}
}

// TieredConfigFromSettings converts cache settings into tiered config.
func TieredConfigFromSettings(settings CacheSettings) TieredVectorStoreConfig {
	cfg := DefaultTieredVectorStoreConfig()
	cfg.ColdVectorEnabled = settings.ColdVectorEnabled
	cfg.ColdVectorQueryEnabled = settings.ColdVectorQueryEnabled
	if strings.TrimSpace(settings.ColdVectorBackend) != "" {
		cfg.ColdVectorBackend = strings.TrimSpace(strings.ToLower(settings.ColdVectorBackend))
	}
	cfg.ColdVectorDualWriteEnabled = settings.ColdVectorDualWriteEnabled
	if settings.ColdVectorSimilarityThreshold > 0 {
		cfg.ColdVectorSimilarityThreshold = settings.ColdVectorSimilarityThreshold
	}
	if settings.ColdVectorTopK > 0 {
		cfg.ColdVectorTopK = settings.ColdVectorTopK
	}
	if settings.HotMemoryHighWatermarkPercent > 0 {
		cfg.HotMemoryHighWatermarkPercent = settings.HotMemoryHighWatermarkPercent
	}
	if settings.HotMemoryReliefPercent > 0 {
		cfg.HotMemoryReliefPercent = settings.HotMemoryReliefPercent
	}
	if settings.HotToColdBatchSize > 0 {
		cfg.HotToColdBatchSize = settings.HotToColdBatchSize
	}
	if settings.HotToColdIntervalSeconds > 0 {
		cfg.HotToColdInterval = time.Duration(settings.HotToColdIntervalSeconds) * time.Second
	}
	cfg.normalize()
	return cfg
}

// NewTieredVectorStore creates a new tiered vector store orchestrator.
func NewTieredVectorStore(hot VectorCacheStore, coldStores map[string]ColdVectorStore, cfg TieredVectorStoreConfig) *TieredVectorStore {
	cfg.normalize()
	storeMap := make(map[string]ColdVectorStore, len(coldStores))
	for backend, store := range coldStores {
		if store == nil {
			continue
		}
		storeMap[strings.ToLower(strings.TrimSpace(backend))] = store
	}

	t := &TieredVectorStore{
		hot:        hot,
		coldStores: storeMap,
		cfg:        cfg,
		lastErrors: map[string]string{},
	}
	if hotOps, ok := hot.(HotVectorTierControl); ok {
		t.hotOps = hotOps
	}
	return t
}

func (c *TieredVectorStoreConfig) normalize() {
	def := DefaultTieredVectorStoreConfig()
	if c.ColdVectorBackend == "" {
		c.ColdVectorBackend = def.ColdVectorBackend
	}
	c.ColdVectorBackend = strings.ToLower(strings.TrimSpace(c.ColdVectorBackend))
	if c.ColdVectorBackend != ColdVectorBackendSQLite && c.ColdVectorBackend != ColdVectorBackendQdrant {
		c.ColdVectorBackend = def.ColdVectorBackend
	}
	if c.ColdVectorSimilarityThreshold <= 0 || c.ColdVectorSimilarityThreshold > 1 {
		c.ColdVectorSimilarityThreshold = def.ColdVectorSimilarityThreshold
	}
	if c.ColdVectorTopK <= 0 {
		c.ColdVectorTopK = def.ColdVectorTopK
	}
	if c.HotMemoryHighWatermarkPercent <= 0 || c.HotMemoryHighWatermarkPercent > 100 {
		c.HotMemoryHighWatermarkPercent = def.HotMemoryHighWatermarkPercent
	}
	if c.HotMemoryReliefPercent <= 0 || c.HotMemoryReliefPercent > 100 {
		c.HotMemoryReliefPercent = def.HotMemoryReliefPercent
	}
	if c.HotMemoryReliefPercent >= c.HotMemoryHighWatermarkPercent {
		c.HotMemoryReliefPercent = c.HotMemoryHighWatermarkPercent - 5
		if c.HotMemoryReliefPercent <= 0 {
			c.HotMemoryReliefPercent = c.HotMemoryHighWatermarkPercent * 0.8
		}
	}
	if c.HotToColdBatchSize <= 0 {
		c.HotToColdBatchSize = def.HotToColdBatchSize
	}
	if c.HotToColdInterval <= 0 {
		c.HotToColdInterval = def.HotToColdInterval
	}
	if c.HotToColdMaxBatchesPerRound <= 0 {
		c.HotToColdMaxBatchesPerRound = def.HotToColdMaxBatchesPerRound
	}
}

// UpdateConfig applies latest runtime settings.
func (t *TieredVectorStore) UpdateConfig(cfg TieredVectorStoreConfig) {
	cfg.normalize()
	t.mu.Lock()
	t.cfg = cfg
	t.mu.Unlock()
}

// SetColdStore sets or replaces one cold backend implementation.
func (t *TieredVectorStore) SetColdStore(backend string, store ColdVectorStore) {
	if t == nil {
		return
	}
	backend = strings.ToLower(strings.TrimSpace(backend))
	if backend == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.coldStores == nil {
		t.coldStores = map[string]ColdVectorStore{}
	}
	t.coldStores[backend] = store
}

// EnsureIndex ensures hot index and cold schema.
func (t *TieredVectorStore) EnsureIndex(ctx context.Context) error {
	if t == nil {
		return nil
	}
	if t.hot != nil {
		if err := t.hot.EnsureIndex(ctx); err != nil {
			return err
		}
	}
	for backend, store := range t.collectColdTargets(true) {
		if store == nil {
			continue
		}
		if err := store.EnsureSchema(ctx); err != nil {
			t.recordColdError(backend, err)
		}
	}
	return nil
}

// RebuildIndex rebuilds hot index only. Cold schema does not require rebuild.
func (t *TieredVectorStore) RebuildIndex(ctx context.Context) error {
	if t == nil || t.hot == nil {
		return nil
	}
	return t.hot.RebuildIndex(ctx)
}

// GetExact reads hot exact first, then optional cold exact.
func (t *TieredVectorStore) GetExact(ctx context.Context, cacheKey string) (*VectorCacheDocument, error) {
	if t == nil {
		return nil, nil
	}
	if t.hot != nil {
		doc, err := t.hot.GetExact(ctx, cacheKey)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			return doc, nil
		}
	}

	if !t.isColdQueryEnabled() {
		return nil, nil
	}
	cold := t.activeColdStore()
	if cold == nil {
		return nil, nil
	}
	doc, err := cold.GetExact(ctx, cacheKey)
	if err != nil {
		t.recordColdError(t.activeBackend(), err)
		return nil, nil
	}
	if doc != nil {
		t.promoteDocAsync(doc)
	}
	return doc, nil
}

// VectorSearch reads hot first and fail-open falls back to cold search.
func (t *TieredVectorStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]VectorSearchHit, error) {
	if t == nil {
		return []VectorSearchHit{}, nil
	}
	if t.hot != nil {
		hits, err := t.hot.VectorSearch(ctx, intent, vector, topK, minSimilarity)
		if err != nil {
			return nil, err
		}
		if len(hits) > 0 {
			return hits, nil
		}
	}

	if !t.isColdQueryEnabled() || len(vector) == 0 {
		return []VectorSearchHit{}, nil
	}

	cold := t.activeColdStore()
	if cold == nil {
		return []VectorSearchHit{}, nil
	}

	cfg := t.getConfig()
	coldTopK := cfg.ColdVectorTopK
	if coldTopK <= 0 {
		coldTopK = topK
	}
	coldThreshold := cfg.ColdVectorSimilarityThreshold
	if coldThreshold <= 0 {
		coldThreshold = minSimilarity
	}
	hits, err := cold.VectorSearch(ctx, intent, vector, coldTopK, coldThreshold)
	if err != nil {
		t.recordColdError(t.activeBackend(), err)
		return []VectorSearchHit{}, nil
	}
	if len(hits) > 0 {
		t.promoteByCacheKeyAsync(hits[0].CacheKey)
	}
	return hits, nil
}

// Upsert writes hot first and archives into cold based on config.
func (t *TieredVectorStore) Upsert(ctx context.Context, doc *VectorCacheDocument) error {
	if t == nil || doc == nil {
		return nil
	}
	now := time.Now().Unix()
	if doc.CreateTS <= 0 {
		doc.CreateTS = now
	}
	if doc.LastHitTS <= 0 {
		doc.LastHitTS = doc.CreateTS
	}
	if strings.TrimSpace(doc.Tier) == "" {
		doc.Tier = VectorTierHot
	}
	if t.hot != nil {
		if err := t.hot.Upsert(ctx, doc); err != nil {
			return err
		}
	}

	if !t.isColdEnabled() {
		return nil
	}

	coldDoc := *doc
	coldDoc.Tier = VectorTierCold
	for backend, store := range t.collectColdTargets(false) {
		if store == nil {
			continue
		}
		if err := store.Upsert(ctx, &coldDoc); err != nil {
			t.recordColdError(backend, err)
		}
	}
	return nil
}

// Delete removes one key from hot and configured cold stores.
func (t *TieredVectorStore) Delete(ctx context.Context, cacheKey string) error {
	if t == nil {
		return nil
	}
	var firstErr error
	if t.hot != nil {
		if err := t.hot.Delete(ctx, cacheKey); err != nil {
			firstErr = err
		}
	}
	if t.isColdEnabled() {
		for backend, store := range t.collectColdTargets(false) {
			if store == nil {
				continue
			}
			if err := store.Delete(ctx, cacheKey); err != nil {
				t.recordColdError(backend, err)
			}
		}
	}
	return firstErr
}

// TouchTTL updates hot TTL only.
func (t *TieredVectorStore) TouchTTL(ctx context.Context, cacheKey string, ttlSec int64) error {
	if t == nil || t.hot == nil {
		return nil
	}
	return t.hot.TouchTTL(ctx, cacheKey, ttlSec)
}

// Stats returns hot stats to keep compatibility with existing API.
func (t *TieredVectorStore) Stats(ctx context.Context) (VectorStoreStats, error) {
	if t == nil || t.hot == nil {
		return VectorStoreStats{Enabled: false}, nil
	}
	return t.hot.Stats(ctx)
}

// TierStats returns hot/cold tier status and counters.
func (t *TieredVectorStore) TierStats(ctx context.Context) (TieredVectorStoreStats, error) {
	cfg := t.getConfig()
	hotUsage := 0.0
	if t.hotOps != nil {
		if usage, err := t.hotOps.MemoryUsagePercent(ctx); err == nil {
			hotUsage = usage
		}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	coldStats := make(map[string]ColdVectorStoreStats, len(t.coldStores))
	for backend, store := range t.coldStores {
		if store == nil {
			continue
		}
		stats, err := store.Stats(ctx)
		if err != nil {
			stats = ColdVectorStoreStats{
				Backend:   backend,
				Available: false,
				LastError: err.Error(),
			}
		}
		if lastErr, ok := t.lastErrors[backend]; ok && lastErr != "" {
			stats.LastError = lastErr
		}
		coldStats[backend] = stats
	}

	return TieredVectorStoreStats{
		Enabled:                       t.hot != nil,
		ColdVectorEnabled:             cfg.ColdVectorEnabled,
		ColdVectorQueryEnabled:        cfg.ColdVectorQueryEnabled,
		ColdVectorBackend:             cfg.ColdVectorBackend,
		ColdVectorDualWriteEnabled:    cfg.ColdVectorDualWriteEnabled,
		ColdVectorSimilarityThreshold: cfg.ColdVectorSimilarityThreshold,
		ColdVectorTopK:                cfg.ColdVectorTopK,
		HotMemoryUsagePercent:         hotUsage,
		HotMemoryHighWatermarkPercent: cfg.HotMemoryHighWatermarkPercent,
		HotMemoryReliefPercent:        cfg.HotMemoryReliefPercent,
		HotToColdBatchSize:            cfg.HotToColdBatchSize,
		HotToColdIntervalSeconds:      int(cfg.HotToColdInterval.Seconds()),
		HotToColdMaxBatchesPerRound:   cfg.HotToColdMaxBatchesPerRound,
		MigrationRuns:                 t.counters.migrationRuns,
		MigrationMoved:                t.counters.migrationMoved,
		MigrationFailed:               t.counters.migrationFailed,
		PromoteSuccess:                t.counters.promoteSuccess,
		PromoteFailed:                 t.counters.promoteFailed,
		ColdBackends:                  coldStats,
	}, nil
}

// TriggerMigrate runs one migration round controlled by memory watermark.
func (t *TieredVectorStore) TriggerMigrate(ctx context.Context) (TierMigrationResult, error) {
	result := TierMigrationResult{}
	if t == nil {
		return result, nil
	}
	if !t.isColdEnabled() || t.hot == nil || t.hotOps == nil {
		return result, nil
	}

	activeBackend := t.activeBackend()
	cold := t.activeColdStore()
	if cold == nil {
		return result, fmt.Errorf("cold backend %q is not configured", activeBackend)
	}

	cfg := t.getConfig()
	usage, err := t.hotOps.MemoryUsagePercent(ctx)
	if err != nil {
		return result, err
	}
	result.BeforeMemoryUsagePercent = usage
	result.AfterMemoryUsagePercent = usage
	if usage < cfg.HotMemoryHighWatermarkPercent {
		return result, nil
	}

	result.Triggered = true
	t.mu.Lock()
	t.counters.migrationRuns++
	t.mu.Unlock()

	for round := 0; round < cfg.HotToColdMaxBatchesPerRound; round++ {
		candidates, err := t.hotOps.ListMigrationCandidates(ctx, cfg.HotToColdBatchSize)
		if err != nil {
			return result, err
		}
		if len(candidates) == 0 {
			break
		}

		result.BatchRuns++
		for _, doc := range candidates {
			if doc == nil || strings.TrimSpace(doc.CacheKey) == "" {
				continue
			}
			coldDoc := *doc
			coldDoc.Tier = VectorTierCold
			coldDoc.MigrateTS = time.Now().Unix()
			if coldDoc.LastHitTS <= 0 {
				coldDoc.LastHitTS = coldDoc.CreateTS
			}

			if err := cold.Upsert(ctx, &coldDoc); err != nil {
				result.FailedCount++
				t.recordColdError(activeBackend, err)
				continue
			}
			if err := t.hot.Delete(ctx, doc.CacheKey); err != nil {
				result.FailedCount++
				continue
			}
			result.MigratedCount++
		}

		usage, err = t.hotOps.MemoryUsagePercent(ctx)
		if err == nil {
			result.AfterMemoryUsagePercent = usage
		}
		if usage <= cfg.HotMemoryReliefPercent {
			break
		}
	}

	t.mu.Lock()
	t.counters.migrationMoved += int64(result.MigratedCount)
	t.counters.migrationFailed += int64(result.FailedCount)
	t.mu.Unlock()

	return result, nil
}

// Promote loads one doc from cold and writes back to hot.
func (t *TieredVectorStore) Promote(ctx context.Context, cacheKey string) error {
	if t == nil || strings.TrimSpace(cacheKey) == "" {
		return nil
	}
	if t.hot == nil {
		return errors.New("hot vector store not initialized")
	}
	cold := t.activeColdStore()
	if cold == nil {
		return errors.New("cold vector store not initialized")
	}
	doc, err := cold.GetExact(ctx, cacheKey)
	if err != nil {
		t.recordColdError(t.activeBackend(), err)
		t.countPromote(false)
		return err
	}
	if doc == nil {
		t.countPromote(false)
		return fmt.Errorf("cache_key %s not found in cold tier", cacheKey)
	}
	doc.Tier = VectorTierHot
	doc.MigrateTS = 0
	doc.LastHitTS = time.Now().Unix()
	if err := t.hot.Upsert(ctx, doc); err != nil {
		t.countPromote(false)
		return err
	}
	if doc.TTLSec > 0 {
		_ = t.hot.TouchTTL(ctx, doc.CacheKey, doc.TTLSec)
	}
	t.countPromote(true)
	return nil
}

// StartHotToColdWorker starts periodic memory watermark migration.
func (t *TieredVectorStore) StartHotToColdWorker(ctx context.Context) {
	if t == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	t.workerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(t.getConfig().HotToColdInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_, _ = t.TriggerMigrate(ctx)
				}
			}
		}()
	})
}

func (t *TieredVectorStore) promoteByCacheKeyAsync(cacheKey string) {
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		_ = t.Promote(ctx, cacheKey)
	}()
}

func (t *TieredVectorStore) promoteDocAsync(doc *VectorCacheDocument) {
	if doc == nil || t.hot == nil {
		return
	}
	copyDoc := *doc
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		copyDoc.Tier = VectorTierHot
		copyDoc.LastHitTS = time.Now().Unix()
		if err := t.hot.Upsert(ctx, &copyDoc); err != nil {
			t.countPromote(false)
			return
		}
		if copyDoc.TTLSec > 0 {
			_ = t.hot.TouchTTL(ctx, copyDoc.CacheKey, copyDoc.TTLSec)
		}
		t.countPromote(true)
	}()
}

func (t *TieredVectorStore) countPromote(success bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if success {
		t.counters.promoteSuccess++
		return
	}
	t.counters.promoteFailed++
}

func (t *TieredVectorStore) recordColdError(backend string, err error) {
	if strings.TrimSpace(backend) == "" || err == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.lastErrors == nil {
		t.lastErrors = map[string]string{}
	}
	t.lastErrors[backend] = err.Error()
}

func (t *TieredVectorStore) getConfig() TieredVectorStoreConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cfg
}

func (t *TieredVectorStore) isColdEnabled() bool {
	return t.getConfig().ColdVectorEnabled
}

func (t *TieredVectorStore) isColdQueryEnabled() bool {
	cfg := t.getConfig()
	return cfg.ColdVectorEnabled && cfg.ColdVectorQueryEnabled
}

func (t *TieredVectorStore) activeBackend() string {
	return t.getConfig().ColdVectorBackend
}

func (t *TieredVectorStore) activeColdStore() ColdVectorStore {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.coldStores == nil {
		return nil
	}
	return t.coldStores[t.cfg.ColdVectorBackend]
}

func (t *TieredVectorStore) collectColdTargets(includeDisabled bool) map[string]ColdVectorStore {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make(map[string]ColdVectorStore)
	if !includeDisabled && !t.cfg.ColdVectorEnabled {
		return result
	}
	if t.coldStores == nil {
		return result
	}

	active := t.cfg.ColdVectorBackend
	if store := t.coldStores[active]; store != nil {
		result[active] = store
	}
	if t.cfg.ColdVectorDualWriteEnabled {
		for backend, store := range t.coldStores {
			if store == nil {
				continue
			}
			result[backend] = store
		}
	}
	return result
}
