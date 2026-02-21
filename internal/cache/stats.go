package cache

import (
	"context"
	"sync/atomic"
	"time"
)

// Stats tracks cache performance metrics
type Stats struct {
	hits            int64
	misses          int64
	tokensSaved     int64
	requestsSaved   int64
	evictions       int64
	errors          int64
	totalLatency    int64 // in nanoseconds
	totalOperations int64
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{}
}

// RecordHit records a cache hit
func (s *Stats) RecordHit(latency time.Duration) {
	atomic.AddInt64(&s.hits, 1)
	atomic.AddInt64(&s.totalLatency, int64(latency))
	atomic.AddInt64(&s.totalOperations, 1)
}

// RecordMiss records a cache miss
func (s *Stats) RecordMiss(latency time.Duration) {
	atomic.AddInt64(&s.misses, 1)
	atomic.AddInt64(&s.totalLatency, int64(latency))
	atomic.AddInt64(&s.totalOperations, 1)
}

// RecordTokensSaved records tokens saved from cache
func (s *Stats) RecordTokensSaved(tokens int64) {
	atomic.AddInt64(&s.tokensSaved, tokens)
}

// RecordRequestSaved records a request saved from cache
func (s *Stats) RecordRequestSaved() {
	atomic.AddInt64(&s.requestsSaved, 1)
}

// RecordEviction records a cache eviction
func (s *Stats) RecordEviction() {
	atomic.AddInt64(&s.evictions, 1)
}

// RecordError records a cache error
func (s *Stats) RecordError() {
	atomic.AddInt64(&s.errors, 1)
}

// StatsSnapshot represents a point-in-time snapshot of cache statistics
type StatsSnapshot struct {
	Hits            int64   `json:"hits"`
	Misses          int64   `json:"misses"`
	HitRate         float64 `json:"hit_rate"`
	TokensSaved     int64   `json:"tokens_saved"`
	RequestsSaved   int64   `json:"requests_saved"`
	Evictions       int64   `json:"evictions"`
	Errors          int64   `json:"errors"`
	AvgLatencyNs    int64   `json:"avg_latency_ns"`
	TotalOperations int64   `json:"total_operations"`
	Timestamp       int64   `json:"timestamp"`
}

// Snapshot returns a snapshot of current statistics
func (s *Stats) Snapshot() StatsSnapshot {
	hits := atomic.LoadInt64(&s.hits)
	misses := atomic.LoadInt64(&s.misses)
	totalOps := atomic.LoadInt64(&s.totalOperations)

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	var avgLatency int64
	if totalOps > 0 {
		avgLatency = atomic.LoadInt64(&s.totalLatency) / totalOps
	}

	return StatsSnapshot{
		Hits:            hits,
		Misses:          misses,
		HitRate:         hitRate,
		TokensSaved:     atomic.LoadInt64(&s.tokensSaved),
		RequestsSaved:   atomic.LoadInt64(&s.requestsSaved),
		Evictions:       atomic.LoadInt64(&s.evictions),
		Errors:          atomic.LoadInt64(&s.errors),
		AvgLatencyNs:    avgLatency,
		TotalOperations: totalOps,
		Timestamp:       time.Now().Unix(),
	}
}

// Reset resets all statistics to zero
func (s *Stats) Reset() {
	atomic.StoreInt64(&s.hits, 0)
	atomic.StoreInt64(&s.misses, 0)
	atomic.StoreInt64(&s.tokensSaved, 0)
	atomic.StoreInt64(&s.requestsSaved, 0)
	atomic.StoreInt64(&s.evictions, 0)
	atomic.StoreInt64(&s.errors, 0)
	atomic.StoreInt64(&s.totalLatency, 0)
	atomic.StoreInt64(&s.totalOperations, 0)
}

// StatsCollector collects and aggregates cache statistics
type StatsCollector struct {
	stats map[string]*Stats
}

// NewStatsCollector creates a new stats collector
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		stats: make(map[string]*Stats),
	}
}

// GetStats returns stats for a named cache, creating if not exists
func (c *StatsCollector) GetStats(name string) *Stats {
	stats, ok := c.stats[name]
	if !ok {
		stats = NewStats()
		c.stats[name] = stats
	}
	return stats
}

// AllStats returns all cache statistics
func (c *StatsCollector) AllStats() map[string]StatsSnapshot {
	result := make(map[string]StatsSnapshot)
	for name, stats := range c.stats {
		result[name] = stats.Snapshot()
	}
	return result
}

// GlobalStatsCollector is the default stats collector
var GlobalStatsCollector = NewStatsCollector()

// GetCacheStats returns stats for a named cache from the global collector
func GetCacheStats(name string) *Stats {
	return GlobalStatsCollector.GetStats(name)
}

// GetAllCacheStats returns all cache statistics from the global collector
func GetAllCacheStats() map[string]StatsSnapshot {
	return GlobalStatsCollector.AllStats()
}

// TrackedCache wraps a Cache with statistics tracking
type TrackedCache struct {
	Cache
	stats *Stats
	name  string
}

// NewTrackedCache creates a cache wrapper that tracks statistics
func NewTrackedCache(cache Cache, name string) *TrackedCache {
	return &TrackedCache{
		Cache: cache,
		stats: GlobalStatsCollector.GetStats(name),
		name:  name,
	}
}

// Get retrieves a value and records statistics
func (c *TrackedCache) Get(ctx context.Context, key string, dest interface{}) error {
	start := time.Now()
	err := c.Cache.Get(ctx, key, dest)
	latency := time.Since(start)

	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
		} else {
			c.stats.RecordError()
		}
		return err
	}

	c.stats.RecordHit(latency)
	return nil
}

// Set stores a value and records statistics
func (c *TrackedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.Cache.Set(ctx, key, value, ttl)
}

// Delete removes a value
func (c *TrackedCache) Delete(ctx context.Context, key string) error {
	return c.Cache.Delete(ctx, key)
}

// GetStats returns the statistics for this cache
func (c *TrackedCache) GetStats() *Stats {
	return c.stats
}
