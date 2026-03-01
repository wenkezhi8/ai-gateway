package admin

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/metrics"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DashboardHandler handles dashboard data requests.
type DashboardHandler struct {
	registry *provider.Registry
	manager  *limiter.AccountManager
	cache    *cache.Manager
	mu       sync.RWMutex

	// In-memory stats for demo (in production, use metrics store)
	requestsToday int64
	totalRequests int64
	successCount  int64
	failureCount  int64
	totalLatency  int64
	totalTokens   int64
	requestTrends []RequestTrend
	alerts        []AlertListItem

	alertCooldown time.Duration
	lastAlerts    map[string]time.Time

	// Model stats
	modelStats map[string]*ModelStatData

	// Realtime stats (per minute)
	minuteRequests  int64
	minuteTokens    int64
	minuteErrors    int64
	minuteLatency   int64
	lastMinuteReset time.Time
}

type ModelStatData struct {
	Requests int64
	Tokens   int64
}

// RealtimeStats represents realtime metrics.
type RealtimeStats struct {
	Timestamp         time.Time     `json:"timestamp"`
	RequestsPerMinute int64         `json:"requests_per_minute"`
	TokensPerMinute   int64         `json:"tokens_per_minute"`
	AvgLatencyMs      int64         `json:"avg_latency_ms"`
	ErrorRate         float64       `json:"error_rate"`
	ActiveConnections int           `json:"active_connections"`
	TopModels         []ModelStat   `json:"top_models"`
	RecentErrors      []RecentError `json:"recent_errors"`
}

