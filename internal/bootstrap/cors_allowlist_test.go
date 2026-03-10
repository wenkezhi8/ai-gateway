package bootstrap

import (
	"os"
	"testing"

	"ai-gateway/internal/config"
)

func TestApplyCORSAllowOriginsFromConfig_SetsEnvWhenUnset(t *testing.T) {
	prev, had := os.LookupEnv("CORS_ALLOW_ORIGINS")
	_ = os.Unsetenv("CORS_ALLOW_ORIGINS")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("CORS_ALLOW_ORIGINS", prev)
			return
		}
		_ = os.Unsetenv("CORS_ALLOW_ORIGINS")
	})

	cfg := &config.Config{
		Server: config.ServerConfig{
			CORSAllowOrigins: "https://console.example.com,https://ops.example.com",
		},
	}

	ApplyCORSAllowOriginsFromConfig(cfg)

	got := os.Getenv("CORS_ALLOW_ORIGINS")
	want := "https://console.example.com,https://ops.example.com"
	if got != want {
		t.Fatalf("CORS_ALLOW_ORIGINS=%q, want %q", got, want)
	}
}

func TestApplyCORSAllowOriginsFromConfig_KeepExplicitEnv(t *testing.T) {
	prev, had := os.LookupEnv("CORS_ALLOW_ORIGINS")
	_ = os.Setenv("CORS_ALLOW_ORIGINS", "https://explicit.example.com")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("CORS_ALLOW_ORIGINS", prev)
			return
		}
		_ = os.Unsetenv("CORS_ALLOW_ORIGINS")
	})

	cfg := &config.Config{
		Server: config.ServerConfig{
			CORSAllowOrigins: "https://config.example.com",
		},
	}

	ApplyCORSAllowOriginsFromConfig(cfg)

	if got := os.Getenv("CORS_ALLOW_ORIGINS"); got != "https://explicit.example.com" {
		t.Fatalf("CORS_ALLOW_ORIGINS=%q, want explicit env to win", got)
	}
}
