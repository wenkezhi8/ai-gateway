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
	Enabled               bool `json:"enabled"`
	MaxPending            int  `json:"max_pending"`
	RequestTimeoutSeconds int  `json:"request_timeout_seconds"`
}

// CacheSettings represents high-level cache configuration.
type CacheSettings struct {
	Enabled                       bool               `json:"enabled"`
	Strategy                      CacheStrategy      `json:"strategy"`
	SimilarityThreshold           float64            `json:"similarity_threshold"` // 0-1
	DefaultTTLSeconds             int                `json:"default_ttl_seconds"`
	MaxEntries                    int                `json:"max_entries"`
	EvictionPolicy                string             `json:"eviction_policy"`
	VectorEnabled                 bool               `json:"vector_enabled"`
	VectorDimension               int                `json:"vector_dimension"`
	VectorQueryTimeoutMs          int                `json:"vector_query_timeout_ms"`
	VectorThresholds              map[string]float64 `json:"vector_thresholds"`
	VectorPipelineEnabled         bool               `json:"vector_pipeline_enabled"`
	VectorStandardKeyVersion      string             `json:"vector_standard_key_version"`
	VectorEmbeddingProvider       string             `json:"vector_embedding_provider"`
	VectorOllamaBaseURL           string             `json:"vector_ollama_base_url"`
	VectorOllamaEmbeddingModel    string             `json:"vector_ollama_embedding_model"`
	VectorOllamaEmbeddingDimension int               `json:"vector_ollama_embedding_dimension"`
	VectorOllamaEmbeddingTimeoutMs int               `json:"vector_ollama_embedding_timeout_ms"`
	VectorOllamaEndpointMode      string             `json:"vector_ollama_endpoint_mode"`
	VectorWritebackEnabled        bool               `json:"vector_writeback_enabled"`
	ColdVectorEnabled             bool               `json:"cold_vector_enabled"`
	ColdVectorQueryEnabled        bool               `json:"cold_vector_query_enabled"`
	ColdVectorBackend             string             `json:"cold_vector_backend"`
	ColdVectorDualWriteEnabled    bool               `json:"cold_vector_dual_write_enabled"`
	ColdVectorSimilarityThreshold float64            `json:"cold_vector_similarity_threshold"`
	ColdVectorTopK                int                `json:"cold_vector_top_k"`
	HotMemoryHighWatermarkPercent float64            `json:"hot_memory_high_watermark_percent"`
	HotMemoryReliefPercent        float64            `json:"hot_memory_relief_percent"`
	HotToColdBatchSize            int                `json:"hot_to_cold_batch_size"`
	HotToColdIntervalSeconds      int                `json:"hot_to_cold_interval_seconds"`
	ColdVectorQdrantURL           string             `json:"cold_vector_qdrant_url"`
	ColdVectorQdrantAPIKey        string             `json:"cold_vector_qdrant_api_key"`
	ColdVectorQdrantCollection    string             `json:"cold_vector_qdrant_collection"`
	ColdVectorQdrantTimeoutMs     int                `json:"cold_vector_qdrant_timeout_ms"`
	Dedup                         DedupSettings      `json:"dedup"`
}

// DefaultCacheSettings returns recommended default settings.
func DefaultCacheSettings() CacheSettings {
	return CacheSettings{
		Enabled:              true,
		Strategy:             CacheStrategySemantic,
		SimilarityThreshold:  0.92,
		DefaultTTLSeconds:    int((30 * time.Minute).Seconds()),
		MaxEntries:           10000,
		EvictionPolicy:       "lru",
		VectorEnabled:        false,
		VectorDimension:      1024,
		VectorQueryTimeoutMs: 1200,
		VectorThresholds: map[string]float64{
			"calc":      0.97,
			"translate": 0.96,
			"weather":   0.95,
			"qa":        0.93,
			"chat":      0.92,
		},
		VectorPipelineEnabled:          true,
		VectorStandardKeyVersion:       "v2",
		VectorEmbeddingProvider:        "ollama",
		VectorOllamaBaseURL:            "http://127.0.0.1:11434",
		VectorOllamaEmbeddingModel:     "nomic-embed-text",
		VectorOllamaEmbeddingDimension: 1024,
		VectorOllamaEmbeddingTimeoutMs: 1500,
		VectorOllamaEndpointMode:       "auto",
		VectorWritebackEnabled:         true,
		ColdVectorEnabled:             false,
		ColdVectorQueryEnabled:        true,
		ColdVectorBackend:             ColdVectorBackendSQLite,
		ColdVectorDualWriteEnabled:    false,
		ColdVectorSimilarityThreshold: 0.92,
		ColdVectorTopK:                1,
		HotMemoryHighWatermarkPercent: 75,
		HotMemoryReliefPercent:        65,
		HotToColdBatchSize:            500,
		HotToColdIntervalSeconds:      30,
		ColdVectorQdrantCollection:    "ai_gateway_cold_vectors",
		ColdVectorQdrantTimeoutMs:     1500,
		Dedup: DedupSettings{
			Enabled:               true,
			MaxPending:            1000,
			RequestTimeoutSeconds: 30,
		},
	}
}
