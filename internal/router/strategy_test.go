//nolint:godot
package router

import (
	"context"
	"testing"

	"ai-gateway/internal/provider"

	"github.com/stretchr/testify/assert"
)

// mockProvider implements provider.Provider for testing
type mockProvider struct {
	name    string
	models  []string
	enabled bool
}

func newMockProvider(models []string, enabled bool) *mockProvider {
	return &mockProvider{
		name:    "test",
		models:  models,
		enabled: enabled,
	}
}

func (m *mockProvider) Name() string            { return m.name }
func (m *mockProvider) Models() []string        { return m.models }
func (m *mockProvider) IsEnabled() bool         { return m.enabled }
func (m *mockProvider) SetEnabled(enabled bool) { m.enabled = enabled }
func (m *mockProvider) Chat(_ context.Context, _ *provider.ChatRequest) (*provider.ChatResponse, error) {
	return nil, nil
}
func (m *mockProvider) StreamChat(_ context.Context, _ *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	return nil, nil
}
func (m *mockProvider) ValidateKey(_ context.Context) bool { return true }

func TestProviderInfo_Available(t *testing.T) {
	tests := []struct {
		name     string
		info     *ProviderInfo
		expected bool
	}{
		{
			name: "fully available",
			info: &ProviderInfo{
				Provider:   newMockProvider([]string{"gpt-4"}, true),
				Healthy:    true,
				QuotaLimit: 0, // unlimited
			},
			expected: true,
		},
		{
			name: "disabled provider",
			info: &ProviderInfo{
				Provider:   newMockProvider([]string{"gpt-4"}, false),
				Healthy:    true,
				QuotaLimit: 0,
			},
			expected: false,
		},
		{
			name: "unhealthy",
			info: &ProviderInfo{
				Provider:   newMockProvider([]string{"gpt-4"}, true),
				Healthy:    false,
				QuotaLimit: 0,
			},
			expected: false,
		},
		{
			name: "quota exceeded",
			info: &ProviderInfo{
				Provider:   newMockProvider([]string{"gpt-4"}, true),
				Healthy:    true,
				QuotaLimit: 100,
				QuotaUsed:  100,
			},
			expected: false,
		},
		{
			name: "quota available",
			info: &ProviderInfo{
				Provider:   newMockProvider([]string{"gpt-4"}, true),
				Healthy:    true,
				QuotaLimit: 100,
				QuotaUsed:  50,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.Available()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderInfo_QuotaRemaining(t *testing.T) {
	tests := []struct {
		name     string
		info     *ProviderInfo
		expected int64
	}{
		{
			name: "unlimited quota",
			info: &ProviderInfo{
				QuotaLimit: 0,
			},
			expected: -1,
		},
		{
			name: "half remaining",
			info: &ProviderInfo{
				QuotaLimit: 100,
				QuotaUsed:  50,
			},
			expected: 50,
		},
		{
			name: "fully used",
			info: &ProviderInfo{
				QuotaLimit: 100,
				QuotaUsed:  100,
			},
			expected: 0,
		},
		{
			name: "overused",
			info: &ProviderInfo{
				QuotaLimit: 100,
				QuotaUsed:  150,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.QuotaRemaining()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseStrategyType(t *testing.T) {
	tests := []struct {
		input    string
		expected StrategyType
	}{
		{"failover", StrategyFailover},
		{"roundrobin", StrategyRoundRobin},
		{"cost", StrategyCostOptimized},
		{"weighted", StrategyWeighted},
		{"unknown", StrategyRoundRobin}, // default
		{"", StrategyRoundRobin},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseStrategyType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequest_Fields(t *testing.T) {
	req := Request{
		Model:      "gpt-4",
		UserID:     "user-123",
		TokensUsed: 100,
		Priority:   1,
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	assert.Equal(t, "gpt-4", req.Model)
	assert.Equal(t, "user-123", req.UserID)
	assert.Equal(t, 100, req.TokensUsed)
	assert.Equal(t, 1, req.Priority)
	assert.Len(t, req.Messages, 1)
}

func TestMessage_Fields(t *testing.T) {
	msg := Message{
		Role:    "system",
		Content: "You are a helpful assistant.",
	}

	assert.Equal(t, "system", msg.Role)
	assert.Equal(t, "You are a helpful assistant.", msg.Content)
}

func TestProviderInfo_Fields(t *testing.T) {
	info := &ProviderInfo{
		Provider:   newMockProvider([]string{"gpt-4"}, true),
		Weight:     10,
		Priority:   5,
		Cost:       0.02,
		IsPrimary:  true,
		Healthy:    true,
		QuotaUsed:  500,
		QuotaLimit: 1000,
	}

	assert.Equal(t, 10, info.Weight)
	assert.Equal(t, 5, info.Priority)
	assert.Equal(t, 0.02, info.Cost)
	assert.True(t, info.IsPrimary)
	assert.True(t, info.Available())
}

func TestStrategyType_Constants(t *testing.T) {
	assert.Equal(t, StrategyType("failover"), StrategyFailover)
	assert.Equal(t, StrategyType("roundrobin"), StrategyRoundRobin)
	assert.Equal(t, StrategyType("cost"), StrategyCostOptimized)
	assert.Equal(t, StrategyType("weighted"), StrategyWeighted)
}
