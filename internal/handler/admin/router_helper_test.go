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
		{"latest", "latest", "latest"},
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
			expected: "latest",
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
