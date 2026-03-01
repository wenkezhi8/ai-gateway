//nolint:godot
package provider

import (
	"fmt"
	"sync"
)

// Registry manages all AI providers
type Registry struct {
	providers map[string]Provider
	factories map[string]FactoryFunc
	mu        sync.RWMutex
}

// FactoryFunc is a function that creates a provider from config
type FactoryFunc func(cfg *ProviderConfig) Provider

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		factories: make(map[string]FactoryFunc),
	}
}

// RegisterFactory registers a provider factory
func (r *Registry) RegisterFactory(name string, factory FactoryFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Register adds a provider instance to the registry
func (r *Registry) Register(name string, p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = p
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// GetByModel finds a provider that supports the given model
func (r *Registry) GetByModel(model string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.providers {
		if !p.IsEnabled() {
			continue
		}
		for _, m := range p.Models() {
			if m == model {
				return p, true
			}
		}
	}
	return nil, false
}

// Remove removes a provider from the registry
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, name)
}

// List returns all registered providers
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// ListEnabled returns all enabled providers
func (r *Registry) ListEnabled() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0)
	for _, p := range r.providers {
		if p.IsEnabled() {
			providers = append(providers, p)
		}
	}
	return providers
}

// CreateProvider creates a provider instance using registered factory
func (r *Registry) CreateProvider(cfg *ProviderConfig) (Provider, error) {
	r.mu.RLock()
	factory, ok := r.factories[cfg.Name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", cfg.Name)
	}

	return factory(cfg), nil
}

// CreateAndRegister creates a provider and registers it
func (r *Registry) CreateAndRegister(cfg *ProviderConfig) (Provider, error) {
	p, err := r.CreateProvider(cfg)
	if err != nil {
		return nil, err
	}

	r.Register(cfg.Name, p)
	return p, nil
}

// LoadFromConfig loads providers from configuration
func (r *Registry) LoadFromConfig(configs []ProviderConfig) error {
	for _, cfg := range configs {
		if _, err := r.CreateAndRegister(&cfg); err != nil {
			return fmt.Errorf("failed to load provider %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// GetFactoryNames returns all registered factory names
func (r *Registry) GetFactoryNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Global registry instance
var globalRegistry *Registry
var once sync.Once

// GetRegistry returns the global registry instance
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = NewRegistry()
	})
	return globalRegistry
}

// RegisterProvider registers a provider to the global registry
func RegisterProvider(name string, p Provider) {
	GetRegistry().Register(name, p)
}

// GetProvider retrieves a provider from the global registry
func GetProvider(name string) (Provider, bool) {
	return GetRegistry().Get(name)
}

// GetProviderByModel finds a provider that supports the given model
func GetProviderByModel(model string) (Provider, bool) {
	return GetRegistry().GetByModel(model)
}

// ListProviders returns all providers from the global registry
func ListProviders() []Provider {
	return GetRegistry().List()
}

// ListEnabledProviders returns all enabled providers from the global registry
func ListEnabledProviders() []Provider {
	return GetRegistry().ListEnabled()
}

// ClearRegistry clears the global registry (for testing only)
func ClearRegistry() {
	globalRegistry = nil
	once = sync.Once{}
}
