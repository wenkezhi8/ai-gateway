package cache

import (
	"context"
	"encoding/json"
	"time"
)

// VectorCacheDocument is the unified JSON document stored in Redis Stack.
type VectorCacheDocument struct {
	CacheKey        string            `json:"cache_key"`
	Intent          string            `json:"intent"`
	TaskType        string            `json:"task_type"`
	Slots           map[string]string `json:"slots"`
	NormalizedQuery string            `json:"normalized_query"`
	Vector          []float64         `json:"vector"`
	Response        any               `json:"response"`
	Provider        string            `json:"provider"`
	Model           string            `json:"model"`
	QualityScore    float64           `json:"quality_score"`
	CreateTS        int64             `json:"create_ts"`
	ExpireTS        int64             `json:"expire_ts"`
	TTLSec          int64             `json:"ttl_sec"`
}

// VectorSearchHit represents one vector recall result.
type VectorSearchHit struct {
	RedisKey   string          `json:"redis_key"`
	CacheKey   string          `json:"cache_key"`
	Intent     string          `json:"intent"`
	Score      float64         `json:"score"`      // cosine distance
	Similarity float64         `json:"similarity"` // 1 - distance
	Response   json.RawMessage `json:"response"`
}

// VectorStoreStats captures runtime vector store metadata.
type VectorStoreStats struct {
	Enabled      bool   `json:"enabled"`
	IndexName    string `json:"index_name"`
	KeyPrefix    string `json:"key_prefix"`
	Dimension    int    `json:"dimension"`
	QueryTimeout int64  `json:"query_timeout_ms"`
}

// VectorCacheStore abstracts vector cache operations.
type VectorCacheStore interface {
	EnsureIndex(ctx context.Context) error
	RebuildIndex(ctx context.Context) error
	GetExact(ctx context.Context, cacheKey string) (*VectorCacheDocument, error)
	VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]VectorSearchHit, error)
	Upsert(ctx context.Context, doc *VectorCacheDocument) error
	Delete(ctx context.Context, cacheKey string) error
	TouchTTL(ctx context.Context, cacheKey string, ttlSec int64) error
	Stats(ctx context.Context) (VectorStoreStats, error)
}

// RedisStackVectorConfig configures Redis Stack vector search behavior.
type RedisStackVectorConfig struct {
	Enabled      bool
	IndexName    string
	KeyPrefix    string
	Dimension    int
	QueryTimeout time.Duration
}

// DefaultRedisStackVectorConfig returns production-ready defaults.
func DefaultRedisStackVectorConfig() RedisStackVectorConfig {
	return RedisStackVectorConfig{
		Enabled:      true,
		IndexName:    "idx_ai_cache_v2",
		KeyPrefix:    "ai:v2:cache:",
		Dimension:    1024,
		QueryTimeout: 1500 * time.Millisecond,
	}
}

