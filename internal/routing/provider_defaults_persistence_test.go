package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSmartRouter_LoadProviderDefaults_ShouldReplaceNotMerge(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Errorf("restore wd: %v", chdirErr)
		}
	})

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	saved := map[string]string{
		"openai": "gpt-4o-mini",
	}
	raw, err := json.Marshal(saved)
	if err != nil {
		t.Fatalf("marshal defaults: %v", err)
	}
	if err := os.WriteFile(filepath.Join("data", "provider_defaults.json"), raw, 0o644); err != nil {
		t.Fatalf("write provider defaults: %v", err)
	}

	router := NewSmartRouter()
	defaults := router.GetProviderDefaults()

	if len(defaults) != 1 {
		t.Fatalf("expected exactly one provider default after load, got %d: %#v", len(defaults), defaults)
	}
	if defaults["openai"] != "gpt-4o-mini" {
		t.Fatalf("expected openai default to be gpt-4o-mini, got %q", defaults["openai"])
	}
	if _, ok := defaults["deepseek"]; ok {
		t.Fatalf("expected deepseek default to be absent after replace-load")
	}
}

func TestSmartRouter_SetProviderDefaults_ShouldReplaceSnapshot(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Errorf("restore wd: %v", chdirErr)
		}
	})

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	router := NewSmartRouter()
	router.SetProviderDefaults(map[string]string{
		"openai":   "gpt-4o",
		"deepseek": "deepseek-chat",
	})
	router.SetProviderDefaults(map[string]string{
		" openai ": " gpt-4o-mini ",
		"":         "ignored",
		"zhipu":    "",
	})

	defaults := router.GetProviderDefaults()
	if len(defaults) != 1 {
		t.Fatalf("expected replace semantics with one key, got %d: %#v", len(defaults), defaults)
	}
	if defaults["openai"] != "gpt-4o-mini" {
		t.Fatalf("expected trimmed openai default gpt-4o-mini, got %q", defaults["openai"])
	}
	if _, ok := defaults["deepseek"]; ok {
		t.Fatalf("expected deepseek key removed after replace")
	}
}
