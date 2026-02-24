package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRuleStore_Match_PriorityAndPattern(t *testing.T) {
	store := NewRuleStore()

	store.Create(&CacheRule{
		Pattern:  "*",
		TTL:      60,
		Priority: "low",
		Enabled:  true,
	})
	store.Create(&CacheRule{
		Pattern:  "chat:*",
		TTL:      3600,
		Priority: "high",
		Enabled:  true,
	})

	ttl, ok := store.Match("chat", "gpt-4")
	assert.True(t, ok)
	assert.Equal(t, time.Hour, ttl)
}

func TestRuleStore_Match_ModelFilter(t *testing.T) {
	store := NewRuleStore()

	store.Create(&CacheRule{
		Pattern:     "fact:*",
		ModelFilter: "gpt-4*",
		TTL:         120,
		Priority:    "medium",
		Enabled:     true,
	})

	ttl, ok := store.Match("fact", "gpt-4o-mini")
	assert.True(t, ok)
	assert.Equal(t, 2*time.Minute, ttl)

	_, ok = store.Match("fact", "claude-3")
	assert.False(t, ok)
}
