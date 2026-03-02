package config

import "testing"

func TestConfig_GetEditionConfig_DefaultBasic(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	edition := cfg.GetEditionConfig()

	if edition.Type != EditionBasic {
		t.Fatalf("edition type = %q, want %q", edition.Type, EditionBasic)
	}
}

func TestEditionDefinitions_EnterpriseDependencies(t *testing.T) {
	t.Parallel()

	edition := EditionDefinitions[EditionEnterprise]
	if len(edition.Dependencies) == 0 {
		t.Fatal("enterprise dependencies should not be empty")
	}

	want := []string{"redis", "ollama", "qdrant"}
	for i := range want {
		if edition.Dependencies[i] != want[i] {
			t.Fatalf("enterprise dependencies[%d] = %q, want %q", i, edition.Dependencies[i], want[i])
		}
	}
}
