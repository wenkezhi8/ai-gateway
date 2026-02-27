package bootstrap

import (
	"ai-gateway/internal/config"
	"ai-gateway/internal/provider"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestInitProviderRegistry_RegistersGoogleFactory(t *testing.T) {
	provider.ClearRegistry()
	t.Cleanup(provider.ClearRegistry)

	cfg := &config.Config{}
	logger := logrus.New()

	registry := InitProviderRegistry(cfg, logger)
	_, err := registry.CreateProvider(&provider.ProviderConfig{
		Name:    "google",
		APIKey:  "test-key",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		Models:  []string{"gemini-3.1-pro-preview"},
		Enabled: true,
	})

	require.NoError(t, err)
}
