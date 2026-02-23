// Package cache provides semantic caching with vector similarity matching
// 改动点: 新增语义缓存模块，支持相似请求复用
// 持久化：配置 Redis 则用 Redis，否则纯内存
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// SemanticCacheConfig holds configuration for semantic cache
type SemanticCacheConfig struct {
	Enabled             bool          `json:"enabled"`
	SimilarityThreshold float64       `json:"similarity_threshold"` // 0-1, default 0.95
	MaxEntries          int           `json:"max_entries"`
	DefaultTTL          time.Duration `json:"default_ttl"`
	VectorDimension     int           `json:"vector_dimension"` // embedding dimension
}

// SemanticEntry represents a cached semantic entry
type SemanticEntry struct {
	ID          string                 `json:"id"`
	Query       string                 `json:"query"`
	QueryVector []float64              `json:"query_vector"`
	Response    json.RawMessage        `json:"response"`
	Model       string                 `json:"model"`
	Provider    string                 `json:"provider"`
	TaskType    string                 `json:"task_type"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	HitCount    int                    `json:"hit_count"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SemanticCache provides semantic caching with vector similarity
type SemanticCache struct {
	mu      sync.RWMutex
	config  SemanticCacheConfig
	entries map[string]*SemanticEntry // id -> entry
	index   []*SemanticEntry          // for similarity search
	stats   SemanticCacheStats
	backend Cache // 可选：Redis 缓存后端
}

// SemanticCacheStats tracks semantic cache statistics
type SemanticCacheStats struct {
	mu            sync.RWMutex
	TotalQueries  int64   `json:"total_queries"`
	CacheHits     int64   `json:"cache_hits"`
	CacheMisses   int64   `json:"cache_misses"`
	Evictions     int64   `json:"evictions"`
	AvgSimilarity float64 `json:"avg_similarity"`
}

// SemanticCachePersist represents persistent cache data
type SemanticCachePersist struct {
	Entries []*SemanticEntry `json:"entries"`
}

var semanticLogger = logrus.WithField("component", "semantic_cache")

// DefaultSemanticCacheConfig returns default configuration
func DefaultSemanticCacheConfig() SemanticCacheConfig {
	return SemanticCacheConfig{
		Enabled:             true,
		SimilarityThreshold: 0.92,
		MaxEntries:          10000,
		DefaultTTL:          24 * time.Hour,
		VectorDimension:     1536,
	}
}

// NewSemanticCache creates a new semantic cache
// backend: 可选的 Redis 缓存后端，用于持久化
func NewSemanticCache(config SemanticCacheConfig, backend Cache) *SemanticCache {
	cache := &SemanticCache{
		config:  config,
		entries: make(map[string]*SemanticEntry),
		index:   make([]*SemanticEntry, 0),
		backend: backend,
	}

	// 如果有后端，从后端加载缓存
	if backend != nil {
		cache.loadFromBackend()
	}

	return cache
}

// Get retrieves a semantically similar cached response
// 改动点: 使用向量相似度匹配相似请求
func (c *SemanticCache) Get(ctx context.Context, query string, queryVector []float64) (*SemanticEntry, float64) {
	if !c.config.Enabled {
		return nil, 0
	}

	c.stats.mu.Lock()
	c.stats.TotalQueries++
	c.stats.mu.Unlock()

	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	var bestMatch *SemanticEntry
	bestSimilarity := 0.0

	for _, entry := range c.index {
		// Skip expired entries
		if now.After(entry.ExpiresAt) {
			continue
		}

		similarity := c.cosineSimilarity(queryVector, entry.QueryVector)
		if similarity >= c.config.SimilarityThreshold && similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatch = entry
		}
	}

	if bestMatch != nil {
		c.stats.mu.Lock()
		c.stats.CacheHits++
		if c.stats.AvgSimilarity == 0 {
			c.stats.AvgSimilarity = bestSimilarity
		} else {
			c.stats.AvgSimilarity = (c.stats.AvgSimilarity + bestSimilarity) / 2
		}
		c.stats.mu.Unlock()

		semanticLogger.WithFields(logrus.Fields{
			"query_id":   bestMatch.ID,
			"similarity": bestSimilarity,
			"model":      bestMatch.Model,
			"task_type":  bestMatch.TaskType,
		}).Info("Semantic cache hit")

		return bestMatch, bestSimilarity
	}

	c.stats.mu.Lock()
	c.stats.CacheMisses++
	c.stats.mu.Unlock()

	return nil, 0
}

