package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var storageLogger = logrus.New()

type StorageConfig struct {
	Path string
}

type MemoryStorage struct {
	mu sync.RWMutex

	accounts    map[string]map[string]interface{}
	modelScores map[string]map[string]interface{}
	apiKeys     map[string]map[string]interface{}
	usageLogs   []map[string]interface{}
	config      map[string]string

	path string
}

var (
	globalStorage     *MemoryStorage
	globalStorageOnce sync.Once
)

func GetSQLite() *MemoryStorage {
	globalStorageOnce.Do(func() {
		path := os.Getenv("AI_GATEWAY_DB_PATH")
		if path == "" {
			path = "data/storage.json"
		}
		var err error
		globalStorage, err = NewMemoryStorage(StorageConfig{Path: path})
		if err != nil {
			storageLogger.Fatalf("Failed to initialize storage: %v", err)
		}
	})
	return globalStorage
}

func NewMemoryStorage(config StorageConfig) (*MemoryStorage, error) {
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	s := &MemoryStorage{
		accounts:    make(map[string]map[string]interface{}),
		modelScores: make(map[string]map[string]interface{}),
		apiKeys:     make(map[string]map[string]interface{}),
		usageLogs:   make([]map[string]interface{}, 0),
		config:      make(map[string]string),
		path:        config.Path,
	}

	s.load()

	storageLogger.Info("Memory storage initialized")
	return s, nil
}

func (s *MemoryStorage) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}

	var saved struct {
		Accounts    map[string]map[string]interface{} `json:"accounts"`
		ModelScores map[string]map[string]interface{} `json:"model_scores"`
		ApiKeys     map[string]map[string]interface{} `json:"api_keys"`
		Config      map[string]string                 `json:"config"`
	}

	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}

	if saved.Accounts != nil {
		s.accounts = saved.Accounts
	}
	if saved.ModelScores != nil {
		s.modelScores = saved.ModelScores
	}
	if saved.ApiKeys != nil {
		s.apiKeys = saved.ApiKeys
	}
	if saved.Config != nil {
		s.config = saved.Config
	}
}

func (s *MemoryStorage) save() {
	data := struct {
		Accounts    map[string]map[string]interface{} `json:"accounts"`
		ModelScores map[string]map[string]interface{} `json:"model_scores"`
		ApiKeys     map[string]map[string]interface{} `json:"api_keys"`
		Config      map[string]string                 `json:"config"`
	}{
		Accounts:    s.accounts,
		ModelScores: s.modelScores,
		ApiKeys:     s.apiKeys,
		Config:      s.config,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		storageLogger.WithError(err).Error("Failed to marshal storage data")
		return
	}

	if err := os.WriteFile(s.path, jsonData, 0644); err != nil {
		storageLogger.WithError(err).Error("Failed to save storage data")
	}
}

func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.save()
	return nil
}

func (s *MemoryStorage) SaveAccount(account map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := account["id"].(string)
	if !ok {
		return fmt.Errorf("account id is required")
	}

	account["updated_at"] = time.Now().Format(time.RFC3339)
	s.accounts[id] = account
	s.save()

	return nil
}

func (s *MemoryStorage) GetAccounts() ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]map[string]interface{}, 0, len(s.accounts))
	for _, acc := range s.accounts {
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (s *MemoryStorage) DeleteAccount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.accounts, id)
	s.save()
	return nil
}

func (s *MemoryStorage) SaveModelScore(model string, score map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	score["updated_at"] = time.Now().Format(time.RFC3339)
	s.modelScores[model] = score
	s.save()
	return nil
}

func (s *MemoryStorage) GetModelScores() (map[string]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]map[string]interface{})
	for k, v := range s.modelScores {
		result[k] = v
	}
	return result, nil
}

func (s *MemoryStorage) LogUsage(log map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log["id"] = len(s.usageLogs) + 1
	log["created_at"] = time.Now().Format(time.RFC3339)
	s.usageLogs = append(s.usageLogs, log)

	if len(s.usageLogs) > 10000 {
		s.usageLogs = s.usageLogs[len(s.usageLogs)-5000:]
	}

	return nil
}

func (s *MemoryStorage) GetUsageLogs(limit int, offset int) ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if offset >= len(s.usageLogs) {
		return []map[string]interface{}{}, nil
	}

	end := offset + limit
	if end > len(s.usageLogs) {
		end = len(s.usageLogs)
	}

	result := make([]map[string]interface{}, 0, end-offset)
	for i := len(s.usageLogs) - 1 - offset; i >= 0 && i >= len(s.usageLogs)-end; i-- {
		result = append(result, s.usageLogs[i])
	}

	return result, nil
}

func (s *MemoryStorage) GetConfig(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config[key], nil
}

func (s *MemoryStorage) SetConfig(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config[key] = value
	s.save()
	return nil
}

func (s *MemoryStorage) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"accounts": len(s.accounts),
		"models":   len(s.modelScores),
		"usage":    len(s.usageLogs),
		"api_keys": len(s.apiKeys),
	}
}

func (s *MemoryStorage) SaveAPIKey(key map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := key["id"].(string)
	if !ok {
		return fmt.Errorf("api key id is required")
	}

	key["created_at"] = time.Now().Format(time.RFC3339)
	s.apiKeys[id] = key
	s.save()
	return nil
}

func (s *MemoryStorage) GetAPIKeys() ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]map[string]interface{}, 0, len(s.apiKeys))
	for _, k := range s.apiKeys {
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *MemoryStorage) DeleteAPIKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.apiKeys, id)
	s.save()
	return nil
}
