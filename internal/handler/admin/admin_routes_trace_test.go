package admin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraceDeleteRoute_ShouldBeRegistered(t *testing.T) {
	adminFile := filepath.Join("..", "..", "handler", "admin", "admin.go")
	content, err := os.ReadFile(adminFile)
	if err != nil {
		t.Fatalf("read admin.go failed: %v", err)
	}

	if !strings.Contains(string(content), "traceGroup.DELETE(\"\", handlers.Trace.ClearTraces)") {
		t.Fatalf("trace clear route registration missing in RegisterRoutes")
	}
}

func TestAlertHistoryDeleteRoute_ShouldBeRegistered(t *testing.T) {
	adminFile := filepath.Join("..", "..", "handler", "admin", "admin.go")
	content, err := os.ReadFile(adminFile)
	if err != nil {
		t.Fatalf("read admin.go failed: %v", err)
	}

	if !strings.Contains(string(content), "alerts.DELETE(\"/history\", handlers.Alert.ClearHistory)") {
		t.Fatalf("alert history delete route registration missing in RegisterRoutes")
	}
}
