package admin

import (
	"ai-gateway/internal/storage"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	ID            int64  `json:"id"`
	Timestamp     int64  `json:"timestamp"`
	Model         string `json:"model"`
	Provider      string `json:"provider"`
	UserID        string `json:"user_id,omitempty"`
	APIKey        string `json:"api_key,omitempty"`
	Tokens        int64  `json:"tokens"`
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	LatencyMs     int64  `json:"latency_ms"`
	TTFTMs        int64  `json:"ttft_ms"`
	CacheHit      bool   `json:"cache_hit"`
	Success       bool   `json:"success"`
	ErrorType     string `json:"error_type,omitempty"`
	TaskType      string `json:"task_type,omitempty"`
	Difficulty    string `json:"difficulty,omitempty"`
	ExperimentTag string `json:"experiment_tag,omitempty"`
	DomainTag     string `json:"domain_tag,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type UsageStatsResponse struct {
	TotalRequests int64                 `json:"total_requests"`
	TotalTokens   int64                 `json:"total_tokens"`
	CacheHits     int64                 `json:"cache_hits"`
	CacheMisses   int64                 `json:"cache_misses"`
	SavedTokens   int64                 `json:"saved_tokens"`
	SavedRequests int64                 `json:"saved_requests"`
	CacheHitRate  float64               `json:"cache_hit_rate"`
	AvgLatencyMs  int64                 `json:"avg_latency_ms"`
	ModelStats    map[string]ModelUsage `json:"model_stats"`
}

type ModelUsage struct {
	Requests int64 `json:"requests"`
	Tokens   int64 `json:"tokens"`
}

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
		item := UsageLogResponse{
			ID:            getInt64(log, "id"),
			Timestamp:     getInt64(log, "timestamp"),
			Model:         getString(log, "model"),
			Provider:      getString(log, "provider"),
			UserID:        getString(log, "user_id"),
			APIKey:        apiKey,
			Tokens:        getInt64(log, "tokens"),
			InputTokens:   getInt64(log, "input_tokens"),
			OutputTokens:  getInt64(log, "output_tokens"),
			LatencyMs:     getInt64(log, "latency_ms"),
			TTFTMs:        getInt64(log, "ttft_ms"),
			CacheHit:      getBool(log, "cache_hit"),
			Success:       getBool(log, "success"),
			ErrorType:     getString(log, "error_type"),
			TaskType:      getString(log, "task_type"),
			Difficulty:    getString(log, "difficulty"),
			ExperimentTag: getString(log, "experiment_tag"),
			DomainTag:     getString(log, "domain_tag"),
			CreatedAt:     getString(log, "created_at"),
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
		TotalRequests: getInt64FromMap(stats, "total_requests"),
		TotalTokens:   getInt64FromMap(stats, "total_tokens"),
		CacheHits:     getInt64FromMap(stats, "cache_hits"),
		CacheMisses:   getInt64FromMap(stats, "cache_misses"),
		SavedTokens:   getInt64FromMap(stats, "saved_tokens"),
		SavedRequests: getInt64FromMap(stats, "saved_requests"),
		CacheHitRate:  getFloat64FromMap(stats, "cache_hit_rate"),
		AvgLatencyMs:  getInt64FromMap(stats, "avg_latency_ms"),
		ModelStats:    modelStats,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
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
