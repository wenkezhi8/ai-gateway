package container

import (
	"testing"

	"ai-gateway/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContainer(t *testing.T) {
	c1 := GetContainer()
	c2 := GetContainer()

	assert.NotNil(t, c1)
	assert.Same(t, c1, c2, "GetContainer should return singleton")
}

func TestServiceContainer_Initialize(t *testing.T) {
	c := &ServiceContainer{}
	cfg := &config.Config{
		Server: config.ServerConfig{Port: "8566"},
	}

	err := c.Initialize(cfg)
	require.NoError(t, err)
	assert.True(t, c.IsInitialized())

	assert.NotNil(t, c.Config())
	assert.NotNil(t, c.Registry())
	assert.NotNil(t, c.CacheManager())
	assert.NotNil(t, c.SmartRouter())
	assert.NotNil(t, c.Storage())
	assert.NotNil(t, c.Security())
}

func TestServiceContainer_Initialize_Idempotent(t *testing.T) {
	c := &ServiceContainer{}
	cfg := &config.Config{Server: config.ServerConfig{Port: "8566"}}

	err1 := c.Initialize(cfg)
	err2 := c.Initialize(cfg)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.True(t, c.IsInitialized())
}

func TestServiceContainer_GetService(t *testing.T) {
	c := &ServiceContainer{}
	cfg := &config.Config{Server: config.ServerConfig{Port: "8566"}}
	err := c.Initialize(cfg)
	require.NoError(t, err)

	tests := []struct {
		name        string
		serviceName string
		shouldExist bool
	}{
		{"config", "config", true},
		{"registry", "registry", true},
		{"cacheManager", "cacheManager", true},
		{"smartRouter", "smartRouter", true},
		{"storage", "storage", true},
		{"security", "security", true},
		{"unknown", "unknown", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := c.GetService(tt.serviceName)
			if tt.shouldExist {
				assert.NotNil(t, service, "service %s should exist", tt.serviceName)
			} else {
				assert.Nil(t, service, "service %s should not exist", tt.serviceName)
			}
		})
	}
}

func TestServiceContainer_Accessors_BeforeInit(t *testing.T) {
	c := &ServiceContainer{}

	assert.Nil(t, c.Config())
	assert.Nil(t, c.Registry())
	assert.Nil(t, c.CacheManager())
	assert.Nil(t, c.SmartRouter())
	assert.Nil(t, c.Storage())
	assert.Nil(t, c.Security())
	assert.False(t, c.IsInitialized())
}

func TestServiceContainer_Close(t *testing.T) {
	c := &ServiceContainer{}
	cfg := &config.Config{Server: config.ServerConfig{Port: "8566"}}
	err := c.Initialize(cfg)
	require.NoError(t, err)

	err = c.Close()
	require.NoError(t, err)
}
