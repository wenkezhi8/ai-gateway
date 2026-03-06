//nolint:godot,gocyclo,goconst,gocritic,revive,exhaustive,unused
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/handler/admin"
	"ai-gateway/internal/intent"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/middleware"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"
	"ai-gateway/internal/storage"
	"ai-gateway/internal/tracing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Max request body size (10MB) - protects against DoS attacks
const maxRequestBodySize = 10 * 1024 * 1024

// Default models for each provider type
var defaultProviderModels = map[string][]string{
	"openai": {
		"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4-turbo-preview",
		"gpt-4", "gpt-3.5-turbo", "gpt-3.5-turbo-16k",
		"o1", "o1-mini", "o1-preview", "o3-mini",
		"gpt-5", "gpt-5-mini", "gpt-5.3-codex", "gpt-5.3-codex-spark", "gpt-5.2-codex", "gpt-5.2 codex",
	},
	"anthropic": {
		"claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20240620",
		"claude-3-5-haiku-20241022", "claude-3-opus-20240229",
		"claude-3-sonnet-20240229", "claude-3-haiku-20240307",
	},
	"claude": {
		"claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20240620",
		"claude-3-5-haiku-20241022", "claude-3-opus-20240229",
		"claude-3-sonnet-20240229", "claude-3-haiku-20240307",
	},
	"deepseek": {
		"deepseek-chat", "deepseek-coder", "deepseek-reasoner",
	},
	"zhipu": {
		"glm-4-plus", "glm-4-0520", "glm-4-air", "glm-4-airx",
		"glm-4-long", "glm-4-flash",
	},
	"qwen": {
		"qwen-max", "qwen-max-longcontext", "qwen-plus",
		"qwen-turbo", "qwen-long",
	},
	"moonshot": {
		"kimi-k2.5", "moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k",
	},
	"volcengine": {
		"doubao-pro-32k", "doubao-pro-128k", "doubao-pro-256k",
		"doubao-lite-32k", "doubao-lite-128k",
	},
	"minimax": {
		"abab6.5s-chat", "abab6.5g-chat", "abab6.5t-chat",
		"abab5.5-chat", "abab5.5s-chat",
	},
	"baichuan": {
		"Baichuan4", "Baichuan3-Turbo", "Baichuan3-Turbo-128k",
	},
	"google": {
		"gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash",
	},
}

// GetSmartRouter returns the global smart router instance
func GetSmartRouter() *routing.SmartRouter {
	return routing.GetGlobalSmartRouter()
}

// ProxyHandler handles AI provider proxy requests
type ProxyHandler struct {
	config            *config.Config
	registry          *provider.Registry
	accountManager    *limiter.AccountManager
	smartRouter       *routing.SmartRouter
	cache             *cache.Manager
	deduplicator      *cache.RequestDeduplicator
	semanticCache     *cache.SemanticCache
	modelMappingCache *cache.ModelMappingCache
	vectorStore       cache.VectorCacheStore
	vectorPipeline    *VectorPipeline
	textNormalizer    *cache.TextNormalizer
	traceRecorder     *tracing.SpanRecorder
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(cfg *config.Config, accountManager *limiter.AccountManager, cacheManager *cache.Manager) *ProxyHandler {
	// Initialize semantic cache
	// 改动点: 如果配置了 Redis，使用 Redis 作为后端；否则纯内存
	var backendCache cache.Cache
	if cacheManager != nil {
		backendCache = cacheManager.Cache()
	}
	semanticCache := cache.NewSemanticCache(cache.DefaultSemanticCacheConfig(), backendCache)
	if cacheManager != nil {
		cacheManager.SetSemanticCache(semanticCache)
	}

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			semanticCache.Cleanup()
		}
	}()

	modelMappingCache := cache.NewModelMappingCache(cache.ModelMappingConfig{
		MaxSize: 1000,
		TTL:     24 * time.Hour,
	})

	var vectorStore cache.VectorCacheStore
	if cacheManager != nil {
		vectorStore = cacheManager.GetVectorStore()
	}

	settingsGetter := cache.DefaultCacheSettings
	if cacheManager != nil {
		settingsGetter = cacheManager.GetSettings
	}
	textNormalizer := cache.NewTextNormalizer()
	vectorPipeline := NewVectorPipeline(vectorStore, textNormalizer, settingsGetter)

	// Initialize trace recorder
	var traceRecorder *tracing.SpanRecorder
	if db := storage.GetSQLiteStorage().GetDB(); db != nil {
		traceRecorder = tracing.NewSpanRecorder(db)
	}

	return &ProxyHandler{
		config:            cfg,
		registry:          provider.GetRegistry(),
		accountManager:    accountManager,
		smartRouter:       GetSmartRouter(),
		cache:             cacheManager,
		deduplicator:      cache.GetRequestDeduplicator(),
		semanticCache:     semanticCache,
		modelMappingCache: modelMappingCache,
		vectorStore:       vectorStore,
		vectorPipeline:    vectorPipeline,
		textNormalizer:    textNormalizer,
		traceRecorder:     traceRecorder,
	}
}

