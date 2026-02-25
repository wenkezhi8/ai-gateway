package admin

import (
	"ai-gateway/internal/cache"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CacheHandler handles cache management requests
type CacheHandler struct {
	manager  *cache.Manager
	settings cache.CacheSettings
	mu       sync.RWMutex
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(manager *cache.Manager) *CacheHandler {
	return &CacheHandler{
		manager:  manager,
		settings: manager.GetSettings(),
	}
}

// GetCacheStats returns cache statistics
// GET /api/admin/cache/stats
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	allStats := h.manager.GetAllStats()

	response := CacheStatsResponse{
		TokenSavings: h.manager.GetTokenSavings(),
	}

	// Convert stats for each cache type
	if stat, ok := allStats["request"]; ok {
		entries, sizeBytes := h.manager.GetEntriesStats("request")
		response.RequestCache = CacheStatDetail{
			Hits:         stat.Hits,
			Misses:       stat.Misses,
			HitRate:      stat.HitRate,
			Entries:      int64(entries),
			SizeBytes:    sizeBytes,
			AvgLatencyMs: stat.AvgLatencyNs / int64(time.Millisecond),
			Evictions:    stat.Evictions,
		}
	}

	if stat, ok := allStats["context"]; ok {
		entries, sizeBytes := h.manager.GetEntriesStats("context")
		response.ContextCache = CacheStatDetail{
			Hits:         stat.Hits,
			Misses:       stat.Misses,
			HitRate:      stat.HitRate,
			Entries:      int64(entries),
			SizeBytes:    sizeBytes,
			AvgLatencyMs: stat.AvgLatencyNs / int64(time.Millisecond),
			Evictions:    stat.Evictions,
		}
	}

	if stat, ok := allStats["route"]; ok {
		entries, sizeBytes := h.manager.GetEntriesStats("route")
		response.RouteCache = CacheStatDetail{
			Hits:         stat.Hits,
			Misses:       stat.Misses,
			HitRate:      stat.HitRate,
			Entries:      int64(entries),
			SizeBytes:    sizeBytes,
			AvgLatencyMs: stat.AvgLatencyNs / int64(time.Millisecond),
			Evictions:    stat.Evictions,
		}
	}

	if stat, ok := allStats["usage"]; ok {
		entries, sizeBytes := h.manager.GetEntriesStats("usage")
		response.UsageCache = CacheStatDetail{
			Hits:         stat.Hits,
			Misses:       stat.Misses,
			HitRate:      stat.HitRate,
			Entries:      int64(entries),
			SizeBytes:    sizeBytes,
			AvgLatencyMs: stat.AvgLatencyNs / int64(time.Millisecond),
			Evictions:    stat.Evictions,
		}
	}

	if stat, ok := allStats["response"]; ok {
		entries, sizeBytes := h.manager.GetEntriesStats("response")
		response.ResponseCache = CacheStatDetail{
			Hits:         stat.Hits,
			Misses:       stat.Misses,
			HitRate:      stat.HitRate,
			Entries:      int64(entries),
			SizeBytes:    sizeBytes,
			AvgLatencyMs: stat.AvgLatencyNs / int64(time.Millisecond),
			Evictions:    stat.Evictions,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// ClearCache clears all caches
// DELETE /api/admin/cache
func (h *CacheHandler) ClearCache(c *gin.Context) {
	cacheType := c.Query("type") // request, context, route, usage, response, all

	ctx := context.Background()
	var err error

	switch cacheType {
	case "request":
		err = h.manager.Cache().DeleteByPattern(ctx, "req:*")
	case "context":
		err = h.manager.Cache().DeleteByPattern(ctx, "ctx:*")
	case "route":
		err = h.manager.Cache().DeleteByPattern(ctx, "route:*")
	case "usage":
		err = h.manager.Cache().DeleteByPattern(ctx, "usage:*")
	case "response":
		err = h.manager.Cache().DeleteByPattern(ctx, "ai-response:*")
	case "all", "":
		err = h.manager.InvalidateAll(ctx)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_type",
				"message": "Invalid cache type. Valid options: request, context, route, usage, response, all",
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "clear_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"type":    cacheType,
			"message": "Cache cleared successfully",
		},
	})
}

// UpdateCacheConfig updates cache configuration
// PUT /api/admin/cache/config
func (h *CacheHandler) UpdateCacheConfig(c *gin.Context) {
	var req CacheConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	settings := h.manager.GetSettings()

	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.Strategy != nil {
		switch *req.Strategy {
		case string(cache.CacheStrategySemantic), string(cache.CacheStrategyExact), string(cache.CacheStrategyPrefix):
			settings.Strategy = cache.CacheStrategy(*req.Strategy)
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_strategy",
					"message": "Invalid cache strategy",
				},
			})
			return
		}
	}
	if req.SimilarityThreshold != nil {
		value := *req.SimilarityThreshold
		// Allow 0-100 input from UI
		if value > 1 {
			value = value / 100
		}
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		settings.SimilarityThreshold = value
	}
	if req.DefaultTTLSeconds != nil {
		settings.DefaultTTLSeconds = *req.DefaultTTLSeconds
	}
	if req.MaxEntries != nil {
		settings.MaxEntries = *req.MaxEntries
	}
	if req.EvictionPolicy != nil {
		settings.EvictionPolicy = *req.EvictionPolicy
	}
	if req.Dedup != nil {
		if req.Dedup.Enabled != nil {
			settings.Dedup.Enabled = *req.Dedup.Enabled
		}
		if req.Dedup.MaxPending != nil {
			settings.Dedup.MaxPending = *req.Dedup.MaxPending
		}
		if req.Dedup.RequestTimeoutSeconds != nil {
			settings.Dedup.RequestTimeoutSeconds = *req.Dedup.RequestTimeoutSeconds
		}
	}

	// Apply base cache settings
	h.manager.UpdateSettings(settings)

	// Optional per-cache TTL overrides
	if req.RequestTTL != nil && h.manager.RequestCache != nil {
		h.manager.RequestCache.SetDefaultTTL(time.Duration(*req.RequestTTL) * time.Second)
	}
	if req.ContextTTL != nil && h.manager.ContextCache != nil {
		h.manager.ContextCache.SetDefaultTTL(time.Duration(*req.ContextTTL) * time.Second)
	}
	if req.RouteTTL != nil && h.manager.RouteCache != nil {
		h.manager.RouteCache.SetDefaultTTL(time.Duration(*req.RouteTTL) * time.Second)
	}

	h.settings = settings

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Cache configuration updated",
			"config":  settings,
		},
	})
}

