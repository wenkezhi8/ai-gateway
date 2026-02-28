package router

import (
	"ai-gateway/internal/audit"
	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/constants"
	"ai-gateway/internal/docs"
	"ai-gateway/internal/handler"
	"ai-gateway/internal/handler/admin"
	authHandler "ai-gateway/internal/handler/auth"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/middleware"
	"ai-gateway/internal/provider"

	"github.com/gin-gonic/gin"

	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strings"
)

// New creates and configures a new Gin router
func New(cfg *config.Config) *gin.Engine {
	return NewWithAuth(cfg, middleware.AuthConfig{Enabled: false})
}

// NewWithAuth creates and configures a new Gin router with auth settings
func NewWithAuth(cfg *config.Config, authCfg middleware.AuthConfig) *gin.Engine {
	return NewFull(cfg, authCfg, nil, nil, nil)
}

// RouterConfig holds all configuration for the router
type RouterConfig struct {
	AuthCfg       middleware.AuthConfig
	JWTConfig     middleware.JWTConfig
	AuditLogger   *audit.Logger
	EnableSwagger bool
}

// NewFull creates and configures a new Gin router with all features
func NewFull(
	cfg *config.Config,
	authCfg middleware.AuthConfig,
	accountManager *limiter.AccountManager,
	cacheManager *cache.Manager,
	registry *provider.Registry,
) *gin.Engine {
	return NewFullWithConfig(cfg, RouterConfig{
		AuthCfg:       authCfg,
		JWTConfig:     middleware.JWTConfig{},
		AuditLogger:   nil,
		EnableSwagger: true,
	}, accountManager, cacheManager, registry)
}

