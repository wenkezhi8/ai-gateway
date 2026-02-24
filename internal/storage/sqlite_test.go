package storage

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStorage(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)
	require.NotNil(t, storage)

	assert.NotNil(t, storage.accounts)
	assert.NotNil(t, storage.modelScores)
	assert.NotNil(t, storage.apiKeys)
	assert.NotNil(t, storage.config)
}

func TestMemoryStorage_SaveAndGetAccount(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	account := map[string]interface{}{
		"id":       "acc1",
		"name":     "Test Account",
		"provider": "openai",
	}

	err = storage.SaveAccount(account)
	require.NoError(t, err)

	accounts, err := storage.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "acc1", accounts[0]["id"])
	assert.Equal(t, "Test Account", accounts[0]["name"])
}

func TestMemoryStorage_SaveAccount_NoID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	account := map[string]interface{}{
		"name": "Test Account",
	}

	err = storage.SaveAccount(account)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestMemoryStorage_DeleteAccount(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	account := map[string]interface{}{"id": "acc1", "name": "Test"}
	err = storage.SaveAccount(account)
	require.NoError(t, err)

	err = storage.DeleteAccount("acc1")
	require.NoError(t, err)

	accounts, err := storage.GetAccounts()
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestMemoryStorage_SaveAndGetModelScore(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	score := map[string]interface{}{
		"score":    8.5,
		"provider": "openai",
	}

	err = storage.SaveModelScore("gpt-4", score)
	require.NoError(t, err)

	scores, err := storage.GetModelScores()
	require.NoError(t, err)
	require.Contains(t, scores, "gpt-4")
	assert.Equal(t, 8.5, scores["gpt-4"]["score"])
}

func TestMemoryStorage_LogAndGetUsage(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	log1 := map[string]interface{}{"model": "gpt-4", "tokens": 100}
	log2 := map[string]interface{}{"model": "gpt-3.5", "tokens": 50}

	err = storage.LogUsage(log1)
	require.NoError(t, err)

	err = storage.LogUsage(log2)
	require.NoError(t, err)

	logs, err := storage.GetUsageLogs(10, 0)
	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestMemoryStorage_GetUsageLogs_Offset(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		storage.LogUsage(map[string]interface{}{"index": i})
	}

	logs, err := storage.GetUsageLogs(2, 0)
	require.NoError(t, err)
	assert.Len(t, logs, 2)

	logs, err = storage.GetUsageLogs(2, 10)
	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestMemoryStorage_Config(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	err = storage.SetConfig("key1", "value1")
	require.NoError(t, err)

	val, err := storage.GetConfig("key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)

	val, err = storage.GetConfig("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, val)
}

func TestMemoryStorage_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	storage.SaveAccount(map[string]interface{}{"id": "acc1"})
	storage.SaveModelScore("model1", map[string]interface{}{"score": 8})
	storage.LogUsage(map[string]interface{}{"tokens": 100})

	stats := storage.GetStats()
	assert.Equal(t, 1, stats["accounts"])
	assert.Equal(t, 1, stats["models"])
	assert.Equal(t, 1, stats["usage"])
}

func TestMemoryStorage_APIKeys(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	key := map[string]interface{}{
		"id":   "key1",
		"name": "Test Key",
	}

	err = storage.SaveAPIKey(key)
	require.NoError(t, err)

	keys, err := storage.GetAPIKeys()
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "key1", keys[0]["id"])

	err = storage.DeleteAPIKey("key1")
	require.NoError(t, err)

	keys, err = storage.GetAPIKeys()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestMemoryStorage_SaveAPIKey_NoID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	key := map[string]interface{}{"name": "Test Key"}

	err = storage.SaveAPIKey(key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestMemoryStorage_Close(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	storage.SaveAccount(map[string]interface{}{"id": "acc1"})

	err = storage.Close()
	require.NoError(t, err)

	assert.FileExists(t, path)
}

func TestMemoryStorage_PersistAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage1, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	storage1.SaveAccount(map[string]interface{}{"id": "acc1", "name": "Test"})
	storage1.SetConfig("key1", "value1")
	storage1.Close()

	storage2, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	accounts, err := storage2.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "acc1", accounts[0]["id"])

	val, err := storage2.GetConfig("key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestMemoryStorage_Load_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)
	require.NotNil(t, storage)

	accounts, err := storage.GetAccounts()
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestMemoryStorage_LogUsage_Truncate(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_storage.json")

	storage, err := NewMemoryStorage(StorageConfig{Path: path})
	require.NoError(t, err)

	for i := 0; i < 10001; i++ {
		storage.LogUsage(map[string]interface{}{"index": i})
	}

	stats := storage.GetStats()
	assert.LessOrEqual(t, stats["usage"], 10000)
}
