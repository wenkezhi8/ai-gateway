package storage

import (
	"ai-gateway/internal/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	}

	for _, schema := range schemas {
		if _, err := s.db.Exec(schema); err != nil {
			return fmt.Errorf("schema error: %w", err)
		}
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

	rows, err := s.db.Query(`SELECT model, provider, quality_score, speed_score, cost_score, enabled, is_custom, created_at, updated_at FROM model_scores`)
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
	return scores, nil
}

func (s *SQLiteStorage) GetEnabledModelScores() (map[string]*models.ModelScoreRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT model, provider, quality_score, speed_score, cost_score, enabled, is_custom, created_at, updated_at FROM model_scores WHERE enabled = 1 AND model NOT IN (SELECT model FROM deleted_models)`)
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
	return scores, nil
}

func (s *SQLiteStorage) DeleteModelScore(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
	return feedbacks, nil
}

func (s *SQLiteStorage) GetFeedbackStats() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	var totalCount int64
	s.db.QueryRow(`SELECT COUNT(*) FROM feedback`).Scan(&totalCount)
	stats["total_count"] = totalCount

	var avgRating sql.NullFloat64
	s.db.QueryRow(`SELECT AVG(rating) FROM feedback WHERE rating > 0`).Scan(&avgRating)
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
	stats["model_stats"] = modelStats

	return stats, nil
}

func (s *SQLiteStorage) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var accountCount, modelCount, userCount, apiKeyCount, feedbackCount int64
	s.db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&accountCount)
	s.db.QueryRow(`SELECT COUNT(*) FROM model_scores`).Scan(&modelCount)
	s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount)
	s.db.QueryRow(`SELECT COUNT(*) FROM api_keys`).Scan(&apiKeyCount)
	s.db.QueryRow(`SELECT COUNT(*) FROM feedback`).Scan(&feedbackCount)

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

	accounts, _ := s.GetAllAccounts()
	data.Accounts = accounts

	scores, _ := s.GetAllModelScores()
	data.ModelScores = scores

	defaults, _ := s.GetAllProviderDefaults()
	data.ProviderDefaults = defaults

	config, _ := s.GetRouterConfig()
	data.RouterConfig = config

	keys, _ := s.GetAllAPIKeys()
	data.APIKeys = keys

	users, _ := s.GetAllUsers()
	data.Users = users

	deleted, _ := s.GetAllDeletedModels()
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

func intFromInt64(i int64) int {
	return int(i)
}

type UsageFilter struct {
	Model     string
	Provider  string
	StartTime int64
	EndTime   int64
}

type UsageLog struct {
	ID           int64  `json:"id"`
	Timestamp    int64  `json:"timestamp"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	UserID       string `json:"user_id"`
	APIKey       string `json:"api_key"`
	Tokens       int64  `json:"tokens"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	LatencyMs    int64  `json:"latency_ms"`
	TTFTMs       int64  `json:"ttft_ms"`
	CacheHit     bool   `json:"cache_hit"`
	Success      bool   `json:"success"`
	ErrorType    string `json:"error_type"`
	TaskType     string `json:"task_type"`
	Difficulty   string `json:"difficulty"`
	CreatedAt    string `json:"created_at"`
}

func GetSQLite() *SQLiteStorage {
	return GetSQLiteStorage()
}

func (s *SQLiteStorage) LogUsage(log map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO usage_logs (request_id, model, provider, tokens, latency_ms, cache_hit, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		log["request_id"], log["model"], log["provider"], log["tokens"], log["latency_ms"], log["cache_hit"], now)
	return err
}

func (s *SQLiteStorage) GetUsageLogsWithFilter(filter UsageFilter, limit, offset int) ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, model, provider, tokens, latency_ms, cache_hit, created_at FROM usage_logs WHERE 1=1`
	args := []interface{}{}

	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}
	if filter.Provider != "" {
		query += " AND provider = ?"
		args = append(args, filter.Provider)
	}
	if filter.StartTime > 0 {
		query += " AND created_at >= ?"
		args = append(args, time.UnixMilli(filter.StartTime).Format(time.RFC3339))
	}
	if filter.EndTime > 0 {
		query += " AND created_at <= ?"
		args = append(args, time.UnixMilli(filter.EndTime).Format(time.RFC3339))
	}

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
		var model, provider, createdAt string
		var tokens, latencyMs int64
		var cacheHitInt int
		if err := rows.Scan(&id, &model, &provider, &tokens, &latencyMs, &cacheHitInt, &createdAt); err != nil {
			return nil, err
		}
		logs = append(logs, map[string]interface{}{
			"id":         id,
			"model":      model,
			"provider":   provider,
			"tokens":     tokens,
			"latency_ms": latencyMs,
			"cache_hit":  cacheHitInt == 1,
			"created_at": createdAt,
		})
	}
	return logs, nil
}

func (s *SQLiteStorage) GetUsageStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalRequests, totalTokens, cacheHits, cacheMisses int64
	var totalLatency int64

	s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(tokens), 0), COALESCE(SUM(latency_ms), 0) FROM usage_logs`).Scan(&totalRequests, &totalTokens, &totalLatency)
	s.db.QueryRow(`SELECT COUNT(*) FROM usage_logs WHERE cache_hit = 1`).Scan(&cacheHits)
	s.db.QueryRow(`SELECT COUNT(*) FROM usage_logs WHERE cache_hit = 0`).Scan(&cacheMisses)

	var avgLatency int64
	if totalRequests > 0 {
		avgLatency = totalLatency / totalRequests
	}

	var cacheHitRate float64
	totalCache := cacheHits + cacheMisses
	if totalCache > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCache) * 100
	}

	rows, err := s.db.Query(`SELECT model, COUNT(*) as requests, COALESCE(SUM(tokens), 0) as tokens FROM usage_logs GROUP BY model`)
	if err != nil {
		rows.Close()
	}
	modelStats := make(map[string]map[string]int64)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var model string
			var requests, tokens int64
			if err := rows.Scan(&model, &requests, &tokens); err == nil {
				modelStats[model] = map[string]int64{"requests": requests, "tokens": tokens}
			}
		}
	}

	return map[string]interface{}{
		"total_requests": totalRequests,
		"total_tokens":   totalTokens,
		"cache_hits":     cacheHits,
		"cache_misses":   cacheMisses,
		"cache_hit_rate": cacheHitRate,
		"avg_latency_ms": avgLatency,
		"model_stats":    modelStats,
	}
}
