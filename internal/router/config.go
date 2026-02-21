package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// StrategyConfig holds configuration for a single strategy
type StrategyConfig struct {
	Name     string         `json:"name" yaml:"name"`
	Type     StrategyType   `json:"type" yaml:"type"`
	Enabled  bool           `json:"enabled" yaml:"enabled"`
	Params   map[string]interface{} `json:"params" yaml:"params"`
}

// ProviderRoutingConfig holds routing configuration for a provider
type ProviderRoutingConfig struct {
	Name     string  `json:"name" yaml:"name"`
	Weight   int     `json:"weight" yaml:"weight"`
	Priority int     `json:"priority" yaml:"priority"`
	Cost     float64 `json:"cost" yaml:"cost"`
	IsPrimary bool   `json:"is_primary" yaml:"is_primary"`
}

// RoutingConfig holds all routing configuration
type RoutingConfig struct {
	DefaultStrategy StrategyType              `json:"default_strategy" yaml:"default_strategy"`
	Strategies      []StrategyConfig          `json:"strategies" yaml:"strategies"`
	Providers       []ProviderRoutingConfig   `json:"providers" yaml:"providers"`
	Rules           []RoutingRule             `json:"rules" yaml:"rules"`
}

// RoutingRule defines routing rules based on conditions
type RoutingRule struct {
	Name       string       `json:"name" yaml:"name"`
	Condition  RuleCondition `json:"condition" yaml:"condition"`
	Strategy   StrategyType `json:"strategy" yaml:"strategy"`
	Providers  []string     `json:"providers" yaml:"providers"`
	Priority   int          `json:"priority" yaml:"priority"`
}

// RuleCondition defines conditions for routing rules
type RuleCondition struct {
	Model    string            `json:"model" yaml:"model"`
	UserID   string            `json:"user_id" yaml:"user_id"`
	Headers  map[string]string `json:"headers" yaml:"headers"`
	Extra    map[string]interface{} `json:"extra" yaml:"extra"`
}

// ConfigManager manages routing configuration
type ConfigManager struct {
	config    *RoutingConfig
	configPath string
	mu        sync.RWMutex
	onChange  []func(*RoutingConfig)
}

// NewConfigManager creates a new config manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: &RoutingConfig{
			DefaultStrategy: StrategyRoundRobin,
			Strategies:      []StrategyConfig{},
			Providers:       []ProviderRoutingConfig{},
			Rules:           []RoutingRule{},
		},
		onChange: []func(*RoutingConfig){},
	}
}

// Load loads configuration from file
func (m *ConfigManager) Load(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}

	m.configPath = path

	// Determine file type by extension
	ext := filepath.Ext(path)
	var cfg RoutingConfig

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return err
		}
	default: // default to JSON
		if err := json.Unmarshal(data, &cfg); err != nil {
			return err
		}
	}

	m.config = &cfg
	return nil
}

// Save saves configuration to file
func (m *ConfigManager) Save(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var data []byte
	var err error

	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(m.config)
	default:
		data, err = json.MarshalIndent(m.config, "", "  ")
	}

	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Get returns the current configuration
func (m *ConfigManager) Get() *RoutingConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Set updates the configuration
func (m *ConfigManager) Set(cfg *RoutingConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = cfg

	// Notify listeners
	for _, fn := range m.onChange {
		fn(cfg)
	}
}

// SetDefaultStrategy sets the default strategy
func (m *ConfigManager) SetDefaultStrategy(strategy StrategyType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.DefaultStrategy = strategy
}

// AddProviderConfig adds or updates a provider configuration
func (m *ConfigManager) AddProviderConfig(cfg ProviderRoutingConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and update existing or append new
	for i, p := range m.config.Providers {
		if p.Name == cfg.Name {
			m.config.Providers[i] = cfg
			return
		}
	}
	m.config.Providers = append(m.config.Providers, cfg)
}

// RemoveProviderConfig removes a provider configuration
func (m *ConfigManager) RemoveProviderConfig(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.config.Providers {
		if p.Name == name {
			m.config.Providers = append(m.config.Providers[:i], m.config.Providers[i+1:]...)
			return
		}
	}
}

// OnChange registers a callback for configuration changes
func (m *ConfigManager) OnChange(fn func(*RoutingConfig)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = append(m.onChange, fn)
}

// Reload reloads configuration from the last loaded file
func (m *ConfigManager) Reload() error {
	if m.configPath == "" {
		return nil
	}
	return m.Load(m.configPath)
}
