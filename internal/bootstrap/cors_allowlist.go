package bootstrap

import (
	"os"
	"strings"

	"ai-gateway/internal/config"
)

// ApplyCORSAllowOriginsFromConfig keeps env-first behavior and uses config as fallback.
func ApplyCORSAllowOriginsFromConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}

	if strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS")) != "" {
		return
	}

	corsAllowOrigins := strings.TrimSpace(cfg.Server.CORSAllowOrigins)
	if corsAllowOrigins == "" {
		return
	}

	_ = os.Setenv("CORS_ALLOW_ORIGINS", corsAllowOrigins)
}
