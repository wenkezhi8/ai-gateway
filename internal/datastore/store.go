package datastore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var storeLogger = logrus.New()

const dataDir = "data"
const unifiedStoreFile = "data/store.json"

type UnifiedStore struct {
	mu sync.RWMutex

	Accounts         map[string]*AccountRecord    `json:"accounts"`
	ModelScores      map[string]*ModelScoreRecord `json:"model_scores"`
	ProviderDefaults map[string]string            `json:"provider_defaults"`
	RouterConfig     *RouterConfigRecord          `json:"router_config"`
	APIKeys          map[string]*APIKeyRecord     `json:"api_keys"`
	DeletedModels    map[string]bool              `json:"deleted_models"`
	Users            map[string]*UserRecord       `json:"users"`

	lastSaved time.Time
	filePath  string
}

type AccountRecord struct {
	ID           string `json:"id"`
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key"`
	Priority     int    `json:"priority"`
	Enabled      bool   `json:"enabled"`
	QuotaLimit   int64  `json:"quota_limit"`
	QuotaUsed    int64  `json:"quota_used"`
	QuotaResetAt string `json:"quota_reset_at,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ModelScoreRecord struct {
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	QualityScore int    `json:"quality_score"`
	SpeedScore   int    `json:"speed_score"`
	CostScore    int    `json:"cost_score"`
	Enabled      bool   `json:"enabled"`
	IsCustom     bool   `json:"is_custom"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type RouterConfigRecord struct {
	DefaultStrategy string `json:"default_strategy"`
	DefaultModel    string `json:"default_model"`
	UseAutoMode     bool   `json:"use_auto_mode"`
}

type APIKeyRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Permissions string `json:"permissions"`
	Enabled     bool   `json:"enabled"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type UserRecord struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	Email        string `json:"email,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

var (
	globalStore     *UnifiedStore
	globalStoreOnce sync.Once
)

func GetStore() *UnifiedStore {
	globalStoreOnce.Do(func() {
		var err error
		globalStore, err = NewUnifiedStore(unifiedStoreFile)
		if err != nil {
			storeLogger.Fatalf("Failed to initialize unified store: %v", err)
		}
	})
	return globalStore
}

func NewUnifiedStore(filePath string) (*UnifiedStore, error) {
	s := &UnifiedStore{
		filePath:         filePath,
		Accounts:         make(map[string]*AccountRecord),
		ModelScores:      make(map[string]*ModelScoreRecord),
		ProviderDefaults: make(map[string]string),
		RouterConfig:     &RouterConfigRecord{DefaultStrategy: "auto", DefaultModel: "deepseek-chat", UseAutoMode: true},
		APIKeys:          make(map[string]*APIKeyRecord),
		DeletedModels:    make(map[string]bool),
		Users:            make(map[string]*UserRecord),
	}

	if err := s.load(); err != nil {
		storeLogger.WithError(err).Warn("Failed to load store, starting fresh")
	}

	storeLogger.Info("Unified store initialized")
	return s, nil
}

func (s *UnifiedStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return s.migrateFromOldFiles()
		}
		return err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return err
	}

	if s.Accounts == nil {
		s.Accounts = make(map[string]*AccountRecord)
	}
	if s.ModelScores == nil {
		s.ModelScores = make(map[string]*ModelScoreRecord)
	}
	if s.ProviderDefaults == nil {
		s.ProviderDefaults = make(map[string]string)
	}
	if s.APIKeys == nil {
		s.APIKeys = make(map[string]*APIKeyRecord)
	}
	if s.DeletedModels == nil {
		s.DeletedModels = make(map[string]bool)
	}
	if s.Users == nil {
		s.Users = make(map[string]*UserRecord)
	}
	if s.RouterConfig == nil {
		s.RouterConfig = &RouterConfigRecord{DefaultStrategy: "auto", DefaultModel: "deepseek-chat", UseAutoMode: true}
	}

	return nil
}

func (s *UnifiedStore) migrateFromOldFiles() error {
	storeLogger.Info("Migrating from old files...")

	if data, err := os.ReadFile("data/accounts.json"); err == nil {
		var accounts map[string]*AccountRecord
		if err := json.Unmarshal(data, &accounts); err == nil {
			s.Accounts = accounts
		}
	}

	if data, err := os.ReadFile("data/model_scores.json"); err == nil {
		var scores map[string]*ModelScoreRecord
		if err := json.Unmarshal(data, &scores); err == nil {
			s.ModelScores = scores
		}
	}

	if data, err := os.ReadFile("data/provider_defaults.json"); err == nil {
		var defaults map[string]string
		if err := json.Unmarshal(data, &defaults); err == nil {
			s.ProviderDefaults = defaults
		}
	}

	if data, err := os.ReadFile("data/router_config.json"); err == nil {
		var config RouterConfigRecord
		if err := json.Unmarshal(data, &config); err == nil {
			s.RouterConfig = &config
		}
	}

	if data, err := os.ReadFile("data/api_keys.json"); err == nil {
		var keys map[string]*APIKeyRecord
		if err := json.Unmarshal(data, &keys); err == nil {
			s.APIKeys = keys
		}
	}

	return s.save()
}

