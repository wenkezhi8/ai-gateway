package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var cacheTaskTypeAssessor = routing.NewDifficultyAssessor()

// CacheHandler handles cache management requests.
type CacheHandler struct {
	manager           *cache.Manager
	settings          cache.CacheSettings
	modelMappingCache *cache.ModelMappingCache
	mu                sync.RWMutex
}

const emptyResponseCleanupInterval = 15 * time.Minute
const defaultRuntimeConfigPath = "./configs/config.json"

const (
	vectorEmbeddingProviderOllama = "ollama"
	taskTypeUnknown               = "unknown"
	taskTypeSourceHeuristic       = "heuristic"
)

// NewCacheHandler creates a new cache handler.
func NewCacheHandler(manager *cache.Manager) *CacheHandler {
	h := &CacheHandler{
		manager:  manager,
		settings: manager.GetSettings(),
	}
	h.startEmptyResponseCleaner()
	return h
}

// SetModelMappingCache sets the model mapping cache.
func (h *CacheHandler) SetModelMappingCache(mmc *cache.ModelMappingCache) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.modelMappingCache = mmc
}

func (h *CacheHandler) startEmptyResponseCleaner() {
	go func() {
		// Startup cleanup once, then periodic cleanup.
		ctx := context.Background()
		deleted, failed := h.cleanupEmptyResponseEntries(ctx)
		if deleted > 0 || failed > 0 {
			logrus.WithFields(logrus.Fields{
				"deleted": deleted,
				"failed":  failed,
			}).Info("Startup cleanup for empty response cache entries completed")
		}

		ticker := time.NewTicker(emptyResponseCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			deleted, failed := h.cleanupEmptyResponseEntries(ctx)
			if deleted > 0 || failed > 0 {
				logrus.WithFields(logrus.Fields{
					"deleted": deleted,
					"failed":  failed,
				}).Info("Periodic cleanup for empty response cache entries completed")
			}
		}
	}()
}

