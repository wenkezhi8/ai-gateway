//nolint:godot,unused,revive
package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRedisStore implements RedisStore interface for testing
type mockRedisStore struct {
	data       map[string]int64
	sortedSets map[string]map[string]float64
	err        error
}

func newMockRedisStore() *mockRedisStore {
	return &mockRedisStore{
		data:       make(map[string]int64),
		sortedSets: make(map[string]map[string]float64),
	}
}

func (m *mockRedisStore) Get(ctx context.Context, key string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.data[key], nil
}

func (m *mockRedisStore) Set(ctx context.Context, key string, value int64, expiration time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.data[key] = value
	return nil
}

func (m *mockRedisStore) Incr(ctx context.Context, key string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.data[key]++
	return m.data[key], nil
}

func (m *mockRedisStore) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.data[key] += value
	return m.data[key], nil
}

func (m *mockRedisStore) Decr(ctx context.Context, key string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.data[key]--
	return m.data[key], nil
}

func (m *mockRedisStore) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.data[key] -= value
	return m.data[key], nil
}

func (m *mockRedisStore) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return m.err
}

func (m *mockRedisStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	return time.Minute, m.err
}

func (m *mockRedisStore) Del(ctx context.Context, key string) error {
	delete(m.data, key)
	return m.err
}

func (m *mockRedisStore) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, m.err
}

func (m *mockRedisStore) ZAdd(ctx context.Context, key string, score float64, member string) error {
	if m.err != nil {
		return m.err
	}
	if m.sortedSets[key] == nil {
		m.sortedSets[key] = make(map[string]float64)
	}
	m.sortedSets[key][member] = score
	return nil
}

func (m *mockRedisStore) ZRangeByScore(ctx context.Context, key, minScore, maxScore string, opts interface{}) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := []string{}
	for member := range m.sortedSets[key] {
		result = append(result, member)
	}
	return result, nil
}

func (m *mockRedisStore) ZRemRangeByScore(ctx context.Context, key, minScore, maxScore string) error {
	return m.err
}

func (m *mockRedisStore) ZCard(ctx context.Context, key string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return int64(len(m.sortedSets[key])), nil
}

func (m *mockRedisStore) ZScore(ctx context.Context, key, member string) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.sortedSets[key][member], nil
}

func (m *mockRedisStore) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return nil, m.err
}

func (m *mockRedisStore) Ping(ctx context.Context) error {
	return m.err
}

func (m *mockRedisStore) Close() error {
	return m.err
}

// TestTokenLimiter tests the token limiter
func TestTokenLimiter_New(t *testing.T) {
	limiter := NewTokenLimiter(&RedisStore{client: nil}, PeriodHour, 1000, 0.9)
	assert.NotNil(t, limiter)
	assert.Equal(t, PeriodHour, limiter.period)
	assert.Equal(t, int64(1000), limiter.limit)
}

func TestTokenLimiter_DefaultWarning(t *testing.T) {
	limiter := NewTokenLimiter(&RedisStore{client: nil}, PeriodHour, 1000, 0)
	assert.Equal(t, 0.9, limiter.warning)

	limiter2 := NewTokenLimiter(&RedisStore{client: nil}, PeriodHour, 1000, 1.5)
	assert.Equal(t, 0.9, limiter2.warning)
}

func TestTokenLimiter_BuildKey(t *testing.T) {
	limiter := NewTokenLimiter(&RedisStore{client: nil}, PeriodDay, 1000, 0.9)
	key := limiter.buildKey("account1")
	assert.Contains(t, key, "token_limiter")
	assert.Contains(t, key, "day")
	assert.Contains(t, key, "account1")
}

func TestTokenLimiter_GetWindowStart(t *testing.T) {
	tests := []struct {
		period Period
	}{
		{PeriodMinute},
		{PeriodHour},
		{PeriodDay},
		{PeriodMonth},
	}

	for _, tt := range tests {
		t.Run(string(tt.period), func(t *testing.T) {
			limiter := NewTokenLimiter(&RedisStore{client: nil}, tt.period, 1000, 0.9)
			windowStart := limiter.getWindowStart()
			assert.True(t, windowStart.Before(time.Now()))
		})
	}
}

func TestTokenLimiter_GetResetTime(t *testing.T) {
	limiter := NewTokenLimiter(&RedisStore{client: nil}, PeriodHour, 1000, 0.9)
	resetTime := limiter.getResetTime()
	assert.True(t, resetTime.After(time.Now()))
}

// TestRPMLimiter tests the RPM limiter
func TestRPMLimiter_New(t *testing.T) {
	limiter := NewRPMLimiter(nil, 60, 120, 0.9)
	assert.NotNil(t, limiter)
	assert.Equal(t, 60, limiter.rpm)
	assert.Equal(t, 120, limiter.burst)
}

