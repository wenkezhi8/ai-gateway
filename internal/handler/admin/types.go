package admin

import "time"

// Account management types

// AccountRequest represents a request to create/update an account.
type AccountRequest struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Provider          string                 `json:"provider"`
	APIKey            string                 `json:"api_key"`
	BaseURL           string                 `json:"base_url"`
	Enabled           *bool                  `json:"enabled,omitempty"`
	Priority          int                    `json:"priority"`
	Limits            map[string]LimitConfig `json:"limits"`
	CodingPlanEnabled *bool                  `json:"coding_plan_enabled,omitempty"`
}

// LimitConfig represents a limit configuration.
type LimitConfig struct {
	Type    string  `json:"type"`    // token, rpm, concurrent, request
	Period  string  `json:"period"`  // minute, hour, 5hour, day, week, month
	Limit   int64   `json:"limit"`   // max value
	Warning float64 `json:"warning"` // warning threshold
}

// LimitConfigMap is a map of limit type to config (for JSON serialization).
type LimitConfigMap map[string]LimitConfig

// AccountResponse represents an account in responses.
type AccountResponse struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Provider          string                 `json:"provider"`
	APIKey            string                 `json:"api_key"`
	BaseURL           string                 `json:"base_url"`
	Enabled           bool                   `json:"enabled"`
	Priority          int                    `json:"priority"`
	IsActive          bool                   `json:"is_active"`
	Limits            map[string]LimitConfig `json:"limits"`
	Usage             *AccountUsageResponse  `json:"usage,omitempty"`
	LastSwitch        time.Time              `json:"last_switch,omitempty"`
	PlanType          string                 `json:"plan_type,omitempty"`
	CodingPlanEnabled bool                   `json:"coding_plan_enabled,omitempty"`
}

// AccountUsageResponse represents account usage data.
type AccountUsageResponse struct {
	TokensUsed    int64   `json:"tokens_used"`
	TokenLimit    int64   `json:"token_limit"`
	TokenPercent  float64 `json:"token_percent"`
	RequestsCount int64   `json:"requests_count"`
	RPM           int     `json:"rpm"`
	RPMLimit      int     `json:"rpm_limit"`
	WarningLevel  string  `json:"warning_level"`
	// Multi-level usage for Coding Plan
	Hour5Used    int64   `json:"hour5_used,omitempty"`
	Hour5Limit   int64   `json:"hour5_limit,omitempty"`
	Hour5Percent float64 `json:"hour5_percent,omitempty"`
	WeekUsed     int64   `json:"week_used,omitempty"`
	WeekLimit    int64   `json:"week_limit,omitempty"`
	WeekPercent  float64 `json:"week_percent,omitempty"`
	MonthUsed    int64   `json:"month_used,omitempty"`
	MonthLimit   int64   `json:"month_limit,omitempty"`
	MonthPercent float64 `json:"month_percent,omitempty"`
}

// Provider management types

