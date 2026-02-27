package admin

import (
	"path/filepath"
	"testing"
	"time"
)

func TestClassifierSwitchTaskStore_CreateGetUpdate_Reopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "switch-tasks.db")

	store, err := newClassifierSwitchTaskStore(dbPath)
	if err != nil {
		t.Fatalf("create task store failed: %v", err)
	}
	defer store.Close()

	now := time.Now().UnixMilli()
	task := &ClassifierSwitchTask{
		TaskID:        "task-001",
		TargetModel:   "qwen3:4b",
		OriginalModel: "qwen2.5:0.5b-instruct",
		Status:        ClassifierSwitchTaskStatusPending,
		StartedAt:     now,
		UpdatedAt:     now,
		DeadlineAt:    now + 180000,
		Attempts:      0,
	}

	if err := store.Create(task); err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	loaded, err := store.Get(task.TaskID)
	if err != nil {
		t.Fatalf("get task failed: %v", err)
	}
	if loaded == nil {
		t.Fatalf("loaded task is nil")
	}
	if loaded.TargetModel != task.TargetModel {
		t.Fatalf("target model = %s, want %s", loaded.TargetModel, task.TargetModel)
	}

	loaded.Status = ClassifierSwitchTaskStatusRunning
	loaded.Attempts = 2
	loaded.LastError = ""
	loaded.UpdatedAt = time.Now().UnixMilli()
	if err := store.Update(loaded); err != nil {
		t.Fatalf("update task failed: %v", err)
	}

	updated, err := store.Get(task.TaskID)
	if err != nil {
		t.Fatalf("get updated task failed: %v", err)
	}
	if updated == nil || updated.Status != ClassifierSwitchTaskStatusRunning {
		t.Fatalf("status = %v, want %v", updated.Status, ClassifierSwitchTaskStatusRunning)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close store failed: %v", err)
	}

	reopened, err := newClassifierSwitchTaskStore(dbPath)
	if err != nil {
		t.Fatalf("reopen task store failed: %v", err)
	}
	defer reopened.Close()

	afterReopen, err := reopened.Get(task.TaskID)
	if err != nil {
		t.Fatalf("get task after reopen failed: %v", err)
	}
	if afterReopen == nil {
		t.Fatalf("task missing after reopen")
	}
	if afterReopen.Status != ClassifierSwitchTaskStatusRunning {
		t.Fatalf("status after reopen = %v, want %v", afterReopen.Status, ClassifierSwitchTaskStatusRunning)
	}
}
