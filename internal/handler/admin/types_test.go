package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsageLogResponse_Fields(t *testing.T) {
	resp := UsageLogResponse{
		ID:           1,
		Timestamp:    1234567890,
		Model:        "gpt-4",
		Provider:     "openai",
		UserID:       "user-1",
		APIKey:       "sk-test",
		Tokens:       1000,
		InputTokens:  500,
		OutputTokens: 500,
		LatencyMs:    1000,
		TTFTMs:       200,
		CacheHit:     true,
		Success:      true,
		ErrorType:    "",
		TaskType:     "chat",
		Difficulty:   "medium",
		CreatedAt:    "2024-01-01T00:00:00Z",
	}

	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, int64(1234567890), resp.Timestamp)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Equal(t, "openai", resp.Provider)
	assert.Equal(t, "user-1", resp.UserID)
	assert.Equal(t, "sk-test", resp.APIKey)
	assert.Equal(t, int64(1000), resp.Tokens)
	assert.Equal(t, int64(500), resp.InputTokens)
	assert.Equal(t, int64(500), resp.OutputTokens)
	assert.Equal(t, int64(1000), resp.LatencyMs)
	assert.Equal(t, int64(200), resp.TTFTMs)
	assert.True(t, resp.CacheHit)
	assert.True(t, resp.Success)
	assert.Equal(t, "", resp.ErrorType)
	assert.Equal(t, "chat", resp.TaskType)
	assert.Equal(t, "medium", resp.Difficulty)
	assert.Equal(t, "2024-01-01T00:00:00Z", resp.CreatedAt)
}

func TestUsageStatsResponse_Fields(t *testing.T) {
	stats := UsageStatsResponse{
		TotalRequests: 1000,
		TotalTokens:   500000,
		CacheHits:     400,
		CacheMisses:   600,
		CacheHitRate:  40.0,
		AvgLatencyMs:  1500,
		ModelStats: map[string]ModelUsage{
			"gpt-4":         {Requests: 500, Tokens: 300000},
			"gpt-3.5-turbo": {Requests: 500, Tokens: 200000},
		},
	}

	assert.Equal(t, int64(1000), stats.TotalRequests)
	assert.Equal(t, int64(500000), stats.TotalTokens)
	assert.Equal(t, int64(400), stats.CacheHits)
	assert.Equal(t, int64(600), stats.CacheMisses)
	assert.Equal(t, 40.0, stats.CacheHitRate)
	assert.Equal(t, int64(1500), stats.AvgLatencyMs)
	assert.Len(t, stats.ModelStats, 2)
}

func TestModelUsage_Fields(t *testing.T) {
	usage := ModelUsage{
		Requests: 100,
		Tokens:   50000,
	}

	assert.Equal(t, int64(100), usage.Requests)
	assert.Equal(t, int64(50000), usage.Tokens)
}

func TestGetInt64(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected int64
	}{
		{
			name:     "int64 value",
			m:        map[string]interface{}{"key": int64(100)},
			key:      "key",
			expected: 100,
		},
		{
			name:     "int value",
			m:        map[string]interface{}{"key": 100},
			key:      "key",
			expected: 100,
		},
		{
			name:     "float64 value",
			m:        map[string]interface{}{"key": float64(100.5)},
			key:      "key",
			expected: 100,
		},
		{
			name:     "missing key",
			m:        map[string]interface{}{},
			key:      "key",
			expected: 0,
		},
		{
			name:     "invalid type",
			m:        map[string]interface{}{"key": "string"},
			key:      "key",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInt64(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "string value",
			m:        map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "missing key",
			m:        map[string]interface{}{},
			key:      "key",
			expected: "",
		},
		{
			name:     "invalid type",
			m:        map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected bool
	}{
		{
			name:     "true value",
			m:        map[string]interface{}{"key": true},
			key:      "key",
			expected: true,
		},
		{
			name:     "false value",
			m:        map[string]interface{}{"key": false},
			key:      "key",
			expected: false,
		},
		{
			name:     "missing key",
			m:        map[string]interface{}{},
			key:      "key",
			expected: false,
		},
		{
			name:     "invalid type",
			m:        map[string]interface{}{"key": "true"},
			key:      "key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBool(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInt64FromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected int64
	}{
		{
			name:     "int64 value",
			m:        map[string]interface{}{"key": int64(100)},
			key:      "key",
			expected: 100,
		},
		{
			name:     "int value",
			m:        map[string]interface{}{"key": 100},
			key:      "key",
			expected: 100,
		},
		{
			name:     "float64 value",
			m:        map[string]interface{}{"key": float64(100.5)},
			key:      "key",
			expected: 100,
		},
		{
			name:     "missing key",
			m:        map[string]interface{}{},
			key:      "key",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInt64FromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFloat64FromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected float64
	}{
		{
			name:     "float64 value",
			m:        map[string]interface{}{"key": float64(100.5)},
			key:      "key",
			expected: 100.5,
		},
		{
			name:     "int64 value",
			m:        map[string]interface{}{"key": int64(100)},
			key:      "key",
			expected: 100.0,
		},
		{
			name:     "int value",
			m:        map[string]interface{}{"key": 100},
			key:      "key",
			expected: 100.0,
		},
		{
			name:     "missing key",
			m:        map[string]interface{}{},
			key:      "key",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFloat64FromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