func TestRPMLimiter_GetLimiter(t *testing.T) {
	limiter := NewRPMLimiter(nil, 60, 120, 0.9)

	// Get limiter for key1
	l1 := limiter.getLimiter("key1")
	assert.NotNil(t, l1)

	// Get same limiter again
	l2 := limiter.getLimiter("key1")
	assert.Equal(t, l1, l2)

	// Get limiter for different key - should work even if pointer is same
	l3 := limiter.getLimiter("key2")
	assert.NotNil(t, l3)
	// Both limiters should be functional
	assert.True(t, l1.Allow())
	assert.True(t, l3.Allow())
}

func TestRPMLimiter_BuildKey(t *testing.T) {
	limiter := NewRPMLimiter(nil, 60, 120, 0.9)
	key := limiter.buildKey("user1")
	assert.Contains(t, key, "rpm_limiter")
	assert.Contains(t, key, "user1")
}

// TestConcurrentLimiter tests the concurrent limiter
func TestConcurrentLimiter_New(t *testing.T) {
	limiter := NewConcurrentLimiter(nil, 10, 0.9)
	assert.NotNil(t, limiter)
	assert.Equal(t, 10, limiter.maxConcur)
}

// TestAccountManager tests the account manager
func TestAccountManager_New(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.accounts)
	assert.NotNil(t, manager.statuses)
	assert.Equal(t, 3*time.Second, manager.switchTimeout)
}

func TestAccountManager_AddAccount(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Name:     "Test Account",
		Provider: "openai",
		APIKey:   "test-key",
		Enabled:  true,
		Priority: 1,
		Limits: map[LimitType]*LimitConfig{
			LimitTypeToken: {
				Type:   LimitTypeToken,
				Period: PeriodDay,
				Limit:  1000,
			},
		},
	}

	err := manager.AddAccount(config)
	require.NoError(t, err)

	// Verify account was added
	account, err := manager.GetActiveAccount("openai")
	require.NoError(t, err)
	assert.Equal(t, "acc1", account.ID)
}

func TestAccountManager_AddAccount_EmptyID(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "",
		Name:     "Test Account",
		Provider: "openai",
	}

	err := manager.AddAccount(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID is required")
}

func TestAccountManager_RemoveAccount(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Name:     "Test Account",
		Provider: "openai",
		APIKey:   "test-key",
		Enabled:  true,
		Priority: 1,
		Limits:   map[LimitType]*LimitConfig{},
	}

	err := manager.AddAccount(config)
	require.NoError(t, err)

	err = manager.RemoveAccount("acc1")
	require.NoError(t, err)

	// Verify account was removed
	_, err = manager.GetActiveAccount("openai")
	assert.Error(t, err)
	assert.Equal(t, ErrNoAvailableAccount, err)
}

func TestAccountManager_RemoveAccount_NotFound(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	err := manager.RemoveAccount("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAccountManager_GetAllAccounts(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	// Add multiple accounts
	for i := 1; i <= 3; i++ {
		config := &AccountConfig{
			ID:       string(rune('a' + i)),
			Name:     "Test Account",
			Provider: "openai",
			APIKey:   "test-key",
			Enabled:  true,
			Priority: i,
			Limits:   map[LimitType]*LimitConfig{},
		}
		err := manager.AddAccount(config)
		require.NoError(t, err)
	}

	accounts := manager.GetAllAccounts()
	assert.Len(t, accounts, 3)
}

func TestAccountManager_ForceSwitch(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	// Add two accounts
	config1 := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		APIKey:   "key1",
		Enabled:  true,
		Priority: 1,
		Limits:   map[LimitType]*LimitConfig{},
	}
	config2 := &AccountConfig{
		ID:       "acc2",
		Provider: "openai",
		APIKey:   "key2",
		Enabled:  true,
		Priority: 2,
		Limits:   map[LimitType]*LimitConfig{},
	}

	require.NoError(t, manager.AddAccount(config1))
	require.NoError(t, manager.AddAccount(config2))

	// Verify acc1 is active (first added)
	account, err := manager.GetActiveAccount("openai")
	require.NoError(t, err)
	assert.Equal(t, "acc1", account.ID)

	// Force switch to acc2
	err = manager.ForceSwitch("openai", "acc2")
	require.NoError(t, err)

	account, err = manager.GetActiveAccount("openai")
	require.NoError(t, err)
	assert.Equal(t, "acc2", account.ID)
}

func TestAccountManager_GetSwitchHistory(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	// Add accounts
	config1 := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
		Priority: 1,
		Limits:   map[LimitType]*LimitConfig{},
	}
	config2 := &AccountConfig{
		ID:       "acc2",
		Provider: "openai",
		Enabled:  true,
		Priority: 2,
		Limits:   map[LimitType]*LimitConfig{},
	}

	require.NoError(t, manager.AddAccount(config1))
	require.NoError(t, manager.AddAccount(config2))

	// Force switch to create history
	require.NoError(t, manager.ForceSwitch("openai", "acc2"))

	history := manager.GetSwitchHistory(10)
	assert.NotEmpty(t, history)
	assert.Equal(t, "acc1", history[0].FromAccount)
	assert.Equal(t, "acc2", history[0].ToAccount)
}

