//nolint:revive
package limiter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuotaManager_New(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.configs)
}

func TestQuotaManager_SetAndGetQuota(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)

	config := &QuotaConfig{
		UserID:       "user1",
		DailyLimit:   1000,
		MonthlyLimit: 30000,
		TokenLimit:   100000,
	}

	manager.SetQuota(config)

	retrieved, ok := manager.GetQuota("user1")
	require.True(t, ok)
	assert.Equal(t, config.UserID, retrieved.UserID)
	assert.Equal(t, config.DailyLimit, retrieved.DailyLimit)
}

func TestQuotaManager_GetQuota_NotFound(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)

	_, ok := manager.GetQuota("non-existent")
	assert.False(t, ok)
}

func TestQuotaManager_CheckQuota_NoConfig(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)
	ctx := context.Background()

	// No quota configured should allow request
	allowed, err := manager.CheckQuota(ctx, "user1", "openai")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestQuotaManager_CheckQuota_WithinLimit(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)
	ctx := context.Background()

	// Set quota
	config := &QuotaConfig{
		UserID:     "user1",
		DailyLimit: 1000,
	}
	manager.SetQuota(config)

	// Initial usage is 0, should be allowed
	allowed, err := manager.CheckQuota(ctx, "user1", "openai")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestQuotaManager_CheckQuota_Exceeded(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)
	ctx := context.Background()

	// Set quota
	config := &QuotaConfig{
		UserID:     "user1",
		DailyLimit: 1, // Very low limit
	}
	manager.SetQuota(config)

	// Increment usage
	err := tracker.IncrementUsage(ctx, "user1", "openai", 100)
	require.NoError(t, err)

	// Now check - should exceed daily limit
	allowed, err := manager.CheckQuota(ctx, "user1", "openai")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestQuotaManager_ConsumeQuota(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)
	ctx := context.Background()

	config := &QuotaConfig{
		UserID:     "user1",
		DailyLimit: 1000,
	}
	manager.SetQuota(config)

	err := manager.ConsumeQuota(ctx, "user1", "openai", 100)
	require.NoError(t, err)
}

func TestQuotaManager_ConsumeQuota_Exceeded(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)
	ctx := context.Background()

	config := &QuotaConfig{
		UserID:     "user1",
		DailyLimit: 1,
	}
	manager.SetQuota(config)

	// First consume to exceed limit
	err := tracker.IncrementUsage(ctx, "user1", "openai", 100)
	require.NoError(t, err)

	// Second consume should fail
	err = manager.ConsumeQuota(ctx, "user1", "openai", 50)
	assert.ErrorIs(t, err, ErrQuotaExceeded)
}

func TestQuotaManager_ProviderLimits(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)
	ctx := context.Background()

	config := &QuotaConfig{
		UserID:     "user1",
		DailyLimit: 1000,
		Providers: map[string]int64{
			"openai": 5, // Small limit for testing
		},
	}
	manager.SetQuota(config)

	// Increment to exceed provider limit (Incr adds 1 each call)
	for i := 0; i < 5; i++ {
		err := tracker.IncrementUsage(ctx, "user1", "openai", 1)
		require.NoError(t, err)
	}

	// Should be blocked by provider limit
	allowed, err := manager.CheckQuota(ctx, "user1", "openai")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestQuotaManager_UpdateQuota(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)

	// Set initial quota
	config1 := &QuotaConfig{
		UserID:     "user1",
		DailyLimit: 1000,
	}
	manager.SetQuota(config1)

	// Update quota
	config2 := &QuotaConfig{
		UserID:     "user1",
		DailyLimit: 5000,
	}
	manager.SetQuota(config2)

	// Verify updated
	retrieved, ok := manager.GetQuota("user1")
	require.True(t, ok)
	assert.Equal(t, int64(5000), retrieved.DailyLimit)
}

func TestQuotaManager_Concurrent(t *testing.T) {
	store := newMockStore()
	tracker := NewLegacyUsageTracker(store)
	manager := NewQuotaManager(tracker)

	// Set initial config
	config := &QuotaConfig{
		UserID:     "user1",
		DailyLimit: 10000,
	}
	manager.SetQuota(config)

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = manager.GetQuota("user1")
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				newConfig := &QuotaConfig{
					UserID:     "user1",
					DailyLimit: 1000 + int64(j),
				}
				manager.SetQuota(newConfig)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}

func TestQuotaConfig_Fields(t *testing.T) {
	config := &QuotaConfig{
		UserID:       "test-user",
		DailyLimit:   1000,
		MonthlyLimit: 30000,
		TokenLimit:   100000,
		Providers: map[string]int64{
			"openai":    500,
			"anthropic": 500,
		},
	}

	assert.Equal(t, "test-user", config.UserID)
	assert.Equal(t, int64(1000), config.DailyLimit)
	assert.Equal(t, int64(30000), config.MonthlyLimit)
	assert.Equal(t, int64(100000), config.TokenLimit)
	assert.Len(t, config.Providers, 2)
}
