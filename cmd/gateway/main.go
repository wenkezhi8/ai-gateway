package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ai-gateway/internal/bootstrap"
	"ai-gateway/internal/config"
	"ai-gateway/internal/middleware"
	"ai-gateway/internal/router"
)

var jwtSecret string

type serverRuntimeConfig struct {
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	shutdownTimeout   time.Duration
	maxHeaderBytes    int
}

func main() {
	logger := bootstrap.NewLogger()

	cfg, loadErr := config.Load()
	if loadErr != nil {
		logger.WithError(loadErr).Fatal("Failed to load configuration")
	}
	if err := cfg.Validate(); err != nil {
		logger.WithError(err).Fatal("Invalid configuration")
	}
	bootstrap.ApplyCORSAllowOriginsFromConfig(cfg)

	configWatcher := bootstrap.SetupConfigWatcher(cfg, logger)
	if configWatcher != nil {
		defer configWatcher.Close()
	}

	auditLogger := bootstrap.InitAuditLogger()
	defer auditLogger.Close()

	registry := bootstrap.InitProviderRegistry(cfg, logger)
	accountManager := bootstrap.InitAccountManager(cfg, logger)
	cacheManager, cacheErr := bootstrap.InitCacheManager(cfg, logger)
	if cacheErr != nil {
		logger.WithError(cacheErr).Fatal("Failed to initialize cache manager")
	}
	metricsSrv := bootstrap.StartMetricsServer(logger)

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

	r := router.NewFullWithConfig(cfg, &routerCfg, accountManager, cacheManager, registry)

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Server.Port
	}

	runtimeCfg, runtimeWarnings := loadServerRuntimeConfig(os.Getenv)
	for _, warning := range runtimeWarnings {
		logger.Warnf("Runtime config warning: %s", warning)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadTimeout:       runtimeCfg.readTimeout,
		ReadHeaderTimeout: runtimeCfg.readHeaderTimeout,
		WriteTimeout:      runtimeCfg.writeTimeout,
		IdleTimeout:       runtimeCfg.idleTimeout,
		MaxHeaderBytes:    runtimeCfg.maxHeaderBytes,
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), runtimeCfg.shutdownTimeout)
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

func loadServerRuntimeConfig(getenv func(string) string) (cfg serverRuntimeConfig, warnings []string) {
	cfg = serverRuntimeConfig{
		readTimeout:       15 * time.Second,
		readHeaderTimeout: 10 * time.Second,
		writeTimeout:      30 * time.Second,
		idleTimeout:       60 * time.Second,
		shutdownTimeout:   30 * time.Second,
		maxHeaderBytes:    1 << 20,
	}
	warnings = make([]string, 0)

	cfg.readTimeout, warnings = parseEnvDuration(getenv, "SERVER_READ_TIMEOUT", cfg.readTimeout, warnings)
	cfg.readHeaderTimeout, warnings = parseEnvDuration(getenv, "SERVER_READ_HEADER_TIMEOUT", cfg.readHeaderTimeout, warnings)
	cfg.writeTimeout, warnings = parseEnvDuration(getenv, "SERVER_WRITE_TIMEOUT", cfg.writeTimeout, warnings)
	cfg.idleTimeout, warnings = parseEnvDuration(getenv, "SERVER_IDLE_TIMEOUT", cfg.idleTimeout, warnings)
	cfg.shutdownTimeout, warnings = parseEnvDuration(getenv, "SERVER_SHUTDOWN_TIMEOUT", cfg.shutdownTimeout, warnings)
	cfg.maxHeaderBytes, warnings = parseEnvInt(getenv, "SERVER_MAX_HEADER_BYTES", cfg.maxHeaderBytes, warnings)

	return cfg, warnings
}

func parseEnvDuration(getenv func(string) string, key string, fallback time.Duration, warnings []string) (duration time.Duration, updatedWarnings []string) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback, warnings
	}
	v, err := time.ParseDuration(raw)
	if err != nil || v <= 0 {
		warnings = append(warnings, fmt.Sprintf("invalid %s=%q, fallback=%s", key, raw, fallback))
		return fallback, warnings
	}
	return v, warnings
}

func parseEnvInt(getenv func(string) string, key string, fallback int, warnings []string) (value int, updatedWarnings []string) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback, warnings
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		warnings = append(warnings, fmt.Sprintf("invalid %s=%q, fallback=%d", key, raw, fallback))
		return fallback, warnings
	}
	return v, warnings
}
