package vectordb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	internalqdrant "ai-gateway/internal/qdrant"
	"ai-gateway/internal/storage"

	"github.com/sirupsen/logrus"
)

var (
	// ErrCollectionExists means collection name already exists.
	ErrCollectionExists = errors.New("collection already exists")
	// ErrCollectionNotFound means collection is missing.
	ErrCollectionNotFound = errors.New("collection not found")
	// ErrBackendUnavailable means vector backend is unavailable.
	ErrBackendUnavailable = errors.New("vector backend unavailable")
	// ErrImportJobNotFound means import job is missing.
	ErrImportJobNotFound = errors.New("import job not found")
	// ErrImportJobRetryExceeded means retry count reached max retries.
	ErrImportJobRetryExceeded = errors.New("import job retry exceeded")
	// ErrAlertRuleNotFound means alert rule is missing.
	ErrAlertRuleNotFound = errors.New("alert rule not found")
	// ErrVectorAPIKeyNotFound means vector api key is missing.
	ErrVectorAPIKeyNotFound = errors.New("vector api key not found")
	// ErrBackupTaskNotFound means backup task is missing.
	ErrBackupTaskNotFound = errors.New("backup task not found")
)

// CollectionRepository persists collection metadata.
type CollectionRepository interface {
	Create(ctx context.Context, col *Collection) error
	Get(ctx context.Context, name string) (*Collection, error)
	List(ctx context.Context, query *ListCollectionsQuery) ([]Collection, error)
	Update(ctx context.Context, name string, req *UpdateCollectionRequest) error
	Delete(ctx context.Context, name string) error
	CreateImportJob(ctx context.Context, job *ImportJob) error
	GetImportJob(ctx context.Context, id string) (*ImportJob, error)
	ListImportJobs(ctx context.Context, query *ListImportJobsQuery) ([]ImportJob, error)
	SummarizeImportJobs(ctx context.Context, query *ListImportJobsQuery) (*ImportJobSummary, error)
	UpdateImportJobStatus(ctx context.Context, id string, req *UpdateImportJobStatusRequest) error
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	ListAuditLogs(ctx context.Context, query *ListAuditLogsQuery) ([]AuditLog, error)
	CreateAlertRule(ctx context.Context, rule *AlertRule) error
	ListAlertRules(ctx context.Context) ([]AlertRule, error)
	UpdateAlertRule(ctx context.Context, id int64, req *UpdateAlertRuleRequest) error
	DeleteAlertRule(ctx context.Context, id int64) error
	CreateVectorAPIKey(ctx context.Context, key *VectorAPIKey) error
	ListVectorAPIKeys(ctx context.Context) ([]VectorAPIKey, error)
	GetVectorAPIKeyByHash(ctx context.Context, keyHash string) (*VectorAPIKey, error)
	DeleteVectorAPIKey(ctx context.Context, id int64) error
	CreateBackupTask(ctx context.Context, task *BackupTask) error
	GetBackupTask(ctx context.Context, id int64) (*BackupTask, error)
	ListBackupTasks(ctx context.Context, query *ListBackupsQuery) ([]BackupTask, error)
	UpdateBackupTask(ctx context.Context, id int64, req *UpdateBackupTaskRequest) error
}

// CollectionBackend manages vector collection lifecycle.
type CollectionBackend interface {
	CreateCollection(ctx context.Context, name string, dimension int, metric string) error
	DeleteCollection(ctx context.Context, name string) error
	GetCollectionInfo(ctx context.Context, name string) (*internalqdrant.CollectionInfo, error)
	UpsertPoints(ctx context.Context, collectionName string, points []internalqdrant.UpsertPoint) error
	Search(ctx context.Context, collectionName string, vector []float32, topK int, minScore float32) ([]SearchResult, error)
	GetByID(ctx context.Context, collectionName, id string) (*SearchResult, error)
}

type qdrantBackend struct {
	client *internalqdrant.Client
}

var _ CollectionBackend = (*qdrantBackend)(nil)

func (b *qdrantBackend) CreateCollection(ctx context.Context, name string, dimension int, metric string) error {
	if b == nil || b.client == nil {
		return ErrBackendUnavailable
	}
	return b.client.CreateCollection(ctx, name, dimension, metric)
}

