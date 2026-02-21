package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

type ReloadCallback func(*Config)

type ConfigWatcher struct {
	configPath string
	config     *Config
	mu         sync.RWMutex
	callbacks  []ReloadCallback
	watcher    *fsnotify.Watcher
	logger     *logrus.Logger
	stopCh     chan struct{}
}

func NewConfigWatcher(configPath string, initialConfig *Config, logger *logrus.Logger) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		absPath = configPath
	}

	cw := &ConfigWatcher{
		configPath: absPath,
		config:     initialConfig,
		watcher:    watcher,
		logger:     logger,
		stopCh:     make(chan struct{}),
		callbacks:  make([]ReloadCallback, 0),
	}

	configDir := filepath.Dir(absPath)
	if err := watcher.Add(configDir); err != nil {
		watcher.Close()
		return nil, err
	}

	go cw.watchLoop()

	return cw, nil
}

func (cw *ConfigWatcher) watchLoop() {
	for {
		select {
		case <-cw.stopCh:
			return
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if filepath.Base(event.Name) == filepath.Base(cw.configPath) {
					cw.logger.Info("Configuration file changed, reloading...")
					cw.reload()
				}
			}
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			cw.logger.WithError(err).Error("Config watcher error")
		}
	}
}

func (cw *ConfigWatcher) reload() {
	newCfg, err := loadFromFile(cw.configPath)
	if err != nil {
		cw.logger.WithError(err).Error("Failed to reload configuration")
		return
	}

	if err := newCfg.Validate(); err != nil {
		cw.logger.WithError(err).Error("Invalid configuration, keeping old config")
		return
	}

	cw.mu.Lock()
	cw.config = newCfg
	callbacks := make([]ReloadCallback, len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mu.Unlock()

	cw.logger.Info("Configuration reloaded successfully")

	for _, cb := range callbacks {
		go cb(newCfg)
	}
}

func loadFromFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	file, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(file, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (cw *ConfigWatcher) GetConfig() *Config {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.config
}

func (cw *ConfigWatcher) OnReload(callback ReloadCallback) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

func (cw *ConfigWatcher) Close() error {
	close(cw.stopCh)
	return cw.watcher.Close()
}

func (cw *ConfigWatcher) ReloadManually() error {
	cw.reload()
	return nil
}

var (
	globalWatcher *ConfigWatcher
	watcherMu     sync.RWMutex
)

func InitWatcher(configPath string, initialConfig *Config, logger *logrus.Logger) error {
	watcherMu.Lock()
	defer watcherMu.Unlock()

	if globalWatcher != nil {
		globalWatcher.Close()
	}

	watcher, err := NewConfigWatcher(configPath, initialConfig, logger)
	if err != nil {
		return err
	}

	globalWatcher = watcher
	return nil
}

func GetWatchedConfig() *Config {
	watcherMu.RLock()
	defer watcherMu.RUnlock()

	if globalWatcher == nil {
		return nil
	}
	return globalWatcher.GetConfig()
}

func OnConfigReload(callback ReloadCallback) {
	watcherMu.RLock()
	defer watcherMu.RUnlock()

	if globalWatcher != nil {
		globalWatcher.OnReload(callback)
	}
}

func CloseWatcher() {
	watcherMu.Lock()
	defer watcherMu.Unlock()

	if globalWatcher != nil {
		globalWatcher.Close()
		globalWatcher = nil
	}
}