// GET /api/admin/cache/stats.
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

	if rc, ok := h.manager.Cache().(*cache.RedisCache); ok {
		if info, err := rc.GetClient().Info(context.Background(), "stats").Result(); err == nil {
			hits, misses := parseRedisHitStats(info)
			response.RedisHits = hits
			response.RedisMisses = misses
			if hits+misses > 0 {
				response.RedisHitRate = float64(hits) / float64(hits+misses)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func parseRedisHitStats(info string) (hits, misses int64) {
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "keyspace_hits:") {
			if _, err := fmt.Sscanf(trimmed, "keyspace_hits:%d", &hits); err != nil {
				continue
			}
		}
		if strings.HasPrefix(line, "keyspace_misses:") {
			if _, err := fmt.Sscanf(trimmed, "keyspace_misses:%d", &misses); err != nil {
				continue
			}
		}
	}
	return hits, misses
}

// DELETE /api/admin/cache.
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

// PUT /api/admin/cache/config.
//
//nolint:gocyclo // Backward-compatible field-by-field validation/update for many optional settings.
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
			value /= 100
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
	if req.VectorEnabled != nil {
		settings.VectorEnabled = *req.VectorEnabled
	}
	if req.VectorDimension != nil && *req.VectorDimension > 0 {
		settings.VectorDimension = *req.VectorDimension
	}
	if req.VectorQueryTimeoutMs != nil && *req.VectorQueryTimeoutMs > 0 {
		settings.VectorQueryTimeoutMs = *req.VectorQueryTimeoutMs
	}
	if req.VectorThresholds != nil {
		thresholds := make(map[string]float64, len(req.VectorThresholds))
		for k, v := range req.VectorThresholds {
			if v <= 0 || v > 1 {
				continue
			}
			thresholds[strings.ToLower(strings.TrimSpace(k))] = v
		}
		settings.VectorThresholds = thresholds
	}
	if req.VectorPipelineEnabled != nil {
		settings.VectorPipelineEnabled = *req.VectorPipelineEnabled
	}
	if req.VectorStandardKeyVersion != nil {
		value := strings.TrimSpace(*req.VectorStandardKeyVersion)
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_vector_standard_key_version",
					"message": "vector_standard_key_version cannot be empty",
				},
			})
			return
		}
		settings.VectorStandardKeyVersion = value
	}
	if req.VectorEmbeddingProvider != nil {
		value := strings.ToLower(strings.TrimSpace(*req.VectorEmbeddingProvider))
		if value != vectorEmbeddingProviderOllama {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_vector_embedding_provider",
					"message": "vector_embedding_provider must be ollama",
				},
			})
			return
		}
		settings.VectorEmbeddingProvider = value
	}
	if req.VectorOllamaBaseURL != nil {
		value := strings.TrimSpace(*req.VectorOllamaBaseURL)
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_vector_ollama_base_url",
					"message": "vector_ollama_base_url cannot be empty",
				},
			})
			return
		}
		settings.VectorOllamaBaseURL = value
	}
	if req.VectorOllamaEmbeddingModel != nil {
		value := strings.TrimSpace(*req.VectorOllamaEmbeddingModel)
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_vector_ollama_embedding_model",
					"message": "vector_ollama_embedding_model cannot be empty",
				},
			})
			return
		}
		settings.VectorOllamaEmbeddingModel = value
	}
	if req.VectorOllamaEmbeddingDimension != nil {
		if *req.VectorOllamaEmbeddingDimension <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_vector_ollama_embedding_dimension",
					"message": "vector_ollama_embedding_dimension must be positive",
				},
			})
			return
		}
		settings.VectorOllamaEmbeddingDimension = *req.VectorOllamaEmbeddingDimension
	}
	if req.VectorOllamaEmbeddingTimeoutMs != nil {
		if *req.VectorOllamaEmbeddingTimeoutMs <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_vector_ollama_embedding_timeout_ms",
					"message": "vector_ollama_embedding_timeout_ms must be positive",
				},
			})
			return
		}
		settings.VectorOllamaEmbeddingTimeoutMs = *req.VectorOllamaEmbeddingTimeoutMs
	}
	if req.VectorOllamaEndpointMode != nil {
		value := strings.ToLower(strings.TrimSpace(*req.VectorOllamaEndpointMode))
		if value != cache.OllamaEndpointModeAuto && value != cache.OllamaEndpointModeEmbed && value != cache.OllamaEndpointModeEmbeddings {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_vector_ollama_endpoint_mode",
					"message": "vector_ollama_endpoint_mode must be auto/embed/embeddings",
				},
			})
			return
		}
		settings.VectorOllamaEndpointMode = value
	}
	if req.VectorWritebackEnabled != nil {
		settings.VectorWritebackEnabled = *req.VectorWritebackEnabled
	}
	if req.ColdVectorEnabled != nil {
		settings.ColdVectorEnabled = *req.ColdVectorEnabled
	}
	if req.ColdVectorQueryEnabled != nil {
		settings.ColdVectorQueryEnabled = *req.ColdVectorQueryEnabled
	}
	if req.ColdVectorBackend != nil {
		backend := strings.ToLower(strings.TrimSpace(*req.ColdVectorBackend))
		if backend != cache.ColdVectorBackendSQLite && backend != cache.ColdVectorBackendQdrant {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_cold_backend",
					"message": "cold_vector_backend must be sqlite or qdrant",
				},
			})
			return
		}
		settings.ColdVectorBackend = backend
	}
	if req.ColdVectorDualWriteEnabled != nil {
		settings.ColdVectorDualWriteEnabled = *req.ColdVectorDualWriteEnabled
	}
	if req.ColdVectorSimilarityThreshold != nil {
		value := *req.ColdVectorSimilarityThreshold
		if value > 1 {
			value /= 100
		}
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		settings.ColdVectorSimilarityThreshold = value
	}
	if req.ColdVectorTopK != nil && *req.ColdVectorTopK > 0 {
		settings.ColdVectorTopK = *req.ColdVectorTopK
	}
	if req.HotMemoryHighWatermarkPercent != nil {
		value := *req.HotMemoryHighWatermarkPercent
		if value < 1 {
			value = 1
		}
		if value > 100 {
			value = 100
		}
		settings.HotMemoryHighWatermarkPercent = value
	}
	if req.HotMemoryReliefPercent != nil {
		value := *req.HotMemoryReliefPercent
		if value < 1 {
			value = 1
		}
		if value > 100 {
			value = 100
		}
		settings.HotMemoryReliefPercent = value
	}
	if settings.HotMemoryReliefPercent >= settings.HotMemoryHighWatermarkPercent {
		settings.HotMemoryReliefPercent = settings.HotMemoryHighWatermarkPercent - 5
		if settings.HotMemoryReliefPercent < 1 {
			settings.HotMemoryReliefPercent = 1
		}
	}
	if req.HotToColdBatchSize != nil && *req.HotToColdBatchSize > 0 {
		settings.HotToColdBatchSize = *req.HotToColdBatchSize
	}
	if req.HotToColdIntervalSeconds != nil && *req.HotToColdIntervalSeconds > 0 {
		settings.HotToColdIntervalSeconds = *req.HotToColdIntervalSeconds
	}
	if req.ColdVectorQdrantURL != nil {
		settings.ColdVectorQdrantURL = strings.TrimSpace(*req.ColdVectorQdrantURL)
	}
	if req.ColdVectorQdrantAPIKey != nil {
		settings.ColdVectorQdrantAPIKey = strings.TrimSpace(*req.ColdVectorQdrantAPIKey)
	}
	if req.ColdVectorQdrantCollection != nil {
		settings.ColdVectorQdrantCollection = strings.TrimSpace(*req.ColdVectorQdrantCollection)
	}
	if req.ColdVectorQdrantTimeoutMs != nil && *req.ColdVectorQdrantTimeoutMs > 0 {
		settings.ColdVectorQdrantTimeoutMs = *req.ColdVectorQdrantTimeoutMs
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
	if tiered := h.manager.GetTieredVectorStore(); tiered != nil {
		tiered.UpdateConfig(cache.TieredConfigFromSettings(settings))
		qdrantStore := cache.NewQdrantColdVectorStore(cache.QdrantColdVectorStoreConfig{
			URL:        settings.ColdVectorQdrantURL,
			APIKey:     settings.ColdVectorQdrantAPIKey,
			Collection: settings.ColdVectorQdrantCollection,
			Timeout:    time.Duration(settings.ColdVectorQdrantTimeoutMs) * time.Millisecond,
			Dimension:  settings.VectorDimension,
		})
		tiered.SetColdStore(cache.ColdVectorBackendQdrant, qdrantStore)
	}

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

	if err := persistVectorCacheSettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "persist_config_failed",
				"message": err.Error(),
			},
		})
		return
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

// GET /api/admin/cache/vector/tier/stats.
func (h *CacheHandler) GetVectorTierStats(c *gin.Context) {
	store := h.manager.GetTieredVectorStore()
	if store == nil {
		respondVectorStoreUnavailable(c, "tiered vector store is not initialized")
		return
	}

	stats, err := store.TierStats(c.Request.Context())
	if err != nil {
		respondVectorStoreStatsError(c, "vector_tier_stats_failed", err)
		return
	}

	respondVectorStoreStatsSuccess(c, stats)
}

func respondVectorStoreUnavailable(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled": false,
			"message": message,
		},
	})
}

func respondVectorStoreStatsError(c *gin.Context, code string, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": err.Error(),
		},
	})
}

func respondVectorStoreStatsSuccess(c *gin.Context, stats any) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// POST /api/admin/cache/vector/tier/migrate.
func (h *CacheHandler) TriggerVectorTierMigrate(c *gin.Context) {
	store := h.manager.GetTieredVectorStore()
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "vector_tier_unavailable",
				"message": "tiered vector store is not initialized",
			},
		})
		return
	}

	result, err := store.TriggerMigrate(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "vector_tier_migrate_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// POST /api/admin/cache/vector/tier/promote.
func (h *CacheHandler) PromoteVectorTierEntry(c *gin.Context) {
	store := h.manager.GetTieredVectorStore()
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "vector_tier_unavailable",
				"message": "tiered vector store is not initialized",
			},
		})
		return
	}

	var req struct {
		CacheKey string `json:"cache_key"`
	}
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
	req.CacheKey = strings.TrimSpace(req.CacheKey)
	if req.CacheKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_cache_key",
				"message": "cache_key is required",
			},
		})
		return
	}

	if err := store.Promote(c.Request.Context(), req.CacheKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "vector_tier_promote_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cache_key": req.CacheKey,
			"message":   "promotion completed",
		},
	})
}