type RecentError struct {
	Timestamp time.Time `json:"timestamp"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Error     string    `json:"error"`
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(
	registry *provider.Registry,
	manager *limiter.AccountManager,
	cache *cache.Manager,
) *DashboardHandler {
	handler := &DashboardHandler{
		registry:        registry,
		manager:         manager,
		cache:           cache,
		requestTrends:   make([]RequestTrend, 0),
		alerts:          make([]AlertListItem, 0),
		alertCooldown:   defaultAlertCooldown,
		lastAlerts:      make(map[string]time.Time),
		modelStats:      make(map[string]*ModelStatData),
		lastMinuteReset: time.Now(),
	}
	handler.loadPersistedState()
	return handler
}

//nolint:gocyclo
func (h *DashboardHandler) loadPersistedState() {
	store := storage.GetSQLite()
	if store == nil {
		return
	}

	const maxTrends = 288
	const maxAlerts = 100

	summary, summaryOK, summaryErr := store.LoadDashboardSummary()
	models, modelErr := store.LoadDashboardModelStats()
	trends, trendErr := store.LoadDashboardTrends(maxTrends)
	alerts, alertErr := store.LoadDashboardAlerts(maxAlerts)

	if summaryErr != nil {
		logrus.WithError(summaryErr).Warn("Failed to load dashboard summary")
	}
	if modelErr != nil {
		logrus.WithError(modelErr).Warn("Failed to load dashboard model stats")
	}
	if trendErr != nil {
		logrus.WithError(trendErr).Warn("Failed to load dashboard trends")
	}
	if alertErr != nil {
		logrus.WithError(alertErr).Warn("Failed to load dashboard alerts")
	}

	if !summaryOK && modelErr != nil && trendErr != nil && alertErr != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if summaryOK {
		h.totalRequests = summary.TotalRequests
		h.requestsToday = summary.RequestsToday
		h.successCount = summary.SuccessCount
		h.failureCount = summary.FailureCount
		h.totalLatency = summary.TotalLatency
		h.totalTokens = summary.TotalTokens
	}

	if modelErr == nil {
		for _, item := range models {
			h.modelStats[item.Model] = &ModelStatData{
				Requests: item.Requests,
				Tokens:   item.Tokens,
			}
		}
	}

	if trendErr == nil {
		for _, item := range trends {
			h.requestTrends = append(h.requestTrends, RequestTrend{
				Timestamp: time.UnixMilli(item.Timestamp),
				Requests:  item.Requests,
				Success:   item.Success,
				Failed:    item.Failed,
				Latency:   item.Latency,
			})
		}
	}

	if alertErr == nil {
		for _, item := range alerts {
			h.alerts = append(h.alerts, AlertListItem{
				ID:           item.ID,
				Type:         item.Type,
				Level:        item.Level,
				Message:      item.Message,
				AccountID:    item.AccountID,
				Provider:     item.Provider,
				Timestamp:    time.UnixMilli(item.Timestamp),
				Acknowledged: item.Acknowledged,
			})
		}
	}
}

// GET /api/admin/dashboard/stats.
//
//nolint:gocyclo
func (h *DashboardHandler) GetStats(c *gin.Context) {
	providerUsageMap := make(map[string]storage.ProviderUsageStat)
	if store := storage.GetSQLite(); store != nil {
		if usageStats, err := store.GetProviderUsageStats(); err != nil {
			logrus.WithError(err).Warn("Failed to load provider usage stats")
		} else {
			for _, item := range usageStats {
				providerUsageMap[item.Provider] = item
			}
		}
	}

	// Get provider stats
	providers := h.registry.ListEnabled()
	providerStats := make([]ProviderStat, 0)
	activeProviders := 0

	for _, p := range providers {
		activeProviders++
		usage := providerUsageMap[p.Name()]
		providerStats = append(providerStats, ProviderStat{
			Name:        p.Name(),
			Requests:    usage.Requests,
			Tokens:      usage.Tokens,
			SuccessRate: usage.SuccessRate,
			AvgLatency:  usage.AvgLatency,
		})
	}

	// Get account stats
	accounts := h.manager.GetAllAccounts()
	activeAccounts := 0
	for _, acc := range accounts {
		if acc.Enabled {
			activeAccounts++
		}
	}

	// Get cache stats
	var cacheHitRate float64
	if h.cache != nil {
		allStats := h.cache.GetAllStats()
		// Combine all cache hit rates
		totalHits := 0.0
		totalOps := 0.0
		for _, stat := range allStats {
			totalHits += float64(stat.Hits)
			totalOps += float64(stat.Hits + stat.Misses)
		}
		if totalOps > 0 {
			cacheHitRate = totalHits / totalOps * 100
		}
	}

	// Calculate success rate
	var successRate float64
	if h.totalRequests > 0 {
		successRate = float64(h.successCount) / float64(h.totalRequests) * 100
	}

	// Calculate average latency
	var avgLatency int64
	if h.successCount > 0 {
		avgLatency = h.totalLatency / h.successCount
	}

	// Get top models from actual stats
	topModels := make([]ModelStat, 0)
	h.mu.RLock()
	for model, stats := range h.modelStats {
		topModels = append(topModels, ModelStat{
			Name:     model,
			Requests: stats.Requests,
			Tokens:   stats.Tokens,
		})
	}
	h.mu.RUnlock()

	// Sort by requests descending
	for i := 0; i < len(topModels); i++ {
		for j := i + 1; j < len(topModels); j++ {
			if topModels[j].Requests > topModels[i].Requests {
				topModels[i], topModels[j] = topModels[j], topModels[i]
			}
		}
	}

	// Limit to top 5
	if len(topModels) > 5 {
		topModels = topModels[:5]
	}

	// If no real data yet, show placeholder
	if len(topModels) == 0 {
		topModels = []ModelStat{
			{Name: "暂无数据", Requests: 0, Tokens: 0},
		}
	}

	stats := DashboardStats{
		TotalRequests:   h.totalRequests,
		RequestsToday:   h.requestsToday,
		SuccessRate:     successRate,
		AvgLatencyMs:    avgLatency,
		TotalTokens:     h.totalTokens,
		ActiveAccounts:  activeAccounts,
		ActiveProviders: activeProviders,
		CacheHitRate:    cacheHitRate,
		ProviderStats:   providerStats,
		TopModels:       topModels,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GET /api/admin/dashboard/requests.
func (h *DashboardHandler) GetRequestTrends(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "24h")

	h.mu.RLock()
	storedTrends := make([]RequestTrend, len(h.requestTrends))
	copy(storedTrends, h.requestTrends)
	h.mu.RUnlock()

	now := time.Now()
	var points []RequestTrend

	// Generate base time slots
	switch timeRange {
	case "1h":
		for i := 11; i >= 0; i-- {
			t := now.Add(-time.Duration(i*5) * time.Minute)
			points = append(points, RequestTrend{
				Timestamp: t,
				Requests:  0,
				Success:   0,
				Failed:    0,
				Latency:   0,
			})
		}
	case "7d":
		for i := 6; i >= 0; i-- {
			t := now.Add(-time.Duration(i) * 24 * time.Hour)
			points = append(points, RequestTrend{
				Timestamp: t,
				Requests:  0,
				Success:   0,
				Failed:    0,
				Latency:   0,
			})
		}
	default: // 24h
		for i := 23; i >= 0; i-- {
			t := now.Add(-time.Duration(i) * time.Hour)
			points = append(points, RequestTrend{
				Timestamp: t,
				Requests:  0,
				Success:   0,
				Failed:    0,
				Latency:   0,
			})
		}
	}

	// Merge real data into time slots
	for _, trend := range storedTrends {
		for i := range points {
			diff := trend.Timestamp.Sub(points[i].Timestamp)
			if diff >= 0 && diff < time.Hour {
				points[i].Requests += trend.Requests
				points[i].Success += trend.Success
				points[i].Failed += trend.Failed
				if points[i].Latency == 0 {
					points[i].Latency = trend.Latency
				} else {
					points[i].Latency = (points[i].Latency + trend.Latency) / 2
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    points,
	})
}

// GET /api/admin/dashboard/alerts.
func (h *DashboardHandler) GetAlerts(c *gin.Context) {
	// Get query parameters
	limit := 50
	acknowledged := c.Query("acknowledged")

	alerts := make([]AlertListItem, 0)

	h.mu.RLock()
	for i := range h.alerts {
		alert := h.alerts[i]
		if acknowledged == "" ||
			(acknowledged == "true" && alert.Acknowledged) ||
			(acknowledged == "false" && !alert.Acknowledged) {
			alerts = append(alerts, alert)
		}
	}
	h.mu.RUnlock()

	// Limit results
	if len(alerts) > limit {
		alerts = alerts[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    alerts,
	})
}

// POST /api/admin/dashboard/alerts/:id/acknowledge.
func (h *DashboardHandler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")

	h.mu.Lock()
	found := false

	for i := range h.alerts {
		if h.alerts[i].ID != alertID {
			continue
		}
		h.alerts[i].Acknowledged = true
		found = true
		break
	}
	h.mu.Unlock()

	if found {
		if store := storage.GetSQLite(); store != nil {
			if err := store.UpdateDashboardAlertAcknowledged(alertID, true); err != nil {
				logrus.WithError(err).Warn("Failed to persist dashboard alert acknowledgement")
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"id":      alertID,
				"message": "Alert acknowledged",
			},
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_found",
			"message": "Alert not found",
		},
	})
}

// GET /api/admin/dashboard/providers/:provider/metrics.
func (h *DashboardHandler) GetProviderMetrics(c *gin.Context) {
	providerName := c.Param("provider")

	// Check if metrics are available
	m := metrics.GetMetrics()
	if m == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"provider": providerName,
				"message":  "Metrics not initialized",
			},
		})
		return
	}

	// In production, this would query Prometheus metrics
	// For now, return placeholder data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"provider":     providerName,
			"requests":     0,
			"success_rate": 0,
			"avg_latency":  0,
			"tokens":       0,
		},
	})
}

// GET /api/admin/dashboard/models/:model/metrics.
func (h *DashboardHandler) GetModelMetrics(c *gin.Context) {
	model := c.Param("model")

	// Check if metrics are available
	m := metrics.GetMetrics()
	if m == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"model":   model,
				"message": "Metrics not initialized",
			},
		})
		return
	}

	// In production, this would query Prometheus metrics
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model":        model,
			"requests":     0,
			"tokens":       0,
			"avg_latency":  0,
			"cache_hits":   0,
			"cache_misses": 0,
		},
	})
}

// GET /api/admin/dashboard/system.
//
//nolint:gocyclo
func (h *DashboardHandler) GetSystemStatus(c *gin.Context) {
	// Check providers
	providers := h.registry.ListEnabled()

	// Check cache
	cacheHealthy := false
	if h.cache != nil {
		cacheHealthy = true
	}

	// Check accounts
	accounts := h.manager.GetAllAccounts()
	activeCount := 0
	for _, acc := range accounts {
		if acc.Enabled {
			activeCount++
		}
	}

	// Calculate provider distribution from model stats
	h.mu.RLock()
	distribution := make(map[string]int64)
	for model, stats := range h.modelStats {
		// Map model to provider
		provider := "other"
		if strings.Contains(model, "gpt") || strings.Contains(model, "o1") {
			provider = "openai"
		} else if strings.Contains(model, "claude") {
			provider = "anthropic"
		} else if strings.Contains(model, "deepseek") {
			provider = "deepseek"
		} else if strings.Contains(model, "qwen") {
			provider = "qwen"
		} else if strings.Contains(model, "glm") {
			provider = "zhipu"
		} else if strings.Contains(model, "moonshot") || strings.Contains(model, "kimi") {
			provider = "moonshot"
		} else if strings.Contains(model, "doubao") {
			provider = "volcengine"
		} else if strings.Contains(model, "gemini") {
			provider = "google"
		} else if strings.Contains(model, "yi-") {
			provider = "yi"
		} else if strings.Contains(model, "Baichuan") {
			provider = "baichuan"
		} else if strings.Contains(model, "abab") {
			provider = "minimax"
		} else if strings.Contains(model, "mistral") {
			provider = "mistral"
		}
		distribution[provider] += stats.Requests
	}
	h.mu.RUnlock()

	// If no real data, show enabled providers
	if len(distribution) == 0 {
		for _, p := range providers {
			distribution[p.Name()] = 1
		}
	}

	// Build provider details
	providerDetails := make([]gin.H, 0)
	for _, p := range providers {
		providerDetails = append(providerDetails, gin.H{
			"name":         p.Name(),
			"models":       p.Models(),
			"enabled":      p.IsEnabled(),
			"requests":     distribution[p.Name()],
			"tokens":       0,
			"success_rate": 100.0,
			"avg_latency":  0,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":       "healthy",
			"timestamp":    time.Now(),
			"providers":    providerDetails,
			"distribution": distribution,
			"total":        len(providers),
			"cache": gin.H{
				"healthy": cacheHealthy,
			},
			"accounts": gin.H{
				"total":  len(accounts),
				"active": activeCount,
			},
		},
	})
}

// UpdateStats updates internal statistics (called by other handlers).
//
//nolint:gocyclo
func (h *DashboardHandler) UpdateStats(success bool, latency, tokens int64, model ...string) {
	var (
		summary       storage.DashboardSummary
		modelName     string
		modelSnapshot *ModelStatData
		trendSnapshot RequestTrend
		hasTrend      bool
	)

	h.mu.Lock()

	now := time.Now()

	// Reset per-minute stats if a minute has passed
	if now.Sub(h.lastMinuteReset) >= time.Minute {
		h.minuteRequests = 0
		h.minuteTokens = 0
		h.minuteErrors = 0
		h.minuteLatency = 0
		h.lastMinuteReset = now
	}

	h.totalRequests++
	h.requestsToday++
	h.totalTokens += tokens
	h.totalLatency += latency

	// Per-minute stats
	h.minuteRequests++
	h.minuteTokens += tokens
	h.minuteLatency += latency

	if success {
		h.successCount++
	} else {
		h.failureCount++
		h.minuteErrors++
	}

	// Update model stats if model provided
	if len(model) > 0 && model[0] != "" {
		if h.modelStats[model[0]] == nil {
			h.modelStats[model[0]] = &ModelStatData{}
		}
		h.modelStats[model[0]].Requests++
		h.modelStats[model[0]].Tokens += tokens
		modelName = model[0]
		modelSnapshot = &ModelStatData{
			Requests: h.modelStats[model[0]].Requests,
			Tokens:   h.modelStats[model[0]].Tokens,
		}
	}

	// Add to trends
	if len(h.requestTrends) == 0 || now.Sub(h.requestTrends[len(h.requestTrends)-1].Timestamp) >= 5*time.Minute {
		h.requestTrends = append(h.requestTrends, RequestTrend{
			Timestamp: now,
			Requests:  1,
			Success:   boolToInt(success),
			Failed:    boolToInt(!success),
			Latency:   latency,
		})

		if len(h.requestTrends) > 288 {
			h.requestTrends = h.requestTrends[1:]
		}
	} else {
		last := &h.requestTrends[len(h.requestTrends)-1]
		last.Requests++
		last.Latency = (last.Latency*(last.Requests-1) + latency) / last.Requests
		if success {
			last.Success++
		} else {
			last.Failed++
		}
	}

	if len(h.requestTrends) > 0 {
		trendSnapshot = h.requestTrends[len(h.requestTrends)-1]
		hasTrend = true
	}

	summary = storage.DashboardSummary{
		TotalRequests: h.totalRequests,
		RequestsToday: h.requestsToday,
		SuccessCount:  h.successCount,
		FailureCount:  h.failureCount,
		TotalLatency:  h.totalLatency,
		TotalTokens:   h.totalTokens,
		UpdatedAt:     now,
	}

	h.mu.Unlock()

	store := storage.GetSQLite()
	if store == nil {
		return
	}

	if err := store.SaveDashboardSummary(summary); err != nil {
		logrus.WithError(err).Warn("Failed to persist dashboard summary")
	}
	if modelSnapshot != nil && modelName != "" {
		if err := store.SaveDashboardModelStat(modelName, modelSnapshot.Requests, modelSnapshot.Tokens); err != nil {
			logrus.WithError(err).Warn("Failed to persist dashboard model stats")
		}
	}
	if hasTrend {
		if err := store.SaveDashboardTrend(storage.DashboardTrend{
			Timestamp: trendSnapshot.Timestamp.UnixMilli(),
			Requests:  trendSnapshot.Requests,
			Success:   trendSnapshot.Success,
			Failed:    trendSnapshot.Failed,
			Latency:   trendSnapshot.Latency,
		}); err != nil {
			logrus.WithError(err).Warn("Failed to persist dashboard trends")
		}
	}
}

// GET /api/admin/dashboard/realtime.
func (h *DashboardHandler) GetRealtime(c *gin.Context) {
	h.mu.RLock()

	// Calculate per-minute stats
	var avgLatency int64
	if h.minuteRequests > 0 {
		avgLatency = h.minuteLatency / h.minuteRequests
	}

	var errorRate float64
	if h.minuteRequests > 0 {
		errorRate = float64(h.minuteErrors) / float64(h.minuteRequests) * 100
	}

	// Get top models
	topModels := make([]ModelStat, 0)
	for model, stats := range h.modelStats {
		topModels = append(topModels, ModelStat{
			Name:     model,
			Requests: stats.Requests,
			Tokens:   stats.Tokens,
		})
	}

	// Sort by requests
	for i := 0; i < len(topModels); i++ {
		for j := i + 1; j < len(topModels); j++ {
			if topModels[j].Requests > topModels[i].Requests {
				topModels[i], topModels[j] = topModels[j], topModels[i]
			}
		}
	}
	if len(topModels) > 5 {
		topModels = topModels[:5]
	}

	h.mu.RUnlock()

	// Get active connections (approximate)
	activeConnections := 0
	if h.manager != nil {
		accounts := h.manager.GetAllAccounts()
		for _, acc := range accounts {
			if acc.Enabled {
				activeConnections++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": RealtimeStats{
			Timestamp:         time.Now(),
			RequestsPerMinute: h.minuteRequests,
			TokensPerMinute:   h.minuteTokens,
			AvgLatencyMs:      avgLatency,
			ErrorRate:         errorRate,
			ActiveConnections: activeConnections,
			TopModels:         topModels,
			RecentErrors:      []RecentError{},
		},
	})
}

// AddAlert adds an alert to the list.
//
//nolint:gocritic
func (h *DashboardHandler) AddAlert(alert AlertListItem) {
	h.mu.Lock()

	now := alert.Timestamp
	if now.IsZero() {
		now = time.Now()
		alert.Timestamp = now
	}

	if h.alertCooldown > 0 {
		if h.lastAlerts == nil {
			h.lastAlerts = make(map[string]time.Time)
		}
		key := buildAlertDedupKey(alert.Type, alert.Level, alert.Message, alert.AccountID, alert.Provider)
		if last, ok := h.lastAlerts[key]; ok && now.Sub(last) < h.alertCooldown {
			h.mu.Unlock()
			return
		}
		h.lastAlerts[key] = now
	}

	h.alerts = append(h.alerts, alert)

	// Keep only last 100 alerts
	if len(h.alerts) > 100 {
		h.alerts = h.alerts[1:]
	}

	h.mu.Unlock()

	if store := storage.GetSQLite(); store != nil {
		if err := store.SaveDashboardAlert(storage.DashboardAlert{
			ID:           alert.ID,
			Type:         alert.Type,
			Level:        alert.Level,
			Message:      alert.Message,
			AccountID:    alert.AccountID,
			Provider:     alert.Provider,
			Timestamp:    alert.Timestamp.UnixMilli(),
			Acknowledged: alert.Acknowledged,
		}); err != nil {
			logrus.WithError(err).Warn("Failed to persist dashboard alert")
		}
	}
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
