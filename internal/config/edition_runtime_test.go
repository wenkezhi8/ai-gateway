package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_EditionRuntimeAndDependencyVersions(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	if cfg.Edition.Runtime != string(EditionRuntimeDocker) {
		t.Fatalf("edition.runtime = %q, want %q", cfg.Edition.Runtime, EditionRuntimeDocker)
	}
	if len(cfg.Edition.DependencyVersions) == 0 {
		t.Fatal("edition.dependency_versions should not be empty")
	}
	if cfg.Edition.DependencyVersions["redis"] == "" {
		t.Fatal("edition.dependency_versions.redis should not be empty")
	}
	if cfg.Edition.DependencyVersions["ollama"] == "" {
		t.Fatal("edition.dependency_versions.ollama should not be empty")
	}
	if cfg.Edition.DependencyVersions["qdrant"] == "" {
		t.Fatal("edition.dependency_versions.qdrant should not be empty")
	}
}

func TestLoadFromPath_EditionRuntimeDefaultsWhenMissing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	raw := `{"edition":{"type":"basic"}}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath failed: %v", err)
	}

	if cfg.Edition.Runtime != string(EditionRuntimeDocker) {
		t.Fatalf("edition.runtime = %q, want %q", cfg.Edition.Runtime, EditionRuntimeDocker)
	}
	if cfg.Edition.DependencyVersions["redis"] == "" {
		t.Fatal("redis default dependency version should not be empty")
	}
	if cfg.Edition.DependencyVersions["ollama"] == "" {
		t.Fatal("ollama default dependency version should not be empty")
	}
	if cfg.Edition.DependencyVersions["qdrant"] == "" {
		t.Fatal("qdrant default dependency version should not be empty")
	}
}

func TestUpdateEditionInFile_ShouldKeepRuntimeAndDependencyVersions(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := DefaultConfig()
	cfg.Edition.Type = string(EditionEnterprise)
	cfg.Edition.Runtime = string(EditionRuntimeNative)
	cfg.Edition.DependencyVersions = map[string]string{
		"redis":  "7.2.0-v18",
		"ollama": "0.6.0",
		"qdrant": "1.13.0",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config failed: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	updated, err := UpdateEditionInFile(configPath, EditionBasic)
	if err != nil {
		t.Fatalf("UpdateEditionInFile failed: %v", err)
	}

	if updated.Edition.Type != string(EditionBasic) {
		t.Fatalf("edition.type = %q, want %q", updated.Edition.Type, EditionBasic)
	}
	if updated.Edition.Runtime != string(EditionRuntimeNative) {
		t.Fatalf("edition.runtime = %q, want %q", updated.Edition.Runtime, EditionRuntimeNative)
	}
	if updated.Edition.DependencyVersions["redis"] != "7.2.0-v18" {
		t.Fatalf("redis version = %q, want %q", updated.Edition.DependencyVersions["redis"], "7.2.0-v18")
	}
	if updated.Edition.DependencyVersions["ollama"] != "0.6.0" {
		t.Fatalf("ollama version = %q, want %q", updated.Edition.DependencyVersions["ollama"], "0.6.0")
	}
	if updated.Edition.DependencyVersions["qdrant"] != "1.13.0" {
		t.Fatalf("qdrant version = %q, want %q", updated.Edition.DependencyVersions["qdrant"], "1.13.0")
	}
}
