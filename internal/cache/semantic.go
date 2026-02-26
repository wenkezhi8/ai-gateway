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

	"ai-gateway/internal/metrics"

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
	ID           string                 `json:"id"`
	Query        string                 `json:"query"`
	QueryVector  []float64              `json:"query_vector"`
	Response     json.RawMessage        `json:"response"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	TaskType     string                 `json:"task_type"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	HitCount     int                    `json:"hit_count"`
	Metadata     map[string]interface{} `json:"metadata"`
	QualityScore float64                `json:"quality_score"` // 质量评分 0-100
	Validated    bool                   `json:"validated"`     // 是否已通过质量校验
}

// QualityChecker defines the interface for cache quality validation
type QualityChecker interface {
	Validate(entry *SemanticEntry) (bool, float64, string)
	// 返回: 是否通过, 质量评分, 原因
}

// DefaultQualityChecker provides basic quality validation
type DefaultQualityChecker struct {
	minResponseLength int
	maxResponseLength int
}

// NewDefaultQualityChecker creates a new quality checker
func NewDefaultQualityChecker() *DefaultQualityChecker {
	return &DefaultQualityChecker{
		minResponseLength: 10,
		maxResponseLength: 100000,
	}
}

// Validate checks if a cache entry meets quality standards
func (qc *DefaultQualityChecker) Validate(entry *SemanticEntry) (bool, float64, string) {
	// 1. 响应完整性检查
	if len(entry.Response) == 0 {
		return false, 0, "empty response"
	}

	// 2. 响应长度检查
	respLen := len(entry.Response)
	if respLen < qc.minResponseLength {
		return false, 30, "response too short"
	}
	if respLen > qc.maxResponseLength {
		return false, 50, "response too long"
	}

	// 3. JSON 有效性检查
	var jsonObj interface{}
	if err := json.Unmarshal(entry.Response, &jsonObj); err != nil {
		return false, 40, "invalid JSON response"
	}

	// 4. 基础质量评分
	baseScore := 70.0

	// 响应长度加分（适中长度更好）
	if respLen > 100 && respLen < 10000 {
		baseScore += 10
	}

	// 模型加分（已知高质量模型）
	highQualityModels := map[string]bool{
		"gpt-4": true, "gpt-4o": true, "gpt-4-turbo": true,
		"claude-3-5-sonnet": true, "claude-3-opus": true,
		"deepseek-chat": true, "deepseek-coder": true,
	}
	if highQualityModels[entry.Model] {
		baseScore += 10
	}

	// 任务类型加分
	taskTypeBonus := map[string]float64{
		"math": 10, // 数学结果确定性高
		"fact": 5,  // 事实查询
		"code": 5,  // 代码生成
	}
	if bonus, ok := taskTypeBonus[entry.TaskType]; ok {
		baseScore += bonus
	}

	// 确保分数在 0-100 之间
	if baseScore > 100 {
		baseScore = 100
	}

	return true, baseScore, "passed validation"
}

// FactFreshnessChecker checks if fact-based entries are still fresh
type FactFreshnessChecker struct {
	maxAge time.Duration
}

// NewFactFreshnessChecker creates a checker for fact-based content
func NewFactFreshnessChecker() *FactFreshnessChecker {
	return &FactFreshnessChecker{
		maxAge: 7 * 24 * time.Hour, // 7 天
	}
}

// Validate checks fact freshness
func (fc *FactFreshnessChecker) Validate(entry *SemanticEntry) (bool, float64, string) {
	if entry.TaskType != "fact" {
		return true, 100, "not a fact query"
	}

	age := time.Since(entry.CreatedAt)
	if age > fc.maxAge {
		return false, 50, "fact entry expired"
	}

	// 根据新鲜度计算评分
	freshnessRatio := 1 - (float64(age) / float64(fc.maxAge))
	score := 60 + freshnessRatio*40

	return true, score, "fact entry is fresh"
}

// SemanticCache provides semantic caching with vector similarity
type SemanticCache struct {
	mu              sync.RWMutex
	config          SemanticCacheConfig
	entries         map[string]*SemanticEntry // id -> entry
	index           []*SemanticEntry          // for similarity search
	stats           SemanticCacheStats
	backend         Cache          // 可选：Redis 缓存后端
	qualityChecker  QualityChecker // 质量校验器
	minQualityScore float64        // 最低质量分阈值
	persistCh       chan semanticPersistJob
}

// SemanticCacheStats tracks semantic cache statistics (internal, contains mutex)
type SemanticCacheStats struct {
	mu            sync.RWMutex
	TotalQueries  int64   `json:"total_queries"`
	CacheHits     int64   `json:"cache_hits"`
	CacheMisses   int64   `json:"cache_misses"`
	Evictions     int64   `json:"evictions"`
	AvgSimilarity float64 `json:"avg_similarity"`
}

