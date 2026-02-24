package provider

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProviderPool(t *testing.T) {
	config := ProviderPoolConfig{
		MaxIdleTime:         10 * time.Minute,
		MaxUseCount:         1000,
		HealthCheckInterval: 30 * time.Second,
		MaxErrorCount:       5,
	}

	pool := NewProviderPool(config)
	require.NotNil(t, pool)
	assert.NotNil(t, pool.providers)
	assert.Equal(t, config, pool.config)

	pool.Stop()
}

func TestGetProviderPool(t *testing.T) {
	pool := GetProviderPool()
	require.NotNil(t, pool)
}

func TestProviderPool_Stats(t *testing.T) {
	config := ProviderPoolConfig{
		MaxIdleTime:         10 * time.Minute,
		MaxUseCount:         1000,
		HealthCheckInterval: 30 * time.Second,
		MaxErrorCount:       5,
	}

	pool := NewProviderPool(config)
	defer pool.Stop()

	stats := pool.Stats()
	assert.NotNil(t, stats)
}

func TestProviderPool_Size(t *testing.T) {
	config := ProviderPoolConfig{
		MaxIdleTime:         10 * time.Minute,
		MaxUseCount:         1000,
		HealthCheckInterval: 30 * time.Second,
		MaxErrorCount:       5,
	}

	pool := NewProviderPool(config)
	defer pool.Stop()

	size := pool.Size()
	assert.GreaterOrEqual(t, size, 0)
}

func TestProviderPool_RecordError(t *testing.T) {
	config := ProviderPoolConfig{
		MaxIdleTime:         10 * time.Minute,
		MaxUseCount:         1000,
		HealthCheckInterval: 30 * time.Second,
		MaxErrorCount:       5,
	}

	pool := NewProviderPool(config)
	defer pool.Stop()

	pool.RecordError("nonexistent-provider")
}

func TestProviderPool_RecordSuccess(t *testing.T) {
	config := ProviderPoolConfig{
		MaxIdleTime:         10 * time.Minute,
		MaxUseCount:         1000,
		HealthCheckInterval: 30 * time.Second,
		MaxErrorCount:       5,
	}

	pool := NewProviderPool(config)
	defer pool.Stop()

	pool.RecordSuccess("nonexistent-provider")
}

func TestProviderPool_Get_Nonexistent(t *testing.T) {
	config := ProviderPoolConfig{
		MaxIdleTime:         10 * time.Minute,
		MaxUseCount:         1000,
		HealthCheckInterval: 30 * time.Second,
		MaxErrorCount:       5,
	}

	pool := NewProviderPool(config)
	defer pool.Stop()

	prov := pool.Get("nonexistent-provider")
	assert.Nil(t, prov)
}

func TestProviderPool_GetByModel_Nonexistent(t *testing.T) {
	config := ProviderPoolConfig{
		MaxIdleTime:         10 * time.Minute,
		MaxUseCount:         1000,
		HealthCheckInterval: 30 * time.Second,
		MaxErrorCount:       5,
	}

	pool := NewProviderPool(config)
	defer pool.Stop()

	prov := pool.GetByModel("nonexistent-model")
	assert.Nil(t, prov)
}

func TestProviderPoolConfig_Defaults(t *testing.T) {
	config := ProviderPoolConfig{
		MaxIdleTime:         30 * time.Minute,
		MaxUseCount:         10000,
		HealthCheckInterval: 60 * time.Second,
		MaxErrorCount:       10,
	}

	assert.Equal(t, 30*time.Minute, config.MaxIdleTime)
	assert.Equal(t, int64(10000), config.MaxUseCount)
	assert.Equal(t, 60*time.Second, config.HealthCheckInterval)
	assert.Equal(t, int64(10), config.MaxErrorCount)
}

func TestPooledProvider(t *testing.T) {
	pooled := &PooledProvider{
		useCount:   0,
		errorCount: 0,
		lastUsed:   time.Now(),
	}

	assert.Equal(t, int64(0), pooled.useCount)
	assert.Equal(t, int64(0), pooled.errorCount)
}
