package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-gateway/internal/models"
)

func TestSQLiteStorage_Accounts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Create and Get Account", func(t *testing.T) {
		acc := &models.AccountRecord{
			ID:       "test-1",
			Provider: "openai",
			APIKey:   "sk-test",
			Priority: 1,
			Enabled:  true,
		}
		err := store.SaveAccount(acc)
		require.NoError(t, err)

		got, err := store.GetAccount("test-1")
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
		assert.Equal(t, acc.Provider, got.Provider)
		assert.Equal(t, acc.APIKey, got.APIKey)
		assert.Equal(t, acc.Priority, got.Priority)
		assert.Equal(t, acc.Enabled, got.Enabled)
	})

	t.Run("Get Non-Existing Account", func(t *testing.T) {
		_, err := store.GetAccount("non-existent")
		require.NoError(t, err)
	})

	t.Run("Get All Accounts", func(t *testing.T) {
		err := store.SaveAccount(&models.AccountRecord{
			ID:       "test-2",
			Provider: "anthropic",
			APIKey:   "sk-ant",
			Enabled:  true,
		})
		require.NoError(t, err)

		accounts, err := store.GetAllAccounts()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(accounts), 2)
	})

	t.Run("Delete Account", func(t *testing.T) {
		err := store.SaveAccount(&models.AccountRecord{
			ID:       "test-delete",
			Provider: "openai",
			APIKey:   "sk-del",
			Enabled:  true,
		})
		require.NoError(t, err)

		err = store.DeleteAccount("test-delete")
		require.NoError(t, err)

		_, err = store.GetAccount("test-delete")
		require.NoError(t, err)
	})
}

func TestSQLiteStorage_ModelScores(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Create and Get Model Score", func(t *testing.T) {
		score := &models.ModelScoreRecord{
			Model:        "gpt-4",
			Provider:     "openai",
			QualityScore: 85,
			SpeedScore:   90,
			CostScore:    70,
			Enabled:      true,
			IsCustom:     true,
		}
		err := store.SaveModelScore("gpt-4", score)
		require.NoError(t, err)

		got, err := store.GetModelScore("gpt-4")
		require.NoError(t, err)
		assert.Equal(t, score.Model, got.Model)
		assert.Equal(t, score.Provider, got.Provider)
		assert.Equal(t, score.QualityScore, got.QualityScore)
		assert.Equal(t, score.SpeedScore, got.SpeedScore)
		assert.Equal(t, score.CostScore, got.CostScore)
		assert.Equal(t, score.Enabled, got.Enabled)
		assert.Equal(t, score.IsCustom, got.IsCustom)
	})

	t.Run("Get Non-Existing Model Score", func(t *testing.T) {
		_, err := store.GetModelScore("non-existent")
		require.NoError(t, err)
	})

	t.Run("Get All Model Scores", func(t *testing.T) {
		err := store.SaveModelScore("claude-3", &models.ModelScoreRecord{
			Model:        "claude-3",
			Provider:     "anthropic",
			QualityScore: 80,
			SpeedScore:   85,
			CostScore:    75,
			Enabled:      true,
		})
		require.NoError(t, err)

		scores, err := store.GetAllModelScores()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(scores), 2)
	})

	t.Run("Get Enabled Model Scores", func(t *testing.T) {
		err := store.SaveModelScore("gpt-3.5", &models.ModelScoreRecord{
			Model:        "gpt-3.5",
			Provider:     "openai",
			QualityScore: 70,
			SpeedScore:   80,
			CostScore:    80,
			Enabled:      true,
		})
		require.NoError(t, err)

		enabledScores, err := store.GetEnabledModelScores()
		require.NoError(t, err)
		assert.Greater(t, len(enabledScores), 0)

		for _, score := range enabledScores {
			assert.True(t, score.Enabled)
		}
	})

	t.Run("Delete Model Score", func(t *testing.T) {
		err := store.SaveModelScore("delete-model", &models.ModelScoreRecord{
			Model:        "delete-model",
			Provider:     "openai",
			QualityScore: 75,
			SpeedScore:   80,
			CostScore:    75,
			Enabled:      true,
		})
		require.NoError(t, err)

		err = store.DeleteModelScore("delete-model")
		require.NoError(t, err)

		_, err = store.GetModelScore("delete-model")
		require.NoError(t, err)
	})

	t.Run("Mark Model Deleted", func(t *testing.T) {
		err := store.MarkModelDeleted("deleted-model")
		require.NoError(t, err)

		deleted, err := store.IsModelDeleted("deleted-model")
		require.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("Restore Model", func(t *testing.T) {
		err := store.MarkModelDeleted("restore-model")
		require.NoError(t, err)
		err = store.RestoreModel("restore-model")
		require.NoError(t, err)

		deleted, err := store.IsModelDeleted("restore-model")
		require.NoError(t, err)
		assert.False(t, deleted)
	})

	t.Run("Get All Deleted Models", func(t *testing.T) {
		err := store.MarkModelDeleted("deleted-1")
		require.NoError(t, err)
		err = store.MarkModelDeleted("deleted-2")
		require.NoError(t, err)

		deleted, err := store.GetAllDeletedModels()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(deleted), 2)
	})
}

