package strategies

import (
	"ai-gateway/internal/router"
	"errors"
	"sync/atomic"
)

// RoundRobinStrategy implements round-robin load balancing
type RoundRobinStrategy struct {
	name   string
	counter uint64
}

// NewRoundRobinStrategy creates a new round-robin strategy
func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{
		name:    "roundrobin",
		counter: 0,
	}
}

// Name returns the strategy name
func (s *RoundRobinStrategy) Name() string {
	return s.name
}

// Select chooses a provider using round-robin algorithm
func (s *RoundRobinStrategy) Select(providers []*router.ProviderInfo, req *router.Request) (*router.ProviderInfo, error) {
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

	// Get next index using atomic increment for thread safety
	idx := atomic.AddUint64(&s.counter, 1) - 1
	selectedIdx := int(idx % uint64(len(available)))

	return available[selectedIdx], nil
}

// Reset resets the counter (useful for testing)
func (s *RoundRobinStrategy) Reset() {
	atomic.StoreUint64(&s.counter, 0)
}
