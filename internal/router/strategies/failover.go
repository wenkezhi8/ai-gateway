package strategies

import (
	"ai-gateway/internal/router"
	"errors"
	"sort"
)

// FailoverStrategy implements primary/backup routing
type FailoverStrategy struct {
	name string
}

// NewFailoverStrategy creates a new failover strategy
func NewFailoverStrategy() *FailoverStrategy {
	return &FailoverStrategy{
		name: "failover",
	}
}

// Name returns the strategy name
func (s *FailoverStrategy) Name() string {
	return s.name
}

// Select chooses a provider using failover logic
// Priority order: Primary > Secondary by priority value (lower = higher priority)
func (s *FailoverStrategy) Select(providers []*router.ProviderInfo, req *router.Request) (*router.ProviderInfo, error) {
	if len(providers) == 0 {
		return nil, errors.New("no providers available")
	}

	// Filter available providers
	available := make([]*router.ProviderInfo, 0)
	for _, p := range providers {
		if p.Available() {
			available = append(available, p)
		}
	}

	if len(available) == 0 {
		return nil, errors.New("no available providers")
	}

	// Sort by: Primary first, then by priority (ascending)
	sort.Slice(available, func(i, j int) bool {
		// Primary providers come first
		if available[i].IsPrimary != available[j].IsPrimary {
			return available[i].IsPrimary
		}
		// Then sort by priority (lower value = higher priority)
		return available[i].Priority < available[j].Priority
	})

	// Return the first (highest priority) provider
	return available[0], nil
}
