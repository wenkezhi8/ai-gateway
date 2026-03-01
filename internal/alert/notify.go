package alert

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// AlertLevel represents the severity level of an alert.
//
//nolint:revive // Kept for package API compatibility.
type AlertLevel string

const (
	AlertLevelCritical AlertLevel = "critical"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelInfo     AlertLevel = "info"
)

// Alert represents an alert event.
type Alert struct {
	Name        string                 `json:"name"`
	Level       AlertLevel             `json:"level"`
	Message     string                 `json:"message"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	StartsAt    time.Time              `json:"startsAt"`
	EndsAt      *time.Time             `json:"endsAt,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// NotifierConfig holds configuration for alert notifiers.
type NotifierConfig struct {
	// DingTalk configuration
	DingTalkWebhook string `json:"dingtalk_webhook"`
	DingTalkSecret  string `json:"dingtalk_secret"`

	// Email configuration
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from"`
	EmailTo      string `json:"email_to"`

	// General settings
	EnabledLevels []AlertLevel `json:"enabled_levels"`
}

// Notifier handles alert notifications.
type Notifier struct {
	config *NotifierConfig
	client *http.Client
}

// NewNotifier creates a new alert notifier.
func NewNotifier(cfg *NotifierConfig) *Notifier {
	if cfg.EnabledLevels == nil {
		cfg.EnabledLevels = []AlertLevel{AlertLevelCritical, AlertLevelWarning, AlertLevelInfo}
	}

	return &Notifier{
		config: cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send sends an alert through all configured channels.
//
//nolint:gocritic // Kept by-value for API compatibility.
func (n *Notifier) Send(alert Alert) error {
	alertCopy := alert
	return n.send(&alertCopy)
}

func (n *Notifier) send(alert *Alert) error {
	if !n.isLevelEnabled(alert.Level) {
		log.Printf("[Alert] Level %s not enabled, skipping: %s", alert.Level, alert.Name)
		return nil
	}

	var errors []string

	// Send to DingTalk
	if n.config.DingTalkWebhook != "" {
		if err := n.sendDingTalk(alert); err != nil {
			errors = append(errors, fmt.Sprintf("dingtalk: %v", err))
		}
	}

	// Send Email
	if n.config.SMTPHost != "" && n.config.EmailTo != "" {
		if err := n.sendEmail(alert); err != nil {
			errors = append(errors, fmt.Sprintf("email: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// isLevelEnabled checks if an alert level is enabled.
func (n *Notifier) isLevelEnabled(level AlertLevel) bool {
	for _, l := range n.config.EnabledLevels {
		if l == level {
			return true
		}
	}
	return false
}

// DingTalkMessage represents a DingTalk webhook message.
type DingTalkMessage struct {
	MsgType  string                   `json:"msgtype"`
	Markdown *DingTalkMarkdownContent `json:"markdown,omitempty"`
	Text     *DingTalkTextContent     `json:"text,omitempty"`
}

// DingTalkMarkdownContent represents markdown content for DingTalk.
type DingTalkMarkdownContent struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// DingTalkTextContent represents text content for DingTalk.
type DingTalkTextContent struct {
	Content string `json:"content"`
}

// sendDingTalk sends an alert to DingTalk.
func (n *Notifier) sendDingTalk(alert *Alert) error {
	levelEmoji := n.getLevelEmoji(alert.Level)
	title := fmt.Sprintf("%s [%s] %s", levelEmoji, strings.ToUpper(string(alert.Level)), alert.Name)

	var textBuilder strings.Builder
	textBuilder.WriteString(fmt.Sprintf("### %s\n\n", title))
	textBuilder.WriteString(fmt.Sprintf("**Level:** %s\n\n", alert.Level))
	textBuilder.WriteString(fmt.Sprintf("**Message:** %s\n\n", alert.Message))
	textBuilder.WriteString(fmt.Sprintf("**Time:** %s\n\n", alert.StartsAt.Format("2006-01-02 15:04:05")))

	if len(alert.Labels) > 0 {
		textBuilder.WriteString("**Labels:**\n\n")
		for k, v := range alert.Labels {
			textBuilder.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
		textBuilder.WriteString("\n")
	}

	if len(alert.Annotations) > 0 {
		textBuilder.WriteString("**Details:**\n\n")
		for k, v := range alert.Annotations {
			textBuilder.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
		textBuilder.WriteString("\n")
	}

	textBuilder.WriteString("\n---\n*AI Gateway Monitoring System*")

	msg := DingTalkMessage{
		MsgType: "markdown",
		Markdown: &DingTalkMarkdownContent{
			Title: title,
			Text:  textBuilder.String(),
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal dingtalk message: %w", err)
	}

	webhookURL, err := buildDingTalkWebhookURL(n.config.DingTalkWebhook, n.config.DingTalkSecret, time.Now())
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create dingtalk request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send dingtalk request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode dingtalk response: %w", err)
	}

	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return fmt.Errorf("dingtalk error: %v", result["errmsg"])
	}

	log.Printf("[Alert] DingTalk notification sent: %s", alert.Name)
	return nil
}

func buildDingTalkWebhookURL(webhook, secret string, ts time.Time) (string, error) {
	if strings.TrimSpace(webhook) == "" {
		return "", fmt.Errorf("dingtalk webhook is empty")
	}
	if strings.TrimSpace(secret) == "" {
		return webhook, nil
	}

	timestamp := strconv.FormatInt(ts.UnixMilli(), 10)
	stringToSign := timestamp + "\n" + secret

	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(stringToSign)); err != nil {
		return "", fmt.Errorf("failed to calculate dingtalk signature: %w", err)
	}

	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	parsed, err := url.Parse(webhook)
	if err != nil {
		return "", fmt.Errorf("invalid dingtalk webhook: %w", err)
	}
	query := parsed.Query()
	query.Set("timestamp", timestamp)
	query.Set("sign", signature)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// sendEmail sends an alert via email.
func (n *Notifier) sendEmail(alert *Alert) error {
	subject := fmt.Sprintf("[%s] %s - AI Gateway Alert", strings.ToUpper(string(alert.Level)), alert.Name)

	var bodyBuilder strings.Builder
	bodyBuilder.WriteString(fmt.Sprintf("Alert: %s\n", alert.Name))
	bodyBuilder.WriteString(fmt.Sprintf("Level: %s\n", alert.Level))
	bodyBuilder.WriteString(fmt.Sprintf("Message: %s\n", alert.Message))
	bodyBuilder.WriteString(fmt.Sprintf("Time: %s\n\n", alert.StartsAt.Format("2006-01-02 15:04:05")))

	if len(alert.Labels) > 0 {
		bodyBuilder.WriteString("Labels:\n")
		for k, v := range alert.Labels {
			bodyBuilder.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
		bodyBuilder.WriteString("\n")
	}

	if len(alert.Annotations) > 0 {
		bodyBuilder.WriteString("Details:\n")
		for k, v := range alert.Annotations {
			bodyBuilder.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
		bodyBuilder.WriteString("\n")
	}

	bodyBuilder.WriteString("\n--\nAI Gateway Monitoring System\n")

	// Construct email message
	msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s",
		n.config.SMTPFrom, n.config.EmailTo, subject, bodyBuilder.String())

	// SMTP authentication
	var auth smtp.Auth
	if n.config.SMTPUser != "" && n.config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", n.config.SMTPUser, n.config.SMTPPassword, n.config.SMTPHost)
	}

	// Send email
	addr := fmt.Sprintf("%s:%d", n.config.SMTPHost, n.config.SMTPPort)
	recipients := strings.Split(n.config.EmailTo, ",")

	if err := smtp.SendMail(addr, auth, n.config.SMTPFrom, recipients, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("[Alert] Email notification sent: %s", alert.Name)
	return nil
}

// getLevelEmoji returns an emoji for the alert level.
func (n *Notifier) getLevelEmoji(level AlertLevel) string {
	switch level {
	case AlertLevelCritical:
		return "\U0001F534" // Red circle
	case AlertLevelWarning:
		return "\U0001F7E1" // Yellow circle
	case AlertLevelInfo:
		return "\U0001F535" // Blue circle
	default:
		return "\u26A0\uFE0F" // Warning sign
	}
}

// FormatAlertFromPrometheus formats a Prometheus alert manager webhook payload.
func FormatAlertFromPrometheus(payload map[string]interface{}) ([]Alert, error) {
	status := extractString(payload, "status")
	alertsRaw, ok := payload["alerts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid alerts format in payload")
	}
	alerts := make([]Alert, 0, len(alertsRaw))
	now := time.Now()

	for _, alertRaw := range alertsRaw {
		alertMap, ok := alertRaw.(map[string]interface{})
		if !ok {
			continue
		}
		alerts = append(alerts, buildAlert(alertMap, status, now))
	}

	return alerts, nil
}

func buildAlert(alertMap map[string]interface{}, status string, now time.Time) Alert {
	labels := convertStringMap(alertMap["labels"])
	annotations := convertStringMap(alertMap["annotations"])

	alert := Alert{
		Name:        labels["alertname"],
		Level:       parseAlertLevel(labels["severity"]),
		Message:     firstNonEmpty(annotations["message"], annotations["summary"]),
		Labels:      labels,
		Annotations: annotations,
		StartsAt:    parseStartsAt(alertMap["startsAt"], now),
	}

	if status == "resolved" {
		resolvedAt := now
		alert.EndsAt = &resolvedAt
	}

	return alert
}

func convertStringMap(raw interface{}) map[string]string {
	result := make(map[string]string)
	rawMap, ok := raw.(map[string]interface{})
	if !ok {
		return result
	}

	for k, v := range rawMap {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}

	return result
}

func parseAlertLevel(severity string) AlertLevel {
	switch severity {
	case "critical":
		return AlertLevelCritical
	case "warning":
		return AlertLevelWarning
	default:
		return AlertLevelInfo
	}
}

func parseStartsAt(raw interface{}, fallback time.Time) time.Time {
	startsAt, ok := raw.(string)
	if !ok {
		return fallback
	}

	parsed, err := time.Parse(time.RFC3339, startsAt)
	if err != nil {
		return fallback
	}

	return parsed
}

func extractString(values map[string]interface{}, key string) string {
	raw, ok := values[key]
	if !ok {
		return ""
	}
	result, ok := raw.(string)
	if !ok {
		return ""
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
