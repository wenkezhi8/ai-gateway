package admin

import (
	"ai-gateway/internal/cache"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"
	"ai-gateway/internal/storage"
	vectordb "ai-gateway/internal/vector-db"

	"github.com/gin-gonic/gin"
)

var globalDashboardHandler *DashboardHandler
var globalAPIKeyHandler *APIKeyHandler

// Handlers contains all admin handlers.
type Handlers struct {
	Account     *AccountHandler
	Provider    *ProviderHandler
	Routing     *RoutingHandler
	Cache       *CacheHandler
	Knowledge   *KnowledgeHandler
	Dashboard   *DashboardHandler
	SmartRouter *RouterHandler
	APIKey      *APIKeyHandler
	Upload      *UploadHandler
	Alert       *AlertHandler
	Feedback    *FeedbackHandler
	Ops         *OpsHandler
	Usage       *UsageHandler
	Settings    *SettingsHandler
	Trace       *TraceHandler
	VectorDB    *vectordb.CollectionHandler
}

// NewHandlers creates all admin handlers.
func NewHandlers(
	accountManager *limiter.AccountManager,
	registry *provider.Registry,
	cacheManager *cache.Manager,
) *Handlers {
	smartRouter := routing.GetGlobalSmartRouter()
	SetGlobalRouter(smartRouter)

	// Initialize feedback collector
	feedbackCollector := routing.NewFeedbackCollector(smartRouter.GetDifficultyAssessor(), smartRouter)
	InitFeedbackHandler(feedbackCollector)

	// Initialize usage handler with storage
	usageHandler := NewUsageHandler(storage.GetSQLiteStorage())

	handlers := &Handlers{
		Account:     NewAccountHandler(accountManager),
		Provider:    NewProviderHandler(registry, accountManager),
		Routing:     NewRoutingHandler(),
		Cache:       NewCacheHandler(cacheManager),
		Knowledge:   NewKnowledgeHandler(storage.GetSQLiteStorage().GetDB()),
		Dashboard:   NewDashboardHandler(registry, accountManager, cacheManager),
		SmartRouter: NewRouterHandler(smartRouter, cacheManager),
		APIKey:      NewAPIKeyHandler(),
		Upload:      NewUploadHandler(),
		Alert:       NewAlertHandler(),
		Feedback:    GetFeedbackHandler(),
		Ops:         NewOpsHandler(),
		Usage:       usageHandler,
		Settings:    NewSettingsHandler(""),
		Trace:       NewTraceHandler(storage.GetSQLiteStorage().GetDB()),
		VectorDB:    vectordb.NewCollectionHandler(vectordb.NewService()),
	}
	// 改动点: 定时健康检测并触发告警
	handlers.Ops.StartHealthMonitor(handlers.Alert, handlers.Dashboard)
	globalDashboardHandler = handlers.Dashboard
	globalAPIKeyHandler = handlers.APIKey
	return handlers
}

// GetDashboardHandler returns the global dashboard handler.
func GetDashboardHandler() *DashboardHandler {
	return globalDashboardHandler
}

// GetAPIKeyHandler returns the global API key handler.
func GetAPIKeyHandler() *APIKeyHandler {
	return globalAPIKeyHandler
}

//nolint:revive // keep compatibility for external callers.
func GetApiKeyHandler() *APIKeyHandler {
	return GetAPIKeyHandler()
}

