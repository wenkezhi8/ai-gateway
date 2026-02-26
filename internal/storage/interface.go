package storage

import (
	"ai-gateway/internal/models"
)

type Storage interface {
	SaveAccount(account *models.AccountRecord) error
	GetAccount(id string) (*models.AccountRecord, error)
	GetAllAccounts() (map[string]*models.AccountRecord, error)
	DeleteAccount(id string) error

	SaveModelScore(model string, score *models.ModelScoreRecord) error
	GetModelScore(model string) (*models.ModelScoreRecord, error)
	GetAllModelScores() (map[string]*models.ModelScoreRecord, error)
	GetEnabledModelScores() (map[string]*models.ModelScoreRecord, error)
	DeleteModelScore(model string) error

	SaveUser(username string, user *models.UserRecord) error
	GetUser(username string) (*models.UserRecord, error)
	GetAllUsers() (map[string]*models.UserRecord, error)
	DeleteUser(username string) error

	SaveAPIKey(id string, key *models.APIKeyRecord) error
	GetAPIKey(id string) (*models.APIKeyRecord, error)
	GetAllAPIKeys() (map[string]*models.APIKeyRecord, error)
	DeleteAPIKey(id string) error

	GetProviderDefault(provider string) (string, error)
	SetProviderDefault(provider, model string) error
	GetAllProviderDefaults() (map[string]string, error)

	GetRouterConfig() (*models.RouterConfigRecord, error)
	SetRouterConfig(config *models.RouterConfigRecord) error

	MarkModelDeleted(model string) error
	IsModelDeleted(model string) (bool, error)
	RestoreModel(model string) error
	GetAllDeletedModels() ([]string, error)

	SaveFeedback(feedback *models.FeedbackRecord) error
	GetFeedback(limit, offset int) ([]*models.FeedbackRecord, error)
	GetFeedbackStats() (map[string]interface{}, error)

	GetStats() map[string]interface{}
	Close() error
}
