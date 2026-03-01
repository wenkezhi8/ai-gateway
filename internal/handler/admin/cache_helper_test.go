package admin

import (
	"testing"

	"time"

	"ai-gateway/internal/cache"
)

func TestExtractAIFromBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "valid body with content",
			body:     `{"choices":[{"message":{"content":"Hello world"}}]}`,
			expected: "Hello world",
		},
		{
			name:     "empty body",
			body:     `{}`,
			expected: "",
		},
		{
			name:     "invalid JSON",
			body:     `invalid`,
			expected: "",
		},
		{
			name:     "no choices",
			body:     `{"model":"gpt-4"}`,
			expected: "",
		},
		{
			name:     "empty choices",
			body:     `{"choices":[]}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAIFromBody([]byte(tt.body))
			if result != tt.expected {
				t.Errorf("extractAIFromBody() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractAIFromAny(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string value",
			value:    `{"choices":[{"message":{"content":"test"}}]}`,
			expected: "test",
		},
		{
			name:     "bytes value",
			value:    []byte(`{"choices":[{"message":{"content":"bytes test"}}]}`),
			expected: "bytes test",
		},
		{
			name:     "nil value",
			value:    nil,
			expected: "",
		},
		{
			name:     "map value",
			value:    map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"content": "map test"}}}},
			expected: "map test",
		},
		{
			name:     "empty string",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAIFromAny(tt.value)
			if result != tt.expected {
				t.Errorf("extractAIFromAny() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTruncatePreview(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short string",
			input:    "Hello",
			expected: "Hello",
		},
		{
			name:     "long string",
			input:    "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore",
			expected: "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore",
		},
		{
			name:     "exactly 120 chars",
			input:    string(make([]byte, 120)),
			expected: string(make([]byte, 120)),
		},
		{
			name:     "over 120 chars",
			input:    string(make([]byte, 150)),
			expected: string(make([]byte, 120)) + "...",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncatePreview(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("truncatePreview() len = %d, want %d", len(result), len(tt.expected))
			}
		})
	}
}

func TestExtractModelProvider(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		wantModel    string
		wantProvider string
	}{
		{
			name:         "with model and provider",
			value:        map[string]interface{}{"model": "gpt-4", "provider": "openai"},
			wantModel:    "gpt-4",
			wantProvider: "openai",
		},
		{
			name:         "with Model and Provider (capitalized)",
			value:        map[string]interface{}{"Model": "claude-3", "Provider": "anthropic"},
			wantModel:    "claude-3",
			wantProvider: "anthropic",
		},
		{
			name:         "empty map",
			value:        map[string]interface{}{},
			wantModel:    "",
			wantProvider: "",
		},
		{
			name:         "nil value",
			value:        nil,
			wantModel:    "",
			wantProvider: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, provider := extractModelProvider(tt.value)
			if model != tt.wantModel {
				t.Errorf("extractModelProvider() model = %q, want %q", model, tt.wantModel)
			}
			if provider != tt.wantProvider {
				t.Errorf("extractModelProvider() provider = %q, want %q", provider, tt.wantProvider)
			}
		})
	}
}

func TestExtractModelFromBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "valid model",
			body:     `{"model":"gpt-4"}`,
			expected: "gpt-4",
		},
		{
			name:     "no model",
			body:     `{"prompt":"hello"}`,
			expected: "",
		},
		{
			name:     "invalid JSON",
			body:     `invalid`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractModelFromBody([]byte(tt.body))
			if result != tt.expected {
				t.Errorf("extractModelFromBody() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractHitCountFromValue(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		wantCount int
		wantOk    bool
	}{
		{
			name:      "with hit_count",
			value:     map[string]interface{}{"hit_count": 5},
			wantCount: 5,
			wantOk:    true,
		},
		{
			name:      "with HitCount",
			value:     map[string]interface{}{"HitCount": 10},
			wantCount: 10,
			wantOk:    true,
		},
		{
			name:      "no hit count",
			value:     map[string]interface{}{"model": "gpt-4"},
			wantCount: 0,
			wantOk:    false,
		},
		{
			name:      "nil value",
			value:     nil,
			wantCount: 0,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, ok := extractHitCountFromValue(tt.value)
			if count != tt.wantCount {
				t.Errorf("extractHitCountFromValue() count = %d, want %d", count, tt.wantCount)
			}
			if ok != tt.wantOk {
				t.Errorf("extractHitCountFromValue() ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

func TestNumberToInt(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected int
		wantOk   bool
	}{
		{"int value", 42, 42, true},
		{"int64 value", int64(42), 42, true},
		{"float64 value", float64(42.5), 42, true},
		{"string value", "42", 0, false},
		{"nil value", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := numberToInt(tt.value)
			if result != tt.expected {
				t.Errorf("numberToInt() = %d, want %d", result, tt.expected)
			}
			if ok != tt.wantOk {
				t.Errorf("numberToInt() ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

func TestFilterReadableEntries(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour)

	tests := []struct {
		name     string
		entries  []*cache.CacheEntryInfo
		expected int
	}{
		{
			name:     "nil entries",
			entries:  nil,
			expected: 0,
		},
		{
			name:     "empty entries",
			entries:  []*cache.CacheEntryInfo{},
			expected: 0,
		},
		{
			name: "valid entry",
			entries: []*cache.CacheEntryInfo{
				{
					Key:       "test-key",
					Model:     "gpt-4",
					Provider:  "openai",
					CreatedAt: now,
					ExpiresAt: &expiresAt,
				},
			},
			expected: 1,
		},
		{
			name: "invalid entry - no model",
			entries: []*cache.CacheEntryInfo{
				{
					Key:      "test-key",
					Provider: "openai",
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterReadableEntries(tt.entries)
			if len(result) != tt.expected {
				t.Errorf("filterReadableEntries() len = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestIsInvalidEntry(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(-time.Hour)

	tests := []struct {
		name     string
		entry    *cache.CacheEntryInfo
		expected bool
	}{
		{
			name:     "nil entry",
			entry:    nil,
			expected: true,
		},
		{
			name: "valid entry",
			entry: &cache.CacheEntryInfo{
				Key:       "test",
				Model:     "gpt-4",
				Provider:  "openai",
				CreatedAt: now,
				ExpiresAt: &expiresAt,
			},
			expected: false,
		},
		{
			name: "no model",
			entry: &cache.CacheEntryInfo{
				Key:      "test",
				Provider: "openai",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInvalidEntry(tt.entry)
			if result != tt.expected {
				t.Errorf("isInvalidEntry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEnrichEntryFromDetail(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour)

	t.Run("nil entry", func(_ *testing.T) {
		enrichEntryFromDetail(nil, &cache.CacheEntryDetail{})
	})

	t.Run("nil detail", func(_ *testing.T) {
		entry := &cache.CacheEntryInfo{}
		enrichEntryFromDetail(entry, nil)
	})

	t.Run("with detail data", func(_ *testing.T) {
		entry := &cache.CacheEntryInfo{}
		detail := &cache.CacheEntryDetail{
			Hits:      10,
			TTL:       3600,
			ExpiresAt: &expiresAt,
			Value: map[string]interface{}{
				"model":     "gpt-4",
				"provider":  "openai",
				"hit_count": 5,
			},
		}
		enrichEntryFromDetail(entry, detail)
	})
}

func TestExtractHitModels(t *testing.T) {
	tests := []struct {
		name   string
		value  interface{}
		wantOk bool
	}{
		{
			name: "valid hit_models",
			value: map[string]interface{}{
				"hit_models": map[string]interface{}{"gpt-4": 5, "gpt-3.5": 3},
			},
			wantOk: true,
		},
		{
			name: "valid HitModels",
			value: map[string]interface{}{
				"HitModels": map[string]interface{}{"claude-3": 2},
			},
			wantOk: true,
		},
		{
			name: "no hit_models",
			value: map[string]interface{}{
				"model": "gpt-4",
			},
			wantOk: false,
		},
		{
			name:   "nil value",
			value:  nil,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := extractHitModels(tt.value)
			if ok != tt.wantOk {
				t.Errorf("extractHitModels() ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

func TestParseHitModels(t *testing.T) {
	tests := []struct {
		name   string
		value  interface{}
		wantOk bool
	}{
		{
			name: "valid map[string]interface{}",
			value: map[string]interface{}{
				"gpt-4": 5, "gpt-3.5": 3,
			},
			wantOk: true,
		},
		{
			name:   "empty map",
			value:  map[string]interface{}{},
			wantOk: false,
		},
		{
			name:   "invalid value type",
			value:  "invalid",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := parseHitModels(tt.value)
			if ok != tt.wantOk {
				t.Errorf("parseHitModels() ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

func TestSelectPrimaryModel(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		result := selectPrimaryModel(map[string]int{})
		if result != "-" {
			t.Errorf("selectPrimaryModel() = %q, want -", result)
		}
	})

	t.Run("single model", func(t *testing.T) {
		result := selectPrimaryModel(map[string]int{"gpt-4": 5})
		if result != "gpt-4" {
			t.Errorf("selectPrimaryModel() = %q, want gpt-4", result)
		}
	})

	t.Run("multiple models", func(t *testing.T) {
		result := selectPrimaryModel(map[string]int{"gpt-4": 10, "gpt-3.5": 5})
		expected := "gpt-4 等2个"
		if result != expected {
			t.Errorf("selectPrimaryModel() = %q, want %q", result, expected)
		}
	})
}

func TestAggregateCacheEntries(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour)

	entries := []*cache.CacheEntryInfo{
		{
			Key:       "key1",
			Model:     "gpt-4",
			Provider:  "openai",
			TaskType:  "code",
			Hits:      5,
			CreatedAt: now,
			ExpiresAt: &expiresAt,
		},
		{
			Key:       "key2",
			Model:     "gpt-4",
			Provider:  "openai",
			TaskType:  "code",
			Hits:      3,
			CreatedAt: now,
			ExpiresAt: &expiresAt,
		},
		{
			Key:       "key3",
			Model:     "gpt-3.5",
			Provider:  "openai",
			TaskType:  "text",
			Hits:      2,
			CreatedAt: now,
			ExpiresAt: &expiresAt,
		},
	}

	result := aggregateCacheEntries(entries)
	if len(result) == 0 {
		t.Error("aggregateCacheEntries() returned empty result")
	}
}

func TestMergeModelStats(t *testing.T) {
	t.Run("merge two stats", func(t *testing.T) {
		stats1 := map[string]int{"gpt-4": 5}
		stats2 := map[string]int{"gpt-4": 3, "gpt-3.5": 2}

		result := mergeModelStats(stats1, stats2, "gpt-4")
		if result["gpt-4"] != 8 {
			t.Errorf("mergeModelStats() gpt-4 count = %d, want 8", result["gpt-4"])
		}
	})

	t.Run("nil stats", func(t *testing.T) {
		result := mergeModelStats(nil, nil, "gpt-4")
		if result == nil {
			t.Error("mergeModelStats() should return empty map for nil inputs")
		}
	})
}

func TestCacheEntryInfoValueField(t *testing.T) {
	entry := &cache.CacheEntryInfo{
		Value: map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"content": "test response",
					},
				},
			},
		},
	}

	if entry.Value == nil {
		t.Error("CacheEntryInfo.Value should be populated")
	}

	valueMap, ok := entry.Value.(map[string]interface{})
	if !ok {
		t.Error("CacheEntryInfo.Value should be a map")
	}

	choices, ok := valueMap["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Error("CacheEntryInfo.Value should contain choices array")
	}
}
