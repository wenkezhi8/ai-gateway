package datastore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUnifiedStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_store.json")

	store, err := NewUnifiedStore(storePath)
	require.NoError(t, err)
	require.NotNil(t, store)

	assert.NotNil(t, store.Accounts)
	assert.NotNil(t, store.ModelScores)
	assert.NotNil(t, store.RouterConfig)
	assert.Equal(t, "auto", store.RouterConfig.DefaultStrategy)
}

func TestUnifiedStore_ModelScore(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	score := &ModelScoreRecord{
		Model:        "gpt-4",
		Provider:     "openai",
		QualityScore: 90,
		SpeedScore:   80,
		CostScore:    70,
		Enabled:      true,
	}

	err = store.SetModelScore("gpt-4", score)
	require.NoError(t, err)

	retrieved := store.GetModelScore("gpt-4")
	require.NotNil(t, retrieved)
	assert.Equal(t, 90, retrieved.QualityScore)
	assert.True(t, retrieved.IsCustom)
	assert.NotEmpty(t, retrieved.CreatedAt)
	assert.NotEmpty(t, retrieved.UpdatedAt)

	allScores := store.GetAllModelScores()
	assert.Contains(t, allScores, "gpt-4")

	enabledScores := store.GetEnabledModelScores()
	assert.Contains(t, enabledScores, "gpt-4")
}

func TestUnifiedStore_DeleteModelScore(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	store.SetModelScore("gpt-4", &ModelScoreRecord{Model: "gpt-4", Enabled: true})

	err = store.DeleteModelScore("gpt-4")
	require.NoError(t, err)

	assert.Nil(t, store.GetModelScore("gpt-4"))
	assert.True(t, store.IsModelDeleted("gpt-4"))
}

func TestUnifiedStore_RestoreModel(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	store.DeleteModelScore("gpt-4")

	err = store.RestoreModel("gpt-4")
	require.NoError(t, err)
	assert.False(t, store.IsModelDeleted("gpt-4"))
}

func TestUnifiedStore_Account(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	account := &AccountRecord{
		ID:       "acc-1",
		Provider: "openai",
		APIKey:   "sk-test",
		Enabled:  true,
	}

	err = store.SetAccount("acc-1", account)
	require.NoError(t, err)

	retrieved := store.GetAccount("acc-1")
	require.NotNil(t, retrieved)
	assert.Equal(t, "openai", retrieved.Provider)
	assert.NotEmpty(t, retrieved.CreatedAt)

	allAccounts := store.GetAllAccounts()
	assert.Contains(t, allAccounts, "acc-1")
}

func TestUnifiedStore_DeleteAccount(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	store.SetAccount("acc-1", &AccountRecord{ID: "acc-1"})

	err = store.DeleteAccount("acc-1")
	require.NoError(t, err)
	assert.Nil(t, store.GetAccount("acc-1"))
}

func TestUnifiedStore_ProviderDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	err = store.SetProviderDefault("openai", "gpt-4")
	require.NoError(t, err)

	assert.Equal(t, "gpt-4", store.GetProviderDefault("openai"))

	defaults := map[string]string{
		"anthropic": "claude-3",
		"openai":    "gpt-4o",
	}
	err = store.SetAllProviderDefaults(defaults)
	require.NoError(t, err)

	allDefaults := store.GetAllProviderDefaults()
	assert.Equal(t, "claude-3", allDefaults["anthropic"])
}

func TestUnifiedStore_RouterConfig(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	config := &RouterConfigRecord{
		DefaultStrategy: "round_robin",
		DefaultModel:    "claude-3",
		UseAutoMode:     false,
	}

	err = store.SetRouterConfig(config)
	require.NoError(t, err)

	retrieved := store.GetRouterConfig()
	assert.Equal(t, "round_robin", retrieved.DefaultStrategy)
	assert.Equal(t, "claude-3", retrieved.DefaultModel)
}

func TestUnifiedStore_User(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	user := &UserRecord{
		Username:     "testuser",
		PasswordHash: "hashed",
		Role:         "admin",
	}

	err = store.SetUser("testuser", user)
	require.NoError(t, err)

	retrieved := store.GetUser("testuser")
	require.NotNil(t, retrieved)
	assert.Equal(t, "admin", retrieved.Role)

	allUsers := store.GetAllUsers()
	assert.Contains(t, allUsers, "testuser")

	err = store.DeleteUser("testuser")
	require.NoError(t, err)
	assert.Nil(t, store.GetUser("testuser"))
}