// GetCacheConfig returns current cache configuration
// GET /api/admin/cache/config
func (h *CacheHandler) GetCacheConfig(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	settings := h.manager.GetSettings()
	dedupConfig, enabled := cache.GetRequestDeduplicator().GetConfig()
	settings.Dedup.Enabled = enabled
	settings.Dedup.MaxPending = dedupConfig.MaxPending
	settings.Dedup.RequestTimeoutSeconds = int(dedupConfig.RequestTimeout.Seconds())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// InvalidateProvider invalidates cache for a specific provider
// DELETE /api/admin/cache/provider/:provider
func (h *CacheHandler) InvalidateProvider(c *gin.Context) {
	provider := c.Param("provider")

	ctx := context.Background()
	if err := h.manager.InvalidateProvider(ctx, provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalidate_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"provider": provider,
			"message":  "Provider cache invalidated successfully",
		},
	})
}

// InvalidateModel invalidates cache for a specific model
// DELETE /api/admin/cache/model/:model
func (h *CacheHandler) InvalidateModel(c *gin.Context) {
	model := c.Param("model")
	provider := c.Query("provider")
	if provider == "" {
		provider = "*"
	}

	ctx := context.Background()
	if err := h.manager.InvalidateModel(ctx, provider, model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalidate_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model":    model,
			"provider": provider,
			"message":  "Model cache invalidated successfully",
		},
	})
}

