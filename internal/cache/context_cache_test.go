package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultContextCacheConfig(t *testing.T) {
	cfg := DefaultContextCacheConfig()

	assert.Equal(t, 100, cfg.MaxMessages)
	assert.Equal(t, int64(8000), cfg.MaxTokens)
	assert.Equal(t, int64(6000), cfg.SummaryThreshold)
	assert.Equal(t, 24*time.Hour, cfg.DefaultTTL)
	assert.True(t, cfg.EnableSummary)
}

func TestNewContextCache(t *testing.T) {
	cache := NewMemoryCache()
	cfg := DefaultContextCacheConfig()

	cc := NewContextCache(cache, cfg)

	require.NotNil(t, cc)
	assert.NotNil(t, cc.cache)
	assert.NotNil(t, cc.stats)
}

func TestContextCache_CreateSession(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	session, err := cc.CreateSession(ctx, "session-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, "session-1", session.SessionID)
	assert.Equal(t, "user-1", session.UserID)
	assert.NotNil(t, session.Messages)
	assert.False(t, session.CreatedAt.IsZero())
}

func TestContextCache_GetSession(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	cc.CreateSession(ctx, "session-1", "user-1")

	session, err := cc.GetSession(ctx, "session-1")

	require.NoError(t, err)
	assert.Equal(t, "session-1", session.SessionID)
}

func TestContextCache_GetSession_NotFound(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	_, err := cc.GetSession(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, ErrSessionNotFound, err)
}

func TestContextCache_AddMessage(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	cc.CreateSession(ctx, "session-1", "user-1")

	msg := Message{
		Role:    "user",
		Content: "Hello",
		Tokens:  10,
	}

	err := cc.AddMessage(ctx, "session-1", msg)
	require.NoError(t, err)

	session, _ := cc.GetSession(ctx, "session-1")
	assert.Len(t, session.Messages, 1)
	assert.Equal(t, int64(10), session.TotalTokens)
}

func TestContextCache_AddMessage_CreateNewSession(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	msg := Message{Role: "user", Content: "Hello", Tokens: 10}

	err := cc.AddMessage(ctx, "new-session", msg)
	require.NoError(t, err)

	session, err := cc.GetSession(ctx, "new-session")
	require.NoError(t, err)
	assert.Len(t, session.Messages, 1)
}

func TestContextCache_DeleteSession(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	cc.CreateSession(ctx, "session-1", "user-1")

	err := cc.DeleteSession(ctx, "session-1")
	require.NoError(t, err)

	_, err = cc.GetSession(ctx, "session-1")
	assert.Error(t, err)
}

func TestContextCache_UpdateSession(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	session, _ := cc.CreateSession(ctx, "session-1", "user-1")

	session.Messages = append(session.Messages, Message{Role: "user", Content: "Test"})
	err := cc.UpdateSession(ctx, session)

	require.NoError(t, err)

	updated, _ := cc.GetSession(ctx, "session-1")
	assert.Len(t, updated.Messages, 1)
}

func TestContextCache_ExtendSession(t *testing.T) {
	cfg := DefaultContextCacheConfig()
	cfg.DefaultTTL = time.Hour
	cc := NewContextCache(NewMemoryCache(), cfg)

	ctx := context.Background()
	cc.CreateSession(ctx, "session-1", "user-1")

	time.Sleep(10 * time.Millisecond)

	err := cc.ExtendSession(ctx, "session-1", time.Hour)
	require.NoError(t, err)

	extended, _ := cc.GetSession(ctx, "session-1")
	assert.NotNil(t, extended)
}

func TestContextCache_GetRecentMessages(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	cc.CreateSession(ctx, "session-1", "user-1")

	for i := 0; i < 5; i++ {
		cc.AddMessage(ctx, "session-1", Message{Role: "user", Content: "msg", Tokens: 1})
	}

	messages, err := cc.GetRecentMessages(ctx, "session-1", 3)
	require.NoError(t, err)
	assert.Len(t, messages, 3)
}

func TestContextCache_ClearMessages(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	ctx := context.Background()
	cc.CreateSession(ctx, "session-1", "user-1")
	cc.AddMessage(ctx, "session-1", Message{Role: "user", Content: "Test", Tokens: 10})

	err := cc.ClearMessages(ctx, "session-1")
	require.NoError(t, err)

	session, _ := cc.GetSession(ctx, "session-1")
	assert.Empty(t, session.Messages)
	assert.Equal(t, int64(0), session.TotalTokens)
}

func TestContextCache_GetStats(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	stats := cc.GetStats()
	assert.NotNil(t, stats)
}

func TestContextCache_GetTokenSavings(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	savings := cc.GetTokenSavings()
	assert.GreaterOrEqual(t, savings, int64(0))
}

func TestContextCache_SetSummarizer(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	cc.SetSummarizer(nil)
}

func TestContextCache_SessionKey(t *testing.T) {
	cc := NewContextCache(NewMemoryCache(), DefaultContextCacheConfig())

	key := cc.sessionKey("test-session")
	assert.Equal(t, "session:test-session", key)
}

func TestContextCache_MaxMessagesLimit(t *testing.T) {
	cfg := DefaultContextCacheConfig()
	cfg.MaxMessages = 5

	cc := NewContextCache(NewMemoryCache(), cfg)

	ctx := context.Background()
	cc.CreateSession(ctx, "session-1", "user-1")

	for i := 0; i < 10; i++ {
		cc.AddMessage(ctx, "session-1", Message{Role: "user", Content: "msg", Tokens: 1})
	}

	session, _ := cc.GetSession(ctx, "session-1")
	assert.LessOrEqual(t, len(session.Messages), 5)
}

func TestContextCache_ExpiredSession(t *testing.T) {
	cfg := DefaultContextCacheConfig()
	cfg.DefaultTTL = 100 * time.Millisecond

	cc := NewContextCache(NewMemoryCache(), cfg)

	ctx := context.Background()
	session, _ := cc.CreateSession(ctx, "session-1", "user-1")

	session.ExpiresAt = time.Now().Add(-time.Hour)
	cc.UpdateSession(ctx, session)

	cc.AddMessage(ctx, "session-1", Message{Role: "user", Content: "test", Tokens: 1})

	updated, _ := cc.GetSession(ctx, "session-1")
	assert.True(t, updated.ExpiresAt.After(time.Now()))
}