func TestSQLiteStorage_Users(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Create and Get User", func(t *testing.T) {
		user := &models.UserRecord{
			Username:     "admin",
			PasswordHash: "hashed-password",
			Role:         "admin",
			Email:        "admin@example.com",
		}
		err := store.SaveUser("admin", user)
		require.NoError(t, err)

		got, err := store.GetUser("admin")
		require.NoError(t, err)
		assert.Equal(t, user.Username, got.Username)
		assert.Equal(t, user.PasswordHash, got.PasswordHash)
		assert.Equal(t, user.Role, got.Role)
		assert.Equal(t, user.Email, got.Email)
	})

	t.Run("Get Non-Existing User", func(t *testing.T) {
		_, err := store.GetUser("non-existent")
		require.NoError(t, err)
	})

	t.Run("Get All Users", func(t *testing.T) {
		err := store.SaveUser("user1", &models.UserRecord{
			Username:     "user1",
			PasswordHash: "hash1",
			Role:         "user",
		})
		require.NoError(t, err)

		users, err := store.GetAllUsers()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 2)
	})

	t.Run("Delete User", func(t *testing.T) {
		err := store.SaveUser("delete-user", &models.UserRecord{
			Username:     "delete-user",
			PasswordHash: "hash",
			Role:         "user",
		})
		require.NoError(t, err)

		err = store.DeleteUser("delete-user")
		require.NoError(t, err)

		_, err = store.GetUser("delete-user")
		require.NoError(t, err)
	})
}

func TestSQLiteStorage_APIKeys(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Create and Get API Key", func(t *testing.T) {
		key := &models.APIKeyRecord{
			Name:        "Test Key",
			Key:         "sk-test-key",
			Permissions: "read,write",
			Enabled:     true,
			LastUsedAt:  "2024-01-01T00:00:00Z",
			ExpiresAt:   "2025-01-01T00:00:00Z",
		}
		err := store.SaveAPIKey("key-1", key)
		require.NoError(t, err)

		got, err := store.GetAPIKey("key-1")
		require.NoError(t, err)
		assert.Equal(t, key.Name, got.Name)
		assert.Equal(t, key.Key, got.Key)
		assert.Equal(t, key.Permissions, got.Permissions)
		assert.Equal(t, key.Enabled, got.Enabled)
	})

	t.Run("Get Non-Existing API Key", func(t *testing.T) {
		_, err := store.GetAPIKey("non-existent")
		require.NoError(t, err)
	})

	t.Run("Get All API Keys", func(t *testing.T) {
		err := store.SaveAPIKey("key-2", &models.APIKeyRecord{
			Name:        "Key 2",
			Key:         "sk-2",
			Permissions: "read",
			Enabled:     true,
		})
		require.NoError(t, err)

		keys, err := store.GetAllAPIKeys()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 2)
	})

	t.Run("Delete API Key", func(t *testing.T) {
		err := store.SaveAPIKey("delete-key", &models.APIKeyRecord{
			Name:        "Delete Key",
			Key:         "sk-delete",
			Permissions: "admin",
			Enabled:     true,
		})
		require.NoError(t, err)

		err = store.DeleteAPIKey("delete-key")
		require.NoError(t, err)

		_, err = store.GetAPIKey("delete-key")
		require.NoError(t, err)
	})
}

