package router

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigManager(t *testing.T) {
	mgr := NewConfigManager()
	require.NotNil(t, mgr)
	assert.Equal(t, StrategyRoundRobin, mgr.Get().DefaultStrategy)
}

func TestConfigManager_Load_JSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	cfg := RoutingConfig{
		DefaultStrategy: StrategyWeighted,
		Strategies: []StrategyConfig{
			{Name: "weighted", Type: StrategyWeighted, Enabled: true},
		},
		Providers: []ProviderRoutingConfig{
			{Name: "openai", Weight: 100, Priority: 1},
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	mgr := NewConfigManager()
	err = mgr.Load(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, StrategyWeighted, mgr.Get().DefaultStrategy)
	assert.Len(t, mgr.Get().Providers, 1)
}

func TestConfigManager_Load_YAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	yamlContent := `
default_strategy: failover
providers:
  - name: openai
    weight: 100
    priority: 1
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	tmpFile.Close()

	mgr := NewConfigManager()
	err = mgr.Load(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, StrategyFailover, mgr.Get().DefaultStrategy)
}

func TestConfigManager_Load_NotFound(t *testing.T) {
	mgr := NewConfigManager()
	err := mgr.Load("/nonexistent/config.json")
	assert.Error(t, err)
}

func TestConfigManager_Save_JSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	mgr := NewConfigManager()
	mgr.SetDefaultStrategy(StrategyFailover)

	err = mgr.Save(tmpFile.Name())
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var cfg RoutingConfig
	err = json.Unmarshal(data, &cfg)
	require.NoError(t, err)
	assert.Equal(t, StrategyFailover, cfg.DefaultStrategy)
}

func TestConfigManager_Set(t *testing.T) {
	mgr := NewConfigManager()

	newCfg := &RoutingConfig{
		DefaultStrategy: StrategyCostOptimized,
		Providers: []ProviderRoutingConfig{
			{Name: "anthropic", Weight: 100},
		},
	}

	changed := false
	mgr.OnChange(func(_ *RoutingConfig) {
		changed = true
	})

	mgr.Set(newCfg)

	assert.Equal(t, newCfg, mgr.Get())
	assert.True(t, changed)
}

func TestConfigManager_SetDefaultStrategy(t *testing.T) {
	mgr := NewConfigManager()

	mgr.SetDefaultStrategy(StrategyCostOptimized)
	assert.Equal(t, StrategyCostOptimized, mgr.Get().DefaultStrategy)
}

func TestConfigManager_AddProviderConfig(t *testing.T) {
	mgr := NewConfigManager()

	cfg := ProviderRoutingConfig{
		Name:     "openai",
		Weight:   100,
		Priority: 1,
	}

	mgr.AddProviderConfig(cfg)
	assert.Len(t, mgr.Get().Providers, 1)

	updated := ProviderRoutingConfig{
		Name:     "openai",
		Weight:   200,
		Priority: 2,
	}
	mgr.AddProviderConfig(updated)
	assert.Len(t, mgr.Get().Providers, 1)
	assert.Equal(t, 200, mgr.Get().Providers[0].Weight)
}

func TestConfigManager_RemoveProviderConfig(t *testing.T) {
	mgr := NewConfigManager()

	mgr.AddProviderConfig(ProviderRoutingConfig{Name: "openai"})
	mgr.AddProviderConfig(ProviderRoutingConfig{Name: "anthropic"})

	assert.Len(t, mgr.Get().Providers, 2)

	mgr.RemoveProviderConfig("openai")
	assert.Len(t, mgr.Get().Providers, 1)
	assert.Equal(t, "anthropic", mgr.Get().Providers[0].Name)
}

func TestConfigManager_Reload(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	cfg := RoutingConfig{
		DefaultStrategy: StrategyRoundRobin,
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	mgr := NewConfigManager()
	err = mgr.Load(tmpFile.Name())
	require.NoError(t, err)

	err = mgr.Reload()
	require.NoError(t, err)
}

func TestConfigManager_Reload_NoFile(t *testing.T) {
	mgr := NewConfigManager()
	err := mgr.Reload()
	assert.NoError(t, err)
}
