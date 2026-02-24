package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("key not found")
)

type cacheItem struct {
	value     []byte
	expiresAt time.Time
	createdAt time.Time
	hits      int
	ttl       int
	preview   string
	model     string
	provider  string
}

type CacheMeta struct {
	Size      int
	Hits      int
	CreatedAt time.Time
	TTL       int
	Preview   string
	Model     string
	Provider  string
}

// MemoryCache is an in-memory cache implementation
type MemoryCache struct {
	items map[string]*cacheItem
	mu    sync.RWMutex
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]*cacheItem),
	}

	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.Lock()
	item, ok := c.items[key]
	if ok {
		item.hits++
	}
	c.mu.Unlock()

	if !ok {
		return ErrNotFound
	}

	if time.Now().After(item.expiresAt) && !item.expiresAt.IsZero() {
		c.Delete(ctx, key)
		return ErrNotFound
	}

	return json.Unmarshal(item.value, dest)
}

// Set stores a value in the cache
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	preview := ""
	if len(data) > 200 {
		preview = string(data[:200]) + "..."
	} else {
		preview = string(data)
	}

	c.items[key] = &cacheItem{
		value:     data,
		expiresAt: expiresAt,
		createdAt: time.Now(),
		ttl:       int(ttl.Seconds()),
		preview:   preview,
	}

	return nil
}

// SetWithMeta stores a value with metadata
func (c *MemoryCache) SetWithMeta(ctx context.Context, key string, value interface{}, ttl time.Duration, model, provider string) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	preview := ""
	if len(data) > 200 {
		preview = string(data[:200]) + "..."
	} else {
		preview = string(data)
	}

	c.items[key] = &cacheItem{
		value:     data,
		expiresAt: expiresAt,
		createdAt: time.Now(),
		ttl:       int(ttl.Seconds()),
		preview:   preview,
		model:     model,
		provider:  provider,
	}

	return nil
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

// Exists checks if a key exists
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return false, nil
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		c.Delete(ctx, key)
		return false, nil
	}

	return true, nil
}

// DeleteByPattern removes all keys matching a pattern (supports * wildcard)
func (c *MemoryCache) DeleteByPattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := strings.TrimSuffix(pattern, "*")

	for key := range c.items {
		if strings.HasPrefix(key, prefix) {
			delete(c.items, key)
		}
	}

	return nil
}

// Keys returns all keys matching a pattern
func (c *MemoryCache) Keys(pattern string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0)
	prefix := strings.TrimSuffix(pattern, "*")
	isWildcard := strings.HasSuffix(pattern, "*")

	for key := range c.items {
		if isWildcard {
			if strings.HasPrefix(key, prefix) {
				keys = append(keys, key)
			}
		} else {
			if key == pattern {
				keys = append(keys, key)
			}
		}
	}

	return keys
}

// GetMeta returns metadata for a key
func (c *MemoryCache) GetMeta(key string) *CacheMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil
	}

	return &CacheMeta{
		Size:      len(item.value),
		Hits:      item.hits,
		CreatedAt: item.createdAt,
		TTL:       item.ttl,
		Preview:   item.preview,
		Model:     item.model,
		Provider:  item.provider,
	}
}

// cleanup removes expired items periodically
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
