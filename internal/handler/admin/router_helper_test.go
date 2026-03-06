package admin

import (
	"testing"
)

func TestGenerateID(t *testing.T) {
	id := generateID()
	if len(id) != 8 {
		t.Errorf("generateID() len = %d, want 8", len(id))
	}

	id2 := generateID()
	if id == id2 {
		t.Errorf("generateID() should generate unique IDs")
	}
}

func TestNormalizeAutoMode(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"auto", "auto", "auto"},
		{"default", "default", "default"},
		{"fixed", "fixed", "fixed"},
		{"latest", "latest", "auto"},
		{"unknown", "unknown", "auto"},
		{"empty", "", "auto"},
		{"random", "random_value", "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeAutoMode(tt.value)
			if result != tt.expected {
				t.Errorf("normalizeAutoMode(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestParseAutoModeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fallback string
		expected string
	}{
		{
			name:     "string auto",
			input:    `"auto"`,
			fallback: "default",
			expected: "auto",
		},
		{
			name:     "string fixed",
			input:    `"fixed"`,
			fallback: "auto",
			expected: "fixed",
		},
		{
			name:     "string default",
			input:    `"default"`,
			fallback: "auto",
			expected: "default",
		},
		{
			name:     "string latest",
			input:    `"latest"`,
			fallback: "auto",
			expected: "auto",
		},
		{
			name:     "boolean true",
			input:    "true",
			fallback: "fixed",
			expected: "auto",
		},
		{
			name:     "boolean false",
			input:    "false",
			fallback: "auto",
			expected: "fixed",
		},
		{
			name:     "null",
			input:    "null",
			fallback: "default",
			expected: "default",
		},
		{
			name:     "empty",
			input:    "",
			fallback: "fixed",
			expected: "fixed",
		},
		{
			name:     "invalid JSON",
			input:    "invalid",
			fallback: "auto",
			expected: "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAutoModeJSON([]byte(tt.input), tt.fallback)
			if result != tt.expected {
				t.Errorf("parseAutoModeJSON(%q, %q) = %q, want %q", tt.input, tt.fallback, result, tt.expected)
			}
		})
	}
}

func TestResolveAutoModeMigrationNotice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "latest should return notice", input: "latest", expected: true},
		{name: "auto should not return notice", input: "auto", expected: false},
		{name: "fixed should not return notice", input: "fixed", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notice := resolveAutoModeMigrationNotice(tt.input)
			if tt.expected && notice == "" {
				t.Fatalf("expected non-empty migration notice for input %q", tt.input)
			}
			if !tt.expected && notice != "" {
				t.Fatalf("expected empty migration notice for input %q, got %q", tt.input, notice)
			}
		})
	}
}

func TestBuildUseAutoModeContract(t *testing.T) {
	contract := buildUseAutoModeContract()

	if len(contract.AllowedModes) != 3 {
		t.Fatalf("expected 3 allowed modes, got %d", len(contract.AllowedModes))
	}

	if contract.AllowedModes[0] != "auto" || contract.AllowedModes[1] != "default" || contract.AllowedModes[2] != "fixed" {
		t.Fatalf("unexpected allowed modes: %#v", contract.AllowedModes)
	}

	if contract.DeprecatedMappings["latest"] != "auto" {
		t.Fatalf("expected latest => auto mapping, got %#v", contract.DeprecatedMappings)
	}

	if contract.MigrationHint == "" {
		t.Fatal("expected non-empty migration hint")
	}
}
