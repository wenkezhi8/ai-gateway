package cache

import "time"

// CacheStrategy defines cache matching strategy.
type CacheStrategy string

const (
	CacheStrategySemantic CacheStrategy = "semantic"
	CacheStrategyExact    CacheStrategy = "exact"
	CacheStrategyPrefix   CacheStrategy = "prefix"
)

// DedupSettings represents request deduplication configuration.
type DedupSettings struct {
	Enabled              bool `json:"enabled"`
	MaxPending           int  `json:"max_pending"`
	RequestTimeoutSeconds int `json:"request_timeout_seconds"`
}

// CacheSettings represents high-level cache configuration.
type CacheSettings struct {
	Enabled              bool          `json:"enabled"`
	Strategy             CacheStrategy `json:"strategy"`
	SimilarityThreshold  float64       `json:"similarity_threshold"` // 0-1
	DefaultTTLSeconds    int           `json:"default_ttl_seconds"`
	MaxEntries           int           `json:"max_entries"`
	EvictionPolicy       string        `json:"eviction_policy"`
	VectorEnabled        bool          `json:"vector_enabled"`
	VectorDimension      int           `json:"vector_dimension"`
	VectorQueryTimeoutMs int           `json:"vector_query_timeout_ms"`
	VectorThresholds     map[string]float64 `json:"vector_thresholds"`
	Dedup                DedupSettings `json:"dedup"`
}

// DefaultCacheSettings returns recommended default settings.
func DefaultCacheSettings() CacheSettings {
	return CacheSettings{
		Enabled:             true,
		Strategy:            CacheStrategySemantic,
		SimilarityThreshold: 0.92,
		DefaultTTLSeconds:   int((30 * time.Minute).Seconds()),
		MaxEntries:          10000,
		EvictionPolicy:      "lru",
		VectorEnabled:       false,
		VectorDimension:     1024,
		VectorQueryTimeoutMs: 1200,
		VectorThresholds: map[string]float64{
			"calc":      0.97,
			"translate": 0.96,
			"weather":   0.95,
			"qa":        0.93,
			"chat":      0.92,
		},
		Dedup: DedupSettings{
			Enabled:              true,
			MaxPending:           1000,
			RequestTimeoutSeconds: 30,
		},
	}
}
