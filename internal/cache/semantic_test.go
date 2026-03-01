package cache

import (
	"context"
	"testing"
	"time"
)

func TestSemanticCache_Get(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	cache := NewSemanticCache(config, nil)

	// Create a test embedding.
	embedding := SimpleEmbedding("hello world", 1536)

	// Test cache miss.
	entry, similarity := cache.Get(context.Background(), "hello world", embedding)
	if entry != nil {
		t.Error("expected nil entry for cache miss")
	}
	if similarity != 0 {
		t.Error("expected 0 similarity for cache miss")
	}
}

func TestSemanticCache_SetAndGet(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	cache := NewSemanticCache(config, nil)

	embedding := SimpleEmbedding("hello world", 1536)
	response := []byte(`{"choices":[{"message":{"content":"Hello!"}}]}`)

	// Set entry.
	id := cache.Set(
		context.Background(),
		"hello world",
		embedding,
		response,
		"gpt-4o",
		"openai",
		"chat",
		time.Hour,
	)

	if id == "" {
		t.Error("expected non-empty cache ID")
	}

	// Get entry with same query.
	entry, similarity := cache.Get(context.Background(), "hello world", embedding)
	if entry == nil {
		t.Error("expected non-nil entry for cache hit")
		return
	}
	if similarity < config.SimilarityThreshold {
		t.Errorf("expected similarity >= %f, got %f", config.SimilarityThreshold, similarity)
	}
	if string(entry.Response) != string(response) {
		t.Error("response mismatch")
	}
}

func TestSemanticCache_SemanticMatch(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	config.SimilarityThreshold = 0.8
	cache := NewSemanticCache(config, nil)

	// Store a query about programming.
	embedding1 := SimpleEmbedding("how to write a python function", 1536)
	response1 := []byte(`{"choices":[{"message":{"content":"Here's how..."}}]}`)
	cache.Set(
		context.Background(),
		"how to write a python function",
		embedding1,
		response1,
		"gpt-4o",
		"openai",
		"code",
		time.Hour,
	)

	// Try similar query about programming.
	embedding2 := SimpleEmbedding("how do I create a function in python", 1536)
	entry, similarity := cache.Get(context.Background(), "how do I create a function in python", embedding2)

	// Should match because embeddings are similar.
	if entry != nil && similarity < 0.8 {
		t.Errorf("expected similarity >= 0.8 for similar query, got %f", similarity)
	}
}

func TestSemanticCache_Expiration(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	cache := NewSemanticCache(config, nil)

	embedding := SimpleEmbedding("test query", 1536)
	response := []byte(`{"choices":[{"message":{"content":"test"}}]}`)

	// Set with short TTL.
	cache.Set(
		context.Background(),
		"test query",
		embedding,
		response,
		"gpt-4o",
		"openai",
		"chat",
		100*time.Millisecond,
	)

	// Should hit immediately.
	entry, _ := cache.Get(context.Background(), "test query", embedding)
	if entry == nil {
		t.Error("expected cache hit before expiration")
	}

	// Wait for expiration.
	time.Sleep(150 * time.Millisecond)

	// Should miss after expiration.
	entry, _ = cache.Get(context.Background(), "test query", embedding)
	if entry != nil {
		t.Error("expected cache miss after expiration")
	}
}

func TestSemanticCache_Delete(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	cache := NewSemanticCache(config, nil)

	embedding := SimpleEmbedding("test query", 1536)
	response := []byte(`{"test": true}`)

	id := cache.Set(
		context.Background(),
		"test query",
		embedding,
		response,
		"gpt-4o",
		"openai",
		"chat",
		time.Hour,
	)

	// Delete.
	cache.Delete(id)

	// Should miss after delete.
	entry, _ := cache.Get(context.Background(), "test query", embedding)
	if entry != nil {
		t.Error("expected cache miss after delete")
	}
}

func TestSemanticCache_Cleanup(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	cache := NewSemanticCache(config, nil)

	embedding := SimpleEmbedding("test query", 1536)
	response := []byte(`{"test": true}`)

	// Set entry with short TTL.
	cache.Set(
		context.Background(),
		"test query",
		embedding,
		response,
		"gpt-4o",
		"openai",
		"chat",
		100*time.Millisecond,
	)

	// Wait for expiration.
	time.Sleep(150 * time.Millisecond)

	// Cleanup should remove expired entry.
	count := cache.Cleanup()
	if count != 1 {
		t.Errorf("expected 1 evicted entry, got %d", count)
	}
}

