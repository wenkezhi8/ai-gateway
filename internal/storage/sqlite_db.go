package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// SQLiteStorage provides SQLite-based persistent storage
type SQLiteStorage struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

var (
	sqliteInstance     *SQLiteStorage
	sqliteInstanceOnce sync.Once
	sqliteLogger       = logrus.WithField("component", "sqlite")
)

// GetSQLiteStorage returns the global SQLite storage instance
func GetSQLiteStorage() *SQLiteStorage {
	sqliteInstanceOnce.Do(func() {
		path := os.Getenv("AI_GATEWAY_SQLITE_PATH")
		if path == "" {
			path = "data/ai-gateway.db"
		}
		var err error
		sqliteInstance, err = NewSQLiteStorage(path)
		if err != nil {
			sqliteLogger.Fatalf("Failed to initialize SQLite: %v", err)
		}
	})
	return sqliteInstance
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir failed: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db failed: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	storage := &SQLiteStorage{db: db, path: path}

	if err := storage.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate failed: %w", err)
	}

	sqliteLogger.WithField("path", path).Info("SQLite initialized")
	return storage, nil
}

// migrate runs database migrations
func (s *SQLiteStorage) migrate() error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			provider TEXT NOT NULL,
			api_key TEXT,
			base_url TEXT,
			models TEXT,
			enabled INTEGER DEFAULT 1,
			priority INTEGER DEFAULT 0,
			weight INTEGER DEFAULT 100,
			daily_quota INTEGER DEFAULT 0,
			daily_used INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_provider ON accounts(provider)`,

		`CREATE TABLE IF NOT EXISTS model_scores (
			model TEXT PRIMARY KEY,
			score REAL NOT NULL DEFAULT 0,
			avg_latency INTEGER DEFAULT 0,
			success_rate REAL DEFAULT 1.0,
			total_requests INTEGER DEFAULT 0,
			updated_at INTEGER NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
	}

	for _, schema := range schemas {
		if _, err := s.db.Exec(schema); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// Ping checks connectivity
func (s *SQLiteStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

// SaveAccount saves an account
func (s *SQLiteStorage) SaveAccount(account map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, _ := account["id"].(string)
	name, _ := account["name"].(string)
	provider, _ := account["provider"].(string)
	apiKey, _ := account["api_key"].(string)
	baseURL, _ := account["base_url"].(string)
	models, _ := account["models"].(string)
	enabled := 1
	if e, ok := account["enabled"].(bool); ok && !e {
		enabled = 0
	}
	now := time.Now().Unix()

	query := `INSERT OR REPLACE INTO accounts 
		(id, name, provider, api_key, base_url, models, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM accounts WHERE id = ?), ?), ?)`

	_, err := s.db.Exec(query, id, name, provider, apiKey, baseURL, models, enabled, id, now, now)
	return err
}

// GetAccounts retrieves all accounts
func (s *SQLiteStorage) GetAccounts() ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, name, provider, api_key, base_url, models, enabled, created_at, updated_at FROM accounts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []map[string]interface{}
	for rows.Next() {
		var id, name, provider, apiKey, baseURL, models sql.NullString
		var enabled sql.NullInt64
		var createdAt, updatedAt sql.NullInt64

		if err := rows.Scan(&id, &name, &provider, &apiKey, &baseURL, &models, &enabled, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		accounts = append(accounts, map[string]interface{}{
			"id":         id.String,
			"name":       name.String,
			"provider":   provider.String,
			"api_key":    apiKey.String,
			"base_url":   baseURL.String,
			"models":     models.String,
			"enabled":    enabled.Int64 == 1,
			"created_at": createdAt.Int64,
			"updated_at": updatedAt.Int64,
		})
	}

	return accounts, nil
}

// DeleteAccount deletes an account
func (s *SQLiteStorage) DeleteAccount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

// SaveModelScore saves a model score
func (s *SQLiteStorage) SaveModelScore(model string, score map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scoreVal, _ := score["score"].(float64)
	avgLatency, _ := score["avg_latency"].(int64)
	successRate, _ := score["success_rate"].(float64)
	totalReqs, _ := score["total_requests"].(int64)
	now := time.Now().Unix()

	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO model_scores (model, score, avg_latency, success_rate, total_requests, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		model, scoreVal, avgLatency, successRate, totalReqs, now,
	)
	return err
}

// GetModelScores retrieves all model scores
func (s *SQLiteStorage) GetModelScores() (map[string]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT model, score, avg_latency, success_rate, total_requests, updated_at FROM model_scores`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := make(map[string]map[string]interface{})
	for rows.Next() {
		var model string
		var score, successRate float64
		var avgLatency, totalReqs, updatedAt int64

		if err := rows.Scan(&model, &score, &avgLatency, &successRate, &totalReqs, &updatedAt); err != nil {
			return nil, err
		}

		scores[model] = map[string]interface{}{
			"score":          score,
			"avg_latency":    avgLatency,
			"success_rate":   successRate,
			"total_requests": totalReqs,
			"updated_at":     updatedAt,
		}
	}

	return scores, nil
}

// GetConfig retrieves a config value
func (s *SQLiteStorage) GetConfig(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value string
	err := s.db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetConfig sets a config value
func (s *SQLiteStorage) SetConfig(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec(`INSERT OR REPLACE INTO config (key, value, updated_at) VALUES (?, ?, ?)`, key, value, now)
	return err
}

// GetStats returns storage statistics
func (s *SQLiteStorage) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var accountCount int64
	s.db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&accountCount)

	var modelCount int64
	s.db.QueryRow(`SELECT COUNT(*) FROM model_scores`).Scan(&modelCount)

	var dbSize int64
	if info, err := os.Stat(s.path); err == nil {
		dbSize = info.Size()
	}

	return map[string]interface{}{
		"accounts":   accountCount,
		"models":     modelCount,
		"db_size":    dbSize,
		"db_size_mb": float64(dbSize) / 1024 / 1024,
	}
}

// Vacuum optimizes the database
func (s *SQLiteStorage) Vacuum() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`VACUUM`)
	return err
}
