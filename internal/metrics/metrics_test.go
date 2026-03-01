package metrics

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatusCodeClass(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{300, "3xx"},
		{304, "3xx"},
		{400, "4xx"},
		{404, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{502, "5xx"},
		{503, "5xx"},
		{100, "unknown"},
		{0, "unknown"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := statusCodeClass(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorTypeFromStatus(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{200, "unknown"},
		{400, "client_error"},
		{404, "client_error"},
		{499, "client_error"},
		{500, "server_error"},
		{502, "server_error"},
		{503, "server_error"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := errorTypeFromStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{errors.New("request timeout"), "timeout"},
		{errors.New("connection refused"), "connection_refused"},
		{errors.New("context canceled"), "canceled"},
		{errors.New("rate limit exceeded"), "rate_limited"},
		{errors.New("unknown error"), "unknown"},
		{errors.New("something else"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			result := categorizeError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"timeout error", "timeout", true},
		{"", "", true},
		{"a", "a", true},
		{"a", "b", false},
		{"", "a", false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewMetrics(t *testing.T) {
	// Initialize once for all tests
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()
	assert.NotNil(t, m)
	assert.NotNil(t, m.RequestsTotal)
	assert.NotNil(t, m.RequestsSuccess)
	assert.NotNil(t, m.RequestsFailed)
	assert.NotNil(t, m.ResponseTime)
	assert.NotNil(t, m.TokensTotal)
	assert.NotNil(t, m.CacheHits)
	assert.NotNil(t, m.CacheMisses)
	assert.NotNil(t, m.ProviderRequestsTotal)
	assert.NotNil(t, m.RateLimitExceeded)
	assert.NotNil(t, m.ActiveConnections)
}

func TestMetrics_RecordRequest(_ *testing.T) {
	// Initialize global metrics if not already done
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	// Record successful request
	m.RecordRequest("GET", "/health", 200, 10*time.Millisecond)

	// Record failed request
	m.RecordRequest("POST", "/api/chat", 500, 100*time.Millisecond)

	// Record client error
	m.RecordRequest("GET", "/api/invalid", 404, 5*time.Millisecond)

	// No assertion - just verify no panic
}

func TestMetrics_RecordProviderRequest_Success(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.RecordProviderRequest("openai", "gpt-4", "/chat", true, 500*time.Millisecond, nil)

	// No assertion - just verify no panic
}

func TestMetrics_RecordProviderRequest_Failed(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	err := errors.New("timeout error")
	m.RecordProviderRequest("anthropic", "claude-3", "/chat", false, 2*time.Second, err)

	// No assertion - just verify no panic
}

func TestMetrics_RecordTokenUsage(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.RecordTokenUsage("openai", "gpt-4", 100, 50)

	// No assertion - just verify no panic
}

func TestMetrics_RecordCacheHit(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.RecordCacheHit("response")
	m.RecordCacheHit("context")

	// No assertion - just verify no panic
}

func TestMetrics_RecordCacheMiss(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.RecordCacheMiss("response")
	m.RecordCacheMiss("context")

	// No assertion - just verify no panic
}

func TestMetrics_SetCacheHitRate(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.SetCacheHitRate("response", 0.85)
	m.SetCacheHitRate("context", 0.92)

	// No assertion - just verify no panic
}

func TestMetrics_RecordRateLimitExceeded(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.RecordRateLimitExceeded("user-123", "rpm")
	m.RecordRateLimitExceeded("user-456", "token")

	// No assertion - just verify no panic
}

func TestMetrics_SetTokenUsagePercent(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.SetTokenUsagePercent("user-123", "gpt-4", 75.5)
	m.SetTokenUsagePercent("user-456", "claude-3", 90.0)

	// No assertion - just verify no panic
}

func TestMetrics_SetProviderFailureRate(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.SetProviderFailureRate("openai", 0.02)
	m.SetProviderFailureRate("anthropic", 0.01)

	// No assertion - just verify no panic
}

func TestMetrics_ActiveConnections(_ *testing.T) {
	if GetMetrics() == nil {
		Init()
	}
	m := GetMetrics()

	m.IncActiveConnections("/chat")
	m.IncActiveConnections("/chat")
	m.DecActiveConnections("/chat")

	// No assertion - just verify no panic
}

func TestInit(t *testing.T) {
	Init()
	assert.NotNil(t, GetMetrics())
}

func TestDefaultBuckets(t *testing.T) {
	assert.NotEmpty(t, defaultResponseTimeBuckets)
	assert.NotEmpty(t, defaultProviderLatencyBuckets)

	// Verify buckets are sorted
	for i := 1; i < len(defaultResponseTimeBuckets); i++ {
		assert.Greater(t, defaultResponseTimeBuckets[i], defaultResponseTimeBuckets[i-1])
	}

	for i := 1; i < len(defaultProviderLatencyBuckets); i++ {
		assert.Greater(t, defaultProviderLatencyBuckets[i], defaultProviderLatencyBuckets[i-1])
	}
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "ai_gateway", Namespace)
	assert.Equal(t, "api", Subsystem)
}
