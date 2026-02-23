package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Namespace and subsystem for metrics
const (
	Namespace = "ai_gateway"
	Subsystem = "api"
)

// Metrics holds all Prometheus metrics for the AI Gateway
type Metrics struct {
	// Request metrics
	RequestsTotal   *prometheus.CounterVec
	RequestsSuccess *prometheus.CounterVec
	RequestsFailed  *prometheus.CounterVec

	// Response time metrics
	ResponseTime    *prometheus.HistogramVec
	ResponseTimeP99 *prometheus.GaugeVec

	// Token consumption metrics
	TokensTotal       *prometheus.CounterVec
	TokensPrompt      *prometheus.CounterVec
	TokensCompletion  *prometheus.CounterVec
	TokenUsagePercent *prometheus.GaugeVec

	// Cache metrics
	CacheHits    *prometheus.CounterVec
	CacheMisses  *prometheus.CounterVec
	CacheHitRate *prometheus.GaugeVec

	// Provider metrics
	ProviderRequestsTotal   *prometheus.CounterVec
	ProviderRequestsSuccess *prometheus.CounterVec
	ProviderRequestsFailed  *prometheus.CounterVec
	ProviderLatency         *prometheus.HistogramVec
	ProviderFailureRate     *prometheus.GaugeVec

	// Rate limiter metrics
	RateLimitExceeded *prometheus.CounterVec

	// Active connections
	ActiveConnections *prometheus.GaugeVec

	// Multimodal metrics
	MultimodalRequests *prometheus.CounterVec
	ImageUploads       *prometheus.CounterVec
	FileUploads        *prometheus.CounterVec

	// Web search metrics
	WebSearchRequests *prometheus.CounterVec
	WebSearchLatency  *prometheus.HistogramVec
	WebSearchSuccess  *prometheus.CounterVec
	WebSearchFailures *prometheus.CounterVec

	// Deep thinking metrics
	DeepThinkRequests *prometheus.CounterVec
	DeepThinkLatency  *prometheus.HistogramVec
	ReasoningLength   *prometheus.HistogramVec

	// Routing metrics
	ModelSwitches    *prometheus.CounterVec
	CascadeFallbacks *prometheus.CounterVec
	RouteFailures    *prometheus.CounterVec
}

// Default buckets for response time histogram (in seconds)
var defaultResponseTimeBuckets = []float64{
	0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1.0, 2.0, 5.0, 10.0,
}

// Default buckets for provider latency histogram (in seconds)
var defaultProviderLatencyBuckets = []float64{
	0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 0.75, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0,
}

var (
	// Global metrics instance
	instance *Metrics
)

// Init initializes the global metrics instance
func Init() {
	if instance == nil {
		instance = NewMetrics()
	}
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	return instance
}

