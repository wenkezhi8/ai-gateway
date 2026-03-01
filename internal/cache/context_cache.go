package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrContextTooLong  = errors.New("context too long")
)

// ContextCacheConfig holds configuration for context caching.
type ContextCacheConfig struct {
	MaxMessages      int           // Maximum messages per session
	MaxTokens        int64         // Maximum tokens before summarization
	SummaryThreshold int64         // Token threshold to trigger summarization
	DefaultTTL       time.Duration // Default session TTL
	EnableSummary    bool          // Enable auto-summarization
}

// DefaultContextCacheConfig returns default configuration.
func DefaultContextCacheConfig() ContextCacheConfig {
	return ContextCacheConfig{
		MaxMessages:      100,
		MaxTokens:        8000,
		SummaryThreshold: 6000,
		DefaultTTL:       24 * time.Hour,
		EnableSummary:    true,
	}
}

// Message represents a single message in a conversation.
type Message struct {
	Role      string          `json:"role"` // user, assistant, system
	Content   string          `json:"content"`
	Tokens    int64           `json:"tokens"`
	Timestamp time.Time       `json:"timestamp"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// SessionContext represents a conversation session's context.
type SessionContext struct {
	SessionID    string    `json:"session_id"`
	UserID       string    `json:"user_id"`
	Messages     []Message `json:"messages"`
	TotalTokens  int64     `json:"total_tokens"`
	Summary      string    `json:"summary,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	IsSummarized bool      `json:"is_summarized"`
}

// ContextCache handles caching of conversation contexts.
type ContextCache struct {
	cache      Cache
	stats      *Stats
	config     ContextCacheConfig
	summarizer Summarizer
}

// Summarizer defines the interface for context summarization.
type Summarizer interface {
	Summarize(ctx context.Context, messages []Message) (string, int64, error)
}

// NewContextCache creates a new context cache.
func NewContextCache(cache Cache, config ContextCacheConfig) *ContextCache {
	return &ContextCache{
		cache:  cache,
		stats:  GlobalStatsCollector.GetStats("context"),
		config: config,
	}
}

// SetSummarizer sets the summarizer for auto-summarization.
func (c *ContextCache) SetSummarizer(summarizer Summarizer) {
	c.summarizer = summarizer
}

// SetDefaultTTL updates the default TTL for context cache.
func (c *ContextCache) SetDefaultTTL(ttl time.Duration) {
	if ttl > 0 {
		c.config.DefaultTTL = ttl
	}
}

// sessionKey generates the cache key for a session.
func (c *ContextCache) sessionKey(sessionID string) string {
	return "session:" + sessionID
}

// GetSession retrieves a session's context.
func (c *ContextCache) GetSession(ctx context.Context, sessionID string) (*SessionContext, error) {
	start := time.Now()

	key := c.sessionKey(sessionID)
	var session SessionContext
	err := c.cache.Get(ctx, key, &session)

	latency := time.Since(start)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
			return nil, ErrSessionNotFound
		}
		c.stats.RecordError()
		return nil, err
	}

	c.stats.RecordHit(latency)

	// Calculate token savings from having context cached.
	// Each context message would need to be re-processed without cache.
	tokenSavings := session.TotalTokens
	c.stats.RecordTokensSaved(tokenSavings)

	return &session, nil
}

// CreateSession creates a new session context.
func (c *ContextCache) CreateSession(ctx context.Context, sessionID, userID string) (*SessionContext, error) {
	now := time.Now()
	session := &SessionContext{
		SessionID:   sessionID,
		UserID:      userID,
		Messages:    make([]Message, 0),
		TotalTokens: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(c.config.DefaultTTL),
	}

	key := c.sessionKey(sessionID)
	if err := c.cache.Set(ctx, key, session, c.config.DefaultTTL); err != nil {
		return nil, err
	}

	return session, nil
}

