package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-gateway/internal/bootstrap"
	"ai-gateway/internal/config"
	"ai-gateway/internal/middleware"
	"ai-gateway/internal/router"
)

var jwtSecret string

func main() {
	logger := bootstrap.NewLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}
	if err := cfg.Validate(); err != nil {
		logger.WithError(err).Fatal("Invalid configuration")
	}

	configWatcher := bootstrap.SetupConfigWatcher(cfg, logger)
	if configWatcher != nil {
		defer configWatcher.Close()
	}

	auditLogger := bootstrap.InitAuditLogger()
	defer auditLogger.Close()

	registry := bootstrap.InitProviderRegistry(cfg, logger)
	accountManager := bootstrap.InitAccountManager(cfg, logger)
	cacheManager := bootstrap.InitCacheManager(cfg, logger)
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
