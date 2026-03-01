package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ai-gateway/pkg/logger"
)

var _ = logger.WithField("component", "cache")

type PendingRequest struct {
	Key       string
	CreatedAt time.Time
	Waiters   []chan *DedupResponse
	Response  *DedupResponse
	Done      bool
}

type DedupResponse struct {
	Data  interface{}
	Error error
}

type RequestDeduplicatorConfig struct {
	MaxPending      int
	RequestTimeout  time.Duration
	CleanupInterval time.Duration
}

type RequestDeduplicator struct {
	mu       sync.RWMutex
	pending  map[string]*PendingRequest
	config   RequestDeduplicatorConfig
	stats    DedupStats
	stopChan chan struct{}
	enabled  bool
}

type DedupStats struct {
	mu             sync.RWMutex
	TotalRequests  int64
	Deduplicated   int64
	UniqueRequests int64
	Timeouts       int64
}

var (
	globalDeduplicator     *RequestDeduplicator
	globalDeduplicatorOnce sync.Once
)

func GetRequestDeduplicator() *RequestDeduplicator {
	globalDeduplicatorOnce.Do(func() {
		globalDeduplicator = NewRequestDeduplicator(RequestDeduplicatorConfig{
			MaxPending:      1000,
			RequestTimeout:  30 * time.Second,
			CleanupInterval: 10 * time.Second,
		})
	})
	return globalDeduplicator
}

func NewRequestDeduplicator(config RequestDeduplicatorConfig) *RequestDeduplicator {
	d := &RequestDeduplicator{
		pending:  make(map[string]*PendingRequest),
		config:   config,
		stopChan: make(chan struct{}),
		enabled:  true,
	}

	go d.cleanupLoop()

	return d
}

func (d *RequestDeduplicator) GenerateKey(prompt, model string, params map[string]interface{}) string {
	data := struct {
		Prompt string
		Model  string
		Params map[string]interface{}
	}{
		Prompt: prompt,
		Model:  model,
		Params: params,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

func (d *RequestDeduplicator) Do(ctx context.Context, key string, fn func() (interface{}, error)) (interface{}, error) {
	d.mu.RLock()
	enabled := d.enabled
	requestTimeout := d.config.RequestTimeout
	d.mu.RUnlock()

	if !enabled {
		return fn()
	}

	d.stats.mu.Lock()
	d.stats.TotalRequests++
	d.stats.mu.Unlock()

	d.mu.Lock()

	if pending, ok := d.pending[key]; ok && !pending.Done {
		ch := make(chan *DedupResponse, 1)
		pending.Waiters = append(pending.Waiters, ch)
		d.mu.Unlock()

		d.stats.mu.Lock()
		d.stats.Deduplicated++
		d.stats.mu.Unlock()

		select {
		case resp := <-ch:
			return resp.Data, resp.Error
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(requestTimeout):
			d.stats.mu.Lock()
			d.stats.Timeouts++
			d.stats.mu.Unlock()
			return nil, ErrTimeout
		}
	}

	if len(d.pending) >= d.config.MaxPending {
		d.mu.Unlock()
		return fn()
	}

	pending := &PendingRequest{
		Key:       key,
		CreatedAt: time.Now(),
		Waiters:   make([]chan *DedupResponse, 0),
	}
	d.pending[key] = pending
	d.mu.Unlock()

	d.stats.mu.Lock()
	d.stats.UniqueRequests++
	d.stats.mu.Unlock()

	result, err := fn()

	d.mu.Lock()
	defer d.mu.Unlock()

	resp := &DedupResponse{
		Data:  result,
		Error: err,
	}

	for _, waiter := range pending.Waiters {
		select {
		case waiter <- resp:
		default:
		}
	}

	delete(d.pending, key)

	return result, err
}

// UpdateConfig updates dedup configuration at runtime.
func (d *RequestDeduplicator) UpdateConfig(config RequestDeduplicatorConfig, enabled *bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.config = config
	if enabled != nil {
		d.enabled = *enabled
	}
}

// GetConfig returns current configuration and enabled flag.
func (d *RequestDeduplicator) GetConfig() (RequestDeduplicatorConfig, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config, d.enabled
}

func (d *RequestDeduplicator) GetStats() map[string]interface{} {
	d.stats.mu.RLock()
	defer d.stats.mu.RUnlock()

	d.mu.RLock()
	pendingCount := len(d.pending)
	d.mu.RUnlock()

	return map[string]interface{}{
		"total_requests":  d.stats.TotalRequests,
		"deduplicated":    d.stats.Deduplicated,
		"unique_requests": d.stats.UniqueRequests,
		"pending_count":   pendingCount,
		"dedup_rate":      d.getDedupRate(),
		"timeouts":        d.stats.Timeouts,
	}
}

func (d *RequestDeduplicator) getDedupRate() float64 {
	if d.stats.TotalRequests == 0 {
		return 0
	}
	return float64(d.stats.Deduplicated) / float64(d.stats.TotalRequests)
}

func (d *RequestDeduplicator) cleanupLoop() {
	d.mu.RLock()
	interval := d.config.CleanupInterval
	d.mu.RUnlock()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.cleanup()
		case <-d.stopChan:
			return
		}
	}
}

func (d *RequestDeduplicator) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for key, pending := range d.pending {
		if now.Sub(pending.CreatedAt) > d.config.RequestTimeout {
			resp := &DedupResponse{
				Data:  nil,
				Error: ErrTimeout,
			}
			for _, waiter := range pending.Waiters {
				select {
				case waiter <- resp:
				default:
				}
			}
			delete(d.pending, key)
		}
	}
}

func (d *RequestDeduplicator) Stop() {
	close(d.stopChan)
}

func (d *RequestDeduplicator) PendingCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.pending)
}

var ErrTimeout = fmt.Errorf("request timeout")
