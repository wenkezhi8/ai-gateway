package limiter

import (
	"context"
	"sync"
	"time"
)

// StickySessionManager manages session-to-account bindings
type StickySessionManager struct {
	store        *RedisStore
	localSession sync.Map // fallback for when Redis is unavailable
	localResp    sync.Map
	config       SchedulerConfig
}

// NewStickySessionManager creates a new sticky session manager
func NewStickySessionManager(store *RedisStore, config SchedulerConfig) *StickySessionManager {
	return &StickySessionManager{
		store:  store,
		config: config,
	}
}

// localSessionEntry holds a local session binding
type localSessionEntry struct {
	accountID string
	expiresAt time.Time
}

// BindSession binds a session hash to an account ID
func (m *StickySessionManager) BindSession(ctx context.Context, provider, sessionHash, accountID string) error {
	if m == nil || sessionHash == "" || accountID == "" {
		return nil
	}

	ttl := m.config.StickySessionTTL
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}

	// Try Redis first
	if m.store != nil {
		return m.store.SetSessionAccount(ctx, provider, sessionHash, accountID, ttl)
	}

	// Fallback to local
	m.localSession.Store(m.sessionKey(provider, sessionHash), &localSessionEntry{
		accountID: accountID,
		expiresAt: time.Now().Add(ttl),
	})

	return nil
}

// GetSessionAccount gets the account ID bound to a session hash
func (m *StickySessionManager) GetSessionAccount(ctx context.Context, provider, sessionHash string) (string, error) {
	if m == nil || sessionHash == "" {
		return "", nil
	}

	// Try Redis first
	if m.store != nil {
		accountID, err := m.store.GetSessionAccount(ctx, provider, sessionHash)
		if err != nil {
			// Fallback to local on error
			return m.getLocalSessionAccount(provider, sessionHash), nil
		}
		return accountID, nil
	}

	// Use local
	return m.getLocalSessionAccount(provider, sessionHash), nil
}

// getLocalSessionAccount gets from local storage
func (m *StickySessionManager) getLocalSessionAccount(provider, sessionHash string) string {
	key := m.sessionKey(provider, sessionHash)
	value, ok := m.localSession.Load(key)
	if !ok {
		return ""
	}

	entry := value.(*localSessionEntry)
	if time.Now().After(entry.expiresAt) {
		m.localSession.Delete(key)
		return ""
	}

	return entry.accountID
}

// DeleteSession deletes a session binding
func (m *StickySessionManager) DeleteSession(ctx context.Context, provider, sessionHash string) error {
	if m == nil || sessionHash == "" {
		return nil
	}

	// Try Redis first
	if m.store != nil {
		_ = m.store.DeleteSession(ctx, provider, sessionHash)
	}

	// Also delete local
	m.localSession.Delete(m.sessionKey(provider, sessionHash))

	return nil
}

// RefreshSessionTTL refreshes the TTL of a session binding
func (m *StickySessionManager) RefreshSessionTTL(ctx context.Context, provider, sessionHash string) error {
	if m == nil || sessionHash == "" {
		return nil
	}

	ttl := m.config.StickySessionTTL
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}

	// Try Redis first
	if m.store != nil {
		_ = m.store.RefreshSessionTTL(ctx, provider, sessionHash, ttl)
	}

	// Also refresh local
	key := m.sessionKey(provider, sessionHash)
	if value, ok := m.localSession.Load(key); ok {
		entry := value.(*localSessionEntry)
		entry.expiresAt = time.Now().Add(ttl)
	}

	return nil
}

// BindResponse binds a response ID to an account ID
func (m *StickySessionManager) BindResponse(ctx context.Context, responseID, accountID string) error {
	if m == nil || responseID == "" || accountID == "" {
		return nil
	}

	ttl := m.config.ResponseBindTTL
	if ttl <= 0 {
		ttl = 1 * time.Hour
	}

	// Try Redis first
	if m.store != nil {
		return m.store.SetResponseAccount(ctx, responseID, accountID, ttl)
	}

	// Fallback to local
	m.localResp.Store(responseID, &localSessionEntry{
		accountID: accountID,
		expiresAt: time.Now().Add(ttl),
	})

	return nil
}

// GetResponseAccount gets the account ID bound to a response ID
func (m *StickySessionManager) GetResponseAccount(ctx context.Context, responseID string) (string, error) {
	if m == nil || responseID == "" {
		return "", nil
	}

	// Try Redis first
	if m.store != nil {
		accountID, err := m.store.GetResponseAccount(ctx, responseID)
		if err != nil {
			// Fallback to local on error
			return m.getLocalResponseAccount(responseID), nil
		}
		return accountID, nil
	}

	// Use local
	return m.getLocalResponseAccount(responseID), nil
}

// getLocalResponseAccount gets from local storage
func (m *StickySessionManager) getLocalResponseAccount(responseID string) string {
	value, ok := m.localResp.Load(responseID)
	if !ok {
		return ""
	}

	entry := value.(*localSessionEntry)
	if time.Now().After(entry.expiresAt) {
		m.localResp.Delete(responseID)
		return ""
	}

	return entry.accountID
}

// sessionKey generates a session key
func (m *StickySessionManager) sessionKey(provider, sessionHash string) string {
	return provider + ":" + sessionHash
}

// CleanupExpired cleans up expired local entries (call periodically)
func (m *StickySessionManager) CleanupExpired() int {
	count := 0
	now := time.Now()

	m.localSession.Range(func(key, value interface{}) bool {
		entry := value.(*localSessionEntry)
		if now.After(entry.expiresAt) {
			m.localSession.Delete(key)
			count++
		}
		return true
	})

	m.localResp.Range(func(key, value interface{}) bool {
		entry := value.(*localSessionEntry)
		if now.After(entry.expiresAt) {
			m.localResp.Delete(key)
			count++
		}
		return true
	})

	return count
}

// ShouldClearSticky checks if sticky session should be cleared for an account
func ShouldClearSticky(account *AccountConfig, stats *AccountRuntimeStats, requestedModel string) bool {
	if account == nil {
		return false
	}

	// Clear if account is disabled
	if !account.Enabled {
		return true
	}

	// Clear if account health is unhealthy
	if account.HealthStatus == HealthStatusUnhealthy {
		return true
	}

	// Clear if error rate is too high
	if stats != nil && stats.ErrorRateEWMA > 0.5 {
		return true
	}

	return false
}
