package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ai-gateway/internal/constants"
)

// Config holds all configuration for the AI Gateway.
type Config struct {
	Server       ServerConfig       `json:"server"`
	Redis        RedisConfig        `json:"redis"`
	Database     DatabaseConfig     `json:"database"`
	Providers    []ProviderConfig   `json:"providers"`
	Limiter      LimiterConfig      `json:"limiter"`
	Accounts     []AccountConfig    `json:"accounts"`
	IntentEngine IntentEngineConfig `json:"intent_engine"`
	VectorCache  VectorCacheConfig  `json:"vector_cache"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port string `json:"port"`
	Mode string `json:"mode"`
}

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// DatabaseConfig holds SQLite configuration.
type DatabaseConfig struct {
	Path string `json:"path"`
}

// ProviderConfig holds AI provider configuration.
type ProviderConfig struct {
	Name    string   `json:"name"`
	APIKey  string   `json:"api_key"`
	BaseURL string   `json:"base_url"`
	Enabled bool     `json:"enabled"`
	Models  []string `json:"models,omitempty"`
}

// LimiterConfig holds rate limiter configuration.
type LimiterConfig struct {
	Enabled          bool    `json:"enabled"`
	Rate             int     `json:"rate"`              // requests per second
	Burst            int     `json:"burst"`             // burst size
	PerUser          bool    `json:"per_user"`          // limit per user
	SwitchTimeoutMs  int     `json:"switch_timeout_ms"` // max time for account switch (ms)
	WarningThreshold float64 `json:"warning_threshold"` // warning threshold (0.9 = 90%)
	CheckIntervalMs  int     `json:"check_interval_ms"` // usage check interval (ms)
}

// AccountConfig holds account configuration with limits.
type AccountConfig struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Provider string                 `json:"provider"`
	APIKey   string                 `json:"api_key"`
	BaseURL  string                 `json:"base_url"`
	Enabled  bool                   `json:"enabled"`
	Priority int                    `json:"priority"`
	Limits   map[string]LimitConfig `json:"limits"`
}

// IntentEngineConfig holds local intent+embedding engine configuration.
type IntentEngineConfig struct {
	Enabled           bool   `json:"enabled"`
	BaseURL           string `json:"base_url"`
	TimeoutMs         int    `json:"timeout_ms"`
	Language          string `json:"language"`
	ExpectedDimension int    `json:"expected_dimension"`
}

// VectorCacheConfig holds Redis Stack vector cache configuration.
type VectorCacheConfig struct {
	Enabled                       bool               `json:"enabled"`
	IndexName                     string             `json:"index_name"`
	KeyPrefix                     string             `json:"key_prefix"`
	Dimension                     int                `json:"dimension"`
	QueryTimeoutMs                int                `json:"query_timeout_ms"`
	Thresholds                    map[string]float64 `json:"thresholds"`
	TTLSeconds                    map[string]int64   `json:"ttl_seconds"`
	PipelineEnabled               bool               `json:"pipeline_enabled"`
	StandardKeyVersion            string             `json:"standard_key_version"`
	EmbeddingProvider             string             `json:"embedding_provider"`
	OllamaBaseURL                 string             `json:"ollama_base_url"`
	OllamaEmbeddingModel          string             `json:"ollama_embedding_model"`
	OllamaEmbeddingDimension      int                `json:"ollama_embedding_dimension"`
	OllamaEmbeddingTimeoutMs      int                `json:"ollama_embedding_timeout_ms"`
	OllamaEndpointMode            string             `json:"ollama_endpoint_mode"`
	WritebackEnabled              bool               `json:"writeback_enabled"`
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
	ColdVectorSQLitePath          string             `json:"cold_vector_sqlite_path"`
	ColdVectorQdrantURL           string             `json:"cold_vector_qdrant_url"`
	ColdVectorQdrantAPIKey        string             `json:"cold_vector_qdrant_api_key"`
	ColdVectorQdrantCollection    string             `json:"cold_vector_qdrant_collection"`
	ColdVectorQdrantTimeoutMs     int                `json:"cold_vector_qdrant_timeout_ms"`
}

// LimitConfig holds a single limit configuration.
type LimitConfig struct {
	Type    string  `json:"type"`    // token, rpm, concurrent
	Period  string  `json:"period"`  // minute, hour, day, month
	Limit   int64   `json:"limit"`   // max value
	Warning float64 `json:"warning"` // warning threshold
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: constants.ServerPort,
			Mode: "debug",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
		Database: DatabaseConfig{
			Path: "./data/ai-gateway.db",
		},
		Providers: []ProviderConfig{},
		Limiter: LimiterConfig{
			Enabled:          true,
			Rate:             100,
			Burst:            200,
			PerUser:          true,
			SwitchTimeoutMs:  3000,
			WarningThreshold: 0.9,
			CheckIntervalMs:  5000,
		},
		Accounts: []AccountConfig{},
		IntentEngine: IntentEngineConfig{
			Enabled:           false,
			BaseURL:           "http://127.0.0.1:18566",
			TimeoutMs:         1500,
			Language:          "zh-CN",
			ExpectedDimension: 1024,
		},
		VectorCache: VectorCacheConfig{
			Enabled:        true,
			IndexName:      "idx_ai_cache_v2",
			KeyPrefix:      "ai:v2:cache:",
			Dimension:      1024,
			QueryTimeoutMs: 1200,
			Thresholds: map[string]float64{
				"calc":      0.97,
				"translate": 0.96,
				"weather":   0.95,
				"qa":        0.93,
				"chat":      0.92,
			},
			TTLSeconds: map[string]int64{
				"calc":      30 * 24 * 3600,
				"translate": 14 * 24 * 3600,
				"weather":   30 * 60,
				"qa":        24 * 3600,
				"chat":      12 * 3600,
			},
			PipelineEnabled:               true,
			StandardKeyVersion:            "v2",
			EmbeddingProvider:             "ollama",
			OllamaBaseURL:                 "http://127.0.0.1:11434",
			OllamaEmbeddingModel:          "nomic-embed-text",
			OllamaEmbeddingDimension:      1024,
			OllamaEmbeddingTimeoutMs:      1500,
			OllamaEndpointMode:            "auto",
			WritebackEnabled:              true,
			ColdVectorEnabled:             false,
			ColdVectorQueryEnabled:        true,
			ColdVectorBackend:             "sqlite",
			ColdVectorDualWriteEnabled:    false,
			ColdVectorSimilarityThreshold: 0.92,
			ColdVectorTopK:                1,
			HotMemoryHighWatermarkPercent: 75,
			HotMemoryReliefPercent:        65,
			HotToColdBatchSize:            500,
			HotToColdIntervalSeconds:      30,
			ColdVectorSQLitePath:          "data/ai-gateway-cold-vectors.db",
			ColdVectorQdrantCollection:    "ai_gateway_cold_vectors",
			ColdVectorQdrantTimeoutMs:     1500,
		},
	}
}

// Load reads configuration from file and environment.
//
//nolint:gocyclo // Keep centralized env+file loading flow to avoid behavior drift.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs/config.json"
	}

	if _, err := os.Stat(configPath); err == nil {
		file, err := os.ReadFile(filepath.Clean(configPath))
		if err != nil {
			return nil, err
		}

		// Expand environment variables in the config file
		file = []byte(expandEnvVars(string(file)))

		if err := json.Unmarshal(file, cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables if set
	// Server configuration
	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Server.Port = port
	}
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		cfg.Server.Mode = mode
	}

	// Redis configuration
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		if p, err := strconv.Atoi(redisPort); err == nil {
			cfg.Redis.Port = p
		}
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}
	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		if d, err := strconv.Atoi(redisDB); err == nil {
			cfg.Redis.DB = d
		}
	}

	// Intent engine configuration
	if enabled := os.Getenv("INTENT_ENGINE_ENABLED"); enabled != "" {
		cfg.IntentEngine.Enabled = parseBool(enabled)
	}
	if baseURL := os.Getenv("INTENT_ENGINE_BASE_URL"); baseURL != "" {
		cfg.IntentEngine.BaseURL = baseURL
	}
	if timeout := os.Getenv("INTENT_ENGINE_TIMEOUT_MS"); timeout != "" {
		if v, err := strconv.Atoi(timeout); err == nil {
			cfg.IntentEngine.TimeoutMs = v
		}
	}
	if language := os.Getenv("INTENT_ENGINE_LANGUAGE"); language != "" {
		cfg.IntentEngine.Language = language
	}
	if dim := os.Getenv("INTENT_ENGINE_EXPECTED_DIMENSION"); dim != "" {
		if v, err := strconv.Atoi(dim); err == nil {
			cfg.IntentEngine.ExpectedDimension = v
		}
	}

	// Vector cache configuration
	if enabled := os.Getenv("VECTOR_CACHE_ENABLED"); enabled != "" {
		cfg.VectorCache.Enabled = parseBool(enabled)
	}
	if indexName := os.Getenv("VECTOR_CACHE_INDEX_NAME"); indexName != "" {
		cfg.VectorCache.IndexName = indexName
	}
	if keyPrefix := os.Getenv("VECTOR_CACHE_KEY_PREFIX"); keyPrefix != "" {
		cfg.VectorCache.KeyPrefix = keyPrefix
	}
	if dim := os.Getenv("VECTOR_CACHE_DIMENSION"); dim != "" {
		if v, err := strconv.Atoi(dim); err == nil {
			cfg.VectorCache.Dimension = v
		}
	}
	if timeout := os.Getenv("VECTOR_CACHE_QUERY_TIMEOUT_MS"); timeout != "" {
		if v, err := strconv.Atoi(timeout); err == nil {
			cfg.VectorCache.QueryTimeoutMs = v
		}
	}
	if enabled := os.Getenv("VECTOR_PIPELINE_ENABLED"); enabled != "" {
		cfg.VectorCache.PipelineEnabled = parseBool(enabled)
	}
	if version := os.Getenv("VECTOR_STANDARD_KEY_VERSION"); version != "" {
		cfg.VectorCache.StandardKeyVersion = strings.TrimSpace(version)
	}
	if provider := os.Getenv("VECTOR_EMBEDDING_PROVIDER"); provider != "" {
		cfg.VectorCache.EmbeddingProvider = strings.ToLower(strings.TrimSpace(provider))
	}
	if baseURL := os.Getenv("VECTOR_OLLAMA_BASE_URL"); baseURL != "" {
		cfg.VectorCache.OllamaBaseURL = strings.TrimSpace(baseURL)
	}
	if model := os.Getenv("VECTOR_OLLAMA_EMBEDDING_MODEL"); model != "" {
		cfg.VectorCache.OllamaEmbeddingModel = strings.TrimSpace(model)
	}
	if dim := os.Getenv("VECTOR_OLLAMA_EMBEDDING_DIMENSION"); dim != "" {
		if v, err := strconv.Atoi(dim); err == nil {
			cfg.VectorCache.OllamaEmbeddingDimension = v
		}
	}
	if timeout := os.Getenv("VECTOR_OLLAMA_EMBEDDING_TIMEOUT_MS"); timeout != "" {
		if v, err := strconv.Atoi(timeout); err == nil {
			cfg.VectorCache.OllamaEmbeddingTimeoutMs = v
		}
	}
	if mode := os.Getenv("VECTOR_OLLAMA_ENDPOINT_MODE"); mode != "" {
		cfg.VectorCache.OllamaEndpointMode = strings.ToLower(strings.TrimSpace(mode))
	}
	if enabled := os.Getenv("VECTOR_WRITEBACK_ENABLED"); enabled != "" {
		cfg.VectorCache.WritebackEnabled = parseBool(enabled)
	}
	if enabled := os.Getenv("VECTOR_COLD_ENABLED"); enabled != "" {
		cfg.VectorCache.ColdVectorEnabled = parseBool(enabled)
	}
	if enabled := os.Getenv("VECTOR_COLD_QUERY_ENABLED"); enabled != "" {
		cfg.VectorCache.ColdVectorQueryEnabled = parseBool(enabled)
	}
	if backend := os.Getenv("VECTOR_COLD_BACKEND"); backend != "" {
		cfg.VectorCache.ColdVectorBackend = strings.ToLower(strings.TrimSpace(backend))
	}
	if enabled := os.Getenv("VECTOR_COLD_DUAL_WRITE_ENABLED"); enabled != "" {
		cfg.VectorCache.ColdVectorDualWriteEnabled = parseBool(enabled)
	}
	if threshold := os.Getenv("VECTOR_COLD_SIMILARITY_THRESHOLD"); threshold != "" {
		if v, err := strconv.ParseFloat(threshold, 64); err == nil {
			cfg.VectorCache.ColdVectorSimilarityThreshold = v
		}
	}
	if topK := os.Getenv("VECTOR_COLD_TOP_K"); topK != "" {
		if v, err := strconv.Atoi(topK); err == nil {
			cfg.VectorCache.ColdVectorTopK = v
		}
	}
	if watermark := os.Getenv("VECTOR_HOT_MEMORY_HIGH_WATERMARK_PERCENT"); watermark != "" {
		if v, err := strconv.ParseFloat(watermark, 64); err == nil {
			cfg.VectorCache.HotMemoryHighWatermarkPercent = v
		}
	}
	if relief := os.Getenv("VECTOR_HOT_MEMORY_RELIEF_PERCENT"); relief != "" {
		if v, err := strconv.ParseFloat(relief, 64); err == nil {
			cfg.VectorCache.HotMemoryReliefPercent = v
		}
	}
	if batchSize := os.Getenv("VECTOR_HOT_TO_COLD_BATCH_SIZE"); batchSize != "" {
		if v, err := strconv.Atoi(batchSize); err == nil {
			cfg.VectorCache.HotToColdBatchSize = v
		}
	}
	if interval := os.Getenv("VECTOR_HOT_TO_COLD_INTERVAL_SECONDS"); interval != "" {
		if v, err := strconv.Atoi(interval); err == nil {
			cfg.VectorCache.HotToColdIntervalSeconds = v
		}
	}
	if sqlitePath := os.Getenv("VECTOR_COLD_SQLITE_PATH"); sqlitePath != "" {
		cfg.VectorCache.ColdVectorSQLitePath = sqlitePath
	}
	if qdrantURL := os.Getenv("VECTOR_COLD_QDRANT_URL"); qdrantURL != "" {
		cfg.VectorCache.ColdVectorQdrantURL = qdrantURL
	}
	if qdrantKey := os.Getenv("VECTOR_COLD_QDRANT_API_KEY"); qdrantKey != "" {
		cfg.VectorCache.ColdVectorQdrantAPIKey = qdrantKey
	}
	if qdrantCollection := os.Getenv("VECTOR_COLD_QDRANT_COLLECTION"); qdrantCollection != "" {
		cfg.VectorCache.ColdVectorQdrantCollection = qdrantCollection
	}
	if qdrantTimeout := os.Getenv("VECTOR_COLD_QDRANT_TIMEOUT_MS"); qdrantTimeout != "" {
		if v, err := strconv.Atoi(qdrantTimeout); err == nil {
			cfg.VectorCache.ColdVectorQdrantTimeoutMs = v
		}
	}

	// Database configuration
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}

	// Load providers from environment variables (takes precedence over config file)
	// Format: PROVIDER_<NAME>_API_KEY (e.g., PROVIDER_OPENAI_API_KEY)
	cfg.loadProvidersFromEnv()

	return cfg, nil
}

// expandEnvVars expands environment variables in the format ${VAR_NAME} or $VAR_NAME.
func expandEnvVars(s string) string {
	return os.Expand(s, func(key string) string {
		// Handle ${VAR} format
		if strings.HasPrefix(key, "{") && strings.HasSuffix(key, "}") {
			key = key[1 : len(key)-1]
		}
		return os.Getenv(key)
	})
}

// loadProvidersFromEnv loads provider configurations from environment variables
// Environment variable format:
//   - PROVIDER_<NAME>_API_KEY: API key for the provider
//   - PROVIDER_<NAME>_BASE_URL: Base URL for the provider (optional)
//   - PROVIDER_<NAME>_ENABLED: Enable/disable the provider (default: true if API key is set)
//
// Example:
//
//	PROVIDER_OPENAI_API_KEY=sk-xxx
//	PROVIDER_OPENAI_BASE_URL=https://api.openai.com/v1
//	PROVIDER_ANTHROPIC_API_KEY=sk-ant-xxx
func (c *Config) loadProvidersFromEnv() {
	// Define known providers and their env var prefixes
	providerPrefixes := []string{"OPENAI", "ANTHROPIC", "AZURE", "VOLCENGINE", "CLAUDE", "DEEPSEEK"}

	for _, prefix := range providerPrefixes {
		apiKeyEnv := "PROVIDER_" + prefix + "_API_KEY"
		apiKey := os.Getenv(apiKeyEnv)

		if apiKey == "" {
			continue
		}

		// Get provider name (lowercase)
		providerName := strings.ToLower(prefix)
		if prefix == "CLAUDE" {
			providerName = "anthropic" // Claude uses Anthropic API
		}

		// Get base URL from env (optional)
		baseURLEnv := "PROVIDER_" + prefix + "_BASE_URL"
		baseURL := os.Getenv(baseURLEnv)

		// Get enabled status from env (optional, default true)
		enabled := true
		if enabledEnv := os.Getenv("PROVIDER_" + prefix + "_ENABLED"); enabledEnv != "" {
			enabled = strings.EqualFold(enabledEnv, "true") || enabledEnv == "1"
		}

		// Check if provider already exists in config
		found := false
		for i, p := range c.Providers {
			if !strings.EqualFold(p.Name, providerName) {
				continue
			}

			// Override with env values
			c.Providers[i].APIKey = apiKey
			if baseURL != "" {
				c.Providers[i].BaseURL = baseURL
			}
			c.Providers[i].Enabled = enabled
			found = true
			break
		}

		// Add new provider if not found
		if !found {
			c.Providers = append(c.Providers, ProviderConfig{
				Name:    providerName,
				APIKey:  apiKey,
				BaseURL: baseURL,
				Enabled: enabled,
			})
		}
	}
}

// Validate validates the configuration and returns an error if invalid.
//
//nolint:gocyclo // Keep validation in one place for predictable config errors.
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port == "" {
		return &ValidationError{Field: "server.port", Message: "port is required"}
	}

	// Validate port is a valid number
	if port, err := strconv.Atoi(c.Server.Port); err != nil || port <= 0 || port > 65535 {
		return &ValidationError{Field: "server.port", Message: "port must be a valid number between 1 and 65535"}
	}

	if c.Server.Mode != "debug" && c.Server.Mode != "release" && c.Server.Mode != "test" {
		return &ValidationError{Field: "server.mode", Message: "mode must be 'debug', 'release', or 'test'"}
	}

	// Validate at least one provider is configured
	hasEnabledProvider := false
	providerNames := make(map[string]bool)
	for i, p := range c.Providers {
		if !p.Enabled {
			continue
		}

		if p.Name == "" {
			return &ValidationError{Field: "providers", Message: "provider name is required"}
		}

		// Check for duplicate provider names
		if providerNames[p.Name] {
			return &ValidationError{Field: "providers", Message: "duplicate provider name: " + p.Name}
		}
		providerNames[p.Name] = true

		// Validate provider name is supported
		supportedProviders := []string{
			"openai", "anthropic", "azure-openai", "volcengine",
			"deepseek", "zhipu", "qwen", "moonshot", "minimax",
			"baichuan", "yi", "google", "mistral",
		}
		isSupported := false
		for _, supported := range supportedProviders {
			if strings.EqualFold(p.Name, supported) {
				isSupported = true
				break
			}
		}
		if !isSupported {
			return &ValidationError{Field: "providers[" + strconv.Itoa(i) + "].name",
				Message: "unsupported provider: " + p.Name + ", supported: " + strings.Join(supportedProviders, ", ")}
		}

		// Note: API key validation is optional as some providers may use auth headers.
		hasEnabledProvider = true
	}

	if !hasEnabledProvider {
		return &ValidationError{Field: "providers", Message: "at least one enabled provider is required"}
	}

	// Validate limiter configuration if enabled
	if c.Limiter.Enabled {
		if c.Limiter.Rate <= 0 {
			return &ValidationError{Field: "limiter.rate", Message: "rate must be positive"}
		}
		if c.Limiter.Burst <= 0 {
			return &ValidationError{Field: "limiter.burst", Message: "burst must be positive"}
		}
		if c.Limiter.WarningThreshold <= 0 || c.Limiter.WarningThreshold > 1 {
			return &ValidationError{Field: "limiter.warning_threshold", Message: "warning_threshold must be between 0 and 1"}
		}
		if c.Limiter.SwitchTimeoutMs < 0 {
			return &ValidationError{Field: "limiter.switch_timeout_ms", Message: "switch_timeout_ms cannot be negative"}
		}
		if c.Limiter.CheckIntervalMs <= 0 {
			return &ValidationError{Field: "limiter.check_interval_ms", Message: "check_interval_ms must be positive"}
		}
	}

	// Validate account configurations
	accountIDs := make(map[string]bool)
	for i, acc := range c.Accounts {
		if !acc.Enabled {
			continue
		}

		if acc.ID == "" {
			return &ValidationError{Field: "accounts[" + strconv.Itoa(i) + "].id", Message: "account ID is required"}
		}

		// Check for duplicate account IDs
		if accountIDs[acc.ID] {
			return &ValidationError{Field: "accounts[" + strconv.Itoa(i) + "].id", Message: "duplicate account ID: " + acc.ID}
		}
		accountIDs[acc.ID] = true

		if acc.Provider == "" {
			return &ValidationError{Field: "accounts[" + strconv.Itoa(i) + "].provider", Message: "provider is required"}
		}

		// Validate limits if present.
		for limitName, limit := range acc.Limits {
			if limit.Type == "" {
				return &ValidationError{Field: "accounts[" + strconv.Itoa(i) + "].limits." + limitName + ".type",
					Message: "limit type is required"}
			}
			if limit.Limit <= 0 {
				return &ValidationError{Field: "accounts[" + strconv.Itoa(i) + "].limits." + limitName + ".limit",
					Message: "limit must be positive"}
			}
			if limit.Warning < 0 || limit.Warning > 1 {
				return &ValidationError{Field: "accounts[" + strconv.Itoa(i) + "].limits." + limitName + ".warning",
					Message: "warning must be between 0 and 1"}
			}
		}
	}

	if c.IntentEngine.Enabled {
		if strings.TrimSpace(c.IntentEngine.BaseURL) == "" {
			return &ValidationError{Field: "intent_engine.base_url", Message: "base_url is required when intent_engine is enabled"}
		}
		if c.IntentEngine.TimeoutMs <= 0 {
			return &ValidationError{Field: "intent_engine.timeout_ms", Message: "timeout_ms must be positive"}
		}
	}

	if c.VectorCache.Enabled {
		if strings.TrimSpace(c.VectorCache.IndexName) == "" {
			return &ValidationError{Field: "vector_cache.index_name", Message: "index_name is required when vector_cache is enabled"}
		}
		if strings.TrimSpace(c.VectorCache.KeyPrefix) == "" {
			return &ValidationError{Field: "vector_cache.key_prefix", Message: "key_prefix is required when vector_cache is enabled"}
		}
		if c.VectorCache.Dimension <= 0 {
			return &ValidationError{Field: "vector_cache.dimension", Message: "dimension must be positive"}
		}
		if c.VectorCache.ColdVectorSimilarityThreshold < 0 || c.VectorCache.ColdVectorSimilarityThreshold > 1 {
			return &ValidationError{Field: "vector_cache.cold_vector_similarity_threshold", Message: "cold_vector_similarity_threshold must be between 0 and 1"}
		}
		if c.VectorCache.ColdVectorTopK <= 0 {
			return &ValidationError{Field: "vector_cache.cold_vector_top_k", Message: "cold_vector_top_k must be positive"}
		}
		if c.VectorCache.HotMemoryHighWatermarkPercent <= 0 || c.VectorCache.HotMemoryHighWatermarkPercent > 100 {
			return &ValidationError{Field: "vector_cache.hot_memory_high_watermark_percent", Message: "hot_memory_high_watermark_percent must be between 0 and 100"}
		}
		if c.VectorCache.HotMemoryReliefPercent <= 0 || c.VectorCache.HotMemoryReliefPercent >= c.VectorCache.HotMemoryHighWatermarkPercent {
			return &ValidationError{Field: "vector_cache.hot_memory_relief_percent", Message: "hot_memory_relief_percent must be positive and less than high watermark"}
		}
		if c.VectorCache.HotToColdBatchSize <= 0 {
			return &ValidationError{Field: "vector_cache.hot_to_cold_batch_size", Message: "hot_to_cold_batch_size must be positive"}
		}
		if c.VectorCache.HotToColdIntervalSeconds <= 0 {
			return &ValidationError{Field: "vector_cache.hot_to_cold_interval_seconds", Message: "hot_to_cold_interval_seconds must be positive"}
		}
		if c.VectorCache.ColdVectorEnabled {
			backend := strings.ToLower(strings.TrimSpace(c.VectorCache.ColdVectorBackend))
			if backend != "sqlite" && backend != "qdrant" {
				return &ValidationError{Field: "vector_cache.cold_vector_backend", Message: "cold_vector_backend must be sqlite or qdrant"}
			}
			if backend == "qdrant" && strings.TrimSpace(c.VectorCache.ColdVectorQdrantURL) == "" {
				return &ValidationError{Field: "vector_cache.cold_vector_qdrant_url", Message: "cold_vector_qdrant_url is required when qdrant backend is active"}
			}
		}
		if strings.TrimSpace(c.VectorCache.StandardKeyVersion) == "" {
			return &ValidationError{Field: "vector_cache.standard_key_version", Message: "standard_key_version is required when vector_cache is enabled"}
		}
		if provider := strings.ToLower(strings.TrimSpace(c.VectorCache.EmbeddingProvider)); provider == "" || provider != "ollama" {
			return &ValidationError{Field: "vector_cache.embedding_provider", Message: "embedding_provider must be ollama"}
		}
		if strings.TrimSpace(c.VectorCache.OllamaBaseURL) == "" {
			return &ValidationError{Field: "vector_cache.ollama_base_url", Message: "ollama_base_url is required when vector_cache is enabled"}
		}
		if strings.TrimSpace(c.VectorCache.OllamaEmbeddingModel) == "" {
			return &ValidationError{Field: "vector_cache.ollama_embedding_model", Message: "ollama_embedding_model is required when vector_cache is enabled"}
		}
		if c.VectorCache.OllamaEmbeddingDimension <= 0 {
			return &ValidationError{Field: "vector_cache.ollama_embedding_dimension", Message: "ollama_embedding_dimension must be positive"}
		}
		if c.VectorCache.OllamaEmbeddingTimeoutMs <= 0 {
			return &ValidationError{Field: "vector_cache.ollama_embedding_timeout_ms", Message: "ollama_embedding_timeout_ms must be positive"}
		}
		mode := strings.ToLower(strings.TrimSpace(c.VectorCache.OllamaEndpointMode))
		if mode != "auto" && mode != "embed" && mode != "embeddings" {
			return &ValidationError{Field: "vector_cache.ollama_endpoint_mode", Message: "ollama_endpoint_mode must be auto/embed/embeddings"}
		}
	}

	return nil
}

func parseBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error in " + e.Field + ": " + e.Message
}
