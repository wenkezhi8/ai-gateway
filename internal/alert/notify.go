package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// AlertLevel represents the severity level of an alert
type AlertLevel string

const (
	AlertLevelCritical AlertLevel = "critical"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelInfo     AlertLevel = "info"
)

// Alert represents an alert event
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

// NotifierConfig holds configuration for alert notifiers
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

// Notifier handles alert notifications
type Notifier struct {
	config *NotifierConfig
	client *http.Client
}

// NewNotifier creates a new alert notifier
func NewNotifier(cfg *NotifierConfig) *Notifier {
	if cfg.EnabledLevels == nil {
		cfg.EnabledLevels = []AlertLevel{AlertLevelCritical, AlertLevelWarning, AlertLevelInfo}
	}

	return &Notifier{
		config: cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send sends an alert through all configured channels
func (n *Notifier) Send(alert Alert) error {
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

// isLevelEnabled checks if an alert level is enabled
func (n *Notifier) isLevelEnabled(level AlertLevel) bool {
	for _, l := range n.config.EnabledLevels {
		if l == level {
			return true
		}
	}
	return false
}

// DingTalkMessage represents a DingTalk webhook message
type DingTalkMessage struct {
	MsgType  string                   `json:"msgtype"`
	Markdown *DingTalkMarkdownContent `json:"markdown,omitempty"`
	Text     *DingTalkTextContent     `json:"text,omitempty"`
}

// DingTalkMarkdownContent represents markdown content for DingTalk
type DingTalkMarkdownContent struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// DingTalkTextContent represents text content for DingTalk
type DingTalkTextContent struct {
	Content string `json:"content"`
}

// sendDingTalk sends an alert to DingTalk
func (n *Notifier) sendDingTalk(alert Alert) error {
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

	req, err := http.NewRequest("POST", n.config.DingTalkWebhook, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create dingtalk request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// TODO: Add signature support if secret is provided
	// if n.config.DingTalkSecret != "" {
	//     // Add signature to URL
	// }

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

// sendEmail sends an alert via email
func (n *Notifier) sendEmail(alert Alert) error {
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

// getLevelEmoji returns an emoji for the alert level
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

// FormatAlertFromPrometheus formats a Prometheus alert manager webhook payload
func FormatAlertFromPrometheus(payload map[string]interface{}) ([]Alert, error) {
	alerts := make([]Alert, 0)

	status, _ := payload["status"].(string)
	alertsRaw, ok := payload["alerts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid alerts format in payload")
	}

	for _, alertRaw := range alertsRaw {
		alertMap, ok := alertRaw.(map[string]interface{})
		if !ok {
			continue
		}

		alert := Alert{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		}

		if name, ok := alertMap["labels"].(map[string]interface{}); ok {
			if alertname, ok := name["alertname"].(string); ok {
				alert.Name = alertname
			}
			for k, v := range name {
				if str, ok := v.(string); ok {
					alert.Labels[k] = str
				}
			}
		}

		if annotations, ok := alertMap["annotations"].(map[string]interface{}); ok {
			for k, v := range annotations {
				if str, ok := v.(string); ok {
					alert.Annotations[k] = str
				}
			}
		}

		if message, ok := alert.Annotations["message"]; ok {
			alert.Message = message
		} else if summary, ok := alert.Annotations["summary"]; ok {
			alert.Message = summary
		}

		if severity, ok := alert.Labels["severity"]; ok {
			switch severity {
			case "critical":
				alert.Level = AlertLevelCritical
			case "warning":
				alert.Level = AlertLevelWarning
			default:
				alert.Level = AlertLevelInfo
			}
		} else {
			alert.Level = AlertLevelInfo
		}

		if startsAt, ok := alertMap["startsAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339, startsAt); err == nil {
				alert.StartsAt = t
			}
		}
		if alert.StartsAt.IsZero() {
			alert.StartsAt = time.Now()
		}

		if status == "resolved" {
			now := time.Now()
			alert.EndsAt = &now
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}
