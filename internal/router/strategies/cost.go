//nolint:godot
package strategies

import (
	"errors"
	"sort"

	"ai-gateway/internal/router"
)

// CostStrategy implements cost-optimized routing
type CostStrategy struct {
	name string
}

// NewCostStrategy creates a new cost-optimized strategy
func NewCostStrategy() *CostStrategy {
	return &CostStrategy{
		name: "cost",
	}
}

// Name returns the strategy name
func (s *CostStrategy) Name() string {
	return s.name
}

// Select chooses a provider with the lowest cost
func (s *CostStrategy) Select(providers []*router.ProviderInfo, _ *router.Request) (*router.ProviderInfo, error) {
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

	// Sort by cost (ascending)
	sort.Slice(available, func(i, j int) bool {
		// If costs are equal, prefer higher remaining quota
		if available[i].Cost == available[j].Cost {
			return available[i].QuotaRemaining() > available[j].QuotaRemaining()
		}
		return available[i].Cost < available[j].Cost
	})

	return available[0], nil
}