// NewMetrics creates a new Metrics instance with all Prometheus metrics registered
func NewMetrics() *Metrics {
	m := &Metrics{
		// Request metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "requests_total",
				Help:      "Total number of API requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestsSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "requests_success_total",
				Help:      "Total number of successful API requests",
			},
			[]string{"method", "endpoint"},
		),
		RequestsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "requests_failed_total",
				Help:      "Total number of failed API requests",
			},
			[]string{"method", "endpoint", "error_type"},
		),

		// Response time metrics
		ResponseTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "response_time_seconds",
				Help:      "Response time distribution in seconds",
				Buckets:   defaultResponseTimeBuckets,
			},
			[]string{"method", "endpoint"},
		),
		ResponseTimeP99: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "response_time_p99_seconds",
				Help:      "P99 response time in seconds",
			},
			[]string{"endpoint"},
		),

		// Token consumption metrics
		TokensTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "tokens_total",
				Help:      "Total number of tokens consumed",
			},
			[]string{"provider", "model", "type"},
		),
		TokensPrompt: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "tokens_prompt_total",
				Help:      "Total number of prompt tokens consumed",
			},
			[]string{"provider", "model"},
		),
		TokensCompletion: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "tokens_completion_total",
				Help:      "Total number of completion tokens consumed",
			},
			[]string{"provider", "model"},
		),
		TokenUsagePercent: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "token_usage_percent",
				Help:      "Token usage percentage against quota",
			},
			[]string{"user_id", "model"},
		),

		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_type"},
		),
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_type"},
		),
		CacheHitRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "cache_hit_rate",
				Help:      "Cache hit rate percentage",
			},
			[]string{"cache_type"},
		),

		// Provider metrics
		ProviderRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "provider",
				Name:      "requests_total",
				Help:      "Total number of provider requests",
			},
			[]string{"provider", "model", "endpoint"},
		),
		ProviderRequestsSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "provider",
				Name:      "requests_success_total",
				Help:      "Total number of successful provider requests",
			},
			[]string{"provider", "model"},
		),
		ProviderRequestsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "provider",
				Name:      "requests_failed_total",
				Help:      "Total number of failed provider requests",
			},
			[]string{"provider", "model", "error_type"},
		),
		ProviderLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Subsystem: "provider",
				Name:      "latency_seconds",
				Help:      "Provider request latency in seconds",
				Buckets:   defaultProviderLatencyBuckets,
			},
			[]string{"provider", "model"},
		),
		ProviderFailureRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: "provider",
				Name:      "failure_rate",
				Help:      "Provider failure rate percentage",
			},
			[]string{"provider"},
		),

		// Rate limiter metrics
		RateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "ratelimit",
				Name:      "exceeded_total",
				Help:      "Total number of rate limit exceeded events",
			},
			[]string{"user_id", "limit_type"},
		),

		// Active connections
		ActiveConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      "active_connections",
				Help:      "Number of active connections",
			},
			[]string{"endpoint"},
		),

		// Multimodal metrics
		MultimodalRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "multimodal",
				Name:      "requests_total",
				Help:      "Total number of multimodal requests",
			},
			[]string{"provider", "model"},
		),
		ImageUploads: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "multimodal",
				Name:      "image_uploads_total",
				Help:      "Total number of image uploads",
			},
			[]string{"provider"},
		),
		FileUploads: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "multimodal",
				Name:      "file_uploads_total",
				Help:      "Total number of file uploads",
			},
			[]string{"provider", "file_type"},
		),

		// Web search metrics
		WebSearchRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "websearch",
				Name:      "requests_total",
				Help:      "Total number of web search requests",
			},
			[]string{"provider"},
		),
		WebSearchLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Subsystem: "websearch",
				Name:      "latency_seconds",
				Help:      "Web search latency in seconds",
				Buckets:   []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"provider"},
		),
		WebSearchSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "websearch",
				Name:      "success_total",
				Help:      "Total number of successful web searches",
			},
			[]string{"provider"},
		),
		WebSearchFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "websearch",
				Name:      "failures_total",
				Help:      "Total number of failed web searches",
			},
			[]string{"provider", "error_type"},
		),

		// Deep thinking metrics
		DeepThinkRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "deepthink",
				Name:      "requests_total",
				Help:      "Total number of deep thinking requests",
			},
			[]string{"provider", "model"},
		),
		DeepThinkLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Subsystem: "deepthink",
				Name:      "latency_seconds",
				Help:      "Deep thinking latency in seconds",
				Buckets:   []float64{1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0},
			},
			[]string{"provider", "model"},
		),
		ReasoningLength: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Subsystem: "deepthink",
				Name:      "reasoning_length_chars",
				Help:      "Length of reasoning content in characters",
				Buckets:   []float64{100, 500, 1000, 2500, 5000, 10000, 25000},
			},
			[]string{"provider", "model"},
		),

		// Routing metrics
		ModelSwitches: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "routing",
				Name:      "model_switches_total",
				Help:      "Total number of model switches",
			},
			[]string{"from_model", "to_model", "reason"},
		),
		CascadeFallbacks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "routing",
				Name:      "cascade_fallbacks_total",
				Help:      "Total number of cascade fallbacks",
			},
			[]string{"from_tier", "to_tier"},
		),
		RouteFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "routing",
				Name:      "failures_total",
				Help:      "Total number of routing failures",
			},
			[]string{"reason"},
		),
	}

	return m
}

// RecordRequest records a request metric
func (m *Metrics) RecordRequest(method, endpoint string, statusCode int, duration time.Duration) {
	status := prometheus.Labels{
		"method":   method,
		"endpoint": endpoint,
		"status":   statusCodeClass(statusCode),
	}

	m.RequestsTotal.With(status).Inc()
	m.ResponseTime.WithLabelValues(method, endpoint).Observe(duration.Seconds())

	if statusCode >= 200 && statusCode < 400 {
		m.RequestsSuccess.WithLabelValues(method, endpoint).Inc()
	} else {
		errorType := errorTypeFromStatus(statusCode)
		m.RequestsFailed.WithLabelValues(method, endpoint, errorType).Inc()
	}
}

// RecordProviderRequest records a provider request metric
func (m *Metrics) RecordProviderRequest(provider, model, endpoint string, success bool, duration time.Duration, err error) {
	m.ProviderRequestsTotal.WithLabelValues(provider, model, endpoint).Inc()
	m.ProviderLatency.WithLabelValues(provider, model).Observe(duration.Seconds())

	if success {
		m.ProviderRequestsSuccess.WithLabelValues(provider, model).Inc()
	} else {
		errorType := "unknown"
		if err != nil {
			errorType = categorizeError(err)
		}
		m.ProviderRequestsFailed.WithLabelValues(provider, model, errorType).Inc()
	}
}