func TestSQLiteStorage_Config(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Provider Defaults", func(t *testing.T) {
		err := store.SetProviderDefault("openai", "gpt-4")
		require.NoError(t, err)

		model, err := store.GetProviderDefault("openai")
		require.NoError(t, err)
		assert.Equal(t, "gpt-4", model)
	})

	t.Run("Get Non-Existing Provider Default", func(t *testing.T) {
		model, err := store.GetProviderDefault("non-existent")
		require.NoError(t, err)
		assert.Empty(t, model)
	})

	t.Run("Get All Provider Defaults", func(t *testing.T) {
		err := store.SetProviderDefault("anthropic", "claude-3")
		require.NoError(t, err)
		err = store.SetProviderDefault("deepseek", "deepseek-chat")
		require.NoError(t, err)

		defaults, err := store.GetAllProviderDefaults()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(defaults), 3)
	})

	t.Run("Router Config", func(t *testing.T) {
		config := &models.RouterConfigRecord{
			DefaultStrategy: "auto",
			DefaultModel:    "gpt-4",
			UseAutoMode:     true,
		}
		err := store.SetRouterConfig(config)
		require.NoError(t, err)

		got, err := store.GetRouterConfig()
		require.NoError(t, err)
		assert.Equal(t, config.DefaultStrategy, got.DefaultStrategy)
		assert.Equal(t, config.DefaultModel, got.DefaultModel)
		assert.Equal(t, config.UseAutoMode, got.UseAutoMode)
	})
}

func TestSQLiteStorage_Stats(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Get Stats", func(t *testing.T) {
		stats := store.GetStats()
		assert.NotEmpty(t, stats)
		assert.Contains(t, stats, "accounts")
		assert.Contains(t, stats, "models")
		assert.Contains(t, stats, "users")
		assert.Contains(t, stats, "api_keys")
		assert.Contains(t, stats, "feedback")
		assert.Contains(t, stats, "db_size")
	})
}

func TestSQLiteStorage_Concurrent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	done := make(chan bool)
	errCh := make(chan error, 20)

	t.Run("Concurrent Account Operations", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				acc := &models.AccountRecord{
					ID:       string(rune(idx)),
					Provider: "openai",
					APIKey:   "sk-test",
					Enabled:  true,
				}
				if err := store.SaveAccount(acc); err != nil {
					errCh <- err
					return
				}
				if _, err := store.GetAccount(acc.ID); err != nil {
					errCh <- err
				}
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
		select {
		case err := <-errCh:
			require.NoError(t, err)
		default:
		}
	})

	t.Run("Concurrent Model Score Operations", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				score := &models.ModelScoreRecord{
					Model:        string(rune(idx)),
					Provider:     "openai",
					QualityScore: 80 + idx,
					SpeedScore:   80 + idx,
					CostScore:    80 + idx,
					Enabled:      true,
				}
				if err := store.SaveModelScore(score.Model, score); err != nil {
					errCh <- err
					return
				}
				if _, err := store.GetModelScore(score.Model); err != nil {
					errCh <- err
				}
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
		select {
		case err := <-errCh:
			require.NoError(t, err)
		default:
		}
	})
}

func TestSQLiteStorage_Feedback(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Save and Get Feedback", func(t *testing.T) {
		feedback := &models.FeedbackRecord{
			RequestID:  "req-1",
			Model:      "gpt-4",
			Provider:   "openai",
			TaskType:   "code",
			Rating:     5,
			Comment:    "Great!",
			LatencyMs:  1000,
			TokensUsed: 1000,
			CacheHit:   false,
			CreatedAt:  time.Now().Format(time.RFC3339),
		}
		err := store.SaveFeedback(feedback)
		require.NoError(t, err)

		feedbacks, err := store.GetFeedback(10, 0)
		require.NoError(t, err)
		assert.Greater(t, len(feedbacks), 0)

		assert.Equal(t, feedback.RequestID, feedbacks[0].RequestID)
		assert.Equal(t, feedback.Model, feedbacks[0].Model)
		assert.Equal(t, feedback.Provider, feedbacks[0].Provider)
		assert.Equal(t, feedback.TaskType, feedbacks[0].TaskType)
		assert.Equal(t, feedback.Rating, feedbacks[0].Rating)
		assert.Equal(t, feedback.Comment, feedbacks[0].Comment)
	})

	t.Run("Get Feedback Stats", func(t *testing.T) {
		err := store.SaveFeedback(&models.FeedbackRecord{
			RequestID:  "req-2",
			Model:      "gpt-4",
			Provider:   "openai",
			Rating:     4,
			LatencyMs:  800,
			TokensUsed: 800,
		})
		require.NoError(t, err)

		stats, err := store.GetFeedbackStats()
		require.NoError(t, err)
		assert.NotEmpty(t, stats)
	})
}

