package storage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLiteStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	assert.FileExists(t, dbPath)
}

func TestNewSQLiteStorage_InvalidPath(t *testing.T) {
	_, err := NewSQLiteStorage("/nonexistent/path/test.db")
	assert.Error(t, err)
}

func TestSQLiteStorage_Ping(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	err = storage.Ping()
	assert.NoError(t, err)
}

func TestSQLiteStorage_SaveAndGetAccount(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	account := map[string]interface{}{
		"id":       "test-1",
		"name":     "Test Account",
		"provider": "openai",
		"api_key":  "sk-test",
		"base_url": "https://api.openai.com",
		"models":   "gpt-4,gpt-3.5",
		"enabled":  true,
	}

	err = storage.SaveAccount(account)
	require.NoError(t, err)

	accounts, err := storage.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)

	assert.Equal(t, "test-1", accounts[0]["id"])
	assert.Equal(t, "Test Account", accounts[0]["name"])
	assert.Equal(t, "openai", accounts[0]["provider"])
	assert.Equal(t, true, accounts[0]["enabled"])
}

func TestSQLiteStorage_SaveAccount_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	account := map[string]interface{}{
		"id":       "test-2",
		"name":     "Disabled Account",
		"provider": "anthropic",
		"enabled":  false,
	}

	err = storage.SaveAccount(account)
	require.NoError(t, err)

	accounts, err := storage.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, false, accounts[0]["enabled"])
}

func TestSQLiteStorage_DeleteAccount(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	account := map[string]interface{}{
		"id":       "test-3",
		"name":     "To Delete",
		"provider": "openai",
	}

	err = storage.SaveAccount(account)
	require.NoError(t, err)

	err = storage.DeleteAccount("test-3")
	require.NoError(t, err)

	accounts, err := storage.GetAccounts()
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestSQLiteStorage_SaveAndGetModelScore(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	score := map[string]interface{}{
		"score":          85.5,
		"avg_latency":    int64(150),
		"success_rate":   0.95,
		"total_requests": int64(1000),
	}

	err = storage.SaveModelScore("gpt-4", score)
	require.NoError(t, err)

	scores, err := storage.GetModelScores()
	require.NoError(t, err)
	require.Contains(t, scores, "gpt-4")

	assert.Equal(t, 85.5, scores["gpt-4"]["score"])
	assert.Equal(t, int64(150), scores["gpt-4"]["avg_latency"])
	assert.Equal(t, 0.95, scores["gpt-4"]["success_rate"])
}

func TestSQLiteStorage_GetAndSetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	value, err := storage.GetConfig("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, value)

	err = storage.SetConfig("test-key", "test-value")
	require.NoError(t, err)

	value, err = storage.GetConfig("test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", value)
}

func TestSQLiteStorage_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	storage.SaveAccount(map[string]interface{}{
		"id":       "acc-1",
		"name":     "Account 1",
		"provider": "openai",
	})

	storage.SaveModelScore("model-1", map[string]interface{}{
		"score": 90.0,
	})

	stats := storage.GetStats()
	assert.Equal(t, int64(1), stats["accounts"])
	assert.Equal(t, int64(1), stats["models"])
	assert.Greater(t, stats["db_size"].(int64), int64(0))
}

func TestSQLiteStorage_Vacuum(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	err = storage.Vacuum()
	assert.NoError(t, err)
}

func TestSQLiteStorage_UpdateAccount(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewSQLiteStorage(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	account := map[string]interface{}{
		"id":       "update-1",
		"name":     "Original Name",
		"provider": "openai",
	}
	storage.SaveAccount(account)

	updated := map[string]interface{}{
		"id":       "update-1",
		"name":     "Updated Name",
		"provider": "anthropic",
	}
	err = storage.SaveAccount(updated)
	require.NoError(t, err)

	accounts, err := storage.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "Updated Name", accounts[0]["name"])
	assert.Equal(t, "anthropic", accounts[0]["provider"])
}

func TestGetSQLiteStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "global_test.db")

	os.Setenv("AI_GATEWAY_SQLITE_PATH", dbPath)
	defer os.Unsetenv("AI_GATEWAY_SQLITE_PATH")

	sqliteInstance = nil
	sqliteInstanceOnce = sync.Once{}

	storage := GetSQLiteStorage()
	require.NotNil(t, storage)
	defer storage.Close()

	assert.Equal(t, storage, GetSQLiteStorage())
}