func (b *qdrantBackend) DeleteCollection(ctx context.Context, name string) error {
	if b == nil || b.client == nil {
		return ErrBackendUnavailable
	}
	return b.client.DeleteCollection(ctx, name)
}

func (b *qdrantBackend) GetCollectionInfo(ctx context.Context, name string) (*internalqdrant.CollectionInfo, error) {
	if b == nil || b.client == nil {
		return nil, ErrBackendUnavailable
	}
	return b.client.GetCollectionInfo(ctx, name)
}

func (b *qdrantBackend) UpsertPoints(ctx context.Context, collectionName string, points []internalqdrant.UpsertPoint) error {
	if b == nil || b.client == nil {
		return ErrBackendUnavailable
	}
	return b.client.UpsertPoints(ctx, collectionName, points)
}

func (b *qdrantBackend) Search(ctx context.Context, collectionName string, vector []float32, topK int, minScore float32) ([]SearchResult, error) {
	if b == nil || b.client == nil {
		return nil, ErrBackendUnavailable
	}
	items, err := b.client.Search(ctx, collectionName, vector, topK, minScore)
	if err != nil {
		return nil, errors.Join(ErrBackendUnavailable, fmt.Errorf("search backend failed: %w", err))
	}
	results := make([]SearchResult, 0, len(items))
	for idx := range items {
		results = append(results, SearchResult{
			ID:      items[idx].ID,
			Score:   items[idx].Score,
			Payload: items[idx].Payload,
		})
	}
	return results, nil
}

func (b *qdrantBackend) GetByID(ctx context.Context, collectionName, id string) (*SearchResult, error) {
	if b == nil || b.client == nil {
		return nil, ErrBackendUnavailable
	}
	item, err := b.client.GetByID(ctx, collectionName, id)
	if err != nil {
		return nil, errors.Join(ErrBackendUnavailable, fmt.Errorf("get point by id from backend failed: %w", err))
	}
	return &SearchResult{ID: item.ID, Score: item.Score, Payload: item.Payload}, nil
}

// Service handles collection CRUD.
type Service struct {
	repo    CollectionRepository
	backend CollectionBackend
}

type ServiceConfig struct {
	DB             *sql.DB
	QdrantHTTPAddr string
	QdrantAPIKey   string
}

// NewService creates a vector db service backed by sqlite and qdrant.
func NewService() *Service {
	return NewServiceWithConfig(ServiceConfig{
		DB:             storage.GetSQLiteStorage().GetDB(),
		QdrantHTTPAddr: strings.TrimSpace(os.Getenv("AI_GATEWAY_QDRANT_URL")),
		QdrantAPIKey:   strings.TrimSpace(os.Getenv("AI_GATEWAY_QDRANT_API_KEY")),
	})
}

func NewServiceWithConfig(cfg ServiceConfig) *Service {
	repo, err := NewSQLiteRepository(cfg.DB)
	if err != nil {
		logrus.WithError(err).Warn("failed to initialize vector sqlite repository")
	}

	httpAddr := strings.TrimSpace(cfg.QdrantHTTPAddr)
	if httpAddr == "" {
		httpAddr = "http://localhost:6334"
	}
	apiKey := strings.TrimSpace(cfg.QdrantAPIKey)

	client, err := internalqdrant.NewQdrantClient(httpAddr, apiKey, "")
	if err != nil {
		logrus.WithError(err).Warn("failed to initialize qdrant client")
		return &Service{repo: repo}
	}

	return NewServiceWithDeps(repo, &qdrantBackend{client: client})
}

// NewServiceWithDeps creates service with explicit dependencies.
func NewServiceWithDeps(repo CollectionRepository, backend CollectionBackend) *Service {
	return &Service{repo: repo, backend: backend}
}

func (s *Service) GetRepository() CollectionRepository {
	if s == nil {
		return nil
	}
	return s.repo
}