func TestSQLiteStorage_Export(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Export data", func(t *testing.T) {
		err := store.SaveAccount(&models.AccountRecord{
			ID:       "export-1",
			Provider: "openai",
			APIKey:   "sk-export",
			Enabled:  true,
		})
		require.NoError(t, err)

		data, err := store.Export()
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Contains(t, string(data), "accounts")
	})
}

func TestSQLiteStorage_UsageLogs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Log and Query Usage", func(t *testing.T) {
		err := store.LogUsage(map[string]interface{}{
			"request_id":     "req-usage-1",
			"timestamp":      time.Now().UnixMilli(),
			"model":          "gpt-4o-mini",
			"provider":       "openai",
			"user_id":        "1",
			"api_key":        "sk****test",
			"tokens":         int64(120),
			"input_tokens":   int64(80),
			"output_tokens":  int64(40),
			"latency_ms":     int64(350),
			"ttft_ms":        int64(120),
			"cache_hit":      true,
			"success":        true,
			"task_type":      "code",
			"difficulty":     "medium",
			"experiment_tag": "exp-a",
			"domain_tag":     "finance",
			"usage_source":   "actual",
		})
		require.NoError(t, err)

		err = store.LogUsage(map[string]interface{}{
			"request_id":     "req-usage-2",
			"timestamp":      time.Now().UnixMilli(),
			"model":          "gpt-4o-mini",
			"provider":       "openai",
			"tokens":         int64(90),
			"input_tokens":   int64(50),
			"output_tokens":  int64(40),
			"latency_ms":     int64(420),
			"ttft_ms":        int64(100),
			"cache_hit":      false,
			"success":        true,
			"experiment_tag": "exp-b",
			"domain_tag":     "general",
			"usage_source":   "estimated",
		})
		require.NoError(t, err)

		logs, err := store.GetUsageLogsWithFilter(UsageFilter{
			Model:         "gpt-4o-mini",
			Provider:      "openai",
			ExperimentTag: "exp-a",
			DomainTag:     "finance",
		}, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 1)

		assert.Equal(t, "gpt-4o-mini", logs[0]["model"])
		assert.Equal(t, "openai", logs[0]["provider"])
		assert.Equal(t, int64(120), logs[0]["tokens"])
		assert.Equal(t, int64(80), logs[0]["input_tokens"])
		assert.Equal(t, int64(40), logs[0]["output_tokens"])
		assert.Equal(t, int64(350), logs[0]["latency_ms"])
		assert.Equal(t, int64(120), logs[0]["ttft_ms"])
		assert.Equal(t, true, logs[0]["cache_hit"])
		assert.Equal(t, true, logs[0]["success"])
		assert.Equal(t, "exp-a", logs[0]["experiment_tag"])
		assert.Equal(t, "finance", logs[0]["domain_tag"])
		assert.Equal(t, "actual", logs[0]["usage_source"])

		tagFiltered, err := store.GetUsageLogsWithFilter(UsageFilter{
			ExperimentTag: "exp-b",
			DomainTag:     "general",
		}, 10, 0)
		require.NoError(t, err)
		require.Len(t, tagFiltered, 1)
		assert.Equal(t, "exp-b", tagFiltered[0]["experiment_tag"])
		assert.Equal(t, "general", tagFiltered[0]["domain_tag"])
		assert.Equal(t, "estimated", tagFiltered[0]["usage_source"])
	})

	t.Run("Usage Stats", func(t *testing.T) {
		stats := store.GetUsageStats()
		assert.Equal(t, int64(2), stats["total_requests"])
		assert.Equal(t, int64(210), stats["total_tokens"])
		assert.Equal(t, int64(1), stats["cache_hits"])
		assert.Equal(t, int64(1), stats["cache_misses"])

		providerStats, err := store.GetProviderUsageStats()
		require.NoError(t, err)
		require.Len(t, providerStats, 1)
		assert.Equal(t, "openai", providerStats[0].Provider)
		assert.Equal(t, int64(2), providerStats[0].Requests)
		assert.Equal(t, int64(210), providerStats[0].Tokens)
		assert.Equal(t, 100.0, providerStats[0].SuccessRate)
		assert.Equal(t, int64(385), providerStats[0].AvgLatency)
	})

	t.Run("Usage Stats With Filter Should Include SavedTokens", func(t *testing.T) {
		now := time.Now().UnixMilli()

		require.NoError(t, store.LogUsage(map[string]interface{}{
			"request_id": "req-usage-3",
			"timestamp":  now - 10*60*1000,
			"model":      "qwen2.5:3b",
			"provider":   "ollama",
			"tokens":     int64(300),
			"latency_ms": int64(20),
			"cache_hit":  true,
			"success":    true,
			"task_type":  "chat",
		}))

		require.NoError(t, store.LogUsage(map[string]interface{}{
			"request_id": "req-usage-4",
			"timestamp":  now - 20*60*1000,
			"model":      "qwen2.5:3b",
			"provider":   "ollama",
			"tokens":     int64(120),
			"latency_ms": int64(35),
			"cache_hit":  true,
			"success":    false,
			"task_type":  "chat",
		}))

		require.NoError(t, store.LogUsage(map[string]interface{}{
			"request_id": "req-usage-5",
			"timestamp":  now - 30*60*1000,
			"model":      "qwen2.5:3b",
			"provider":   "ollama",
			"tokens":     int64(90),
			"latency_ms": int64(45),
			"cache_hit":  false,
			"success":    true,
			"task_type":  "qa",
		}))

		require.NoError(t, store.LogUsage(map[string]interface{}{
			"request_id": "req-usage-6",
			"timestamp":  now - 10*24*60*60*1000,
			"model":      "qwen2.5:3b",
			"provider":   "ollama",
			"tokens":     int64(500),
			"latency_ms": int64(60),
			"cache_hit":  true,
			"success":    true,
			"task_type":  "qa",
		}))

		filteredStats := store.GetUsageStatsWithFilter(UsageFilter{
			Model:     "qwen2.5:3b",
			TaskType:  "chat",
			StartTime: now - 24*60*60*1000,
		})

		assert.Equal(t, int64(2), filteredStats["total_requests"])
		assert.Equal(t, int64(420), filteredStats["total_tokens"])
		assert.Equal(t, int64(2), filteredStats["cache_hits"])
		assert.Equal(t, int64(0), filteredStats["cache_misses"])
		assert.Equal(t, int64(300), filteredStats["saved_tokens"])
		assert.Equal(t, int64(1), filteredStats["saved_requests"])
		assert.Equal(t, 100.0, filteredStats["cache_hit_rate"])
	})
}

