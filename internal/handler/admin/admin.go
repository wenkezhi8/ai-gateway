package admin

import (
	"ai-gateway/internal/cache"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

var globalDashboardHandler *DashboardHandler

// Handlers contains all admin handlers
type Handlers struct {
	Account     *AccountHandler
	Provider    *ProviderHandler
	Routing     *RoutingHandler
	Cache       *CacheHandler
	Dashboard   *DashboardHandler
	SmartRouter *RouterHandler
	ApiKey      *ApiKeyHandler
	Upload      *UploadHandler
	Alert       *AlertHandler
}

// NewHandlers creates all admin handlers
func NewHandlers(
	accountManager *limiter.AccountManager,
	registry *provider.Registry,
	cacheManager *cache.Manager,
) *Handlers {
	smartRouter := routing.NewSmartRouter()
	SetGlobalRouter(smartRouter)
	handlers := &Handlers{
		Account:     NewAccountHandler(accountManager),
		Provider:    NewProviderHandler(registry),
		Routing:     NewRoutingHandler(),
		Cache:       NewCacheHandler(cacheManager),
		Dashboard:   NewDashboardHandler(registry, accountManager, cacheManager),
		SmartRouter: NewRouterHandler(smartRouter),
		ApiKey:      NewApiKeyHandler(),
		Upload:      NewUploadHandler(),
		Alert:       NewAlertHandler(),
	}
	globalDashboardHandler = handlers.Dashboard
	return handlers
}

// GetDashboardHandler returns the global dashboard handler
func GetDashboardHandler() *DashboardHandler {
	return globalDashboardHandler
}

// RegisterRoutes registers all admin routes
func RegisterRoutes(r *gin.RouterGroup, handlers *Handlers) {
	// Account management routes
	accounts := r.Group("/accounts")
	{
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
	}

	// Provider management routes
	providers := r.Group("/providers")
	{
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
	}

	// Routing strategy routes
	routingGroup := r.Group("/routing")
	{
		routingGroup.GET("", handlers.Routing.GetRouting)
		routingGroup.PUT("", handlers.Routing.UpdateRouting)
		routingGroup.GET("/strategies", handlers.Routing.GetStrategies)
		routingGroup.PUT("/models/:model/strategy", handlers.Routing.SetModelStrategy)
		routingGroup.PUT("/providers/:provider/weight", handlers.Routing.SetProviderWeight)
		routingGroup.POST("/reset", handlers.Routing.ResetRouting)
	}

	// Smart router routes (智能模型选择)
	routerGroup := r.Group("/router")
	{
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
	}

	// Cache management routes
	cacheGroup := r.Group("/cache")
	{
		cacheGroup.GET("/stats", handlers.Cache.GetCacheStats)
		cacheGroup.DELETE("", handlers.Cache.ClearCache)
		cacheGroup.GET("/config", handlers.Cache.GetCacheConfig)
		cacheGroup.PUT("/config", handlers.Cache.UpdateCacheConfig)
		cacheGroup.DELETE("/provider/:provider", handlers.Cache.InvalidateProvider)
		cacheGroup.DELETE("/model/:model", handlers.Cache.InvalidateModel)
		cacheGroup.GET("/health", handlers.Cache.GetCacheHealth)
		cacheGroup.GET("/summary", handlers.Cache.GetCacheSummary)
	}

	// Dashboard routes
	dashboard := r.Group("/dashboard")
	{
		dashboard.GET("/stats", handlers.Dashboard.GetStats)
		dashboard.GET("/requests", handlers.Dashboard.GetRequestTrends)
		dashboard.GET("/realtime", handlers.Dashboard.GetRealtime)
		dashboard.GET("/alerts", handlers.Dashboard.GetAlerts)
		dashboard.POST("/alerts/:id/acknowledge", handlers.Dashboard.AcknowledgeAlert)
		dashboard.GET("/providers/:provider/metrics", handlers.Dashboard.GetProviderMetrics)
		dashboard.GET("/models/:model/metrics", handlers.Dashboard.GetModelMetrics)
		dashboard.GET("/system", handlers.Dashboard.GetSystemStatus)
	}

	// API Key management routes
	apiKeys := r.Group("/api-keys")
	{
		apiKeys.GET("", handlers.ApiKey.ListApiKeys)
		apiKeys.POST("", handlers.ApiKey.CreateApiKey)
		apiKeys.GET("/:id", handlers.ApiKey.GetApiKey)
		apiKeys.PUT("/:id", handlers.ApiKey.UpdateApiKey)
		apiKeys.DELETE("/:id", handlers.ApiKey.DeleteApiKey)
	}

	// Upload routes
	upload := r.Group("/upload")
	{
		upload.POST("/logo", handlers.Upload.UploadLogo)
	}

	// Alert management routes
	alerts := r.Group("/alerts")
	{
		alerts.GET("/stats", handlers.Alert.GetStats)
		alerts.GET("/rules", handlers.Alert.GetRules)
		alerts.POST("/rules", handlers.Alert.CreateRule)
		alerts.PUT("/rules/:id", handlers.Alert.UpdateRule)
		alerts.DELETE("/rules/:id", handlers.Alert.DeleteRule)
		alerts.GET("/history", handlers.Alert.GetHistory)
		alerts.GET("/:id", handlers.Alert.GetAlertDetail)
		alerts.PUT("/:id/resolve", handlers.Alert.ResolveAlert)
	}
}