// GET /api/admin/cache/config.
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

// DELETE /api/admin/cache/provider/:provider.
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

// DELETE /api/admin/cache/model/:model.
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

// GET /api/admin/cache/health.
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

func persistVectorCacheSettings(settings *cache.CacheSettings) error {
	configPath := strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if configPath == "" {
		configPath = defaultRuntimeConfigPath
	}
	configPath = filepath.Clean(configPath)

	root, err := loadConfigMap(configPath)
	if err != nil {
		return fmt.Errorf("load config file: %w", err)
	}

	vectorCfg, ok := root["vector_cache"].(map[string]any)
	if !ok || vectorCfg == nil {
		vectorCfg = map[string]any{}
	}

	vectorCfg["enabled"] = settings.VectorEnabled
	vectorCfg["dimension"] = settings.VectorDimension
	vectorCfg["query_timeout_ms"] = settings.VectorQueryTimeoutMs
	vectorCfg["pipeline_enabled"] = settings.VectorPipelineEnabled
	vectorCfg["standard_key_version"] = settings.VectorStandardKeyVersion
	vectorCfg["embedding_provider"] = settings.VectorEmbeddingProvider
	vectorCfg["ollama_base_url"] = settings.VectorOllamaBaseURL
	vectorCfg["ollama_embedding_model"] = settings.VectorOllamaEmbeddingModel
	vectorCfg["ollama_embedding_dimension"] = settings.VectorOllamaEmbeddingDimension
	vectorCfg["ollama_embedding_timeout_ms"] = settings.VectorOllamaEmbeddingTimeoutMs
	vectorCfg["ollama_endpoint_mode"] = settings.VectorOllamaEndpointMode
	vectorCfg["writeback_enabled"] = settings.VectorWritebackEnabled
	if len(settings.VectorThresholds) > 0 {
		vectorCfg["thresholds"] = settings.VectorThresholds
	}
	vectorCfg["cold_vector_enabled"] = settings.ColdVectorEnabled
	vectorCfg["cold_vector_query_enabled"] = settings.ColdVectorQueryEnabled
	vectorCfg["cold_vector_backend"] = settings.ColdVectorBackend
	vectorCfg["cold_vector_dual_write_enabled"] = settings.ColdVectorDualWriteEnabled
	vectorCfg["cold_vector_similarity_threshold"] = settings.ColdVectorSimilarityThreshold
	vectorCfg["cold_vector_top_k"] = settings.ColdVectorTopK
	vectorCfg["hot_memory_high_watermark_percent"] = settings.HotMemoryHighWatermarkPercent
	vectorCfg["hot_memory_relief_percent"] = settings.HotMemoryReliefPercent
	vectorCfg["hot_to_cold_batch_size"] = settings.HotToColdBatchSize
	vectorCfg["hot_to_cold_interval_seconds"] = settings.HotToColdIntervalSeconds
	vectorCfg["cold_vector_qdrant_url"] = settings.ColdVectorQdrantURL
	vectorCfg["cold_vector_qdrant_api_key"] = settings.ColdVectorQdrantAPIKey
	vectorCfg["cold_vector_qdrant_collection"] = settings.ColdVectorQdrantCollection
	vectorCfg["cold_vector_qdrant_timeout_ms"] = settings.ColdVectorQdrantTimeoutMs

	root["vector_cache"] = vectorCfg
	if err := writeConfigMapAtomic(configPath, root); err != nil {
		return fmt.Errorf("persist config file: %w", err)
	}
	return nil
}

func loadConfigMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	root := map[string]any{}
	if strings.TrimSpace(string(data)) == "" {
		return root, nil
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	return root, nil
}

