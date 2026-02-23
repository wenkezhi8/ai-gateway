package admin

import (
	"ai-gateway/internal/cache"
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CacheHandler handles cache management requests
type CacheHandler struct {
	manager *cache.Manager
	config  *CacheConfigRequest
	mu      sync.RWMutex
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(manager *cache.Manager) *CacheHandler {
	return &CacheHandler{
		manager: manager,
		config: &CacheConfigRequest{
			RequestTTL: 3600,
			ContextTTL: 1800,
			RouteTTL:   300,
			MaxSize:    10000,
		},
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
		response.RequestCache = CacheStatDetail{
			Hits:      stat.Hits,
			Misses:    stat.Misses,
			HitRate:   stat.HitRate,
			Size:      stat.TotalOperations,
			Evictions: stat.Evictions,
		}
	}

	if stat, ok := allStats["context"]; ok {
		response.ContextCache = CacheStatDetail{
			Hits:      stat.Hits,
			Misses:    stat.Misses,
			HitRate:   stat.HitRate,
			Size:      stat.TotalOperations,
			Evictions: stat.Evictions,
		}
	}

	if stat, ok := allStats["route"]; ok {
		response.RouteCache = CacheStatDetail{
			Hits:      stat.Hits,
			Misses:    stat.Misses,
			HitRate:   stat.HitRate,
			Size:      stat.TotalOperations,
			Evictions: stat.Evictions,
		}
	}

	if stat, ok := allStats["usage"]; ok {
		response.UsageCache = CacheStatDetail{
			Hits:      stat.Hits,
			Misses:    stat.Misses,
			HitRate:   stat.HitRate,
			Size:      stat.TotalOperations,
			Evictions: stat.Evictions,
		}
	}

	if stat, ok := allStats["response"]; ok {
		response.ResponseCache = CacheStatDetail{
			Hits:      stat.Hits,
			Misses:    stat.Misses,
			HitRate:   stat.HitRate,
			Size:      stat.TotalOperations,
			Evictions: stat.Evictions,
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

	// Update config (note: actual cache reconfiguration would require cache package support)
	h.config = &req

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Cache configuration updated",
			"config":  h.config,
		},
	})
}

// GetCacheConfig returns current cache configuration
// GET /api/admin/cache/config
func (h *CacheHandler) GetCacheConfig(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.config,
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
	err := h.manager.HealthCheck(ctx)

	healthy := err == nil

	response := gin.H{
		"healthy":   healthy,
		"timestamp": time.Now(),
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
