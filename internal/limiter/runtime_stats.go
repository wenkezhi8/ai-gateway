//nolint:godot,revive
package limiter

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// EWMATracker tracks exponential weighted moving average values
type EWMATracker struct {
	alpha     float64
	valueBits atomic.Uint64
}

// NewEWMATracker creates a new EWMA tracker with the given smoothing factor
func NewEWMATracker(alpha float64) *EWMATracker {
	t := &EWMATracker{alpha: alpha}
	t.valueBits.Store(math.Float64bits(math.NaN()))
	return t
}

// Update updates the EWMA with a new sample
func (t *EWMATracker) Update(sample float64) {
	if t == nil {
		return
	}
	for {
		oldBits := t.valueBits.Load()
		oldValue := math.Float64frombits(oldBits)
		var newValue float64
		if math.IsNaN(oldValue) {
			newValue = sample
		} else {
			newValue = t.alpha*sample + (1-t.alpha)*oldValue
		}
		if t.valueBits.CompareAndSwap(oldBits, math.Float64bits(newValue)) {
			return
		}
	}
}

// Get returns the current EWMA value
func (t *EWMATracker) Get() float64 {
	if t == nil {
		return 0
	}
	value := math.Float64frombits(t.valueBits.Load())
	if math.IsNaN(value) {
		return 0
	}
	return value
}

// RuntimeStatsManager manages runtime statistics for accounts
type RuntimeStatsManager struct {
	stats sync.Map
	alpha float64
}

// NewRuntimeStatsManager creates a new runtime stats manager
func NewRuntimeStatsManager(alpha float64) *RuntimeStatsManager {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.2
	}
	return &RuntimeStatsManager{alpha: alpha}
}

// accountRuntimeStat holds the runtime statistics for a single account
type accountRuntimeStat struct {
	errorRateEWMA *EWMATracker
	ttftMsEWMA    *EWMATracker
	successCount  atomic.Int64
	errorCount    atomic.Int64
	lastUpdate    atomic.Int64
}

// getOrCreate gets or creates runtime stats for an account
func (m *RuntimeStatsManager) getOrCreate(accountID string) *accountRuntimeStat {
	if value, ok := m.stats.Load(accountID); ok {
		if stat, ok := value.(*accountRuntimeStat); ok {
			return stat
		}
	}

	stat := &accountRuntimeStat{
		errorRateEWMA: NewEWMATracker(m.alpha),
		ttftMsEWMA:    NewEWMATracker(m.alpha),
	}
	stat.lastUpdate.Store(time.Now().UnixNano())

	actual, loaded := m.stats.LoadOrStore(accountID, stat)
	if loaded {
		if loadedStat, ok := actual.(*accountRuntimeStat); ok {
			return loadedStat
		}
	}
	return stat
}

// ReportResult reports a request result for an account
func (m *RuntimeStatsManager) ReportResult(accountID string, success bool, ttftMs int64) {
	if m == nil || accountID == "" {
		return
	}

	stat := m.getOrCreate(accountID)

	// Update error rate EWMA
	errorSample := 0.0
	if !success {
		errorSample = 1.0
	}
	stat.errorRateEWMA.Update(errorSample)

	// Update TTFT EWMA if provided
	if ttftMs > 0 {
		stat.ttftMsEWMA.Update(float64(ttftMs))
	}

	// Update counters
	if success {
		stat.successCount.Add(1)
	} else {
		stat.errorCount.Add(1)
	}

	stat.lastUpdate.Store(time.Now().UnixNano())
}

// GetStats returns the runtime statistics for an account
func (m *RuntimeStatsManager) GetStats(accountID string) *AccountRuntimeStats {
	if m == nil || accountID == "" {
		return nil
	}

	value, ok := m.stats.Load(accountID)
	if !ok {
		return &AccountRuntimeStats{
			AccountID:      accountID,
			ErrorRateEWMA:  0,
			TTFTMsEWMA:     0,
			SuccessCount:   0,
			ErrorCount:     0,
			LastUpdateTime: time.Time{},
		}
	}

	stat, ok := value.(*accountRuntimeStat)
	if !ok {
		return nil
	}
	lastUpdateNano := stat.lastUpdate.Load()

	return &AccountRuntimeStats{
		AccountID:      accountID,
		ErrorRateEWMA:  clamp01(stat.errorRateEWMA.Get()),
		TTFTMsEWMA:     stat.ttftMsEWMA.Get(),
		SuccessCount:   stat.successCount.Load(),
		ErrorCount:     stat.errorCount.Load(),
		LastUpdateTime: time.Unix(0, lastUpdateNano),
	}
}

// GetErrorRate returns the error rate EWMA for an account
func (m *RuntimeStatsManager) GetErrorRate(accountID string) float64 {
	if m == nil || accountID == "" {
		return 0
	}

	value, ok := m.stats.Load(accountID)
	if !ok {
		return 0
	}

	stat, ok := value.(*accountRuntimeStat)
	if !ok {
		return 0
	}
	return clamp01(stat.errorRateEWMA.Get())
}

// GetTTFT returns the TTFT EWMA for an account in milliseconds
func (m *RuntimeStatsManager) GetTTFT(accountID string) float64 {
	if m == nil || accountID == "" {
		return 0
	}

	value, ok := m.stats.Load(accountID)
	if !ok {
		return 0
	}

	stat, ok := value.(*accountRuntimeStat)
	if !ok {
		return 0
	}
	return stat.ttftMsEWMA.Get()
}

// Reset resets the statistics for an account
func (m *RuntimeStatsManager) Reset(accountID string) {
	if m == nil || accountID == "" {
		return
	}
	m.stats.Delete(accountID)
}

// ResetAll resets all statistics
func (m *RuntimeStatsManager) ResetAll() {
	if m == nil {
		return
	}
	m.stats.Range(func(key, value interface{}) bool {
		m.stats.Delete(key)
		return true
	})
}

// Size returns the number of accounts being tracked
func (m *RuntimeStatsManager) Size() int {
	if m == nil {
		return 0
	}
	count := 0
	m.stats.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// clamp01 clamps a value to the range [0, 1]
func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