// GetCacheHealth returns cache health status
// GET /api/admin/cache/health
func (h *CacheHandler) GetCacheHealth(c *gin.Context) {
	ctx := context.Background()
	start := time.Now()
	err := h.manager.HealthCheck(ctx)
	latency := time.Since(start)

	healthy := err == nil
	backend := "memory"
	if _, ok := h.manager.Cache().(*cache.RedisCache); ok {
		backend = "redis"
	}
	persistent := backend == "redis"
	degraded := !persistent

	response := gin.H{
		"status":     map[bool]string{true: "healthy", false: "unhealthy"}[healthy],
		"backend":    backend,
		"persistent": persistent,
		"degraded":   degraded,
		"latency_ms": latency.Milliseconds(),
		"timestamp":  time.Now(),
	}

	if degraded {
		response["reason"] = "cache backend is memory"
	}

	if !healthy {
		response["error"] = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetCacheSummary returns a summary of cache state
// GET /api/admin/cache/summary
func (h *CacheHandler) GetCacheSummary(c *gin.Context) {
	summary := h.manager.Summary()

	c.Data(http.StatusOK, "application/json", summary)
}

// GetCacheQualityConfig returns cache quality configuration
// GET /api/admin/cache/quality-config
func (h *CacheHandler) GetCacheQualityConfig(c *gin.Context) {
	// 获取语义缓存的质量配置
	semanticCache := h.manager.GetSemanticCache()
	if semanticCache == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"enabled":           false,
				"min_quality_score": 0,
				"message":           "Semantic cache not available",
			},
		})
		return
	}

	config := semanticCache.GetQualityConfig()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":           true,
			"min_quality_score": config["min_quality_score"],
			"checker_type":      config["checker_type"],
		},
	})
}

// UpdateCacheQualityConfigRequest represents quality config update request
type UpdateCacheQualityConfigRequest struct {
	MinQualityScore *float64 `json:"min_quality_score"`
}

// UpdateCacheQualityConfig updates cache quality configuration
// PUT /api/admin/cache/quality-config
func (h *CacheHandler) UpdateCacheQualityConfig(c *gin.Context) {
	var req UpdateCacheQualityConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	semanticCache := h.manager.GetSemanticCache()
	if semanticCache == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_available",
				"message": "Semantic cache not available",
			},
		})
		return
	}

	if req.MinQualityScore != nil {
		semanticCache.SetMinQualityScore(*req.MinQualityScore)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache quality configuration updated",
		"data": gin.H{
			"min_quality_score": *req.MinQualityScore,
		},
	})
}

// InvalidateLowQualityCache removes low quality cache entries
// POST /api/admin/cache/invalidate-low-quality
func (h *CacheHandler) InvalidateLowQualityCache(c *gin.Context) {
	semanticCache := h.manager.GetSemanticCache()
	if semanticCache == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"invalidated": 0,
				"message":     "Semantic cache not available",
			},
		})
		return
	}

	count := semanticCache.InvalidateLowQuality()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"invalidated": count,
			"message":     "Low quality cache entries invalidated",
		},
	})
}

// GetCacheRules returns all cache rules
// GET /api/admin/cache/rules
func (h *CacheHandler) GetCacheRules(c *gin.Context) {
	rules := cache.GetRuleStore().List()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rules,
	})
}

