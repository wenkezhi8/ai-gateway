package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAccountRequest_Fields(t *testing.T) {
	enabled := true
	priority := 1
	req := AccountRequest{
		ID:                "acc-1",
		Name:              "Test Account",
		Provider:          "openai",
		APIKey:            "sk-test",
		BaseURL:           "https://api.openai.com",
		Enabled:           &enabled,
		Priority:          priority,
		CodingPlanEnabled: &enabled,
	}

	assert.Equal(t, "acc-1", req.ID)
	assert.Equal(t, "Test Account", req.Name)
	assert.Equal(t, "openai", req.Provider)
	assert.Equal(t, "sk-test", req.APIKey)
	assert.Equal(t, "https://api.openai.com", req.BaseURL)
	assert.True(t, *req.Enabled)
	assert.Equal(t, 1, req.Priority)
	assert.True(t, *req.CodingPlanEnabled)
}

func TestLimitConfig_Fields(t *testing.T) {
	cfg := LimitConfig{
		Type:    "token",
		Period:  "day",
		Limit:   100000,
		Warning: 0.8,
	}

	assert.Equal(t, "token", cfg.Type)
	assert.Equal(t, "day", cfg.Period)
	assert.Equal(t, int64(100000), cfg.Limit)
	assert.Equal(t, 0.8, cfg.Warning)
}

func TestAccountResponse_Fields(t *testing.T) {
	resp := AccountResponse{
		ID:                "acc-1",
		Name:              "Test Account",
		Provider:          "openai",
		APIKey:            "sk-test",
		BaseURL:           "https://api.openai.com",
		Enabled:           true,
		Priority:          1,
		IsActive:          true,
		LastSwitch:        time.Now(),
		PlanType:          "pro",
		CodingPlanEnabled: true,
	}

	assert.Equal(t, "acc-1", resp.ID)
	assert.Equal(t, "Test Account", resp.Name)
	assert.Equal(t, "openai", resp.Provider)
	assert.True(t, resp.Enabled)
	assert.Equal(t, 1, resp.Priority)
	assert.True(t, resp.IsActive)
	assert.Equal(t, "pro", resp.PlanType)
	assert.True(t, resp.CodingPlanEnabled)
}

func TestAccountUsageResponse_Fields(t *testing.T) {
	usage := AccountUsageResponse{
		TokensUsed:    50000,
		TokenLimit:    100000,
		TokenPercent:  50.0,
		RequestsCount: 1000,
		RPM:           100,
		RPMLimit:      1000,
		WarningLevel:  "normal",
		Hour5Used:     10000,
		Hour5Limit:    20000,
		Hour5Percent:  50.0,
		WeekUsed:      50000,
		WeekLimit:     100000,
		WeekPercent:   50.0,
		MonthUsed:     200000,
		MonthLimit:    500000,
		MonthPercent:  40.0,
	}

	assert.Equal(t, int64(50000), usage.TokensUsed)
	assert.Equal(t, int64(100000), usage.TokenLimit)
	assert.Equal(t, 50.0, usage.TokenPercent)
	assert.Equal(t, int64(1000), usage.RequestsCount)
	assert.Equal(t, 100, usage.RPM)
	assert.Equal(t, 1000, usage.RPMLimit)
	assert.Equal(t, "normal", usage.WarningLevel)
}

func TestProviderRequest_Fields(t *testing.T) {
	req := ProviderRequest{
		Name:    "openai",
		APIKey:  "sk-test",
		BaseURL: "https://api.openai.com",
		Models:  []string{"gpt-4", "gpt-3.5-turbo"},
		Enabled: true,
		Extra:   map[string]interface{}{"org": "test"},
	}

	assert.Equal(t, "openai", req.Name)
	assert.Equal(t, "sk-test", req.APIKey)
	assert.Equal(t, "https://api.openai.com", req.BaseURL)
	assert.Len(t, req.Models, 2)
	assert.True(t, req.Enabled)
}

func TestProviderResponse_Fields(t *testing.T) {
	resp := ProviderResponse{
		Name:         "openai",
		BaseURL:      "https://api.openai.com",
		Models:       []string{"gpt-4"},
		Enabled:      true,
		Healthy:      true,
		AccountCount: 5,
		LastCheck:    time.Now(),
	}

	assert.Equal(t, "openai", resp.Name)
	assert.True(t, resp.Enabled)
	assert.True(t, resp.Healthy)
	assert.Equal(t, 5, resp.AccountCount)
}

func TestProviderTestResult_Fields(t *testing.T) {
	result := ProviderTestResult{
		Success:      true,
		Message:      "Connection successful",
		ResponseTime: 100,
		Timestamp:    time.Now(),
	}

	assert.True(t, result.Success)
	assert.Equal(t, "Connection successful", result.Message)
	assert.Equal(t, int64(100), result.ResponseTime)
}

