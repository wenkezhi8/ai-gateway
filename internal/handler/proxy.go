package handler

import (
	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/handler/admin"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/middleware"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

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
		"o1", "o1-mini", "o1-preview",
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

// Global smart router instance
var (
	globalSmartRouter     *routing.SmartRouter
	globalSmartRouterOnce sync.Once
)

// GetSmartRouter returns the global smart router instance
func GetSmartRouter() *routing.SmartRouter {
	globalSmartRouterOnce.Do(func() {
		globalSmartRouter = routing.NewSmartRouter()
	})
	return globalSmartRouter
}

// ProxyHandler handles AI provider proxy requests
type ProxyHandler struct {
	config         *config.Config
	registry       *provider.Registry
	accountManager *limiter.AccountManager
	smartRouter    *routing.SmartRouter
	cache          *cache.Manager
	deduplicator   *cache.RequestDeduplicator
	semanticCache  *cache.SemanticCache
	embeddingSvc   cache.EmbeddingProvider
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

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			semanticCache.Cleanup()
		}
	}()

	return &ProxyHandler{
		config:         cfg,
		registry:       provider.GetRegistry(),
		accountManager: accountManager,
		smartRouter:    GetSmartRouter(),
		cache:          cacheManager,
		deduplicator:   cache.GetRequestDeduplicator(),
		semanticCache:  semanticCache,
	}
}