// RegisterRoutes registers all admin routes.
func RegisterRoutes(r *gin.RouterGroup, handlers *Handlers) {
	// Account management routes
	accounts := r.Group("/accounts")
	accounts.GET("", handlers.Account.ListAccounts)
	accounts.POST("", handlers.Account.CreateAccount)
	accounts.GET("/switch-history", handlers.Account.GetSwitchHistory)
	accounts.GET("/:id", handlers.Account.GetAccount)
	accounts.PUT("/:id", handlers.Account.UpdateAccount)
	accounts.PUT("/:id/status", handlers.Account.UpdateAccountStatus)
	accounts.DELETE("/:id", handlers.Account.DeleteAccount)
	accounts.GET("/:id/usage", handlers.Account.GetAccountUsage)
	accounts.GET("/:id/fetch-models", handlers.Account.FetchModels)
	accounts.POST("/:id/switch", handlers.Account.ForceSwitchAccount)

	// Provider management routes
	providers := r.Group("/providers")
	providers.GET("", handlers.Provider.ListProviders)
	providers.GET("/configs", handlers.Account.GetProviderConfigs)
	providers.POST("", handlers.Provider.CreateProvider)
	providers.GET("/strategies", handlers.Routing.GetStrategies)
	providers.GET("/:id", handlers.Provider.GetProvider)
	providers.PUT("/:id", handlers.Provider.UpdateProvider)
	providers.DELETE("/:id", handlers.Provider.DeleteProvider)
	providers.POST("/:id/test", handlers.Provider.TestProvider)
	providers.POST("/:id/enable", handlers.Provider.EnableProvider)
	providers.POST("/:id/disable", handlers.Provider.DisableProvider)
	providers.GET("/:id/models", handlers.Provider.GetProviderModels)

	// Routing strategy routes
	routingGroup := r.Group("/routing")
	routingGroup.GET("", handlers.Routing.GetRouting)
	routingGroup.PUT("", handlers.Routing.UpdateRouting)
	routingGroup.GET("/strategies", handlers.Routing.GetStrategies)
	routingGroup.PUT("/models/:model/strategy", handlers.Routing.SetModelStrategy)
	routingGroup.PUT("/providers/:provider/weight", handlers.Routing.SetProviderWeight)
	routingGroup.POST("/reset", handlers.Routing.ResetRouting)

	// Smart router routes (智能模型选择)
	routerGroup := r.Group("/router")
	routerGroup.GET("/config", handlers.SmartRouter.GetRouterConfig)
	routerGroup.PUT("/config", handlers.SmartRouter.UpdateRouterConfig)
	routerGroup.GET("/models", handlers.SmartRouter.GetModelScores)
	routerGroup.PUT("/models/:model", handlers.SmartRouter.UpdateModelScore)
	routerGroup.DELETE("/models/:model", handlers.SmartRouter.DeleteModelScore)
	routerGroup.GET("/available-models", handlers.SmartRouter.GetAvailableModels)
	routerGroup.GET("/top-models", handlers.SmartRouter.GetTopModels)
	routerGroup.POST("/select", handlers.SmartRouter.SelectModel)
	routerGroup.GET("/provider-defaults", handlers.SmartRouter.GetProviderDefaults)
	routerGroup.PUT("/provider-defaults", handlers.SmartRouter.UpdateProviderDefaults)
	routerGroup.GET("/ttl-config", handlers.SmartRouter.GetTTLConfig)
	routerGroup.PUT("/ttl-config", handlers.SmartRouter.UpdateTTLConfig)
	// Cascade rules
	routerGroup.GET("/cascade-rules", handlers.SmartRouter.GetCascadeRules)
	routerGroup.GET("/cascade-rules/:taskType/:difficulty", handlers.SmartRouter.GetCascadeRule)
	routerGroup.PUT("/cascade-rules", handlers.SmartRouter.UpdateCascadeRule)
	routerGroup.DELETE("/cascade-rules/:taskType/:difficulty", handlers.SmartRouter.DeleteCascadeRule)
	routerGroup.POST("/cascade-rules/reset", handlers.SmartRouter.ResetCascadeRules)
	// Task model mapping
	routerGroup.GET("/task-model-mapping", handlers.SmartRouter.GetTaskModelMapping)
	routerGroup.PUT("/task-model-mapping", handlers.SmartRouter.UpdateTaskModelMapping)
	routerGroup.GET("/classifier/health", handlers.SmartRouter.GetClassifierHealth)
	routerGroup.GET("/classifier/stats", handlers.SmartRouter.GetClassifierStats)
	routerGroup.GET("/classifier/models", handlers.SmartRouter.GetClassifierModels)
	routerGroup.GET("/ollama/dual-model/config", handlers.SmartRouter.GetOllamaDualModelConfig)
	routerGroup.PUT("/ollama/dual-model/config", handlers.SmartRouter.UpdateOllamaDualModelConfig)
	routerGroup.POST("/classifier/switch", handlers.SmartRouter.SwitchClassifierModel)
	routerGroup.POST("/classifier/switch-async", handlers.SmartRouter.SwitchClassifierModelAsync)
	routerGroup.GET("/classifier/switch-tasks/:taskId", handlers.SmartRouter.GetSwitchClassifierTask)
	routerGroup.GET("/ollama/status", handlers.SmartRouter.GetOllamaSetupStatus)
	routerGroup.POST("/ollama/install", handlers.SmartRouter.InstallOllama)
	routerGroup.POST("/ollama/start", handlers.SmartRouter.StartOllama)
	routerGroup.POST("/ollama/stop", handlers.SmartRouter.StopOllama)
	routerGroup.POST("/ollama/pull", handlers.SmartRouter.PullOllamaModel)

	// Cache management routes
	cacheGroup := r.Group("/cache")
	cacheGroup.GET("/stats", handlers.Cache.GetCacheStats)
	cacheGroup.DELETE("", handlers.Cache.ClearCache)
	cacheGroup.GET("/config", handlers.Cache.GetCacheConfig)
	cacheGroup.PUT("/config", handlers.Cache.UpdateCacheConfig)
	cacheGroup.DELETE("/provider/:provider", handlers.Cache.InvalidateProvider)
	cacheGroup.DELETE("/model/:model", handlers.Cache.InvalidateModel)
	cacheGroup.GET("/health", handlers.Cache.GetCacheHealth)
	cacheGroup.GET("/summary", handlers.Cache.GetCacheSummary)
	cacheGroup.GET("/vector/stats", handlers.Cache.GetVectorStats)
	cacheGroup.POST("/vector/rebuild", handlers.Cache.RebuildVectorIndex)
	cacheGroup.GET("/vector/pipeline/health", handlers.Cache.GetVectorPipelineHealth)
	cacheGroup.POST("/vector/pipeline/test", handlers.Cache.TestVectorPipeline)
	cacheGroup.GET("/vector/tier/stats", handlers.Cache.GetVectorTierStats)
	cacheGroup.POST("/vector/tier/migrate", handlers.Cache.TriggerVectorTierMigrate)
	cacheGroup.POST("/vector/tier/promote", handlers.Cache.PromoteVectorTierEntry)
	cacheGroup.GET("/semantic-signatures", handlers.Cache.GetSemanticSignatures)
	cacheGroup.GET("/quality-config", handlers.Cache.GetCacheQualityConfig)
	cacheGroup.PUT("/quality-config", handlers.Cache.UpdateCacheQualityConfig)
	cacheGroup.POST("/invalidate-low-quality", handlers.Cache.InvalidateLowQualityCache)
	// Cache rules
	cacheGroup.GET("/rules", handlers.Cache.GetCacheRules)
	cacheGroup.POST("/rules", handlers.Cache.CreateCacheRule)
	cacheGroup.PUT("/rules/:id", handlers.Cache.UpdateCacheRule)
	cacheGroup.DELETE("/rules/:id", handlers.Cache.DeleteCacheRule)
	// Cache entries management
	cacheGroup.GET("/entries", handlers.Cache.GetCacheEntries)
	cacheGroup.POST("/entries/cleanup-invalid", handlers.Cache.CleanupInvalidEntries)
	cacheGroup.POST("/entries/cleanup-empty", handlers.Cache.CleanupEmptyResponseEntries)
	cacheGroup.POST("/entries/delete-group", handlers.Cache.DeleteCacheEntryGroup)
	cacheGroup.GET("/entries/*key", handlers.Cache.GetCacheEntryDetail)
	cacheGroup.DELETE("/entries/*key", handlers.Cache.DeleteCacheEntry)
	// Cache warmup and export
	cacheGroup.POST("/test-entry", handlers.Cache.AddTestCacheEntry)
	cacheGroup.GET("/export", handlers.Cache.ExportCacheEntries)
	cacheGroup.GET("/trend", handlers.Cache.GetCacheTrend)
	// Model name mapping cache
	cacheGroup.GET("/model-mappings", handlers.Cache.GetModelMappings)
	cacheGroup.DELETE("/model-mappings", handlers.Cache.ClearModelMappings)
	cacheGroup.POST("/model-mappings/cleanup", handlers.Cache.CleanupModelMappings)

	knowledgeGroup := r.Group("/knowledge")
	{
		documents := knowledgeGroup.Group("/documents")
		documents.GET("", handlers.Knowledge.ListDocuments)
		documents.POST("/upload", handlers.Knowledge.UploadDocument)
		documents.GET("/:id", handlers.Knowledge.GetDocument)
		documents.DELETE("/:id", handlers.Knowledge.DeleteDocument)
		documents.POST("/:id/vectorize", handlers.Knowledge.VectorizeDocument)

		chunks := knowledgeGroup.Group("/chunks")
		chunks.GET("", handlers.Knowledge.ListChunks)
		chunks.GET("/:id", handlers.Knowledge.GetChunk)

		chat := knowledgeGroup.Group("/chat")
		chat.POST("/message", handlers.Knowledge.ChatMessage)

		config := knowledgeGroup.Group("/config")
		config.GET("", handlers.Knowledge.GetConfig)
		config.PUT("", handlers.Knowledge.UpdateConfig)
	}

	// Trace routes
	traceGroup := r.Group("/traces")
	traceGroup.GET("", handlers.Trace.GetTraces)
	traceGroup.GET("/:request_id", handlers.Trace.GetTraceDetail)

	// Dashboard routes
	dashboard := r.Group("/dashboard")
	dashboard.GET("/stats", handlers.Dashboard.GetStats)
	dashboard.GET("/requests", handlers.Dashboard.GetRequestTrends)
	dashboard.GET("/realtime", handlers.Dashboard.GetRealtime)
	dashboard.GET("/alerts", handlers.Dashboard.GetAlerts)
	dashboard.POST("/alerts/:id/acknowledge", handlers.Dashboard.AcknowledgeAlert)
	dashboard.GET("/providers/:provider/metrics", handlers.Dashboard.GetProviderMetrics)
	dashboard.GET("/models/:model/metrics", handlers.Dashboard.GetModelMetrics)
	dashboard.GET("/system", handlers.Dashboard.GetSystemStatus)

	// API Key management routes
	apiKeys := r.Group("/api-keys")
	apiKeys.GET("", handlers.APIKey.ListAPIKeys)
	apiKeys.POST("", handlers.APIKey.CreateAPIKey)
	apiKeys.GET("/:id", handlers.APIKey.GetAPIKey)
	apiKeys.PUT("/:id", handlers.APIKey.UpdateAPIKey)
	apiKeys.DELETE("/:id", handlers.APIKey.DeleteAPIKey)

	// Upload routes
	upload := r.Group("/upload")
	upload.POST("/logo", handlers.Upload.UploadLogo)

	// Alert management routes
	alerts := r.Group("/alerts")
	alerts.GET("/stats", handlers.Alert.GetStats)
	alerts.GET("/rules", handlers.Alert.GetRules)
	alerts.POST("/rules", handlers.Alert.CreateRule)
	alerts.PUT("/rules/:id", handlers.Alert.UpdateRule)
	alerts.DELETE("/rules/:id", handlers.Alert.DeleteRule)
	alerts.GET("/history", handlers.Alert.GetHistory)
	alerts.GET("/:id", handlers.Alert.GetAlertDetail)
	alerts.PUT("/:id/resolve", handlers.Alert.ResolveAlert)

	// Feedback and evaluation routes
	feedback := r.Group("/feedback")
	feedback.POST("", handlers.Feedback.SubmitFeedback)
	feedback.GET("/stats", handlers.Feedback.GetFeedbackStats)
	feedback.GET("/performance", handlers.Feedback.GetAllPerformance)
	feedback.GET("/performance/:model", handlers.Feedback.GetModelPerformance)
	feedback.GET("/top-models", handlers.Feedback.GetTopModels)
	feedback.GET("/recent", handlers.Feedback.GetRecentFeedback)
	feedback.GET("/task-type-distribution", handlers.Feedback.GetTaskTypeDistribution)
	feedback.POST("/optimize", handlers.Feedback.TriggerOptimization)

	// Ops monitoring routes
	ops := r.Group("/ops")
	opsRoutes := []struct {
		path    string
		handler gin.HandlerFunc
	}{
		{path: "/dashboard", handler: handlers.Ops.GetDashboard},
		{path: "/system", handler: handlers.Ops.GetSystemInfo},
		{path: "/realtime", handler: handlers.Ops.GetRealtime},
		{path: "/resources", handler: handlers.Ops.GetResources},
		{path: "/diagnosis", handler: handlers.Ops.GetDiagnosis},
		{path: "/services", handler: handlers.Ops.GetServices},
		{path: "/health-checks", handler: handlers.Ops.GetHealthChecks},
		{path: "/events", handler: handlers.Ops.GetEvents},
		{path: "/providers/health", handler: handlers.Ops.GetProviderHealth},
		{path: "/export", handler: handlers.Ops.ExportMetrics},
	}
	for _, route := range opsRoutes {
		ops.GET(route.path, route.handler)
	}

	// Usage routes
	usage := r.Group("/usage")
	usage.GET("/logs", handlers.Usage.GetUsageLogs)
	usage.GET("/stats", handlers.Usage.GetUsageStats)

	// UI settings routes
	settings := r.Group("/settings")
	settings.GET("/ui", handlers.Settings.GetUISettings)
	settings.PUT("/ui", handlers.Settings.UpdateUISettings)

	// Vector DB collection routes
	vectorDBCollectionsGroup := r.Group("/vector-db/collections")
	vectorDBCollectionsGroup.POST("", handlers.VectorDB.CreateCollection)
	vectorDBCollectionsGroup.GET("", handlers.VectorDB.ListCollections)
	vectorDBCollectionsGroup.GET("/:name", handlers.VectorDB.GetCollection)
	vectorDBCollectionsGroup.PUT("/:name", handlers.VectorDB.UpdateCollection)
	vectorDBCollectionsGroup.DELETE("/:name", handlers.VectorDB.DeleteCollection)
}
