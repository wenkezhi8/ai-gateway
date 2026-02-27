package admin

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestClassifierSwitchExecutor_SuccessWithinWindow(t *testing.T) {
	store, err := newClassifierSwitchTaskStore(filepath.Join(t.TempDir(), "switch.db"))
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer store.Close()

	now := time.Unix(1700000000, 0)
	task := &ClassifierSwitchTask{
		TaskID:        "task-success",
		TargetModel:   "qwen3:4b",
		OriginalModel: "qwen2.5:0.5b-instruct",
		Status:        ClassifierSwitchTaskStatusPending,
		StartedAt:     now.UnixMilli(),
		UpdatedAt:     now.UnixMilli(),
		DeadlineAt:    now.Add(180 * time.Second).UnixMilli(),
	}
	if err := store.Create(task); err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	attempt := 0
	h := &RouterHandler{switchTaskStore: store}
	h.nowFn = func() time.Time { return now }
	h.sleepFn = func(d time.Duration) { now = now.Add(d) }
	h.probeSwitchFn = func(targetModel, originalModel string) error {
		attempt++
		if attempt == 1 {
			return errors.New("ollama request failed: context deadline exceeded")
		}
		return nil
	}

	h.executeSwitchTask("task-success")

	updated, err := store.Get("task-success")
	if err != nil {
		t.Fatalf("load task failed: %v", err)
	}
	if updated.Status != ClassifierSwitchTaskStatusSuccess {
		t.Fatalf("status = %s, want %s", updated.Status, ClassifierSwitchTaskStatusSuccess)
	}
	if updated.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", updated.Attempts)
	}
}

func TestClassifierSwitchExecutor_TimeoutReturnsFriendlyMessage(t *testing.T) {
	store, err := newClassifierSwitchTaskStore(filepath.Join(t.TempDir(), "switch.db"))
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer store.Close()

	now := time.Unix(1700000100, 0)
	task := &ClassifierSwitchTask{
		TaskID:        "task-timeout",
		TargetModel:   "qwen3:4b",
		OriginalModel: "qwen2.5:0.5b-instruct",
		Status:        ClassifierSwitchTaskStatusPending,
		StartedAt:     now.UnixMilli(),
		UpdatedAt:     now.UnixMilli(),
		DeadlineAt:    now.Add(4 * time.Second).UnixMilli(),
	}
	if err := store.Create(task); err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	h := &RouterHandler{switchTaskStore: store}
	h.nowFn = func() time.Time { return now }
	h.sleepFn = func(d time.Duration) { now = now.Add(d) }
	h.probeSwitchFn = func(targetModel, originalModel string) error {
		return errors.New("ollama request failed: Post \"http://127.0.0.1:11434/api/chat\": context deadline exceeded")
	}

	h.executeSwitchTask("task-timeout")

	updated, err := store.Get("task-timeout")
	if err != nil {
		t.Fatalf("load task failed: %v", err)
	}
	if updated.Status != ClassifierSwitchTaskStatusTimeout {
		t.Fatalf("status = %s, want %s", updated.Status, ClassifierSwitchTaskStatusTimeout)
	}
	if updated.LastError != classifierSwitchTimeoutMessage {
		t.Fatalf("timeout message = %q, want %q", updated.LastError, classifierSwitchTimeoutMessage)
	}
}
