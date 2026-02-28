package limiter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ConcurrencyManager manages concurrent request slots for accounts
type ConcurrencyManager struct {
	store      *RedisStore
	localSlots sync.Map // fallback for when Redis is unavailable
}

// NewConcurrencyManager creates a new concurrency manager
func NewConcurrencyManager(store *RedisStore) *ConcurrencyManager {
	return &ConcurrencyManager{
		store: store,
	}
}

// localSlot tracks local concurrency (for fallback)
type localSlot struct {
	count atomic.Int64
}

// AcquireResult represents the result of trying to acquire a slot
type AcquireResult struct {
	Acquired    bool
	Current     int64
	Max         int
	ReleaseFunc func()
}

// TryAcquire tries to acquire a concurrency slot for an account
func (m *ConcurrencyManager) TryAcquire(ctx context.Context, accountID string, maxConcurrency int) (*AcquireResult, error) {
	if m == nil || accountID == "" {
		return &AcquireResult{Acquired: false}, nil
	}

	// If maxConcurrency is 0, unlimited concurrency
	if maxConcurrency <= 0 {
		return &AcquireResult{
			Acquired:    true,
			Current:     0,
			Max:         0,
			ReleaseFunc: func() {},
		}, nil
	}

	// Try Redis first if available
	if m.store != nil {
		current, acquired, err := m.store.TryAcquireSlot(ctx, accountID, maxConcurrency)
		if err != nil {
			// Fallback to local on Redis error
			return m.tryAcquireLocal(accountID, maxConcurrency)
		}

		result := &AcquireResult{
			Acquired: acquired,
			Current:  current,
			Max:      maxConcurrency,
		}

		if acquired {
			result.ReleaseFunc = func() {
				_ = m.store.ReleaseSlot(context.Background(), accountID)
			}
		}

		return result, nil
	}

	// Use local slots
	return m.tryAcquireLocal(accountID, maxConcurrency)
}

// tryAcquireLocal tries to acquire a slot using local memory
func (m *ConcurrencyManager) tryAcquireLocal(accountID string, maxConcurrency int) (*AcquireResult, error) {
	// Get or create local slot
	value, _ := m.localSlots.LoadOrStore(accountID, &localSlot{})
	slot := value.(*localSlot)

	// Try to increment
	for {
		current := slot.count.Load()
		if current >= int64(maxConcurrency) {
			return &AcquireResult{
				Acquired: false,
				Current:  current,
				Max:      maxConcurrency,
			}, nil
		}

		if slot.count.CompareAndSwap(current, current+1) {
			return &AcquireResult{
				Acquired: true,
				Current:  current + 1,
				Max:      maxConcurrency,
				ReleaseFunc: func() {
					for {
						val := slot.count.Load()
						if val <= 0 {
							return
						}
						if slot.count.CompareAndSwap(val, val-1) {
							return
						}
					}
				},
			}, nil
		}
	}
}

// Release releases a concurrency slot for an account
func (m *ConcurrencyManager) Release(ctx context.Context, accountID string) error {
	if m == nil || accountID == "" {
		return nil
	}

	// Try Redis first
	if m.store != nil {
		if err := m.store.ReleaseSlot(ctx, accountID); err != nil {
			// Fallback to local
			m.releaseLocal(accountID)
		}
		return nil
	}

	// Use local slots
	m.releaseLocal(accountID)
	return nil
}

// releaseLocal releases a local slot
func (m *ConcurrencyManager) releaseLocal(accountID string) {
	value, ok := m.localSlots.Load(accountID)
	if !ok {
		return
	}

	slot := value.(*localSlot)
	for {
		current := slot.count.Load()
		if current <= 0 {
			return
		}
		if slot.count.CompareAndSwap(current, current-1) {
			return
		}
	}
}

// GetLoadInfo gets the current load information for an account
func (m *ConcurrencyManager) GetLoadInfo(ctx context.Context, accountID string, maxConcurrency int) (*AccountLoadInfo, error) {
	if m == nil || accountID == "" {
		return &AccountLoadInfo{
			AccountID:         accountID,
			CurrentConcurrent: 0,
			MaxConcurrency:    maxConcurrency,
			LoadRate:          0,
		}, nil
	}

	var currentConcurrent int64

	// Try Redis first
	if m.store != nil {
		val, err := m.store.GetConcurrent(ctx, accountID)
		if err == nil {
			currentConcurrent = val
		} else {
			// Fallback to local
			currentConcurrent = m.getLocalConcurrent(accountID)
		}
	} else {
		currentConcurrent = m.getLocalConcurrent(accountID)
	}

	loadRate := 0.0
	if maxConcurrency > 0 {
		loadRate = float64(currentConcurrent) / float64(maxConcurrency) * 100
		if loadRate > 100 {
			loadRate = 100
		}
	}

	return &AccountLoadInfo{
		AccountID:         accountID,
		CurrentConcurrent: int(currentConcurrent),
		MaxConcurrency:    maxConcurrency,
		LoadRate:          loadRate,
	}, nil
}

// getLocalConcurrent gets the local concurrent count
func (m *ConcurrencyManager) getLocalConcurrent(accountID string) int64 {
	value, ok := m.localSlots.Load(accountID)
	if !ok {
		return 0
	}
	return value.(*localSlot).count.Load()
}

// GetBatchLoadInfo gets load information for multiple accounts
func (m *ConcurrencyManager) GetBatchLoadInfo(ctx context.Context, accounts []*AccountConfig) map[string]*AccountLoadInfo {
	result := make(map[string]*AccountLoadInfo)

	for _, acc := range accounts {
		if acc == nil {
			continue
		}
		loadInfo, _ := m.GetLoadInfo(ctx, acc.ID, acc.Concurrency)
		if loadInfo != nil {
			result[acc.ID] = loadInfo
		}
	}

	return result
}

// Reset resets the concurrency count for an account (for cleanup)
func (m *ConcurrencyManager) Reset(ctx context.Context, accountID string) error {
	if m == nil || accountID == "" {
		return nil
	}

	// Clear local
	m.localSlots.Delete(accountID)

	// Clear Redis (set to 0)
	if m.store != nil {
		return m.store.SetConcurrentWithExpire(ctx, accountID, 0, time.Hour)
	}

	return nil
}

// ResetAll resets all concurrency counts
func (m *ConcurrencyManager) ResetAll() {
	m.localSlots = sync.Map{}
}