// ChatCompletions proxies chat completion requests to AI providers
// 改动点: 集成请求去重、难度评估、缓存策略
func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
	startTime := time.Now()

	// 生成 Request ID 并设置到 context
	requestID := tracing.GenerateRequestID()
	ctx := tracing.SetRequestIDToContext(c.Request.Context(), requestID)
	c.Request = c.Request.WithContext(ctx)
	c.Set("request_id", requestID)
	c.Header("X-Request-ID", requestID)

	// 记录 Span 0: HTTP 请求入口
	if h.traceRecorder != nil {
		h.traceRecorder.RecordSimpleSpan(ctx, "http.entry", map[string]interface{}{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"request_id": requestID,
		})
	}

	// Limit request body size to prevent DoS attacks
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodySize)

	// 记录 Span 1: 请求解析
	parseStart := time.Now()
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if err.Error() == "http: request body too large" {
			errorMessage := "Request body exceeds maximum size of 10MB"
			h.recordHTTPResponseErrorSpan(ctx, startTime, "", "", "", errorMessage, http.StatusRequestEntityTooLarge)
			Error(c, http.StatusRequestEntityTooLarge, "request_too_large", errorMessage)
			return
		}
		errorMessage := "Invalid request body: " + err.Error()
		h.recordHTTPResponseErrorSpan(ctx, startTime, "", "", "", errorMessage, http.StatusBadRequest)
		BadRequest(c, errorMessage)
		return
	}
	if h.traceRecorder != nil {
		h.traceRecorder.RecordSimpleSpan(ctx, "handler.parse-request", map[string]interface{}{
			"duration_ms": time.Since(parseStart).Milliseconds(),
			"model":       req.Model,
			"stream":      req.Stream,
		})
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, req.Provider, "", err.Error(), http.StatusBadRequest)
		BadRequest(c, err.Error())
		return
	}
	req.Messages = sanitizeChatMessagesForIntentAndUpstream(req.Messages)
	explicitReasoningEffort := strings.TrimSpace(req.ReasoningEffort) != ""

	// Get user context
	userID := middleware.GetUserID(c)
	apiKey := resolveAPIKeyFromRequest(c)

	// Extract prompt from messages for smart routing and assessment
	prompt := ""
	contextStr := ""
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			prompt = getTextContent(msg.Content)
		} else if msg.Role == "system" {
			contextStr = getTextContent(msg.Content)
		}
	}

	// 记录 Span 2: 分类器评估
	assessStart := time.Now()
	assessment := h.smartRouter.AssessDifficulty(prompt, contextStr)
	if h.traceRecorder != nil {
		assessAttrs := map[string]interface{}{
			"duration_ms":     time.Since(assessStart).Milliseconds(),
			"task_type":       assessment.TaskType,
			"difficulty":      assessment.Difficulty,
			"recommended_ttl": assessment.SuggestedTTL,
			"user_id":         userID,
		}
		for key, value := range buildPromptPreviewAttributes(prompt) {
			assessAttrs[key] = value
		}
		h.traceRecorder.RecordSimpleSpan(ctx, "classifier.assess", assessAttrs)
	}
	recommendedTTL := assessment.SuggestedTTL
	classifierCfg := h.smartRouter.GetClassifierConfig()
	controlCfg := classifierCfg.Control
	normalizedQuery := ""
	experimentTag := ""
	domainTag := ""
	if assessment.ControlSignals != nil {
		normalizedQuery = strings.TrimSpace(assessment.ControlSignals.NormalizedQuery)
		experimentTag = strings.TrimSpace(assessment.ControlSignals.ExperimentTag)
		domainTag = strings.TrimSpace(assessment.ControlSignals.DomainTag)
	}
	semanticQuery := buildSemanticCacheWriteQuery(
		prompt,
		normalizedQuery,
		assessment.SemanticSignature,
		controlCfg.Enable && controlCfg.NormalizedQueryReadEnable,
	)
	if controlCfg.Enable && controlCfg.RiskTagEnable {
		logControlRiskSignals(assessment)
	}
	for k, v := range buildControlHeaders(controlCfg, assessment) {
		c.Header(k, v)
	}
	if shouldBlockByRisk(controlCfg, assessment) {
		errorMessage := "Request blocked by control risk policy"
		h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, req.Provider, prompt, errorMessage, http.StatusForbidden)
		Error(c, http.StatusForbidden, "risk_blocked", errorMessage)
		return
	}
	if controlCfg.Enable {
		logControlRoutingSignals(assessment)
	}
	applyControlToolGate(&req, controlCfg, assessment)
	applyControlGenerationHints(&req, controlCfg, assessment)

	if ttl, ok := cache.GetRuleStore().Match(string(assessment.TaskType), req.Model); ok {
		recommendedTTL = ttl
	}
	recommendedTTL = applyControlTTLBand(recommendedTTL, controlCfg, assessment.ControlSignals)
	cacheWriteAllowed := shouldAllowCacheWrite(controlCfg, assessment.ControlSignals)

	cacheSettings := cache.DefaultCacheSettings()
	if h.cache != nil {
		cacheSettings = h.cache.GetSettings()
	}
	cacheEnabled := h.cache != nil && cacheSettings.Enabled && recommendedTTL > 0
	allowSemantic := cacheEnabled &&
		cacheSettings.Strategy == cache.CacheStrategySemantic &&
		shouldAllowSemanticCache(assessment.TaskType)
	cacheKeyPayload := buildResponseCacheKeyPayload(&req, assessment.TaskType, prompt)
	cacheModelDimension := responseCacheModelDimension(assessment.TaskType, req.Model)

	logrus.WithFields(logrus.Fields{
		"task_type":       assessment.TaskType,
		"difficulty":      assessment.Difficulty,
		"recommended_ttl": recommendedTTL,
		"classifier":      assessment.Source,
		"fallback_reason": assessment.FallbackReason,
	}).Info("Request difficulty assessment")

	// Handle "auto", "latest", and "default" model selection
	originalRequestModel := strings.ToLower(strings.TrimSpace(req.Model))
	isVirtualModelRequest := originalRequestModel == "auto" || originalRequestModel == "latest" || originalRequestModel == "default"
	requestedModel := req.Model
	if isVirtualModelRequest {
		// Get available models from account manager
		var availableModels []string
		if h.accountManager != nil {
			for _, acc := range h.accountManager.GetAllAccounts() {
				if acc.Enabled {
					availableModels = append(availableModels, acc.Provider)
				}
			}
		}

		// Get provider if specified
		providerName := req.Provider
		logrus.WithFields(logrus.Fields{
			"model":    req.Model,
			"provider": providerName,
		}).Debug("Chat completion request")

		if providerName != "" {
			// Use provider-specific selection
			requestedModel = h.smartRouter.SelectModelForProvider(req.Model, providerName, prompt, availableModels)
		} else {
			// No provider specified - use global selection with assessment
			switch req.Model {
			case "latest":
				requestedModel = h.smartRouter.SelectModelWithStrategy("latest", routing.StrategyQuality, prompt, availableModels)
			case "default":
				config := h.smartRouter.GetConfig()
				if config.DefaultModel != "" {
					requestedModel = config.DefaultModel
				} else {
					requestedModel = h.smartRouter.SelectModelWithStrategy("default", routing.StrategyAuto, prompt, availableModels)
				}
			default:
				// Use assessment-based model selection
				requestedModel, _ = h.smartRouter.SelectModelWithAssessment("auto", prompt, contextStr, availableModels)
			}
		}

		req.Model = requestedModel

		// Also set provider if not specified
		if req.Provider == "" {
			req.Provider = h.smartRouter.GetProviderForModel(requestedModel)
		}
	}

	requestedModel = h.resolveCanonicalModelID(requestedModel)
	req.Model = requestedModel

	requestedProvider := normalizeProviderName(req.Provider)
	req.Provider = requestedProvider

	if !isVirtualModelRequest {
		resolvedProvider, ok := h.resolveProviderFromModelRegistry(requestedModel)
		if !ok {
			errorMessage := "模型未在模型管理中注册，请先在 /model-management 绑定服务商"
			h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, requestedProvider, prompt, errorMessage, http.StatusBadRequest)
			Error(c, http.StatusBadRequest, ErrCodeModelNotReg, errorMessage)
			return
		}
		if requestedProvider != "" && requestedProvider != resolvedProvider {
			logrus.WithFields(logrus.Fields{
				"requested_provider": requestedProvider,
				"resolved_provider":  resolvedProvider,
				"model":              requestedModel,
			}).Warn("Request provider conflicts with model registry mapping; using model registry provider")
		}
		req.Provider = resolvedProvider
	} else if req.Provider == "" {
		req.Provider = inferProviderFromModel(requestedModel)
	}

	usageMeta := buildUsageRuntimeMeta(c, &req, nil, h.accountManager)

	semanticCandidates := buildSemanticQueryCandidates(controlCfg.Enable && controlCfg.NormalizedQueryReadEnable, normalizedQuery, semanticQuery, prompt)

	// V2 cache read path: Ollama dual-model pipeline + Redis Stack exact/vector retrieval.
	cacheV2Start := time.Now()
	intentResult, v2CachedBody, v2CacheHit, v2HitLayer, v2HitKey := h.processCacheV2Read(
		c.Request.Context(),
		prompt,
		normalizedQuery,
		string(assessment.TaskType),
		cacheSettings,
	)
	if h.traceRecorder != nil {
		result := "miss"
		if v2CacheHit {
			result = "hit"
		}
		preview, full, truncated := tracing.ExtractResponseTextPreview(v2CachedBody, 200, 4000)
		h.traceRecorder.RecordSpanWithResult(ctx, "cache.read-v2", result, map[string]interface{}{
			"duration_ms":      time.Since(cacheV2Start).Milliseconds(),
			"hit":              v2CacheHit,
			"layer":            v2HitLayer,
			"task_type":        string(assessment.TaskType),
			"answer_preview":   preview,
			"answer_full":      full,
			"answer_truncated": truncated,
		})
	}
	if v2CacheHit && len(v2CachedBody) > 0 {
		tokenUsage := extractUsageTokensFromBody(v2CachedBody)
		resolvedUsage, usageSource := resolveUsageWithFallback(prompt, extractAssistantTextFromBody(v2CachedBody), tokenUsage)
		h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, req.Provider, time.Since(startTime), resolvedUsage.Total, true, true, 0, resolvedUsage.Prompt, resolvedUsage.CachedRead, usageSource, string(assessment.TaskType), string(assessment.Difficulty), "", experimentTag, domainTag)
		admin.RecordRequestResult(req.Model, req.Provider, assessment.TaskType, assessment.Difficulty, true, time.Since(startTime).Milliseconds(), resolvedUsage.Total)
		if h.traceRecorder != nil {
			attrs := map[string]interface{}{
				"duration_ms": time.Since(startTime).Milliseconds(),
				"model":       req.Model,
				"provider":    req.Provider,
				"cache_hit":   true,
				"cache_layer": v2HitLayer,
				"status_code": http.StatusOK,
			}
			for key, value := range buildTraceMessageAttributes(prompt, v2CachedBody) {
				attrs[key] = value
			}
			h.traceRecorder.RecordSimpleSpan(ctx, "http.response", attrs)
		}
		c.Header("X-Local-Cache-Hit", "1")
		c.Header("X-Cache-Layer", v2HitLayer)
		if req.Stream {
			h.writeCachedResponseAsStream(c, req.Model, v2CachedBody)
		} else {
			c.Data(http.StatusOK, "application/json", v2CachedBody)
		}
		if v2HitKey != "" && h.vectorStore != nil {
			if err := h.vectorStore.TouchTTL(c.Request.Context(), v2HitKey, h.intentTTLSeconds(intentResult)); err != nil {
				logrus.WithError(err).WithField("cache_key", v2HitKey).Debug("failed to refresh vector cache ttl")
			}
		}
		return
	}

	// Try semantic cache first (supports both non-streaming and streaming requests)
	if allowSemantic && h.semanticCache != nil {
		similarityThreshold := semanticThresholdForDifficulty(cacheSettings.SimilarityThreshold, assessment.Difficulty)

		for _, candidateQuery := range semanticCandidates {
			queryVector := cache.SimpleEmbedding(candidateQuery, 1536)
			cacheSemStart := time.Now()
			semanticEntry, similarity := h.semanticCache.Get(c.Request.Context(), candidateQuery, queryVector)
			if h.traceRecorder != nil {
				result := "miss"
				if semanticEntry != nil {
					result = "hit"
				}
				preview, full, truncated := tracing.ExtractResponseTextPreview(func() []byte {
					if semanticEntry == nil {
						return nil
					}
					return semanticEntry.Response
				}(), 200, 4000)
				h.traceRecorder.RecordSpanWithResult(ctx, "cache.read-semantic", result, map[string]interface{}{
					"duration_ms":      time.Since(cacheSemStart).Milliseconds(),
					"hit":              semanticEntry != nil,
					"similarity":       similarity,
					"threshold":        similarityThreshold,
					"answer_preview":   preview,
					"answer_full":      full,
					"answer_truncated": truncated,
				})
			}
			if semanticEntry == nil || similarity < similarityThreshold {
				continue
			}

			logrus.WithFields(logrus.Fields{
				"model":              req.Model,
				"similarity":         similarity,
				"threshold":          similarityThreshold,
				"task_type":          assessment.TaskType,
				"cache_id":           semanticEntry.ID,
				"semantic_candidate": candidateQuery,
			}).Info("Semantic cache hit")

			h.semanticCache.IncrementHitCount(semanticEntry.ID)
			tokenUsage := extractUsageTokensFromBody(semanticEntry.Response)
			resolvedUsage, usageSource := resolveUsageWithFallback(prompt, extractAssistantTextFromBody(semanticEntry.Response), tokenUsage)
			h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, req.Provider, time.Since(startTime), resolvedUsage.Total, true, true, 0, resolvedUsage.Prompt, resolvedUsage.CachedRead, usageSource, string(assessment.TaskType), string(assessment.Difficulty), "", experimentTag, domainTag)
			admin.RecordRequestResult(req.Model, req.Provider, assessment.TaskType, assessment.Difficulty, true, time.Since(startTime).Milliseconds(), resolvedUsage.Total)
			if h.traceRecorder != nil {
				attrs := map[string]interface{}{
					"duration_ms": time.Since(startTime).Milliseconds(),
					"model":       req.Model,
					"provider":    req.Provider,
					"cache_hit":   true,
					"cache_layer": "semantic",
					"status_code": http.StatusOK,
				}
				for key, value := range buildTraceMessageAttributes(prompt, semanticEntry.Response) {
					attrs[key] = value
				}
				h.traceRecorder.RecordSimpleSpan(ctx, "http.response", attrs)
			}
			c.Header("X-Local-Cache-Hit", "1")
			c.Header("X-Cache-Layer", "semantic")
			if req.Stream {
				h.writeCachedResponseAsStream(c, req.Model, semanticEntry.Response)
			} else {
				c.Data(http.StatusOK, "application/json", semanticEntry.Response)
			}
			return
		}
	}

	// Try exact cache
	if cacheEnabled && h.cache.ResponseCache != nil {
		cacheKey, keyErr := h.cache.ResponseCache.GenerateKey(req.Provider, cacheModelDimension, cacheKeyPayload)
		if keyErr != nil {
			logrus.WithError(keyErr).Warn("failed to generate exact cache key")
		} else {
			cacheExactStart := time.Now()
			cached, err := h.cache.ResponseCache.Get(c.Request.Context(), cacheKey)
			if h.traceRecorder != nil {
				result := "miss"
				if err == nil && cached != nil {
					result = "hit"
				}
				preview, full, truncated := tracing.ExtractResponseTextPreview(func() []byte {
					if cached == nil {
						return nil
					}
					return cached.Body
				}(), 200, 4000)
				h.traceRecorder.RecordSpanWithResult(ctx, "cache.read-exact", result, map[string]interface{}{
					"duration_ms":      time.Since(cacheExactStart).Milliseconds(),
					"hit":              err == nil && cached != nil,
					"cache_key":        cacheKey,
					"answer_preview":   preview,
					"answer_full":      full,
					"answer_truncated": truncated,
				})
			}
			if err == nil && cached != nil {
				if !hasMeaningfulCachedResponse(cached.Body) {
					logrus.WithFields(logrus.Fields{
						"model":     req.Model,
						"task_type": assessment.TaskType,
						"cache_key": cacheKey,
					}).Warn("Skip invalid cached response without meaningful content")
					if delErr := h.cache.Cache().Delete(c.Request.Context(), cacheKey); delErr != nil {
						logrus.WithError(delErr).WithField("cache_key", cacheKey).Debug("failed to delete invalid cache entry")
					}
				} else {
					logrus.WithFields(logrus.Fields{
						"model":     req.Model,
						"task_type": assessment.TaskType,
					}).Info("Response cache hit")

					tokenUsage := extractUsageTokensFromBody(cached.Body)
					resolvedUsage, usageSource := resolveUsageWithFallback(prompt, extractAssistantTextFromBody(cached.Body), tokenUsage)
					h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, req.Provider, time.Since(startTime), resolvedUsage.Total, true, true, 0, resolvedUsage.Prompt, resolvedUsage.CachedRead, usageSource, string(assessment.TaskType), string(assessment.Difficulty), "", experimentTag, domainTag)
					admin.RecordRequestResult(req.Model, req.Provider, assessment.TaskType, assessment.Difficulty, true, time.Since(startTime).Milliseconds(), resolvedUsage.Total)
					h.persistResponseCacheHit(c.Request.Context(), cacheKey, cached, req.Model)
					if h.traceRecorder != nil {
						attrs := map[string]interface{}{
							"duration_ms": time.Since(startTime).Milliseconds(),
							"model":       req.Model,
							"provider":    req.Provider,
							"cache_hit":   true,
							"cache_layer": "exact",
							"status_code": cached.StatusCode,
						}
						for key, value := range buildTraceMessageAttributes(prompt, cached.Body) {
							attrs[key] = value
						}
						h.traceRecorder.RecordSimpleSpan(ctx, "http.response", attrs)
					}
					c.Header("X-Local-Cache-Hit", "1")
					c.Header("X-Cache-Layer", "exact")
					if req.Stream {
						h.writeCachedResponseAsStream(c, req.Model, cached.Body)
					} else {
						c.Data(cached.StatusCode, "application/json", cached.Body)
					}
					return
				}
			}
		}
	}

	providerType := req.Provider
	if providerType == "" {
		providerType = inferProviderFromModel(requestedModel)
	}

	providerStart := time.Now()
	targetProvider, providerSelectErr := h.getProviderForRequest(c, requestedModel, providerType)
	if h.traceRecorder != nil {
		providerProtocol := providerType
		if targetProvider != nil {
			providerProtocol = targetProvider.Name()
		}
		h.traceRecorder.RecordSimpleSpan(ctx, "provider.select", map[string]interface{}{
			"duration_ms":   time.Since(providerStart).Milliseconds(),
			"model":         requestedModel,
			"provider":      providerType,
			"provider_type": providerProtocol,
			"provider_name": providerType,
			"user_id":       userID,
		})
	}
	usageMeta = buildUsageRuntimeMeta(c, &req, nil, h.accountManager)
	if providerSelectErr != nil {
		h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerType, time.Since(startTime), 0, false, false, 0, 0, 0, "actual", string(assessment.TaskType), string(assessment.Difficulty), "", experimentTag, domainTag)
		admin.RecordRequestResult(req.Model, req.Provider, assessment.TaskType, assessment.Difficulty, false, time.Since(startTime).Milliseconds(), 0)
		h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerType, prompt, providerSelectErr.Error(), http.StatusServiceUnavailable)
		Error(c, http.StatusServiceUnavailable, ErrCodeProviderError, providerSelectErr.Error())
		return
	}

	// Keep compatibility with existing scheduler release hook if set by other paths.
	if releaseFuncValue, ok := c.Get("scheduler_release_func"); ok {
		if releaseFunc, ok := releaseFuncValue.(func()); ok && releaseFunc != nil {
			defer releaseFunc()
		}
	}

	// Get default temperature based on model
	defaultTemp := getDefaultTemperature(req.Model)

	// Convert to provider request format
	providerReq := &provider.ChatRequest{
		Model:       req.Model,
		Temperature: getFloat64(req.Temperature, defaultTemp),
		MaxTokens:   getInt(req.MaxTokens, 0),
		Stream:      req.Stream,
		Extra:       buildProviderExtraFromChatRequest(&req),
	}

	// 改动点: 透传深度思考开关到 provider 层（用于支持推理输出的模型）
	if req.DeepThink {
		if providerReq.Extra == nil {
			providerReq.Extra = map[string]interface{}{}
		}
		providerReq.Extra["deep_think"] = true
		providerReq.Extra["reasoning"] = true
	}

	// Convert messages
	for _, msg := range req.Messages {
		pm := provider.ChatMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}
		if len(msg.ToolCalls) > 0 {
			pm.ToolCalls = make([]provider.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				pm.ToolCalls[i] = provider.ToolCall{
					Index: tc.Index,
					ID:    tc.ID,
					Type:  tc.Type,
					Function: provider.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		providerReq.Messages = append(providerReq.Messages, pm)
	}

	// Convert tools
	if len(req.Tools) > 0 {
		providerReq.Tools = make([]provider.Tool, len(req.Tools))
		for i, t := range req.Tools {
			providerReq.Tools[i] = provider.Tool{
				Type: t.Type,
				Function: provider.Function{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}
	providerReq.ToolChoice = req.ToolChoice

	originalModelID := h.resolveCanonicalModelID(providerReq.Model)
	effectiveModel := originalModelID

	// 1. 先检查缓存
	if h.modelMappingCache != nil {
		if cached, ok := h.modelMappingCache.GetEffectiveModel(req.Provider, originalModelID); ok {
			cachedModel := h.resolveCanonicalModelID(cached)
			if cachedModel == originalModelID {
				effectiveModel = cachedModel
				logrus.WithFields(logrus.Fields{
					"provider":        req.Provider,
					"original_model":  originalModelID,
					"effective_model": effectiveModel,
					"source":          "cache",
				}).Debug("Using cached model name mapping")
			} else {
				logrus.WithFields(logrus.Fields{
					"provider":               req.Provider,
					"original_model":         originalModelID,
					"cached_effective_model": cachedModel,
				}).Warn("Ignoring unsafe cached model mapping")
			}
		}
	}

	providerReq.Model = effectiveModel
	req.Model = effectiveModel

	// Handle streaming request
	if req.Stream {
		h.handleStreamResponse(c, targetProvider, providerReq, explicitReasoningEffort, usageMeta, userID, apiKey, prompt, semanticQuery, allowSemantic, cacheEnabled, cacheWriteAllowed, recommendedTTL, string(assessment.TaskType), string(assessment.Source), assessment.Difficulty, req.Provider, cacheModelDimension, cacheKeyPayload, originalModelID, experimentTag, domainTag)
		return
	}

	// Use deduplicator for non-streaming requests
	// 改动点: 使用请求去重避免重复计算
	dedupKey := h.deduplicator.GenerateKey(prompt, req.Model, map[string]interface{}{
		"temperature": providerReq.Temperature,
		"max_tokens":  providerReq.MaxTokens,
	})

	// 记录 Span 7: 上游调用
	upstreamStart := time.Now()
	result, err := h.deduplicator.Do(c.Request.Context(), dedupKey, func() (interface{}, error) {
		providerResult, providerErr := targetProvider.Chat(c.Request.Context(), providerReq)
		return providerResult, providerErr
	})
	reasoningEffortDowngraded := false
	if h.traceRecorder != nil {
		providerProtocol := targetProvider.Name()
		h.traceRecorder.RecordSimpleSpan(ctx, "provider.chat", map[string]interface{}{
			"duration_ms":   time.Since(upstreamStart).Milliseconds(),
			"model":         providerReq.Model,
			"provider":      providerType,
			"provider_type": providerProtocol,
			"provider_name": providerType,
			"stream":        providerReq.Stream,
			"success":       err == nil,
		})
	}

	if err != nil {
		if explicitReasoningEffort && isReasoningEffortUnsupportedError(err) {
			if downgradedReq, ok := cloneProviderRequestWithoutReasoningEffort(providerReq); ok {
				retryResult, retryErr := h.deduplicator.Do(c.Request.Context(), dedupKey+":reasoning-downgraded", func() (interface{}, error) {
					providerResult, providerErr := targetProvider.Chat(c.Request.Context(), downgradedReq)
					return providerResult, providerErr
				})
				if retryErr == nil {
					providerReq = downgradedReq
					result = retryResult
					err = nil
					reasoningEffortDowngraded = true
				} else {
					err = retryErr
				}
			}
		}
	}

	if err != nil {
		// CHANGED: include provider/user/api info in usage logs for error paths.
		providerName := req.Provider
		if providerName == "" && targetProvider != nil {
			providerName = targetProvider.Name()
		}
		h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, time.Since(startTime), 0, false, false, 0, 0, 0, "actual", string(assessment.TaskType), string(assessment.Difficulty), "", experimentTag, domainTag)
		admin.RecordRequestResult(req.Model, req.Provider, assessment.TaskType, assessment.Difficulty, false, time.Since(startTime).Milliseconds(), 0)
		logMsg := fmt.Sprintf("Provider request failed: %v", err)
		if providerErr, ok := err.(*provider.ProviderError); ok {
			logMsg = fmt.Sprintf("Provider request failed [%s]: %s (code: %d, retryable: %v)",
				providerErr.Provider, providerErr.Message, providerErr.Code, providerErr.Retryable)
		}
		logrus.WithField("provider_error", logMsg).Warn("Upstream provider request failed")
		if providerErr, ok := err.(*provider.ProviderError); ok {
			statusCode := providerErr.Code
			if statusCode < http.StatusBadRequest || statusCode >= 600 {
				statusCode = http.StatusBadGateway
			}
			h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerName, prompt, providerErr.Message, statusCode)
			Error(c, statusCode, ErrCodeProviderError, providerErr.Message)
			return
		}
		h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerName, prompt, err.Error(), http.StatusBadGateway)
		ProviderError(c, err.Error(), "")
		return
	}
	resolvedProviderName := req.Provider
	if resolvedProviderName == "" && targetProvider != nil {
		resolvedProviderName = targetProvider.Name()
	}

	resp, ok := result.(*provider.ChatResponse)
	if !ok {
		h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, resolvedProviderName, prompt, "Provider returned invalid response type", http.StatusBadGateway)
		ProviderError(c, "Provider returned invalid response type", "invalid_provider_response")
		return
	}

	// Check for provider error in response
	if resp.Error != nil {
		if explicitReasoningEffort && isReasoningEffortUnsupportedError(resp.Error) {
			if downgradedReq, ok := cloneProviderRequestWithoutReasoningEffort(providerReq); ok {
				retryResp, retryErr := targetProvider.Chat(c.Request.Context(), downgradedReq)
				if retryErr == nil && retryResp != nil && retryResp.Error == nil {
					providerReq = downgradedReq
					resp = retryResp
					reasoningEffortDowngraded = true
				} else if retryErr != nil {
					if providerErr, ok := retryErr.(*provider.ProviderError); ok {
						resp.Error = providerErr
					}
				}
			}
		}
	}

	if resp.Error != nil {
		// CHANGED: include provider/user/api info in usage logs for provider error responses.
		providerName := req.Provider
		if providerName == "" && targetProvider != nil {
			providerName = targetProvider.Name()
		}
		h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, time.Since(startTime), 0, false, false, 0, 0, 0, "actual", string(assessment.TaskType), string(assessment.Difficulty), "", experimentTag, domainTag)
		admin.RecordRequestResult(req.Model, req.Provider, assessment.TaskType, assessment.Difficulty, false, time.Since(startTime).Milliseconds(), 0)
		h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerName, prompt, resp.Error.Message, http.StatusBadGateway)
		ProviderError(c, resp.Error.Message, resp.Error.Type)
		return
	}

	// Update model success rate
	h.smartRouter.UpdateModelSuccessRate(req.Model, assessment.TaskType, true)

	resolvedUsage, usageSource := resolveUsageWithFallback(
		prompt,
		extractAssistantTextFromProviderChoices(resp.Choices),
		usageTokens{
			Prompt:     resp.Usage.PromptTokens,
			Completion: resp.Usage.CompletionTokens,
			Total:      resp.Usage.TotalTokens,
			CachedRead: resp.Usage.CachedReadTokens,
		},
	)

	// Record metrics
	latency := time.Since(startTime)
	// CHANGED: include provider/user/api info and prompt tokens in usage logs.
	providerName := req.Provider
	if providerName == "" && targetProvider != nil {
		providerName = targetProvider.Name()
	}
	h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, latency, resolvedUsage.Total, true, false, 0, resolvedUsage.Prompt, resolvedUsage.CachedRead, usageSource, string(assessment.TaskType), string(assessment.Difficulty), "", experimentTag, domainTag)
	admin.RecordRequestResult(req.Model, req.Provider, assessment.TaskType, assessment.Difficulty, true, latency.Milliseconds(), resolvedUsage.Total)

	// Record successful model name mapping if different from original
	if h.modelMappingCache != nil && originalModelID != "" && originalModelID != providerReq.Model {
		h.modelMappingCache.RecordSuccess(req.Provider, originalModelID, providerReq.Model)
	}

	// Build response
	response := ChatCompletionResponse{
		ID:                resp.ID,
		Object:            resp.Object,
		Created:           resp.Created,
		Model:             resp.Model,
		SystemFingerprint: "",
		Choices:           convertChoices(resp.Choices),
		Usage: Usage{
			PromptTokens:     resolvedUsage.Prompt,
			CompletionTokens: resolvedUsage.Completion,
			TotalTokens:      resolvedUsage.Total,
			CachedReadTokens: resolvedUsage.CachedRead,
		},
	}
	if reasoningEffortDowngraded {
		response.GatewayMeta = &GatewayMeta{ReasoningEffortDowngraded: true}
	}

	// Cache the response if applicable
	logrus.WithFields(logrus.Fields{
		"cache_enabled":       cacheEnabled,
		"cache_write_allowed": cacheWriteAllowed,
		"response_cache_nil":  h.cache == nil || h.cache.ResponseCache == nil,
		"recommended_ttl":     recommendedTTL,
	}).Info("Cache check")

	if cacheEnabled && cacheWriteAllowed && h.cache.ResponseCache != nil && hasMeaningfulAssistantResponse(&response) {
		responseBody, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			logrus.WithError(marshalErr).Warn("failed to marshal chat completion response for cache")
		} else {
			cacheKey, keyErr := h.cache.ResponseCache.GenerateKey(req.Provider, cacheModelDimension, cacheKeyPayload)
			if keyErr != nil {
				logrus.WithError(keyErr).Warn("failed to generate response cache key")
			} else {
				cachedResp := &cache.CachedResponse{
					StatusCode:     http.StatusOK,
					Headers:        map[string]string{"Content-Type": "application/json"},
					Body:           responseBody,
					CreatedAt:      time.Now(),
					HitCount:       0,
					HitModels:      map[string]int64{req.Model: 1},
					Provider:       req.Provider,
					Model:          req.Model,
					Prompt:         prompt,
					TaskType:       string(assessment.TaskType),
					TaskTypeSource: string(assessment.Source),
				}

				// Use SetWithTaskType to record task type for filtering
				if err := h.writeResponseCacheEntry(c.Request.Context(), cacheKey, cachedResp, recommendedTTL, req.Model, req.Provider, string(assessment.TaskType), string(assessment.Source)); err != nil {
					logrus.WithError(err).WithField("cache_key", cacheKey).Warn("failed to write response cache entry")
				}

				if shouldUsePromptOnlyCache(assessment.TaskType) {
					h.pruneDuplicateResponseEntries(c.Request.Context(), req.Provider, req.Model, string(assessment.TaskType), prompt, cacheKey)
				}

				logrus.WithFields(logrus.Fields{
					"model":     req.Model,
					"ttl":       recommendedTTL,
					"task_type": assessment.TaskType,
					"cache_key": cacheKey,
				}).Info("Response cached")
			}
		}
	} else if !cacheWriteAllowed {
		logrus.WithFields(logrus.Fields{
			"model":     req.Model,
			"task_type": assessment.TaskType,
			"cache_reason": func() string {
				if assessment.ControlSignals == nil {
					return ""
				}
				return assessment.ControlSignals.CacheReason
			}(),
		}).Info("Response cache write skipped by control signal")
	} else if recommendedTTL == 0 {
		logrus.WithFields(logrus.Fields{
			"model":     req.Model,
			"task_type": assessment.TaskType,
		}).Debug("Response not cached (TTL=0)")
	} else {
		logrus.WithFields(logrus.Fields{
			"model": req.Model,
		}).Warn("Cache not available")
	}

	// Store in semantic cache for similar query matching
	// 改动点: 存储到语义缓存供相似请求复用
	if allowSemantic && cacheWriteAllowed && h.semanticCache != nil && assessment.TaskType != routing.TaskTypeCreative && hasMeaningfulAssistantResponse(&response) {
		responseBody, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			logrus.WithError(marshalErr).Warn("failed to marshal semantic cache response")
		} else {
			queryVector := cache.SimpleEmbedding(semanticQuery, 1536)
			h.semanticCache.Set(
				c.Request.Context(),
				semanticQuery,
				queryVector,
				responseBody,
				req.Model,
				req.Provider,
				string(assessment.TaskType),
				recommendedTTL,
			)
			logrus.WithFields(logrus.Fields{
				"model":     req.Model,
				"task_type": assessment.TaskType,
			}).Debug("Semantic cache entry stored")
		}
	}

	// 记录 Span 8: 缓存写入
	if h.traceRecorder != nil {
		resolvedProviderProtocol := ""
		if targetProvider != nil {
			resolvedProviderProtocol = targetProvider.Name()
		}
		h.traceRecorder.RecordSimpleSpan(ctx, "cache.write", map[string]interface{}{
			"model":         req.Model,
			"provider":      providerType,
			"provider_type": resolvedProviderProtocol,
			"provider_name": providerType,
			"task_type":     string(assessment.TaskType),
			"ttl":           recommendedTTL,
		})
	}

	// V2 cache async write path.
	h.processCacheV2Write(c.Request.Context(), prompt, normalizedQuery, intentResult, req.Provider, req.Model, assessment.TaskType, response)

	// 记录 Span 9: 响应返回
	if h.traceRecorder != nil {
		resolvedProviderProtocol := ""
		if targetProvider != nil {
			resolvedProviderProtocol = targetProvider.Name()
		}
		responseBody, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			responseBody = nil
		}
		attrs := map[string]interface{}{
			"duration_ms":   time.Since(startTime).Milliseconds(),
			"model":         req.Model,
			"provider":      providerType,
			"provider_type": resolvedProviderProtocol,
			"provider_name": providerType,
			"success":       true,
		}
		for key, value := range buildTraceMessageAttributes(prompt, responseBody) {
			attrs[key] = value
		}
		h.traceRecorder.RecordSimpleSpan(ctx, "http.response", attrs)
	}

	// Return response in OpenAI-compatible format
	c.Header("X-Local-Cache-Hit", "0")
	c.Header("X-Cache-Layer", "none")
	c.JSON(http.StatusOK, response)
}

