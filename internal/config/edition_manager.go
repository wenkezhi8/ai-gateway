package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveConfigPath() string {
	configPath := strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if configPath == "" {
		configPath = "./configs/config.json"
	}
	return configPath
}

func LoadFromPath(configPath string) (*Config, error) {
	cfg := DefaultConfig()
	file, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(file, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveToPath(configPath string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(configPath), data, 0o640)
}

func IsValidEditionType(editionType EditionType) bool {
	_, ok := EditionDefinitions[editionType]
	return ok
}

func UpdateEditionInFile(configPath string, editionType EditionType) (*Config, error) {
	if !IsValidEditionType(editionType) {
		return nil, fmt.Errorf("invalid edition type: %s", editionType)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		return nil, err
	}

	cfg.Edition.Type = string(editionType)
	def := EditionDefinitions[editionType]

	cfg.VectorCache.Enabled = def.Features.VectorCache
	if !def.Features.ColdHotTiering {
		cfg.VectorCache.ColdVectorEnabled = false
		cfg.VectorCache.ColdVectorDualWriteEnabled = false
	}

	if err := SaveToPath(configPath, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