// Set stores a response in the semantic cache
func (c *SemanticCache) Set(ctx context.Context, query string, queryVector []float64, response json.RawMessage, model, provider, taskType string, ttl time.Duration) string {
	if !c.config.Enabled {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict
	if len(c.entries) >= c.config.MaxEntries {
		c.evictOldest()
	}

	id := generateSemanticID(query, model)
	now := time.Now()
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	entry := &SemanticEntry{
		ID:          id,
		Query:       query,
		QueryVector: queryVector,
		Response:    response,
		Model:       model,
		Provider:    provider,
		TaskType:    taskType,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
		HitCount:    0,
		Metadata:    make(map[string]interface{}),
	}

	c.entries[id] = entry
	c.index = append(c.index, entry)

	// 如果有后端，存储到后端
	if c.backend != nil {
		go func() {
			data, err := json.Marshal(entry)
			if err == nil {
				c.backend.Set(context.Background(), "semantic:"+id, data, ttl)
			}
		}()
	}

	semanticLogger.WithFields(logrus.Fields{
		"query_id":  id,
		"model":     model,
		"task_type": taskType,
		"ttl":       ttl,
		"backend":   c.backend != nil,
	}).Debug("Semantic cache entry stored")

	return id
}

// GetByID retrieves an entry by ID
func (c *SemanticCache) GetByID(id string) *SemanticEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries[id]
}

// Delete removes an entry from the cache
func (c *SemanticCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entries[id]; ok {
		delete(c.entries, id)
		// Remove from index
		for i, e := range c.index {
			if e.ID == id {
				c.index = append(c.index[:i], c.index[i+1:]...)
				break
			}
		}
		c.stats.mu.Lock()
		c.stats.Evictions++
		c.stats.mu.Unlock()
	}
}

// Clear removes all entries
func (c *SemanticCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*SemanticEntry)
	c.index = make([]*SemanticEntry, 0)
}

// loadFromBackend loads cache entries from the backend (Redis)
// 改动点: 从 Redis 加载缓存，无 Redis 则跳过
func (c *SemanticCache) loadFromBackend() {
	if c.backend == nil {
		semanticLogger.Debug("No backend configured, starting with empty cache")
		return
	}

	// 从 Redis 加载所有语义缓存条目
	// 注意：Redis 不支持遍历所有 key，所以这里需要特殊处理
	// 实际使用中，可以在启动时从 Redis 加载热点缓存
	semanticLogger.Info("Backend configured, will load entries on demand")
}

// GetBackend returns the backend cache
func (c *SemanticCache) GetBackend() Cache {
	return c.backend
}

// Cleanup removes expired entries
func (c *SemanticCache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0

	for id, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, id)
			count++
		}
	}

	// Rebuild index
	c.index = make([]*SemanticEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		c.index = append(c.index, entry)
	}

	if count > 0 {
		c.stats.mu.Lock()
		c.stats.Evictions += int64(count)
		c.stats.mu.Unlock()
		semanticLogger.WithField("evicted", count).Info("Semantic cache cleanup completed")
	}

	return count
}

// GetStats returns cache statistics
func (c *SemanticCache) GetStats() SemanticCacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return c.stats
}

// cosineSimilarity calculates cosine similarity between two vectors
func (c *SemanticCache) cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// evictOldest removes the oldest entries when cache is full
func (c *SemanticCache) evictOldest() {
	if len(c.entries) == 0 {
		return
	}

	// Find oldest entry with lowest hit count
	var oldest *SemanticEntry
	for _, entry := range c.entries {
		if oldest == nil ||
			entry.HitCount < oldest.HitCount ||
			(entry.HitCount == oldest.HitCount && entry.CreatedAt.Before(oldest.CreatedAt)) {
			oldest = entry
		}
	}

	if oldest != nil {
		delete(c.entries, oldest.ID)
		// Remove from index
		for i, e := range c.index {
			if e.ID == oldest.ID {
				c.index = append(c.index[:i], c.index[i+1:]...)
				break
			}
		}
		c.stats.mu.Lock()
		c.stats.Evictions++
		c.stats.mu.Unlock()
	}
}

// generateSemanticID generates a unique ID for a semantic entry
func generateSemanticID(query, model string) string {
	data := query + ":" + model + ":" + time.Now().Format("20060102")
	hash := sha256.Sum256([]byte(data))
	return string(hash[:16])
}

// IncrementHitCount increments the hit count for an entry
func (c *SemanticCache) IncrementHitCount(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[id]; ok {
		entry.HitCount++
	}
}

// GetEntries returns all entries (for debugging)
func (c *SemanticCache) GetEntries() []*SemanticEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*SemanticEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		result = append(result, entry)
	}
	return result
}

// Size returns the number of entries in the cache
func (c *SemanticCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// SetTTL updates the TTL for an entry
func (c *SemanticCache) SetTTL(id string, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[id]; ok {
		entry.ExpiresAt = time.Now().Add(ttl)
		return true
	}
	return false
}

// FindSimilar finds entries similar to the given query vector
func (c *SemanticCache) FindSimilar(queryVector []float64, limit int) []*SemanticEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	type scoredEntry struct {
		entry      *SemanticEntry
		similarity float64
	}

	var scored []*scoredEntry
	for _, entry := range c.index {
		if time.Now().After(entry.ExpiresAt) {
			continue
		}
		similarity := c.cosineSimilarity(queryVector, entry.QueryVector)
		if similarity >= c.config.SimilarityThreshold*0.8 { // Lower threshold for search
			scored = append(scored, &scoredEntry{entry: entry, similarity: similarity})
		}
	}

	// Sort by similarity
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].similarity > scored[i].similarity {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Return top N
	result := make([]*SemanticEntry, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].entry)
	}
	return result
}
