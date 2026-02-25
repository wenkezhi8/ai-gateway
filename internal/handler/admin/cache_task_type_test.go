package admin

import "testing"

func TestNormalizeTaskType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "other alias", input: "other", expected: "unknown"},
		{name: "long context alias", input: "long_context", expected: "long_text"},
		{name: "keep math", input: "math", expected: "math"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTaskType(tt.input); got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestResolveTaskType(t *testing.T) {
	tests := []struct {
		name            string
		metaTaskType    string
		payloadTaskType string
		userMsg         string
		expected        string
	}{
		{
			name:            "prefer non-unknown meta",
			metaTaskType:    "code",
			payloadTaskType: "math",
			userMsg:         "1+1",
			expected:        "code",
		},
		{
			name:            "fallback to payload alias",
			metaTaskType:    "",
			payloadTaskType: "long_context",
			userMsg:         "",
			expected:        "long_text",
		},
		{
			name:            "infer from message when unknown",
			metaTaskType:    "unknown",
			payloadTaskType: "",
			userMsg:         "1+1",
			expected:        "math",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveTaskType(tt.metaTaskType, tt.payloadTaskType, tt.userMsg); got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}
