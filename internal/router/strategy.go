//nolint:godot
package router

import (
	"ai-gateway/internal/provider"
)

// Request represents a routing request with context
type Request struct {
	Model      string                 `json:"model"`
	UserID     string                 `json:"user_id"`
	Messages   []Message              `json:"messages"`
	Extra      map[string]interface{} `json:"-"`
	TokensUsed int                    `json:"tokens_used"`
	Priority   int                    `json:"priority"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ProviderInfo extends Provider with routing metadata
type ProviderInfo struct {
	provider.Provider
	Weight     int     `json:"weight"`
	Priority   int     `json:"priority"`
	Cost       float64 `json:"cost"`
	IsPrimary  bool    `json:"is_primary"`
	Healthy    bool    `json:"healthy"`
	QuotaUsed  int64   `json:"quota_used"`
	QuotaLimit int64   `json:"quota_limit"`
	LastError  error   `json:"-"`
	LastUsed   int64   `json:"last_used"` // Unix timestamp
}

// Available returns true if the provider is available for routing
func (p *ProviderInfo) Available() bool {
	return p.IsEnabled() && p.Healthy &&
		(p.QuotaLimit == 0 || p.QuotaUsed < p.QuotaLimit)
}

// QuotaRemaining returns the remaining quota
func (p *ProviderInfo) QuotaRemaining() int64 {
	if p.QuotaLimit == 0 {
		return -1 // unlimited
	}
	remaining := p.QuotaLimit - p.QuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Strategy defines the interface for routing strategies
type Strategy interface {
	// Name returns the strategy name
	Name() string

	// Select chooses a provider from the available providers
	Select(providers []*ProviderInfo, req *Request) (*ProviderInfo, error)
}

// StrategyType defines the available strategy types
type StrategyType string

const (
	// StrategyFailover uses primary/backup mode
	StrategyFailover StrategyType = "failover"
	// StrategyRoundRobin uses round-robin load balancing
	StrategyRoundRobin StrategyType = "roundrobin"
	// StrategyCostOptimized selects the lowest cost provider
	StrategyCostOptimized StrategyType = "cost"
	// StrategyWeighted uses weighted distribution
	StrategyWeighted StrategyType = "weighted"
)

// ParseStrategyType converts a string to StrategyType
func ParseStrategyType(s string) StrategyType {
	switch s {
	case "failover":
		return StrategyFailover
	case "roundrobin":
		return StrategyRoundRobin
	case "cost":
		return StrategyCostOptimized
	case "weighted":
		return StrategyWeighted
	default:
		return StrategyRoundRobin // default
	}
}