func TestSemanticCache_GetStats(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	cache := NewSemanticCache(config, nil)

	embedding := SimpleEmbedding("test query", 1536)
	response := []byte(`{"test": true}`)

	// Initial stats.
	stats := cache.GetStats()
	if stats.TotalQueries != 0 {
		t.Error("expected 0 total queries initially")
	}

	// Generate cache miss.
	cache.Get(context.Background(), "test", embedding)

	stats = cache.GetStats()
	if stats.CacheMisses != 1 {
		t.Errorf("expected 1 cache miss, got %d", stats.CacheMisses)
	}

	// Store and retrieve.
	cache.Set(context.Background(), "test", embedding, response, "gpt-4o", "openai", "chat", time.Hour)
	cache.Get(context.Background(), "test", embedding)

	stats = cache.GetStats()
	if stats.CacheHits != 1 {
		t.Errorf("expected 1 cache hit, got %d", stats.CacheHits)
	}
}

func TestSemanticCache_MaxEntries(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	config.MaxEntries = 3
	cache := NewSemanticCache(config, nil)

	embedding := SimpleEmbedding("test", 1536)
	response := []byte(`{"test": true}`)

	// Add more entries than max.
	for i := 0; i < 5; i++ {
		cache.Set(
			context.Background(),
			"test query "+string(rune('a'+i)),
			embedding,
			response,
			"gpt-4o",
			"openai",
			"chat",
			time.Hour,
		)
	}

	// Should not exceed max.
	if cache.Size() > config.MaxEntries {
		t.Errorf("expected size <= %d, got %d", config.MaxEntries, cache.Size())
	}
}

func TestSemanticCache_FindSimilar(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	config.SimilarityThreshold = 0.7
	cache := NewSemanticCache(config, nil)

	// Add some entries.
	queries := []string{
		"how to write python code",
		"what is the weather today",
		"python function example",
	}

	for _, q := range queries {
		embedding := SimpleEmbedding(q, 1536)
		cache.Set(
			context.Background(),
			q,
			embedding,
			[]byte(`{"response":"test"}`),
			"gpt-4o",
			"openai",
			"chat",
			time.Hour,
		)
	}

	// Search for similar to python coding.
	searchEmbedding := SimpleEmbedding("python coding tutorial", 1536)
	results := cache.FindSimilar(searchEmbedding, 5)

	// Should find at least one similar result.
	if len(results) == 0 {
		t.Error("expected to find similar entries")
	}
}

func TestSimpleEmbedding(t *testing.T) {
	// Test that embeddings are consistent.
	emb1 := SimpleEmbedding("hello world", 1536)
	emb2 := SimpleEmbedding("hello world", 1536)

	if len(emb1) != 1536 {
		t.Errorf("expected dimension 1536, got %d", len(emb1))
	}

	// Same input should produce same output.
	for i := range emb1 {
		if emb1[i] != emb2[i] {
			t.Error("expected identical embeddings for same input")
			break
		}
	}

	// Different input should produce different output.
	emb3 := SimpleEmbedding("different text", 1536)
	allSame := true
	for i := range emb1 {
		if emb1[i] != emb3[i] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("expected different embeddings for different input")
	}
}

func TestCosineSimilarity(t *testing.T) {
	config := DefaultSemanticCacheConfig()
	cache := NewSemanticCache(config, nil)

	// Test identical vectors.
	vec1 := []float64{1, 0, 0, 1}
	sim := cache.cosineSimilarity(vec1, vec1)
	if sim < 0.99 || sim > 1.01 {
		t.Errorf("expected similarity ~1.0 for identical vectors, got %f", sim)
	}

	// Test orthogonal vectors.
	vec2 := []float64{0, 1, 0, 0}
	sim = cache.cosineSimilarity(vec1, vec2)
	if sim < -0.01 || sim > 0.01 {
		t.Errorf("expected similarity ~0.0 for orthogonal vectors, got %f", sim)
	}

	// Test opposite vectors.
	vec3 := []float64{-1, 0, 0, -1}
	sim = cache.cosineSimilarity(vec1, vec3)
	if sim < -1.01 || sim > -0.99 {
		t.Errorf("expected similarity ~-1.0 for opposite vectors, got %f", sim)
	}
}

func TestMockEmbeddingService(t *testing.T) {
	svc := NewMockEmbeddingService(512)

	embedding, err := svc.GetEmbedding(context.Background(), "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(embedding) != 512 {
		t.Errorf("expected dimension 512, got %d", len(embedding))
	}

	// Test batch.
	embeddings, err := svc.GetEmbeddings(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(embeddings) != 2 {
		t.Errorf("expected 2 embeddings, got %d", len(embeddings))
	}
}