func TestSQLiteStorage_UsageStatsWithFilter_EmptyDataset(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test-empty-usage-stats.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	stats := store.GetUsageStatsWithFilter(UsageFilter{
		Model:    "none",
		TaskType: "none",
	})

	assert.Equal(t, int64(0), stats["total_requests"])
	assert.Equal(t, int64(0), stats["total_tokens"])
	assert.Equal(t, int64(0), stats["cache_hits"])
	assert.Equal(t, int64(0), stats["cache_misses"])
	assert.Equal(t, int64(0), stats["saved_tokens"])
	assert.Equal(t, int64(0), stats["saved_requests"])
	assert.Equal(t, float64(0), stats["cache_hit_rate"])
	assert.Equal(t, int64(0), stats["avg_latency_ms"])
}

func TestSQLiteStorage_ClearUsageLogs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test-clear-usage-logs.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.LogUsage(map[string]interface{}{
		"request_id": "req-clear-1",
		"timestamp":  time.Now().UnixMilli(),
		"model":      "gpt-4o-mini",
		"provider":   "openai",
		"tokens":     int64(120),
		"cache_hit":  true,
		"success":    true,
	}))
	require.NoError(t, store.LogUsage(map[string]interface{}{
		"request_id": "req-clear-2",
		"timestamp":  time.Now().UnixMilli(),
		"model":      "gpt-4o-mini",
		"provider":   "openai",
		"tokens":     int64(80),
		"cache_hit":  false,
		"success":    true,
	}))
	deleted, err := store.ClearUsageLogs()
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	logs, err := store.GetUsageLogsWithFilter(UsageFilter{}, 10, 0)
	require.NoError(t, err)
	assert.Len(t, logs, 0)
}