// NewFullWithConfig creates and configures a new Gin router with extended options
func NewFullWithConfig(
	cfg *config.Config,
	routerCfg RouterConfig,
	accountManager *limiter.AccountManager,
	cacheManager *cache.Manager,
	registry *provider.Registry,
) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	if routerCfg.AuditLogger != nil {
		auditMiddleware := audit.NewAuditMiddleware(routerCfg.AuditLogger)
		r.Use(auditMiddleware.Middleware())
	}

	if cfg.Limiter.Enabled {
		r.Use(middleware.RateLimiter(cfg.Limiter))
	}

	if routerCfg.EnableSwagger {
		docs.SetupSwaggerRoutes(r)
	}

	healthHandler := handler.NewHealthHandler()
	proxyHandler := handler.NewProxyHandler(cfg, accountManager, cacheManager)
	searchHandler := handler.NewSearchHandler()

	r.GET("/health", healthHandler.Check)
	r.GET("/ready", healthHandler.CheckReady)

	// 改动点: 仅在调试模式或显式启用时暴露 pprof
	if cfg.Server.Mode == "debug" || os.Getenv("PPROF_ENABLED") == "1" {
		pprofGroup := r.Group("/debug/pprof")
		{
			pprofGroup.GET("/", gin.WrapF(pprof.Index))
			pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
			pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
			pprofGroup.POST("/symbol", gin.WrapF(pprof.Symbol))
			pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
			pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
			pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
			pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
			pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
			pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
			pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
			pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
		}
	}

	// Docs alias for Swagger UI
	r.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/index.html")
	})

	// API v1 routes (统一入口)
	r.GET(constants.ApiV1Prefix, proxyHandler.APIv1Root)
	r.POST(constants.ApiV1Prefix, proxyHandler.ChatCompletions)
	r.GET(constants.ChatCompletions, proxyHandler.APIv1ChatCompletionsInfo)
	apiV1 := r.Group(constants.ApiV1Prefix)
	{
		apiV1.POST("/chat/completions", proxyHandler.ChatCompletions)
		apiV1.POST("/completions", proxyHandler.Completions)
		apiV1.POST("/embeddings", proxyHandler.Embeddings)
		apiV1.GET("/providers", proxyHandler.ListProviders)
		apiV1.GET("/models", proxyHandler.ListModels)
		apiV1.GET("/config/providers", proxyHandler.ListConfiguredProviders)
		apiV1.POST("/search", searchHandler.Search)
	}

	// Anthropic-compatible routes
	r.GET(constants.ApiAnthropicBasePrefix, proxyHandler.AnthropicRoot)
	r.POST(constants.ApiAnthropicBasePrefix, proxyHandler.AnthropicMessages)
	r.GET(constants.ApiAnthropicPrefix+"/messages", proxyHandler.AnthropicMessagesInfo)
	anthropicV1 := r.Group(constants.ApiAnthropicPrefix)
	{
		anthropicV1.POST("/messages", proxyHandler.AnthropicMessages)
	}

	if routerCfg.JWTConfig.Secret != "" {
		authH := authHandler.NewAuthHandler(routerCfg.JWTConfig, routerCfg.AuditLogger)

		authGroup := r.Group(constants.AuthPrefix)
		{
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/logout", middleware.JWTAuth(routerCfg.JWTConfig), authH.Logout)
			authGroup.GET("/me", middleware.JWTAuth(routerCfg.JWTConfig), authH.GetCurrentUser)
			authGroup.POST("/change-password", middleware.JWTAuth(routerCfg.JWTConfig), authH.ChangePassword)
			authGroup.PUT("/profile", middleware.JWTAuth(routerCfg.JWTConfig), authH.UpdateProfile)
			authGroup.POST("/refresh", middleware.JWTAuth(routerCfg.JWTConfig), authH.RefreshToken)
			authGroup.POST("/validate", authH.ValidateToken)

			adminAuth := authGroup.Group("")
			adminAuth.Use(middleware.JWTAuth(routerCfg.JWTConfig), middleware.RequireRole("admin"))
			adminAuth.GET("/users", authH.ListUsers)
			adminAuth.POST("/users", authH.CreateUser)
			adminAuth.DELETE("/users/:username", authH.DeleteUser)
		}
	}

	if routerCfg.AuditLogger != nil {
		auditGroup := r.Group("/api/audit")
		if routerCfg.JWTConfig.Secret != "" {
			auditGroup.Use(middleware.JWTAuth(routerCfg.JWTConfig))
		}
		{
			auditGroup.GET("/logs", audit.AuditHandler(routerCfg.AuditLogger))
		}
	}

	if accountManager != nil && registry != nil {
		if registry == nil {
			registry = provider.GetRegistry()
		}

		adminHandlers := admin.NewHandlers(accountManager, registry, cacheManager)

		// Set model mapping cache from proxy handler
		if mmc := proxyHandler.GetModelMappingCache(); mmc != nil {
			adminHandlers.Cache.SetModelMappingCache(mmc)
		}

		adminGroup := r.Group(constants.AdminPrefix)
		{
			if routerCfg.JWTConfig.Secret != "" {
				adminGroup.Use(middleware.JWTAuth(routerCfg.JWTConfig))
			}
			admin.RegisterRoutes(adminGroup, adminHandlers)
		}
	}

	// Serve frontend static files
	// Check for static dir in multiple locations
	staticDirs := []string{
		"./web/dist",
		"./dist",
		"../web/dist",
		"web/dist",
	}

	// Also check environment variable
	if envDir := os.Getenv("STATIC_DIR"); envDir != "" {
		staticDirs = append([]string{envDir}, staticDirs...)
	}

	var staticDir string
	for _, dir := range staticDirs {
		absDir, _ := filepath.Abs(dir)
		if _, err := os.Stat(absDir); err == nil {
			staticDir = absDir
			break
		}
	}

	if staticDir != "" {
		r.Static("/assets", filepath.Join(staticDir, "assets"))
		r.Static("/logos", filepath.Join(staticDir, "logos"))
		r.StaticFile("/vite.svg", filepath.Join(staticDir, "vite.svg"))

		// SPA fallback - serve index.html for unmatched routes
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// Skip API routes - be more specific to avoid catching frontend routes like /api-management
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/swagger") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.File(filepath.Join(staticDir, "index.html"))
		})
	}

	return r
}