// CreateCacheRuleRequest represents create cache rule request
type CreateCacheRuleRequest struct {
	Pattern     string `json:"pattern" binding:"required"`
	ModelFilter string `json:"model_filter"`
	TTL         int    `json:"ttl" binding:"required"`
	Priority    string `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// CreateCacheRule creates a new cache rule
// POST /api/admin/cache/rules
func (h *CacheHandler) CreateCacheRule(c *gin.Context) {
	var req CreateCacheRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	rule := &cache.CacheRule{
		Pattern:     req.Pattern,
		ModelFilter: req.ModelFilter,
		TTL:         req.TTL,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
	}

	rule = cache.GetRuleStore().Create(rule)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rule,
	})
}

// UpdateCacheRuleRequest represents update cache rule request
type UpdateCacheRuleRequest struct {
	Pattern     *string `json:"pattern"`
	ModelFilter *string `json:"model_filter"`
	TTL         *int    `json:"ttl"`
	Priority    *string `json:"priority"`
	Enabled     *bool   `json:"enabled"`
}

// UpdateCacheRule updates a cache rule
// PUT /api/admin/cache/rules/:id
func (h *CacheHandler) UpdateCacheRule(c *gin.Context) {
	id := c.Param("id")
	ruleID := 0
	if _, err := fmt.Sscanf(id, "%d", &ruleID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_id",
				"message": "Invalid rule ID",
			},
		})
		return
	}

	var req UpdateCacheRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	rule, ok := cache.GetRuleStore().Update(ruleID, func(rule *cache.CacheRule) {
		if req.Pattern != nil {
			rule.Pattern = *req.Pattern
		}
		if req.ModelFilter != nil {
			rule.ModelFilter = *req.ModelFilter
		}
		if req.TTL != nil {
			rule.TTL = *req.TTL
		}
		if req.Priority != nil {
			rule.Priority = *req.Priority
		}
		if req.Enabled != nil {
			rule.Enabled = *req.Enabled
		}
	})
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Cache rule not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rule,
	})
}

// DeleteCacheRule deletes a cache rule
// DELETE /api/admin/cache/rules/:id
func (h *CacheHandler) DeleteCacheRule(c *gin.Context) {
	id := c.Param("id")
	ruleID := 0
	if _, err := fmt.Sscanf(id, "%d", &ruleID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_id",
				"message": "Invalid rule ID",
			},
		})
		return
	}

	if !cache.GetRuleStore().Delete(ruleID) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Cache rule not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache rule deleted",
	})
}

// CacheEntry represents a cache entry for display
type CacheEntry struct {
	Key       string     `json:"key"`
	Type      string     `json:"type"`
	Size      int        `json:"size"`
	Hits      int        `json:"hits"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	TTL       int        `json:"ttl"`
	Preview   string     `json:"preview"`
	Model     string     `json:"model,omitempty"`
	Provider  string     `json:"provider,omitempty"`
}

// GetCacheEntries returns paginated cache entries
// GET /api/admin/cache/entries
func (h *CacheHandler) GetCacheEntries(c *gin.Context) {
	cacheType := c.Query("type")
	search := c.Query("search")
	taskType := c.Query("task_type")
	if taskType == "other" {
		taskType = "unknown"
	}
	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	entries := h.manager.ListEntries(cacheType, search)
	ctx := context.Background()

	if taskType != "" {
		filtered := make([]*cache.CacheEntryInfo, 0, len(entries))
		for _, entry := range entries {
			detail, err := h.manager.GetEntryDetail(ctx, entry.Key)
			if err != nil {
				continue
			}
			entryTaskType, userMsg, aiResp := extractCacheSummary(detail.Value)
			if entry.TaskType == "" && entryTaskType != "" {
				entry.TaskType = entryTaskType
			}
			if userMsg != "" {
				entry.UserMessage = userMsg
			}
			if aiResp != "" {
				entry.AIResponse = aiResp
			}
			if entry.TaskType == taskType {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	total := len(entries)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pageEntries := entries[start:end]
	// Enrich page entries with preview data when not already filled
	for _, entry := range pageEntries {
		if entry.UserMessage != "" || entry.AIResponse != "" {
			continue
		}
		detail, err := h.manager.GetEntryDetail(ctx, entry.Key)
		if err != nil {
			continue
		}
		entryTaskType, userMsg, aiResp := extractCacheSummary(detail.Value)
		if entry.TaskType == "" && entryTaskType != "" {
			entry.TaskType = entryTaskType
		}
		if userMsg != "" {
			entry.UserMessage = userMsg
		}
		if aiResp != "" {
			entry.AIResponse = aiResp
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"entries":   pageEntries,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetCacheEntryDetail returns detail of a cache entry
// GET /api/admin/cache/entries/*
func (h *CacheHandler) GetCacheEntryDetail(c *gin.Context) {
	key := c.Param("key")
	// Gin 的 *key 通配符会包含前导斜杠，需要去掉
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "invalid_key", "message": "Key is required"},
		})
		return
	}

	ctx := context.Background()
	entry, err := h.manager.GetEntryDetail(ctx, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   gin.H{"code": "not_found", "message": "Cache entry not found"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entry,
	})
}

// DeleteCacheEntry deletes a cache entry
// DELETE /api/admin/cache/entries/*
func (h *CacheHandler) DeleteCacheEntry(c *gin.Context) {
	key := c.Param("key")
	// Gin 的 *key 通配符会包含前导斜杠，需要去掉
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "invalid_key", "message": "Key is required"},
		})
		return
	}

	ctx := context.Background()
	if err := h.manager.Cache().Delete(ctx, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "delete_failed", "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache entry deleted",
	})
}