// getProviderForRequest gets the appropriate provider for the request
// First resolves accounts with strict provider matching, then falls back to provider_type compatibility.
// Falls back to registry providers only when account selection is unavailable.
func (h *ProxyHandler) getProviderForRequest(c *gin.Context, model string, providerName string) (provider.Provider, error) {
	providerName = normalizeProviderName(providerName)
	if providerName == "" {
		return nil, &provider.ProviderError{
			Message:   "No available provider for model: " + model,
			Code:      http.StatusServiceUnavailable,
			Retryable: false,
		}
	}
	backendProvider := normalizeProviderName(mapProviderName(providerName))
	var lastCreateErr error

	// Try route cache first
	if h.cache != nil && h.cache.RouteCache != nil && providerName != "" {
		cacheKey := model + ":" + providerName
		cached, err := h.cache.RouteCache.Get(context.Background(), cacheKey, nil)
		if err == nil && cached != nil {
			logrus.WithFields(logrus.Fields{
				"model":    model,
				"provider": providerName,
			}).Debug("Route cache hit")
		}
		_ = cached // We still need to create provider from account, but cache records the hit
	}

	// Tier 1 / Tier 2 account selection
	if h.accountManager != nil && providerName != "" {
		tierOneAccounts, tierOneExists, tierTwoAccounts := h.collectProviderAccounts(providerName, backendProvider)
		if tierOneExists {
			if len(tierOneAccounts) == 0 {
				return nil, &provider.ProviderError{
					Message:   "Provider '" + providerName + "' is disabled. Please enable it in the provider settings.",
					Code:      http.StatusServiceUnavailable,
					Retryable: false,
				}
			}
			selected := selectAccountByPriority(tierOneAccounts)
			if selected != nil {
				p, createErr := h.createProviderFromAccount(c, selected, providerName, model, "")
				if createErr == nil {
					return p, nil
				}
				lastCreateErr = createErr
			}
		} else if len(tierTwoAccounts) > 0 {
			selected := selectAccountByPriority(tierTwoAccounts)
			if selected != nil {
				p, createErr := h.createProviderFromAccount(c, selected, providerName, model, "provider_type_compatible")
				if createErr == nil {
					return p, nil
				}
				lastCreateErr = createErr
			}
		}
	}

	// Registry fallback keeps compatibility for deployments that still rely on static provider registration.
	candidates := []string{providerName, backendProvider}
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = normalizeProviderName(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		if p, ok := h.registry.Get(candidate); ok && p.IsEnabled() {
			return p, nil
		}
	}

	targetProvider, ok := h.registry.GetByModel(model)
	if ok {
		return targetProvider, nil
	}

	if lastCreateErr != nil {
		return nil, &provider.ProviderError{
			Message:   lastCreateErr.Error(),
			Code:      http.StatusBadGateway,
			Retryable: false,
		}
	}
	return nil, &provider.ProviderError{
		Message:   "No available provider for model: " + model,
		Code:      http.StatusServiceUnavailable,
		Retryable: false,
	}
}

func (h *ProxyHandler) collectProviderAccounts(providerName, backendProvider string) (tierOneEnabled []*limiter.AccountConfig, tierOneExists bool, tierTwoEnabled []*limiter.AccountConfig) {
	if h.accountManager == nil {
		return nil, false, nil
	}

	for _, acc := range h.accountManager.GetAllAccounts() {
		if acc == nil {
			continue
		}
		accountProvider := normalizeProviderName(acc.Provider)
		accountProviderType := normalizeProviderName(acc.ProviderType)
		if accountProviderType == "" {
			accountProviderType = accountProvider
		}

		if accountProvider == providerName {
			tierOneExists = true
			if acc.Enabled {
				tierOneEnabled = append(tierOneEnabled, acc)
			}
			continue
		}

		if accountProviderType == backendProvider && acc.Enabled {
			tierTwoEnabled = append(tierTwoEnabled, acc)
		}
	}

	sortAccountCandidates(tierOneEnabled)
	sortAccountCandidates(tierTwoEnabled)

	return tierOneEnabled, tierOneExists, tierTwoEnabled
}

func sortAccountCandidates(accounts []*limiter.AccountConfig) {
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].Priority == accounts[j].Priority {
			return accounts[i].ID < accounts[j].ID
		}
		return accounts[i].Priority > accounts[j].Priority
	})
}

func selectAccountByPriority(accounts []*limiter.AccountConfig) *limiter.AccountConfig {
	if len(accounts) == 0 {
		return nil
	}
	return accounts[0]
}

func (h *ProxyHandler) createProviderFromAccount(c *gin.Context, account *limiter.AccountConfig, providerName, model, fallbackType string) (provider.Provider, error) {
	if account == nil {
		return nil, fmt.Errorf("account is nil")
	}

	backendProvider := normalizeProviderName(account.ProviderType)
	if backendProvider == "" {
		backendProvider = normalizeProviderName(mapProviderName(account.Provider))
	}
	if backendProvider == "" {
		backendProvider = normalizeProviderName(mapProviderName(providerName))
	}

	models := getModelsForProvider(providerName)
	if len(models) == 0 {
		models = getModelsForProvider(account.Provider)
	}

	provConfig := &provider.ProviderConfig{
		Name:    backendProvider,
		APIKey:  account.APIKey,
		BaseURL: account.BaseURL,
		Models:  models,
		Enabled: true,
	}

	p, err := h.registry.CreateProvider(provConfig)
	if err != nil {
		return nil, err
	}

	if c != nil {
		c.Set("selected_account_id", account.ID)
		c.Set("selected_account_name", strings.TrimSpace(account.Name))
		c.Set("scheduler_account_id", account.ID)
		c.Header("X-Account-ID", account.ID)
		if fallbackType != "" {
			c.Set("fallback_account_type", fallbackType)
			c.Header("X-Account-Fallback", fallbackType)
		}
	}

	if h.cache != nil && h.cache.RouteCache != nil && providerName != "" {
		cacheKey := model + ":" + providerName
		if setErr := h.cache.RouteCache.Set(context.Background(), cacheKey, nil, &cache.RouteDecision{
			Provider: providerName,
			Model:    model,
		}); setErr != nil {
			logrus.WithError(setErr).WithField("cache_key", cacheKey).Debug("failed to write route cache")
		}
	}

	return p, nil
}

// getBaseURLForProvider returns the base URL for a provider
func getBaseURLForProvider(providerName string) string {
	urls := map[string]string{
		"openai":     "https://api.openai.com/v1",
		"anthropic":  "https://api.anthropic.com/v1",
		"deepseek":   "https://api.deepseek.com",
		"moonshot":   "https://api.moonshot.cn/v1",
		"kimi":       "https://api.moonshot.cn/v1",
		"qwen":       "https://dashscope.aliyuncs.com/compatible-mode/v1",
		"zhipu":      "https://open.bigmodel.cn/api/paas/v4",
		"baichuan":   "https://api.baichuan-ai.com/v1",
		"minimax":    "https://api.minimax.chat/v1",
		"volcengine": "https://ark.cn-beijing.volces.com/api/v3",
		"yi":         "https://api.lingyiwanwu.com/v1",
		"google":     "https://generativelanguage.googleapis.com/v1beta",
		"mistral":    "https://api.mistral.ai/v1",
		"ernie":      "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat",
		"hunyuan":    "https://api.hunyuan.cloud.tencent.com/v1",
		"spark":      "https://spark-api-open.xf-yun.com/v1",
	}

	if url, ok := urls[providerName]; ok {
		return url
	}
	return ""
}

// mapProviderName maps frontend provider names to backend provider types
// Returns "openai" for OpenAI-compatible APIs, otherwise returns the original name
func mapProviderName(frontendProvider string) string {
	frontendProvider = normalizeProviderName(frontendProvider)

	// Providers that use OpenAI-compatible API
	openaiCompatible := map[string]bool{
		"openai":       true,
		"deepseek":     true,
		"moonshot":     true,
		"kimi":         true,
		"qwen":         true,
		"zhipu":        true,
		"baichuan":     true,
		"minimax":      true,
		"volcengine":   true,
		"yi":           true,
		"azure-openai": true,
		"mistral":      true,
	}

	if openaiCompatible[frontendProvider] {
		return "openai"
	}
	return frontendProvider
}

// getModelsForProvider returns common models for a provider type
func getModelsForProvider(providerName string) []string {
	models := map[string][]string{
		"openai": {
			"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4", "gpt-3.5-turbo",
			"o1", "o1-preview", "o1-mini", "o3-mini",
			"gpt-5", "gpt-5-mini", "gpt-5.3-codex", "gpt-5.3-codex-spark", "gpt-5.2-codex", "gpt-5.2 codex",
		},
		"anthropic": {
			"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022",
			"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307",
		},
		"deepseek": {
			"deepseek-chat", "deepseek-reasoner", "deepseek-coder",
		},
		"qwen": {
			"qwen-max", "qwen-plus", "qwen-turbo", "qwen-long",
			"qwen-vl-max", "qwen-vl-plus",
			"qwen2.5-72b-instruct", "qwen2.5-32b-instruct", "qwen2.5-14b-instruct", "qwen2.5-7b-instruct",
		},
		"zhipu": {
			"glm-4-plus", "glm-4-0520", "glm-4", "glm-4-air", "glm-4-airx",
			"glm-4-long", "glm-4-flash", "glm-4v-plus", "glm-4v",
		},
		"moonshot": {
			"kimi-k2.5", "kimi-k2-0905-preview", "kimi-k2-turbo-preview",
			"kimi-k2-thinking", "kimi-k2-thinking-turbo",
			"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k",
		},
		"minimax": {
			"abab6.5s-chat", "abab6.5g-chat", "abab6.5t-chat",
			"abab5.5-chat", "abab5.5s-chat",
		},
		"baichuan": {
			"Baichuan4", "Baichuan3-Turbo", "Baichuan3-Turbo-128k", "Baichuan2-Turbo",
		},
		"volcengine": {
			"doubao-pro-256k", "doubao-pro-128k", "doubao-pro-32k",
			"doubao-lite-128k", "doubao-lite-32k", "doubao-thinking-pro",
		},
		"ernie": {
			"ernie-4.0-8k", "ernie-4.0", "ernie-3.5-8k", "ernie-3.5",
			"ernie-speed-128k", "ernie-speed-8k",
		},
		"hunyuan": {
			"hunyuan-turbo", "hunyuan-pro", "hunyuan-standard", "hunyuan-lite", "hunyuan-code",
		},
		"spark": {
			"spark-4.0-ultra", "spark-3.5-max", "spark-3.0", "spark-2.0", "spark-lite",
		},
		"google": {
			"gemini-3.1-pro-preview", "gemini-2.5-pro", "gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash",
		},
	}

	if m, ok := models[providerName]; ok {
		return m
	}
	return []string{}
}

func normalizeProviderName(providerName string) string {
	provider := strings.ToLower(strings.TrimSpace(providerName))
	switch provider {
	case "", "auto":
		return ""
	case "claude":
		return "anthropic"
	case "kimi":
		return "moonshot"
	default:
		return provider
	}
}

func inferProviderFromModel(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	m = strings.ReplaceAll(m, "_", "-")
	switch {
	case strings.HasPrefix(m, "gpt-"), strings.HasPrefix(m, "o1"), strings.HasPrefix(m, "o3"), strings.HasPrefix(m, "o4"), strings.Contains(m, "codex"):
		return "openai"
	case strings.HasPrefix(m, "claude"):
		return "anthropic"
	case strings.HasPrefix(m, "deepseek"):
		return "deepseek"
	case strings.HasPrefix(m, "qwen"):
		return "qwen"
	case strings.HasPrefix(m, "glm"), strings.HasPrefix(m, "zhipu"):
		return "zhipu"
	case strings.HasPrefix(m, "moonshot"), strings.HasPrefix(m, "kimi"):
		return "moonshot"
	case strings.HasPrefix(m, "abab"):
		return "minimax"
	case strings.HasPrefix(m, "baichuan"):
		return "baichuan"
	case strings.HasPrefix(m, "doubao"):
		return "volcengine"
	case strings.HasPrefix(m, "gemini"):
		return "google"
	default:
		return ""
	}
}

