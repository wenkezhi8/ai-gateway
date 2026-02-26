package cache

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ModelNameMapping struct {
	Provider       string    `json:"provider"`
	OriginalModel  string    `json:"original_model"`
	EffectiveModel string    `json:"effective_model"`
	LastUsed       time.Time `json:"last_used"`
	SuccessCount   int       `json:"success_count"`
	LastProvider   string    `json:"last_provider,omitempty"`
}

type ModelMappingCache struct {
	mappings map[string]*ModelNameMapping
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
}

type ModelMappingConfig struct {
	MaxSize int           `json:"max_size"`
	TTL     time.Duration `json:"ttl"`
}

func NewModelMappingCache(config ModelMappingConfig) *ModelMappingCache {
	if config.MaxSize <= 0 {
		config.MaxSize = 1000
	}
	if config.TTL <= 0 {
		config.TTL = 24 * time.Hour
	}

	c := &ModelMappingCache{
		mappings: make(map[string]*ModelNameMapping),
		maxSize:  config.MaxSize,
		ttl:      config.TTL,
	}

	go c.cleanupLoop()

	return c
}

func (c *ModelMappingCache) key(provider, model string) string {
	return provider + "::" + model
}

func (c *ModelMappingCache) GetEffectiveModel(provider, model string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.key(provider, model)
	mapping, exists := c.mappings[key]
	if !exists {
		return "", false
	}

	if time.Since(mapping.LastUsed) > c.ttl {
		return "", false
	}

	return mapping.EffectiveModel, true
}

func (c *ModelMappingCache) RecordSuccess(provider, originalModel, effectiveModel string) {
	if originalModel == effectiveModel {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.key(provider, originalModel)

	if mapping, exists := c.mappings[key]; exists {
		mapping.SuccessCount++
		mapping.LastUsed = time.Now()
		mapping.LastProvider = provider
		logrus.WithFields(logrus.Fields{
			"provider":        provider,
			"original_model":  originalModel,
			"effective_model": effectiveModel,
			"success_count":   mapping.SuccessCount,
		}).Debug("Model mapping cache updated")
		return
	}

	if len(c.mappings) >= c.maxSize {
		c.evictOldest()
	}

	c.mappings[key] = &ModelNameMapping{
		Provider:       provider,
		OriginalModel:  originalModel,
		EffectiveModel: effectiveModel,
		LastUsed:       time.Now(),
		SuccessCount:   1,
		LastProvider:   provider,
	}

	logrus.WithFields(logrus.Fields{
		"provider":        provider,
		"original_model":  originalModel,
		"effective_model": effectiveModel,
	}).Info("Model mapping cached")
}

func (c *ModelMappingCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, mapping := range c.mappings {
		if oldestKey == "" || mapping.LastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = mapping.LastUsed
		}
	}

	if oldestKey != "" {
		delete(c.mappings, oldestKey)
		logrus.WithField("key", oldestKey).Debug("Evicted oldest model mapping")
	}
}

func (c *ModelMappingCache) cleanupLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.Cleanup()
	}
}

func (c *ModelMappingCache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := 0

	for key, mapping := range c.mappings {
		if now.Sub(mapping.LastUsed) > c.ttl {
			delete(c.mappings, key)
			expired++
		}
	}

	if expired > 0 {
		logrus.WithField("expired_count", expired).Info("Cleaned up expired model mappings")
	}

	return expired
}

func (c *ModelMappingCache) GetAll() []*ModelNameMapping {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*ModelNameMapping, 0, len(c.mappings))
	for _, mapping := range c.mappings {
		result = append(result, mapping)
	}
	return result
}

func (c *ModelMappingCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"total_entries": len(c.mappings),
		"max_size":      c.maxSize,
		"ttl_seconds":   c.ttl.Seconds(),
	}
}

func (c *ModelMappingCache) Clear() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := len(c.mappings)
	c.mappings = make(map[string]*ModelNameMapping)

	logrus.WithField("cleared_count", count).Info("Cleared all model mappings")
	return count
}

func (c *ModelMappingCache) Remove(provider, model string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.key(provider, model)
	if _, exists := c.mappings[key]; exists {
		delete(c.mappings, key)
		return true
	}
	return false
}