func TestRoutingConfig_Fields(t *testing.T) {
	cfg := RoutingConfig{
		DefaultStrategy: "auto",
		ModelStrategies: map[string]string{
			"gpt-4": "quality",
		},
		ProviderWeights: map[string]int{
			"openai": 10,
		},
		FailoverConfig: &FailoverConfig{
			MaxRetries:       3,
			RetryDelayMs:     1000,
			HealthCheckSec:   30,
			CircuitBreaker:   true,
			FailureThreshold: 5,
		},
	}

	assert.Equal(t, "auto", cfg.DefaultStrategy)
	assert.Equal(t, "quality", cfg.ModelStrategies["gpt-4"])
	assert.Equal(t, 10, cfg.ProviderWeights["openai"])
	assert.NotNil(t, cfg.FailoverConfig)
}

func TestFailoverConfig_Fields(t *testing.T) {
	cfg := FailoverConfig{
		MaxRetries:       3,
		RetryDelayMs:     1000,
		HealthCheckSec:   30,
		CircuitBreaker:   true,
		FailureThreshold: 5,
	}

	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 1000, cfg.RetryDelayMs)
	assert.Equal(t, 30, cfg.HealthCheckSec)
	assert.True(t, cfg.CircuitBreaker)
	assert.Equal(t, 5, cfg.FailureThreshold)
}

func TestCacheStatsResponse_Fields(t *testing.T) {
	resp := CacheStatsResponse{
		RequestCache:  CacheStatDetail{Hits: 100, Misses: 50, HitRate: 66.6},
		ContextCache:  CacheStatDetail{Hits: 80, Misses: 20, HitRate: 80.0},
		RouteCache:    CacheStatDetail{Hits: 90, Misses: 10, HitRate: 90.0},
		UsageCache:    CacheStatDetail{Hits: 70, Misses: 30, HitRate: 70.0},
		ResponseCache: CacheStatDetail{Hits: 60, Misses: 40, HitRate: 60.0},
		TokenSavings:  10000,
		RedisHits:     200,
		RedisMisses:   50,
		RedisHitRate:  0.8,
	}

	assert.Equal(t, int64(100), resp.RequestCache.Hits)
	assert.Equal(t, int64(50), resp.RequestCache.Misses)
	assert.Equal(t, 66.6, resp.RequestCache.HitRate)
	assert.Equal(t, int64(10000), resp.TokenSavings)
	assert.Equal(t, int64(200), resp.RedisHits)
	assert.Equal(t, int64(50), resp.RedisMisses)
	assert.Equal(t, 0.8, resp.RedisHitRate)
}

func TestCacheStatDetail_Fields(t *testing.T) {
	detail := CacheStatDetail{
		Hits:         1000,
		Misses:       200,
		HitRate:      83.3,
		SizeBytes:    1024000,
		Entries:      500,
		AvgLatencyMs: 10,
		MaxSize:      10485760,
		Evictions:    50,
	}

	assert.Equal(t, int64(1000), detail.Hits)
	assert.Equal(t, int64(200), detail.Misses)
	assert.Equal(t, 83.3, detail.HitRate)
	assert.Equal(t, int64(1024000), detail.SizeBytes)
	assert.Equal(t, int64(500), detail.Entries)
}

func TestCacheConfigRequest_Fields(t *testing.T) {
	enabled := true
	threshold := 0.8
	ttl := 3600
	entries := 10000
	req := CacheConfigRequest{
		Enabled:             &enabled,
		Strategy:            strp("semantic"),
		SimilarityThreshold: &threshold,
		DefaultTTLSeconds:   &ttl,
		MaxEntries:          &entries,
		EvictionPolicy:      strp("lru"),
		RequestTTL:          &ttl,
		ContextTTL:          &ttl,
		RouteTTL:            &ttl,
	}

	assert.True(t, *req.Enabled)
	assert.Equal(t, "semantic", *req.Strategy)
	assert.Equal(t, 0.8, *req.SimilarityThreshold)
	assert.Equal(t, 3600, *req.DefaultTTLSeconds)
}

func strp(s string) *string {
	return &s
}

func TestDashboardStats_Fields(t *testing.T) {
	stats := DashboardStats{
		TotalRequests:   10000,
		RequestsToday:   1000,
		SuccessRate:     95.5,
		AvgLatencyMs:    500,
		TotalTokens:     5000000,
		ActiveAccounts:  10,
		ActiveProviders: 5,
		CacheHitRate:    70.0,
		ProviderStats: []ProviderStat{
			{Name: "openai", Requests: 5000, Tokens: 2500000, SuccessRate: 96.0, AvgLatency: 450},
		},
		TopModels: []ModelStat{
			{Name: "gpt-4", Requests: 3000, Tokens: 2000000},
		},
	}

	assert.Equal(t, int64(10000), stats.TotalRequests)
	assert.Equal(t, int64(1000), stats.RequestsToday)
	assert.Equal(t, 95.5, stats.SuccessRate)
	assert.Equal(t, 10, stats.ActiveAccounts)
	assert.Equal(t, 5, stats.ActiveProviders)
	assert.Len(t, stats.ProviderStats, 1)
	assert.Len(t, stats.TopModels, 1)
}