// AddMessage adds a message to a session.
//
//nolint:gocritic // Value parameter kept to preserve existing method signature.
func (c *ContextCache) AddMessage(ctx context.Context, sessionID string, message Message) error {
	session, err := c.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			// Create new session if not found.
			session, err = c.CreateSession(ctx, sessionID, "")
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Set timestamp if not set.
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Add message.
	session.Messages = append(session.Messages, message)
	session.TotalTokens += message.Tokens
	session.UpdatedAt = time.Now()

	// Check if we need to summarize.
	if c.config.EnableSummary && session.TotalTokens > c.config.SummaryThreshold && c.summarizer != nil {
		if err := c.summarizeSession(ctx, session); err != nil {
			_ = err
		}
	}

	// Check max messages limit.
	if c.config.MaxMessages > 0 && len(session.Messages) > c.config.MaxMessages {
		// Remove oldest messages (keep system message if present).
		keepCount := c.config.MaxMessages
		if len(session.Messages) > 0 && session.Messages[0].Role == "system" {
			// Keep system message + (max-1) recent messages.
			session.Messages = append(
				[]Message{session.Messages[0]},
				session.Messages[len(session.Messages)-keepCount+1:]...,
			)
		} else {
			session.Messages = session.Messages[len(session.Messages)-keepCount:]
		}
	}

	// Save updated session.
	key := c.sessionKey(sessionID)
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = c.config.DefaultTTL
		session.ExpiresAt = time.Now().Add(ttl)
	}

	return c.cache.Set(ctx, key, session, ttl)
}

// summarizeSession creates a summary of old messages to reduce context size.
func (c *ContextCache) summarizeSession(ctx context.Context, session *SessionContext) error {
	if c.summarizer == nil {
		return nil
	}

	// Keep last N messages, summarize the rest.
	keepCount := 10 // Keep last 10 messages
	if len(session.Messages) <= keepCount {
		return nil
	}

	// Messages to summarize.
	toSummarize := session.Messages[:len(session.Messages)-keepCount]

	// Generate summary.
	summary, tokens, err := c.summarizer.Summarize(ctx, toSummarize)
	if err != nil {
		return err
	}

	// Update session with summary.
	if session.Summary != "" {
		session.Summary = session.Summary + "\n\n" + summary
	} else {
		session.Summary = summary
	}

	// Calculate token savings.
	oldTokens := int64(0)
	for _, m := range toSummarize {
		oldTokens += m.Tokens
	}
	session.TotalTokens = session.TotalTokens - oldTokens + tokens

	// Keep only recent messages.
	if len(session.Messages) > 0 && session.Messages[0].Role == "system" {
		// Preserve system message.
		session.Messages = append(
			[]Message{session.Messages[0]},
			session.Messages[len(session.Messages)-keepCount:]...,
		)
	} else {
		session.Messages = session.Messages[len(session.Messages)-keepCount:]
	}

	session.IsSummarized = true
	c.stats.RecordTokensSaved(oldTokens - tokens)

	return nil
}

// UpdateSession updates an entire session.
func (c *ContextCache) UpdateSession(ctx context.Context, session *SessionContext) error {
	session.UpdatedAt = time.Now()

	key := c.sessionKey(session.SessionID)
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = c.config.DefaultTTL
		session.ExpiresAt = time.Now().Add(ttl)
	}

	return c.cache.Set(ctx, key, session, ttl)
}

// DeleteSession removes a session.
func (c *ContextCache) DeleteSession(ctx context.Context, sessionID string) error {
	key := c.sessionKey(sessionID)
	return c.cache.Delete(ctx, key)
}

// ExtendSession extends the TTL of a session.
func (c *ContextCache) ExtendSession(ctx context.Context, sessionID string, extension time.Duration) error {
	session, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ExpiresAt = time.Now().Add(extension)
	return c.UpdateSession(ctx, session)
}

// GetStats returns cache statistics.
func (c *ContextCache) GetStats() StatsSnapshot {
	return c.stats.Snapshot()
}

// GetTokenSavings returns total tokens saved from context caching.
func (c *ContextCache) GetTokenSavings() int64 {
	return c.stats.Snapshot().TokensSaved
}

// GetRecentMessages gets the most recent N messages from a session.
func (c *ContextCache) GetRecentMessages(ctx context.Context, sessionID string, count int) ([]Message, error) {
	session, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if count >= len(session.Messages) {
		return session.Messages, nil
	}

	return session.Messages[len(session.Messages)-count:], nil
}

// ClearMessages removes all messages from a session but keeps the session.
func (c *ContextCache) ClearMessages(ctx context.Context, sessionID string) error {
	session, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Messages = make([]Message, 0)
	session.TotalTokens = 0
	session.Summary = ""
	session.IsSummarized = false

	return c.UpdateSession(ctx, session)
}
