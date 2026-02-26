package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProviderModels(t *testing.T) {
	tests := []struct {
		provider   string
		wantModels int
	}{
		{"openai", 9},
		{"anthropic", 5},
		{"deepseek", 3},
		{"qwen", 7},
		{"zhipu", 8},
		{"moonshot", 3},
		{"minimax", 4},
		{"baichuan", 3},
		{"volcengine", 3},
		{"ernie", 3},
		{"yi", 4},
		{"google", 3},
		{"mistral", 3},
		{"hunyuan", 3},
		{"spark", 2},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			models := GetProviderModels(tt.provider)
			assert.Len(t, models, tt.wantModels)
			if tt.wantModels > 0 {
				assert.NotEmpty(t, models[0])
			}
		})
	}
}

func TestGetProviderBaseURL(t *testing.T) {
	tests := []struct {
		provider     string
		wantURL      string
		wantNotEmpty bool
	}{
		{"openai", "https://api.openai.com/v1", true},
		{"anthropic", "https://api.anthropic.com/v1", true},
		{"deepseek", "https://api.deepseek.com", true},
		{"qwen", "https://dashscope.aliyuncs.com/compatible-mode/v1", true},
		{"zhipu", "https://open.bigmodel.cn/api/paas/v4", true},
		{"moonshot", "https://api.moonshot.cn/v1", true},
		{"minimax", "https://api.minimax.chat/v1", true},
		{"baichuan", "https://api.baichuan-ai.com/v1", true},
		{"volcengine", "https://ark.cn-beijing.volces.com/api/v3", true},
		{"ernie", "https://aip.baidubce.com/rpc/2.0/ai_custom/v1", true},
		{"yi", "https://api.lingyiwanwu.com/v1", true},
		{"google", "https://generativelanguage.googleapis.com/v1beta", true},
		{"mistral", "https://api.mistral.ai/v1", true},
		{"hunyuan", "https://hunyuan.tencentcloudapi.com", true},
		{"spark", "https://spark-api-open.xf-yun.com/v1", true},
		{"nonexistent", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			url := GetProviderBaseURL(tt.provider)
			if tt.wantNotEmpty {
				assert.Equal(t, tt.wantURL, url)
			} else {
				assert.Empty(t, url)
			}
		})
	}
}

func TestIsProviderSupported(t *testing.T) {
	tests := []struct {
		provider string
		want     bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"deepseek", true},
		{"qwen", true},
		{"zhipu", true},
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			supported := IsProviderSupported(tt.provider)
			assert.Equal(t, tt.want, supported)
		})
	}
}

func TestGetAllProviders(t *testing.T) {
	providers := GetAllProviders()

	assert.GreaterOrEqual(t, len(providers), 15)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
	assert.Contains(t, providers, "deepseek")
	assert.Contains(t, providers, "qwen")
	assert.Contains(t, providers, "zhipu")
}

func TestAccountRecord_Fields(t *testing.T) {
	acc := &AccountRecord{
		ID:         "test-id",
		Provider:   "openai",
		APIKey:     "sk-test",
		Priority:   1,
		Enabled:    true,
		QuotaLimit: 100000,
		QuotaUsed:  50000,
	}

	assert.Equal(t, "test-id", acc.ID)
	assert.Equal(t, "openai", acc.Provider)
	assert.Equal(t, "sk-test", acc.APIKey)
	assert.Equal(t, 1, acc.Priority)
	assert.True(t, acc.Enabled)
	assert.Equal(t, int64(100000), acc.QuotaLimit)
	assert.Equal(t, int64(50000), acc.QuotaUsed)
}

func TestModelScoreRecord_Fields(t *testing.T) {
	score := &ModelScoreRecord{
		Model:        "gpt-4",
		Provider:     "openai",
		QualityScore: 85,
		SpeedScore:   90,
		CostScore:    70,
		Enabled:      true,
		IsCustom:     true,
	}

	assert.Equal(t, "gpt-4", score.Model)
	assert.Equal(t, "openai", score.Provider)
	assert.Equal(t, 85, score.QualityScore)
	assert.Equal(t, 90, score.SpeedScore)
	assert.Equal(t, 70, score.CostScore)
	assert.True(t, score.Enabled)
	assert.True(t, score.IsCustom)
}

