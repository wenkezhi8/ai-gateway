package alert

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/webhook", handler.WebhookHandler)
	return router
}

func TestHandler_WebhookHandler_ValidPayload(t *testing.T) {
	notifier := NewNotifier(&NotifierConfig{
		EnabledLevels: []AlertLevel{AlertLevelCritical, AlertLevelWarning, AlertLevelInfo},
	})
	handler := NewHandler(notifier)

	router := newTestRouter(handler)

	// Create a valid Prometheus alert payload
	payload := map[string]interface{}{
		"alerts": []map[string]interface{}{
			{
				"status": "firing",
				"labels": map[string]interface{}{
					"alertname": "HighErrorRate",
					"severity":  "critical",
				},
				"annotations": map[string]interface{}{
					"summary":     "High error rate detected",
					"description": "Error rate is above 5%",
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed (or fail gracefully if notification not configured)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusPartialContent)
}

func TestHandler_WebhookHandler_InvalidJSON(t *testing.T) {
	notifier := NewNotifier(&NotifierConfig{
		EnabledLevels: []AlertLevel{AlertLevelCritical, AlertLevelWarning, AlertLevelInfo},
	})
	handler := NewHandler(notifier)

	router := newTestRouter(handler)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_WebhookHandler_EmptyBody(t *testing.T) {
	notifier := NewNotifier(&NotifierConfig{
		EnabledLevels: []AlertLevel{AlertLevelCritical, AlertLevelWarning, AlertLevelInfo},
	})
	handler := NewHandler(notifier)

	router := newTestRouter(handler)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Empty body should not cause error
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}

func TestAlert_Fields(t *testing.T) {
	alert := Alert{
		Name:    "TestAlert",
		Level:   AlertLevelCritical,
		Message: "Test summary",
		Labels:  map[string]string{"env": "prod"},
	}

	assert.Equal(t, "TestAlert", alert.Name)
	assert.Equal(t, AlertLevelCritical, alert.Level)
	assert.Equal(t, "Test summary", alert.Message)
	assert.Equal(t, map[string]string{"env": "prod"}, alert.Labels)
}

func TestNotifierConfig_Fields(t *testing.T) {
	cfg := NotifierConfig{
		DingTalkWebhook: "https://oapi.dingtalk.com/robot/send?access_token=xxx",
		DingTalkSecret:  "secret",
		SMTPHost:        "smtp.example.com",
		SMTPPort:        587,
		SMTPUser:        "user",
		SMTPPassword:    "pass",
		SMTPFrom:        "alerts@example.com",
		EmailTo:         "admin@example.com",
		EnabledLevels:   []AlertLevel{AlertLevelCritical, AlertLevelWarning},
	}

	assert.Equal(t, "https://oapi.dingtalk.com/robot/send?access_token=xxx", cfg.DingTalkWebhook)
	assert.Equal(t, "secret", cfg.DingTalkSecret)
	assert.Equal(t, "smtp.example.com", cfg.SMTPHost)
	assert.Equal(t, 587, cfg.SMTPPort)
	assert.Equal(t, "user", cfg.SMTPUser)
	assert.Equal(t, "pass", cfg.SMTPPassword)
	assert.Equal(t, "alerts@example.com", cfg.SMTPFrom)
	assert.Equal(t, "admin@example.com", cfg.EmailTo)
	assert.Equal(t, []AlertLevel{AlertLevelCritical, AlertLevelWarning}, cfg.EnabledLevels)
}

func TestFormatAlertFromPrometheus(t *testing.T) {
	// Create proper payload with []interface{} for alerts
	alertsRaw := []interface{}{
		map[string]interface{}{
			"status": "firing",
			"labels": map[string]interface{}{
				"alertname": "TestAlert",
				"severity":  "warning",
			},
			"annotations": map[string]interface{}{
				"summary":     "Test summary",
				"description": "Test description",
			},
		},
	}
	payload := map[string]interface{}{
		"alerts": alertsRaw,
	}

	alerts, err := FormatAlertFromPrometheus(payload)
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	assert.Equal(t, "TestAlert", alerts[0].Name)
	assert.Equal(t, AlertLevelWarning, alerts[0].Level)
	assert.Equal(t, "Test summary", alerts[0].Message)
}

func TestFormatAlertFromPrometheus_EmptyAlerts(t *testing.T) {
	payload := map[string]interface{}{
		"alerts": []interface{}{},
	}

	alerts, err := FormatAlertFromPrometheus(payload)
	require.NoError(t, err)
	assert.Len(t, alerts, 0)
}

func TestFormatAlertFromPrometheus_NoAlertsField(t *testing.T) {
	payload := map[string]interface{}{
		"status": "ok",
	}

	_, err := FormatAlertFromPrometheus(payload)
	assert.Error(t, err)
}

func TestNewNotifier(t *testing.T) {
	cfg := &NotifierConfig{
		EnabledLevels: []AlertLevel{AlertLevelCritical, AlertLevelWarning, AlertLevelInfo},
	}

	notifier := NewNotifier(cfg)
	assert.NotNil(t, notifier)
}

func TestNewNotifier_DefaultLevels(t *testing.T) {
	cfg := &NotifierConfig{}

	notifier := NewNotifier(cfg)
	assert.NotNil(t, notifier)
	assert.NotNil(t, cfg.EnabledLevels) // Should be set to default
}