func TestUnifiedStore_APIKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	key := &APIKeyRecord{
		ID:          "key-1",
		Name:        "Test Key",
		Key:         "sk-xxx",
		Permissions: "read,write",
		Enabled:     true,
	}

	err = store.SetAPIKey("key-1", key)
	require.NoError(t, err)

	retrieved := store.GetAPIKey("key-1")
	require.NotNil(t, retrieved)
	assert.Equal(t, "Test Key", retrieved.Name)

	allKeys := store.GetAllAPIKeys()
	assert.Contains(t, allKeys, "key-1")

	err = store.DeleteAPIKey("key-1")
	require.NoError(t, err)
	assert.Nil(t, store.GetAPIKey("key-1"))
}

func TestUnifiedStore_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	store.SetAccount("acc-1", &AccountRecord{ID: "acc-1"})
	store.SetModelScore("gpt-4", &ModelScoreRecord{Model: "gpt-4"})
	store.SetUser("user1", &UserRecord{Username: "user1"})

	stats := store.GetStats()
	assert.Equal(t, 1, stats["accounts"])
	assert.Equal(t, 1, stats["model_scores"])
	assert.Equal(t, 1, stats["users"])
}

func TestUnifiedStore_ExportImport(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	store.SetAccount("acc-1", &AccountRecord{ID: "acc-1", Provider: "openai"})
	store.SetModelScore("gpt-4", &ModelScoreRecord{Model: "gpt-4"})

	data, err := store.Export()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	newStore, err := NewUnifiedStore(filepath.Join(tmpDir, "store2.json"))
	require.NoError(t, err)

	err = newStore.Import(data)
	require.NoError(t, err)

	assert.NotNil(t, newStore.GetAccount("acc-1"))
	assert.NotNil(t, newStore.GetModelScore("gpt-4"))
}

func TestUnifiedStore_Save(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "save_test.json")
	store, err := NewUnifiedStore(storePath)
	require.NoError(t, err)

	store.SetAccount("acc-1", &AccountRecord{ID: "acc-1"})

	data, err := os.ReadFile(storePath)
	require.NoError(t, err)

	var loaded UnifiedStore
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.Contains(t, loaded.Accounts, "acc-1")
}

func TestUnifiedStore_EnabledModelScores_ExcludesDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "store.json"))
	require.NoError(t, err)

	store.SetModelScore("gpt-4", &ModelScoreRecord{Model: "gpt-4", Enabled: true})
	store.SetModelScore("claude-3", &ModelScoreRecord{Model: "claude-3", Enabled: true})
	store.DeleteModelScore("gpt-4")

	enabled := store.GetEnabledModelScores()
	assert.NotContains(t, enabled, "gpt-4")
	assert.Contains(t, enabled, "claude-3")
}

func TestGetStore(t *testing.T) {
	globalStore = nil
	globalStoreOnce = sync.Once{}

	store1 := GetStore()
	require.NotNil(t, store1)

	store2 := GetStore()
	assert.Equal(t, store1, store2)
}

func TestUnifiedStore_LoadExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "existing.json")

	existingData := map[string]interface{}{
		"accounts": map[string]interface{}{
			"acc-1": map[string]interface{}{
				"id":       "acc-1",
				"provider": "openai",
				"enabled":  true,
			},
		},
		"model_scores": map[string]interface{}{},
		"router_config": map[string]interface{}{
			"default_strategy": "weighted",
			"default_model":    "gpt-4",
			"use_auto_mode":    false,
		},
	}

	data, _ := json.MarshalIndent(existingData, "", "  ")
	os.WriteFile(storePath, data, 0644)

	store, err := NewUnifiedStore(storePath)
	require.NoError(t, err)

	acc := store.GetAccount("acc-1")
	require.NotNil(t, acc)
	assert.Equal(t, "openai", acc.Provider)

	config := store.GetRouterConfig()
	assert.Equal(t, "weighted", config.DefaultStrategy)
}
