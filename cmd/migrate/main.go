package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ai-gateway/internal/models"
	"ai-gateway/internal/storage"
	gatewaylogger "ai-gateway/pkg/logger"
)

var logger = gatewaylogger.WithField("component", "migrate")

func main() {
	sourceDir := flag.String("source", "data", "Source directory containing JSON files")
	dbPath := flag.String("db", "data/ai-gateway.db", "Target SQLite database path")
	verify := flag.Bool("verify", false, "Verify migration without writing")
	backup := flag.Bool("backup", true, "Create backup of existing database")
	flag.Parse()

	logger.Info("Starting JSON to SQLite migration...")
	logger.Infof("Source directory: %s", *sourceDir)
	logger.Infof("Target database: %s", *dbPath)

	if *verify {
		logger.Info("Verification mode - no changes will be made")
		if err := verifyMigration(*sourceDir); err != nil {
			logger.Fatalf("Verification failed: %v", err)
		}
		logger.Info("Verification completed successfully")
		return
	}

	if *backup {
		if err := createBackup(*dbPath); err != nil {
			logger.Warnf("Failed to create backup: %v", err)
		}
	}

	store, err := storage.NewSQLiteStorage(*dbPath)
	if err != nil {
		logger.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer store.Close()

	stats := &MigrationStats{}

	if err := migrateAccounts(store, *sourceDir, stats); err != nil {
		logger.Errorf("Failed to migrate accounts: %v", err)
	}

	if err := migrateModelScores(store, *sourceDir, stats); err != nil {
		logger.Errorf("Failed to migrate model scores: %v", err)
	}

	if err := migrateProviderDefaults(store, *sourceDir, stats); err != nil {
		logger.Errorf("Failed to migrate provider defaults: %v", err)
	}

	if err := migrateRouterConfig(store, *sourceDir, stats); err != nil {
		logger.Errorf("Failed to migrate router config: %v", err)
	}

	if err := migrateAPIKeys(store, *sourceDir, stats); err != nil {
		logger.Errorf("Failed to migrate API keys: %v", err)
	}

	if err := migrateUsers(store, *sourceDir, stats); err != nil {
		logger.Errorf("Failed to migrate users: %v", err)
	}

	if err := migrateDeletedModels(store, *sourceDir, stats); err != nil {
		logger.Errorf("Failed to migrate deleted models: %v", err)
	}

	printStats(stats)
	logger.Info("Migration completed!")
}

type MigrationStats struct {
	Accounts         int
	ModelScores      int
	ProviderDefaults int
	RouterConfig     bool
	APIKeys          int
	Users            int
	DeletedModels    int
	Errors           []string
}

func (s *MigrationStats) AddError(msg string) {
	s.Errors = append(s.Errors, msg)
}

func printStats(stats *MigrationStats) {
	fmt.Println("\n=== Migration Statistics ===")
	fmt.Printf("Accounts:         %d\n", stats.Accounts)
	fmt.Printf("Model Scores:     %d\n", stats.ModelScores)
	fmt.Printf("Provider Defaults: %d\n", stats.ProviderDefaults)
	fmt.Printf("Router Config:    %v\n", stats.RouterConfig)
	fmt.Printf("API Keys:         %d\n", stats.APIKeys)
	fmt.Printf("Users:            %d\n", stats.Users)
	fmt.Printf("Deleted Models:   %d\n", stats.DeletedModels)
	if len(stats.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range stats.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}
	fmt.Println("============================")
}

func createBackup(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil
	}

	backupPath := dbPath + ".backup." + time.Now().Format("20060102-150405")
	if err := copyFile(dbPath, backupPath); err != nil {
		return err
	}
	logger.Infof("Backup created: %s", backupPath)
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}

func verifyMigration(sourceDir string) error {
	files := []string{
		"accounts.json",
		"model_scores.json",
		"provider_defaults.json",
		"router_config.json",
		"api_keys.json",
		"users.json",
	}

	found := 0
	for _, f := range files {
		path := filepath.Join(sourceDir, f)
		if _, err := os.Stat(path); err == nil {
			logger.Infof("Found: %s", path)
			found++
		}
	}

	if found == 0 {
		return fmt.Errorf("no JSON files found in %s", sourceDir)
	}

	logger.Infof("Found %d JSON files to migrate", found)
	return nil
}

func readOptionalFile(path, missingMessage string) (data []byte, found bool, err error) {
	data, err = os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug(missingMessage)
			return nil, false, nil
		}
		return nil, false, err
	}

	return data, true, nil
}

func migrateJSONMap[T any](sourceDir, fileName string) (records map[string]*T, found bool, err error) {
	path := filepath.Join(sourceDir, fileName)
	data, found, err := readOptionalFile(path, fileName+" not found, skipping")
	if err != nil || !found {
		return nil, found, err
	}

	if err := json.Unmarshal(data, &records); err != nil {
		return nil, true, fmt.Errorf("parse %s: %w", fileName, err)
	}

	return records, true, nil
}

func ensureTimestamps(createdAt, updatedAt *string) {
	now := time.Now().Format(time.RFC3339)
	if *createdAt == "" {
		*createdAt = now
	}
	if *updatedAt == "" {
		*updatedAt = now
	}
}

func migrateRecords[T any](records map[string]*T, migrate func(string, *T) error, onError func(string, error)) int {
	migrated := 0
	for key, record := range records {
		if err := migrate(key, record); err != nil {
			onError(key, err)
			continue
		}
		migrated++
	}

	return migrated
}