// SemanticCacheStatsData represents cache statistics for external use (no mutex)
type SemanticCacheStatsData struct {
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
		config:          config,
		entries:         make(map[string]*SemanticEntry),
		index:           make([]*SemanticEntry, 0),
		backend:         backend,
		qualityChecker:  NewDefaultQualityChecker(),
		minQualityScore: 60.0, // 默认最低质量分
		persistCh:       make(chan semanticPersistJob, 512),
	}

	// 如果有后端，从后端加载缓存
	if backend != nil {
		cache.loadFromBackend()
		go cache.persistLoop()
	}

	return cache
}

// UpdateConfig updates semantic cache configuration at runtime.
func (c *SemanticCache) UpdateConfig(config SemanticCacheConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = config

	// If max entries reduced, evict until within limit.
	for c.config.MaxEntries > 0 && len(c.entries) > c.config.MaxEntries {
		c.evictOldest()
	}
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
// 改动点: 添加质量校验，只有通过校验的响应才会被缓存
func (c *SemanticCache) Set(ctx context.Context, query string, queryVector []float64, response json.RawMessage, model, provider, taskType string, ttl time.Duration) string {
	if !c.config.Enabled {
		return ""
	}

	id := generateSemanticID(query, model)
	now := time.Now()
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	entry := &SemanticEntry{
		ID:           id,
		Query:        query,
		QueryVector:  queryVector,
		Response:     response,
		Model:        model,
		Provider:     provider,
		TaskType:     taskType,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
		HitCount:     0,
		Metadata:     make(map[string]interface{}),
		QualityScore: 0,
		Validated:    false,
	}

	// 质量校验
	if c.qualityChecker != nil {
		passed, score, reason := c.qualityChecker.Validate(entry)
		entry.QualityScore = score
		entry.Validated = passed

		if !passed {
			semanticLogger.WithFields(logrus.Fields{
				"query_id": id,
				"model":    model,
				"score":    score,
				"reason":   reason,
			}).Warn("Cache entry rejected by quality checker")
			return "" // 不缓存低质量响应
		}

		// 低于最低质量分阈值也不缓存
		if score < c.minQualityScore {
			semanticLogger.WithFields(logrus.Fields{
				"query_id":          id,
				"model":             model,
				"score":             score,
				"min_quality_score": c.minQualityScore,
			}).Debug("Cache entry quality score below threshold")
			return ""
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict
	if len(c.entries) >= c.config.MaxEntries {
		c.evictOldest()
	}

	c.entries[id] = entry
	c.index = append(c.index, entry)

	// 如果有后端，存储到后端
	if c.backend != nil {
		data, err := json.Marshal(entry)
		if err == nil {
			select {
			case c.persistCh <- semanticPersistJob{key: "semantic:" + id, data: data, ttl: ttl}:
				if m := metrics.GetMetrics(); m != nil {
					m.RecordSemanticPersistEnqueued("semantic")
					m.SetSemanticPersistQueueDepth("semantic", len(c.persistCh))
				}
			default:
				if m := metrics.GetMetrics(); m != nil {
					m.RecordSemanticPersistDropped("semantic")
					m.SetSemanticPersistQueueDepth("semantic", len(c.persistCh))
				}
				semanticLogger.WithField("query_id", id).Warn("Semantic cache persist queue full, dropping entry")
			}
		}
	}

	semanticLogger.WithFields(logrus.Fields{
		"query_id":      id,
		"model":         model,
		"task_type":     taskType,
		"ttl":           ttl,
		"quality_score": entry.QualityScore,
		"backend":       c.backend != nil,
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

// SetQualityChecker sets the quality checker
func (c *SemanticCache) SetQualityChecker(checker QualityChecker) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.qualityChecker = checker
}

// SetMinQualityScore sets the minimum quality score threshold
func (c *SemanticCache) SetMinQualityScore(score float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.minQualityScore = score
}

// GetQualityConfig returns current quality configuration
func (c *SemanticCache) GetQualityConfig() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]interface{}{
		"min_quality_score": c.minQualityScore,
		"checker_type":      "default",
	}
}

// InvalidateLowQuality removes entries below the quality threshold
func (c *SemanticCache) InvalidateLowQuality() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for id, entry := range c.entries {
		if entry.QualityScore < c.minQualityScore {
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
		semanticLogger.WithField("count", count).Info("Invalidated low quality cache entries")
	}

	return count
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

// GetStats returns cache statistics (copy without mutex)
func (c *SemanticCache) GetStats() SemanticCacheStatsData {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return SemanticCacheStatsData{
		TotalQueries:  c.stats.TotalQueries,
		CacheHits:     c.stats.CacheHits,
		CacheMisses:   c.stats.CacheMisses,
		Evictions:     c.stats.Evictions,
		AvgSimilarity: c.stats.AvgSimilarity,
	}
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

type semanticPersistJob struct {
	key  string
	data []byte
	ttl  time.Duration
}

func (c *SemanticCache) persistLoop() {
	for job := range c.persistCh {
		if m := metrics.GetMetrics(); m != nil {
			m.SetSemanticPersistQueueDepth("semantic", len(c.persistCh))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = c.backend.Set(ctx, job.key, job.data, job.ttl)
		cancel()
	}
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
