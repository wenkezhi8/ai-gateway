package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfgContent := `{"server": {"port": "8080"}}`
	err := os.WriteFile(configPath, []byte(cfgContent), 0644)
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := DefaultConfig()
	watcher, err := NewConfigWatcher(configPath, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, watcher)

	defer watcher.Close()

	assert.Equal(t, cfg, watcher.GetConfig())
}

func TestConfigWatcher_OnReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfgContent := `{
		"server": {"port": "8080"},
		"providers": [
			{"name": "openai", "enabled": true}
		]
	}`
	err := os.WriteFile(configPath, []byte(cfgContent), 0644)
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := DefaultConfig()
	watcher, err := NewConfigWatcher(configPath, cfg, logger)
	require.NoError(t, err)
	defer watcher.Close()

	reloadCalled := make(chan struct{}, 1)
	watcher.OnReload(func(_ *Config) {
		select {
		case reloadCalled <- struct{}{}:
		default:
		}
	})

	err = watcher.ReloadManually()
	require.NoError(t, err)

	select {
	case <-reloadCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("expected reload callback to be called")
	}
}

func TestConfigWatcher_ReloadManually(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfgContent := `{"server": {"port": "8080"}}`
	err := os.WriteFile(configPath, []byte(cfgContent), 0644)
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := DefaultConfig()
	watcher, err := NewConfigWatcher(configPath, cfg, logger)
	require.NoError(t, err)
	defer watcher.Close()

	err = watcher.ReloadManually()
	assert.NoError(t, err)
}

func TestConfigWatcher_GetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfgContent := `{"server": {"port": "9090"}}`
	err := os.WriteFile(configPath, []byte(cfgContent), 0644)
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := DefaultConfig()
	watcher, err := NewConfigWatcher(configPath, cfg, logger)
	require.NoError(t, err)
	defer watcher.Close()

	retrieved := watcher.GetConfig()
	assert.NotNil(t, retrieved)
}

func TestLoadFromFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	cfgContent := `{
		"server": {"port": "9999"},
		"redis": {"host": "localhost"}
	}`
	_, err = tmpFile.WriteString(cfgContent)
	require.NoError(t, err)
	tmpFile.Close()

	cfg, err := loadFromFile(tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "9999", cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Redis.Host)
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := loadFromFile("/nonexistent/config.json")
	assert.Error(t, err)
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(`{invalid json}`)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = loadFromFile(tmpFile.Name())
	assert.Error(t, err)
}

func TestGlobalWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfgContent := `{"server": {"port": "8080"}}`
	err := os.WriteFile(configPath, []byte(cfgContent), 0644)
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := DefaultConfig()

	err = InitWatcher(configPath, cfg, logger)
	require.NoError(t, err)
	defer CloseWatcher()

	retrieved := GetWatchedConfig()
	assert.NotNil(t, retrieved)

	OnConfigReload(func(_ *Config) {})
}
