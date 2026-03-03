package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/models"

	// Register sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

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

func (s *SQLiteStorage) migrate() error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			api_key TEXT,
			priority INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			quota_limit INTEGER DEFAULT 0,
			quota_used INTEGER DEFAULT 0,
			quota_reset_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_provider ON accounts(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_enabled ON accounts(enabled)`,

		`CREATE TABLE IF NOT EXISTS model_scores (
			model TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			quality_score INTEGER DEFAULT 0,
			speed_score INTEGER DEFAULT 0,
			cost_score INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			is_custom INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_model_scores_enabled ON model_scores(enabled)`,

		`CREATE TABLE IF NOT EXISTS provider_defaults (
			provider TEXT PRIMARY KEY,
			model TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS router_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			key TEXT NOT NULL,
			permissions TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			last_used_at TEXT,
			created_at TEXT NOT NULL,
			expires_at TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			email TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS deleted_models (
			model TEXT PRIMARY KEY,
			deleted_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS feedback (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL,
			model TEXT NOT NULL,
			provider TEXT NOT NULL,
			task_type TEXT,
			rating INTEGER,
			comment TEXT,
			latency_ms INTEGER,
			tokens_used INTEGER,
			cache_hit INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_feedback_model ON feedback(model)`,
		`CREATE INDEX IF NOT EXISTS idx_feedback_created ON feedback(created_at)`,

		`CREATE TABLE IF NOT EXISTS request_traces (
			id TEXT PRIMARY KEY,
			request_id TEXT NOT NULL,
			trace_id TEXT NOT NULL,
			span_id TEXT NOT NULL,
			parent_span_id TEXT,
			operation TEXT NOT NULL,
			status TEXT NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			attributes TEXT,
			events TEXT,
			user_id TEXT,
			method TEXT,
			path TEXT,
			model TEXT,
			provider TEXT,
			error TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_request_id ON request_traces(request_id)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_trace_id ON request_traces(trace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_created_at ON request_traces(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_operation ON request_traces(operation)`,

		`CREATE TABLE IF NOT EXISTS usage_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT,
			timestamp INTEGER NOT NULL DEFAULT 0,
			model TEXT NOT NULL,
			provider TEXT,
			account TEXT DEFAULT '',
			user_id TEXT,
			api_key TEXT,
			user_agent TEXT DEFAULT '',
			request_type TEXT DEFAULT 'non_stream',
			inference_intensity TEXT DEFAULT '',
			tokens INTEGER DEFAULT 0,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			latency_ms INTEGER DEFAULT 0,
			ttft_ms INTEGER DEFAULT 0,
			cache_hit INTEGER DEFAULT 0,
			success INTEGER DEFAULT 1,
			error_type TEXT,
			task_type TEXT,
			difficulty TEXT,
			experiment_tag TEXT,
			domain_tag TEXT,
			usage_source TEXT DEFAULT 'actual',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_model ON usage_logs(model)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_provider ON usage_logs(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_task_type ON usage_logs(task_type)`,
		`CREATE TABLE IF NOT EXISTS dashboard_summary (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			total_requests INTEGER DEFAULT 0,
			requests_today INTEGER DEFAULT 0,
			success_count INTEGER DEFAULT 0,
			failure_count INTEGER DEFAULT 0,
			total_latency INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS dashboard_models (
			model TEXT PRIMARY KEY,
			requests INTEGER DEFAULT 0,
			tokens INTEGER DEFAULT 0,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS dashboard_trends (
			ts INTEGER PRIMARY KEY,
			requests INTEGER DEFAULT 0,
			success INTEGER DEFAULT 0,
			failed INTEGER DEFAULT 0,
			latency INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dashboard_trends_ts ON dashboard_trends(ts)`,
		`CREATE TABLE IF NOT EXISTS dashboard_alerts (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			account_id TEXT,
			provider TEXT,
			timestamp INTEGER NOT NULL,
			acknowledged INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dashboard_alerts_ts ON dashboard_alerts(timestamp)`,
	}

	for _, schema := range schemas {
		if _, err := s.db.Exec(schema); err != nil {
			return fmt.Errorf("schema error: %w", err)
		}
	}
	if err := s.ensureUsageLogsColumns(); err != nil {
		return fmt.Errorf("ensure usage logs columns failed: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func (s *SQLiteStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

func (s *SQLiteStorage) SaveAccount(account *models.AccountRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	query := `INSERT OR REPLACE INTO accounts 
		(id, provider, api_key, priority, enabled, quota_limit, quota_used, quota_reset_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM accounts WHERE id = ?), ?), ?)`

	_, err := s.db.Exec(query,
		account.ID, account.Provider, account.APIKey, account.Priority, boolToInt(account.Enabled),
		account.QuotaLimit, account.QuotaUsed, account.QuotaResetAt,
		account.ID, now, now)
	return err
}

func (s *SQLiteStorage) GetAccount(id string) (*models.AccountRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var acc models.AccountRecord
	var apiKey, quotaResetAt sql.NullString
	var enabledInt int
	err := s.db.QueryRow(`SELECT id, provider, api_key, priority, enabled, quota_limit, quota_used, quota_reset_at, created_at, updated_at FROM accounts WHERE id = ?`, id).
		Scan(&acc.ID, &acc.Provider, &apiKey, &acc.Priority, &enabledInt, &acc.QuotaLimit, &acc.QuotaUsed, &quotaResetAt, &acc.CreatedAt, &acc.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	acc.APIKey = apiKey.String
	acc.QuotaResetAt = quotaResetAt.String
	acc.Enabled = enabledInt == 1
	return &acc, nil
}

func (s *SQLiteStorage) GetAllAccounts() (map[string]*models.AccountRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, provider, api_key, priority, enabled, quota_limit, quota_used, quota_reset_at, created_at, updated_at FROM accounts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make(map[string]*models.AccountRecord)
	for rows.Next() {
		var acc models.AccountRecord
		var apiKey, quotaResetAt sql.NullString
		var enabledInt int
		if err := rows.Scan(&acc.ID, &acc.Provider, &apiKey, &acc.Priority, &enabledInt, &acc.QuotaLimit, &acc.QuotaUsed, &quotaResetAt, &acc.CreatedAt, &acc.UpdatedAt); err != nil {
			return nil, err
		}
		acc.APIKey = apiKey.String
		acc.QuotaResetAt = quotaResetAt.String
		acc.Enabled = enabledInt == 1
		accounts[acc.ID] = &acc
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (s *SQLiteStorage) DeleteAccount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

func (s *SQLiteStorage) SaveModelScore(model string, score *models.ModelScoreRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	query := `INSERT OR REPLACE INTO model_scores 
		(model, provider, quality_score, speed_score, cost_score, enabled, is_custom, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM model_scores WHERE model = ?), ?), ?)`

	_, err := s.db.Exec(query,
		model, score.Provider, score.QualityScore, score.SpeedScore, score.CostScore,
		boolToInt(score.Enabled), boolToInt(score.IsCustom),
		model, now, now)
	return err
}

func (s *SQLiteStorage) GetModelScore(model string) (*models.ModelScoreRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var score models.ModelScoreRecord
	var enabledInt, isCustomInt int
	err := s.db.QueryRow(`SELECT model, provider, quality_score, speed_score, cost_score, enabled, is_custom, created_at, updated_at FROM model_scores WHERE model = ?`, model).
		Scan(&score.Model, &score.Provider, &score.QualityScore, &score.SpeedScore, &score.CostScore, &enabledInt, &isCustomInt, &score.CreatedAt, &score.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	score.Enabled = enabledInt == 1
	score.IsCustom = isCustomInt == 1
	return &score, nil
}

func (s *SQLiteStorage) GetAllModelScores() (map[string]*models.ModelScoreRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queryModelScores(`SELECT model, provider, quality_score, speed_score, cost_score, enabled, is_custom, created_at, updated_at FROM model_scores`)
}

func (s *SQLiteStorage) queryModelScores(query string, args ...interface{}) (map[string]*models.ModelScoreRecord, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := make(map[string]*models.ModelScoreRecord)
	for rows.Next() {
		var score models.ModelScoreRecord
		var enabledInt, isCustomInt int
		if err := rows.Scan(&score.Model, &score.Provider, &score.QualityScore, &score.SpeedScore, &score.CostScore, &enabledInt, &isCustomInt, &score.CreatedAt, &score.UpdatedAt); err != nil {
			return nil, err
		}
		score.Enabled = enabledInt == 1
		score.IsCustom = isCustomInt == 1
		scores[score.Model] = &score
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return scores, nil
}

func (s *SQLiteStorage) GetEnabledModelScores() (map[string]*models.ModelScoreRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queryModelScores(`SELECT model, provider, quality_score, speed_score, cost_score, enabled, is_custom, created_at, updated_at FROM model_scores WHERE enabled = 1 AND model NOT IN (SELECT model FROM deleted_models)`)
}

func (s *SQLiteStorage) DeleteModelScore(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			sqliteLogger.WithError(rbErr).Warn("rollback transaction failed")
		}
	}()

	if _, err := tx.Exec(`DELETE FROM model_scores WHERE model = ?`, model); err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	if _, err := tx.Exec(`INSERT OR REPLACE INTO deleted_models (model, deleted_at) VALUES (?, ?)`, model, now); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStorage) SaveUser(username string, user *models.UserRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	query := `INSERT OR REPLACE INTO users 
		(username, password_hash, role, email, created_at, updated_at)
		VALUES (?, ?, ?, ?, COALESCE((SELECT created_at FROM users WHERE username = ?), ?), ?)`

	_, err := s.db.Exec(query,
		username, user.PasswordHash, user.Role, user.Email,
		username, now, now)
	return err
}

func (s *SQLiteStorage) GetUser(username string) (*models.UserRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var user models.UserRecord
	var email sql.NullString
	err := s.db.QueryRow(`SELECT username, password_hash, role, email, created_at, updated_at FROM users WHERE username = ?`, username).
		Scan(&user.Username, &user.PasswordHash, &user.Role, &email, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.Email = email.String
	return &user, nil
}

func (s *SQLiteStorage) GetAllUsers() (map[string]*models.UserRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT username, password_hash, role, email, created_at, updated_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[string]*models.UserRecord)
	for rows.Next() {
		var user models.UserRecord
		var email sql.NullString
		if err := rows.Scan(&user.Username, &user.PasswordHash, &user.Role, &email, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		user.Email = email.String
		users[user.Username] = &user
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *SQLiteStorage) DeleteUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM users WHERE username = ?`, username)
	return err
}

func (s *SQLiteStorage) SaveAPIKey(id string, key *models.APIKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	query := `INSERT OR REPLACE INTO api_keys 
		(id, name, key, permissions, enabled, last_used_at, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM api_keys WHERE id = ?), ?), ?)`

	_, err := s.db.Exec(query,
		id, key.Name, key.Key, key.Permissions, boolToInt(key.Enabled), key.LastUsedAt,
		id, now, key.ExpiresAt)
	return err
}

func (s *SQLiteStorage) GetAPIKey(id string) (*models.APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var key models.APIKeyRecord
	var lastUsedAt, expiresAt sql.NullString
	var enabledInt int
	err := s.db.QueryRow(`SELECT id, name, key, permissions, enabled, last_used_at, created_at, expires_at FROM api_keys WHERE id = ?`, id).
		Scan(&key.ID, &key.Name, &key.Key, &key.Permissions, &enabledInt, &lastUsedAt, &key.CreatedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	key.Enabled = enabledInt == 1
	key.LastUsedAt = lastUsedAt.String
	key.ExpiresAt = expiresAt.String
	return &key, nil
}

func (s *SQLiteStorage) GetAllAPIKeys() (map[string]*models.APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, name, key, permissions, enabled, last_used_at, created_at, expires_at FROM api_keys`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make(map[string]*models.APIKeyRecord)
	for rows.Next() {
		var key models.APIKeyRecord
		var lastUsedAt, expiresAt sql.NullString
		var enabledInt int
		if err := rows.Scan(&key.ID, &key.Name, &key.Key, &key.Permissions, &enabledInt, &lastUsedAt, &key.CreatedAt, &expiresAt); err != nil {
			return nil, err
		}
		key.Enabled = enabledInt == 1
		key.LastUsedAt = lastUsedAt.String
		key.ExpiresAt = expiresAt.String
		keys[key.ID] = &key
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *SQLiteStorage) DeleteAPIKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM api_keys WHERE id = ?`, id)
	return err
}

func (s *SQLiteStorage) GetProviderDefault(provider string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var model string
	err := s.db.QueryRow(`SELECT model FROM provider_defaults WHERE provider = ?`, provider).Scan(&model)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return model, err
}

func (s *SQLiteStorage) SetProviderDefault(provider, model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT OR REPLACE INTO provider_defaults (provider, model, updated_at) VALUES (?, ?, ?)`, provider, model, now)
	return err
}

func (s *SQLiteStorage) GetAllProviderDefaults() (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT provider, model FROM provider_defaults`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	defaults := make(map[string]string)
	for rows.Next() {
		var provider, model string
		if err := rows.Scan(&provider, &model); err != nil {
			return nil, err
		}
		defaults[provider] = model
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defaults, nil
}

func (s *SQLiteStorage) GetRouterConfig() (*models.RouterConfigRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT key, value FROM router_config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configMap := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		configMap[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	config := &models.RouterConfigRecord{
		DefaultStrategy: configMap["default_strategy"],
		DefaultModel:    configMap["default_model"],
		UseAutoMode:     configMap["use_auto_mode"] == "true",
	}
	return config, nil
}

func (s *SQLiteStorage) SetRouterConfig(config *models.RouterConfigRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`INSERT OR REPLACE INTO router_config (key, value) VALUES 
		('default_strategy', ?),
		('default_model', ?),
		('use_auto_mode', ?)`,
		config.DefaultStrategy, config.DefaultModel, fmt.Sprintf("%v", config.UseAutoMode))
	return err
}

func (s *SQLiteStorage) MarkModelDeleted(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT OR REPLACE INTO deleted_models (model, deleted_at) VALUES (?, ?)`, model, now)
	return err
}

func (s *SQLiteStorage) IsModelDeleted(model string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM deleted_models WHERE model = ?`, model).Scan(&count)
	return count > 0, err
}

func (s *SQLiteStorage) RestoreModel(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM deleted_models WHERE model = ?`, model)
	return err
}

func (s *SQLiteStorage) GetAllDeletedModels() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT model FROM deleted_models`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []string
	for rows.Next() {
		var model string
		if err := rows.Scan(&model); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return models, nil
}

func (s *SQLiteStorage) SaveFeedback(feedback *models.FeedbackRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO feedback (request_id, model, provider, task_type, rating, comment, latency_ms, tokens_used, cache_hit, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feedback.RequestID, feedback.Model, feedback.Provider, feedback.TaskType, feedback.Rating, feedback.Comment, feedback.LatencyMs, feedback.TokensUsed, boolToInt(feedback.CacheHit), now)
	return err
}

func (s *SQLiteStorage) GetFeedback(limit, offset int) ([]*models.FeedbackRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, request_id, model, provider, task_type, rating, comment, latency_ms, tokens_used, cache_hit, created_at FROM feedback ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedbacks []*models.FeedbackRecord
	for rows.Next() {
		var f models.FeedbackRecord
		var taskType, comment sql.NullString
		var cacheHitInt int
		if err := rows.Scan(&f.ID, &f.RequestID, &f.Model, &f.Provider, &taskType, &f.Rating, &comment, &f.LatencyMs, &f.TokensUsed, &cacheHitInt, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.TaskType = taskType.String
		f.Comment = comment.String
		f.CacheHit = cacheHitInt == 1
		feedbacks = append(feedbacks, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return feedbacks, nil
}

func (s *SQLiteStorage) GetFeedbackStats() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	var totalCount int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM feedback`).Scan(&totalCount); err != nil {
		return nil, err
	}
	stats["total_count"] = totalCount

	var avgRating sql.NullFloat64
	if err := s.db.QueryRow(`SELECT AVG(rating) FROM feedback WHERE rating > 0`).Scan(&avgRating); err != nil {
		return nil, err
	}
	stats["avg_rating"] = avgRating.Float64

	rows, err := s.db.Query(`SELECT model, COUNT(*) as count, AVG(rating) as avg_rating FROM feedback GROUP BY model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	modelStats := make(map[string]map[string]interface{})
	for rows.Next() {
		var model string
		var count int64
		var avgRating sql.NullFloat64
		if err := rows.Scan(&model, &count, &avgRating); err != nil {
			return nil, err
		}
		modelStats[model] = map[string]interface{}{
			"count":      count,
			"avg_rating": avgRating.Float64,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stats["model_stats"] = modelStats

	return stats, nil
}

func (s *SQLiteStorage) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := func(query string) int64 {
		var value int64
		if err := s.db.QueryRow(query).Scan(&value); err != nil {
			sqliteLogger.WithError(err).WithField("query", query).Warn("count query failed")
			return 0
		}
		return value
	}

	accountCount := count(`SELECT COUNT(*) FROM accounts`)
	modelCount := count(`SELECT COUNT(*) FROM model_scores`)
	userCount := count(`SELECT COUNT(*) FROM users`)
	apiKeyCount := count(`SELECT COUNT(*) FROM api_keys`)
	feedbackCount := count(`SELECT COUNT(*) FROM feedback`)

	var dbSize int64
	if info, err := os.Stat(s.path); err == nil {
		dbSize = info.Size()
	}

	return map[string]interface{}{
		"accounts":   accountCount,
		"models":     modelCount,
		"users":      userCount,
		"api_keys":   apiKeyCount,
		"feedback":   feedbackCount,
		"db_size":    dbSize,
		"db_size_mb": float64(dbSize) / 1024 / 1024,
	}
}

func (s *SQLiteStorage) Vacuum() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`VACUUM`)
	return err
}

func (s *SQLiteStorage) Export() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := struct {
		Accounts         map[string]*models.AccountRecord    `json:"accounts"`
		ModelScores      map[string]*models.ModelScoreRecord `json:"model_scores"`
		ProviderDefaults map[string]string                   `json:"provider_defaults"`
		RouterConfig     *models.RouterConfigRecord          `json:"router_config"`
		APIKeys          map[string]*models.APIKeyRecord     `json:"api_keys"`
		Users            map[string]*models.UserRecord       `json:"users"`
		DeletedModels    []string                            `json:"deleted_models"`
	}{}

	accounts, err := s.GetAllAccounts()
	if err != nil {
		return nil, err
	}
	data.Accounts = accounts

	scores, err := s.GetAllModelScores()
	if err != nil {
		return nil, err
	}
	data.ModelScores = scores

	defaults, err := s.GetAllProviderDefaults()
	if err != nil {
		return nil, err
	}
	data.ProviderDefaults = defaults

	config, err := s.GetRouterConfig()
	if err != nil {
		return nil, err
	}
	data.RouterConfig = config

	keys, err := s.GetAllAPIKeys()
	if err != nil {
		return nil, err
	}
	data.APIKeys = keys

	users, err := s.GetAllUsers()
	if err != nil {
		return nil, err
	}
	data.Users = users

	deleted, err := s.GetAllDeletedModels()
	if err != nil {
		return nil, err
	}
	data.DeletedModels = deleted

	return json.MarshalIndent(data, "", "  ")
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i == 1
}

type UsageFilter struct {
	Model         string
	Provider      string
	TaskType      string
	ExperimentTag string
	DomainTag     string
	StartTime     int64
	EndTime       int64
}

type UsageLog struct {
	ID                 int64  `json:"id"`
	Timestamp          int64  `json:"timestamp"`
	Model              string `json:"model"`
	Provider           string `json:"provider"`
	Account            string `json:"account"`
	UserID             string `json:"user_id"`
	APIKey             string `json:"api_key"`
	UserAgent          string `json:"user_agent"`
	RequestType        string `json:"request_type"`
	InferenceIntensity string `json:"inference_intensity"`
	Tokens             int64  `json:"tokens"`
	InputTokens        int64  `json:"input_tokens"`
	OutputTokens       int64  `json:"output_tokens"`
	LatencyMs          int64  `json:"latency_ms"`
	TTFTMs             int64  `json:"ttft_ms"`
	CacheHit           bool   `json:"cache_hit"`
	Success            bool   `json:"success"`
	ErrorType          string `json:"error_type"`
	TaskType           string `json:"task_type"`
	Difficulty         string `json:"difficulty"`
	ExperimentTag      string `json:"experiment_tag"`
	DomainTag          string `json:"domain_tag"`
	UsageSource        string `json:"usage_source"`
	CreatedAt          string `json:"created_at"`
}

type DashboardSummary struct {
	TotalRequests int64
	RequestsToday int64
	SuccessCount  int64
	FailureCount  int64
	TotalLatency  int64
	TotalTokens   int64
	UpdatedAt     time.Time
}

type DashboardModelStat struct {
	Model    string
	Requests int64
	Tokens   int64
}

type DashboardTrend struct {
	Timestamp int64
	Requests  int64
	Success   int64
	Failed    int64
	Latency   int64
}

type DashboardAlert struct {
	ID           string
	Type         string
	Level        string
	Message      string
	AccountID    string
	Provider     string
	Timestamp    int64
	Acknowledged bool
}

type ProviderUsageStat struct {
	Provider    string
	Requests    int64
	Tokens      int64
	SuccessRate float64
	AvgLatency  int64
}

func GetSQLite() *SQLiteStorage {
	return GetSQLiteStorage()
}

func (s *SQLiteStorage) LogUsage(log map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	timestamp := usageInt64Value(log, "timestamp")
	if timestamp <= 0 {
		timestamp = time.Now().UnixMilli()
	}

	_, err := s.db.Exec(`INSERT INTO usage_logs (
		request_id, timestamp, model, provider, account, user_id, api_key, user_agent, request_type, inference_intensity,
		tokens, input_tokens, output_tokens, latency_ms, ttft_ms, cache_hit, success, error_type, task_type,
		difficulty, experiment_tag, domain_tag, usage_source, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usageStringValue(log, "request_id"),
		timestamp,
		usageStringValue(log, "model"),
		usageStringValue(log, "provider"),
		usageStringValue(log, "account"),
		usageStringValue(log, "user_id"),
		usageStringValue(log, "api_key"),
		usageStringValue(log, "user_agent"),
		usageRequestTypeValue(log),
		usageInferenceIntensityValue(log),
		usageInt64Value(log, "tokens"),
		usageInt64Value(log, "input_tokens"),
		usageInt64Value(log, "output_tokens"),
		usageInt64Value(log, "latency_ms"),
		usageInt64Value(log, "ttft_ms"),
		boolToInt(usageBoolValue(log, "cache_hit")),
		boolToInt(usageBoolValue(log, "success")),
		usageStringValue(log, "error_type"),
		usageStringValue(log, "task_type"),
		usageStringValue(log, "difficulty"),
		usageStringValue(log, "experiment_tag"),
		usageStringValue(log, "domain_tag"),
		usageSourceValue(log),
		now,
	)
	return err
}

//nolint:gocritic // Keep value receiver for compatibility with existing callers.
func (s *SQLiteStorage) GetUsageLogsWithFilter(filter UsageFilter, limit, offset int) ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	whereClause, args := buildUsageWhereClause(filter)
	//nolint:gosec // whereClause is built from fixed whitelisted fragments.
	query := `SELECT id, timestamp, model, provider, account, user_id, api_key, user_agent, request_type, inference_intensity,
		tokens, input_tokens, output_tokens, latency_ms, ttft_ms, cache_hit, success, error_type, task_type, difficulty,
		experiment_tag, domain_tag, usage_source, created_at
		FROM usage_logs WHERE ` + whereClause

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int64
		var timestamp, tokens, inputTokens, outputTokens, latencyMs, ttftMs int64
		var model, createdAt string
		var cacheHitInt, successInt int
		var provider, account, userID, apiKey, userAgent, requestType, inferenceIntensity sql.NullString
		var errorType, taskType, difficulty, experimentTag, domainTag, usageSource sql.NullString
		if err := rows.Scan(
			&id, &timestamp, &model, &provider, &account, &userID, &apiKey, &userAgent, &requestType, &inferenceIntensity,
			&tokens, &inputTokens, &outputTokens, &latencyMs, &ttftMs, &cacheHitInt, &successInt, &errorType, &taskType,
			&difficulty, &experimentTag, &domainTag, &usageSource, &createdAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, map[string]interface{}{
			"id":                  id,
			"timestamp":           timestamp,
			"model":               model,
			"provider":            provider.String,
			"account":             account.String,
			"user_id":             userID.String,
			"api_key":             apiKey.String,
			"user_agent":          userAgent.String,
			"request_type":        requestType.String,
			"inference_intensity": inferenceIntensity.String,
			"tokens":              tokens,
			"input_tokens":        inputTokens,
			"output_tokens":       outputTokens,
			"latency_ms":          latencyMs,
			"ttft_ms":             ttftMs,
			"cache_hit":           cacheHitInt == 1,
			"success":             successInt == 1,
			"error_type":          errorType.String,
			"task_type":           taskType.String,
			"difficulty":          difficulty.String,
			"experiment_tag":      experimentTag.String,
			"domain_tag":          domainTag.String,
			"usage_source":        usageSource.String,
			"created_at":          createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *SQLiteStorage) ClearUsageLogs() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`DELETE FROM usage_logs`)
	if err != nil {
		return 0, err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *SQLiteStorage) ensureUsageLogsColumns() error {
	required := map[string]string{
		"task_type":           "TEXT",
		"experiment_tag":      "TEXT",
		"domain_tag":          "TEXT",
		"usage_source":        "TEXT DEFAULT 'actual'",
		"account":             "TEXT DEFAULT ''",
		"user_agent":          "TEXT DEFAULT ''",
		"request_type":        "TEXT DEFAULT 'non_stream'",
		"inference_intensity": "TEXT DEFAULT ''",
	}

	rows, err := s.db.Query(`PRAGMA table_info(usage_logs)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	existing := map[string]struct{}{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for col, colType := range required {
		if _, ok := existing[col]; ok {
			continue
		}
		if _, err := s.db.Exec(fmt.Sprintf(`ALTER TABLE usage_logs ADD COLUMN %s %s`, col, colType)); err != nil {
			return err
		}
	}

	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_logs_experiment_tag ON usage_logs(experiment_tag)`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_logs_domain_tag ON usage_logs(domain_tag)`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_logs_task_type ON usage_logs(task_type)`); err != nil {
		return err
	}

	return nil
}

//nolint:gocritic // Keep value receiver for compatibility with existing callers.
func buildUsageWhereClause(filter UsageFilter) (string, []interface{}) {
	conditions := []string{"1=1"}
	args := make([]interface{}, 0, 8)

	if filter.Model != "" {
		conditions = append(conditions, "model = ?")
		args = append(args, filter.Model)
	}
	if filter.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.TaskType != "" {
		conditions = append(conditions, "task_type = ?")
		args = append(args, filter.TaskType)
	}
	if filter.ExperimentTag != "" {
		conditions = append(conditions, "experiment_tag = ?")
		args = append(args, filter.ExperimentTag)
	}
	if filter.DomainTag != "" {
		conditions = append(conditions, "domain_tag = ?")
		args = append(args, filter.DomainTag)
	}
	if filter.StartTime > 0 {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, time.UnixMilli(filter.StartTime).Format(time.RFC3339))
	}
	if filter.EndTime > 0 {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, time.UnixMilli(filter.EndTime).Format(time.RFC3339))
	}

	return strings.Join(conditions, " AND "), args
}

func usageStringValue(log map[string]interface{}, key string) string {
	v, ok := log[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func usageSourceValue(log map[string]interface{}) string {
	source := strings.ToLower(strings.TrimSpace(usageStringValue(log, "usage_source")))
	if source == "estimated" {
		return source
	}
	return "actual"
}

func usageRequestTypeValue(log map[string]interface{}) string {
	requestType := strings.ToLower(strings.TrimSpace(usageStringValue(log, "request_type")))
	if requestType == "stream" || requestType == "non_stream" {
		return requestType
	}
	return "non_stream"
}

func usageInferenceIntensityValue(log map[string]interface{}) string {
	intensity := strings.ToLower(strings.TrimSpace(usageStringValue(log, "inference_intensity")))
	switch intensity {
	case "low", "medium", "high", "xhigh":
		return intensity
	default:
		return ""
	}
}

func usageInt64Value(log map[string]interface{}, key string) int64 {
	v, ok := log[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	default:
		return 0
	}
}

func usageBoolValue(log map[string]interface{}, key string) bool {
	v, ok := log[key]
	if !ok || v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case int:
		return b == 1
	case int64:
		return b == 1
	default:
		return false
	}
}

func (s *SQLiteStorage) GetUsageStats() map[string]interface{} {
	return s.GetUsageStatsWithFilter(UsageFilter{})
}

//nolint:gocritic // Keep value receiver for compatibility with existing callers.
func (s *SQLiteStorage) GetUsageStatsWithFilter(filter UsageFilter) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	whereClause, args := buildUsageWhereClause(filter)

	var totalRequests, totalTokens, cacheHits, cacheMisses int64
	var totalLatency int64
	var savedTokens, savedRequests int64

	statsQuery := `SELECT
		COUNT(*),
		COALESCE(SUM(tokens), 0),
		COALESCE(SUM(latency_ms), 0),
		COALESCE(SUM(CASE WHEN cache_hit = 1 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN cache_hit = 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN cache_hit = 1 AND success = 1 THEN tokens ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN cache_hit = 1 AND success = 1 THEN 1 ELSE 0 END), 0)
	FROM usage_logs WHERE ` + whereClause
	if err := s.db.QueryRow(statsQuery, args...).Scan(
		&totalRequests,
		&totalTokens,
		&totalLatency,
		&cacheHits,
		&cacheMisses,
		&savedTokens,
		&savedRequests,
	); err != nil {
		sqliteLogger.WithError(err).Warn("query usage stats failed")
		return map[string]interface{}{
			"total_requests": int64(0),
			"total_tokens":   int64(0),
			"cache_hits":     int64(0),
			"cache_misses":   int64(0),
			"saved_tokens":   int64(0),
			"saved_requests": int64(0),
			"cache_hit_rate": float64(0),
			"avg_latency_ms": int64(0),
			"model_stats":    map[string]map[string]int64{},
		}
	}

	var avgLatency int64
	if totalRequests > 0 {
		avgLatency = totalLatency / totalRequests
	}

	var cacheHitRate float64
	totalCache := cacheHits + cacheMisses
	if totalCache > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCache) * 100
	}

	modelStatsArgs := append([]interface{}{}, args...)
	//nolint:gosec // whereClause is built from fixed whitelisted fragments.
	rows, err := s.db.Query(`SELECT model, COUNT(*) as requests, COALESCE(SUM(tokens), 0) as tokens
		FROM usage_logs WHERE `+whereClause+` GROUP BY model`, modelStatsArgs...)
	modelStats := make(map[string]map[string]int64)
	if err != nil {
		sqliteLogger.WithError(err).Warn("query model usage stats failed")
	} else {
		defer rows.Close()
		for rows.Next() {
			var model string
			var requests, tokens int64
			if scanErr := rows.Scan(&model, &requests, &tokens); scanErr != nil {
				sqliteLogger.WithError(scanErr).Warn("scan model usage stats failed")
				continue
			}
			modelStats[model] = map[string]int64{"requests": requests, "tokens": tokens}
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			sqliteLogger.WithError(rowsErr).Warn("iterate model usage stats failed")
		}
	}

	return map[string]interface{}{
		"total_requests": totalRequests,
		"total_tokens":   totalTokens,
		"cache_hits":     cacheHits,
		"cache_misses":   cacheMisses,
		"saved_tokens":   savedTokens,
		"saved_requests": savedRequests,
		"cache_hit_rate": cacheHitRate,
		"avg_latency_ms": avgLatency,
		"model_stats":    modelStats,
	}
}

func (s *SQLiteStorage) GetProviderUsageStats() ([]ProviderUsageStat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT
		provider,
		COUNT(*) as requests,
		COALESCE(SUM(tokens), 0) as tokens,
		COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) as success_count,
		COALESCE(SUM(latency_ms), 0) as total_latency
	FROM usage_logs
	WHERE provider IS NOT NULL AND provider != ''
	GROUP BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]ProviderUsageStat, 0)
	for rows.Next() {
		var provider string
		var requests, tokens, successCount, totalLatency int64
		if err := rows.Scan(&provider, &requests, &tokens, &successCount, &totalLatency); err != nil {
			return nil, err
		}
		avgLatency := int64(0)
		successRate := 0.0
		if requests > 0 {
			avgLatency = totalLatency / requests
			successRate = float64(successCount) / float64(requests) * 100
		}
		stats = append(stats, ProviderUsageStat{
			Provider:    provider,
			Requests:    requests,
			Tokens:      tokens,
			SuccessRate: successRate,
			AvgLatency:  avgLatency,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *SQLiteStorage) SaveDashboardSummary(summary DashboardSummary) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	updatedAt := summary.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	_, err := s.db.Exec(`INSERT INTO dashboard_summary (
		id, total_requests, requests_today, success_count, failure_count, total_latency, total_tokens, updated_at
	) VALUES (1, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		total_requests = excluded.total_requests,
		requests_today = excluded.requests_today,
		success_count = excluded.success_count,
		failure_count = excluded.failure_count,
		total_latency = excluded.total_latency,
		total_tokens = excluded.total_tokens,
		updated_at = excluded.updated_at`,
		summary.TotalRequests,
		summary.RequestsToday,
		summary.SuccessCount,
		summary.FailureCount,
		summary.TotalLatency,
		summary.TotalTokens,
		updatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStorage) LoadDashboardSummary() (DashboardSummary, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var summary DashboardSummary
	var updatedAt sql.NullString

	err := s.db.QueryRow(`SELECT total_requests, requests_today, success_count, failure_count, total_latency, total_tokens, updated_at
		FROM dashboard_summary WHERE id = 1`).
		Scan(
			&summary.TotalRequests,
			&summary.RequestsToday,
			&summary.SuccessCount,
			&summary.FailureCount,
			&summary.TotalLatency,
			&summary.TotalTokens,
			&updatedAt,
		)
	if err == sql.ErrNoRows {
		return summary, false, nil
	}
	if err != nil {
		return summary, false, err
	}
	if updatedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, updatedAt.String); parseErr == nil {
			summary.UpdatedAt = t
		}
	}
	return summary, true, nil
}

func (s *SQLiteStorage) SaveDashboardModelStat(model string, requests, tokens int64) error {
	if strings.TrimSpace(model) == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO dashboard_models (model, requests, tokens, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(model) DO UPDATE SET
			requests = excluded.requests,
			tokens = excluded.tokens,
			updated_at = excluded.updated_at`,
		model,
		requests,
		tokens,
		now,
	)
	return err
}

func (s *SQLiteStorage) LoadDashboardModelStats() ([]DashboardModelStat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT model, requests, tokens FROM dashboard_models`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]DashboardModelStat, 0)
	for rows.Next() {
		var item DashboardModelStat
		if err := rows.Scan(&item.Model, &item.Requests, &item.Tokens); err != nil {
			return nil, err
		}
		stats = append(stats, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func (s *SQLiteStorage) SaveDashboardTrend(trend DashboardTrend) error {
	if trend.Timestamp <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`INSERT INTO dashboard_trends (ts, requests, success, failed, latency)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(ts) DO UPDATE SET
			requests = excluded.requests,
			success = excluded.success,
			failed = excluded.failed,
			latency = excluded.latency`,
		trend.Timestamp,
		trend.Requests,
		trend.Success,
		trend.Failed,
		trend.Latency,
	)
	return err
}

func (s *SQLiteStorage) LoadDashboardTrends(limit int) ([]DashboardTrend, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT ts, requests, success, failed, latency FROM dashboard_trends ORDER BY ts DESC`
	args := []interface{}{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trends := make([]DashboardTrend, 0)
	for rows.Next() {
		var item DashboardTrend
		if err := rows.Scan(&item.Timestamp, &item.Requests, &item.Success, &item.Failed, &item.Latency); err != nil {
			return nil, err
		}
		trends = append(trends, item)
	}

	for i, j := 0, len(trends)-1; i < j; i, j = i+1, j-1 {
		trends[i], trends[j] = trends[j], trends[i]
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return trends, nil
}

func (s *SQLiteStorage) UpdateDashboardAlertAcknowledged(alertID string, acknowledged bool) error {
	if strings.TrimSpace(alertID) == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`UPDATE dashboard_alerts SET acknowledged = ? WHERE id = ?`,
		boolToInt(acknowledged),
		alertID,
	)
	return err
}

//nolint:gocritic // Keep value parameter for compatibility with existing callers.
func (s *SQLiteStorage) SaveDashboardAlert(alert DashboardAlert) error {
	if strings.TrimSpace(alert.ID) == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`INSERT INTO dashboard_alerts (
		id, type, level, message, account_id, provider, timestamp, acknowledged
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		type = excluded.type,
		level = excluded.level,
		message = excluded.message,
		account_id = excluded.account_id,
		provider = excluded.provider,
		timestamp = excluded.timestamp,
		acknowledged = excluded.acknowledged`,
		alert.ID,
		alert.Type,
		alert.Level,
		alert.Message,
		alert.AccountID,
		alert.Provider,
		alert.Timestamp,
		boolToInt(alert.Acknowledged),
	)
	return err
}

func (s *SQLiteStorage) LoadDashboardAlerts(limit int) ([]DashboardAlert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, type, level, message, account_id, provider, timestamp, acknowledged
		FROM dashboard_alerts ORDER BY timestamp DESC`
	args := []interface{}{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]DashboardAlert, 0)
	for rows.Next() {
		var item DashboardAlert
		var acknowledged int
		var accountID, provider sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.Level,
			&item.Message,
			&accountID,
			&provider,
			&item.Timestamp,
			&acknowledged,
		); err != nil {
			return nil, err
		}
		item.AccountID = accountID.String
		item.Provider = provider.String
		item.Acknowledged = intToBool(acknowledged)
		alerts = append(alerts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i, j := 0, len(alerts)-1; i < j; i, j = i+1, j-1 {
		alerts[i], alerts[j] = alerts[j], alerts[i]
	}

	return alerts, nil
}

// GetDB returns the underlying database connection.
func (s *SQLiteStorage) GetDB() *sql.DB {
	return s.db
}
