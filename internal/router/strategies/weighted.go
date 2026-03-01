//nolint:godot
package strategies

import (
	"errors"
	"math/rand"
	"sort"
	"time"

	"ai-gateway/internal/router"
)

// WeightedStrategy implements weighted routing
type WeightedStrategy struct {
	name string
	rng  *rand.Rand
}

// NewWeightedStrategy creates a new weighted strategy
func NewWeightedStrategy() *WeightedStrategy {
	return &WeightedStrategy{
		name: "weighted",
		//nolint:gosec // non-cryptographic randomness is sufficient for load balancing.
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Name returns the strategy name
func (s *WeightedStrategy) Name() string {
	return s.name
}

// Select chooses a provider based on weights
func (s *WeightedStrategy) Select(providers []*router.ProviderInfo, _ *router.Request) (*router.ProviderInfo, error) {
	if len(providers) == 0 {
		return nil, errors.New("no providers available")
	}

	// Filter available providers
	available := make([]*router.ProviderInfo, 0)
	totalWeight := 0
	for _, p := range providers {
		if p.Available() && p.Weight > 0 {
			available = append(available, p)
			totalWeight += p.Weight
		}
	}

	if len(available) == 0 {
		return nil, errors.New("no available providers with positive weight")
	}

	// If only one provider, return it
	if len(available) == 1 {
		return available[0], nil
	}

	// Sort by weight for consistent ordering (optional, for reproducibility)
	sort.Slice(available, func(i, j int) bool {
		return available[i].Name() < available[j].Name()
	})

	// Generate random value and select based on cumulative weight
	r := s.rng.Intn(totalWeight)
	cumulative := 0
	for _, p := range available {
		cumulative += p.Weight
		if r < cumulative {
			return p, nil
		}
	}

	// Fallback to last provider (should not reach here)
	return available[len(available)-1], nil
}
