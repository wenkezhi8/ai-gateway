package config

import (
	"ai-gateway/internal/constants"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, constants.ServerPort, cfg.Server.Port)
	assert.Equal(t, "debug", cfg.Server.Mode)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.True(t, cfg.Limiter.Enabled)
	assert.Equal(t, 100, cfg.Limiter.Rate)
	assert.Equal(t, 200, cfg.Limiter.Burst)
	assert.False(t, cfg.IntentEngine.Enabled)
	assert.True(t, cfg.VectorCache.Enabled)
	assert.Equal(t, 1024, cfg.VectorCache.Dimension)
}

func TestConfig_Fields(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: "9090",
			Mode: "release",
		},
		Redis: RedisConfig{
			Host:     "redis.example.com",
			Port:     6380,
			Password: "secret",
			DB:       1,
		},
		Database: DatabaseConfig{
			Path: "/data/gateway.db",
		},
		Providers: []ProviderConfig{
			{Name: "openai", APIKey: "sk-test", BaseURL: "https://api.openai.com", Enabled: true},
		},
		Limiter: LimiterConfig{
			Enabled: true,
			Rate:    50,
			Burst:   100,
			PerUser: true,
		},
		IntentEngine: IntentEngineConfig{
			Enabled:   true,
			BaseURL:   "http://127.0.0.1:18566",
			TimeoutMs: 1500,
			Language:  "zh-CN",
		},
		VectorCache: VectorCacheConfig{
			Enabled:        true,
			Dimension:      1024,
			QueryTimeoutMs: 1200,
		},
	}

	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "redis.example.com", cfg.Redis.Host)
	assert.Equal(t, 1, cfg.Redis.DB)
	assert.Len(t, cfg.Providers, 1)
	assert.Equal(t, 50, cfg.Limiter.Rate)
	assert.True(t, cfg.IntentEngine.Enabled)
	assert.Equal(t, 1024, cfg.VectorCache.Dimension)
}

func TestLoad_NoFile(t *testing.T) {
	// Set a non-existent config path
	originalPath := os.Getenv("CONFIG_PATH")
	defer os.Setenv("CONFIG_PATH", originalPath)

	os.Setenv("CONFIG_PATH", "/non/existent/path/config.json")

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should return default config
	assert.Equal(t, constants.ServerPort, cfg.Server.Port)
}

func TestLoad_FromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"server": {
			"port": "3000",
			"mode": "release"
		},
		"redis": {
			"host": "custom.redis.com",
			"port": 6380
		},
		"providers": [
			{
				"name": "custom-provider",
				"api_key": "custom-key",
				"base_url": "https://custom.api.com",
				"enabled": true
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set config path
	originalPath := os.Getenv("CONFIG_PATH")
	defer os.Setenv("CONFIG_PATH", originalPath)
	os.Setenv("CONFIG_PATH", configPath)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "3000", cfg.Server.Port)
	assert.Equal(t, "release", cfg.Server.Mode)
	assert.Equal(t, "custom.redis.com", cfg.Redis.Host)
	assert.Len(t, cfg.Providers, 1)
	assert.Equal(t, "custom-provider", cfg.Providers[0].Name)
}

func TestLoad_EnvOverride(t *testing.T) {
	// Set environment variables
	originalPort := os.Getenv("SERVER_PORT")
	originalMode := os.Getenv("GIN_MODE")
	originalRedis := os.Getenv("REDIS_HOST")
	originalIntentEnabled := os.Getenv("INTENT_ENGINE_ENABLED")
	originalIntentURL := os.Getenv("INTENT_ENGINE_BASE_URL")
	originalVectorDim := os.Getenv("VECTOR_CACHE_DIMENSION")
	defer func() {
		os.Setenv("SERVER_PORT", originalPort)
		os.Setenv("GIN_MODE", originalMode)
		os.Setenv("REDIS_HOST", originalRedis)
		os.Setenv("INTENT_ENGINE_ENABLED", originalIntentEnabled)
		os.Setenv("INTENT_ENGINE_BASE_URL", originalIntentURL)
		os.Setenv("VECTOR_CACHE_DIMENSION", originalVectorDim)
	}()

	os.Setenv("SERVER_PORT", "5000")
	os.Setenv("GIN_MODE", "test")
	os.Setenv("REDIS_HOST", "env.redis.com")
	os.Setenv("INTENT_ENGINE_ENABLED", "true")
	os.Setenv("INTENT_ENGINE_BASE_URL", "http://localhost:18566")
	os.Setenv("VECTOR_CACHE_DIMENSION", "768")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "5000", cfg.Server.Port)
	assert.Equal(t, "test", cfg.Server.Mode)
	assert.Equal(t, "env.redis.com", cfg.Redis.Host)
	assert.True(t, cfg.IntentEngine.Enabled)
	assert.Equal(t, "http://localhost:18566", cfg.IntentEngine.BaseURL)
	assert.Equal(t, 768, cfg.VectorCache.Dimension)
}

func TestServerConfig_Fields(t *testing.T) {
	cfg := ServerConfig{
		Port: "8081",
		Mode: "release",
	}

	assert.Equal(t, "8081", cfg.Port)
	assert.Equal(t, "release", cfg.Mode)
}

func TestRedisConfig_Fields(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "password",
		DB:       2,
	}

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 6379, cfg.Port)
	assert.Equal(t, "password", cfg.Password)
	assert.Equal(t, 2, cfg.DB)
}

func TestProviderConfig_Fields(t *testing.T) {
	cfg := ProviderConfig{
		Name:    "anthropic",
		APIKey:  "sk-ant-test",
		BaseURL: "https://api.anthropic.com",
		Enabled: true,
	}

	assert.Equal(t, "anthropic", cfg.Name)
	assert.True(t, cfg.Enabled)
}

func TestLimiterConfig_Fields(t *testing.T) {
	cfg := LimiterConfig{
		Enabled:          true,
		Rate:             100,
		Burst:            200,
		PerUser:          true,
		SwitchTimeoutMs:  3000,
		WarningThreshold: 0.9,
		CheckIntervalMs:  5000,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 100, cfg.Rate)
	assert.Equal(t, 200, cfg.Burst)
	assert.Equal(t, 3000, cfg.SwitchTimeoutMs)
}

func TestAccountConfig_Fields(t *testing.T) {
	cfg := AccountConfig{
		ID:       "acc-123",
		Name:     "Primary Account",
		Provider: "openai",
		APIKey:   "sk-test",
		BaseURL:  "https://api.openai.com",
		Enabled:  true,
		Priority: 10,
		Limits: map[string]LimitConfig{
			"daily": {Type: "token", Period: "day", Limit: 10000},
		},
	}

	assert.Equal(t, "acc-123", cfg.ID)
	assert.Equal(t, 10, cfg.Priority)
	assert.Len(t, cfg.Limits, 1)
}

func TestLimitConfig_Fields(t *testing.T) {
	cfg := LimitConfig{
		Type:    "token",
		Period:  "day",
		Limit:   100000,
		Warning: 0.9,
	}

	assert.Equal(t, "token", cfg.Type)
	assert.Equal(t, "day", cfg.Period)
	assert.Equal(t, int64(100000), cfg.Limit)
}

func TestConfig_JSONMarshal(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: "8080"},
		Providers: []ProviderConfig{
			{Name: "test", Enabled: true},
		},
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var unmarshaled Config
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, cfg.Server.Port, unmarshaled.Server.Port)
	assert.Len(t, unmarshaled.Providers, 1)
}
