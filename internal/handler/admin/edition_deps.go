package admin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/config"

	"github.com/redis/go-redis/v9"
)

type DependencyStatus struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message"`
}

const defaultOllamaBaseURL = "http://127.0.0.1:11434"

var (
	redisProbe = func(redisCfg config.RedisConfig) error {
		client := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
			Password: redisCfg.Password,
			DB:       redisCfg.DB,
		})
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return client.Ping(ctx).Err()
	}

	ollamaProbe = func(baseURL string) error {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/api/tags")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code %d", resp.StatusCode)
		}
		return nil
	}

	qdrantProbe = func(baseURL string) error {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/collections")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code %d", resp.StatusCode)
		}
		return nil
	}
)

func checkAllDependencies(cfg *config.Config) map[string]DependencyStatus {
	status := map[string]DependencyStatus{}

	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	if err := redisProbe(cfg.Redis); err != nil {
		status["redis"] = DependencyStatus{Name: "Redis", Address: redisAddr, Healthy: false, Message: err.Error()}
	} else {
		status["redis"] = DependencyStatus{Name: "Redis", Address: redisAddr, Healthy: true, Message: "正常"}
	}

	ollamaURL := strings.TrimSpace(cfg.VectorCache.OllamaBaseURL)
	if ollamaURL == "" {
		ollamaURL = defaultOllamaBaseURL
	}
	if err := ollamaProbe(ollamaURL); err != nil {
		status["ollama"] = DependencyStatus{Name: "Ollama", Address: ollamaURL, Healthy: false, Message: err.Error()}
	} else {
		status["ollama"] = DependencyStatus{Name: "Ollama", Address: ollamaURL, Healthy: true, Message: "正常"}
	}

	qdrantURL := strings.TrimSpace(cfg.VectorCache.ColdVectorQdrantURL)
	if qdrantURL == "" {
		status["qdrant"] = DependencyStatus{Name: "Qdrant", Address: "未配置", Healthy: false, Message: "Qdrant URL 未配置"}
		return status
	}
	if err := qdrantProbe(qdrantURL); err != nil {
		status["qdrant"] = DependencyStatus{Name: "Qdrant", Address: qdrantURL, Healthy: false, Message: err.Error()}
	} else {
		status["qdrant"] = DependencyStatus{Name: "Qdrant", Address: qdrantURL, Healthy: true, Message: "正常"}
	}

	return status
}
