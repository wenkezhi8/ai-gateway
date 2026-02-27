package datastore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func waitStoreFile(t *testing.T, storePath string) UnifiedStore {
	t.Helper()

	var loaded UnifiedStore
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(storePath)
		if err != nil {
			return false
		}
		if err := json.Unmarshal(data, &loaded); err != nil {
			return false
		}
		return true
	}, 2*time.Second, 20*time.Millisecond)

	return loaded
}

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
	require.NoError(t, store.Flush())

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

func TestUnifiedStore_AsyncWrite(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "async_store.json")

	store, err := NewUnifiedStore(storePath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	err = store.SetProviderDefault("openai", "gpt-4o")
	require.NoError(t, err)

	assert.Equal(t, "gpt-4o", store.GetProviderDefault("openai"))

	require.Eventually(t, func() bool {
		loaded := waitStoreFile(t, storePath)
		return loaded.ProviderDefaults["openai"] == "gpt-4o"
	}, 2*time.Second, 20*time.Millisecond)
}

func TestUnifiedStore_BatchWrite(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "batch_store.json")

	store, err := NewUnifiedStore(storePath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	for i := 0; i < 30; i++ {
		key := fmt.Sprintf("acc-%d", i)
		err = store.SetAccount(key, &AccountRecord{ID: key, Provider: "openai", Enabled: true})
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		store.mu.RLock()
		defer store.mu.RUnlock()
		return len(store.pendingOps) >= 2
	}, time.Second, 10*time.Millisecond)

	err = store.Flush()
	require.NoError(t, err)

	loaded := waitStoreFile(t, storePath)
	assert.Len(t, loaded.Accounts, 30)
}

func TestUnifiedStore_GracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "shutdown_store.json")

	store, err := NewUnifiedStore(storePath)
	require.NoError(t, err)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("model-%d", i)
		err = store.SetModelScore(key, &ModelScoreRecord{Model: key, Enabled: true})
		require.NoError(t, err)
	}

	require.NoError(t, store.Close())

	reloaded, err := NewUnifiedStore(storePath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reloaded.Close() })

	assert.Len(t, reloaded.GetAllModelScores(), 50)
}

func TestUnifiedStore_WriteBuffer(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "buffer_store.json")

	store, err := NewUnifiedStore(storePath)
	require.NoError(t, err)

	const total = 400
	var wg sync.WaitGroup
	wg.Add(total)

	start := time.Now()
	for i := 0; i < total; i++ {
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("provider-%d", i)
			_ = store.SetProviderDefault(key, "gpt-4o")
		}(i)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("write buffer blocked SetProviderDefault calls")
	}

	assert.Less(t, time.Since(start), 2*time.Second)
	require.NoError(t, store.Close())

	reloaded, err := NewUnifiedStore(storePath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reloaded.Close() })

	assert.Len(t, reloaded.GetAllProviderDefaults(), total)
}

func BenchmarkUnifiedStore_SyncWrite(b *testing.B) {
	tmpDir := b.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "sync_store.json"))
	require.NoError(b, err)
	b.Cleanup(func() { _ = store.Close() })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("provider-%d", i)
		require.NoError(b, store.SetProviderDefault(key, "gpt-4o"))
		require.NoError(b, store.Flush())
	}
}

func BenchmarkUnifiedStore_AsyncWrite(b *testing.B) {
	tmpDir := b.TempDir()
	store, err := NewUnifiedStore(filepath.Join(tmpDir, "async_bench_store.json"))
	require.NoError(b, err)
	b.Cleanup(func() { _ = store.Close() })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("provider-%d", i)
		require.NoError(b, store.SetProviderDefault(key, "gpt-4o"))
	}
	require.NoError(b, store.Close())
}
