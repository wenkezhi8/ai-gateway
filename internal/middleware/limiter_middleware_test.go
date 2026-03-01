package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/limiter"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLimiterMiddleware(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)
	require.NotNil(t, mw)
}

func TestLimiterMiddleware_RateLimit_NoProvider(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)

	handler := mw.RateLimit()
	handler(c)

	assert.False(t, c.IsAborted())
}

func TestLimiterMiddleware_RateLimit_WithProvider(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	err := manager.AddAccount(&limiter.AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
	})
	require.NoError(t, err)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	c.Set("provider", "openai")

	handler := mw.RateLimit()
	handler(c)

	assert.False(t, c.IsAborted())
	_, exists := c.Get("active_account")
	assert.True(t, exists)
}

func TestLimiterMiddleware_RateLimit_InvalidProvider(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	c.Set("provider", 123)

	handler := mw.RateLimit()
	handler(c)

	assert.False(t, c.IsAborted())
}

func TestLimiterMiddleware_UsageTracking(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)

	handler := mw.UsageTracking()
	handler(c)

	assert.False(t, c.IsAborted())
}

func TestLimiterMiddleware_UsageTracking_WithAccount(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	err := manager.AddAccount(&limiter.AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
	})
	require.NoError(t, err)

	account, err := manager.GetActiveAccount("openai")
	require.NoError(t, err)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	c.Set("active_account", account)

	handler := mw.UsageTracking()
	handler(c)

	assert.False(t, c.IsAborted())
}

func TestLimiterMiddleware_AccountInfo(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	err := manager.AddAccount(&limiter.AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
	})
	require.NoError(t, err)

	account, err := manager.GetActiveAccount("openai")
	require.NoError(t, err)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	c.Set("active_account", account)

	handler := mw.AccountInfo()
	handler(c)

	assert.False(t, c.IsAborted())
	assert.Equal(t, "acc1", w.Header().Get("X-Account-ID"))
}

func TestLimiterMiddleware_AccountInfo_NoAccount(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)

	handler := mw.AccountInfo()
	handler(c)

	assert.False(t, c.IsAborted())
	assert.Empty(t, w.Header().Get("X-Account-ID"))
}

func TestLimiterMiddleware_AccountInfo_InvalidType(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	c.Set("active_account", "invalid")

	handler := mw.AccountInfo()
	handler(c)

	assert.False(t, c.IsAborted())
}

func TestLimiterMiddleware_RequireAccount_NoProvider(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)

	handler := mw.RequireAccount()
	handler(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLimiterMiddleware_RequireAccount_WithHeader(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	err := manager.AddAccount(&limiter.AccountConfig{
		ID:       "acc1",
		Provider: "openai",
		Enabled:  true,
	})
	require.NoError(t, err)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	c.Request.Header.Set("X-Provider", "openai")

	handler := mw.RequireAccount()
	handler(c)

	assert.False(t, c.IsAborted())
	_, exists := c.Get("active_account")
	assert.True(t, exists)
}

func TestLimiterMiddleware_RequireAccount_NoAccount(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	c.Request.Header.Set("X-Provider", "nonexistent")

	handler := mw.RequireAccount()
	handler(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestLimiterMiddleware_GetSwitchHistory(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test?limit=10", http.NoBody)

	handler := mw.GetSwitchHistory()
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLimiterMiddleware_ForceSwitch_InvalidRequest(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", http.NoBody)

	handler := mw.ForceSwitch()
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLimiterMiddleware_GetAccountStatus_NoID(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)

	handler := mw.GetAccountStatus()
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLimiterMiddleware_GetAccountStatus_NotFound(t *testing.T) {
	logger := logrus.New()
	manager := limiter.NewAccountManager(nil, logger)

	mw := NewLimiterMiddleware(manager, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	c.Params = gin.Params{{Key: "account_id", Value: "nonexistent"}}

	handler := mw.GetAccountStatus()
	handler(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