func (s *UnifiedStore) save() error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return err
	}

	s.lastSaved = time.Now()
	return nil
}

func (s *UnifiedStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.save()
}

func (s *UnifiedStore) GetModelScore(model string) *ModelScoreRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ModelScores[model]
}

func (s *UnifiedStore) GetAllModelScores() map[string]*ModelScoreRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*ModelScoreRecord)
	for k, v := range s.ModelScores {
		result[k] = v
	}
	return result
}

func (s *UnifiedStore) GetEnabledModelScores() map[string]*ModelScoreRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*ModelScoreRecord)
	for k, v := range s.ModelScores {
		if v.Enabled && !s.DeletedModels[k] {
			result[k] = v
		}
	}
	return result
}

func (s *UnifiedStore) SetModelScore(model string, score *ModelScoreRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	score.UpdatedAt = now
	if _, exists := s.ModelScores[model]; !exists {
		score.CreatedAt = now
	}
	score.IsCustom = true

	s.ModelScores[model] = score
	delete(s.DeletedModels, model)

	return s.save()
}

func (s *UnifiedStore) DeleteModelScore(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.ModelScores, model)
	s.DeletedModels[model] = true

	return s.save()
}

func (s *UnifiedStore) IsModelDeleted(model string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DeletedModels[model]
}

func (s *UnifiedStore) RestoreModel(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.DeletedModels, model)
	return s.save()
}

func (s *UnifiedStore) GetProviderDefault(provider string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ProviderDefaults[provider]
}

func (s *UnifiedStore) GetAllProviderDefaults() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range s.ProviderDefaults {
		result[k] = v
	}
	return result
}

func (s *UnifiedStore) SetProviderDefault(provider, model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ProviderDefaults[provider] = model
	return s.save()
}

func (s *UnifiedStore) SetAllProviderDefaults(defaults map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ProviderDefaults = defaults
	return s.save()
}

func (s *UnifiedStore) GetRouterConfig() *RouterConfigRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RouterConfig
}

func (s *UnifiedStore) SetRouterConfig(config *RouterConfigRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RouterConfig = config
	return s.save()
}

func (s *UnifiedStore) GetAccount(id string) *AccountRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Accounts[id]
}

func (s *UnifiedStore) GetAllAccounts() map[string]*AccountRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*AccountRecord)
	for k, v := range s.Accounts {
		result[k] = v
	}
	return result
}

func (s *UnifiedStore) SetAccount(id string, account *AccountRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	account.UpdatedAt = now
	if _, exists := s.Accounts[id]; !exists {
		account.CreatedAt = now
	}

	s.Accounts[id] = account
	return s.save()
}

func (s *UnifiedStore) DeleteAccount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Accounts, id)
	return s.save()
}

func (s *UnifiedStore) GetUser(username string) *UserRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Users[username]
}

func (s *UnifiedStore) GetAllUsers() map[string]*UserRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*UserRecord)
	for k, v := range s.Users {
		result[k] = v
	}
	return result
}

func (s *UnifiedStore) SetUser(username string, user *UserRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	user.UpdatedAt = now
	if _, exists := s.Users[username]; !exists {
		user.CreatedAt = now
	}

	s.Users[username] = user
	return s.save()
}

func (s *UnifiedStore) DeleteUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Users, username)
	return s.save()
}

func (s *UnifiedStore) GetAPIKey(id string) *APIKeyRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.APIKeys[id]
}

func (s *UnifiedStore) GetAllAPIKeys() map[string]*APIKeyRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*APIKeyRecord)
	for k, v := range s.APIKeys {
		result[k] = v
	}
	return result
}

func (s *UnifiedStore) SetAPIKey(id string, key *APIKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	if _, exists := s.APIKeys[id]; !exists {
		key.CreatedAt = now
	}

	s.APIKeys[id] = key
	return s.save()
}

func (s *UnifiedStore) DeleteAPIKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.APIKeys, id)
	return s.save()
}

func (s *UnifiedStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"accounts":          len(s.Accounts),
		"model_scores":      len(s.ModelScores),
		"deleted_models":    len(s.DeletedModels),
		"provider_defaults": len(s.ProviderDefaults),
		"api_keys":          len(s.APIKeys),
		"users":             len(s.Users),
		"last_saved":        s.lastSaved.Format(time.RFC3339),
	}
}

func (s *UnifiedStore) Export() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.MarshalIndent(s, "", "  ")
}

func (s *UnifiedStore) Import(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := json.Unmarshal(data, s); err != nil {
		return err
	}

	return s.save()
}
