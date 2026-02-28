package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakeHotTierStore struct {
	exactDocs         map[string]*VectorCacheDocument
	vectorHits        []VectorSearchHit
	upserts           []*VectorCacheDocument
	deleted           []string
	memoryUsage       float64
	migrationDocs     []*VectorCacheDocument
	memoryUsageSeq    []float64
	memoryUsageSeqIdx int
	migrateCallCount  atomic.Int32
}

func (f *fakeHotTierStore) EnsureIndex(ctx context.Context) error { return nil }
func (f *fakeHotTierStore) RebuildIndex(ctx context.Context) error { return nil }
func (f *fakeHotTierStore) GetExact(ctx context.Context, cacheKey string) (*VectorCacheDocument, error) {
	if f.exactDocs == nil {
		return nil, nil
	}
	return f.exactDocs[cacheKey], nil
}
func (f *fakeHotTierStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]VectorSearchHit, error) {
	return f.vectorHits, nil
}
func (f *fakeHotTierStore) Upsert(ctx context.Context, doc *VectorCacheDocument) error {
	if doc == nil {
		return nil
	}
	cp := *doc
	f.upserts = append(f.upserts, &cp)
	return nil
}
func (f *fakeHotTierStore) Delete(ctx context.Context, cacheKey string) error {
	f.deleted = append(f.deleted, cacheKey)
	return nil
}
func (f *fakeHotTierStore) TouchTTL(ctx context.Context, cacheKey string, ttlSec int64) error { return nil }
func (f *fakeHotTierStore) Stats(ctx context.Context) (VectorStoreStats, error) {
	return VectorStoreStats{Enabled: true, IndexName: "hot"}, nil
}
func (f *fakeHotTierStore) MemoryUsagePercent(ctx context.Context) (float64, error) {
	if len(f.memoryUsageSeq) == 0 {
		return f.memoryUsage, nil
	}
	idx := f.memoryUsageSeqIdx
	if idx >= len(f.memoryUsageSeq) {
		idx = len(f.memoryUsageSeq) - 1
	}
	f.memoryUsageSeqIdx++
	return f.memoryUsageSeq[idx], nil
}
func (f *fakeHotTierStore) ListMigrationCandidates(ctx context.Context, batchSize int) ([]*VectorCacheDocument, error) {
	f.migrateCallCount.Add(1)
	if len(f.migrationDocs) == 0 {
		return nil, nil
	}
	if batchSize <= 0 || batchSize >= len(f.migrationDocs) {
		return f.migrationDocs, nil
	}
	return f.migrationDocs[:batchSize], nil
}

type fakeColdStore struct {
	backend      string
	exactDocs    map[string]*VectorCacheDocument
	searchHits   []VectorSearchHit
	upserts      []*VectorCacheDocument
	upsertErr    error
	vectorErr    error
	ensureErr    error
	statsEntries int64
}

func (f *fakeColdStore) EnsureSchema(ctx context.Context) error { return f.ensureErr }
func (f *fakeColdStore) Upsert(ctx context.Context, doc *VectorCacheDocument) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	if doc != nil {
		cp := *doc
		f.upserts = append(f.upserts, &cp)
		if f.exactDocs == nil {
			f.exactDocs = map[string]*VectorCacheDocument{}
		}
		f.exactDocs[doc.CacheKey] = &cp
	}
	return nil
}
func (f *fakeColdStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]VectorSearchHit, error) {
	if f.vectorErr != nil {
		return nil, f.vectorErr
	}
	return f.searchHits, nil
}
func (f *fakeColdStore) GetExact(ctx context.Context, cacheKey string) (*VectorCacheDocument, error) {
	if f.exactDocs == nil {
		return nil, nil
	}
	return f.exactDocs[cacheKey], nil
}
func (f *fakeColdStore) Delete(ctx context.Context, cacheKey string) error { return nil }
func (f *fakeColdStore) Stats(ctx context.Context) (ColdVectorStoreStats, error) {
	return ColdVectorStoreStats{
		Backend:   f.backend,
		Available: true,
		Entries:   f.statsEntries,
	}, nil
}

