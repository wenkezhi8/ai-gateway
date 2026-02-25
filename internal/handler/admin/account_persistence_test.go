package admin

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ai-gateway/internal/limiter"
)

func TestSaveAndLoadSwitchHistory(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	input := []limiter.SwitchEvent{
		{
			FromAccount: "a1",
			ToAccount:   "a2",
			Reason:      "forced switch",
			Timestamp:   time.Now().UTC().Truncate(time.Second),
			Duration:    150 * time.Millisecond,
		},
	}

	if err := SaveSwitchHistoryToFile(input); err != nil {
		t.Fatalf("save switch history: %v", err)
	}

	if _, err := os.Stat(filepath.Join("data", "switch_history.json")); err != nil {
		t.Fatalf("expected switch history file: %v", err)
	}

	loaded, err := LoadPersistedSwitchHistory()
	if err != nil {
		t.Fatalf("load switch history: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
	if loaded[0].FromAccount != input[0].FromAccount || loaded[0].ToAccount != input[0].ToAccount || loaded[0].Reason != input[0].Reason {
		t.Fatalf("loaded event mismatch: %+v", loaded[0])
	}
}

func TestLoadPersistedSwitchHistoryMissingFile(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	history, err := LoadPersistedSwitchHistory()
	if err != nil {
		t.Fatalf("load missing switch history: %v", err)
	}
	if history != nil {
		t.Fatalf("expected nil history for missing file, got %v", history)
	}
}