// CreateCollection creates a collection with backend and metadata persistence.
func (s *Service) CreateCollection(ctx context.Context, req *CreateCollectionRequest) (*Collection, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if s.backend == nil {
		return nil, ErrBackendUnavailable
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}
	metric := defaultString(req.DistanceMetric, "cosine")

	if err := s.backend.CreateCollection(ctx, name, req.Dimension, metric); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			return nil, ErrCollectionExists
		}
		return nil, errors.Join(ErrBackendUnavailable, fmt.Errorf("create backend collection failed: %w", err))
	}

	now := time.Now().UTC()
	collection := &Collection{
		ID:              fmt.Sprintf("col_%d", now.UnixNano()),
		Name:            name,
		Description:     strings.TrimSpace(req.Description),
		Dimension:       req.Dimension,
		DistanceMetric:  metric,
		IndexType:       defaultString(req.IndexType, "hnsw"),
		HNSWM:           16,
		HNSWEFConstruct: 100,
		IVFNList:        1024,
		StorageBackend:  defaultString(req.StorageBackend, "qdrant"),
		Tags:            copyTags(req.Tags),
		Environment:     defaultString(req.Environment, "default"),
		Status:          defaultString(req.Status, "active"),
		VectorCount:     0,
		IndexedCount:    0,
		SizeBytes:       0,
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedBy:       defaultString(req.CreatedBy, "system"),
		IsPublic:        req.IsPublic,
	}
	if err := s.repo.Create(ctx, collection); err != nil {
		if rollbackErr := s.backend.DeleteCollection(ctx, name); rollbackErr != nil {
			logrus.WithError(rollbackErr).WithField("name", name).Error("rollback qdrant collection failed")
		}
		if errors.Is(err, ErrCollectionExists) {
			return nil, err
		}
		return nil, fmt.Errorf("persist collection metadata failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{"name": name, "env": collection.Environment}).Info("vector db collection created")

	result := *collection
	return &result, nil
}

// GetCollection gets one collection by name.
func (s *Service) GetCollection(ctx context.Context, name string) (*Collection, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	key := strings.TrimSpace(name)
	if key == "" {
		return nil, fmt.Errorf("name is required")
	}
	col, err := s.repo.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	result := *col
	result.Tags = copyTags(col.Tags)
	return &result, nil
}

// ListCollections lists collections with filters.
func (s *Service) ListCollections(ctx context.Context, query *ListCollectionsQuery) ([]Collection, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if query == nil {
		query = &ListCollectionsQuery{}
	}
	items, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].Tags = copyTags(items[idx].Tags)
	}
	return items, nil
}

// UpdateCollection updates mutable collection fields.
func (s *Service) UpdateCollection(ctx context.Context, name string, req *UpdateCollectionRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if s.repo == nil {
		return fmt.Errorf("repository is required")
	}

	key := strings.TrimSpace(name)
	if key == "" {
		return fmt.Errorf("name is required")
	}
	if err := s.repo.Update(ctx, key, req); err != nil {
		return err
	}

	logrus.WithField("name", key).Info("vector db collection updated")
	return nil
}

// DeleteCollection deletes one collection by name.
func (s *Service) DeleteCollection(ctx context.Context, name string) error {
	if s.repo == nil {
		return fmt.Errorf("repository is required")
	}
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	key := strings.TrimSpace(name)
	if key == "" {
		return fmt.Errorf("name is required")
	}

	if _, err := s.repo.Get(ctx, key); err != nil {
		return err
	}
	if err := s.backend.DeleteCollection(ctx, key); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return errors.Join(ErrBackendUnavailable, fmt.Errorf("delete backend collection failed: %w", err))
		}
	}
	if err := s.repo.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete metadata failed: %w", err)
	}
	logrus.WithField("name", key).Info("vector db collection deleted")
	return nil
}

// GetCollectionStats gets simple collection statistics.
func (s *Service) GetCollectionStats(ctx context.Context, name string) (*CollectionStats, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}

	key := strings.TrimSpace(name)
	if key == "" {
		return nil, fmt.Errorf("name is required")
	}

	if s.backend != nil {
		info, err := s.backend.GetCollectionInfo(ctx, key)
		if err == nil && info != nil {
			return &CollectionStats{
				Name:         key,
				VectorCount:  info.VectorCount,
				IndexedCount: info.IndexedCount,
				SizeBytes:    info.SizeBytes,
			}, nil
		}
		if err != nil {
			logrus.WithError(err).WithField("name", key).Warn("fallback to metadata stats")
		}
	}

	col, err := s.repo.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	stats := &CollectionStats{
		Name:         col.Name,
		VectorCount:  col.VectorCount,
		IndexedCount: col.IndexedCount,
		SizeBytes:    col.SizeBytes,
	}
	return stats, nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func copyTags(tags []string) []string {
	if tags == nil {
		return nil
	}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		result = append(result, tag)
	}
	return result
}