func TestRouterConfigRecord_Fields(t *testing.T) {
	config := &RouterConfigRecord{
		DefaultStrategy: "auto",
		DefaultModel:    "gpt-4",
		UseAutoMode:     true,
	}

	assert.Equal(t, "auto", config.DefaultStrategy)
	assert.Equal(t, "gpt-4", config.DefaultModel)
	assert.True(t, config.UseAutoMode)
}

func TestAPIKeyRecord_Fields(t *testing.T) {
	key := &APIKeyRecord{
		ID:          "key-1",
		Name:        "Test Key",
		Key:         "sk-test-key",
		Permissions: "read,write",
		Enabled:     true,
		LastUsedAt:  "2024-01-01T00:00:00Z",
		CreatedAt:   "2024-01-01T00:00:00Z",
		ExpiresAt:   "2025-01-01T00:00:00Z",
	}

	assert.Equal(t, "key-1", key.ID)
	assert.Equal(t, "Test Key", key.Name)
	assert.Equal(t, "sk-test-key", key.Key)
	assert.Equal(t, "read,write", key.Permissions)
	assert.True(t, key.Enabled)
	assert.NotEmpty(t, key.LastUsedAt)
	assert.NotEmpty(t, key.CreatedAt)
	assert.NotEmpty(t, key.ExpiresAt)
}

func TestUserRecord_Fields(t *testing.T) {
	user := &UserRecord{
		Username:     "admin",
		PasswordHash: "hashed-password",
		Role:         "admin",
		Email:        "admin@example.com",
		CreatedAt:    "2024-01-01T00:00:00Z",
		UpdatedAt:    "2024-01-01T00:00:00Z",
	}

	assert.Equal(t, "admin", user.Username)
	assert.Equal(t, "hashed-password", user.PasswordHash)
	assert.Equal(t, "admin", user.Role)
	assert.Equal(t, "admin@example.com", user.Email)
	assert.NotEmpty(t, user.CreatedAt)
	assert.NotEmpty(t, user.UpdatedAt)
}

func TestFeedbackRecord_Fields(t *testing.T) {
	feedback := &FeedbackRecord{
		ID:         "feedback-1",
		RequestID:  "req-1",
		Model:      "gpt-4",
		Provider:   "openai",
		TaskType:   "code",
		Rating:     5,
		Comment:    "Great!",
		LatencyMs:  1000,
		TokensUsed: 1000,
		CacheHit:   false,
		CreatedAt:  "2024-01-01T00:00:00Z",
	}

	assert.Equal(t, "feedback-1", feedback.ID)
	assert.Equal(t, "req-1", feedback.RequestID)
	assert.Equal(t, "gpt-4", feedback.Model)
	assert.Equal(t, "openai", feedback.Provider)
	assert.Equal(t, "code", feedback.TaskType)
	assert.Equal(t, 5, feedback.Rating)
	assert.Equal(t, "Great!", feedback.Comment)
	assert.Equal(t, int64(1000), feedback.LatencyMs)
	assert.Equal(t, int64(1000), feedback.TokensUsed)
	assert.False(t, feedback.CacheHit)
	assert.NotEmpty(t, feedback.CreatedAt)
}

func TestProviderModels_AllModelsHaveBaseURL(t *testing.T) {
	providers := GetAllProviders()

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			url := GetProviderBaseURL(provider)
			assert.NotEmpty(t, url, "Provider %s should have base URL", provider)
		})
	}
}

func TestProviderModels_ModelsNotDuplicated(t *testing.T) {
	seen := make(map[string]bool)

	for provider := range ProviderModels {
		models := GetProviderModels(provider)
		for _, model := range models {
			key := provider + ":" + model
			assert.False(t, seen[key], "Model %s:%s is duplicated", provider, model)
			seen[key] = true
		}
	}
}
