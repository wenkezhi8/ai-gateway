package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
)

type UsageHandler struct {
	storage *storage.SQLiteStorage
}

func NewUsageHandler(storage *storage.SQLiteStorage) *UsageHandler {
	return &UsageHandler{
		storage: storage,
	}
}

type UsageLogResponse struct {
	ID                 int64   `json:"id"`
	Timestamp          int64   `json:"timestamp"`
	Model              string  `json:"model"`
	Provider           string  `json:"provider"`
	ServiceProvider    string  `json:"service_provider"`
	Account            string  `json:"account"`
	UserID             string  `json:"user_id,omitempty"`
	APIKey             string  `json:"api_key,omitempty"`
	UserAgent          string  `json:"user_agent,omitempty"`
	Type               string  `json:"type,omitempty"`
	RequestType        string  `json:"request_type,omitempty"`
	InferenceIntensity string  `json:"inference_intensity,omitempty"`
	Tokens             int64   `json:"tokens"`
	InputTokens        int64   `json:"input_tokens"`
	OutputTokens       int64   `json:"output_tokens"`
	CachedReadTokens   int64   `json:"cached_read_tokens"`
	TotalTokens        int64   `json:"total_tokens"`
	SavedTokens        int64   `json:"saved_tokens"`
	LatencyMs          int64   `json:"latency_ms"`
	TotalDuration      int64   `json:"total_duration"`
	TTFTMs             int64   `json:"ttft_ms"`
	TimeToFirstToken   int64   `json:"time_to_first_token"`
	CacheHit           bool    `json:"cache_hit"`
	Success            bool    `json:"success"`
	ErrorType          string  `json:"error_type,omitempty"`
	TaskType           string  `json:"task_type,omitempty"`
	Difficulty         string  `json:"difficulty,omitempty"`
	ExperimentTag      string  `json:"experiment_tag,omitempty"`
	DomainTag          string  `json:"domain_tag,omitempty"`
	UsageSource        string  `json:"usage_source,omitempty"`
	CompressionApplied bool    `json:"compression_applied"`
	CompressionRatio   float64 `json:"compression_ratio"`
	GuardFailed        bool    `json:"guard_failed"`
	FallbackInvoked    bool    `json:"fallback_invoked"`
	FallbackSaved      bool    `json:"fallback_saved"`
	RAGRequested       bool    `json:"rag_requested"`
	RAGUsed            bool    `json:"rag_used"`
	RAGFailed          bool    `json:"rag_failed"`
	CreatedAt          string  `json:"created_at"`
}

type UsageStatsResponse struct {
	TotalRequests          int64                 `json:"total_requests"`
	TotalTokens            int64                 `json:"total_tokens"`
	CacheHits              int64                 `json:"cache_hits"`
	CacheMisses            int64                 `json:"cache_misses"`
	SavedTokens            int64                 `json:"saved_tokens"`
	SavedRequests          int64                 `json:"saved_requests"`
	CacheHitRate           float64               `json:"cache_hit_rate"`
	AvgLatencyMs           int64                 `json:"avg_latency_ms"`
	CompressionTriggered   int64                 `json:"compression_triggered"`
	CompressionTriggerRate float64               `json:"compression_trigger_rate"`
	CompressionRatioAvg    float64               `json:"compression_ratio_avg"`
	GuardFailed            int64                 `json:"guard_failed"`
	GuardFailedRate        float64               `json:"guard_failed_rate"`
	FallbackInvoked        int64                 `json:"fallback_invoked"`
	FallbackRate           float64               `json:"fallback_rate"`
	FallbackSaved          int64                 `json:"fallback_saved"`
	RAGRequested           int64                 `json:"rag_requested"`
	RAGUsed                int64                 `json:"rag_used"`
	RAGFailed              int64                 `json:"rag_failed"`
	NetSavedTokens         int64                 `json:"net_saved_tokens"`
	ModelStats             map[string]ModelUsage `json:"model_stats"`
}

type ModelUsage struct {
	Requests int64 `json:"requests"`
	Tokens   int64 `json:"tokens"`
}