func writeConfigMapAtomic(path string, root map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	mode := os.FileMode(0o644)
	if stat, err := os.Stat(path); err == nil {
		mode = stat.Mode()
	}

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// GET /api/admin/cache/summary.
func (h *CacheHandler) GetCacheSummary(c *gin.Context) {
	summary := h.manager.Summary()

	c.Data(http.StatusOK, "application/json", summary)
}

type vectorPipelineTestRequest struct {
	Query         string  `json:"query" binding:"required"`
	TaskType      string  `json:"task_type"`
	TopK          int     `json:"top_k"`
	MinSimilarity float64 `json:"min_similarity"`
}

func (h *CacheHandler) newOllamaEmbeddingServiceFromSettings(settings *cache.CacheSettings) *cache.OllamaEmbeddingService {
	return cache.NewOllamaEmbeddingService(cache.OllamaEmbeddingConfig{
		BaseURL:      settings.VectorOllamaBaseURL,
		Model:        settings.VectorOllamaEmbeddingModel,
		Timeout:      time.Duration(settings.VectorOllamaEmbeddingTimeoutMs) * time.Millisecond,
		EndpointMode: settings.VectorOllamaEndpointMode,
	})
}

func (h *CacheHandler) resolveVectorThreshold(taskType string, settings *cache.CacheSettings) float64 {
	key := strings.ToLower(strings.TrimSpace(taskType))
	switch key {
	case "math":
		key = "calc"
	case "fact", "reasoning", "code", "long_text":
		key = "qa"
	}
	if settings.VectorThresholds != nil {
		if value, ok := settings.VectorThresholds[key]; ok && value > 0 && value <= 1 {
			return value
		}
	}
	if settings.SimilarityThreshold > 0 && settings.SimilarityThreshold <= 1 {
		return settings.SimilarityThreshold
	}
	return 0.92
}

// GET /api/admin/cache/vector/pipeline/health.
func (h *CacheHandler) GetVectorPipelineHealth(c *gin.Context) {
	settings := h.manager.GetSettings()
	store := h.manager.GetVectorStore()

	data := gin.H{
		"enabled":                    settings.VectorEnabled && settings.VectorPipelineEnabled,
		"vector_enabled":             settings.VectorEnabled,
		"pipeline_enabled":           settings.VectorPipelineEnabled,
		"embedding_provider":         settings.VectorEmbeddingProvider,
		"ollama_base_url":            settings.VectorOllamaBaseURL,
		"ollama_embedding_model":     settings.VectorOllamaEmbeddingModel,
		"ollama_embedding_dimension": settings.VectorOllamaEmbeddingDimension,
		"ollama_endpoint_mode":       settings.VectorOllamaEndpointMode,
		"writeback_enabled":          settings.VectorWritebackEnabled,
	}

	if !settings.VectorEnabled || !settings.VectorPipelineEnabled {
		data["healthy"] = false
		data["message"] = "vector pipeline disabled"
		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
		return
	}

	start := time.Now()
	embedder := h.newOllamaEmbeddingServiceFromSettings(&settings)
	embedding, err := embedder.GetEmbedding(c.Request.Context(), "vector pipeline health check")
	latency := time.Since(start).Milliseconds()
	data["embedding_latency_ms"] = latency
	data["embedding_dimension_actual"] = len(embedding)

	if err != nil {
		data["healthy"] = false
		data["message"] = err.Error()
		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
		return
	}

	var stats cache.VectorStoreStats
	var statsErr error
	if store != nil {
		stats, statsErr = store.Stats(c.Request.Context())
	}
	if statsErr != nil {
		data["healthy"] = false
		data["message"] = statsErr.Error()
		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
		return
	}

	indexDim := stats.Dimension
	if indexDim == 0 {
		indexDim = settings.VectorDimension
	}

	data["vector_index_dimension"] = indexDim
	data["dimension_match"] = len(embedding) == indexDim
	data["healthy"] = len(embedding) == indexDim
	if len(embedding) == indexDim {
		data["message"] = "ok"
	} else {
		data["message"] = "embedding dimension does not match vector index"
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// POST /api/admin/cache/vector/pipeline/test.
func (h *CacheHandler) TestVectorPipeline(c *gin.Context) {
	var req vectorPipelineTestRequest
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

	settings := h.manager.GetSettings()
	if !settings.VectorEnabled || !settings.VectorPipelineEnabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "vector_pipeline_disabled",
				"message": "vector pipeline is disabled",
			},
		})
		return
	}

	taskType := strings.ToLower(strings.TrimSpace(req.TaskType))
	if taskType == "" {
		taskType = taskTypeUnknown
	}
	normalizer := cache.NewTextNormalizer()
	normalizedQuery := normalizer.Normalize(req.Query)
	if normalizedQuery == "" {
		normalizedQuery = strings.TrimSpace(req.Query)
	}
	standardKey := cache.BuildTaskTypeStandardKey(taskType, normalizedQuery)

	embedder := h.newOllamaEmbeddingServiceFromSettings(&settings)
	embedStart := time.Now()
	embedding, err := embedder.GetEmbedding(c.Request.Context(), normalizedQuery)
	embedLatency := time.Since(embedStart).Milliseconds()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "vector_embedding_failed",
				"message": err.Error(),
			},
		})
		return
	}

	store := h.manager.GetVectorStore()
	var hits []cache.VectorSearchHit
	searchLatency := int64(0)
	if store != nil {
		topK := req.TopK
		if topK <= 0 {
			topK = 5
		}
		if topK > 20 {
			topK = 20
		}
		minSimilarity := req.MinSimilarity
		if minSimilarity <= 0 || minSimilarity > 1 {
			minSimilarity = h.resolveVectorThreshold(taskType, &settings)
		}

		searchStart := time.Now()
		hits, err = store.VectorSearch(c.Request.Context(), taskType, embedding, topK, minSimilarity)
		searchLatency = time.Since(searchStart).Milliseconds()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "vector_search_failed",
					"message": err.Error(),
				},
			})
			return
		}
	}

	resultHits := make([]gin.H, 0, len(hits))
	for _, hit := range hits {
		resultHits = append(resultHits, gin.H{
			"cache_key":    hit.CacheKey,
			"intent":       hit.Intent,
			"similarity":   hit.Similarity,
			"score":        hit.Score,
			"response_raw": string(hit.Response),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task_type":             taskType,
			"normalized_query":      normalizedQuery,
			"standard_key":          standardKey,
			"embedding_dimension":   len(embedding),
			"embedding_latency_ms":  embedLatency,
			"vector_search_latency": searchLatency,
			"hits":                  resultHits,
		},
	})
}

// GET /api/admin/cache/vector/stats.
func (h *CacheHandler) GetVectorStats(c *gin.Context) {
	store := h.manager.GetVectorStore()
	if store == nil {
		respondVectorStoreUnavailable(c, "vector store is not initialized")
		return
	}

	stats, err := store.Stats(c.Request.Context())
	if err != nil {
		respondVectorStoreStatsError(c, "vector_stats_failed", err)
		return
	}

	respondVectorStoreStatsSuccess(c, stats)
}

// POST /api/admin/cache/vector/rebuild.
func (h *CacheHandler) RebuildVectorIndex(c *gin.Context) {
	store := h.manager.GetVectorStore()
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "vector_store_unavailable",
				"message": "vector store is not initialized",
			},
		})
		return
	}

	if err := store.RebuildIndex(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "vector_rebuild_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "vector index rebuilt",
		},
	})
}

type SemanticSignatureItem struct {
	Signature string  `json:"signature"`
	TaskType  string  `json:"task_type"`
	Model     string  `json:"model"`
	Provider  string  `json:"provider"`
	HitCount  int     `json:"hit_count"`
	Quality   float64 `json:"quality_score"`
}

