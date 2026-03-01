package admin

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
)

type ClassifierSwitchTaskStatus string

const (
	ClassifierSwitchTaskStatusPending ClassifierSwitchTaskStatus = "pending"
	ClassifierSwitchTaskStatusRunning ClassifierSwitchTaskStatus = "running"
	ClassifierSwitchTaskStatusSuccess ClassifierSwitchTaskStatus = "success"
	ClassifierSwitchTaskStatusFailed  ClassifierSwitchTaskStatus = "failed"
	ClassifierSwitchTaskStatusTimeout ClassifierSwitchTaskStatus = "timeout"
)

type ClassifierSwitchTask struct {
	TaskID        string                     `json:"task_id"`
	TargetModel   string                     `json:"target_model"`
	OriginalModel string                     `json:"original_model"`
	Status        ClassifierSwitchTaskStatus `json:"status"`
	StartedAt     int64                      `json:"started_at"`
	UpdatedAt     int64                      `json:"updated_at"`
	DeadlineAt    int64                      `json:"deadline_at"`
	LastError     string                     `json:"last_error,omitempty"`
	Attempts      int64                      `json:"attempts"`
}

type classifierSwitchTaskStore struct {
	db *sql.DB
	mu sync.RWMutex
}

func newClassifierSwitchTaskStore(path string) (*classifierSwitchTaskStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create switch task store dir: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open switch task store: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &classifierSwitchTaskStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *classifierSwitchTaskStore) migrate() error {
	query := `CREATE TABLE IF NOT EXISTS classifier_switch_tasks (
		task_id TEXT PRIMARY KEY,
		target_model TEXT NOT NULL,
		original_model TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		deadline_at INTEGER NOT NULL,
		last_error TEXT,
		attempts INTEGER NOT NULL DEFAULT 0
	)`
	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("migrate switch task table: %w", err)
	}
	return nil
}

func (s *classifierSwitchTaskStore) Create(task *ClassifierSwitchTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO classifier_switch_tasks (task_id, target_model, original_model, status, started_at, updated_at, deadline_at, last_error, attempts)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.TaskID,
		task.TargetModel,
		task.OriginalModel,
		task.Status,
		task.StartedAt,
		task.UpdatedAt,
		task.DeadlineAt,
		task.LastError,
		task.Attempts,
	)
	if err != nil {
		return fmt.Errorf("create switch task: %w", err)
	}
	return nil
}

func (s *classifierSwitchTaskStore) Get(taskID string) (*ClassifierSwitchTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var task ClassifierSwitchTask
	err := s.db.QueryRow(
		`SELECT task_id, target_model, original_model, status, started_at, updated_at, deadline_at, COALESCE(last_error, ''), attempts
		 FROM classifier_switch_tasks WHERE task_id = ?`,
		taskID,
	).Scan(
		&task.TaskID,
		&task.TargetModel,
		&task.OriginalModel,
		&task.Status,
		&task.StartedAt,
		&task.UpdatedAt,
		&task.DeadlineAt,
		&task.LastError,
		&task.Attempts,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get switch task: %w", err)
	}
	return &task, nil
}

func (s *classifierSwitchTaskStore) Update(task *ClassifierSwitchTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`UPDATE classifier_switch_tasks
		 SET status = ?, updated_at = ?, last_error = ?, attempts = ?, target_model = ?, original_model = ?, deadline_at = ?
		 WHERE task_id = ?`,
		task.Status,
		task.UpdatedAt,
		task.LastError,
		task.Attempts,
		task.TargetModel,
		task.OriginalModel,
		task.DeadlineAt,
		task.TaskID,
	)
	if err != nil {
		return fmt.Errorf("update switch task: %w", err)
	}
	return nil
}

func (s *classifierSwitchTaskStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