// AddTestCacheEntryRequest represents request for adding test cache
type AddTestCacheEntryRequest struct {
	TaskType    string `json:"task_type" binding:"required"`
	UserMessage string `json:"user_message" binding:"required"`
	AIResponse  string `json:"ai_response" binding:"required"`
	Model       string `json:"model"`
	Provider    string `json:"provider"`
	TTL         int    `json:"ttl"` // hours
}

// AddTestCacheEntry adds a test cache entry for warmup
// POST /api/admin/cache/test-entry
func (h *CacheHandler) AddTestCacheEntry(c *gin.Context) {
	var req AddTestCacheEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "invalid_request", "message": err.Error()},
		})
		return
	}

	// Generate a unique key
	key := fmt.Sprintf("test:%s:%d", req.TaskType, time.Now().UnixNano())

	// Build request data
	requestData := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": req.UserMessage},
		},
		"model": req.Model,
	}

	// Build response data
	responseData := map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]string{
					"role":    "assistant",
					"content": req.AIResponse,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     len(req.UserMessage) / 4,
			"completion_tokens": len(req.AIResponse) / 4,
			"total_tokens":      (len(req.UserMessage) + len(req.AIResponse)) / 4,
		},
	}

	ctx := context.Background()
	ttl := time.Duration(req.TTL) * time.Hour
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	// Store request cache
	reqKey := "req:test:" + key
	if mc, ok := h.manager.Cache().(*cache.MemoryCache); ok {
		mc.SetWithTaskType(ctx, reqKey, requestData, ttl, req.Model, req.Provider, req.TaskType)
	} else {
		h.manager.Cache().Set(ctx, reqKey, requestData, ttl)
	}

	// Store response cache
	respKey := "ai-response:test:" + key
	if mc, ok := h.manager.Cache().(*cache.MemoryCache); ok {
		mc.SetWithTaskType(ctx, respKey, responseData, ttl, req.Model, req.Provider, req.TaskType)
	} else {
		h.manager.Cache().Set(ctx, respKey, responseData, ttl)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"request_key":  reqKey,
			"response_key": respKey,
			"task_type":    req.TaskType,
			"ttl_hours":    req.TTL,
			"message":      "Test cache entry added successfully",
		},
	})
}

