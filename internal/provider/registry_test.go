//nolint:godot
package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements Provider interface for testing
type mockProvider struct {
	name    string
	models  []string
	enabled bool
}

func (m *mockProvider) Name() string                       { return m.name }
func (m *mockProvider) Models() []string                   { return m.models }
func (m *mockProvider) IsEnabled() bool                    { return m.enabled }
func (m *mockProvider) SetEnabled(enabled bool)            { m.enabled = enabled }
func (m *mockProvider) ValidateKey(_ context.Context) bool { return true }
func (m *mockProvider) Chat(_ context.Context, _ *ChatRequest) (*ChatResponse, error) {
	return nil, nil
}
func (m *mockProvider) StreamChat(_ context.Context, _ *ChatRequest) (<-chan *StreamChunk, error) {
	return nil, nil
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)
	assert.NotNil(t, r.providers)
	assert.NotNil(t, r.factories)
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "test", models: []string{"model-1"}, enabled: true}

	r.Register("test", p)

	retrieved, ok := r.Get("test")
	require.True(t, ok)
	assert.Equal(t, "test", retrieved.Name())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get("non-existent")
	assert.False(t, ok)
}

func TestRegistry_Remove(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "test", enabled: true}

	r.Register("test", p)
	r.Remove("test")

	_, ok := r.Get("test")
	assert.False(t, ok)
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	p1 := &mockProvider{name: "provider1", enabled: true}
	p2 := &mockProvider{name: "provider2", enabled: true}

	r.Register("provider1", p1)
	r.Register("provider2", p2)

	providers := r.List()
	assert.Len(t, providers, 2)
}

func TestRegistry_ListEnabled(t *testing.T) {
	r := NewRegistry()

	p1 := &mockProvider{name: "enabled", enabled: true}
	p2 := &mockProvider{name: "disabled", enabled: false}

	r.Register("enabled", p1)
	r.Register("disabled", p2)

	providers := r.ListEnabled()
	assert.Len(t, providers, 1)
	assert.Equal(t, "enabled", providers[0].Name())
}

func TestRegistry_GetByModel(t *testing.T) {
	r := NewRegistry()

	p1 := &mockProvider{name: "gpt-provider", models: []string{"gpt-4", "gpt-3.5-turbo"}, enabled: true}
	p2 := &mockProvider{name: "claude-provider", models: []string{"claude-3"}, enabled: true}

	r.Register("gpt", p1)
	r.Register("claude", p2)

	// Find by model
	provider, ok := r.GetByModel("gpt-4")
	require.True(t, ok)
	assert.Equal(t, "gpt-provider", provider.Name())

	provider, ok = r.GetByModel("claude-3")
	require.True(t, ok)
	assert.Equal(t, "claude-provider", provider.Name())

	// Model not found
	_, ok = r.GetByModel("unknown-model")
	assert.False(t, ok)
}

func TestRegistry_GetByModel_DisabledProvider(t *testing.T) {
	r := NewRegistry()

	p := &mockProvider{name: "disabled", models: []string{"model-1"}, enabled: false}
	r.Register("disabled", p)

	// Should not find model in disabled provider
	_, ok := r.GetByModel("model-1")
	assert.False(t, ok)
}

func TestRegistry_RegisterFactory(t *testing.T) {
	r := NewRegistry()

	factory := func(cfg *ProviderConfig) Provider {
		return &mockProvider{name: cfg.Name, enabled: cfg.Enabled}
	}

	r.RegisterFactory("custom", factory)

	names := r.GetFactoryNames()
	assert.Contains(t, names, "custom")
}

func TestRegistry_CreateProvider(t *testing.T) {
	r := NewRegistry()

	factory := func(cfg *ProviderConfig) Provider {
		return &mockProvider{name: cfg.Name, models: cfg.Models, enabled: cfg.Enabled}
	}
	r.RegisterFactory("custom", factory)

	cfg := &ProviderConfig{
		Name:    "custom",
		Models:  []string{"model-1"},
		Enabled: true,
	}

	provider, err := r.CreateProvider(cfg)
	require.NoError(t, err)
	assert.Equal(t, "custom", provider.Name())
}

func TestRegistry_CreateProvider_UnknownFactory(t *testing.T) {
	r := NewRegistry()

	cfg := &ProviderConfig{Name: "unknown"}
	_, err := r.CreateProvider(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestRegistry_CreateAndRegister(t *testing.T) {
	r := NewRegistry()

	factory := func(cfg *ProviderConfig) Provider {
		return &mockProvider{name: cfg.Name, enabled: cfg.Enabled}
	}
	r.RegisterFactory("test", factory)

	cfg := &ProviderConfig{Name: "test", Enabled: true}

	provider, err := r.CreateAndRegister(cfg)
	require.NoError(t, err)
	assert.Equal(t, "test", provider.Name())

	// Verify it's registered
	retrieved, ok := r.Get("test")
	require.True(t, ok)
	assert.Equal(t, provider, retrieved)
}

func TestRegistry_GetFactoryNames(t *testing.T) {
	r := NewRegistry()

	// Register some factories
	factory := func(cfg *ProviderConfig) Provider {
		return &mockProvider{name: cfg.Name, enabled: cfg.Enabled}
	}
	r.RegisterFactory("test1", factory)
	r.RegisterFactory("test2", factory)

	names := r.GetFactoryNames()
	assert.Contains(t, names, "test1")
	assert.Contains(t, names, "test2")
}

func TestRegistry_Concurrent(t *testing.T) {
	_ = t
	r := NewRegistry()

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(_ int) {
			p := &mockProvider{name: "concurrent", enabled: true}
			r.Register("provider", p)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = r.List()
			_ = r.ListEnabled()
			_, _ = r.Get("provider")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestGetRegistry(t *testing.T) {
	r1 := GetRegistry()
	r2 := GetRegistry()

	// Should return the same instance
	assert.Equal(t, r1, r2)
}

func TestRegisterProvider(t *testing.T) {
	// Clear global registry before test
	ClearRegistry()
	defer ClearRegistry()

	p := &mockProvider{name: "global-test", enabled: true}
	RegisterProvider("global-test", p)

	retrieved, ok := GetProvider("global-test")
	require.True(t, ok)
	assert.Equal(t, "global-test", retrieved.Name())
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	assert.NotNil(t, providers)
}

func TestListEnabledProviders(t *testing.T) {
	providers := ListEnabledProviders()
	assert.NotNil(t, providers)
}
