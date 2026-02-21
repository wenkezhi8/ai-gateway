package limiter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore implements Store interface for testing
type mockStore struct {
	data     map[string]int64
	err      error
	incrErr  error
	expiry   map[string]time.Duration
	incrCall int
}

func newMockStore() *mockStore {
	return &mockStore{
		data:   make(map[string]int64),
		expiry: make(map[string]time.Duration),
	}
}

func (m *mockStore) Get(ctx context.Context, key string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	val, ok := m.data[key]
	if !ok {
		return 0, errors.New("key not found")
	}
	return val, nil
}

func (m *mockStore) Incr(ctx context.Context, key string) (int64, error) {
	if m.incrErr != nil {
		return 0, m.incrErr
	}
	m.incrCall++
	m.data[key]++
	return m.data[key], nil
}

func (m *mockStore) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.expiry[key] = expiration
	return nil
}

func TestUsageTracker_New(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	assert.NotNil(t, tracker)
}

func TestUsageTracker_IncrementUsage(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	ctx := context.Background()

	err := tracker.IncrementUsage(ctx, "user1", "openai", 100)
	require.NoError(t, err)

	// Verify the key was created
	key := tracker.buildKey("user1", "openai")
	_, ok := store.data[key]
	assert.True(t, ok)
}

func TestUsageTracker_IncrementUsage_Multiple(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	ctx := context.Background()

	// First increment
	err := tracker.IncrementUsage(ctx, "user1", "openai", 100)
	require.NoError(t, err)

	// Second increment
	err = tracker.IncrementUsage(ctx, "user1", "openai", 50)
	require.NoError(t, err)

	// Verify counter increased
	usage, err := tracker.GetUsage(ctx, "user1", "openai")
	require.NoError(t, err)
	assert.Equal(t, int64(2), usage) // Two increments
}

func TestUsageTracker_GetUsage(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	ctx := context.Background()

	// Get usage for non-existent key - should return 0, not error
	usage, err := tracker.GetUsage(ctx, "user1", "openai")
	require.NoError(t, err)
	assert.Equal(t, int64(0), usage)

	// Increment usage
	err = tracker.IncrementUsage(ctx, "user1", "openai", 100)
	require.NoError(t, err)

	// Get usage
	usage, err = tracker.GetUsage(ctx, "user1", "openai")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, usage, int64(1))
}

func TestUsageTracker_IncrementUsage_StoreError(t *testing.T) {
	store := newMockStore()
	store.incrErr = errors.New("store error")
	tracker := NewLegacyUsageTracker(store)
	ctx := context.Background()

	err := tracker.IncrementUsage(ctx, "user1", "openai", 100)
	assert.Error(t, err)
}

func TestUsageTracker_BuildKey(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)

	key := tracker.buildKey("user123", "anthropic")

	// Key should contain provider and user
	assert.Contains(t, key, "usage:")
	assert.Contains(t, key, "anthropic")
	assert.Contains(t, key, "user123")

	// Key should contain date
	now := time.Now()
	assert.Contains(t, key, now.Format("2006-01-02"))
}

func TestUsageTracker_DifferentProviders(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	ctx := context.Background()

	// Increment for different providers
	err := tracker.IncrementUsage(ctx, "user1", "openai", 100)
	require.NoError(t, err)

	err = tracker.IncrementUsage(ctx, "user1", "anthropic", 200)
	require.NoError(t, err)

	// Keys should be different
	openaiKey := tracker.buildKey("user1", "openai")
	anthropicKey := tracker.buildKey("user1", "anthropic")
	assert.NotEqual(t, openaiKey, anthropicKey)
}

func TestUsageTracker_DifferentUsers(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	ctx := context.Background()

	// Increment for different users
	err := tracker.IncrementUsage(ctx, "user1", "openai", 100)
	require.NoError(t, err)

	err = tracker.IncrementUsage(ctx, "user2", "openai", 200)
	require.NoError(t, err)

	// Keys should be different
	user1Key := tracker.buildKey("user1", "openai")
	user2Key := tracker.buildKey("user2", "openai")
	assert.NotEqual(t, user1Key, user2Key)
}

func TestUsageTracker_ExpirySet(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	ctx := context.Background()

	// First increment should set expiry
	err := tracker.IncrementUsage(ctx, "user1", "openai", 100)
	require.NoError(t, err)

	key := tracker.buildKey("user1", "openai")
	_, hasExpiry := store.expiry[key]
	assert.True(t, hasExpiry)
}

func TestPeriodDuration(t *testing.T) {
	tests := []struct {
		period   Period
		expected time.Duration
		hasError bool
	}{
		{PeriodMinute, time.Minute, false},
		{PeriodHour, time.Hour, false},
		{PeriodDay, 24 * time.Hour, false},
		{PeriodMonth, 30 * 24 * time.Hour, false},
		{Period("invalid"), 0, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.period), func(t *testing.T) {
			duration, err := PeriodDuration(tt.period)
			if tt.hasError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidPeriod)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, duration)
			}
		})
	}
}