// GET /api/admin/cache/semantic-signatures.
func (h *CacheHandler) GetSemanticSignatures(c *gin.Context) {
	semanticCache := h.manager.GetSemanticCache()
	if semanticCache == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []SemanticSignatureItem{}})
		return
	}

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
		if limit < 1 {
			limit = 1
		}
		if limit > 100 {
			limit = 100
		}
	}

	entries := semanticCache.GetEntries()
	items := make([]SemanticSignatureItem, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		items = append(items, SemanticSignatureItem{
			Signature: entry.Query,
			TaskType:  entry.TaskType,
			Model:     entry.Model,
			Provider:  entry.Provider,
			HitCount:  entry.HitCount,
			Quality:   entry.QualityScore,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].HitCount == items[j].HitCount {
			return items[i].Quality > items[j].Quality
		}
		return items[i].HitCount > items[j].HitCount
	})
	if len(items) > limit {
		items = items[:limit]
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// GET /api/admin/cache/quality-config.
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

// UpdateCacheQualityConfigRequest represents quality config update request.
type UpdateCacheQualityConfigRequest struct {
	MinQualityScore *float64 `json:"min_quality_score"`
}

// PUT /api/admin/cache/quality-config.
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

// POST /api/admin/cache/invalidate-low-quality.
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

// GET /api/admin/cache/rules.
func (h *CacheHandler) GetCacheRules(c *gin.Context) {
	rules := cache.GetRuleStore().List()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rules,
	})
}