// ChatCompletions proxies chat completion requests to AI providers
// 改动点: 集成请求去重、难度评估、缓存策略
func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
	startTime := time.Now()

	// Limit request body size to prevent DoS attacks
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodySize)

	// Parse request
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if err.Error() == "http: request body too large" {
			Error(c, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds maximum size of 10MB")
			return
		}
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// Get user context
	userID := middleware.GetUserID(c)
	apiKey := middleware.GetAPIKey(c)

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

	// Assess difficulty and get recommended TTL
	assessment := h.smartRouter.AssessDifficulty(prompt, contextStr)
	recommendedTTL := assessment.SuggestedTTL

	logrus.WithFields(logrus.Fields{
		"task_type":       assessment.TaskType,
		"difficulty":      assessment.Difficulty,
		"recommended_ttl": recommendedTTL,
	}).Debug("Request difficulty assessment")

	// Handle "auto", "latest", and "default" model selection
	requestedModel := req.Model
	if req.Model == "auto" || req.Model == "latest" || req.Model == "default" {
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

	// Try semantic cache first (for non-streaming requests with cacheable task types)
	if !req.Stream && h.semanticCache != nil && assessment.SuggestedTTL > 0 {
		// Use simple embedding for semantic matching (no API call needed)
		queryVector := cache.SimpleEmbedding(prompt, 1536)

		semanticEntry, similarity := h.semanticCache.Get(c.Request.Context(), prompt, queryVector)
		if semanticEntry != nil && similarity >= 0.92 {
			logrus.WithFields(logrus.Fields{
				"model":      req.Model,
				"similarity": similarity,
				"task_type":  assessment.TaskType,
				"cache_id":   semanticEntry.ID,
			}).Info("Semantic cache hit")

			h.semanticCache.IncrementHitCount(semanticEntry.ID)
			h.recordMetrics(userID, apiKey, req.Model, time.Since(startTime), 0, true)
			c.Data(http.StatusOK, "application/json", semanticEntry.Response)
			return
		}
	}

	// Try exact cache (for non-streaming requests)
	if !req.Stream && h.cache != nil && h.cache.ResponseCache != nil && recommendedTTL > 0 {
		cacheKey, _ := h.cache.ResponseCache.GenerateKey(req.Provider, req.Model, req)
		cached, err := h.cache.ResponseCache.Get(c.Request.Context(), cacheKey)
		if err == nil && cached != nil {
			logrus.WithFields(logrus.Fields{
				"model":     req.Model,
				"task_type": assessment.TaskType,
			}).Info("Response cache hit")

			h.recordMetrics(userID, apiKey, req.Model, time.Since(startTime), 0, true)
			c.Data(cached.StatusCode, "application/json", cached.Body)
			return
		}
	}

	// Get provider for the request - try account manager first
	targetProvider, err := h.getProviderForRequest(requestedModel, req.Provider)
	if err != nil {
		h.recordMetrics("", "", req.Model, time.Since(startTime), 0, false)
		Error(c, http.StatusServiceUnavailable, ErrCodeProviderError, err.Error())
		return
	}

	// Get default temperature based on model
	defaultTemp := getDefaultTemperature(req.Model)

	// Convert to provider request format
	providerReq := &provider.ChatRequest{
		Model:       req.Model,
		Temperature: getFloat64(req.Temperature, defaultTemp),
		MaxTokens:   getInt(req.MaxTokens, 0),
		Stream:      req.Stream,
		Extra:       req.Extra,
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

	// Handle streaming request
	if req.Stream {
		h.handleStreamResponse(c, targetProvider, providerReq)
		return
	}

	// Use deduplicator for non-streaming requests
	// 改动点: 使用请求去重避免重复计算
	dedupKey := h.deduplicator.GenerateKey(prompt, req.Model, map[string]interface{}{
		"temperature": providerReq.Temperature,
		"max_tokens":  providerReq.MaxTokens,
	})

	result, err := h.deduplicator.Do(c.Request.Context(), dedupKey, func() (interface{}, error) {
		return targetProvider.Chat(c.Request.Context(), providerReq)
	})

	if err != nil {
		h.recordMetrics(userID, apiKey, req.Model, time.Since(startTime), 0, false)
		logMsg := fmt.Sprintf("Provider request failed: %v", err)
		if providerErr, ok := err.(*provider.ProviderError); ok {
			logMsg = fmt.Sprintf("Provider request failed [%s]: %s (code: %d, retryable: %v)",
				providerErr.Provider, providerErr.Message, providerErr.Code, providerErr.Retryable)
		}
		ProviderError(c, "Provider request failed", logMsg)
		return
	}

	resp := result.(*provider.ChatResponse)

	// Check for provider error in response
	if resp.Error != nil {
		h.recordMetrics(userID, apiKey, req.Model, time.Since(startTime), 0, false)
		ProviderError(c, resp.Error.Message, resp.Error.Type)
		return
	}

	// Update model success rate
	h.smartRouter.UpdateModelSuccessRate(req.Model, assessment.TaskType, true)

	// Record metrics
	latency := time.Since(startTime)
	h.recordMetrics(userID, apiKey, req.Model, latency, resp.Usage.TotalTokens, true)

	// Build response
	response := ChatCompletionResponse{
		ID:                resp.ID,
		Object:            resp.Object,
		Created:           resp.Created,
		Model:             resp.Model,
		SystemFingerprint: "",
		Choices:           convertChoices(resp.Choices),
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Cache the response if applicable
	if h.cache != nil && h.cache.ResponseCache != nil && recommendedTTL > 0 {
		responseBody, _ := json.Marshal(response)
		cacheKey, _ := h.cache.ResponseCache.GenerateKey(req.Provider, req.Model, req)
		cachedResp := &cache.CachedResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       responseBody,
			CreatedAt:  time.Now(),
			Provider:   req.Provider,
			Model:      req.Model,
		}
		h.cache.ResponseCache.Set(c.Request.Context(), cacheKey, cachedResp)
		logrus.WithFields(logrus.Fields{
			"model": req.Model,
			"ttl":   recommendedTTL,
		}).Debug("Response cached")
	}

	// Store in semantic cache for similar query matching
	// 改动点: 存储到语义缓存供相似请求复用
	if h.semanticCache != nil && recommendedTTL > 0 && assessment.TaskType != routing.TaskTypeCreative {
		responseBody, _ := json.Marshal(response)
		queryVector := cache.SimpleEmbedding(prompt, 1536)
		h.semanticCache.Set(
			c.Request.Context(),
			prompt,
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

	// Return response in OpenAI-compatible format
	c.JSON(http.StatusOK, response)
}

// getProviderForRequest gets the appropriate provider for the request
// First tries to use account manager to get active account credentials
// Falls back to registry providers if no account manager or no active account
func (h *ProxyHandler) getProviderForRequest(model string, providerName string) (provider.Provider, error) {
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

	// Try to get provider from account manager
	if h.accountManager != nil && providerName != "" {
		// Map frontend provider to backend provider type
		backendProvider := mapProviderName(providerName)

		// First try to get account by provider name directly (handles coding plan endpoints)
		account := h.accountManager.GetAccountByProvider(providerName)

		if account != nil && account.Enabled {
			logrus.WithFields(logrus.Fields{
				"provider": providerName,
				"backend":  backendProvider,
				"base_url": account.BaseURL,
				"enabled":  account.Enabled,
			}).Info("Found account by provider name")

			provConfig := &provider.ProviderConfig{
				Name:    backendProvider,
				APIKey:  account.APIKey,
				BaseURL: account.BaseURL,
				Models:  getModelsForProvider(providerName),
				Enabled: true,
			}
			p, err := h.registry.CreateProvider(provConfig)
			if err == nil {
				// Cache the route decision
				if h.cache != nil && h.cache.RouteCache != nil {
					cacheKey := model + ":" + providerName
					h.cache.RouteCache.Set(context.Background(), cacheKey, nil, &cache.RouteDecision{
						Provider: providerName,
						Model:    model,
					})
				}
				return p, nil
			}
			logrus.WithError(err).Warn("Failed to create provider from account")
		} else {
			logrus.WithField("provider", providerName).Info("No account found by provider name")
		}

		// Get base URL for this provider
		baseURL := getBaseURLForProvider(providerName)

		// Try both the original provider name and the mapped backend provider type
		// because accounts may be registered with either key
		providerKeys := []string{providerName, backendProvider}

		for _, pk := range providerKeys {
			// First try to get by base URL and provider type (more specific match)
			if baseURL != "" {
				account := h.accountManager.GetAccountByBaseURLAndType(baseURL, pk)
				if account != nil && account.Enabled {
					provConfig := &provider.ProviderConfig{
						Name:    backendProvider,
						APIKey:  account.APIKey,
						BaseURL: account.BaseURL,
						Models:  getModelsForProvider(providerName),
						Enabled: true,
					}
					p, err := h.registry.CreateProvider(provConfig)
					if err == nil {
						return p, nil
					}
				}
			}

			// Fallback: try to get active account by provider type
			account, _ := h.accountManager.GetActiveAccount(pk)
			if account != nil && account.Enabled {
				provConfig := &provider.ProviderConfig{
					Name:    backendProvider,
					APIKey:  account.APIKey,
					BaseURL: account.BaseURL,
					Models:  getModelsForProvider(providerName),
					Enabled: true,
				}
				p, err := h.registry.CreateProvider(provConfig)
				if err == nil {
					return p, nil
				}
			}
		}
	}

	// Check if any accounts exist for this provider
	if h.accountManager != nil && providerName != "" {
		hasAccounts := false
		hasEnabledAccounts := false
		for _, acc := range h.accountManager.GetAllAccounts() {
			if acc.Provider == providerName || acc.Provider == mapProviderName(providerName) {
				hasAccounts = true
				if acc.Enabled {
					hasEnabledAccounts = true
					break
				}
			}
		}

		// If accounts exist but none are enabled, return error (don't fallback to static config)
		if hasAccounts && !hasEnabledAccounts {
			return nil, &provider.ProviderError{
				Message:   "Provider '" + providerName + "' is disabled. Please enable it in the provider settings.",
				Code:      http.StatusServiceUnavailable,
				Retryable: false,
			}
		}
	}

	// Fallback to registry lookup by model (only when no dynamic accounts configured)
	targetProvider, ok := h.registry.GetByModel(model)
	if !ok {
		return nil, &provider.ProviderError{
			Message:   "No available provider for model: " + model,
			Code:      http.StatusServiceUnavailable,
			Retryable: false,
		}
	}
	return targetProvider, nil
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
	}

	if m, ok := models[providerName]; ok {
		return m
	}
	return []string{}
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
func (h *ProxyHandler) handleStreamResponse(c *gin.Context, p provider.Provider, req *provider.ChatRequest) {
	startTime := time.Now()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	ctx := c.Request.Context()

	// Get stream channel from provider
	stream, err := p.StreamChat(ctx, req)
	if err != nil {
		h.recordMetrics("", "", req.Model, time.Since(startTime), 0, false)
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	var totalTokens int

	// Stream chunks to client
	for chunk := range stream {
		if chunk.Usage != nil {
			totalTokens = chunk.Usage.TotalTokens
		}
		if chunk.Done {
			// Send the final chunk with usage (if present) before [DONE]
			if chunk.Usage != nil || len(chunk.Choices) > 0 {
				streamResp := StreamingResponse{
					ID:      chunk.ID,
					Object:  chunk.Object,
					Created: chunk.Created,
					Model:   chunk.Model,
					Choices: make([]StreamChoice, len(chunk.Choices)),
				}
				if chunk.Usage != nil {
					streamResp.Usage = &Usage{
						PromptTokens:     chunk.Usage.PromptTokens,
						CompletionTokens: chunk.Usage.CompletionTokens,
						TotalTokens:      chunk.Usage.TotalTokens,
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
			// Send [DONE] marker
			c.SSEvent("message", "[DONE]")
			c.Writer.Flush()

			// Record metrics for stream completion
			latency := time.Since(startTime)
			h.recordMetrics("", "", req.Model, latency, totalTokens, true)
			break
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
		ProviderError(c, "Provider request failed", err.Error())
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
	providerMap := make(map[string]gin.H)

	providers := h.registry.ListEnabled()
	for _, p := range providers {
		name := p.Name()
		if name == "claude" {
			name = "anthropic"
		}
		providerMap[name] = gin.H{
			"name":    name,
			"models":  p.Models(),
			"enabled": p.IsEnabled(),
		}
	}

	if h.accountManager != nil {
		accounts := h.accountManager.GetAllAccounts()
		for _, acc := range accounts {
			providerName := acc.Provider
			if providerName == "" {
				providerName = acc.ProviderType
			}
			if strings.Contains(acc.BaseURL, "deepseek.com") {
				providerName = "deepseek"
			} else if strings.Contains(acc.BaseURL, "volces.com") || strings.Contains(acc.BaseURL, "volcengine.com") {
				providerName = "volcengine"
			} else if strings.Contains(acc.BaseURL, "dashscope.aliyuncs.com") {
				providerName = "qwen"
			} else if strings.Contains(acc.BaseURL, "zhipuai.cn") || strings.Contains(acc.BaseURL, "bigmodel.cn") {
				providerName = "zhipu"
			} else if strings.Contains(acc.BaseURL, "moonshot.cn") || strings.Contains(acc.BaseURL, "kimi.ai") {
				providerName = "moonshot"
			} else if strings.Contains(acc.BaseURL, "minimax.com") {
				providerName = "minimax"
			} else if strings.Contains(acc.BaseURL, "baichuanai.com") {
				providerName = "baichuan"
			}

			models := []string{}
			if existing, ok := providerMap[providerName]; ok {
				models = existing["models"].([]string)
			} else if defaultModels, ok := defaultProviderModels[providerName]; ok {
				models = defaultModels
			}

			providerMap[providerName] = gin.H{
				"name":    providerName,
				"models":  models,
				"enabled": acc.Enabled,
			}
		}
	}

	result := make([]gin.H, 0, len(providerMap))
	for _, p := range providerMap {
		result = append(result, p)
	}

	Success(c, gin.H{
		"providers": result,
	})
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
	// Update dashboard stats
	if dh := admin.GetDashboardHandler(); dh != nil {
		dh.UpdateStats(success, latency.Milliseconds(), int64(tokens), model)
	}

	// Log metrics
	metricsData, _ := json.Marshal(map[string]interface{}{
		"user_id":    userID,
		"api_key":    maskAPIKey(apiKey),
		"model":      model,
		"latency_ms": latency.Milliseconds(),
		"tokens":     tokens,
		"success":    success,
		"timestamp":  time.Now().Unix(),
	})
	_ = metricsData
}

// maskAPIKey masks an API key for logging
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
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
