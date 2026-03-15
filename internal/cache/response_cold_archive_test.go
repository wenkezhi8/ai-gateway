package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type responseArchiveTestEmbedder struct {
	calls int32
	fail  int32
}

func (s *responseArchiveTestEmbedder) GetEmbedding(_ context.Context, text string) ([]float64, error) {
	atomic.AddInt32(&s.calls, 1)
	if atomic.LoadInt32(&s.fail) > 0 {
		atomic.AddInt32(&s.fail, -1)
		return nil, errors.New("embedding timeout")
	}
	return []float64{0.1, 0.2}, nil
}

func (s *responseArchiveTestEmbedder) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	result := make([][]float64, 0, len(texts))
	for _, text := range texts {
		vec, err := s.GetEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		result = append(result, vec)
	}
	return result, nil
}

func TestResponseColdArchiveService_NotifyWrite_ShouldArchiveReusableResponse(t *testing.T) {
	manager := NewManagerWithCache(NewMemoryCache())
	settings := manager.GetSettings()
	settings.VectorEnabled = true
	settings.VectorOllamaEmbeddingDimension = 2
	settings.ColdVectorEnabled = true
	settings.ColdArchiveEnabled = true
	settings.ColdArchiveMode = ColdArchiveModeReusable
	settings.ColdArchiveNearExpirySeconds = 120
	settings.ColdArchiveScanIntervalSeconds = 1
	manager.UpdateSettings(settings)

	cold := &fakeColdStore{backend: ColdVectorBackendSQLite}
	tiered := NewTieredVectorStore(nil, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredConfigFromSettings(settings))

	embedder := &responseArchiveTestEmbedder{}
	service := NewResponseColdArchiveService(manager, tiered, func(CacheSettings) EmbeddingProvider {
		return embedder
	})
	service.Start(context.Background())

	resp := &CachedResponse{
		Body:      []byte(`{"choices":[{"message":{"role":"assistant","content":"缓存答案"}}]}`),
		CreatedAt: time.Now(),
		Provider:  "openai",
		Model:     "gpt-4o-mini",
		Prompt:    "解释缓存击穿",
		TaskType:  "fact",
	}
	if err := manager.ResponseCache.SetWithTTL(context.Background(), "ai-response:test", resp, 5*time.Minute); err != nil {
		t.Fatalf("seed response cache: %v", err)
	}

	service.NotifyWrite("ai-response:test", resp, 5*time.Minute)

	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		if cold.UpsertCount() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("expected reusable response to be archived into cold store")
}

func TestResponseColdArchiveService_NotifyWrite_ShouldRetryFailedEmbedding(t *testing.T) {
	manager := NewManagerWithCache(NewMemoryCache())
	settings := manager.GetSettings()
	settings.VectorEnabled = true
	settings.VectorOllamaEmbeddingDimension = 2
	settings.ColdVectorEnabled = true
	settings.ColdArchiveEnabled = true
	settings.ColdArchiveMode = ColdArchiveModeAll
	settings.ColdArchiveNearExpirySeconds = 120
	settings.ColdArchiveScanIntervalSeconds = 1
	manager.UpdateSettings(settings)

	cold := &fakeColdStore{backend: ColdVectorBackendSQLite}
	tiered := NewTieredVectorStore(nil, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredConfigFromSettings(settings))

	embedder := &responseArchiveTestEmbedder{fail: 1}
	service := NewResponseColdArchiveService(manager, tiered, func(CacheSettings) EmbeddingProvider {
		return embedder
	})
	service.retryDelay = 30 * time.Millisecond
	service.Start(context.Background())

	resp := &CachedResponse{
		Body:      []byte(`{"choices":[{"message":{"role":"assistant","content":"重试后成功"}}]}`),
		CreatedAt: time.Now(),
		Provider:  "openai",
		Model:     "gpt-4o-mini",
		Prompt:    "解释缓存雪崩",
		TaskType:  "chat",
	}

	service.NotifyWrite("ai-response:retry", resp, 5*time.Minute)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cold.UpsertCount() > 0 {
			if atomic.LoadInt32(&embedder.calls) < 2 {
				t.Fatalf("expected at least 2 embedding attempts, got %d", embedder.calls)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("expected response archive retry to succeed")
}

func TestResponseColdArchiveService_ScanExpiringResponses_ShouldArchiveDueEntries(t *testing.T) {
	manager := NewManagerWithCache(NewMemoryCache())
	settings := manager.GetSettings()
	settings.VectorEnabled = true
	settings.VectorOllamaEmbeddingDimension = 2
	settings.ColdVectorEnabled = true
	settings.ColdArchiveEnabled = true
	settings.ColdArchiveMode = ColdArchiveModeAll
	settings.ColdArchiveNearExpirySeconds = 10
	settings.ColdArchiveScanIntervalSeconds = 1
	manager.UpdateSettings(settings)

	cold := &fakeColdStore{backend: ColdVectorBackendSQLite}
	tiered := NewTieredVectorStore(nil, map[string]ColdVectorStore{
		ColdVectorBackendSQLite: cold,
	}, TieredConfigFromSettings(settings))

	service := NewResponseColdArchiveService(manager, tiered, func(CacheSettings) EmbeddingProvider {
		return &responseArchiveTestEmbedder{}
	})
	service.Start(context.Background())

	resp := &CachedResponse{
		Body:      []byte(`{"choices":[{"message":{"role":"assistant","content":"即将过期"}}]}`),
		CreatedAt: time.Now(),
		Provider:  "openai",
		Model:     "gpt-4o-mini",
		Prompt:    "解释LRU",
		TaskType:  "fact",
	}
	if err := manager.ResponseCache.SetWithTTL(context.Background(), "ai-response:scan", resp, 2*time.Second); err != nil {
		t.Fatalf("seed response cache: %v", err)
	}

	archived, err := service.ScanExpiringResponses(context.Background())
	if err != nil {
		t.Fatalf("scan expiring responses: %v", err)
	}
	if archived == 0 {
		t.Fatal("expected expiring response to be enqueued")
	}

	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		if cold.UpsertCount() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("expected scanned expiring response to be archived")
}