//nolint:gocyclo
func (h *UsageHandler) GetUsageLogs(c *gin.Context) {
	limit := 100
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	filter := storage.UsageFilter{
		Model:         c.Query("model"),
		Provider:      c.Query("provider"),
		TaskType:      c.Query("task_type"),
		ExperimentTag: c.Query("experiment_tag"),
		DomainTag:     c.Query("domain_tag"),
	}

	if start := c.Query("start_time"); start != "" {
		if parsed, err := strconv.ParseInt(start, 10, 64); err == nil {
			filter.StartTime = parsed
		}
	}

	if end := c.Query("end_time"); end != "" {
		if parsed, err := strconv.ParseInt(end, 10, 64); err == nil {
			filter.EndTime = parsed
		}
	}

	if rangeParam := c.Query("range"); rangeParam != "" {
		now := time.Now().UnixMilli()
		switch rangeParam {
		case "24h":
			filter.StartTime = now - 24*60*60*1000
		case "7d":
			filter.StartTime = now - 7*24*60*60*1000
		case "30d":
			filter.StartTime = now - 30*24*60*60*1000
		}
	}

	logs, err := h.storage.GetUsageLogsWithFilter(filter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	response := make([]UsageLogResponse, 0, len(logs))
	for _, log := range logs {
		apiKey := getString(log, "api_key")
		if strings.Contains(apiKey, "****") {
			apiKey = ""
		}
		tokens := getInt64(log, "tokens")
		latencyMs := getInt64(log, "latency_ms")
		ttftMs := getInt64(log, "ttft_ms")
		cacheHit := getBool(log, "cache_hit")
		success := getBool(log, "success")
		savedTokens := int64(0)
		if cacheHit && success {
			savedTokens = tokens
		}
		provider := getString(log, "provider")
		requestType := getString(log, "request_type")
		item := UsageLogResponse{
			ID:                 getInt64(log, "id"),
			Timestamp:          getInt64(log, "timestamp"),
			Model:              getString(log, "model"),
			Provider:           provider,
			ServiceProvider:    provider,
			Account:            getString(log, "account"),
			UserID:             getString(log, "user_id"),
			APIKey:             apiKey,
			UserAgent:          getString(log, "user_agent"),
			Type:               requestType,
			RequestType:        requestType,
			InferenceIntensity: getString(log, "inference_intensity"),
			Tokens:             tokens,
			InputTokens:        getInt64(log, "input_tokens"),
			OutputTokens:       getInt64(log, "output_tokens"),
			CachedReadTokens:   getInt64(log, "cached_read_tokens"),
			TotalTokens:        tokens,
			SavedTokens:        savedTokens,
			LatencyMs:          latencyMs,
			TotalDuration:      latencyMs,
			TTFTMs:             ttftMs,
			TimeToFirstToken:   ttftMs,
			CacheHit:           cacheHit,
			Success:            success,
			ErrorType:          getString(log, "error_type"),
			TaskType:           getString(log, "task_type"),
			Difficulty:         getString(log, "difficulty"),
			ExperimentTag:      getString(log, "experiment_tag"),
			DomainTag:          getString(log, "domain_tag"),
			UsageSource:        getString(log, "usage_source"),
			CompressionApplied: getBool(log, "compression_applied"),
			CompressionRatio:   getFloat64(log, "compression_ratio"),
			GuardFailed:        getBool(log, "guard_failed"),
			FallbackInvoked:    getBool(log, "fallback_invoked"),
			FallbackSaved:      getBool(log, "fallback_saved"),
			RAGRequested:       getBool(log, "rag_requested"),
			RAGUsed:            getBool(log, "rag_used"),
			RAGFailed:          getBool(log, "rag_failed"),
			CreatedAt:          getString(log, "created_at"),
		}
		response = append(response, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"total":   len(response),
	})
}

func (h *UsageHandler) GetUsageStats(c *gin.Context) {
	filter := storage.UsageFilter{
		Model:         c.Query("model"),
		Provider:      c.Query("provider"),
		TaskType:      c.Query("task_type"),
		ExperimentTag: c.Query("experiment_tag"),
		DomainTag:     c.Query("domain_tag"),
	}

	if start := c.Query("start_time"); start != "" {
		if parsed, err := strconv.ParseInt(start, 10, 64); err == nil {
			filter.StartTime = parsed
		}
	}

	if end := c.Query("end_time"); end != "" {
		if parsed, err := strconv.ParseInt(end, 10, 64); err == nil {
			filter.EndTime = parsed
		}
	}

	if rangeParam := c.Query("range"); rangeParam != "" {
		now := time.Now().UnixMilli()
		switch rangeParam {
		case "24h":
			filter.StartTime = now - 24*60*60*1000
		case "7d":
			filter.StartTime = now - 7*24*60*60*1000
		case "30d":
			filter.StartTime = now - 30*24*60*60*1000
		}
	}

	stats := h.storage.GetUsageStatsWithFilter(filter)

	modelStats := make(map[string]ModelUsage)
	if ms, ok := stats["model_stats"].(map[string]map[string]int64); ok {
		for model, data := range ms {
			modelStats[model] = ModelUsage{
				Requests: data["requests"],
				Tokens:   data["tokens"],
			}
		}
	}

	response := UsageStatsResponse{
		TotalRequests:          getInt64FromMap(stats, "total_requests"),
		TotalTokens:            getInt64FromMap(stats, "total_tokens"),
		CacheHits:              getInt64FromMap(stats, "cache_hits"),
		CacheMisses:            getInt64FromMap(stats, "cache_misses"),
		SavedTokens:            getInt64FromMap(stats, "saved_tokens"),
		SavedRequests:          getInt64FromMap(stats, "saved_requests"),
		CacheHitRate:           getFloat64FromMap(stats, "cache_hit_rate"),
		AvgLatencyMs:           getInt64FromMap(stats, "avg_latency_ms"),
		CompressionTriggered:   getInt64FromMap(stats, "compression_triggered"),
		CompressionTriggerRate: getFloat64FromMap(stats, "compression_trigger_rate"),
		CompressionRatioAvg:    getFloat64FromMap(stats, "compression_ratio_avg"),
		GuardFailed:            getInt64FromMap(stats, "guard_failed"),
		GuardFailedRate:        getFloat64FromMap(stats, "guard_failed_rate"),
		FallbackInvoked:        getInt64FromMap(stats, "fallback_invoked"),
		FallbackRate:           getFloat64FromMap(stats, "fallback_rate"),
		FallbackSaved:          getInt64FromMap(stats, "fallback_saved"),
		RAGRequested:           getInt64FromMap(stats, "rag_requested"),
		RAGUsed:                getInt64FromMap(stats, "rag_used"),
		RAGFailed:              getInt64FromMap(stats, "rag_failed"),
		NetSavedTokens:         getInt64FromMap(stats, "net_saved_tokens"),
		ModelStats:             modelStats,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *UsageHandler) ClearUsageLogs(c *gin.Context) {
	deleted, err := h.storage.ClearUsageLogs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted": deleted,
		},
	})
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case float64:
			return int64(val)
		}
	}
	return 0
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0
}

func getInt64FromMap(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case float64:
			return int64(val)
		}
	}
	return 0
}

func getFloat64FromMap(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int64:
			return float64(val)
		case int:
			return float64(val)
		}
	}
	return 0
}
