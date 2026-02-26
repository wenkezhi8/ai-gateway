package alert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifier_IsLevelEnabled(t *testing.T) {
	cfg := &NotifierConfig{
		EnabledLevels: []AlertLevel{AlertLevelCritical, AlertLevelWarning},
	}
	notifier := NewNotifier(cfg)

	assert.True(t, notifier.isLevelEnabled(AlertLevelCritical))
	assert.True(t, notifier.isLevelEnabled(AlertLevelWarning))
	assert.False(t, notifier.isLevelEnabled(AlertLevelInfo))
}

func TestNotifier_Send_LevelNotEnabled(t *testing.T) {
	cfg := &NotifierConfig{
		EnabledLevels: []AlertLevel{AlertLevelCritical},
	}
	notifier := NewNotifier(cfg)

	alert := Alert{
		Name:    "test-alert",
		Level:   AlertLevelInfo,
		Message: "Test message",
	}

	err := notifier.Send(alert)
	assert.NoError(t, err)
}

func TestNotifier_SendDingTalk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var msg DingTalkMessage
		err := json.NewDecoder(r.Body).Decode(&msg)
		require.NoError(t, err)
		assert.Equal(t, "markdown", msg.MsgType)
		assert.NotNil(t, msg.Markdown)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 0, "errmsg": "ok"})
	}))
	defer server.Close()

	cfg := &NotifierConfig{
		DingTalkWebhook: server.URL,
		EnabledLevels:   []AlertLevel{AlertLevelCritical, AlertLevelWarning, AlertLevelInfo},
	}
	notifier := NewNotifier(cfg)

	alert := Alert{
		Name:        "Test Alert",
		Level:       AlertLevelWarning,
		Message:     "Test message",
		StartsAt:    time.Now(),
		Labels:      map[string]string{"env": "test"},
		Annotations: map[string]string{"detail": "test detail"},
	}

	err := notifier.sendDingTalk(alert)
	assert.NoError(t, err)
}

func TestBuildDingTalkWebhookURL_WithSecret(t *testing.T) {
	base := "https://oapi.dingtalk.com/robot/send?access_token=test-token"
	secret := "sec-test-secret"
	now := time.UnixMilli(1700000000000)

	signedURL, err := buildDingTalkWebhookURL(base, secret, now)
	require.NoError(t, err)

	parsed, err := url.Parse(signedURL)
	require.NoError(t, err)
	query := parsed.Query()
	assert.Equal(t, strconv.FormatInt(now.UnixMilli(), 10), query.Get("timestamp"))
	assert.NotEmpty(t, query.Get("sign"))
}

func TestNotifier_SendDingTalk_WithSecretAddsSignatureQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.NotEmpty(t, q.Get("timestamp"))
		assert.NotEmpty(t, q.Get("sign"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 0, "errmsg": "ok"})
	}))
	defer server.Close()

	cfg := &NotifierConfig{
		DingTalkWebhook: server.URL,
		DingTalkSecret:  "sec-test-secret",
		EnabledLevels:   []AlertLevel{AlertLevelWarning},
	}
	notifier := NewNotifier(cfg)

	alert := Alert{
		Name:     "Test Alert",
		Level:    AlertLevelWarning,
		Message:  "Test message",
		StartsAt: time.Now(),
	}

	err := notifier.sendDingTalk(alert)
	assert.NoError(t, err)
}

func TestNotifier_SendDingTalk_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 1, "errmsg": "error"})
	}))
	defer server.Close()

	cfg := &NotifierConfig{
		DingTalkWebhook: server.URL,
		EnabledLevels:   []AlertLevel{AlertLevelWarning},
	}
	notifier := NewNotifier(cfg)

	alert := Alert{
		Name:     "Test Alert",
		Level:    AlertLevelWarning,
		Message:  "Test message",
		StartsAt: time.Now(),
	}

	err := notifier.sendDingTalk(alert)
	assert.Error(t, err)
}

func TestNotifier_Send_NoConfig(t *testing.T) {
	cfg := &NotifierConfig{
		EnabledLevels: []AlertLevel{AlertLevelWarning},
	}
	notifier := NewNotifier(cfg)

	alert := Alert{
		Name:     "Test Alert",
		Level:    AlertLevelWarning,
		Message:  "Test message",
		StartsAt: time.Now(),
	}

	err := notifier.Send(alert)
	assert.NoError(t, err)
}

