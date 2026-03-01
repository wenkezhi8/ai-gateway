package cache

import (
	"container/list"
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
	value          []byte
	expiresAt      time.Time
	createdAt      time.Time
	hits           int
	ttl            int
	preview        string
	model          string
	provider       string
	taskType       string
	taskTypeSource string
}

//nolint:revive // Type name kept for compatibility with existing callers.
type CacheMeta struct {
	Size           int
	Hits           int
	CreatedAt      time.Time
	TTL            int
	Preview        string
	Model          string
	Provider       string
	TaskType       string
	TaskTypeSource string
}

// MemoryCache is an in-memory cache implementation.
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]*cacheItem
	maxItems int
	lruList  *list.List
	lruMap   map[string]*list.Element
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	return NewMemoryCacheWithMaxEntries(0)
}

// NewMemoryCacheWithMaxEntries creates a new in-memory cache with max entries.
// maxEntries <= 0 means unlimited.
func NewMemoryCacheWithMaxEntries(maxEntries int) *MemoryCache {
	if maxEntries < 0 {
		maxEntries = 0
	}

	cache := &MemoryCache{
		items:    make(map[string]*cacheItem),
		maxItems: maxEntries,
	}
	if maxEntries > 0 {
		cache.lruList = list.New()
		cache.lruMap = make(map[string]*list.Element)
	}

	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache.
func (c *MemoryCache) Get(_ context.Context, key string, dest interface{}) error {
	c.mu.Lock()
	item, ok := c.items[key]
	if !ok {
		c.mu.Unlock()
		return ErrNotFound
	}

	if time.Now().After(item.expiresAt) && !item.expiresAt.IsZero() {
		c.deleteItemLocked(key)
		c.mu.Unlock()
		return ErrNotFound
	}

	item.hits++
	c.touchLRULocked(key)
	data := append([]byte(nil), item.value...)
	c.mu.Unlock()

	return json.Unmarshal(data, dest)
}

// Set stores a value in the cache.
func (c *MemoryCache) Set(_ context.Context, key string, value interface{}, ttl time.Duration) error {
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

	c.setItemLocked(key, &cacheItem{
		value:     data,
		expiresAt: expiresAt,
		createdAt: time.Now(),
		ttl:       int(ttl.Seconds()),
		preview:   preview,
	})

	return nil
}

// SetWithMeta stores a value with metadata.
func (c *MemoryCache) SetWithMeta(_ context.Context, key string, value interface{}, ttl time.Duration, model, provider string) error {
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

	c.setItemLocked(key, &cacheItem{
		value:     data,
		expiresAt: expiresAt,
		createdAt: time.Now(),
		ttl:       int(ttl.Seconds()),
		preview:   preview,
		model:     model,
		provider:  provider,
		taskType:  "",
	})

	return nil
}

// SetWithTaskType stores a value with task type metadata.
func (c *MemoryCache) SetWithTaskType(_ context.Context, key string, value interface{}, ttl time.Duration, model, provider, taskType, taskTypeSource string) error {
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

	c.setItemLocked(key, &cacheItem{
		value:          data,
		expiresAt:      expiresAt,
		createdAt:      time.Now(),
		ttl:            int(ttl.Seconds()),
		preview:        preview,
		model:          model,
		provider:       provider,
		taskType:       taskType,
		taskTypeSource: taskTypeSource,
	})

	return nil
}

// Delete removes a value from the cache.
func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteItemLocked(key)
	return nil
}

// Exists checks if a key exists.
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return false, nil
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		if err := c.Delete(ctx, key); err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

// DeleteByPattern removes all keys matching a pattern (supports * wildcard).
func (c *MemoryCache) DeleteByPattern(_ context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := strings.TrimSuffix(pattern, "*")

	for key := range c.items {
		if strings.HasPrefix(key, prefix) {
			c.deleteItemLocked(key)
		}
	}

	return nil
}

// Keys returns all keys matching a pattern.
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

// GetMeta returns metadata for a key.
func (c *MemoryCache) GetMeta(key string) *CacheMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil
	}

	return &CacheMeta{
		Size:           len(item.value),
		Hits:           item.hits,
		CreatedAt:      item.createdAt,
		TTL:            item.ttl,
		Preview:        item.preview,
		Model:          item.model,
		Provider:       item.provider,
		TaskType:       item.taskType,
		TaskTypeSource: item.taskTypeSource,
	}
}

// cleanup removes expired items periodically.
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
				c.deleteItemLocked(key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *MemoryCache) setItemLocked(key string, item *cacheItem) {
	c.items[key] = item

	if c.maxItems <= 0 {
		return
	}

	if elem, ok := c.lruMap[key]; ok {
		c.lruList.MoveToFront(elem)
	} else {
		c.lruMap[key] = c.lruList.PushFront(key)
	}

	for len(c.items) > c.maxItems {
		c.evictOldestLocked()
	}
}

func (c *MemoryCache) touchLRULocked(key string) {
	if c.maxItems <= 0 {
		return
	}

	if elem, ok := c.lruMap[key]; ok {
		c.lruList.MoveToFront(elem)
	}
}

func (c *MemoryCache) evictOldestLocked() {
	if c.maxItems <= 0 {
		return
	}

	oldest := c.lruList.Back()
	if oldest == nil {
		return
	}

	key, ok := oldest.Value.(string)
	if !ok {
		c.lruList.Remove(oldest)
		return
	}

	c.lruList.Remove(oldest)
	delete(c.lruMap, key)
	delete(c.items, key)
}

func (c *MemoryCache) deleteItemLocked(key string) {
	delete(c.items, key)

	if c.maxItems <= 0 {
		return
	}

	if elem, ok := c.lruMap[key]; ok {
		c.lruList.Remove(elem)
		delete(c.lruMap, key)
	}
}
