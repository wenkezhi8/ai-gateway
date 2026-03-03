package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-gateway/internal/config"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type usageRuntimeMetaRecord struct {
	Account            string
	UserAgent          string
	RequestType        string
	InferenceIntensity string
}

func TestProxyHandler_ChatCompletions_ShouldPersistUsageMetaForNonStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	accountManager := limiter.NewAccountManager(nil, nil)
	require.NoError(t, accountManager.AddAccount(&limiter.AccountConfig{
		ID:       "usage-meta-acc-1",
		Name:     "Meta Account",
		Provider: "meta-provider",
		APIKey:   "test-key",
		BaseURL:  "https://api.openai.com/v1",
		Enabled:  true,
		Limits:   map[limiter.LimitType]*limiter.LimitConfig{},
	}))

	h := NewProxyHandler(&config.Config{}, accountManager, nil)
	h.registry.Register("openai", &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"usage-meta-non-stream-model"}, true),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("scheduler_account_id", "usage-meta-acc-1")

	body := `{"provider":"openai","model":"usage-meta-non-stream-model","messages":[{"role":"user","content":"hello usage meta"}],"reasoning_effort":"medium"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "UsageMetaNonStream/1.0")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)

	meta := fetchLatestUsageRuntimeMetaByModel(t, storage.GetSQLiteStorage().GetDB(), "usage-meta-non-stream-model")
	assert.Equal(t, "Meta Account", meta.Account)
	assert.Equal(t, "UsageMetaNonStream/1.0", meta.UserAgent)
	assert.Equal(t, "non_stream", meta.RequestType)
	assert.Equal(t, "medium", meta.InferenceIntensity)
}

func TestProxyHandler_ChatCompletions_ShouldPersistUsageMetaForStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	h.registry.Register("openai", &doneStreamProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"usage-meta-stream-model"}, true),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"provider":"openai","model":"usage-meta-stream-model","stream":true,"messages":[{"role":"user","content":"hello stream usage"}],"reasoning_effort":"xhigh"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "UsageMetaStream/1.0")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)

	meta := fetchLatestUsageRuntimeMetaByModel(t, storage.GetSQLiteStorage().GetDB(), "usage-meta-stream-model")
	assert.Equal(t, "UsageMetaStream/1.0", meta.UserAgent)
	assert.Equal(t, "stream", meta.RequestType)
	assert.Equal(t, "xhigh", meta.InferenceIntensity)
}

func TestProxyHandler_ChatCompletions_ShouldInferHighInferenceIntensityFromDeepThink(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	h.registry.Register("openai", &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"usage-meta-deep-think-model"}, true),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"provider":"openai","model":"usage-meta-deep-think-model","messages":[{"role":"user","content":"hello deep think"}],"deepThink":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "UsageMetaDeepThink/1.0")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)

	meta := fetchLatestUsageRuntimeMetaByModel(t, storage.GetSQLiteStorage().GetDB(), "usage-meta-deep-think-model")
	assert.Equal(t, "high", meta.InferenceIntensity)
	assert.Equal(t, "non_stream", meta.RequestType)
}

func fetchLatestUsageRuntimeMetaByModel(t *testing.T, db *sql.DB, model string) usageRuntimeMetaRecord {
	t.Helper()

	var record usageRuntimeMetaRecord
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err := db.QueryRow(`SELECT account, user_agent, request_type, inference_intensity FROM usage_logs WHERE model = ? ORDER BY id DESC LIMIT 1`, model).
			Scan(&record.Account, &record.UserAgent, &record.RequestType, &record.InferenceIntensity)
		if err == nil {
			return record
		}
		if err == sql.ErrNoRows {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		t.Fatalf("query usage runtime meta failed for model %s: %v", model, err)
	}

	t.Fatalf("no usage runtime meta found for model %s", model)
	return usageRuntimeMetaRecord{}
}