// CreateCacheRuleRequest represents create cache rule request.
type CreateCacheRuleRequest struct {
	Pattern     string `json:"pattern" binding:"required"`
	ModelFilter string `json:"model_filter"`
	TTL         int    `json:"ttl" binding:"required"`
	Priority    string `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// POST /api/admin/cache/rules.
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

// UpdateCacheRuleRequest represents update cache rule request.
type UpdateCacheRuleRequest struct {
	Pattern     *string `json:"pattern"`
	ModelFilter *string `json:"model_filter"`
	TTL         *int    `json:"ttl"`
	Priority    *string `json:"priority"`
	Enabled     *bool   `json:"enabled"`
}

// PUT /api/admin/cache/rules/:id.
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

// DELETE /api/admin/cache/rules/:id.
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

// CacheEntry represents a cache entry for display.
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

// GET /api/admin/cache/entries.
//
//nolint:gocyclo // Supports many query options and response shaping for admin UI.
func (h *CacheHandler) GetCacheEntries(c *gin.Context) {
	cacheType := c.Query("type")
	search := c.Query("search")
	taskType := normalizeTaskType(c.Query("task_type"))
	aggregate := c.Query("aggregate") == "1" || strings.EqualFold(c.Query("aggregate"), "true")
	readableOnly := c.DefaultQuery("readable_only", "1") == "1" || strings.EqualFold(c.Query("readable_only"), "true")
	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		parsedPage, err := strconv.Atoi(p)
		if err == nil {
			page = parsedPage
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		parsedPageSize, err := strconv.Atoi(ps)
		if err == nil {
			pageSize = parsedPageSize
		}
	}

	entries := h.manager.ListEntries(cacheType, search)
	ctx := context.Background()
	for _, entry := range entries {
		detail, err := h.manager.GetEntryDetail(ctx, entry.Key)
		if err != nil {
			continue
		}
		enrichEntryFromDetail(entry, detail)
		entryTaskType, userMsg, aiResp, taskTypeSource := extractCacheSummary(detail.Value)
		entry.TaskType = resolveTaskType(entry.TaskType, entryTaskType, userMsg)
		entry.TaskTypeSource = resolveTaskTypeSource(entry.TaskTypeSource, taskTypeSource, entry.TaskType, userMsg)
		if userMsg != "" {
			entry.UserMessage = userMsg
		}
		if aiResp != "" {
			entry.AIResponse = aiResp
		}
	}

	if taskType != "" {
		filtered := make([]*cache.CacheEntryInfo, 0, len(entries))
		for _, entry := range entries {
			if entry.TaskType == taskType {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	if readableOnly {
		entries = filterReadableEntries(entries)
	}

	if aggregate {
		entries = aggregateCacheEntries(entries)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"entries":       pageEntries,
			"total":         total,
			"page":          page,
			"page_size":     pageSize,
			"aggregate":     aggregate,
			"readable_only": readableOnly,
		},
	})
}

// GET /api/admin/cache/entries/*.
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
	detail, err := h.manager.GetEntryDetail(ctx, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   gin.H{"code": "not_found", "message": "Cache entry not found"},
		})
		return
	}

	entry := &cache.CacheEntryInfo{
		Key:       detail.Key,
		Type:      detail.Type,
		Value:     detail.Value,
		Size:      detail.Size,
		Hits:      detail.Hits,
		CreatedAt: detail.CreatedAt,
		ExpiresAt: detail.ExpiresAt,
		TTL:       detail.TTL,
	}

	enrichEntryFromDetail(entry, detail)

	entryTaskType, userMsg, aiResp, taskTypeSource := extractCacheSummary(detail.Value)
	entry.TaskType = resolveTaskType(entry.TaskType, entryTaskType, userMsg)
	entry.TaskTypeSource = resolveTaskTypeSource(entry.TaskTypeSource, taskTypeSource, entry.TaskType, userMsg)
	if userMsg != "" {
		entry.UserMessage = userMsg
	}
	if aiResp != "" {
		entry.AIResponse = aiResp
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entry,
	})
}

// DELETE /api/admin/cache/entries/*.
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

// POST /api/admin/cache/entries/delete-group.
func (h *CacheHandler) DeleteCacheEntryGroup(c *gin.Context) {
	type reqBody struct {
		TaskType    string `json:"task_type"`
		UserMessage string `json:"user_message"`
		AIResponse  string `json:"ai_response"`
		Model       string `json:"model"`
		Provider    string `json:"provider"`
	}

	var req reqBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "invalid_request", "message": err.Error()},
		})
		return
	}

	normalizedTask := normalizeTaskType(req.TaskType)
	normalizedUser := strings.TrimSpace(req.UserMessage)
	normalizedAI := strings.TrimSpace(req.AIResponse)
	normalizedModel := strings.TrimSpace(req.Model)
	normalizedProvider := strings.TrimSpace(req.Provider)

	ctx := context.Background()
	entries := h.manager.ListEntries("response", "")
	deleted := 0

	for _, entry := range entries {
		detail, err := h.manager.GetEntryDetail(ctx, entry.Key)
		if err != nil {
			continue
		}
		enrichEntryFromDetail(entry, detail)
		entryTaskType, userMsg, aiResp, taskTypeSource := extractCacheSummary(detail.Value)
		entry.TaskType = resolveTaskType(entry.TaskType, entryTaskType, userMsg)
		entry.TaskTypeSource = resolveTaskTypeSource(entry.TaskTypeSource, taskTypeSource, entry.TaskType, userMsg)
		if userMsg != "" {
			entry.UserMessage = userMsg
		}
		if aiResp != "" {
			entry.AIResponse = aiResp
		}

		if normalizeTaskType(entry.TaskType) != normalizedTask {
			continue
		}
		if strings.TrimSpace(entry.UserMessage) != normalizedUser {
			continue
		}
		if strings.TrimSpace(entry.AIResponse) != normalizedAI {
			continue
		}
		if normalizedModel != "" && strings.TrimSpace(entry.Model) != normalizedModel {
			continue
		}
		if normalizedProvider != "" && strings.TrimSpace(entry.Provider) != normalizedProvider {
			continue
		}

		if err := h.manager.Cache().Delete(ctx, entry.Key); err == nil {
			deleted++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted": deleted,
		},
	})
}

// POST /api/admin/cache/entries/cleanup-invalid.
func (h *CacheHandler) CleanupInvalidEntries(c *gin.Context) {
	ctx := context.Background()
	entries := h.manager.ListEntries("", "")

	deleted := 0
	failed := 0
	for _, entry := range entries {
		detail, err := h.manager.GetEntryDetail(ctx, entry.Key)
		if err != nil {
			failed++
			continue
		}
		enrichEntryFromDetail(entry, detail)
		entryTaskType, userMsg, aiResp, taskTypeSource := extractCacheSummary(detail.Value)
		entry.TaskType = resolveTaskType(entry.TaskType, entryTaskType, userMsg)
		entry.TaskTypeSource = resolveTaskTypeSource(entry.TaskTypeSource, taskTypeSource, entry.TaskType, userMsg)
		if userMsg != "" {
			entry.UserMessage = userMsg
		}
		if aiResp != "" {
			entry.AIResponse = aiResp
		}

		if !isInvalidEntry(entry) {
			continue
		}

		if err := h.manager.Cache().Delete(ctx, entry.Key); err != nil {
			failed++
			continue
		}
		deleted++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted": deleted,
			"failed":  failed,
		},
	})
}

// POST /api/admin/cache/entries/cleanup-empty.
func (h *CacheHandler) CleanupEmptyResponseEntries(c *gin.Context) {
	ctx := context.Background()
	deleted, failed := h.cleanupEmptyResponseEntries(ctx)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted": deleted,
			"failed":  failed,
		},
	})
}

func (h *CacheHandler) cleanupEmptyResponseEntries(ctx context.Context) (deleted, failed int) {
	entries := h.manager.ListEntries("response", "")
	for _, entry := range entries {
		if entry == nil {
			continue
		}

		detail, err := h.manager.GetEntryDetail(ctx, entry.Key)
		if err != nil {
			failed++
			continue
		}

		_, _, aiResp, _ := extractCacheSummary(detail.Value)
		if strings.TrimSpace(aiResp) != "" {
			continue
		}

		if err := h.manager.Cache().Delete(ctx, entry.Key); err != nil {
			failed++
			continue
		}
		deleted++
	}

	return deleted, failed
}

// AddTestCacheEntryRequest represents request for adding test cache.
type AddTestCacheEntryRequest struct {
	TaskType    string `json:"task_type" binding:"required"`
	UserMessage string `json:"user_message" binding:"required"`
	AIResponse  string `json:"ai_response" binding:"required"`
	Model       string `json:"model"`
	Provider    string `json:"provider"`
	TTL         int    `json:"ttl"` // hours
}

func (h *CacheHandler) setTestCacheValue(
	ctx context.Context,
	key string,
	value interface{},
	ttl time.Duration,
	req *AddTestCacheEntryRequest,
) error {
	if mc, ok := h.manager.Cache().(*cache.MemoryCache); ok {
		return mc.SetWithTaskType(ctx, key, value, ttl, req.Model, req.Provider, req.TaskType, "manual")
	}
	return h.manager.Cache().Set(ctx, key, value, ttl)
}

// POST /api/admin/cache/test-entry.
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
	if err := h.setTestCacheValue(ctx, reqKey, requestData, ttl, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "set_request_cache_failed", "message": err.Error()},
		})
		return
	}

	// Store response cache
	respKey := "ai-response:test:" + key
	if err := h.setTestCacheValue(ctx, respKey, responseData, ttl, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "set_response_cache_failed", "message": err.Error()},
		})
		return
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

// GET /api/admin/cache/export.
func (h *CacheHandler) ExportCacheEntries(c *gin.Context) {
	taskType := normalizeTaskType(c.Query("task_type"))

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
			entryTaskType := entry.TaskType
			if entryTaskType == "" {
				if extracted, _, _, _ := extractCacheSummary(detail.Value); extracted != "" {
					entryTaskType = extracted
				}
			}
			exportData = append(exportData, map[string]interface{}{
				"key":        entry.Key,
				"type":       entry.Type,
				"task_type":  entryTaskType,
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

// GET /api/admin/cache/trend.
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

//nolint:gocyclo // Handles legacy cache payload variants across multiple schema versions.
func extractCacheSummary(value interface{}) (taskType, userMsg, aiResp, taskTypeSource string) {
	switch v := value.(type) {
	case map[string]interface{}:
		// task type
		if tt, ok := v["task_type"]; ok {
			if parsed, ok := tt.(string); ok {
				taskType = parsed
			}
		}
		if tt, ok := v["TaskType"]; ok && taskType == "" {
			if parsed, ok := tt.(string); ok {
				taskType = parsed
			}
		}
		if ts, ok := v["task_type_source"]; ok {
			if parsed, ok := ts.(string); ok {
				taskTypeSource = parsed
			}
		}
		if ts, ok := v["TaskTypeSource"]; ok && taskTypeSource == "" {
			if parsed, ok := ts.(string); ok {
				taskTypeSource = parsed
			}
		}
		// prompt / user message
		if p, ok := v["prompt"]; ok {
			if parsed, ok := p.(string); ok {
				userMsg = parsed
			}
		}
		if p, ok := v["Prompt"]; ok && userMsg == "" {
			if parsed, ok := p.(string); ok {
				userMsg = parsed
			}
		}
		if p, ok := v["user_message"]; ok && userMsg == "" {
			if parsed, ok := p.(string); ok {
				userMsg = parsed
			}
		}
		if p, ok := v["userMessage"]; ok && userMsg == "" {
			if parsed, ok := p.(string); ok {
				userMsg = parsed
			}
		}
		// messages
		if userMsg == "" {
			if msgs, ok := v["messages"].([]interface{}); ok {
				for _, msg := range msgs {
					if msgMap, ok := msg.(map[string]interface{}); ok {
						if role, ok := msgMap["role"].(string); ok && role == "user" {
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
			aiResp = extractAIFromAny(body)
		}
		if body, ok := v["Body"]; ok && aiResp == "" {
			aiResp = extractAIFromAny(body)
		}
		if resp, ok := v["response"]; ok && aiResp == "" {
			aiResp = extractAIFromAny(resp)
		}
		if resp, ok := v["Response"]; ok && aiResp == "" {
			aiResp = extractAIFromAny(resp)
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

	return taskType, truncatePreview(userMsg), truncatePreview(aiResp), taskTypeSource
}

func resolveTaskType(metaTaskType, payloadTaskType, userMsg string) string {
	normalizedMeta := normalizeTaskType(metaTaskType)
	normalizedPayload := normalizeTaskType(payloadTaskType)

	if normalizedMeta != "" && normalizedMeta != taskTypeUnknown {
		return normalizedMeta
	}
	if normalizedPayload != "" && normalizedPayload != taskTypeUnknown {
		return normalizedPayload
	}

	if strings.TrimSpace(userMsg) != "" {
		inferred := normalizeTaskType(string(cacheTaskTypeAssessor.DetectTaskType(userMsg)))
		if inferred != "" && inferred != taskTypeUnknown {
			return inferred
		}
	}

	if normalizedMeta == taskTypeUnknown || normalizedPayload == taskTypeUnknown {
		return taskTypeUnknown
	}

	return taskTypeUnknown
}

func normalizeTaskType(taskType string) string {
	normalized := strings.ToLower(strings.TrimSpace(taskType))
	switch normalized {
	case "":
		return ""
	case "other":
		return taskTypeUnknown
	case "long_context":
		return "long_text"
	default:
		return normalized
	}
}

func resolveTaskTypeSource(metaSource, payloadSource, resolvedTaskType, userMsg string) string {
	normalizedMeta := normalizeTaskTypeSource(metaSource)
	if normalizedMeta != "" {
		return normalizedMeta
	}

	normalizedPayload := normalizeTaskTypeSource(payloadSource)
	if normalizedPayload != "" {
		return normalizedPayload
	}

	if normalizeTaskType(resolvedTaskType) == taskTypeUnknown {
		return taskTypeUnknown
	}

	if strings.TrimSpace(userMsg) != "" {
		inferred := cacheTaskTypeAssessor.DetectTaskType(userMsg)
		if inferred != routing.TaskTypeUnknown {
			return taskTypeSourceHeuristic
		}
	}

	return "legacy"
}

func normalizeTaskTypeSource(source string) string {
	normalized := strings.ToLower(strings.TrimSpace(source))
	switch normalized {
	case "":
		return ""
	case "llm", "model", vectorEmbeddingProviderOllama, "classifier":
		return vectorEmbeddingProviderOllama
	case "heuristic", "rule", "keyword":
		return taskTypeSourceHeuristic
	case "fallback":
		return "fallback"
	case "manual":
		return "manual"
	default:
		return normalized
	}
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

func extractAIFromAny(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case []byte:
		return extractAIFromBody(v)
	case string:
		return extractAIFromBody([]byte(v))
	case map[string]interface{}, map[interface{}]interface{}, []interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return extractAIFromBody(data)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return extractAIFromBody(data)
	}
}

func truncatePreview(input string) string {
	if len(input) > 120 {
		return input[:120] + "..."
	}
	return input
}

//nolint:gocyclo // Merges metadata from heterogeneous cache entry shapes.
func enrichEntryFromDetail(entry *cache.CacheEntryInfo, detail *cache.CacheEntryDetail) {
	if entry == nil || detail == nil {
		return
	}

	if entry.Hits == 0 && detail.Hits > 0 {
		entry.Hits = detail.Hits
	}
	if entry.TTL == 0 && detail.TTL > 0 {
		entry.TTL = detail.TTL
	}
	if entry.ExpiresAt == nil && detail.ExpiresAt != nil {
		entry.ExpiresAt = detail.ExpiresAt
	}

	if entry.CreatedAt.IsZero() {
		if !detail.CreatedAt.IsZero() {
			entry.CreatedAt = detail.CreatedAt
		} else if detail.ExpiresAt != nil && detail.TTL > 0 {
			entry.CreatedAt = detail.ExpiresAt.Add(-time.Duration(detail.TTL) * time.Second)
		}
	}

	model, provider := extractModelProvider(detail.Value)
	if entry.Model == "" && model != "" {
		entry.Model = model
	}
	if entry.Provider == "" && provider != "" {
		entry.Provider = provider
	}

	if modelStats, ok := extractHitModels(detail.Value); ok {
		entry.ModelStats = modelStats
		entry.Model = selectPrimaryModel(modelStats)
	}

	if hits, ok := extractHitCountFromValue(detail.Value); ok {
		entry.Hits = hits
		entry.HitRecorded = true
	}
}

func extractHitModels(value interface{}) (map[string]int, bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		raw, ok := v["hit_models"]
		if !ok {
			raw, ok = v["HitModels"]
		}
		if !ok {
			return nil, false
		}
		return parseHitModels(raw)
	case map[interface{}]interface{}:
		converted := make(map[string]interface{}, len(v))
		for key, val := range v {
			if ks, ok := key.(string); ok {
				converted[ks] = val
			}
		}
		return extractHitModels(converted)
	default:
		return nil, false
	}
}

func parseHitModels(value interface{}) (map[string]int, bool) {
	result := map[string]int{}
	switch v := value.(type) {
	case map[string]interface{}:
		for k, raw := range v {
			if n, ok := numberToInt(raw); ok {
				result[k] = n
			}
		}
	case map[interface{}]interface{}:
		for key, raw := range v {
			ks, ok := key.(string)
			if !ok {
				continue
			}
			if n, ok := numberToInt(raw); ok {
				result[ks] = n
			}
		}
	default:
		return nil, false
	}
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func extractModelProvider(value interface{}) (model, provider string) {
	switch v := value.(type) {
	case map[string]interface{}:
		if m, ok := v["model"].(string); ok {
			model = m
		}
		if m, ok := v["Model"].(string); ok && model == "" {
			model = m
		}
		if p, ok := v["provider"].(string); ok {
			provider = p
		}
		if p, ok := v["Provider"].(string); ok && provider == "" {
			provider = p
		}

		if model == "" {
			if body, ok := v["body"]; ok {
				switch b := body.(type) {
				case []byte:
					model = extractModelFromBody(b)
				case string:
					model = extractModelFromBody([]byte(b))
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
		return extractModelProvider(converted)
	}

	return model, provider
}

func extractModelFromBody(body []byte) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if model, ok := payload["model"].(string); ok {
		return model
	}
	return ""
}

func extractHitCountFromValue(value interface{}) (int, bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		if raw, ok := v["hit_count"]; ok {
			if n, ok := numberToInt(raw); ok {
				return n, true
			}
		}
		if raw, ok := v["HitCount"]; ok {
			if n, ok := numberToInt(raw); ok {
				return n, true
			}
		}
		return 0, false
	case map[interface{}]interface{}:
		converted := make(map[string]interface{}, len(v))
		for key, val := range v {
			if ks, ok := key.(string); ok {
				converted[ks] = val
			}
		}
		return extractHitCountFromValue(converted)
	default:
		return 0, false
	}
}

func numberToInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func filterReadableEntries(entries []*cache.CacheEntryInfo) []*cache.CacheEntryInfo {
	filtered := make([]*cache.CacheEntryInfo, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if isInvalidEntry(entry) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func isInvalidEntry(entry *cache.CacheEntryInfo) bool {
	if entry == nil {
		return true
	}

	userMsg := strings.TrimSpace(entry.UserMessage)
	aiResp := strings.TrimSpace(entry.AIResponse)
	model := strings.TrimSpace(entry.Model)
	typeName := normalizeTaskType(entry.TaskType)

	if entry.CreatedAt.IsZero() && entry.TTL <= 0 {
		return true
	}
	if typeName == "unknown" && userMsg == "" && aiResp == "" {
		return true
	}
	if strings.Contains(model, ":") && userMsg == "" && aiResp == "" {
		return true
	}

	return false
}

func aggregateCacheEntries(entries []*cache.CacheEntryInfo) []*cache.CacheEntryInfo {
	grouped := make(map[string]*cache.CacheEntryInfo)
	for _, entry := range entries {
		if entry == nil {
			continue
		}

		sig := strings.Join([]string{
			normalizeTaskType(entry.TaskType),
			strings.TrimSpace(entry.UserMessage),
			strings.TrimSpace(entry.AIResponse),
			strings.TrimSpace(entry.Provider),
		}, "|")

		if existing, ok := grouped[sig]; ok {
			existing.GroupCount++
			existing.Hits += entry.Hits
			existing.HitRecorded = existing.HitRecorded || entry.HitRecorded
			existing.ModelStats = mergeModelStats(existing.ModelStats, entry.ModelStats, entry.Model)
			if existing.TaskTypeSource == "" {
				existing.TaskTypeSource = entry.TaskTypeSource
			} else if entry.TaskTypeSource != "" && existing.TaskTypeSource != entry.TaskTypeSource && existing.TaskTypeSource != "mixed" {
				existing.TaskTypeSource = "mixed"
			}
			if entry.CreatedAt.After(existing.CreatedAt) {
				existing.CreatedAt = entry.CreatedAt
				existing.TTL = entry.TTL
				existing.ExpiresAt = entry.ExpiresAt
			}
			existing.Model = selectPrimaryModel(existing.ModelStats)
			continue
		}

		copied := *entry
		copied.GroupCount = 1
		copied.ModelStats = mergeModelStats(nil, entry.ModelStats, entry.Model)
		copied.Model = selectPrimaryModel(copied.ModelStats)
		grouped[sig] = &copied
	}

	result := make([]*cache.CacheEntryInfo, 0, len(grouped))
	for _, entry := range grouped {
		result = append(result, entry)
	}

	return result
}

func mergeModelStats(base, incoming map[string]int, fallbackModel string) map[string]int {
	if base == nil {
		base = map[string]int{}
	}
	for model, count := range incoming {
		if model == "" || count <= 0 {
			continue
		}
		base[model] += count
	}
	if len(incoming) == 0 {
		m := strings.TrimSpace(fallbackModel)
		if m != "" && m != "-" {
			base[m]++
		}
	}
	return base
}

func selectPrimaryModel(stats map[string]int) string {
	if len(stats) == 0 {
		return "-"
	}
	bestModel := ""
	bestCount := -1
	for model, count := range stats {
		if count > bestCount {
			bestModel = model
			bestCount = count
		}
	}
	if len(stats) > 1 {
		return fmt.Sprintf("%s 等%d个", bestModel, len(stats))
	}
	return bestModel
}

// GET /api/admin/cache/model-mappings.
func (h *CacheHandler) GetModelMappings(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.modelMappingCache == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    []interface{}{},
			"stats":   nil,
		})
		return
	}

	mappings := h.modelMappingCache.GetAll()
	stats := h.modelMappingCache.Stats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    mappings,
		"stats":   stats,
	})
}

// DELETE /api/admin/cache/model-mappings.
func (h *CacheHandler) ClearModelMappings(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.modelMappingCache == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"cleared": 0,
		})
		return
	}

	count := h.modelMappingCache.Clear()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"cleared": count,
	})
}

// POST /api/admin/cache/model-mappings/cleanup.
func (h *CacheHandler) CleanupModelMappings(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.modelMappingCache == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"expired": 0,
		})
		return
	}

	expired := h.modelMappingCache.Cleanup()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"expired": expired,
	})
}