func TestNotifier_GetLevelEmoji(t *testing.T) {
	cfg := &NotifierConfig{}
	notifier := NewNotifier(cfg)

	assert.Equal(t, "\U0001F534", notifier.getLevelEmoji(AlertLevelCritical))
	assert.Equal(t, "\U0001F7E1", notifier.getLevelEmoji(AlertLevelWarning))
	assert.Equal(t, "\U0001F535", notifier.getLevelEmoji(AlertLevelInfo))
	assert.Equal(t, "\u26A0\uFE0F", notifier.getLevelEmoji(AlertLevel("unknown")))
}

func TestFormatAlertFromPrometheus_Resolved(t *testing.T) {
	payload := map[string]interface{}{
		"status": "resolved",
		"alerts": []interface{}{
			map[string]interface{}{
				"labels": map[string]interface{}{
					"alertname": "TestAlert",
					"severity":  "warning",
				},
				"annotations": map[string]interface{}{
					"summary": "Test summary",
				},
				"startsAt": "2024-01-01T00:00:00Z",
			},
		},
	}

	alerts, err := FormatAlertFromPrometheus(payload)
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	assert.NotNil(t, alerts[0].EndsAt)
}

func TestFormatAlertFromPrometheus_MissingSeverity(t *testing.T) {
	payload := map[string]interface{}{
		"status": "firing",
		"alerts": []interface{}{
			map[string]interface{}{
				"labels": map[string]interface{}{
					"alertname": "TestAlert",
				},
				"annotations": map[string]interface{}{
					"message": "Test",
				},
			},
		},
	}

	alerts, err := FormatAlertFromPrometheus(payload)
	require.NoError(t, err)
	assert.Equal(t, AlertLevelInfo, alerts[0].Level)
}

func TestFormatAlertFromPrometheus_MissingStartsAt(t *testing.T) {
	payload := map[string]interface{}{
		"status": "firing",
		"alerts": []interface{}{
			map[string]interface{}{
				"labels": map[string]interface{}{
					"alertname": "TestAlert",
					"severity":  "warning",
				},
				"annotations": map[string]interface{}{},
			},
		},
	}

	alerts, err := FormatAlertFromPrometheus(payload)
	require.NoError(t, err)
	assert.False(t, alerts[0].StartsAt.IsZero())
}

func TestAlert_Levels(t *testing.T) {
	assert.Equal(t, AlertLevel("critical"), AlertLevelCritical)
	assert.Equal(t, AlertLevel("warning"), AlertLevelWarning)
	assert.Equal(t, AlertLevel("info"), AlertLevelInfo)
}

func TestDingTalkMessage_Marshal(t *testing.T) {
	msg := DingTalkMessage{
		MsgType: "text",
		Text: &DingTalkTextContent{
			Content: "Test message",
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded DingTalkMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "text", decoded.MsgType)
}

func TestNotifier_SendDingTalk_InvalidURL(t *testing.T) {
	cfg := &NotifierConfig{
		DingTalkWebhook: "://invalid-url",
		EnabledLevels:   []AlertLevel{AlertLevelWarning},
	}
	notifier := NewNotifier(cfg)

	alert := Alert{
		Name:     "Test Alert",
		Level:    AlertLevelWarning,
		Message:  "Test message",
		StartsAt: time.Now(),
	}

	err := notifier.sendDingTalk(alert)
	assert.Error(t, err)
}

func TestDingTalkMarkdownContent(t *testing.T) {
	content := DingTalkMarkdownContent{
		Title: "Test Title",
		Text:  "Test Text",
	}

	assert.Equal(t, "Test Title", content.Title)
	assert.Equal(t, "Test Text", content.Text)
}

func TestDingTalkTextContent(t *testing.T) {
	content := DingTalkTextContent{
		Content: "Test Content",
	}

	assert.Equal(t, "Test Content", content.Content)
}

func TestAlert_WithExtra(t *testing.T) {
	now := time.Now()
	endTime := now.Add(time.Hour)
	alert := Alert{
		Name:        "TestAlert",
		Level:       AlertLevelCritical,
		Message:     "Test message",
		Labels:      map[string]string{"env": "prod"},
		Annotations: map[string]string{"detail": "test"},
		StartsAt:    now,
		EndsAt:      &endTime,
		Extra:       map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "TestAlert", alert.Name)
	assert.Equal(t, AlertLevelCritical, alert.Level)
	assert.Equal(t, now, alert.StartsAt)
	assert.Equal(t, &endTime, alert.EndsAt)
	assert.NotNil(t, alert.Extra)
}