// ProviderRequest represents a request to add/update a provider.
type ProviderRequest struct {
	Name    string                 `json:"name"`
	APIKey  string                 `json:"api_key"`
	BaseURL string                 `json:"base_url"`
	Models  []string               `json:"models"`
	Enabled bool                   `json:"enabled"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

// ProviderResponse represents a provider in responses.
type ProviderResponse struct {
	Name         string    `json:"name"`
	BaseURL      string    `json:"base_url"`
	Models       []string  `json:"models"`
	Enabled      bool      `json:"enabled"`
	Healthy      bool      `json:"healthy"`
	AccountCount int       `json:"account_count"`
	LastCheck    time.Time `json:"last_check,omitempty"`
}

// ProviderTypeResponse represents provider metadata for account forms.
type ProviderTypeResponse struct {
	ID                 string   `json:"id"`
	Label              string   `json:"label"`
	Category           string   `json:"category"`
	Color              string   `json:"color"`
	Logo               string   `json:"logo"`
	DefaultEndpoint    string   `json:"default_endpoint"`
	CodingEndpoint     string   `json:"coding_endpoint"`
	SupportsCodingPlan bool     `json:"supports_coding_plan"`
	Models             []string `json:"models"`
}

// ProviderTestResult represents provider connectivity test result.
type ProviderTestResult struct {
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	ResponseTime int64     `json:"response_time_ms"`
	Timestamp    time.Time `json:"timestamp"`
}

// Routing types

// RoutingConfig represents routing strategy configuration.
type RoutingConfig struct {
	DefaultStrategy string            `json:"default_strategy"`
	ModelStrategies map[string]string `json:"model_strategies,omitempty"`
	ProviderWeights map[string]int    `json:"provider_weights,omitempty"`
	FailoverConfig  *FailoverConfig   `json:"failover_config,omitempty"`
}

// FailoverConfig represents failover settings.
type FailoverConfig struct {
	MaxRetries       int  `json:"max_retries"`
	RetryDelayMs     int  `json:"retry_delay_ms"`
	HealthCheckSec   int  `json:"health_check_sec"`
	CircuitBreaker   bool `json:"circuit_breaker"`
	FailureThreshold int  `json:"failure_threshold"`
}

// Cache types

// CacheStatsResponse represents cache statistics.
type CacheStatsResponse struct {
	RequestCache  CacheStatDetail `json:"request_cache"`
	ContextCache  CacheStatDetail `json:"context_cache"`
	RouteCache    CacheStatDetail `json:"route_cache"`
	UsageCache    CacheStatDetail `json:"usage_cache"`
	ResponseCache CacheStatDetail `json:"response_cache"`
	TokenSavings  int64           `json:"token_savings"`
	RedisHits     int64           `json:"redis_hits,omitempty"`
	RedisMisses   int64           `json:"redis_misses,omitempty"`
	RedisHitRate  float64         `json:"redis_hit_rate,omitempty"`
}

// CacheStatDetail represents detailed cache statistics.
type CacheStatDetail struct {
	Hits         int64   `json:"hits"`
	Misses       int64   `json:"misses"`
	HitRate      float64 `json:"hit_rate"`
	SizeBytes    int64   `json:"size_bytes"`
	Entries      int64   `json:"entries"`
	AvgLatencyMs int64   `json:"avg_latency_ms"`
	MaxSize      int64   `json:"max_size"`
	Evictions    int64   `json:"evictions"`
}

// CacheConfigRequest represents cache configuration update.
type CacheConfigRequest struct {
	Enabled                        *bool              `json:"enabled"`
	Strategy                       *string            `json:"strategy"`
	SimilarityThreshold            *float64           `json:"similarity_threshold"`
	DefaultTTLSeconds              *int               `json:"default_ttl_seconds"`
	MaxEntries                     *int               `json:"max_entries"`
	EvictionPolicy                 *string            `json:"eviction_policy"`
	VectorEnabled                  *bool              `json:"vector_enabled"`
	VectorDimension                *int               `json:"vector_dimension"`
	VectorQueryTimeoutMs           *int               `json:"vector_query_timeout_ms"`
	VectorThresholds               map[string]float64 `json:"vector_thresholds"`
	VectorPipelineEnabled          *bool              `json:"vector_pipeline_enabled"`
	VectorStandardKeyVersion       *string            `json:"vector_standard_key_version"`
	VectorEmbeddingProvider        *string            `json:"vector_embedding_provider"`
	VectorOllamaBaseURL            *string            `json:"vector_ollama_base_url"`
	VectorOllamaEmbeddingModel     *string            `json:"vector_ollama_embedding_model"`
	VectorOllamaEmbeddingDimension *int               `json:"vector_ollama_embedding_dimension"`
	VectorOllamaEmbeddingTimeoutMs *int               `json:"vector_ollama_embedding_timeout_ms"`
	VectorOllamaEndpointMode       *string            `json:"vector_ollama_endpoint_mode"`
	VectorWritebackEnabled         *bool              `json:"vector_writeback_enabled"`
	ColdVectorEnabled              *bool              `json:"cold_vector_enabled"`
	ColdVectorQueryEnabled         *bool              `json:"cold_vector_query_enabled"`
	ColdVectorBackend              *string            `json:"cold_vector_backend"`
	ColdVectorDualWriteEnabled     *bool              `json:"cold_vector_dual_write_enabled"`
	ColdVectorSimilarityThreshold  *float64           `json:"cold_vector_similarity_threshold"`
	ColdVectorTopK                 *int               `json:"cold_vector_top_k"`
	HotMemoryHighWatermarkPercent  *float64           `json:"hot_memory_high_watermark_percent"`
	HotMemoryReliefPercent         *float64           `json:"hot_memory_relief_percent"`
	HotToColdBatchSize             *int               `json:"hot_to_cold_batch_size"`
	HotToColdIntervalSeconds       *int               `json:"hot_to_cold_interval_seconds"`
	ColdVectorQdrantURL            *string            `json:"cold_vector_qdrant_url"`
	ColdVectorQdrantAPIKey         *string            `json:"cold_vector_qdrant_api_key"`
	ColdVectorQdrantCollection     *string            `json:"cold_vector_qdrant_collection"`
	ColdVectorQdrantTimeoutMs      *int               `json:"cold_vector_qdrant_timeout_ms"`
	Dedup                          *struct {
		Enabled               *bool `json:"enabled"`
		MaxPending            *int  `json:"max_pending"`
		RequestTimeoutSeconds *int  `json:"request_timeout_seconds"`
	} `json:"dedup"`

	RequestTTL *int `json:"request_ttl_seconds"`
	ContextTTL *int `json:"context_ttl_seconds"`
	RouteTTL   *int `json:"route_ttl_seconds"`
}

// VectorTierConfigRequest represents vector tier configuration update.
// All fields are optional and support partial update.
type VectorTierConfigRequest struct {
	ColdVectorEnabled             *bool    `json:"cold_vector_enabled"`
	ColdVectorQueryEnabled        *bool    `json:"cold_vector_query_enabled"`
	ColdVectorBackend             *string  `json:"cold_vector_backend"`
	ColdVectorDualWriteEnabled    *bool    `json:"cold_vector_dual_write_enabled"`
	ColdVectorSimilarityThreshold *float64 `json:"cold_vector_similarity_threshold"`
	ColdVectorTopK                *int     `json:"cold_vector_top_k"`
	HotMemoryHighWatermarkPercent *float64 `json:"hot_memory_high_watermark_percent"`
	HotMemoryReliefPercent        *float64 `json:"hot_memory_relief_percent"`
	HotToColdBatchSize            *int     `json:"hot_to_cold_batch_size"`
	HotToColdIntervalSeconds      *int     `json:"hot_to_cold_interval_seconds"`
	ColdVectorQdrantURL           *string  `json:"cold_vector_qdrant_url"`
	ColdVectorQdrantAPIKey        *string  `json:"cold_vector_qdrant_api_key"`
	ColdVectorQdrantCollection    *string  `json:"cold_vector_qdrant_collection"`
	ColdVectorQdrantTimeoutMs     *int     `json:"cold_vector_qdrant_timeout_ms"`
}

// Dashboard types

// DashboardStats represents dashboard overview statistics.
type DashboardStats struct {
	TotalRequests   int64          `json:"total_requests"`
	RequestsToday   int64          `json:"requests_today"`
	SuccessRate     float64        `json:"success_rate"`
	AvgLatencyMs    int64          `json:"avg_latency_ms"`
	TotalTokens     int64          `json:"total_tokens"`
	ActiveAccounts  int            `json:"active_accounts"`
	ActiveProviders int            `json:"active_providers"`
	CacheHitRate    float64        `json:"cache_hit_rate"`
	ProviderStats   []ProviderStat `json:"provider_stats"`
	TopModels       []ModelStat    `json:"top_models"`
}

// ProviderStat represents provider statistics.
type ProviderStat struct {
	Name        string  `json:"name"`
	Requests    int64   `json:"requests"`
	Tokens      int64   `json:"tokens"`
	SuccessRate float64 `json:"success_rate"`
	AvgLatency  int64   `json:"avg_latency_ms"`
}

// ModelStat represents model usage statistics.
type ModelStat struct {
	Name     string `json:"name"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// RequestTrend represents request trend data point.
type RequestTrend struct {
	Timestamp time.Time `json:"timestamp"`
	Requests  int64     `json:"requests"`
	Success   int64     `json:"success"`
	Failed    int64     `json:"failed"`
	Latency   int64     `json:"avg_latency_ms"`
}

// AlertListItem represents an alert in list responses.
type AlertListItem struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Level        string    `json:"level"`
	Message      string    `json:"message"`
	AccountID    string    `json:"account_id,omitempty"`
	Provider     string    `json:"provider,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}

// Gateway management types

// GatewayRequest represents a request to create/update a gateway.
type GatewayRequest struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Endpoint            string `json:"endpoint"`
	APIKey              string `json:"api_key"`
	Description         string `json:"description"`
	Enabled             bool   `json:"enabled"`
	Priority            int    `json:"priority"`
	Timeout             int    `json:"timeout"`
	MaxRetries          int    `json:"max_retries"`
	HealthCheckEnabled  bool   `json:"health_check_enabled"`
	HealthCheckInterval int    `json:"health_check_interval"`
}

// GatewayResponse represents a gateway in responses.
type GatewayResponse struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Endpoint            string    `json:"endpoint"`
	Description         string    `json:"description"`
	Enabled             bool      `json:"enabled"`
	Priority            int       `json:"priority"`
	Timeout             int       `json:"timeout"`
	MaxRetries          int       `json:"max_retries"`
	HealthCheckEnabled  bool      `json:"health_check_enabled"`
	HealthCheckInterval int       `json:"health_check_interval"`
	LastHealthCheck     time.Time `json:"last_health_check,omitempty"`
	Healthy             bool      `json:"healthy"`
	Latency             int64     `json:"latency"`
	RequestCount        int64     `json:"request_count"`
	SuccessRate         float64   `json:"success_rate"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// GatewayTestResult represents gateway connectivity test result.
type GatewayTestResult struct {
	Success   bool      `json:"success"`
	Latency   int64     `json:"latency"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// GatewayHealthHistory represents gateway health check history.
type GatewayHealthHistory struct {
	Timestamp time.Time `json:"timestamp"`
	Healthy   bool      `json:"healthy"`
	Latency   int64     `json:"latency"`
}