func migrateLabeledRecords[T any](
	sourceDir, fileName string,
	migrate func(string, *T) error,
	onError func(string, error),
) (int, error) {
	records, found, err := migrateJSONMap[T](sourceDir, fileName)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}

	return migrateRecords(records, migrate, onError), nil
}

func migrateAccounts(store *storage.SQLiteStorage, sourceDir string, stats *MigrationStats) error {
	path := filepath.Join(sourceDir, "accounts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("accounts.json not found, skipping")
			return nil
		}
		return err
	}

	var accounts map[string]*models.AccountRecord
	if err := json.Unmarshal(data, &accounts); err != nil {
		return fmt.Errorf("parse accounts.json: %w", err)
	}

	for id, acc := range accounts {
		acc.ID = id
		if acc.CreatedAt == "" {
			acc.CreatedAt = time.Now().Format(time.RFC3339)
		}
		if acc.UpdatedAt == "" {
			acc.UpdatedAt = time.Now().Format(time.RFC3339)
		}
		if err := store.SaveAccount(acc); err != nil {
			stats.AddError(fmt.Sprintf("account %s: %v", id, err))
			continue
		}
		stats.Accounts++
	}

	logger.Infof("Migrated %d accounts", stats.Accounts)
	return nil
}

//nolint:dupl // model/user migrations share the same flow with type-specific persistence.
func migrateModelScores(store *storage.SQLiteStorage, sourceDir string, stats *MigrationStats) error {
	migrated, err := migrateLabeledRecords(
		sourceDir,
		"model_scores.json",
		func(model string, score *models.ModelScoreRecord) error {
			score.Model = model
			ensureTimestamps(&score.CreatedAt, &score.UpdatedAt)
			return store.SaveModelScore(model, score)
		},
		func(model string, err error) {
			stats.AddError(fmt.Sprintf("model_score %s: %v", model, err))
		},
	)
	if err != nil {
		return err
	}

	stats.ModelScores += migrated

	logger.Infof("Migrated %d model scores", stats.ModelScores)
	return nil
}

func migrateProviderDefaults(store *storage.SQLiteStorage, sourceDir string, stats *MigrationStats) error {
	path := filepath.Join(sourceDir, "provider_defaults.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("provider_defaults.json not found, skipping")
			return nil
		}
		return err
	}

	var defaults map[string]string
	if err := json.Unmarshal(data, &defaults); err != nil {
		return fmt.Errorf("parse provider_defaults.json: %w", err)
	}

	for provider, model := range defaults {
		if err := store.SetProviderDefault(provider, model); err != nil {
			stats.AddError(fmt.Sprintf("provider_default %s: %v", provider, err))
			continue
		}
		stats.ProviderDefaults++
	}

	logger.Infof("Migrated %d provider defaults", stats.ProviderDefaults)
	return nil
}

func migrateRouterConfig(store *storage.SQLiteStorage, sourceDir string, stats *MigrationStats) error {
	path := filepath.Join(sourceDir, "router_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("router_config.json not found, skipping")
			return nil
		}
		return err
	}

	var config models.RouterConfigRecord
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse router_config.json: %w", err)
	}

	if err := store.SetRouterConfig(&config); err != nil {
		return err
	}

	stats.RouterConfig = true
	logger.Info("Migrated router config")
	return nil
}

func migrateAPIKeys(store *storage.SQLiteStorage, sourceDir string, stats *MigrationStats) error {
	path := filepath.Join(sourceDir, "api_keys.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("api_keys.json not found, skipping")
			return nil
		}
		return err
	}

	var keys map[string]*models.APIKeyRecord
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("parse api_keys.json: %w", err)
	}

	for id, key := range keys {
		key.ID = id
		if key.CreatedAt == "" {
			key.CreatedAt = time.Now().Format(time.RFC3339)
		}
		if err := store.SaveAPIKey(id, key); err != nil {
			stats.AddError(fmt.Sprintf("api_key %s: %v", id, err))
			continue
		}
		stats.APIKeys++
	}

	logger.Infof("Migrated %d API keys", stats.APIKeys)
	return nil
}

//nolint:dupl // model/user migrations share the same flow with type-specific persistence.
func migrateUsers(store *storage.SQLiteStorage, sourceDir string, stats *MigrationStats) error {
	migrated, err := migrateLabeledRecords(
		sourceDir,
		"users.json",
		func(username string, user *models.UserRecord) error {
			user.Username = username
			ensureTimestamps(&user.CreatedAt, &user.UpdatedAt)
			return store.SaveUser(username, user)
		},
		func(username string, err error) {
			stats.AddError(fmt.Sprintf("user %s: %v", username, err))
		},
	)
	if err != nil {
		return err
	}

	stats.Users += migrated

	logger.Infof("Migrated %d users", stats.Users)
	return nil
}

func migrateDeletedModels(store *storage.SQLiteStorage, sourceDir string, stats *MigrationStats) error {
	path := filepath.Join(sourceDir, "store.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("store.json not found, trying individual files")
			return nil
		}
		return err
	}

	var storeData struct {
		DeletedModels map[string]bool `json:"deleted_models"`
	}
	if err := json.Unmarshal(data, &storeData); err != nil {
		return fmt.Errorf("parse store.json: %w", err)
	}

	for model := range storeData.DeletedModels {
		if err := store.MarkModelDeleted(model); err != nil {
			stats.AddError(fmt.Sprintf("deleted_model %s: %v", model, err))
			continue
		}
		stats.DeletedModels++
	}

	logger.Infof("Migrated %d deleted models", stats.DeletedModels)
	return nil
}