// RecordTokenUsage records token consumption metrics
func (m *Metrics) RecordTokenUsage(provider, model string, promptTokens, completionTokens int) {
	m.TokensPrompt.WithLabelValues(provider, model).Add(float64(promptTokens))
	m.TokensCompletion.WithLabelValues(provider, model).Add(float64(completionTokens))
	m.TokensTotal.WithLabelValues(provider, model, "prompt").Add(float64(promptTokens))
	m.TokensTotal.WithLabelValues(provider, model, "completion").Add(float64(completionTokens))
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(cacheType string) {
	m.CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(cacheType string) {
	m.CacheMisses.WithLabelValues(cacheType).Inc()
}

// SetCacheHitRate sets the cache hit rate
func (m *Metrics) SetCacheHitRate(cacheType string, rate float64) {
	m.CacheHitRate.WithLabelValues(cacheType).Set(rate)
}

// RecordMultimodalRequest records a multimodal request
func (m *Metrics) RecordMultimodalRequest(provider, model string) {
	m.MultimodalRequests.WithLabelValues(provider, model).Inc()
}

// RecordImageUpload records an image upload
func (m *Metrics) RecordImageUpload(provider string) {
	m.ImageUploads.WithLabelValues(provider).Inc()
}

// RecordFileUpload records a file upload
func (m *Metrics) RecordFileUpload(provider, fileType string) {
	m.FileUploads.WithLabelValues(provider, fileType).Inc()
}

// RecordWebSearch records a web search request
func (m *Metrics) RecordWebSearch(provider string, duration time.Duration, success bool, err error) {
	m.WebSearchRequests.WithLabelValues(provider).Inc()
	m.WebSearchLatency.WithLabelValues(provider).Observe(duration.Seconds())

	if success {
		m.WebSearchSuccess.WithLabelValues(provider).Inc()
	} else {
		errorType := "unknown"
		if err != nil {
			errorType = categorizeError(err)
		}
		m.WebSearchFailures.WithLabelValues(provider, errorType).Inc()
	}
}

// RecordDeepThink records a deep thinking request
func (m *Metrics) RecordDeepThink(provider, model string, duration time.Duration, reasoningLength int) {
	m.DeepThinkRequests.WithLabelValues(provider, model).Inc()
	m.DeepThinkLatency.WithLabelValues(provider, model).Observe(duration.Seconds())
	m.ReasoningLength.WithLabelValues(provider, model).Observe(float64(reasoningLength))
}

// RecordModelSwitch records a model switch event
func (m *Metrics) RecordModelSwitch(fromModel, toModel, reason string) {
	m.ModelSwitches.WithLabelValues(fromModel, toModel, reason).Inc()
}

// RecordCascadeFallback records a cascade fallback event
func (m *Metrics) RecordCascadeFallback(fromTier, toTier string) {
	m.CascadeFallbacks.WithLabelValues(fromTier, toTier).Inc()
}

// RecordRouteFailure records a routing failure
func (m *Metrics) RecordRouteFailure(reason string) {
	m.RouteFailures.WithLabelValues(reason).Inc()
}

// RecordRateLimitExceeded records a rate limit exceeded event
func (m *Metrics) RecordRateLimitExceeded(userID, limitType string) {
	m.RateLimitExceeded.WithLabelValues(userID, limitType).Inc()
}

// SetTokenUsagePercent sets the token usage percentage
func (m *Metrics) SetTokenUsagePercent(userID, model string, percent float64) {
	m.TokenUsagePercent.WithLabelValues(userID, model).Set(percent)
}

// SetProviderFailureRate sets the provider failure rate
func (m *Metrics) SetProviderFailureRate(provider string, rate float64) {
	m.ProviderFailureRate.WithLabelValues(provider).Set(rate)
}

// PrometheusHandler returns an HTTP handler for Prometheus metrics
func (m *Metrics) PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// IncActiveConnections increments active connections
func (m *Metrics) IncActiveConnections(endpoint string) {
	m.ActiveConnections.WithLabelValues(endpoint).Inc()
}

// DecActiveConnections decrements active connections
func (m *Metrics) DecActiveConnections(endpoint string) {
	m.ActiveConnections.WithLabelValues(endpoint).Dec()
}

// Helper functions

func statusCodeClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

func errorTypeFromStatus(statusCode int) string {
	switch {
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}

func categorizeError(err error) string {
	errStr := err.Error()
	switch {
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "connection refused"):
		return "connection_refused"
	case contains(errStr, "context canceled"):
		return "canceled"
	case contains(errStr, "rate limit"):
		return "rate_limited"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