func normalizeModelAlias(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return model
	}

	m = strings.ReplaceAll(m, "_", "-")
	m = strings.Join(strings.Fields(m), "-")

	switch m {
	case "gpt-5-2-codex", "gpt-5.2-codex", "gpt-5.2-codex-preview", "gpt-5.2-codex-beta", "gpt-5.2-codex-experimental":
		return "gpt-5.2-codex"
	case "gpt-5-3-codex", "gpt-5.3-codex":
		return "gpt-5.3-codex"
	default:
		return m
	}
}

func (h *ProxyHandler) resolveCanonicalModelID(model string) string {
	canonical := normalizeModelAlias(model)
	if canonical == "" {
		return canonical
	}

	if h.smartRouter == nil {
		return canonical
	}

	if score := h.smartRouter.GetModelScore(canonical); score != nil {
		if id := strings.TrimSpace(score.Model); id != "" {
			return id
		}
		return canonical
	}

	for modelID, score := range h.smartRouter.GetAllModelScores() {
		if score == nil {
			continue
		}
		displayName := strings.TrimSpace(score.DisplayName)
		if displayName == "" {
			continue
		}
		if normalizeModelAlias(displayName) != canonical {
			continue
		}

		if id := strings.TrimSpace(modelID); id != "" {
			return id
		}
		if id := strings.TrimSpace(score.Model); id != "" {
			return id
		}
	}

	return canonical
}

func (h *ProxyHandler) resolveProviderFromModelRegistry(model string) (string, bool) {
	if h.smartRouter == nil {
		return "", false
	}

	canonicalModel := h.resolveCanonicalModelID(model)
	if canonicalModel == "" {
		return "", false
	}

	if score := h.smartRouter.GetModelScore(canonicalModel); score != nil {
		providerName := normalizeProviderName(score.Provider)
		if providerName != "" && score.Enabled {
			return providerName, true
		}
	}

	for modelID, score := range h.smartRouter.GetAllModelScores() {
		if score == nil || !score.Enabled {
			continue
		}

		currentModelID := strings.TrimSpace(modelID)
		if currentModelID == "" {
			currentModelID = strings.TrimSpace(score.Model)
		}
		if currentModelID == "" {
			continue
		}

		if normalizeModelAlias(currentModelID) != canonicalModel {
			continue
		}

		providerName := normalizeProviderName(score.Provider)
		if providerName != "" {
			return providerName, true
		}
	}

	return "", false
}

func (h *ProxyHandler) findAnyOpenAICompatibleAccount() *limiter.AccountConfig {
	if h.accountManager == nil {
		return nil
	}

	for _, acc := range h.accountManager.GetAllAccounts() {
		if acc == nil || !acc.Enabled {
			continue
		}

		providerType := strings.ToLower(strings.TrimSpace(acc.ProviderType))
		providerName := normalizeProviderName(acc.Provider)
		baseURL := strings.ToLower(strings.TrimSpace(acc.BaseURL))

		if providerType == "openai" {
			return acc
		}

		switch providerName {
		case "openai", "deepseek", "moonshot", "qwen", "zhipu", "baichuan", "minimax", "volcengine", "yi", "azure-openai", "mistral":
			return acc
		}

		if (strings.Contains(baseURL, "openai") || strings.Contains(baseURL, "/v1") || strings.Contains(baseURL, "compatible")) && !strings.Contains(baseURL, "anthropic") && !strings.Contains(baseURL, "googleapis") {
			return acc
		}
	}

	return nil
}

func isModelNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if providerErr, ok := err.(*provider.ProviderError); ok {
		if message := strings.ToLower(strings.TrimSpace(providerErr.Message)); message != "" {
			msg = message
		}

		if providerErr.Code == http.StatusNotFound {
			return true
		}

		errType := strings.ToLower(strings.TrimSpace(providerErr.Type))
		if strings.Contains(errType, "model_not_found") || strings.Contains(errType, "model-not-found") {
			return true
		}
	}

	keywords := []string{
		"model not found",
		"invalid model",
		"does not exist",
		"unknown model",
		"模型不存在",
		"模型代码",
	}
	for _, kw := range keywords {
		if strings.Contains(msg, strings.ToLower(kw)) {
			return true
		}
	}

	return false
}

func isReasoningEffortUnsupportedError(err error) bool {
	if err == nil {
		return false
	}

	statusCode := 0
	errType := ""
	errMessage := strings.ToLower(strings.TrimSpace(err.Error()))
	if providerErr, ok := err.(*provider.ProviderError); ok {
		statusCode = providerErr.Code
		errType = strings.ToLower(strings.TrimSpace(providerErr.Type))
		if msg := strings.TrimSpace(providerErr.Message); msg != "" {
			errMessage = strings.ToLower(msg)
		}
	}

	if statusCode != 0 && statusCode != http.StatusBadRequest && statusCode != http.StatusUnprocessableEntity {
		return false
	}

	if errType != "" && errType != "invalid_request_error" {
		return false
	}

	reasoningTokens := []string{"reasoning_effort", "reasoning.effort", "reasoning effort", "thinking"}
	hasReasoningToken := false
	for _, token := range reasoningTokens {
		if strings.Contains(errMessage, token) {
			hasReasoningToken = true
			break
		}
	}
	if !hasReasoningToken {
		return false
	}

	unsupportedSignals := []string{
		"unsupported parameter",
		"unknown parameter",
		"is not supported",
		"does not support",
		"unsupported",
		"not supported",
	}
	for _, signal := range unsupportedSignals {
		if strings.Contains(errMessage, signal) {
			return true
		}
	}

	return false
}

func cloneProviderRequestWithoutReasoningEffort(req *provider.ChatRequest) (*provider.ChatRequest, bool) {
	if req == nil {
		return nil, false
	}
	if req.Extra == nil {
		return nil, false
	}
	if _, ok := req.Extra["reasoning_effort"]; !ok {
		return nil, false
	}

	clonedReq := *req
	clonedExtra := make(map[string]interface{}, len(req.Extra))
	for key, value := range req.Extra {
		clonedExtra[key] = value
	}
	delete(clonedExtra, "reasoning_effort")
	if len(clonedExtra) == 0 {
		clonedReq.Extra = nil
	} else {
		clonedReq.Extra = clonedExtra
	}

	if len(req.Messages) > 0 {
		clonedReq.Messages = append([]provider.ChatMessage(nil), req.Messages...)
	}
	if len(req.Tools) > 0 {
		clonedReq.Tools = append([]provider.Tool(nil), req.Tools...)
	}

	return &clonedReq, true
}

