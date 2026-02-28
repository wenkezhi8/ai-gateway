package bootstrap

import (
	"context"
	"net/http"
	"os"
	"time"

	"ai-gateway/internal/audit"
	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/handler/admin"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/metrics"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/provider/claude"
	"ai-gateway/internal/provider/deepseek"
	"ai-gateway/internal/provider/ernie"
	"ai-gateway/internal/provider/google"
	"ai-gateway/internal/provider/openai"
	"ai-gateway/internal/provider/qwen"
	"ai-gateway/internal/provider/volcengine"
	"ai-gateway/internal/provider/zhipu"
	pkglogger "ai-gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func NewLogger() *logrus.Logger {
	return pkglogger.Log
}

func SetupConfigWatcher(cfg *config.Config, logger *logrus.Logger) *config.ConfigWatcher {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs/config.json"
	}

	watcher, err := config.NewConfigWatcher(configPath, cfg, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to create config watcher, hot reload disabled")
		return nil
	}
	watcher.OnReload(func(_ *config.Config) {
		logger.Info("Configuration reloaded")
	})
	return watcher
}

func InitAuditLogger() *audit.Logger {
	return audit.InitLogger("./data/audit.log", 10000)
}

func InitProviderRegistry(cfg *config.Config, logger *logrus.Logger) *provider.Registry {
	registry := provider.GetRegistry()

	registry.RegisterFactory("openai", func(cfg *provider.ProviderConfig) provider.Provider { return openai.NewAdapter(cfg) })
	registry.RegisterFactory("anthropic", func(cfg *provider.ProviderConfig) provider.Provider { return claude.NewAdapter(cfg) })
	registry.RegisterFactory("volcengine", func(cfg *provider.ProviderConfig) provider.Provider { return volcengine.NewAdapter(cfg) })
	registry.RegisterFactory("deepseek", func(cfg *provider.ProviderConfig) provider.Provider { return deepseek.NewAdapter(cfg) })
	registry.RegisterFactory("zhipu", func(cfg *provider.ProviderConfig) provider.Provider { return zhipu.NewAdapter(cfg) })
	registry.RegisterFactory("qwen", func(cfg *provider.ProviderConfig) provider.Provider { return qwen.NewAdapter(cfg) })
	registry.RegisterFactory("ernie", func(cfg *provider.ProviderConfig) provider.Provider { return ernie.NewAdapter(cfg) })
	registry.RegisterFactory("google", func(cfg *provider.ProviderConfig) provider.Provider { return google.NewAdapter(cfg) })

	defaultModels := map[string][]string{
		"openai":     {"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4-turbo-preview", "gpt-4", "gpt-3.5-turbo", "gpt-3.5-turbo-16k", "o1", "o1-mini", "o1-preview"},
		"anthropic":  {"claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20240620", "claude-3-5-haiku-20241022", "claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
		"deepseek":   {"deepseek-chat", "deepseek-coder", "deepseek-reasoner"},
		"zhipu":      {"glm-4-plus", "glm-4-0520", "glm-4-air", "glm-4-airx", "glm-4-long", "glm-4-flash"},
		"qwen":       {"qwen-max", "qwen-max-longcontext", "qwen-plus", "qwen-turbo", "qwen-long"},
		"ernie":      {"ernie-4.0-8k", "ernie-4.0", "ernie-3.5-8k", "ernie-3.5", "ernie-speed-8k", "ernie-speed"},
		"volcengine": {"doubao-pro-32k", "doubao-pro-128k", "doubao-pro-256k", "doubao-lite-32k", "doubao-lite-128k"},
		"google":     {"gemini-3.1-pro-preview", "gemini-2.5-pro", "gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash"},
	}

	for _, p := range cfg.Providers {
		if !p.Enabled {
			continue
		}
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

	return registry
}

func InitAccountManager(cfg *config.Config, logger *logrus.Logger) *limiter.AccountManager {
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

	persistedAccounts, err := admin.LoadPersistedAccounts()
	if err != nil {
		logger.WithError(err).Warn("Failed to load persisted accounts")
	} else {
		for _, acc := range persistedAccounts {
			if _, err := accountManager.GetAccount(acc.ID); err == nil {
				if err := accountManager.UpdateAccount(acc); err != nil {
					logger.WithError(err).Warnf("Failed to update persisted account %s", acc.ID)
				} else {
					logger.Infof("Updated account from persistence: %s (enabled: %v)", acc.ID, acc.Enabled)
				}
			} else {
				if err := accountManager.AddAccount(acc); err != nil {
					logger.WithError(err).Warnf("Failed to add persisted account %s", acc.ID)
				} else {
					logger.Infof("Loaded persisted account: %s (%s)", acc.ID, acc.Provider)
				}
			}
		}
	}

	persistedSwitchHistory, err := admin.LoadPersistedSwitchHistory()
	if err != nil {
		logger.WithError(err).Warn("Failed to load persisted switch history")
	} else if len(persistedSwitchHistory) > 0 {
		accountManager.SetSwitchHistory(persistedSwitchHistory)
		logger.Infof("Loaded persisted switch history: %d records", len(persistedSwitchHistory))
	}

	accountManager.SetSwitchHistorySaver(func(history []limiter.SwitchEvent) {
		if err := admin.SaveSwitchHistoryToFile(history); err != nil {
			logger.WithError(err).Warn("Failed to persist switch history to file")
		}
	})

	return accountManager
}

func InitCacheManager(cfg *config.Config, logger *logrus.Logger) *cache.Manager {
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
		fallback := cache.NewManagerWithCache(cache.NewMemoryCache())
		applyCacheSettingsFromConfig(fallback, cfg)
		return fallback
	}
	logger.Infof("Connected to Redis at %s:%d", cfg.Redis.Host, cfg.Redis.Port)

	applyCacheSettingsFromConfig(cacheManager, cfg)

	// Initialize Redis Stack vector store when Redis backend is available.
	if cfg.VectorCache.Enabled {
		if rc, ok := cacheManager.Cache().(*cache.RedisCache); ok {
			vectorCfg := cache.DefaultRedisStackVectorConfig()
			vectorCfg.Enabled = cfg.VectorCache.Enabled
			vectorCfg.IndexName = cfg.VectorCache.IndexName
			vectorCfg.KeyPrefix = cfg.VectorCache.KeyPrefix
			vectorCfg.Dimension = cfg.VectorCache.Dimension
			vectorCfg.QueryTimeout = time.Duration(cfg.VectorCache.QueryTimeoutMs) * time.Millisecond

			hotStore := cache.NewRedisStackVectorStoreFromRedisCache(rc, vectorCfg)
			if err := hotStore.EnsureIndex(context.Background()); err != nil {
				logger.WithError(err).Warn("Failed to ensure Redis Stack vector index, vector cache disabled")
				cacheManager.SetVectorStore(nil)
				return cacheManager
			}

			coldStores := map[string]cache.ColdVectorStore{}
			sqliteStore, err := cache.NewSQLiteColdVectorStore(cache.SQLiteColdVectorStoreConfig{
				Path: cfg.VectorCache.ColdVectorSQLitePath,
			})
			if err != nil {
				logger.WithError(err).Warn("Failed to initialize sqlite cold vector store")
			} else {
				coldStores[cache.ColdVectorBackendSQLite] = sqliteStore
			}

			qdrantStore := cache.NewQdrantColdVectorStore(cache.QdrantColdVectorStoreConfig{
				URL:        cfg.VectorCache.ColdVectorQdrantURL,
				APIKey:     cfg.VectorCache.ColdVectorQdrantAPIKey,
				Collection: cfg.VectorCache.ColdVectorQdrantCollection,
				Timeout:    time.Duration(cfg.VectorCache.ColdVectorQdrantTimeoutMs) * time.Millisecond,
				Dimension:  cfg.VectorCache.Dimension,
			})
			coldStores[cache.ColdVectorBackendQdrant] = qdrantStore

			tieredCfg := cache.TieredConfigFromSettings(cacheManager.GetSettings())
			tieredStore := cache.NewTieredVectorStore(hotStore, coldStores, tieredCfg)
			_ = tieredStore.EnsureIndex(context.Background())
			tieredStore.StartHotToColdWorker(context.Background())
			cacheManager.SetTieredVectorStore(tieredStore)

			logger.WithFields(logrus.Fields{
				"index":         vectorCfg.IndexName,
				"dimension":     vectorCfg.Dimension,
				"cold_enabled":  tieredCfg.ColdVectorEnabled,
				"cold_backend":  tieredCfg.ColdVectorBackend,
				"cold_dual":     tieredCfg.ColdVectorDualWriteEnabled,
				"cold_query":    tieredCfg.ColdVectorQueryEnabled,
				"hot_watermark": tieredCfg.HotMemoryHighWatermarkPercent,
			}).Info("Redis vector tier cache initialized")
		}
	}

	return cacheManager
}

func applyCacheSettingsFromConfig(cacheManager *cache.Manager, cfg *config.Config) {
	if cacheManager == nil || cfg == nil {
		return
	}

	settings := cacheManager.GetSettings()
	settings.VectorEnabled = cfg.VectorCache.Enabled
	settings.VectorDimension = cfg.VectorCache.Dimension
	settings.VectorQueryTimeoutMs = cfg.VectorCache.QueryTimeoutMs
	settings.VectorPipelineEnabled = cfg.VectorCache.PipelineEnabled
	settings.VectorStandardKeyVersion = cfg.VectorCache.StandardKeyVersion
	settings.VectorEmbeddingProvider = cfg.VectorCache.EmbeddingProvider
	settings.VectorOllamaBaseURL = cfg.VectorCache.OllamaBaseURL
	settings.VectorOllamaEmbeddingModel = cfg.VectorCache.OllamaEmbeddingModel
	settings.VectorOllamaEmbeddingDimension = cfg.VectorCache.OllamaEmbeddingDimension
	settings.VectorOllamaEmbeddingTimeoutMs = cfg.VectorCache.OllamaEmbeddingTimeoutMs
	settings.VectorOllamaEndpointMode = cfg.VectorCache.OllamaEndpointMode
	settings.VectorWritebackEnabled = cfg.VectorCache.WritebackEnabled
	settings.ColdVectorEnabled = cfg.VectorCache.ColdVectorEnabled
	settings.ColdVectorQueryEnabled = cfg.VectorCache.ColdVectorQueryEnabled
	settings.ColdVectorBackend = cfg.VectorCache.ColdVectorBackend
	settings.ColdVectorDualWriteEnabled = cfg.VectorCache.ColdVectorDualWriteEnabled
	settings.ColdVectorSimilarityThreshold = cfg.VectorCache.ColdVectorSimilarityThreshold
	settings.ColdVectorTopK = cfg.VectorCache.ColdVectorTopK
	settings.HotMemoryHighWatermarkPercent = cfg.VectorCache.HotMemoryHighWatermarkPercent
	settings.HotMemoryReliefPercent = cfg.VectorCache.HotMemoryReliefPercent
	settings.HotToColdBatchSize = cfg.VectorCache.HotToColdBatchSize
	settings.HotToColdIntervalSeconds = cfg.VectorCache.HotToColdIntervalSeconds
	settings.ColdVectorQdrantURL = cfg.VectorCache.ColdVectorQdrantURL
	settings.ColdVectorQdrantAPIKey = cfg.VectorCache.ColdVectorQdrantAPIKey
	settings.ColdVectorQdrantCollection = cfg.VectorCache.ColdVectorQdrantCollection
	settings.ColdVectorQdrantTimeoutMs = cfg.VectorCache.ColdVectorQdrantTimeoutMs
	if len(cfg.VectorCache.Thresholds) > 0 {
		settings.VectorThresholds = make(map[string]float64, len(cfg.VectorCache.Thresholds))
		for k, v := range cfg.VectorCache.Thresholds {
			settings.VectorThresholds[k] = v
		}
	}
	cacheManager.UpdateSettings(settings)
}

func StartMetricsServer(logger *logrus.Logger) *http.Server {
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

	return metricsSrv
}