// ExportCacheEntries exports all cache entries
// GET /api/admin/cache/export
func (h *CacheHandler) ExportCacheEntries(c *gin.Context) {
	taskType := c.Query("task_type")
	if taskType == "other" {
		taskType = "unknown"
	}

	entries := h.manager.ListEntries("", "")

	// Filter by task type if specified
	filtered := make([]*cache.CacheEntryInfo, 0)
	for _, entry := range entries {
		if taskType == "" || entry.TaskType == taskType {
			filtered = append(filtered, entry)
		}
	}

	// Get details for each entry
	exportData := make([]map[string]interface{}, 0)
	ctx := context.Background()
	for _, entry := range filtered {
		detail, err := h.manager.GetEntryDetail(ctx, entry.Key)
		if err == nil {
			taskType := entry.TaskType
			if taskType == "" {
				if extracted, _, _ := extractCacheSummary(detail.Value); extracted != "" {
					taskType = extracted
				}
			}
			exportData = append(exportData, map[string]interface{}{
				"key":        entry.Key,
				"type":       entry.Type,
				"task_type":  taskType,
				"model":      entry.Model,
				"provider":   entry.Provider,
				"size":       entry.Size,
				"hits":       entry.Hits,
				"created_at": entry.CreatedAt,
				"ttl":        entry.TTL,
				"value":      detail.Value,
			})
		}
	}

	c.Header("Content-Disposition", "attachment; filename=cache-export-"+time.Now().Format("20060102-150405")+".json")
	c.JSON(http.StatusOK, gin.H{
		"export_time": time.Now().Format(time.RFC3339),
		"task_type":   taskType,
		"total":       len(exportData),
		"entries":     exportData,
	})
}

// GetCacheTrend returns cache usage trend data
// GET /api/admin/cache/trend
func (h *CacheHandler) GetCacheTrend(c *gin.Context) {
	// Generate mock trend data for demonstration
	// In production, this would query actual historical data
	now := time.Now()
	hours := make([]string, 0)
	hitsData := make([]int, 0)
	missesData := make([]int, 0)

	for i := 23; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * time.Hour)
		hours = append(hours, t.Format("15:00"))
		// Mock data - in production this would be real data
		hitsData = append(hitsData, 50+i*2+int(20*float64(i%3)))
		missesData = append(missesData, 10+i+int(5*float64(i%2)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"hours":  hours,
			"hits":   hitsData,
			"misses": missesData,
			"summary": gin.H{
				"total_hits":   1500,
				"total_misses": 300,
				"hit_rate":     83.3,
				"avg_latency":  45,
			},
		},
	})
}

func extractCacheSummary(value interface{}) (string, string, string) {
	// taskType, userMessage, aiResponse
	var taskType string
	var userMsg string
	var aiResp string

	switch v := value.(type) {
	case map[string]interface{}:
		// task type
		if tt, ok := v["task_type"]; ok {
			taskType, _ = tt.(string)
		}
		if tt, ok := v["TaskType"]; ok && taskType == "" {
			taskType, _ = tt.(string)
		}
		// prompt / user message
		if p, ok := v["prompt"]; ok {
			userMsg, _ = p.(string)
		}
		if p, ok := v["Prompt"]; ok && userMsg == "" {
			userMsg, _ = p.(string)
		}
		// messages
		if userMsg == "" {
			if msgs, ok := v["messages"].([]interface{}); ok {
				for _, msg := range msgs {
					if msgMap, ok := msg.(map[string]interface{}); ok {
						if role, _ := msgMap["role"].(string); role == "user" {
							if content, ok := msgMap["content"].(string); ok {
								userMsg = content
								break
							}
						}
					}
				}
			}
		}
		// response body
		if body, ok := v["body"]; ok {
			switch b := body.(type) {
			case []byte:
				aiResp = extractAIFromBody(b)
			case string:
				aiResp = extractAIFromBody([]byte(b))
			}
		}
		// direct choices
		if aiResp == "" {
			if choices, ok := v["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if msg, ok := choice["message"].(map[string]interface{}); ok {
						if content, ok := msg["content"].(string); ok {
							aiResp = content
						}
					}
				}
			}
		}
	case map[interface{}]interface{}:
		converted := make(map[string]interface{}, len(v))
		for key, val := range v {
			if ks, ok := key.(string); ok {
				converted[ks] = val
			}
		}
		return extractCacheSummary(converted)
	}

	return taskType, truncatePreview(userMsg), truncatePreview(aiResp)
}

func extractAIFromBody(body []byte) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if choices, ok := payload["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content
				}
			}
		}
	}
	return ""
}

func truncatePreview(input string) string {
	if len(input) > 120 {
		return input[:120] + "..."
	}
	return input
}
