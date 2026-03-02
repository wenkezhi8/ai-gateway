package admin

import (
	"errors"
	"testing"

	"ai-gateway/internal/config"
)

func TestCheckAllDependencies_AllHealthy(t *testing.T) {
	origRedis := redisProbe
	origOllama := ollamaProbe
	origQdrant := qdrantProbe
	defer func() {
		redisProbe = origRedis
		ollamaProbe = origOllama
		qdrantProbe = origQdrant
	}()

	redisProbe = func(_ config.RedisConfig) error { return nil }
	ollamaProbe = func(_ string) error { return nil }
	qdrantProbe = func(_ string) error { return nil }

	cfg := config.DefaultConfig()
	cfg.VectorCache.OllamaBaseURL = "http://127.0.0.1:11434"
	cfg.VectorCache.ColdVectorQdrantURL = "http://127.0.0.1:6333"

	status := checkAllDependencies(cfg)

	if len(status) != 3 {
		t.Fatalf("status count = %d, want 3", len(status))
	}
	if !status["redis"].Healthy || !status["ollama"].Healthy || !status["qdrant"].Healthy {
		t.Fatalf("expected all healthy, got: %#v", status)
	}
}

func TestCheckAllDependencies_QdrantMissingURL(t *testing.T) {
	origQdrant := qdrantProbe
	defer func() { qdrantProbe = origQdrant }()
	qdrantProbe = func(_ string) error { return errors.New("should not be called") }

	cfg := config.DefaultConfig()
	cfg.VectorCache.ColdVectorQdrantURL = ""

	status := checkAllDependencies(cfg)
	if status["qdrant"].Healthy {
		t.Fatalf("qdrant should be unhealthy when url missing")
	}
}