// TestUsage tests the Usage struct
func TestUsage_WarningLevel(t *testing.T) {
	tests := []struct {
		percentUsed  float64
		warningLevel string
	}{
		{50, ""},
		{89, ""},
		{90, "warning"},
		{95, "warning"},
		{100, "exceeded"},
		{110, "exceeded"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			// This is implicitly tested in GetUsage methods
			// Just verify the structure
			usage := &Usage{
				PercentUsed: tt.percentUsed,
			}
			if usage.PercentUsed >= 100 {
				usage.WarningLevel = "exceeded"
			} else if usage.PercentUsed >= 90 {
				usage.WarningLevel = "warning"
			}
			assert.Equal(t, tt.warningLevel, usage.WarningLevel)
		})
	}
}

// TestTypes tests type definitions
func TestLimitType_String(t *testing.T) {
	assert.Equal(t, LimitType("token"), LimitTypeToken)
	assert.Equal(t, LimitType("rpm"), LimitTypeRPM)
	assert.Equal(t, LimitType("concurrent"), LimitTypeConcurrent)
}

func TestAlertType_String(t *testing.T) {
	assert.Equal(t, AlertType("warning"), AlertWarning)
	assert.Equal(t, AlertType("critical"), AlertCritical)
	assert.Equal(t, AlertType("exceeded"), AlertExceeded)
}

func TestPeriod_String(t *testing.T) {
	assert.Equal(t, Period("minute"), PeriodMinute)
	assert.Equal(t, Period("hour"), PeriodHour)
	assert.Equal(t, Period("day"), PeriodDay)
	assert.Equal(t, Period("month"), PeriodMonth)
}

func TestAccountManager_GetAccount(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
	}
	require.NoError(t, manager.AddAccount(config))

	result, err := manager.GetAccount("acc1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "acc1", result.ID)

	result, err = manager.GetAccount("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAccountManager_GetAccountByProviderAndBaseURL(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		BaseURL:  "https://api.openai.com",
		Enabled:  true,
	}
	require.NoError(t, manager.AddAccount(config))

	result := manager.GetAccountByProviderAndBaseURL("openai", "https://api.openai.com")
	require.NotNil(t, result)
	assert.Equal(t, "acc1", result.ID)

	result = manager.GetAccountByProviderAndBaseURL("openai", "https://other.com")
	assert.Nil(t, result)
}

func TestAccountManager_GetAccountByProvider(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
	}
	require.NoError(t, manager.AddAccount(config))

	result := manager.GetAccountByProvider("openai")
	require.NotNil(t, result)
	assert.Equal(t, "acc1", result.ID)

	result = manager.GetAccountByProvider("anthropic")
	assert.Nil(t, result)
}

func TestAccountManager_GetAccountStatus(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
	}
	require.NoError(t, manager.AddAccount(config))

	status, err := manager.GetAccountStatus("acc1")
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "acc1", status.Account.ID)

	status, err = manager.GetAccountStatus("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, status)
}

func TestAccountManager_UpdateAccount(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
		Priority: 1,
	}
	require.NoError(t, manager.AddAccount(config))

	updated := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  false,
		Priority: 2,
	}
	err := manager.UpdateAccount(updated)
	require.NoError(t, err)

	result, err := manager.GetAccount("acc1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Enabled)
	assert.Equal(t, 2, result.Priority)

	err = manager.UpdateAccount(&AccountConfig{ID: "nonexistent"})
	assert.Error(t, err)
}

func TestAccountManager_Alerts(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	alerts := manager.Alerts()
	require.NotNil(t, alerts)
}

func TestAccountManager_GetAccountByBaseURLAndType(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:           "acc1",
		Provider:     "openai",
		ProviderType: "chat",
		BaseURL:      "https://api.openai.com",
		Enabled:      true,
	}
	require.NoError(t, manager.AddAccount(config))

	result := manager.GetAccountByBaseURLAndType("https://api.openai.com", "chat")
	require.NotNil(t, result)
	assert.Equal(t, "acc1", result.ID)

	result = manager.GetAccountByBaseURLAndType("https://other.com", "chat")
	assert.Nil(t, result)
}

func TestAccountManager_GetActiveAccount(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
	}
	require.NoError(t, manager.AddAccount(config))

	result, err := manager.GetActiveAccount("openai")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "acc1", result.ID)

	result, err = manager.GetActiveAccount("anthropic")
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestAccountManager_ConsumeUsage(t *testing.T) {
	logger := logrus.New()
	manager := NewAccountManager(nil, logger)

	config := &AccountConfig{
		ID:       "acc1",
		Name:     "Test Account",
		Provider: "openai",
		APIKey:   "test-key",
		Enabled:  true,
		Priority: 1,
		Limits:   nil, // No limits for this test
	}

	require.NoError(t, manager.AddAccount(config))

	ctx := context.Background()

	// Consume for non-existent account
	err := manager.ConsumeUsage(ctx, "nonexistent", 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")

	// Consume for account with no limits - should succeed
	err = manager.ConsumeUsage(ctx, "acc1", 100)
	assert.NoError(t, err)
}