func TestProviderStat_Fields(t *testing.T) {
	stat := ProviderStat{
		Name:        "openai",
		Requests:    5000,
		Tokens:      2500000,
		SuccessRate: 96.0,
		AvgLatency:  450,
	}

	assert.Equal(t, "openai", stat.Name)
	assert.Equal(t, int64(5000), stat.Requests)
	assert.Equal(t, int64(2500000), stat.Tokens)
	assert.Equal(t, 96.0, stat.SuccessRate)
}

func TestModelStat_Fields(t *testing.T) {
	stat := ModelStat{
		Name:     "gpt-4",
		Requests: 3000,
		Tokens:   2000000,
	}

	assert.Equal(t, "gpt-4", stat.Name)
	assert.Equal(t, int64(3000), stat.Requests)
	assert.Equal(t, int64(2000000), stat.Tokens)
}

func TestRequestTrend_Fields(t *testing.T) {
	trend := RequestTrend{
		Timestamp: time.Now(),
		Requests:  1000,
		Success:   950,
		Failed:    50,
		Latency:   500,
	}

	assert.Equal(t, int64(1000), trend.Requests)
	assert.Equal(t, int64(950), trend.Success)
	assert.Equal(t, int64(50), trend.Failed)
	assert.Equal(t, int64(500), trend.Latency)
}

func TestAlertListItem_Fields(t *testing.T) {
	item := AlertListItem{
		ID:           "alert-1",
		Type:         "quota_exceeded",
		Level:        "warning",
		Message:      "Token quota exceeded",
		AccountID:    "acc-1",
		Provider:     "openai",
		Timestamp:    time.Now(),
		Acknowledged: false,
	}

	assert.Equal(t, "alert-1", item.ID)
	assert.Equal(t, "quota_exceeded", item.Type)
	assert.Equal(t, "warning", item.Level)
	assert.Equal(t, "acc-1", item.AccountID)
	assert.False(t, item.Acknowledged)
}

func TestGatewayRequest_Fields(t *testing.T) {
	req := GatewayRequest{
		ID:                  "gw-1",
		Name:                "Test Gateway",
		Endpoint:            "https://api.example.com",
		APIKey:              "sk-test",
		Description:         "Test gateway",
		Enabled:             true,
		Priority:            1,
		Timeout:             30,
		MaxRetries:          3,
		HealthCheckEnabled:  true,
		HealthCheckInterval: 60,
	}

	assert.Equal(t, "gw-1", req.ID)
	assert.Equal(t, "Test Gateway", req.Name)
	assert.Equal(t, "https://api.example.com", req.Endpoint)
	assert.True(t, req.Enabled)
	assert.Equal(t, 1, req.Priority)
	assert.Equal(t, 30, req.Timeout)
}

func TestGatewayResponse_Fields(t *testing.T) {
	resp := GatewayResponse{
		ID:                  "gw-1",
		Name:                "Test Gateway",
		Endpoint:            "https://api.example.com",
		Description:         "Test gateway",
		Enabled:             true,
		Priority:            1,
		Timeout:             30,
		MaxRetries:          3,
		HealthCheckEnabled:  true,
		HealthCheckInterval: 60,
		LastHealthCheck:     time.Now(),
		Healthy:             true,
		Latency:             100,
		RequestCount:        1000,
		SuccessRate:         95.0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	assert.Equal(t, "gw-1", resp.ID)
	assert.True(t, resp.Enabled)
	assert.True(t, resp.Healthy)
	assert.Equal(t, int64(100), resp.Latency)
	assert.Equal(t, int64(1000), resp.RequestCount)
}

func TestGatewayTestResult_Fields(t *testing.T) {
	result := GatewayTestResult{
		Success:   true,
		Latency:   100,
		Message:   "Connection successful",
		Timestamp: time.Now(),
	}

	assert.True(t, result.Success)
	assert.Equal(t, int64(100), result.Latency)
	assert.Equal(t, "Connection successful", result.Message)
}

func TestGatewayHealthHistory_Fields(t *testing.T) {
	history := GatewayHealthHistory{
		Timestamp: time.Now(),
		Healthy:   true,
		Latency:   100,
	}

	assert.True(t, history.Healthy)
	assert.Equal(t, int64(100), history.Latency)
}