func (h *ProxyHandler) selectFallbackModel(currentModel, providerName string) string {
	if h.smartRouter == nil {
		return ""
	}

	currentNormalized := normalizeModelAlias(currentModel)

	providerName = normalizeProviderName(providerName)
	if providerName == "" {
		providerName = inferProviderFromModel(currentModel)
	}

	if providerName != "" {
		if def := strings.TrimSpace(h.smartRouter.GetProviderDefault(providerName)); def != "" {
			defNorm := normalizeModelAlias(def)
			if defNorm != "" && defNorm != currentNormalized {
				return defNorm
			}
		}
	}

	scores := h.smartRouter.GetAllModelScores()
	if len(scores) == 0 {
		return ""
	}

	isCodexRequest := strings.Contains(strings.ToLower(currentModel), "codex")
	candidateSet := make(map[string]struct{})
	candidates := make([]string, 0)
	for model, score := range scores {
		if score == nil || !score.Enabled {
			continue
		}
		if providerName != "" && normalizeProviderName(score.Provider) != providerName {
			continue
		}
		if isCodexRequest && !strings.Contains(strings.ToLower(model), "codex") {
			continue
		}
		normalized := normalizeModelAlias(model)
		if normalized == "" || normalized == currentNormalized {
			continue
		}
		if _, exists := candidateSet[normalized]; exists {
			continue
		}
		candidateSet[normalized] = struct{}{}
		candidates = append(candidates, normalized)
	}

	if len(candidates) == 0 && isCodexRequest {
		for model, score := range scores {
			if score == nil || !score.Enabled {
				continue
			}
			if providerName != "" && normalizeProviderName(score.Provider) != providerName {
				continue
			}
			normalized := normalizeModelAlias(model)
			if normalized == "" || normalized == currentNormalized {
				continue
			}
			if _, exists := candidateSet[normalized]; exists {
				continue
			}
			candidateSet[normalized] = struct{}{}
			candidates = append(candidates, normalized)
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Strings(candidates)
	return candidates[0]
}

// convertChoices converts provider choices to handler choices
func convertChoices(choices []provider.Choice) []Choice {
	result := make([]Choice, len(choices))
	for i, ch := range choices {
		var toolCalls []ToolCall
		if len(ch.Message.ToolCalls) > 0 {
			toolCalls = make([]ToolCall, len(ch.Message.ToolCalls))
			for j, tc := range ch.Message.ToolCalls {
				toolCalls[j] = ToolCall{
					Index: tc.Index,
					ID:    tc.ID,
					Type:  tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		result[i] = Choice{
			Index: ch.Index,
			Message: &ChatMessage{
				Role:      ch.Message.Role,
				Content:   ch.Message.Content,
				Name:      ch.Message.Name,
				ToolCalls: toolCalls,
			},
			FinishReason: ch.FinishReason,
		}
	}
	return result
}

// handleStreamResponse handles streaming chat completion
func (h *ProxyHandler) handleStreamResponse(
	c *gin.Context,
	p provider.Provider,
	req *provider.ChatRequest,
	explicitReasoningEffort bool,
	usageMeta usageRuntimeMeta,
	userID string,
	apiKey string,
	prompt string,
	semanticQuery string,
	allowSemantic bool,
	cacheEnabled bool,
	cacheWriteAllowed bool,
	recommendedTTL time.Duration,
	taskType string,
	taskTypeSource string,
	difficulty routing.DifficultyLevel,
	providerName string,
	cacheModelDimension string,
	cacheKeyPayload interface{},
	originalModelID string,
	experimentTag string,
	domainTag string,
) {
	startTime := time.Now()
	reasoningEffortDowngraded := false

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Local-Cache-Hit", "0")

	ctx := c.Request.Context()
	providerProtocol := p.Name()
	providerLabel := strings.TrimSpace(providerName)
	if providerLabel == "" {
		providerLabel = providerProtocol
	}

	// Get stream channel from provider
	streamReq := req
	stream, err := p.StreamChat(ctx, streamReq)
	if err != nil && explicitReasoningEffort && isReasoningEffortUnsupportedError(err) {
		if downgradedReq, ok := cloneProviderRequestWithoutReasoningEffort(streamReq); ok {
			streamReq = downgradedReq
			reasoningEffortDowngraded = true
			stream, err = p.StreamChat(ctx, streamReq)
		}
	}
	if err != nil {
		// CHANGED: include provider/user/api info in usage logs for stream start failures.
		h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, time.Since(startTime), 0, false, false, 0, 0, 0, "actual", taskType, string(difficulty), "", experimentTag, domainTag)
		admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, false, time.Since(startTime).Milliseconds(), 0)
		h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerLabel, prompt, err.Error(), http.StatusBadGateway)
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	var totalTokens int
	var promptTokens int
	var completionTokens int
	var cachedReadTokens int
	var fullContent strings.Builder
	receivedChunks := 0
	hasReasoningOutput := false
	fallbackNeeded := false
	streamCompletedByDone := false
	var ttftMs int64 = 0 // Time to first token
	// Stream chunks to client
streamLoop:
	for {
		select {
		case <-ctx.Done():
			cancelMessage := "stream canceled"
			if ctxErr := ctx.Err(); ctxErr != nil {
				cancelMessage = ctxErr.Error()
			}
			h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, time.Since(startTime), 0, false, false, ttftMs, promptTokens, 0, "actual", taskType, string(difficulty), "", experimentTag, domainTag)
			admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, false, time.Since(startTime).Milliseconds(), 0)
			h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerLabel, prompt, cancelMessage, 499)
			return
		case chunk, ok := <-stream:
			if !ok {
				break streamLoop
			}
			if chunk == nil {
				continue
			}
			if chunk.Error != nil {
				h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, time.Since(startTime), 0, false, false, ttftMs, promptTokens, 0, "actual", taskType, string(difficulty), "", experimentTag, domainTag)
				admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, false, time.Since(startTime).Milliseconds(), 0)
				errorStatusCode := chunk.Error.Code
				if errorStatusCode < http.StatusBadRequest || errorStatusCode >= 600 {
					errorStatusCode = http.StatusBadGateway
				}
				h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerLabel, prompt, chunk.Error.Message, errorStatusCode)
				c.SSEvent("error", gin.H{
					"error": gin.H{
						"code":    chunk.Error.Code,
						"message": chunk.Error.Message,
					},
				})
				c.Writer.Flush()
				return
			}

			receivedChunks++
			if chunk.Usage != nil {
				totalTokens = chunk.Usage.TotalTokens
				promptTokens = chunk.Usage.PromptTokens
				completionTokens = chunk.Usage.CompletionTokens
				cachedReadTokens = chunk.Usage.CachedReadTokens
			}
			for _, ch := range chunk.Choices {
				if ch.Delta != nil && ch.Delta.Content != "" {
					fullContent.WriteString(ch.Delta.Content)
					// Record TTFT on first token
					if ttftMs == 0 {
						ttftMs = time.Since(startTime).Milliseconds()
					}
				}
				if ch.Delta != nil && (strings.TrimSpace(ch.Delta.ReasoningContent) != "" || strings.TrimSpace(ch.Delta.Reasoning) != "") {
					hasReasoningOutput = true
				}
			}
			if chunk.Done {
				if strings.TrimSpace(fullContent.String()) == "" && !hasReasoningOutput {
					fallbackNeeded = true
					break streamLoop
				}

				resolvedUsage, usageSource := resolveUsageWithFallback(
					prompt,
					fullContent.String(),
					usageTokens{
						Prompt:     promptTokens,
						Completion: completionTokens,
						Total:      totalTokens,
						CachedRead: cachedReadTokens,
					},
				)

				// Send the final chunk with usage (if present) before [DONE]
				if chunk.Usage != nil || len(chunk.Choices) > 0 {
					streamResp := StreamingResponse{
						ID:      chunk.ID,
						Object:  chunk.Object,
						Created: chunk.Created,
						Model:   chunk.Model,
						Choices: make([]StreamChoice, len(chunk.Choices)),
					}
					if reasoningEffortDowngraded {
						streamResp.GatewayMeta = &GatewayMeta{ReasoningEffortDowngraded: true}
					}
					if resolvedUsage.Total > 0 {
						streamResp.Usage = &Usage{
							PromptTokens:     resolvedUsage.Prompt,
							CompletionTokens: resolvedUsage.Completion,
							TotalTokens:      resolvedUsage.Total,
							CachedReadTokens: resolvedUsage.CachedRead,
						}
					}
					for i, ch := range chunk.Choices {
						var finishReason *string
						if ch.FinishReason != "" {
							finishReason = &ch.FinishReason
						}
						var delta *ChatMessage
						if ch.Delta != nil {
							delta = &ChatMessage{
								Role:    ch.Delta.Role,
								Content: ch.Delta.Content,
							}
						}
						streamResp.Choices[i] = StreamChoice{
							Index:        ch.Index,
							Delta:        delta,
							FinishReason: finishReason,
						}
					}
					c.SSEvent("message", streamResp)
					c.Writer.Flush()
				}

				if h.traceRecorder != nil {
					h.traceRecorder.RecordSimpleSpan(ctx, "provider.chat", map[string]interface{}{
						"duration_ms":   time.Since(startTime).Milliseconds(),
						"model":         req.Model,
						"provider":      providerLabel,
						"provider_type": providerProtocol,
						"provider_name": providerLabel,
						"stream":        true,
						"success":       true,
					})
				}

				// Send [DONE] marker
				c.SSEvent("message", "[DONE]")
				c.Writer.Flush()

				if h.traceRecorder != nil {
					responsePayload := map[string]interface{}{
						"choices": []map[string]interface{}{
							{
								"index": 0,
								"message": map[string]interface{}{
									"role":    "assistant",
									"content": fullContent.String(),
								},
							},
						},
					}
					responseBody, marshalErr := json.Marshal(responsePayload)
					if marshalErr != nil {
						responseBody = []byte("{}")
					}
					attrs := map[string]interface{}{
						"duration_ms": time.Since(startTime).Milliseconds(),
						"model":       req.Model,
						"provider":    providerName,
						"status_code": http.StatusOK,
						"cache_hit":   false,
					}
					for key, value := range buildTraceMessageAttributes(prompt, responseBody) {
						attrs[key] = value
					}
					h.traceRecorder.RecordSimpleSpan(ctx, "http.response", attrs)
				}

				// Record metrics for stream completion
				latency := time.Since(startTime)
				// CHANGED: include provider/user/api info and prompt tokens in usage logs for stream completion.
				h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, latency, resolvedUsage.Total, true, false, ttftMs, resolvedUsage.Prompt, resolvedUsage.CachedRead, usageSource, taskType, string(difficulty), "", experimentTag, domainTag)
				admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, true, latency.Milliseconds(), resolvedUsage.Total)

				// Record successful model name mapping if different from original
				if h.modelMappingCache != nil && originalModelID != "" && originalModelID != req.Model {
					h.modelMappingCache.RecordSuccess(providerName, originalModelID, req.Model)
				}

				// Record stream response into cache for observability/management page
				// Skip caching when provider returned no text content to avoid empty-cache pollution.
				if cacheEnabled && cacheWriteAllowed && h.cache != nil && h.cache.ResponseCache != nil && recommendedTTL > 0 && strings.TrimSpace(fullContent.String()) != "" {
					responsePayload := map[string]interface{}{
						"id":               chunk.ID,
						"object":           "chat.completion",
						"created":          chunk.Created,
						"model":            req.Model,
						"task_type":        taskType,
						"task_type_source": taskTypeSource,
						"choices": []map[string]interface{}{
							{
								"index": 0,
								"message": map[string]interface{}{
									"role":    "assistant",
									"content": fullContent.String(),
								},
								"finish_reason": "stop",
							},
						},
						"usage": buildCachedUsage(resolvedUsage.Prompt, resolvedUsage.Completion, resolvedUsage.CachedRead, resolvedUsage.Total),
					}

					if body, err := json.Marshal(responsePayload); err == nil {
						cacheKey, keyErr := h.cache.ResponseCache.GenerateKey(providerName, cacheModelDimension, cacheKeyPayload)
						if keyErr == nil {
							cachedResp := &cache.CachedResponse{
								StatusCode:     http.StatusOK,
								Headers:        map[string]string{"Content-Type": "application/json"},
								Body:           body,
								CreatedAt:      time.Now(),
								HitCount:       0,
								HitModels:      map[string]int64{req.Model: 1},
								Provider:       providerName,
								Model:          req.Model,
								Prompt:         prompt,
								TaskType:       taskType,
								TaskTypeSource: taskTypeSource,
							}
							if err := h.writeResponseCacheEntry(c.Request.Context(), cacheKey, cachedResp, recommendedTTL, req.Model, providerName, taskType, taskTypeSource); err != nil {
								logrus.WithError(err).WithField("cache_key", cacheKey).Warn("failed to write streamed response cache entry")
							}
							if shouldUsePromptOnlyCache(routing.TaskType(taskType)) {
								h.pruneDuplicateResponseEntries(c.Request.Context(), providerName, req.Model, taskType, prompt, cacheKey)
							}
						}

						if allowSemantic && h.semanticCache != nil && routing.TaskType(taskType) != routing.TaskTypeCreative {
							queryVector := cache.SimpleEmbedding(semanticQuery, 1536)
							h.semanticCache.Set(
								c.Request.Context(),
								semanticQuery,
								queryVector,
								body,
								req.Model,
								providerName,
								taskType,
								recommendedTTL,
							)
						}
					}
				}
				streamCompletedByDone = true
				break streamLoop
			}
			streamResp := StreamingResponse{
				ID:      chunk.ID,
				Object:  chunk.Object,
				Created: chunk.Created,
				Model:   chunk.Model,
				Choices: make([]StreamChoice, len(chunk.Choices)),
			}

			// Include usage if present
			if chunk.Usage != nil {
				streamResp.Usage = &Usage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
					CachedReadTokens: chunk.Usage.CachedReadTokens,
				}
			}

			for i, ch := range chunk.Choices {
				var finishReason *string
				if ch.FinishReason != "" {
					finishReason = &ch.FinishReason
				}
				var delta *ChatMessage
				if ch.Delta != nil {
					var toolCalls []ToolCall
					if len(ch.Delta.ToolCalls) > 0 {
						toolCalls = make([]ToolCall, len(ch.Delta.ToolCalls))
						for j, tc := range ch.Delta.ToolCalls {
							toolCalls[j] = ToolCall{
								Index: tc.Index,
								ID:    tc.ID,
								Type:  tc.Type,
								Function: FunctionCall{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							}
						}
					}
					delta = &ChatMessage{
						Role:      ch.Delta.Role,
						Content:   ch.Delta.Content,
						ToolCalls: toolCalls,
					}
					// 处理深度思考内容 (DeepSeek R1)
					if ch.Delta.ReasoningContent != "" || ch.Delta.Reasoning != "" {
						reasoning := ch.Delta.ReasoningContent
						if reasoning == "" {
							reasoning = ch.Delta.Reasoning
						}
						delta.ReasoningContent = reasoning
					}
				}
				streamResp.Choices[i] = StreamChoice{
					Index:        ch.Index,
					Delta:        delta,
					FinishReason: finishReason,
				}
			}

			c.SSEvent("message", streamResp)
			c.Writer.Flush()
		}
	}

	if receivedChunks > 0 && !fallbackNeeded && !streamCompletedByDone {
		resolvedUsage, usageSource := resolveUsageWithFallback(
			prompt,
			fullContent.String(),
			usageTokens{
				Prompt:     promptTokens,
				Completion: completionTokens,
				Total:      totalTokens,
				CachedRead: cachedReadTokens,
			},
		)

		if h.traceRecorder != nil {
			h.traceRecorder.RecordSimpleSpan(ctx, "provider.chat", map[string]interface{}{
				"duration_ms":   time.Since(startTime).Milliseconds(),
				"model":         req.Model,
				"provider":      providerLabel,
				"provider_type": providerProtocol,
				"provider_name": providerLabel,
				"stream":        true,
				"success":       true,
			})
		}

		c.SSEvent("message", "[DONE]")
		c.Writer.Flush()

		if h.traceRecorder != nil {
			responsePayload := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": fullContent.String(),
						},
					},
				},
			}
			responseBody, marshalErr := json.Marshal(responsePayload)
			if marshalErr != nil {
				responseBody = []byte("{}")
			}
			attrs := map[string]interface{}{
				"duration_ms": time.Since(startTime).Milliseconds(),
				"model":       req.Model,
				"provider":    providerName,
				"status_code": http.StatusOK,
				"cache_hit":   false,
			}
			for key, value := range buildTraceMessageAttributes(prompt, responseBody) {
				attrs[key] = value
			}
			h.traceRecorder.RecordSimpleSpan(ctx, "http.response", attrs)
		}

		latency := time.Since(startTime)
		h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, latency, resolvedUsage.Total, true, false, ttftMs, resolvedUsage.Prompt, resolvedUsage.CachedRead, usageSource, taskType, string(difficulty), "", experimentTag, domainTag)
		admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, true, latency.Milliseconds(), resolvedUsage.Total)

		if h.modelMappingCache != nil && originalModelID != "" && originalModelID != req.Model {
			h.modelMappingCache.RecordSuccess(providerName, originalModelID, req.Model)
		}

		if cacheEnabled && cacheWriteAllowed && h.cache != nil && h.cache.ResponseCache != nil && recommendedTTL > 0 && strings.TrimSpace(fullContent.String()) != "" {
			responsePayload := map[string]interface{}{
				"id":               fmt.Sprintf("chatcmpl-stream-%d", time.Now().UnixNano()),
				"object":           "chat.completion",
				"created":          time.Now().Unix(),
				"model":            req.Model,
				"task_type":        taskType,
				"task_type_source": taskTypeSource,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": fullContent.String(),
						},
						"finish_reason": "stop",
					},
				},
				"usage": buildCachedUsage(resolvedUsage.Prompt, resolvedUsage.Completion, resolvedUsage.CachedRead, resolvedUsage.Total),
			}

			if body, err := json.Marshal(responsePayload); err == nil {
				cacheKey, keyErr := h.cache.ResponseCache.GenerateKey(providerName, cacheModelDimension, cacheKeyPayload)
				if keyErr == nil {
					cachedResp := &cache.CachedResponse{
						StatusCode:     http.StatusOK,
						Headers:        map[string]string{"Content-Type": "application/json"},
						Body:           body,
						CreatedAt:      time.Now(),
						HitCount:       0,
						HitModels:      map[string]int64{req.Model: 1},
						Provider:       providerName,
						Model:          req.Model,
						Prompt:         prompt,
						TaskType:       taskType,
						TaskTypeSource: taskTypeSource,
					}
					if err := h.writeResponseCacheEntry(c.Request.Context(), cacheKey, cachedResp, recommendedTTL, req.Model, providerName, taskType, taskTypeSource); err != nil {
						logrus.WithError(err).WithField("cache_key", cacheKey).Warn("failed to write streamed response cache entry")
					}
					if shouldUsePromptOnlyCache(routing.TaskType(taskType)) {
						h.pruneDuplicateResponseEntries(c.Request.Context(), providerName, req.Model, taskType, prompt, cacheKey)
					}
				}

				if allowSemantic && h.semanticCache != nil && routing.TaskType(taskType) != routing.TaskTypeCreative {
					queryVector := cache.SimpleEmbedding(semanticQuery, 1536)
					h.semanticCache.Set(
						c.Request.Context(),
						semanticQuery,
						queryVector,
						body,
						req.Model,
						providerName,
						taskType,
						recommendedTTL,
					)
				}
			}
		}
	}

	if !streamCompletedByDone && (receivedChunks == 0 || fallbackNeeded) {
		// Some providers may return an empty stream even though non-stream works.
		// Fallback to a non-stream request and convert it to SSE payload.
		nonStreamReq := *req
		nonStreamReq.Stream = false

		resp, err := p.Chat(ctx, &nonStreamReq)
		if err != nil && explicitReasoningEffort && !reasoningEffortDowngraded && isReasoningEffortUnsupportedError(err) {
			if downgradedReq, ok := cloneProviderRequestWithoutReasoningEffort(&nonStreamReq); ok {
				nonStreamReq = *downgradedReq
				reasoningEffortDowngraded = true
				resp, err = p.Chat(ctx, &nonStreamReq)
			}
		}
		originalModel := nonStreamReq.Model

		if err != nil && !isModelNotFoundError(err) {
			if fallbackModel := h.selectFallbackModel(originalModel, providerName); fallbackModel != "" && fallbackModel != nonStreamReq.Model {
				nonStreamReq.Model = fallbackModel
				resp, err = p.Chat(ctx, &nonStreamReq)
			}
		}

		if err != nil {
			// CHANGED: include provider/user/api info in usage logs for stream fallback failures.
			h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, time.Since(startTime), 0, false, false, 0, 0, 0, "actual", taskType, string(difficulty), "", experimentTag, domainTag)
			admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, false, time.Since(startTime).Milliseconds(), 0)
			h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerLabel, prompt, err.Error(), http.StatusBadGateway)
			c.SSEvent("error", gin.H{"error": err.Error()})
			c.Writer.Flush()
			return
		}

		if resp == nil || len(resp.Choices) == 0 {
			// CHANGED: include provider/user/api info in usage logs for empty fallback responses.
			h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, time.Since(startTime), 0, false, false, 0, 0, 0, "actual", taskType, string(difficulty), "", experimentTag, domainTag)
			admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, false, time.Since(startTime).Milliseconds(), 0)
			h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerLabel, prompt, "Provider returned empty response", http.StatusBadGateway)
			c.SSEvent("error", gin.H{"error": "Provider returned empty response"})
			c.Writer.Flush()
			return
		}

		content := getTextContent(resp.Choices[0].Message.Content)
		if strings.TrimSpace(content) == "" {
			// CHANGED: include provider/user/api info in usage logs for empty fallback content.
			h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, time.Since(startTime), 0, false, false, 0, 0, 0, "actual", taskType, string(difficulty), "", experimentTag, domainTag)
			admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, false, time.Since(startTime).Milliseconds(), 0)
			h.recordHTTPResponseErrorSpan(ctx, startTime, req.Model, providerLabel, prompt, "Provider returned empty content", http.StatusBadGateway)
			c.SSEvent("error", gin.H{"error": "Provider returned empty content"})
			c.Writer.Flush()
			return
		}

		if h.traceRecorder != nil {
			h.traceRecorder.RecordSimpleSpan(ctx, "provider.chat", map[string]interface{}{
				"duration_ms":          time.Since(startTime).Milliseconds(),
				"model":                nonStreamReq.Model,
				"provider":             providerLabel,
				"provider_type":        providerProtocol,
				"provider_name":        providerLabel,
				"stream":               false,
				"success":              true,
				"fallback_from_stream": true,
			})
		}

		resolvedUsage, usageSource := resolveUsageWithFallback(
			prompt,
			content,
			usageTokens{
				Prompt:     resp.Usage.PromptTokens,
				Completion: resp.Usage.CompletionTokens,
				Total:      resp.Usage.TotalTokens,
				CachedRead: resp.Usage.CachedReadTokens,
			},
		)

		created := resp.Created
		if created == 0 {
			created = time.Now().Unix()
		}

		model := resp.Model
		if model == "" {
			model = req.Model
		}

		id := resp.ID
		if id == "" {
			id = fmt.Sprintf("chatcmpl-fallback-%d", time.Now().UnixNano())
		}

		chunk := StreamingResponse{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []StreamChoice{{
				Index: 0,
				Delta: &ChatMessage{Role: "assistant", Content: content},
			}},
		}
		c.SSEvent("message", chunk)
		c.Writer.Flush()

		finishReason := "stop"
		finalChunk := StreamingResponse{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []StreamChoice{{
				Index:        0,
				Delta:        &ChatMessage{},
				FinishReason: &finishReason,
			}},
			Usage: &Usage{
				PromptTokens:     resolvedUsage.Prompt,
				CompletionTokens: resolvedUsage.Completion,
				TotalTokens:      resolvedUsage.Total,
				CachedReadTokens: resolvedUsage.CachedRead,
			},
		}
		if reasoningEffortDowngraded {
			finalChunk.GatewayMeta = &GatewayMeta{ReasoningEffortDowngraded: true}
		}
		c.SSEvent("message", finalChunk)
		c.Writer.Flush()

		c.SSEvent("message", "[DONE]")
		c.Writer.Flush()

		// Persist fallback response to response cache to avoid repeated upstream empty-stream misses.
		if cacheEnabled && cacheWriteAllowed && h.cache != nil && h.cache.ResponseCache != nil && recommendedTTL > 0 {
			responsePayload := map[string]interface{}{
				"id":      id,
				"object":  "chat.completion",
				"created": created,
				"model":   model,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": content,
						},
						"finish_reason": "stop",
					},
				},
				"usage": buildCachedUsage(resolvedUsage.Prompt, resolvedUsage.Completion, resolvedUsage.CachedRead, resolvedUsage.Total),
			}

			if body, marshalErr := json.Marshal(responsePayload); marshalErr == nil {
				cacheKey, keyErr := h.cache.ResponseCache.GenerateKey(providerName, cacheModelDimension, cacheKeyPayload)
				if keyErr == nil {
					cachedResp := &cache.CachedResponse{
						StatusCode:     http.StatusOK,
						Headers:        map[string]string{"Content-Type": "application/json"},
						Body:           body,
						CreatedAt:      time.Now(),
						HitCount:       0,
						HitModels:      map[string]int64{model: 1},
						Provider:       providerName,
						Model:          model,
						Prompt:         prompt,
						TaskType:       taskType,
						TaskTypeSource: taskTypeSource,
					}
					if err := h.writeResponseCacheEntry(c.Request.Context(), cacheKey, cachedResp, recommendedTTL, model, providerName, taskType, taskTypeSource); err != nil {
						logrus.WithError(err).WithField("cache_key", cacheKey).Warn("failed to write fallback cache entry")
					}
					if shouldUsePromptOnlyCache(routing.TaskType(taskType)) {
						h.pruneDuplicateResponseEntries(c.Request.Context(), providerName, model, taskType, prompt, cacheKey)
					}
				}

				if allowSemantic && h.semanticCache != nil && routing.TaskType(taskType) != routing.TaskTypeCreative {
					queryVector := cache.SimpleEmbedding(semanticQuery, 1536)
					h.semanticCache.Set(
						c.Request.Context(),
						semanticQuery,
						queryVector,
						body,
						model,
						providerName,
						taskType,
						recommendedTTL,
					)
				}
			}
		}

		latency := time.Since(startTime)
		// CHANGED: include provider/user/api info and prompt tokens in usage logs for fallback success.
		h.recordMetricsExtendedWithMetaAndUsageSource(usageMeta, userID, apiKey, req.Model, providerName, latency, resolvedUsage.Total, true, false, 0, resolvedUsage.Prompt, resolvedUsage.CachedRead, usageSource, taskType, string(difficulty), "", experimentTag, domainTag)
		admin.RecordRequestResult(req.Model, providerName, routing.TaskType(taskType), difficulty, true, latency.Milliseconds(), resolvedUsage.Total)

		if h.traceRecorder != nil {
			responsePayload := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": content,
						},
					},
				},
			}
			responseBody, marshalErr := json.Marshal(responsePayload)
			if marshalErr != nil {
				responseBody = []byte("{}")
			}
			attrs := map[string]interface{}{
				"duration_ms": time.Since(startTime).Milliseconds(),
				"model":       req.Model,
				"provider":    providerName,
				"status_code": http.StatusOK,
				"cache_hit":   false,
			}
			for key, value := range buildTraceMessageAttributes(prompt, responseBody) {
				attrs[key] = value
			}
			h.traceRecorder.RecordSimpleSpan(ctx, "http.response", attrs)
		}

		// Record successful model name mapping if fallback used a different model
		if h.modelMappingCache != nil && originalModelID != "" && originalModelID != nonStreamReq.Model {
			h.modelMappingCache.RecordSuccess(providerName, originalModelID, nonStreamReq.Model)
		}
	}
}

