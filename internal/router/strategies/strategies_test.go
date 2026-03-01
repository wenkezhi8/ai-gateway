//nolint:godot
package strategies

import (
	"context"
	"testing"

	"ai-gateway/internal/provider"
	"ai-gateway/internal/router"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements provider.Provider interface for testing
type mockProvider struct {
	name    string
	enabled bool
	models  []string
}

func (m *mockProvider) Name() string                       { return m.name }
func (m *mockProvider) Models() []string                   { return m.models }
func (m *mockProvider) IsEnabled() bool                    { return m.enabled }
func (m *mockProvider) SetEnabled(enabled bool)            { m.enabled = enabled }
func (m *mockProvider) ValidateKey(_ context.Context) bool { return true }
func (m *mockProvider) Chat(_ context.Context, _ *provider.ChatRequest) (*provider.ChatResponse, error) {
	return nil, nil
}
func (m *mockProvider) StreamChat(_ context.Context, _ *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	return nil, nil
}

// Helper function to create test providers
func createTestProvider(name string, enabled, healthy bool, priority int, isPrimary bool, weight int, cost float64) *router.ProviderInfo {
	return &router.ProviderInfo{
		Provider:   &mockProvider{name: name, enabled: enabled},
		Healthy:    healthy,
		Priority:   priority,
		IsPrimary:  isPrimary,
		Weight:     weight,
		Cost:       cost,
		QuotaLimit: 1000,
		QuotaUsed:  0,
	}
}

// =====================
// Failover Strategy Tests
// =====================

func TestFailoverStrategy_New(t *testing.T) {
	s := NewFailoverStrategy()
	assert.NotNil(t, s)
	assert.Equal(t, "failover", s.Name())
}

func TestFailoverStrategy_Select_NoProviders(t *testing.T) {
	s := NewFailoverStrategy()
	providers := []*router.ProviderInfo{}
	req := &router.Request{Model: "gpt-4"}

	_, err := s.Select(providers, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no providers available")
}

func TestFailoverStrategy_Select_NoAvailableProviders(t *testing.T) {
	s := NewFailoverStrategy()
	providers := []*router.ProviderInfo{
		createTestProvider("disabled", false, true, 1, false, 10, 0.01),
		createTestProvider("unhealthy", true, false, 1, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	_, err := s.Select(providers, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available providers")
}

func TestFailoverStrategy_Select_PrimaryFirst(t *testing.T) {
	s := NewFailoverStrategy()
	providers := []*router.ProviderInfo{
		createTestProvider("backup1", true, true, 5, false, 10, 0.01),
		createTestProvider("primary", true, true, 10, true, 10, 0.02),
		createTestProvider("backup2", true, true, 3, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	selected, err := s.Select(providers, req)
	require.NoError(t, err)
	assert.Equal(t, "primary", selected.Name())
	assert.True(t, selected.IsPrimary)
}

func TestFailoverStrategy_Select_ByPriority(t *testing.T) {
	s := NewFailoverStrategy()
	providers := []*router.ProviderInfo{
		createTestProvider("low", true, true, 10, false, 10, 0.01),
		createTestProvider("high", true, true, 1, false, 10, 0.01),
		createTestProvider("medium", true, true, 5, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	selected, err := s.Select(providers, req)
	require.NoError(t, err)
	assert.Equal(t, "high", selected.Name())
	assert.Equal(t, 1, selected.Priority)
}

func TestFailoverStrategy_Select_SingleProvider(t *testing.T) {
	s := NewFailoverStrategy()
	providers := []*router.ProviderInfo{
		createTestProvider("only", true, true, 1, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	selected, err := s.Select(providers, req)
	require.NoError(t, err)
	assert.Equal(t, "only", selected.Name())
}

// =====================
// RoundRobin Strategy Tests
// =====================

func TestRoundRobinStrategy_New(t *testing.T) {
	s := NewRoundRobinStrategy()
	assert.NotNil(t, s)
	assert.Equal(t, "roundrobin", s.Name())
}

func TestRoundRobinStrategy_Select_NoProviders(t *testing.T) {
	s := NewRoundRobinStrategy()
	providers := []*router.ProviderInfo{}
	req := &router.Request{Model: "gpt-4"}

	_, err := s.Select(providers, req)
	assert.Error(t, err)
}

func TestRoundRobinStrategy_Select_Rotation(t *testing.T) {
	s := NewRoundRobinStrategy()
	s.Reset()

	providers := []*router.ProviderInfo{
		createTestProvider("a", true, true, 1, false, 10, 0.01),
		createTestProvider("b", true, true, 1, false, 10, 0.01),
		createTestProvider("c", true, true, 1, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	// First selection
	selected1, err := s.Select(providers, req)
	require.NoError(t, err)

	// Second selection
	selected2, err := s.Select(providers, req)
	require.NoError(t, err)

	// Third selection
	_, err = s.Select(providers, req)
	require.NoError(t, err)

	// Fourth selection should wrap around
	selected4, err := s.Select(providers, req)
	require.NoError(t, err)

	// Should cycle through providers
	assert.Equal(t, selected1.Name(), selected4.Name())
	assert.NotEqual(t, selected1.Name(), selected2.Name())
}

func TestRoundRobinStrategy_Select_OnlyAvailable(t *testing.T) {
	s := NewRoundRobinStrategy()
	s.Reset()

	providers := []*router.ProviderInfo{
		createTestProvider("available1", true, true, 1, false, 10, 0.01),
		createTestProvider("disabled", false, true, 1, false, 10, 0.01),
		createTestProvider("available2", true, true, 1, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	selected, err := s.Select(providers, req)
	require.NoError(t, err)
	assert.True(t, selected.Available())
	assert.NotEqual(t, "disabled", selected.Name())
}

func TestRoundRobinStrategy_Reset(t *testing.T) {
	s := NewRoundRobinStrategy()

	providers := []*router.ProviderInfo{
		createTestProvider("a", true, true, 1, false, 10, 0.01),
		createTestProvider("b", true, true, 1, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	// Make a selection
	_, err := s.Select(providers, req)
	require.NoError(t, err)

	// Reset counter
	s.Reset()

	// Counter should be 0 again
	assert.Equal(t, uint64(0), s.counter)
}

// =====================
// Cost Strategy Tests
// =====================

func TestCostStrategy_New(t *testing.T) {
	s := NewCostStrategy()
	assert.NotNil(t, s)
	assert.Equal(t, "cost", s.Name())
}

func TestCostStrategy_Select_NoProviders(t *testing.T) {
	s := NewCostStrategy()
	providers := []*router.ProviderInfo{}
	req := &router.Request{Model: "gpt-4"}

	_, err := s.Select(providers, req)
	assert.Error(t, err)
}

func TestCostStrategy_Select_LowestCost(t *testing.T) {
	s := NewCostStrategy()
	providers := []*router.ProviderInfo{
		createTestProvider("expensive", true, true, 1, false, 10, 0.05),
		createTestProvider("cheap", true, true, 1, false, 10, 0.01),
		createTestProvider("medium", true, true, 1, false, 10, 0.03),
	}
	req := &router.Request{Model: "gpt-4"}

	selected, err := s.Select(providers, req)
	require.NoError(t, err)
	assert.Equal(t, "cheap", selected.Name())
	assert.Equal(t, 0.01, selected.Cost)
}

func TestCostStrategy_Select_SameCostPreferMoreQuota(t *testing.T) {
	s := NewCostStrategy()

	provider1 := createTestProvider("less-quota", true, true, 1, false, 10, 0.01)
	provider1.QuotaUsed = 800 // Only 200 remaining

	provider2 := createTestProvider("more-quota", true, true, 1, false, 10, 0.01)
	provider2.QuotaUsed = 200 // 800 remaining

	providers := []*router.ProviderInfo{provider1, provider2}
	req := &router.Request{Model: "gpt-4"}

	selected, err := s.Select(providers, req)
	require.NoError(t, err)
	assert.Equal(t, "more-quota", selected.Name())
}

// =====================
// Weighted Strategy Tests
// =====================

func TestWeightedStrategy_New(t *testing.T) {
	s := NewWeightedStrategy()
	assert.NotNil(t, s)
	assert.Equal(t, "weighted", s.Name())
}

func TestWeightedStrategy_Select_NoProviders(t *testing.T) {
	s := NewWeightedStrategy()
	providers := []*router.ProviderInfo{}
	req := &router.Request{Model: "gpt-4"}

	_, err := s.Select(providers, req)
	assert.Error(t, err)
}

func TestWeightedStrategy_Select_NoPositiveWeight(t *testing.T) {
	s := NewWeightedStrategy()
	providers := []*router.ProviderInfo{
		createTestProvider("zero", true, true, 1, false, 0, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	_, err := s.Select(providers, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available providers with positive weight")
}

func TestWeightedStrategy_Select_SingleProvider(t *testing.T) {
	s := NewWeightedStrategy()
	providers := []*router.ProviderInfo{
		createTestProvider("only", true, true, 1, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	selected, err := s.Select(providers, req)
	require.NoError(t, err)
	assert.Equal(t, "only", selected.Name())
}

func TestWeightedStrategy_Select_Distribution(t *testing.T) {
	s := NewWeightedStrategy()

	// Create providers with different weights
	providers := []*router.ProviderInfo{
		createTestProvider("heavy", true, true, 1, false, 90, 0.01),
		createTestProvider("light", true, true, 1, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	// Run many selections to verify distribution
	counts := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		selected, err := s.Select(providers, req)
		require.NoError(t, err)
		counts[selected.Name()]++
	}

	// Heavy should be selected roughly 90% of the time
	heavyRatio := float64(counts["heavy"]) / float64(iterations)
	assert.GreaterOrEqual(t, heavyRatio, 0.85) // Allow some variance
	assert.LessOrEqual(t, heavyRatio, 0.95)
}

func TestWeightedStrategy_Select_OnlyAvailableWithWeight(t *testing.T) {
	s := NewWeightedStrategy()
	providers := []*router.ProviderInfo{
		createTestProvider("available", true, true, 1, false, 10, 0.01),
		createTestProvider("disabled", false, true, 1, false, 10, 0.01),
		createTestProvider("zero-weight", true, true, 1, false, 0, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	selected, err := s.Select(providers, req)
	require.NoError(t, err)
	assert.Equal(t, "available", selected.Name())
}

// =====================
// Concurrent Tests
// =====================

func TestRoundRobinStrategy_Concurrent(t *testing.T) {
	s := NewRoundRobinStrategy()
	s.Reset()

	providers := []*router.ProviderInfo{
		createTestProvider("a", true, true, 1, false, 10, 0.01),
		createTestProvider("b", true, true, 1, false, 10, 0.01),
		createTestProvider("c", true, true, 1, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	done := make(chan bool)
	iterations := 100

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				_, err := s.Select(providers, req)
				assert.NoError(t, err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFailoverStrategy_Concurrent(t *testing.T) {
	s := NewFailoverStrategy()

	providers := []*router.ProviderInfo{
		createTestProvider("primary", true, true, 1, true, 10, 0.01),
		createTestProvider("backup", true, true, 2, false, 10, 0.01),
	}
	req := &router.Request{Model: "gpt-4"}

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				selected, err := s.Select(providers, req)
				assert.NoError(t, err)
				assert.Equal(t, "primary", selected.Name())
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