func TestTieredVectorStore_VectorSearch_HotHit_ShouldSkipCold(t *testing.T) {
	hot := &fakeHotTierStore{
		vectorHits: []VectorSearchHit{
			{CacheKey: "intent:calc:expr=1+1", Similarity: 0.99},
		},
	}
	cold := &fakeColdStore{backend: ColdVectorBackendSQLite}
	store := NewTieredVectorStore(hot, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredVectorStoreConfig{
		ColdVectorEnabled:      true,
		ColdVectorQueryEnabled: true,
		ColdVectorBackend:      ColdVectorBackendSQLite,
	})

	hits, err := store.VectorSearch(context.Background(), "calc", []float64{0.1, 0.2}, 1, 0.95)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hot hit, got %d", len(hits))
	}
	if len(cold.upserts) != 0 {
		t.Fatalf("expected cold upsert not called, got %d", len(cold.upserts))
	}
}

func TestTieredVectorStore_VectorSearch_ColdHit_ShouldPromoteToHot(t *testing.T) {
	hot := &fakeHotTierStore{}
	coldDoc := &VectorCacheDocument{
		CacheKey: "intent:qa:key=what-is-cache",
		Intent:   "qa",
		Vector:   []float64{0.3, 0.4},
		Response: map[string]any{"answer": "cached"},
		TTLSec:   3600,
	}
	cold := &fakeColdStore{
		backend: ColdVectorBackendSQLite,
		searchHits: []VectorSearchHit{
			{CacheKey: coldDoc.CacheKey, Similarity: 0.96},
		},
		exactDocs: map[string]*VectorCacheDocument{
			coldDoc.CacheKey: coldDoc,
		},
	}
	store := NewTieredVectorStore(hot, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredVectorStoreConfig{
		ColdVectorEnabled:             true,
		ColdVectorQueryEnabled:        true,
		ColdVectorBackend:             ColdVectorBackendSQLite,
		ColdVectorSimilarityThreshold: 0.95,
	})

	hits, err := store.VectorSearch(context.Background(), "qa", []float64{0.3, 0.4}, 1, 0.97)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(hits) != 1 || hits[0].CacheKey != coldDoc.CacheKey {
		t.Fatalf("expected cold hit, got %+v", hits)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(hot.upserts) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(hot.upserts) == 0 {
		t.Fatal("expected cold hit to trigger async hot promotion")
	}
}

func TestTieredVectorStore_VectorSearch_ColdQueryDisabled_ShouldNotQueryCold(t *testing.T) {
	hot := &fakeHotTierStore{}
	cold := &fakeColdStore{
		backend:    ColdVectorBackendSQLite,
		searchHits: []VectorSearchHit{{CacheKey: "k1", Similarity: 0.99}},
	}
	store := NewTieredVectorStore(hot, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredVectorStoreConfig{
		ColdVectorEnabled:      true,
		ColdVectorQueryEnabled: false,
		ColdVectorBackend:      ColdVectorBackendSQLite,
	})

	hits, err := store.VectorSearch(context.Background(), "qa", []float64{0.1, 0.2}, 1, 0.9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("expected no hits when cold query disabled, got %+v", hits)
	}
}

func TestTieredVectorStore_Upsert_DualWrite_ShouldWriteToTwoBackends(t *testing.T) {
	hot := &fakeHotTierStore{}
	sqliteStore := &fakeColdStore{backend: ColdVectorBackendSQLite}
	qdrantStore := &fakeColdStore{backend: ColdVectorBackendQdrant}
	store := NewTieredVectorStore(hot, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: sqliteStore,
		ColdVectorBackendQdrant: qdrantStore,
	}, TieredVectorStoreConfig{
		ColdVectorEnabled:           true,
		ColdVectorBackend:           ColdVectorBackendSQLite,
		ColdVectorDualWriteEnabled:  true,
		ColdVectorQueryEnabled:      true,
		HotToColdBatchSize:          100,
		HotToColdInterval:           30 * time.Second,
		HotMemoryHighWatermarkPercent: 75,
		HotMemoryReliefPercent:      65,
	})

	err := store.Upsert(context.Background(), &VectorCacheDocument{
		CacheKey: "intent:calc:expr=1+1",
		Intent:   "calc",
		Vector:   []float64{0.1, 0.2},
		Response: map[string]any{"content": "2"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sqliteStore.upserts) != 1 || len(qdrantStore.upserts) != 1 {
		t.Fatalf("expected dual-write to both cold stores, got sqlite=%d qdrant=%d", len(sqliteStore.upserts), len(qdrantStore.upserts))
	}
}

func TestTieredVectorStore_TriggerMigrate_ShouldRelieveMemoryWatermark(t *testing.T) {
	hot := &fakeHotTierStore{
		memoryUsageSeq: []float64{80, 70, 64},
		migrationDocs: []*VectorCacheDocument{
			{CacheKey: "k1", Intent: "qa", Vector: []float64{0.1}, Response: map[string]any{"x": 1}},
			{CacheKey: "k2", Intent: "qa", Vector: []float64{0.2}, Response: map[string]any{"x": 2}},
		},
	}
	cold := &fakeColdStore{backend: ColdVectorBackendSQLite}
	store := NewTieredVectorStore(hot, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredVectorStoreConfig{
		ColdVectorEnabled:               true,
		ColdVectorBackend:               ColdVectorBackendSQLite,
		HotMemoryHighWatermarkPercent:   75,
		HotMemoryReliefPercent:          65,
		HotToColdBatchSize:              1,
		HotToColdMaxBatchesPerRound:     4,
		HotToColdInterval:               30 * time.Second,
		ColdVectorSimilarityThreshold:   0.92,
		ColdVectorTopK:                  1,
		ColdVectorQueryEnabled:          true,
		ColdVectorDualWriteEnabled:      false,
	})

	result, err := store.TriggerMigrate(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.MigratedCount == 0 {
		t.Fatalf("expected migrated docs, got %+v", result)
	}
	if len(cold.upserts) == 0 {
		t.Fatalf("expected docs moved to cold store")
	}
	if len(hot.deleted) == 0 {
		t.Fatalf("expected docs removed from hot store")
	}
}

func TestTieredVectorStore_VectorSearch_ShouldFailOpenWhenColdErrors(t *testing.T) {
	hot := &fakeHotTierStore{}
	cold := &fakeColdStore{
		backend:   ColdVectorBackendSQLite,
		vectorErr: errors.New("cold failed"),
	}
	store := NewTieredVectorStore(hot, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredVectorStoreConfig{
		ColdVectorEnabled:      true,
		ColdVectorQueryEnabled: true,
		ColdVectorBackend:      ColdVectorBackendSQLite,
	})

	hits, err := store.VectorSearch(context.Background(), "qa", []float64{0.1, 0.2}, 1, 0.92)
	if err != nil {
		t.Fatalf("expected fail-open no error, got %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("expected no hits on cold failure, got %+v", hits)
	}
}

func TestTieredVectorStore_Worker_ShouldUseUpdatedIntervalWithoutRestart(t *testing.T) {
	hot := &fakeHotTierStore{
		memoryUsage: 90,
	}
	cold := &fakeColdStore{backend: ColdVectorBackendSQLite}
	store := NewTieredVectorStore(hot, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredVectorStoreConfig{
		ColdVectorEnabled:             true,
		ColdVectorBackend:             ColdVectorBackendSQLite,
		ColdVectorQueryEnabled:        false,
		HotMemoryHighWatermarkPercent: 75,
		HotMemoryReliefPercent:        65,
		HotToColdBatchSize:            1,
		HotToColdInterval:             300 * time.Millisecond,
		HotToColdMaxBatchesPerRound:   1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store.StartHotToColdWorker(ctx)

	time.Sleep(80 * time.Millisecond)
	if got := hot.migrateCallCount.Load(); got != 0 {
		t.Fatalf("expected no migration call before first long interval tick, got %d", got)
	}

	store.UpdateConfig(TieredVectorStoreConfig{
		ColdVectorEnabled:             true,
		ColdVectorBackend:             ColdVectorBackendSQLite,
		ColdVectorQueryEnabled:        false,
		HotMemoryHighWatermarkPercent: 75,
		HotMemoryReliefPercent:        65,
		HotToColdBatchSize:            1,
		HotToColdInterval:             20 * time.Millisecond,
		HotToColdMaxBatchesPerRound:   1,
	})

	deadline := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline) {
		if hot.migrateCallCount.Load() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected worker to pick new interval and run migration, call_count=%d", hot.migrateCallCount.Load())
}