// Completions proxies completion requests to AI providers
func (h *ProxyHandler) Completions(c *gin.Context) {
	startTime := time.Now()

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodySize)

	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if err.Error() == "http: request body too large" {
			Error(c, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds maximum size of 10MB")
			return
		}
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if req.Model == "" {
		BadRequest(c, "model is required")
		return
	}

	if req.Prompt == nil {
		BadRequest(c, "prompt is required")
		return
	}

	targetProvider, ok := h.registry.GetByModel(req.Model)
	if !ok {
		h.recordMetrics("", "", req.Model, time.Since(startTime), 0, false)
		Error(c, http.StatusServiceUnavailable, ErrCodeProviderError, "No available provider for model: "+req.Model)
		return
	}

	prompts := parsePrompt(req.Prompt)
	if len(prompts) == 0 {
		BadRequest(c, "prompt cannot be empty")
		return
	}

	promptText := prompts[0]
	defaultTemp := getDefaultTemperature(req.Model)

	providerReq := &provider.ChatRequest{
		Model:       req.Model,
		Temperature: getFloat64(req.Temperature, defaultTemp),
		MaxTokens:   getInt(req.MaxTokens, 0),
		Stream:      req.Stream,
		Extra:       req.Extra,
		Messages: []provider.ChatMessage{
			{
				Role:    "user",
				Content: promptText,
			},
		},
	}

	if req.Stream {
		h.handleCompletionStreamResponse(c, targetProvider, providerReq, req.Model, startTime)
		return
	}

	ctx := c.Request.Context()
	resp, err := targetProvider.Chat(ctx, providerReq)
	if err != nil {
		h.recordMetrics("", "", req.Model, time.Since(startTime), 0, false)
		if providerErr, ok := err.(*provider.ProviderError); ok {
			statusCode := providerErr.Code
			if statusCode < http.StatusBadRequest || statusCode >= 600 {
				statusCode = http.StatusBadGateway
			}
			Error(c, statusCode, ErrCodeProviderError, providerErr.Message)
			return
		}
		ProviderError(c, err.Error(), "")
		return
	}

	var content string
	if len(resp.Choices) > 0 {
		content = getTextContent(resp.Choices[0].Message.Content)
	}

	Success(c, CompletionResponse{
		ID:      resp.ID,
		Object:  "text_completion",
		Created: resp.Created,
		Model:   resp.Model,
		Choices: []CompletionChoice{
			{
				Text:         content,
				Index:        0,
				FinishReason: resp.Choices[0].FinishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
			CachedReadTokens: resp.Usage.CachedReadTokens,
		},
	})

	latency := time.Since(startTime)
	h.recordMetrics("", "", req.Model, latency, resp.Usage.TotalTokens, true)
}

func parsePrompt(prompt interface{}) []string {
	switch v := prompt.(type) {
	case string:
		return []string{v}
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, p := range v {
			if s, ok := p.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return v
	default:
		return nil
	}
}

func (h *ProxyHandler) handleCompletionStreamResponse(c *gin.Context, p provider.Provider, req *provider.ChatRequest, model string, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	ctx := c.Request.Context()
	stream, err := p.StreamChat(ctx, req)
	if err != nil {
		h.recordMetrics("", "", model, time.Since(startTime), 0, false)
		ProviderError(c, "Provider stream request failed", err.Error())
		return
	}

	for chunk := range stream {
		if chunk.Error != nil {
			h.recordMetrics("", "", model, time.Since(startTime), 0, false)
			c.SSEvent("error", gin.H{
				"error": gin.H{
					"code":    chunk.Error.Code,
					"message": chunk.Error.Message,
				},
			})
			c.Writer.Flush()
			return
		}

		if chunk.Done {
			c.SSEvent("message", map[string]interface{}{
				"id":      chunk.ID,
				"object":  "text_completion_chunk",
				"created": chunk.Created,
				"model":   chunk.Model,
				"choices": []map[string]interface{}{
					{
						"text":          "",
						"index":         0,
						"finish_reason": "stop",
					},
				},
			})
			c.SSEvent("message", "[DONE]")
			c.Writer.Flush()
			break
		}

		var content string
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content = chunk.Choices[0].Delta.Content
		}

		c.SSEvent("message", map[string]interface{}{
			"id":      chunk.ID,
			"object":  "text_completion_chunk",
			"created": chunk.Created,
			"model":   chunk.Model,
			"choices": []map[string]interface{}{
				{
					"text":          content,
					"index":         0,
					"finish_reason": nil,
				},
			},
		})
		c.Writer.Flush()
	}

	latency := time.Since(startTime)
	h.recordMetrics("", "", model, latency, 0, true)
}

// Embeddings proxies embedding requests to AI providers
func (h *ProxyHandler) Embeddings(c *gin.Context) {
	// Limit request body size to prevent DoS attacks
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodySize)

	// Parse request
	var req EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if err.Error() == "http: request body too large" {
			Error(c, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds maximum size of 10MB")
			return
		}
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if req.Model == "" {
		BadRequest(c, "model is required")
		return
	}

	Success(c, EmbeddingResponse{
		Object: "list",
		Data: []EmbeddingData{
			{
				Object:    "embedding",
				Index:     0,
				Embedding: []float64{},
			},
		},
		Model: req.Model,
		Usage: Usage{
			PromptTokens: 0,
			TotalTokens:  0,
		},
	})
}

// ListProviders returns available AI providers
func (h *ProxyHandler) ListProviders(c *gin.Context) {
	providers := h.registry.ListEnabled()

	result := make([]gin.H, 0, len(providers))
	for _, p := range providers {
		result = append(result, gin.H{
			"name":    p.Name(),
			"models":  p.Models(),
			"enabled": p.IsEnabled(),
		})
	}

	Success(c, gin.H{
		"providers": result,
	})
}

// ListConfiguredProviders returns providers from account config (public)
func (h *ProxyHandler) ListConfiguredProviders(c *gin.Context) {
	type providerConfigView struct {
		Name    string   `json:"name"`
		Models  []string `json:"models"`
		Enabled bool     `json:"enabled"`
	}
	type accountProviderStatus struct {
		HasAccount bool
		Enabled    bool
	}

	ensureProvider := func(providerMap map[string]*providerConfigView, providerName string) *providerConfigView {
		if existing, ok := providerMap[providerName]; ok {
			return existing
		}
		next := &providerConfigView{
			Name:   providerName,
			Models: make([]string, 0),
		}
		providerMap[providerName] = next
		return next
	}

	providerMap := make(map[string]*providerConfigView)
	accountStatus := make(map[string]accountProviderStatus)
	smartRouterModels := h.getEnabledSmartRouterModelsByProvider()

	if h.accountManager != nil {
		for _, acc := range h.accountManager.GetAllAccounts() {
			providerName := inferProviderNameFromAccount(acc)
			if providerName == "" {
				continue
			}
			status := accountStatus[providerName]
			status.HasAccount = true
			status.Enabled = status.Enabled || acc.Enabled
			accountStatus[providerName] = status
		}
	}

	for _, p := range h.registry.ListEnabled() {
		providerName := normalizeProviderName(p.Name())
		if providerName == "" {
			continue
		}

		entry := ensureProvider(providerMap, providerName)
		entry.Models = mergeModelLists(entry.Models, p.Models())
		entry.Models = mergeModelLists(entry.Models, smartRouterModels[providerName])

		if status, ok := accountStatus[providerName]; ok && status.HasAccount {
			entry.Enabled = status.Enabled
		} else {
			entry.Enabled = p.IsEnabled()
		}
	}

	for providerName, status := range accountStatus {
		entry := ensureProvider(providerMap, providerName)
		entry.Models = mergeModelLists(entry.Models, getModelsForProvider(providerName))
		entry.Models = mergeModelLists(entry.Models, smartRouterModels[providerName])
		entry.Enabled = status.Enabled
	}

	for providerName, models := range smartRouterModels {
		entry := ensureProvider(providerMap, providerName)
		entry.Models = mergeModelLists(entry.Models, getModelsForProvider(providerName))
		entry.Models = mergeModelLists(entry.Models, models)
		if status, ok := accountStatus[providerName]; ok && status.HasAccount {
			entry.Enabled = status.Enabled
		}
	}

	result := make([]providerConfigView, 0, len(providerMap))
	for _, p := range providerMap {
		if len(p.Models) == 0 {
			p.Models = mergeModelLists(p.Models, getModelsForProvider(p.Name))
		}
		sort.Strings(p.Models)
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	Success(c, gin.H{"providers": result})
}

func mergeModelLists(existing []string, additions []string) []string {
	if len(additions) == 0 {
		return existing
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, model := range existing {
		if normalized := strings.TrimSpace(model); normalized != "" {
			existingSet[normalized] = struct{}{}
		}
	}

	for _, model := range additions {
		normalized := strings.TrimSpace(model)
		if normalized == "" {
			continue
		}
		if _, ok := existingSet[normalized]; ok {
			continue
		}
		existing = append(existing, normalized)
		existingSet[normalized] = struct{}{}
	}

	return existing
}

func inferProviderNameFromAccount(acc *limiter.AccountConfig) string {
	if acc == nil {
		return ""
	}

	providerName := strings.TrimSpace(acc.Provider)
	if providerName == "" {
		providerName = strings.TrimSpace(acc.ProviderType)
	}
	providerName = normalizeProviderName(providerName)

	baseURL := strings.ToLower(strings.TrimSpace(acc.BaseURL))
	switch {
	case strings.Contains(baseURL, "deepseek.com"):
		providerName = "deepseek"
	case strings.Contains(baseURL, "volces.com"), strings.Contains(baseURL, "volcengine.com"):
		providerName = "volcengine"
	case strings.Contains(baseURL, "dashscope.aliyuncs.com"):
		providerName = "qwen"
	case strings.Contains(baseURL, "zhipuai.cn"), strings.Contains(baseURL, "bigmodel.cn"):
		providerName = "zhipu"
	case strings.Contains(baseURL, "moonshot.cn"), strings.Contains(baseURL, "kimi.ai"):
		providerName = "moonshot"
	case strings.Contains(baseURL, "minimax.com"):
		providerName = "minimax"
	case strings.Contains(baseURL, "baichuanai.com"):
		providerName = "baichuan"
	}

	return normalizeProviderName(providerName)
}

func (h *ProxyHandler) getEnabledSmartRouterModelsByProvider() map[string][]string {
	result := make(map[string][]string)
	if h.smartRouter == nil {
		return result
	}

	for modelID, score := range h.smartRouter.GetAllModelScores() {
		if score == nil || !score.Enabled {
			continue
		}

		model := strings.TrimSpace(modelID)
		if model == "" {
			model = strings.TrimSpace(score.Model)
		}
		if model == "" {
			continue
		}

		providerName := normalizeProviderName(score.Provider)
		if providerName == "" {
			providerName = inferProviderFromModel(model)
		}
		if providerName == "" {
			continue
		}

		result[providerName] = mergeModelLists(result[providerName], []string{model})
	}

	return result
}

// ListModels returns available models
func (h *ProxyHandler) ListModels(c *gin.Context) {
	providers := h.registry.ListEnabled()

	modelMap := make(map[string]ModelInfo)
	for _, p := range providers {
		for _, m := range p.Models() {
			if _, exists := modelMap[m]; !exists {
				modelMap[m] = ModelInfo{
					ID:       m,
					Object:   "model",
					Created:  time.Now(),
					OwnedBy:  p.Name(),
					Provider: p.Name(),
					Enabled:  p.IsEnabled(),
				}
			}
		}
	}

	models := make([]ModelInfo, 0, len(modelMap))
	for _, m := range modelMap {
		models = append(models, m)
	}

	Success(c, ModelListResponse{
		Object: "list",
		Data:   models,
	})
}

// recordMetrics records request metrics
func (h *ProxyHandler) recordMetrics(userID, apiKey, model string, latency time.Duration, tokens int, success bool) {
	h.recordMetricsExtended(userID, apiKey, model, "", latency, tokens, success, false, 0, 0, "", "", "", "", "")
}

type usageRuntimeMeta struct {
	Account            string
	UserAgent          string
	RequestType        string
	InferenceIntensity string
}

func buildUsageRuntimeMeta(c *gin.Context, req *ChatCompletionRequest, scheduleResult *ScheduleResult, accountManager *limiter.AccountManager) usageRuntimeMeta {
	meta := usageRuntimeMeta{
		Account:     "-",
		RequestType: "non_stream",
	}

	if req != nil {
		if req.Stream {
			meta.RequestType = "stream"
		}
		reasoningEffort := strings.ToLower(strings.TrimSpace(req.ReasoningEffort))
		switch reasoningEffort {
		case "low", "medium", "high", "xhigh":
			meta.InferenceIntensity = reasoningEffort
		default:
			if req.DeepThink {
				meta.InferenceIntensity = "high"
			}
		}
	}

	if c != nil && c.Request != nil {
		meta.UserAgent = strings.TrimSpace(c.Request.UserAgent())
	}
	if meta.UserAgent == "" {
		meta.UserAgent = "-"
	}

	if c != nil {
		if selectedNameValue, ok := c.Get("selected_account_name"); ok {
			selectedName := strings.TrimSpace(fmt.Sprintf("%v", selectedNameValue))
			if selectedName != "" && selectedName != "<nil>" {
				meta.Account = selectedName
				return meta
			}
		}
		if selectedIDValue, ok := c.Get("selected_account_id"); ok && accountManager != nil {
			selectedID := strings.TrimSpace(fmt.Sprintf("%v", selectedIDValue))
			if selectedID != "" && selectedID != "<nil>" {
				account, err := accountManager.GetAccount(selectedID)
				if err == nil && account != nil {
					if name := strings.TrimSpace(account.Name); name != "" {
						meta.Account = name
						return meta
					}
				}
			}
		}
	}

	if scheduleResult != nil && scheduleResult.Account != nil {
		if name := strings.TrimSpace(scheduleResult.Account.Name); name != "" {
			meta.Account = name
			return meta
		}
	}

	if accountManager == nil {
		return meta
	}

	if c != nil {
		if accountIDValue, ok := c.Get("scheduler_account_id"); ok {
			accountID := strings.TrimSpace(fmt.Sprintf("%v", accountIDValue))
			if accountID != "" && accountID != "<nil>" {
				account, err := accountManager.GetAccount(accountID)
				if err == nil && account != nil {
					if name := strings.TrimSpace(account.Name); name != "" {
						meta.Account = name
						return meta
					}
				}
			}
		}
	}

	providerName := ""
	if req != nil {
		providerName = strings.TrimSpace(req.Provider)
		if providerName == "" {
			providerName = inferProviderFromModel(req.Model)
		}
	}

	if providerName != "" {
		if account := accountManager.GetAccountByProvider(providerName); account != nil {
			if name := strings.TrimSpace(account.Name); name != "" {
				meta.Account = name
			}
		}
	}

	return meta
}

func (h *ProxyHandler) recordMetricsExtended(userID, apiKey, model, provider string, latency time.Duration, tokens int, success bool, cacheHit bool, ttftMs int64, inputTokens int, taskType, difficulty, errorType, experimentTag, domainTag string) {
	h.recordMetricsExtendedWithMetaAndUsageSource(usageRuntimeMeta{}, userID, apiKey, model, provider, latency, tokens, success, cacheHit, ttftMs, inputTokens, 0, "actual", taskType, difficulty, errorType, experimentTag, domainTag)
}

func (h *ProxyHandler) recordMetricsExtendedWithUsageSource(userID, apiKey, model, provider string, latency time.Duration, tokens int, success bool, cacheHit bool, ttftMs int64, inputTokens int, usageSource, taskType, difficulty, errorType, experimentTag, domainTag string) {
	h.recordMetricsExtendedWithMetaAndUsageSource(usageRuntimeMeta{}, userID, apiKey, model, provider, latency, tokens, success, cacheHit, ttftMs, inputTokens, 0, usageSource, taskType, difficulty, errorType, experimentTag, domainTag)
}

func (h *ProxyHandler) recordMetricsExtendedWithMetaAndUsageSource(meta usageRuntimeMeta, userID, apiKey, model, provider string, latency time.Duration, tokens int, success bool, cacheHit bool, ttftMs int64, inputTokens int, cachedReadTokens int, usageSource, taskType, difficulty, errorType, experimentTag, domainTag string) {
	// Update dashboard stats
	if dh := admin.GetDashboardHandler(); dh != nil {
		dh.UpdateStats(success, latency.Milliseconds(), int64(tokens), model)
	}

	// Log to storage for usage tracking
	if storage := storage.GetSQLite(); storage != nil {
		apiKeyDisplay := ""
		if apiKey != "" {
			if keyHandler := admin.GetApiKeyHandler(); keyHandler != nil {
				if name := keyHandler.FindNameByKey(apiKey); name != "" {
					apiKeyDisplay = name
				}
			}
		}
		outputTokens := tokens - inputTokens
		if outputTokens < 0 {
			outputTokens = 0
		}
		if cachedReadTokens < 0 {
			cachedReadTokens = 0
		}
		if err := storage.LogUsage(map[string]interface{}{
			"timestamp":           time.Now().UnixMilli(),
			"model":               model,
			"provider":            provider,
			"account":             strings.TrimSpace(meta.Account),
			"user_id":             userID,
			"api_key":             apiKeyDisplay,
			"user_agent":          strings.TrimSpace(meta.UserAgent),
			"request_type":        strings.TrimSpace(meta.RequestType),
			"inference_intensity": strings.TrimSpace(meta.InferenceIntensity),
			"tokens":              int64(tokens),
			"input_tokens":        int64(inputTokens),
			"output_tokens":       int64(outputTokens),
			"cached_read_tokens":  int64(cachedReadTokens),
			"latency_ms":          latency.Milliseconds(),
			"ttft_ms":             ttftMs,
			"cache_hit":           cacheHit,
			"success":             success,
			"task_type":           taskType,
			"difficulty":          difficulty,
			"error_type":          errorType,
			"experiment_tag":      experimentTag,
			"domain_tag":          domainTag,
			"usage_source":        strings.ToLower(strings.TrimSpace(usageSource)),
		}); err != nil {
			logrus.WithError(err).Debug("failed to persist usage metrics")
		}
	}
}

// maskAPIKey masks an API key for logging
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

type apiKeyValidator interface {
	ValidateAPIKey(apiKey string) bool
}

func resolveAPIKeyFromRequest(c *gin.Context) string {
	return resolveAPIKeyFromRequestWithHandler(c, admin.GetApiKeyHandler())
}

func resolveAPIKeyFromRequestWithHandler(c *gin.Context, keyValidator apiKeyValidator) string {
	if c == nil {
		return ""
	}

	if apiKey := middleware.GetAPIKey(c); apiKey != "" {
		return apiKey
	}

	rawKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
	if rawKey == "" {
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(auth, "Bearer ") {
			auth = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
		rawKey = auth
	}

	if rawKey == "" {
		return ""
	}

	if keyValidator != nil {
		if keyValidator.ValidateAPIKey(rawKey) {
			return rawKey
		}
	}

	return ""
}

// getFloat64 helper
func getFloat64(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}

// getInt helper
func getInt(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

const (
	traceMessagePreviewLimit = 200
	traceMessageFullLimit    = 4000
)

func buildTraceMessageAttributes(prompt string, responseBody []byte) map[string]interface{} {
	aiResponsePreview, aiResponseFull, aiResponseTruncated := tracing.ExtractResponseTextPreview(responseBody, traceMessagePreviewLimit, traceMessageFullLimit)
	userMessagePreview, userMessageFull, userMessageTruncated := buildPromptPreview(prompt)

	return map[string]interface{}{
		"user_message_preview":   userMessagePreview,
		"user_message_full":      userMessageFull,
		"user_message_truncated": userMessageTruncated,
		"ai_response_preview":    aiResponsePreview,
		"ai_response_full":       aiResponseFull,
		"ai_response_truncated":  aiResponseTruncated,
	}
}

func buildTraceErrorMessageAttributes(errorMessage string) map[string]interface{} {
	trimmed := strings.TrimSpace(errorMessage)
	full, truncated := trimTraceTextByRune(trimmed, traceMessageFullLimit)
	preview, _ := trimTraceTextByRune(full, traceMessagePreviewLimit)

	return map[string]interface{}{
		"error_message_preview":   preview,
		"error_message_full":      full,
		"error_message_truncated": truncated,
	}
}

func (h *ProxyHandler) recordHTTPResponseErrorSpan(ctx context.Context, startedAt time.Time, model, providerName, prompt, errorMessage string, statusCode int) {
	if h.traceRecorder == nil {
		return
	}
	if statusCode <= 0 {
		statusCode = http.StatusBadGateway
	}

	attrs := map[string]interface{}{
		"duration_ms": time.Since(startedAt).Milliseconds(),
		"model":       model,
		"status_code": statusCode,
		"success":     false,
		"status":      "error",
		"error":       strings.TrimSpace(errorMessage),
	}
	if trimmedProvider := strings.TrimSpace(providerName); trimmedProvider != "" {
		attrs["provider"] = trimmedProvider
		attrs["provider_name"] = trimmedProvider
	}

	for key, value := range buildPromptPreviewAttributes(prompt) {
		attrs[key] = value
	}
	errorPreviewAttrs := buildTraceErrorMessageAttributes(errorMessage)
	for key, value := range errorPreviewAttrs {
		attrs[key] = value
	}
	attrs["ai_response_preview"] = errorPreviewAttrs["error_message_preview"]
	attrs["ai_response_full"] = errorPreviewAttrs["error_message_full"]
	attrs["ai_response_truncated"] = errorPreviewAttrs["error_message_truncated"]

	h.traceRecorder.RecordSimpleSpan(ctx, "http.response", attrs)
}

func buildPromptPreviewAttributes(prompt string) map[string]interface{} {
	userMessagePreview, userMessageFull, userMessageTruncated := buildPromptPreview(prompt)
	return map[string]interface{}{
		"user_message_preview":   userMessagePreview,
		"user_message_full":      userMessageFull,
		"user_message_truncated": userMessageTruncated,
	}
}

func buildPromptPreview(prompt string) (preview, full string, truncated bool) {
	trimmed := strings.TrimSpace(prompt)
	full, truncated = trimTraceTextByRune(trimmed, traceMessageFullLimit)
	preview, _ = trimTraceTextByRune(full, traceMessagePreviewLimit)
	return preview, full, truncated
}

func trimTraceTextByRune(text string, limit int) (string, bool) {
	if limit <= 0 {
		return "", strings.TrimSpace(text) != ""
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}

	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed, false
	}

	return string(runes[:limit]), true
}

func buildProviderExtraFromChatRequest(req *ChatCompletionRequest) map[string]interface{} {
	if req == nil {
		return nil
	}

	extra := make(map[string]interface{})
	for k, v := range req.Extra {
		extra[k] = v
	}

	if req.TopP != nil {
		extra["top_p"] = *req.TopP
	}
	if req.N != nil {
		extra["n"] = *req.N
	}
	if req.Stop != nil {
		extra["stop"] = req.Stop
	}
	if req.FrequencyPenalty != nil {
		extra["frequency_penalty"] = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		extra["presence_penalty"] = *req.PresencePenalty
	}
	if len(req.LogitBias) > 0 {
		logitBias := make(map[string]interface{}, len(req.LogitBias))
		for key, value := range req.LogitBias {
			logitBias[key] = value
		}
		extra["logit_bias"] = logitBias
	}
	if req.User != "" {
		extra["user"] = req.User
	}

	if req.ReasoningEffort != "" {
		extra["reasoning_effort"] = req.ReasoningEffort
	} else if req.DeepThink {
		extra["reasoning_effort"] = "high"
	}

	if len(extra) == 0 {
		return nil
	}

	return extra
}

func buildResponseCacheKeyPayload(req *ChatCompletionRequest, taskType routing.TaskType, prompt string) interface{} {
	if shouldUsePromptOnlyCache(taskType) {
		return map[string]interface{}{
			"provider":    req.Provider,
			"task_type":   string(taskType),
			"prompt":      strings.TrimSpace(prompt),
			"temperature": getFloat64(req.Temperature, getDefaultTemperature(req.Model)),
			"max_tokens":  getInt(req.MaxTokens, 0),
			"deep_think":  req.DeepThink,
		}
	}
	return req
}

func semanticThresholdForDifficulty(base float64, difficulty routing.DifficultyLevel) float64 {
	if base <= 0 {
		base = 0.92
	}
	switch difficulty {
	case routing.DifficultyLow:
		if base-0.04 < 0.7 {
			return 0.7
		}
		return base - 0.04
	case routing.DifficultyHigh:
		if base+0.03 > 0.98 {
			return 0.98
		}
		return base + 0.03
	default:
		return base
	}
}

func applyControlTTLBand(baseTTL time.Duration, controlCfg routing.ControlConfig, signals *routing.ControlSignals) time.Duration {
	if !controlCfg.Enable || controlCfg.ShadowOnly || !controlCfg.CacheWriteGateEnable || signals == nil {
		return baseTTL
	}
	switch strings.ToLower(strings.TrimSpace(signals.TTLBand)) {
	case "short":
		return time.Hour
	case "medium":
		return 24 * time.Hour
	case "long":
		return 7 * 24 * time.Hour
	default:
		return baseTTL
	}
}

func shouldAllowCacheWrite(controlCfg routing.ControlConfig, signals *routing.ControlSignals) bool {
	if !controlCfg.Enable || controlCfg.ShadowOnly || !controlCfg.CacheWriteGateEnable || signals == nil || signals.Cacheable == nil {
		return true
	}
	return *signals.Cacheable
}

func applyControlToolGate(req *ChatCompletionRequest, controlCfg routing.ControlConfig, assessment *routing.AssessmentResult) {
	if req == nil || !controlCfg.Enable || !controlCfg.ToolGateEnable || assessment == nil || assessment.ControlSignals == nil {
		return
	}
	if assessment.ControlSignals.RAGNeeded != nil && !*assessment.ControlSignals.RAGNeeded && req.DeepThink {
		if controlCfg.ShadowOnly {
			logrus.WithFields(logrus.Fields{
				"task_type":         assessment.TaskType,
				"difficulty":        assessment.Difficulty,
				"assessment_source": assessment.Source,
				"deep_think":        req.DeepThink,
			}).Info("Control RAG gate shadow decision (no mutation)")
		} else {
			logrus.WithFields(logrus.Fields{
				"task_type":         assessment.TaskType,
				"difficulty":        assessment.Difficulty,
				"assessment_source": assessment.Source,
				"deep_think":        req.DeepThink,
			}).Info("Control RAG gate disabled deep think")
			req.DeepThink = false
		}
	}
	if assessment.ControlSignals.ToolNeeded == nil {
		return
	}
	if *assessment.ControlSignals.ToolNeeded {
		return
	}
	if len(req.Tools) == 0 {
		return
	}
	if controlCfg.ShadowOnly {
		logrus.WithFields(logrus.Fields{
			"task_type":         assessment.TaskType,
			"difficulty":        assessment.Difficulty,
			"assessment_source": assessment.Source,
			"tool_count":        len(req.Tools),
		}).Info("Control tool gate shadow decision (no mutation)")
		return
	}

	logrus.WithFields(logrus.Fields{
		"task_type":         assessment.TaskType,
		"difficulty":        assessment.Difficulty,
		"assessment_source": assessment.Source,
		"tool_count":        len(req.Tools),
	}).Info("Control tool gate disabled tool calls")

	req.Tools = nil
	req.ToolChoice = nil
}

func applyControlGenerationHints(req *ChatCompletionRequest, controlCfg routing.ControlConfig, assessment *routing.AssessmentResult) {
	if req == nil || !controlCfg.Enable || !controlCfg.ParameterHintEnable || assessment == nil || assessment.ControlSignals == nil {
		return
	}
	if controlCfg.ShadowOnly {
		logrus.WithFields(logrus.Fields{
			"has_temperature": assessment.ControlSignals.RecommendedTemperature != nil,
			"has_top_p":       assessment.ControlSignals.RecommendedTopP != nil,
			"has_max_tokens":  assessment.ControlSignals.RecommendedMaxTokens != nil,
		}).Info("Control generation hint shadow decision (no mutation)")
		return
	}

	if req.Temperature == nil && assessment.ControlSignals.RecommendedTemperature != nil {
		v := *assessment.ControlSignals.RecommendedTemperature
		req.Temperature = &v
	}
	if req.TopP == nil && assessment.ControlSignals.RecommendedTopP != nil {
		v := *assessment.ControlSignals.RecommendedTopP
		req.TopP = &v
	}
	if req.MaxTokens == nil && assessment.ControlSignals.RecommendedMaxTokens != nil {
		v := *assessment.ControlSignals.RecommendedMaxTokens
		req.MaxTokens = &v
	}
}

func logControlRiskSignals(assessment *routing.AssessmentResult) {
	if assessment == nil || assessment.ControlSignals == nil {
		return
	}
	riskLevel := strings.TrimSpace(strings.ToLower(assessment.ControlSignals.RiskLevel))
	if riskLevel == "" && len(assessment.ControlSignals.RiskTags) == 0 {
		return
	}
	fields := logrus.Fields{
		"task_type":         assessment.TaskType,
		"difficulty":        assessment.Difficulty,
		"risk_level":        riskLevel,
		"risk_tags":         assessment.ControlSignals.RiskTags,
		"fallback_reason":   assessment.FallbackReason,
		"assessment_source": assessment.Source,
	}
	if riskLevel == "high" {
		logrus.WithFields(fields).Warn("Control risk signal detected")
		return
	}
	logrus.WithFields(fields).Info("Control risk signal observed")
}

func shouldBlockByRisk(controlCfg routing.ControlConfig, assessment *routing.AssessmentResult) bool {
	if !controlCfg.Enable || !controlCfg.RiskTagEnable || !controlCfg.RiskBlockEnable || assessment == nil || assessment.ControlSignals == nil {
		return false
	}
	riskLevel := strings.TrimSpace(strings.ToLower(assessment.ControlSignals.RiskLevel))
	if riskLevel != "high" {
		return false
	}

	fields := logrus.Fields{
		"task_type":         assessment.TaskType,
		"difficulty":        assessment.Difficulty,
		"risk_level":        riskLevel,
		"risk_tags":         assessment.ControlSignals.RiskTags,
		"assessment_source": assessment.Source,
	}

	if controlCfg.ShadowOnly {
		logrus.WithFields(fields).Warn("Control risk block shadow decision (no mutation)")
		return false
	}

	logrus.WithFields(fields).Warn("Control risk block enforced")
	return true
}

func logControlRoutingSignals(assessment *routing.AssessmentResult) {
	if assessment == nil || assessment.ControlSignals == nil {
		return
	}
	contextLoad := strings.TrimSpace(strings.ToLower(assessment.ControlSignals.ContextLoad))
	if contextLoad == "" && len(assessment.ControlSignals.ModelFit) == 0 {
		return
	}
	logrus.WithFields(logrus.Fields{
		"task_type":         assessment.TaskType,
		"difficulty":        assessment.Difficulty,
		"context_load":      contextLoad,
		"model_fit_count":   len(assessment.ControlSignals.ModelFit),
		"experiment_tag":    assessment.ControlSignals.ExperimentTag,
		"domain_tag":        assessment.ControlSignals.DomainTag,
		"assessment_source": assessment.Source,
	}).Info("Control routing signal observed")
}

func buildControlHeaders(controlCfg routing.ControlConfig, assessment *routing.AssessmentResult) map[string]string {
	headers := map[string]string{}
	if !controlCfg.Enable || assessment == nil || assessment.ControlSignals == nil {
		return headers
	}
	if v := strings.TrimSpace(assessment.ControlSignals.ExperimentTag); v != "" {
		headers["X-Control-Experiment"] = v
	}
	if v := strings.TrimSpace(assessment.ControlSignals.DomainTag); v != "" {
		headers["X-Control-Domain"] = v
	}
	return headers
}

func buildSemanticQueryCandidates(normalizedEnabled bool, normalizedQuery, semanticSignature, prompt string) []string {
	candidates := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	appendUnique := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		candidates = append(candidates, v)
	}

	// Prefer raw prompt first to reduce over-broad hits from generic signatures.
	appendUnique(prompt)
	if normalizedEnabled {
		appendUnique(normalizedQuery)
	}
	if len(candidates) == 0 {
		appendUnique(semanticSignature)
	}

	return candidates
}

func buildSemanticCacheWriteQuery(prompt, normalizedQuery, semanticSignature string, normalizedEnabled bool) string {
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		return prompt
	}

	if normalizedEnabled {
		normalizedQuery = strings.TrimSpace(normalizedQuery)
		if normalizedQuery != "" {
			return normalizedQuery
		}
	}

	return strings.TrimSpace(semanticSignature)
}

func shouldAllowSemanticCache(taskType routing.TaskType) bool {
	switch taskType {
	case routing.TaskTypeChat, routing.TaskTypeCreative, routing.TaskTypeUnknown:
		return false
	default:
		return true
	}
}

func shouldUsePromptOnlyCache(taskType routing.TaskType) bool {
	switch taskType {
	case routing.TaskTypeMath, routing.TaskTypeFact, routing.TaskTypeTranslate:
		return true
	default:
		return false
	}
}

func responseCacheModelDimension(taskType routing.TaskType, model string) string {
	if shouldUsePromptOnlyCache(taskType) {
		return "provider-scope"
	}
	return model
}

type usageTokens struct {
	Prompt     int
	Completion int
	Total      int
	CachedRead int
}

func estimateTokensByText(text string) int {
	if text == "" {
		return 0
	}

	asciiCount := 0
	nonASCIICount := 0
	for _, r := range text {
		if r <= unicode.MaxASCII {
			asciiCount++
			continue
		}
		nonASCIICount++
	}

	estimate := (float64(asciiCount) / 4.0) + (float64(nonASCIICount) / 1.5)
	if estimate <= 0 {
		return 0
	}
	return int(math.Ceil(estimate))
}

func buildCachedUsage(promptTokens, completionTokens, cachedReadTokens, totalTokens int) map[string]int {
	if promptTokens < 0 {
		promptTokens = 0
	}
	if completionTokens < 0 {
		completionTokens = 0
	}
	if cachedReadTokens < 0 {
		cachedReadTokens = 0
	}
	if totalTokens <= 0 {
		totalTokens = promptTokens + completionTokens
	}
	return map[string]int{
		"prompt_tokens":      promptTokens,
		"completion_tokens":  completionTokens,
		"cached_read_tokens": cachedReadTokens,
		"total_tokens":       totalTokens,
	}
}

func resolveUsageWithFallback(promptText, outputText string, provided usageTokens) (usageTokens, string) {
	provided.Prompt = max(0, provided.Prompt)
	provided.Completion = max(0, provided.Completion)
	provided.Total = max(0, provided.Total)
	provided.CachedRead = max(0, provided.CachedRead)

	if provided.Total <= 0 && (provided.Prompt > 0 || provided.Completion > 0) {
		provided.Total = provided.Prompt + provided.Completion
	}

	if provided.Total > 0 {
		if provided.Prompt <= 0 {
			provided.Prompt = estimateTokensByText(promptText)
		}
		if provided.Completion <= 0 {
			derived := provided.Total - provided.Prompt
			if derived > 0 {
				provided.Completion = derived
			} else {
				provided.Completion = estimateTokensByText(outputText)
			}
		}
		if provided.Total < provided.Prompt+provided.Completion {
			provided.Total = provided.Prompt + provided.Completion
		}
		return provided, "actual"
	}

	estimatedPrompt := estimateTokensByText(promptText)
	estimatedCompletion := estimateTokensByText(outputText)
	estimatedTotal := estimatedPrompt + estimatedCompletion
	return usageTokens{
		Prompt:     estimatedPrompt,
		Completion: estimatedCompletion,
		Total:      estimatedTotal,
		CachedRead: provided.CachedRead,
	}, "estimated"
}

func extractUsageTokensFromBody(body []byte) usageTokens {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return usageTokens{}
	}
	usage, ok := payload["usage"].(map[string]interface{})
	if !ok {
		return usageTokens{}
	}
	tokens := usageTokens{
		Prompt:     extractTokenInt(usage["prompt_tokens"]),
		Completion: extractTokenInt(usage["completion_tokens"]),
		Total:      extractTokenInt(usage["total_tokens"]),
		CachedRead: extractTokenInt(usage["cached_read_tokens"]),
	}
	if tokens.Prompt <= 0 {
		tokens.Prompt = extractTokenInt(usage["input_tokens"])
	}
	if tokens.Completion <= 0 {
		tokens.Completion = extractTokenInt(usage["output_tokens"])
	}
	if tokens.CachedRead <= 0 {
		tokens.CachedRead = extractNestedTokenInt(usage, "prompt_tokens_details", "cached_tokens")
	}
	if tokens.CachedRead <= 0 {
		tokens.CachedRead = extractNestedTokenInt(usage, "input_tokens_details", "cached_tokens")
	}
	if tokens.Total <= 0 {
		tokens.Total = tokens.Prompt + tokens.Completion
	}
	return tokens
}

func extractNestedTokenInt(root map[string]interface{}, parentKey, tokenKey string) int {
	parent, ok := root[parentKey].(map[string]interface{})
	if !ok {
		return 0
	}
	return extractTokenInt(parent[tokenKey])
}

func extractAssistantTextFromBody(body []byte) string {
	var payload struct {
		Choices []struct {
			Message *struct {
				Content interface{} `json:"content"`
			} `json:"message"`
			Delta *struct {
				Content interface{} `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if len(payload.Choices) == 0 {
		return ""
	}
	if payload.Choices[0].Message != nil {
		return getTextContent(payload.Choices[0].Message.Content)
	}
	if payload.Choices[0].Delta != nil {
		return getTextContent(payload.Choices[0].Delta.Content)
	}
	return ""
}

func extractAssistantTextFromProviderChoices(choices []provider.Choice) string {
	if len(choices) == 0 {
		return ""
	}
	return getTextContent(choices[0].Message.Content)
}

func extractTokenInt(raw interface{}) int {
	switch v := raw.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0
		}
		return int(n)
	default:
		return 0
	}
}

func hasMeaningfulAssistantResponse(resp *ChatCompletionResponse) bool {
	if resp == nil || len(resp.Choices) == 0 {
		return false
	}
	msg := resp.Choices[0].Message
	if msg == nil {
		return false
	}
	if strings.TrimSpace(getTextContent(msg.Content)) != "" {
		return true
	}
	return len(msg.ToolCalls) > 0
}

func hasMeaningfulCachedResponse(body []byte) bool {
	var cachedResp ChatCompletionResponse
	if err := json.Unmarshal(body, &cachedResp); err != nil {
		return true
	}
	return hasMeaningfulAssistantResponse(&cachedResp)
}

func (h *ProxyHandler) writeCachedResponseAsStream(c *gin.Context, model string, body []byte) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	var cachedResp ChatCompletionResponse
	if err := json.Unmarshal(body, &cachedResp); err != nil {
		c.Data(http.StatusOK, "application/json", body)
		return
	}

	content := ""
	if len(cachedResp.Choices) > 0 && cachedResp.Choices[0].Message != nil {
		content = getTextContent(cachedResp.Choices[0].Message.Content)
	}

	created := cachedResp.Created
	if created == 0 {
		created = time.Now().Unix()
	}
	streamModel := cachedResp.Model
	if streamModel == "" {
		streamModel = model
	}

	id := cachedResp.ID
	if id == "" {
		id = fmt.Sprintf("chatcmpl-cache-%d", time.Now().UnixNano())
	}

	chunk := StreamingResponse{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   streamModel,
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: &ChatMessage{Role: "assistant", Content: content},
			},
		},
	}
	c.SSEvent("message", chunk)
	c.Writer.Flush()

	finishReason := "stop"
	finalChunk := StreamingResponse{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   streamModel,
		Choices: []StreamChoice{
			{
				Index:        0,
				Delta:        &ChatMessage{},
				FinishReason: &finishReason,
			},
		},
		Usage: &Usage{
			PromptTokens:     cachedResp.Usage.PromptTokens,
			CompletionTokens: cachedResp.Usage.CompletionTokens,
			TotalTokens:      cachedResp.Usage.TotalTokens,
			CachedReadTokens: cachedResp.Usage.CachedReadTokens,
		},
	}
	c.SSEvent("message", finalChunk)
	c.Writer.Flush()

	c.SSEvent("message", "[DONE]")
	c.Writer.Flush()
}

func (h *ProxyHandler) pruneDuplicateResponseEntries(ctx context.Context, providerName, model, taskType, prompt, keepKey string) {
	if h.cache == nil {
		return
	}
	entries := h.cache.ListEntries("response", "")
	if len(entries) == 0 {
		return
	}

	normalizedPrompt := strings.TrimSpace(prompt)
	normalizedTaskType := strings.ToLower(strings.TrimSpace(taskType))
	providerScope := shouldUsePromptOnlyCache(routing.TaskType(taskType))

	for _, entry := range entries {
		if entry == nil || entry.Key == keepKey {
			continue
		}
		if providerName != "" && entry.Provider != "" && entry.Provider != providerName {
			continue
		}
		if !providerScope && model != "" && entry.Model != "" && entry.Model != model {
			continue
		}

		detail, err := h.cache.GetEntryDetail(ctx, entry.Key)
		if err != nil {
			continue
		}
		cachedPrompt, cachedTaskType := extractPromptAndTaskTypeFromCacheValue(detail.Value)
		if strings.TrimSpace(cachedPrompt) != normalizedPrompt {
			continue
		}
		if strings.ToLower(strings.TrimSpace(cachedTaskType)) != normalizedTaskType {
			continue
		}

		if err := h.cache.Cache().Delete(ctx, entry.Key); err != nil {
			logrus.WithError(err).WithField("cache_key", entry.Key).Debug("failed to prune duplicate cache entry")
		}
	}
}

func extractPromptAndTaskTypeFromCacheValue(value interface{}) (string, string) {
	switch v := value.(type) {
	case map[string]interface{}:
		prompt, ok := v["prompt"].(string)
		if !ok {
			prompt = ""
		}
		taskType, ok := v["task_type"].(string)
		if !ok {
			taskType = ""
		}
		if taskType == "" {
			if fallbackTaskType, fallbackOK := v["TaskType"].(string); fallbackOK {
				taskType = fallbackTaskType
			}
		}
		return prompt, taskType
	case map[interface{}]interface{}:
		converted := make(map[string]interface{}, len(v))
		for key, val := range v {
			if ks, ok := key.(string); ok {
				converted[ks] = val
			}
		}
		return extractPromptAndTaskTypeFromCacheValue(converted)
	default:
		return "", ""
	}
}

func (h *ProxyHandler) persistResponseCacheHit(ctx context.Context, cacheKey string, cached *cache.CachedResponse, requestedModel string) {
	if h.cache == nil || h.cache.ResponseCache == nil || cached == nil {
		return
	}

	cached.HitCount++
	if requestedModel != "" {
		if cached.HitModels == nil {
			cached.HitModels = map[string]int64{}
		}
		cached.HitModels[requestedModel]++
	}

	if rc, ok := h.cache.Cache().(*cache.RedisCache); ok {
		ttl, err := rc.TTL(ctx, cacheKey)
		if err != nil || ttl <= 0 {
			return
		}
		if setErr := h.cache.ResponseCache.SetWithTTL(ctx, cacheKey, cached, ttl); setErr != nil {
			logrus.WithError(setErr).WithField("cache_key", cacheKey).Debug("failed to persist cache hit metadata")
		}
	}
}

func (h *ProxyHandler) writeResponseCacheEntry(
	ctx context.Context,
	cacheKey string,
	cachedResp *cache.CachedResponse,
	ttl time.Duration,
	model string,
	provider string,
	taskType string,
	taskTypeSource string,
) error {
	if h.cache == nil || h.cache.ResponseCache == nil || cachedResp == nil {
		return nil
	}
	if mc, ok := h.cache.Cache().(*cache.MemoryCache); ok {
		if err := mc.SetWithTaskType(ctx, cacheKey, cachedResp, ttl, model, provider, taskType, taskTypeSource); err != nil {
			return err
		}
	} else {
		if err := h.cache.ResponseCache.SetWithTTL(ctx, cacheKey, cachedResp, ttl); err != nil {
			return err
		}
	}
	if archiveService := h.cache.GetResponseColdArchiveService(); archiveService != nil {
		archiveService.NotifyWrite(cacheKey, cachedResp, ttl)
	}
	return nil
}

func (h *ProxyHandler) processCacheV2Read(
	ctx context.Context,
	prompt string,
	normalizedQuery string,
	taskType string,
	cacheSettings cache.CacheSettings,
) (*intent.EmbeddingResult, []byte, bool, string, string) {
	if h.vectorStore == nil || !cacheSettings.VectorEnabled {
		return nil, nil, false, "", ""
	}
	pipeline := h.vectorPipeline
	if pipeline == nil {
		settingsCopy := cacheSettings
		pipeline = NewVectorPipeline(h.vectorStore, h.textNormalizer, func() cache.CacheSettings {
			return settingsCopy
		})
	}
	return pipeline.Read(
		ctx,
		prompt,
		normalizedQuery,
		taskType,
		&cacheSettings,
		h.intentThreshold,
	)
}

func (h *ProxyHandler) processCacheV2Write(
	ctx context.Context,
	prompt string,
	normalizedQuery string,
	intentResult *intent.EmbeddingResult,
	providerName string,
	model string,
	taskType routing.TaskType,
	response ChatCompletionResponse,
) {
	if h.vectorStore == nil {
		return
	}

	settings := cache.DefaultCacheSettings()
	if h.cache != nil {
		settings = h.cache.GetSettings()
	}
	pipeline := h.vectorPipeline
	if pipeline == nil {
		settingsCopy := settings
		pipeline = NewVectorPipeline(h.vectorStore, h.textNormalizer, func() cache.CacheSettings {
			return settingsCopy
		})
	}
	if intentResult == nil {
		rebuilt, err := pipeline.buildIntentEmbeddingResult(ctx, prompt, normalizedQuery, string(taskType), settings)
		if err != nil || rebuilt == nil {
			return
		}
		intentResult = rebuilt
	}
	pipeline.Write(
		ctx,
		intentResult,
		providerName,
		model,
		taskType,
		response,
		h.intentTTLSeconds(intentResult),
		&settings,
	)
}

func (h *ProxyHandler) intentThreshold(intentName string, cacheSettings cache.CacheSettings) float64 {
	intentName = normalizeVectorIntentPolicyKey(intentName)
	if cacheSettings.VectorThresholds != nil {
		if v, ok := cacheSettings.VectorThresholds[intentName]; ok && v > 0 && v <= 1 {
			return v
		}
	}
	if cacheSettings.SimilarityThreshold > 0 && cacheSettings.SimilarityThreshold <= 1 {
		return cacheSettings.SimilarityThreshold
	}
	return 0.92
}

func (h *ProxyHandler) intentTTLSeconds(intentResult *intent.EmbeddingResult) int64 {
	defaultTTL := int64((24 * time.Hour).Seconds())
	if h.config == nil {
		return defaultTTL
	}

	intentName := normalizeVectorIntentPolicyKey(intentResult.Intent)
	if h.config.VectorCache.TTLSeconds != nil {
		if ttl, ok := h.config.VectorCache.TTLSeconds[intentName]; ok && ttl > 0 {
			return ttl
		}
	}
	return defaultTTL
}

func normalizeVectorIntentPolicyKey(intentName string) string {
	normalized := strings.ToLower(strings.TrimSpace(intentName))
	switch normalized {
	case "math":
		return "calc"
	case "fact", "reasoning", "code", "long_text":
		return "qa"
	default:
		return normalized
	}
}

// Helper to check context cancellation
func isContextCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// getDefaultTemperature returns the default temperature for a model
// Some models like kimi-k2.5 only accept temperature=1
func getDefaultTemperature(model string) float64 {
	// Models that only accept temperature=1
	strictTempModels := map[string]bool{
		"kimi-k2.5":            true,
		"kimi-k2.5-preview":    true,
		"kimi-k2-0905-preview": true,
	}

	if strictTempModels[model] {
		return 1.0
	}

	// Default temperature
	return 0.7
}

// getTextContent extracts text from Content which can be string or []interface{} (multimodal)
func getTextContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

func sanitizeChatMessagesForIntentAndUpstream(messages []ChatMessage) []ChatMessage {
	if len(messages) == 0 {
		return messages
	}

	sanitized := make([]ChatMessage, len(messages))
	for i, msg := range messages {
		sanitized[i] = msg
		sanitized[i].Content = sanitizeChatMessageContent(msg.Content)
	}
	return sanitized
}

func sanitizeChatMessageContent(content interface{}) interface{} {
	switch v := content.(type) {
	case string:
		return routing.SanitizeIntentInput(v)
	case []interface{}:
		parts := make([]interface{}, len(v))
		for i, item := range v {
			m, ok := item.(map[string]interface{})
			if !ok {
				parts[i] = item
				continue
			}

			copied := make(map[string]interface{}, len(m))
			for key, value := range m {
				copied[key] = value
			}

			if copied["type"] == "text" {
				if text, ok := copied["text"].(string); ok {
					copied["text"] = routing.SanitizeIntentInput(text)
				}
			}

			parts[i] = copied
		}
		return parts
	default:
		return content
	}
}

// GetModelMappingCache returns the model mapping cache for admin handlers
func (h *ProxyHandler) GetModelMappingCache() *cache.ModelMappingCache {
	return h.modelMappingCache
}

// ScheduleContext contains context for account scheduling
type ScheduleContext struct {
	SessionHash        string
	PreviousResponseID string
	ProviderType       string
	Model              string
}

// extractScheduleContext extracts scheduling context from request
func extractScheduleContext(c *gin.Context, model, providerName string) ScheduleContext {
	return ScheduleContext{
		SessionHash:        c.GetHeader("X-Session-Hash"),
		PreviousResponseID: c.GetHeader("X-Previous-Response-ID"),
		ProviderType:       providerName,
		Model:              model,
	}
}

// ScheduleResult contains the result of account scheduling
type ScheduleResult struct {
	Account     *limiter.AccountConfig
	Decision    limiter.ScheduleDecision
	ReleaseFunc func()
}

// getProviderWithScheduler gets provider using the three-layer scheduling strategy
func (h *ProxyHandler) getProviderWithScheduler(ctx context.Context, scheduleCtx ScheduleContext) (*provider.Provider, *ScheduleResult, error) {
	if h.accountManager == nil {
		return nil, nil, &provider.ProviderError{
			Message:   "Account manager not available",
			Code:      http.StatusServiceUnavailable,
			Retryable: false,
		}
	}

	// Build schedule request
	req := limiter.ScheduleRequest{
		ProviderType:       scheduleCtx.ProviderType,
		Model:              scheduleCtx.Model,
		SessionHash:        scheduleCtx.SessionHash,
		PreviousResponseID: scheduleCtx.PreviousResponseID,
	}

	// Select account using scheduler
	account, decision, releaseFunc, err := h.accountManager.SelectAccount(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	if account == nil {
		return nil, nil, &provider.ProviderError{
			Message:   "No available account for provider: " + scheduleCtx.ProviderType,
			Code:      http.StatusServiceUnavailable,
			Retryable: false,
		}
	}

	// Map provider type to backend provider
	backendProvider := mapProviderName(account.Provider)
	if account.ProviderType != "" {
		backendProvider = account.ProviderType
	}

	// Create provider config
	provConfig := &provider.ProviderConfig{
		Name:    backendProvider,
		APIKey:  account.APIKey,
		BaseURL: account.BaseURL,
		Models:  getModelsForProvider(account.Provider),
		Enabled: true,
	}

	// Create provider
	p, err := h.registry.CreateProvider(provConfig)
	if err != nil {
		if releaseFunc != nil {
			releaseFunc()
		}
		return nil, nil, err
	}

	result := &ScheduleResult{
		Account:     account,
		Decision:    decision,
		ReleaseFunc: releaseFunc,
	}

	return &p, result, nil
}

// reportScheduleResult reports the result of a scheduled request
func (h *ProxyHandler) reportScheduleResult(accountID string, success bool, ttftMs int64) {
	if h.accountManager == nil {
		return
	}
	h.accountManager.ReportScheduleResult(accountID, success, ttftMs)
}

// bindResponseToAccount binds a response ID to the account that handled it
func (h *ProxyHandler) bindResponseToAccount(ctx context.Context, responseID, accountID string) {
	if h.accountManager == nil {
		return
	}
	if err := h.accountManager.BindResponseToAccount(ctx, responseID, accountID); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{"response_id": responseID, "account_id": accountID}).Debug("failed to bind response to account")
	}
}
