package cache

import (
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

// CacheRule defines a TTL override rule for cache entries.
//
//nolint:revive // Type name kept for compatibility with existing API payloads.
type CacheRule struct {
	ID          int    `json:"id"`
	Pattern     string `json:"pattern"`      // e.g. "chat:*" or "*:gpt-4*"
	ModelFilter string `json:"model_filter"` // optional model filter
	TTL         int    `json:"ttl"`          // TTL in seconds
	Priority    string `json:"priority"`     // high, medium, low
	Enabled     bool   `json:"enabled"`
}

// RuleStore stores cache rules in memory.
type RuleStore struct {
	mu     sync.RWMutex
	rules  map[int]*CacheRule
	nextID int
}

// NewRuleStore creates a new rule store.
func NewRuleStore() *RuleStore {
	return &RuleStore{
		rules:  make(map[int]*CacheRule),
		nextID: 1,
	}
}

var (
	globalRuleStore     *RuleStore
	globalRuleStoreOnce sync.Once
)

// GetRuleStore returns the global rule store.
func GetRuleStore() *RuleStore {
	globalRuleStoreOnce.Do(func() {
		globalRuleStore = NewRuleStore()
	})
	return globalRuleStore
}

// List returns all rules.
func (s *RuleStore) List() []*CacheRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CacheRule, 0, len(s.rules))
	for _, rule := range s.rules {
		result = append(result, rule)
	}
	return result
}

// Create adds a new rule.
func (s *RuleStore) Create(rule *CacheRule) *CacheRule {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule.ID = s.nextID
	s.nextID++
	s.rules[rule.ID] = rule
	return rule
}

// Update updates an existing rule.
func (s *RuleStore) Update(id int, update func(rule *CacheRule)) (*CacheRule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, ok := s.rules[id]
	if !ok {
		return nil, false
	}
	update(rule)
	return rule, true
}

// Delete removes a rule.
func (s *RuleStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[id]; !ok {
		return false
	}
	delete(s.rules, id)
	return true
}

// Match returns the TTL override for the first matching rule.
func (s *RuleStore) Match(taskType, model string) (time.Duration, bool) {
	s.mu.RLock()
	rules := make([]*CacheRule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, rule)
	}
	s.mu.RUnlock()

	if len(rules) == 0 {
		return 0, false
	}

	sort.SliceStable(rules, func(i, j int) bool {
		return priorityRank(rules[i].Priority) < priorityRank(rules[j].Priority)
	})

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.ModelFilter != "" && !matchPattern(rule.ModelFilter, model) {
			continue
		}
		if rule.Pattern == "" {
			return time.Duration(rule.TTL) * time.Second, true
		}
		key1 := taskType + ":" + model
		key2 := model + ":" + taskType
		if matchPattern(rule.Pattern, key1) || matchPattern(rule.Pattern, key2) || matchPattern(rule.Pattern, taskType) || matchPattern(rule.Pattern, model) {
			return time.Duration(rule.TTL) * time.Second, true
		}
	}
	return 0, false
}

func priorityRank(priority string) int {
	switch strings.ToLower(priority) {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	default:
		return 3
	}
}

func matchPattern(pattern, value string) bool {
	if pattern == "" {
		return true
	}
	matched, err := path.Match(pattern, value)
	if err != nil {
		return false
	}
	return matched
}
