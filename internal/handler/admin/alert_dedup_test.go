package admin

import (
	"encoding/json"
	"fmt"
	"os"
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

	if len(handler.alerts) != 1 {
		t.Fatalf("expected aggregated alerts to stay single row, got %d", len(handler.alerts))
	}
	if handler.alerts[0].TriggerCount != 2 {
		t.Fatalf("expected trigger_count=2 after cooldown retrigger, got %d", handler.alerts[0].TriggerCount)
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

func TestAlertHandler_UpsertAlert_ShouldAggregateTriggerCountAndLastTriggeredAt(t *testing.T) {
	handler := newTestAlertHandler(t)

	first := time.Now().Add(-2 * time.Minute)
	second := first.Add(90 * time.Second)

	handler.UpsertAlert("warning", "system", "memory_warning", "内存使用率偏高: 93.2%", first)
	handler.UpsertAlert("warning", "system", "memory_warning", "内存使用率偏高: 93.1%", second)

	if len(handler.alerts) != 1 {
		t.Fatalf("expected 1 aggregated alert, got %d", len(handler.alerts))
	}

	alert := handler.alerts[0]
	if alert.DedupKey != "memory_warning" {
		t.Fatalf("expected dedup key memory_warning, got %s", alert.DedupKey)
	}
	if alert.TriggerCount != 2 {
		t.Fatalf("expected trigger_count=2, got %d", alert.TriggerCount)
	}
	if alert.LastTriggeredAt != second.Format(time.RFC3339) {
		t.Fatalf("expected last_triggered_at updated to %s, got %s", second.Format(time.RFC3339), alert.LastTriggeredAt)
	}
	if alert.Message != "内存使用率偏高: 93.1%" {
		t.Fatalf("expected latest message to be kept, got %s", alert.Message)
	}
}

func TestAlertHandler_ResolveByDedupKey_AutoResolved(t *testing.T) {
	handler := newTestAlertHandler(t)
	now := time.Now()

	handler.UpsertAlert("warning", "system", "memory_warning", "内存使用率偏高: 89%", now)

	affected := handler.ResolveByDedupKey("system", "memory_warning", true)
	if affected != 1 {
		t.Fatalf("expected affected=1, got %d", affected)
	}
	if len(handler.alerts) != 1 {
		t.Fatalf("expected one alert, got %d", len(handler.alerts))
	}
	if handler.alerts[0].Status != "resolved" {
		t.Fatalf("expected resolved status, got %s", handler.alerts[0].Status)
	}
	if !handler.alerts[0].AutoResolved {
		t.Fatalf("expected auto_resolved=true")
	}
	if handler.alerts[0].ResolvedAt == "" {
		t.Fatalf("expected resolved_at to be populated")
	}
}

func TestAlertHandler_CompactLegacyAlertsOnce_ShouldCollapsePendingDuplicates(t *testing.T) {
	handler := newTestAlertHandler(t)
	base := time.Now().Add(-10 * time.Minute)

	handler.alerts = []AlertRecord{
		{ID: "a1", Time: base.Format(time.RFC3339), Level: "warning", Source: "system", Message: "内存使用率偏高: 91.2%", Status: "pending"},
		{ID: "a2", Time: base.Add(2 * time.Minute).Format(time.RFC3339), Level: "warning", Source: "system", Message: "内存使用率偏高: 91.0%", Status: "pending"},
		{ID: "a3", Time: base.Add(4 * time.Minute).Format(time.RFC3339), Level: "warning", Source: "system", Message: "内存使用率偏高: 90.9%", Status: "pending"},
		{ID: "b1", Time: base.Add(5 * time.Minute).Format(time.RFC3339), Level: "critical", Source: "system", Message: "Goroutine 数过高: 18000", Status: "pending"},
	}

	handler.compactLegacyAlertsOnce()

	pendingByKey := map[string]int{}
	for _, alert := range handler.alerts {
		if alert.Status != "pending" {
			continue
		}
		pendingByKey[alert.DedupKey]++
		if alert.TriggerCount <= 0 {
			t.Fatalf("expected trigger_count > 0 for %s", alert.ID)
		}
		if alert.FirstTriggeredAt == "" || alert.LastTriggeredAt == "" {
			t.Fatalf("expected first/last triggered at for %s", alert.ID)
		}
	}

	if pendingByKey["memory_warning"] != 1 {
		t.Fatalf("expected exactly one pending memory_warning alert, got %d", pendingByKey["memory_warning"])
	}
	if pendingByKey["goroutine_critical"] != 1 {
		t.Fatalf("expected exactly one pending goroutine_critical alert, got %d", pendingByKey["goroutine_critical"])
	}

	for _, alert := range handler.alerts {
		if alert.DedupKey != "memory_warning" {
			continue
		}
		if alert.TriggerCount != 3 {
			t.Fatalf("expected compacted memory_warning trigger_count=3, got %d", alert.TriggerCount)
		}
		expectedFirst := base.Format(time.RFC3339)
		expectedLast := base.Add(4 * time.Minute).Format(time.RFC3339)
		if alert.FirstTriggeredAt != expectedFirst {
			t.Fatalf("expected first_triggered_at=%s, got %s", expectedFirst, alert.FirstTriggeredAt)
		}
		if alert.LastTriggeredAt != expectedLast {
			t.Fatalf("expected last_triggered_at=%s, got %s", expectedLast, alert.LastTriggeredAt)
		}
		if alert.Message == "" {
			t.Fatalf("expected compacted alert message to be non-empty")
		}
		if alert.Message != "内存使用率偏高: 90.9%" {
			t.Fatalf("expected latest message retained, got %s", alert.Message)
		}
		return
	}

	t.Fatalf("expected compacted memory_warning alert, got alerts=%s", fmt.Sprintf("%+v", handler.alerts))
}

func TestNewAlertHandler_ShouldKeepPersistedEmptyRules(t *testing.T) {
	originWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originWD); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
		globalAlertHandler = nil
	})

	alertsPath := filepath.Join(tempDir, "data", "alerts.json")
	if err := os.MkdirAll(filepath.Dir(alertsPath), 0o755); err != nil {
		t.Fatalf("mkdir alerts dir: %v", err)
	}
	if err := os.WriteFile(alertsPath, []byte(`{"rules":[],"alerts":[]}`), 0o640); err != nil {
		t.Fatalf("write alerts file: %v", err)
	}

	globalAlertHandler = nil
	h := NewAlertHandler()

	if len(h.rules) != 0 {
		t.Fatalf("expected persisted empty rules to stay empty, got %d", len(h.rules))
	}

	content, err := os.ReadFile(alertsPath)
	if err != nil {
		t.Fatalf("read alerts file: %v", err)
	}

	var persisted struct {
		Rules []AlertRule `json:"rules"`
	}
	if err := json.Unmarshal(content, &persisted); err != nil {
		t.Fatalf("unmarshal alerts file: %v", err)
	}
	if len(persisted.Rules) != 0 {
		t.Fatalf("expected persisted file rules to remain empty, got %d", len(persisted.Rules))
	}
}

func TestNewAlertHandler_ShouldInjectDefaultRulesWithoutPersistedFile(t *testing.T) {
	originWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originWD); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
		globalAlertHandler = nil
	})

	globalAlertHandler = nil
	h := NewAlertHandler()

	if len(h.rules) == 0 {
		t.Fatalf("expected default rules when no persisted file exists")
	}
}
