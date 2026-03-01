//nolint:revive
package provider

import (
	"context"
	"sync"
	"time"

	"ai-gateway/pkg/logger"
)

var poolLogger = logger.WithField("component", "provider")

type PooledProvider struct {
	provider   Provider
	lastUsed   time.Time
	useCount   int64
	errorCount int64
}

type ProviderPoolConfig struct {
	MaxIdleTime         time.Duration
	MaxUseCount         int64
	HealthCheckInterval time.Duration
	MaxErrorCount       int64
}

type ProviderPool struct {
	mu        sync.RWMutex
	providers map[string]*PooledProvider
	config    ProviderPoolConfig
	stopChan  chan struct{}
	// Synchronizes background goroutines for graceful shutdown testing
	stopWg sync.WaitGroup
	// Ensure Stop() is idempotent
	stopOnce sync.Once
}

var (
	globalPool     *ProviderPool
	globalPoolOnce sync.Once
)

func GetProviderPool() *ProviderPool {
	globalPoolOnce.Do(func() {
		globalPool = NewProviderPool(ProviderPoolConfig{
			MaxIdleTime:         30 * time.Minute,
			MaxUseCount:         10000,
			HealthCheckInterval: 60 * time.Second,
			MaxErrorCount:       10,
		})
	})
	return globalPool
}

func NewProviderPool(config ProviderPoolConfig) *ProviderPool {
	p := &ProviderPool{
		providers: make(map[string]*PooledProvider),
		config:    config,
		stopChan:  make(chan struct{}),
	}
	// Prepare to wait for two background goroutines
	p.stopWg.Add(2)
	go p.backgroundCleanup()
	go p.healthCheck()
	return p
}

func (p *ProviderPool) Get(name string) Provider {
	p.mu.RLock()
	if pooled, ok := p.providers[name]; ok {
		pooled.lastUsed = time.Now()
		pooled.useCount++
		p.mu.RUnlock()
		return pooled.provider
	}
	p.mu.RUnlock()

	registry := GetRegistry()
	prov, ok := registry.Get(name)
	if !ok || prov == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.providers[name] = &PooledProvider{
		provider: prov,
		lastUsed: time.Now(),
		useCount: 1,
	}
	return prov
}

func (p *ProviderPool) GetByModel(model string) Provider {
	registry := GetRegistry()
	prov, ok := registry.GetByModel(model)
	if !ok || prov == nil {
		return nil
	}
	name := prov.Name()
	p.mu.Lock()
	defer p.mu.Unlock()
	if pooled, ok := p.providers[name]; ok {
		pooled.lastUsed = time.Now()
		pooled.useCount++
		return pooled.provider
	}
	p.providers[name] = &PooledProvider{
		provider: prov,
		lastUsed: time.Now(),
		useCount: 1,
	}
	return prov
}

func (p *ProviderPool) RecordError(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pooled, ok := p.providers[name]; ok {
		pooled.errorCount++
		if pooled.errorCount >= p.config.MaxErrorCount {
			poolLogger.WithField("provider", name).Warn("Provider error threshold exceeded")
		}
	}
}

func (p *ProviderPool) RecordSuccess(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pooled, ok := p.providers[name]; ok {
		pooled.errorCount = 0
	}
}

func (p *ProviderPool) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	stats := make(map[string]interface{})
	for name, pooled := range p.providers {
		stats[name] = map[string]interface{}{
			"use_count":   pooled.useCount,
			"error_count": pooled.errorCount,
			"last_used":   pooled.lastUsed,
		}
	}
	return stats
}

func (p *ProviderPool) backgroundCleanup() {
	defer p.stopWg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.cleanup()
		case <-p.stopChan:
			return
		}
	}
}

func (p *ProviderPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for name, pooled := range p.providers {
		if now.Sub(pooled.lastUsed) > p.config.MaxIdleTime {
			delete(p.providers, name)
			poolLogger.WithField("provider", name).Debug("Removed idle provider from pool")
		}
	}
}

func (p *ProviderPool) healthCheck() {
	defer p.stopWg.Done()
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.checkHealth()
		case <-p.stopChan:
			return
		}
	}
}

func (p *ProviderPool) checkHealth() {
	p.mu.RLock()
	providers := make([]*PooledProvider, 0, len(p.providers))
	for _, pooled := range p.providers {
		providers = append(providers, pooled)
	}
	p.mu.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, pooled := range providers {
		if pooled.provider.IsEnabled() {
			if !pooled.provider.ValidateKey(ctx) {
				poolLogger.WithField("provider", pooled.provider.Name()).Warn("Provider key validation failed")
			}
		}
	}
}

func (p *ProviderPool) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopChan)
		p.stopWg.Wait()
	})
}

func (p *ProviderPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.providers)
}
