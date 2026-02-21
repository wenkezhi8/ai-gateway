package limiter

import (
	"context"
	"sync"
)

// QuotaConfig defines quota limits for a user
type QuotaConfig struct {
	UserID       string
	DailyLimit   int64
	MonthlyLimit int64
	TokenLimit   int64
	Providers    map[string]int64 // per-provider limits
}

// QuotaManager manages user quotas
type QuotaManager struct {
	configs map[string]*QuotaConfig
	mu      sync.RWMutex
	tracker *LegacyUsageTracker
}

// NewQuotaManager creates a new quota manager
func NewQuotaManager(tracker *LegacyUsageTracker) *QuotaManager {
	return &QuotaManager{
		configs: make(map[string]*QuotaConfig),
		tracker: tracker,
	}
}

// SetQuota sets the quota for a user
func (m *QuotaManager) SetQuota(config *QuotaConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[config.UserID] = config
}

// GetQuota gets the quota config for a user
func (m *QuotaManager) GetQuota(userID string) (*QuotaConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, ok := m.configs[userID]
	return config, ok
}

// CheckQuota checks if a user has quota remaining
func (m *QuotaManager) CheckQuota(ctx context.Context, userID, provider string) (bool, error) {
	m.mu.RLock()
	config, ok := m.configs[userID]
	m.mu.RUnlock()

	if !ok {
		// No quota configured, allow the request
		return true, nil
	}

	usage, err := m.tracker.GetUsage(ctx, userID, provider)
	if err != nil {
		return false, err
	}

	// Check daily limit
	if config.DailyLimit > 0 && usage >= config.DailyLimit {
		return false, nil
	}

	// Check per-provider limit
	if limit, ok := config.Providers[provider]; ok && limit > 0 && usage >= limit {
		return false, nil
	}

	return true, nil
}

// ConsumeQuota consumes quota for a request
func (m *QuotaManager) ConsumeQuota(ctx context.Context, userID, provider string, tokens int64) error {
	allowed, err := m.CheckQuota(ctx, userID, provider)
	if err != nil {
		return err
	}

	if !allowed {
		return ErrQuotaExceeded
	}

	return m.tracker.IncrementUsage(ctx, userID, provider, tokens)
}
