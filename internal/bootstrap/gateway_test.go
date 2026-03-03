package bootstrap

import (
	"errors"
	"testing"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/provider"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitProviderRegistry_RegistersGoogleFactory(t *testing.T) {
	provider.ClearRegistry()
	t.Cleanup(provider.ClearRegistry)

	cfg := &config.Config{}
	logger := logrus.New()

	registry := InitProviderRegistry(cfg, logger)
	p, err := registry.CreateProvider(&provider.ProviderConfig{
		Name:    "google",
		APIKey:  "test-key",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
		Models:  []string{"gemini-3.1-pro-preview"},
		Enabled: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "google", p.Name())
}

func TestInitCacheManager_VectorInitFailure_ShouldReturnError(t *testing.T) {
	originalNewCacheManager := newCacheManager
	originalInitializeVectorStore := initializeVectorStore
	t.Cleanup(func() {
		newCacheManager = originalNewCacheManager
		initializeVectorStore = originalInitializeVectorStore
	})

	newCacheManager = func(cfg cache.ManagerConfig) (*cache.Manager, error) {
		return cache.NewManagerWithCache(cache.NewMemoryCache()), nil
	}
	initializeVectorStore = func(cfg *config.Config, cacheManager *cache.Manager, logger *logrus.Logger) error {
		return errors.New("redis stack capability check failed")
	}

	cfg := config.DefaultConfig()
	cfg.VectorCache.Enabled = true
	logger := logrus.New()

	manager, err := InitCacheManager(cfg, logger)
	require.Error(t, err)
	require.Nil(t, manager)
	assert.Contains(t, err.Error(), "redis stack capability")
}

func TestInitCacheManager_VectorDisabled_ShouldSkipVectorInitializer(t *testing.T) {
	originalNewCacheManager := newCacheManager
	originalInitializeVectorStore := initializeVectorStore
	t.Cleanup(func() {
		newCacheManager = originalNewCacheManager
		initializeVectorStore = originalInitializeVectorStore
	})

	newCacheManager = func(cfg cache.ManagerConfig) (*cache.Manager, error) {
		return cache.NewManagerWithCache(cache.NewMemoryCache()), nil
	}
	initializeVectorStore = func(cfg *config.Config, cacheManager *cache.Manager, logger *logrus.Logger) error {
		return errors.New("should not be called")
	}

	cfg := config.DefaultConfig()
	cfg.VectorCache.Enabled = false
	logger := logrus.New()

	manager, err := InitCacheManager(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, manager)
}
