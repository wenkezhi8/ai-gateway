package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-gateway/internal/audit"
	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/handler/admin"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/metrics"
	"ai-gateway/internal/middleware"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/provider/claude"
	"ai-gateway/internal/provider/deepseek"
	"ai-gateway/internal/provider/ernie"
	"ai-gateway/internal/provider/openai"
	"ai-gateway/internal/provider/qwen"
	"ai-gateway/internal/provider/volcengine"
	"ai-gateway/internal/provider/zhipu"
	"ai-gateway/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var jwtSecret string

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	logger.SetOutput(os.Stdout)

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		if logLevel, err := logrus.ParseLevel(level); err == nil {
			logger.SetLevel(logLevel)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	if err := cfg.Validate(); err != nil {
		logger.WithError(err).Fatal("Invalid configuration")
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs/config.json"
	}

	configWatcher, err := config.NewConfigWatcher(configPath, cfg, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to create config watcher, hot reload disabled")
	} else {
		defer configWatcher.Close()
		configWatcher.OnReload(func(newCfg *config.Config) {
			logger.Info("Configuration reloaded")
		})
	}

	auditLogger := audit.InitLogger("./data/audit.log", 10000)
	defer auditLogger.Close()

	registry := provider.GetRegistry()

	registry.RegisterFactory("openai", func(cfg *provider.ProviderConfig) provider.Provider {
		return openai.NewAdapter(cfg)
	})
	registry.RegisterFactory("anthropic", func(cfg *provider.ProviderConfig) provider.Provider {
		return claude.NewAdapter(cfg)
	})
	registry.RegisterFactory("volcengine", func(cfg *provider.ProviderConfig) provider.Provider {
		return volcengine.NewAdapter(cfg)
	})
	registry.RegisterFactory("deepseek", func(cfg *provider.ProviderConfig) provider.Provider {
		return deepseek.NewAdapter(cfg)
	})
	registry.RegisterFactory("zhipu", func(cfg *provider.ProviderConfig) provider.Provider {
		return zhipu.NewAdapter(cfg)
	})
	registry.RegisterFactory("qwen", func(cfg *provider.ProviderConfig) provider.Provider {
		return qwen.NewAdapter(cfg)
	})
	registry.RegisterFactory("ernie", func(cfg *provider.ProviderConfig) provider.Provider {
		return ernie.NewAdapter(cfg)
	})

	// Default models for each provider type
	defaultModels := map[string][]string{
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
		"ernie": {
			"ernie-4.0-8k", "ernie-4.0", "ernie-3.5-8k",
			"ernie-3.5", "ernie-speed-8k", "ernie-speed",
		},
		"volcengine": {
			"doubao-pro-32k", "doubao-pro-128k", "doubao-pro-256k",
			"doubao-lite-32k", "doubao-lite-128k",
		},
	}

	for _, p := range cfg.Providers {
		if p.Enabled {
			models := p.Models
			if len(models) == 0 {
				models = defaultModels[p.Name]
			}
			providerCfg := &provider.ProviderConfig{
				Name:    p.Name,
				APIKey:  p.APIKey,
				BaseURL: p.BaseURL,
				Models:  models,
				Enabled: p.Enabled,
			}
			if _, err := registry.CreateAndRegister(providerCfg); err != nil {
				logger.WithError(err).Warnf("Failed to register provider %s", p.Name)
			} else {
				logger.Infof("Registered provider: %s (models: %d)", p.Name, len(models))
			}
		}
	}

	accountManager := limiter.NewAccountManager(nil, nil)
	for _, acc := range cfg.Accounts {
		accountConfig := &limiter.AccountConfig{
			ID:       acc.ID,
			Name:     acc.Name,
			Provider: acc.Provider,
			APIKey:   acc.APIKey,
			BaseURL:  acc.BaseURL,
			Enabled:  acc.Enabled,
			Priority: acc.Priority,
			Limits:   make(map[limiter.LimitType]*limiter.LimitConfig),
		}
		for _, limitCfg := range acc.Limits {
			var limitType limiter.LimitType
			switch limitCfg.Type {
			case "token":
				limitType = limiter.LimitTypeToken
			case "rpm":
				limitType = limiter.LimitTypeRPM
			default:
				continue
			}
			var period limiter.Period
			switch limitCfg.Period {
			case "minute":
				period = limiter.PeriodMinute
			case "hour":
				period = limiter.PeriodHour
			case "day":
				period = limiter.PeriodDay
			case "month":
				period = limiter.PeriodMonth
			default:
				period = limiter.PeriodDay
			}
			accountConfig.Limits[limitType] = &limiter.LimitConfig{
				Type:    limitType,
				Period:  period,
				Limit:   limitCfg.Limit,
				Warning: limitCfg.Warning,
			}
		}
		if err := accountManager.AddAccount(accountConfig); err != nil {
			logger.WithError(err).Warnf("Failed to add account %s", acc.ID)
		} else {
			logger.Infof("Added account: %s (%s)", acc.ID, acc.Provider)
		}
	}

	// Load persisted accounts from file (these override config file accounts)
	persistedAccounts, err := admin.LoadPersistedAccounts()
	if err != nil {
		logger.WithError(err).Warn("Failed to load persisted accounts")
	} else {
		for _, acc := range persistedAccounts {
			if _, err := accountManager.GetAccount(acc.ID); err == nil {
				// Account exists, update it with persisted data (enabled status, limits, etc.)
				if err := accountManager.UpdateAccount(acc); err != nil {
					logger.WithError(err).Warnf("Failed to update persisted account %s", acc.ID)
				} else {
					logger.Infof("Updated account from persistence: %s (enabled: %v)", acc.ID, acc.Enabled)
				}
			} else {
				// New account, add it
				if err := accountManager.AddAccount(acc); err != nil {
					logger.WithError(err).Warnf("Failed to add persisted account %s", acc.ID)
				} else {
					logger.Infof("Loaded persisted account: %s (%s)", acc.ID, acc.Provider)
				}
			}
		}
	}

	cacheManager, err := cache.NewManager(cache.ManagerConfig{
		Redis: cache.RedisConfig{
			Host:     cfg.Redis.Host,
			Port:     cfg.Redis.Port,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		},
		UseRedis: true,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to connect to Redis, falling back to memory cache")
		cacheManager = cache.NewManagerWithCache(cache.NewMemoryCache())
	} else {
		logger.Infof("Connected to Redis at %s:%d", cfg.Redis.Host, cfg.Redis.Port)
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9090"
	}

	metrics.Init()

	metricsRouter := gin.New()
	metricsRouter.GET("/metrics", func(c *gin.Context) {
		if m := metrics.GetMetrics(); m != nil {
			m.PrometheusHandler().ServeHTTP(c.Writer, c.Request)
		} else {
			c.Header("Content-Type", "text/plain")
			c.String(404, "Metrics not initialized")
		}
	})

	metricsSrv := &http.Server{
		Addr:         ":" + metricsPort,
		Handler:      metricsRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Infof("Starting metrics server on port %s", metricsPort)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Warn("Failed to start metrics server")
		}
	}()

	jwtSecret = os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "ai-gateway-default-secret-change-in-production"
	}

	jwtConfig := middleware.JWTConfig{
		Secret:     jwtSecret,
		ExpireTime: 24 * time.Hour,
		Issuer:     "ai-gateway",
	}

	routerCfg := router.RouterConfig{
		AuthCfg:       middleware.AuthConfig{Enabled: false},
		JWTConfig:     jwtConfig,
		AuditLogger:   auditLogger,
		EnableSwagger: true,
	}

	r := router.NewFullWithConfig(cfg, routerCfg, accountManager, cacheManager, registry)

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Server.Port
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Infof("Starting AI Gateway on port %s", port)
		logger.Infof("Swagger UI available at http://localhost:%s/swagger/index.html", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	<-quit
	logger.Info("Shutting down servers...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Shutting down metrics server...")
	metricsShutdownCtx, metricsCancel := context.WithTimeout(shutdownCtx, 5*time.Second)
	defer metricsCancel()

	if err := metricsSrv.Shutdown(metricsShutdownCtx); err != nil {
		logger.WithError(err).Warn("Metrics server forced to shutdown")
	} else {
		logger.Info("Metrics server gracefully stopped")
	}

	logger.Info("Shutting down main server...")
	mainShutdownCtx, mainCancel := context.WithTimeout(shutdownCtx, 25*time.Second)
	defer mainCancel()

	if err := srv.Shutdown(mainShutdownCtx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	} else {
		logger.Info("Main server gracefully stopped")
	}

	if err := cacheManager.Close(); err != nil {
		logger.WithError(err).Error("Error closing cache manager")
	}

	logger.Info("AI Gateway stopped")
}
