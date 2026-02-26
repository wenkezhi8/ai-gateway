package admin

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestAlertHandler(t *testing.T) *AlertHandler {
	t.Helper()
	return &AlertHandler{
		rules:         make([]AlertRule, 0),
		alerts:        make([]AlertRecord, 0),
		dataPath:      filepath.Join(t.TempDir(), "alerts.json"),
		alertCooldown: defaultAlertCooldown,
		lastAlerts:    make(map[string]time.Time),
	}
}

func TestAlertHandler_AddAlert_Dedup(t *testing.T) {
	handler := newTestAlertHandler(t)

	handler.AddAlert("warning", "system", "cpu high")
	handler.AddAlert("warning", "system", "cpu high")

	if len(handler.alerts) != 1 {
		t.Fatalf("expected 1 alert after dedup, got %d", len(handler.alerts))
	}
}

func TestAlertHandler_AddAlert_AllowsAfterCooldown(t *testing.T) {
	handler := newTestAlertHandler(t)
	handler.alertCooldown = time.Second

	handler.AddAlert("warning", "system", "cpu high")

	key := buildAlertDedupKey("warning", "system", "cpu high")
	handler.lastAlerts[key] = time.Now().Add(-2 * time.Second)

	handler.AddAlert("warning", "system", "cpu high")

	if len(handler.alerts) != 2 {
		t.Fatalf("expected 2 alerts after cooldown, got %d", len(handler.alerts))
	}
}

func TestDashboardHandler_AddAlert_Dedup(t *testing.T) {
	handler := &DashboardHandler{
		alerts:        make([]AlertListItem, 0),
		alertCooldown: defaultAlertCooldown,
		lastAlerts:    make(map[string]time.Time),
	}

	now := time.Now()
	handler.AddAlert(AlertListItem{
		Type:      "health",
		Level:     "warning",
		Message:   "memory high",
		Timestamp: now,
	})
	handler.AddAlert(AlertListItem{
		Type:      "health",
		Level:     "warning",
		Message:   "memory high",
		Timestamp: now.Add(10 * time.Second),
	})

	if len(handler.alerts) != 1 {
		t.Fatalf("expected 1 dashboard alert after dedup, got %d", len(handler.alerts))
	}
}

func TestDashboardHandler_AddAlert_AllowsAfterCooldown(t *testing.T) {
	handler := &DashboardHandler{
		alerts:        make([]AlertListItem, 0),
		alertCooldown: 5 * time.Minute,
		lastAlerts:    make(map[string]time.Time),
	}

	now := time.Now()
	handler.AddAlert(AlertListItem{
		Type:      "health",
		Level:     "warning",
		Message:   "memory high",
		Timestamp: now,
	})

	handler.AddAlert(AlertListItem{
		Type:      "health",
		Level:     "warning",
		Message:   "memory high",
		Timestamp: now.Add(10 * time.Minute),
	})

	if len(handler.alerts) != 2 {
		t.Fatalf("expected 2 dashboard alerts after cooldown, got %d", len(handler.alerts))
	}
}
