package vectordb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	internalqdrant "ai-gateway/internal/qdrant"

	_ "github.com/mattn/go-sqlite3"
)

func TestVectorDBService_CreateCollection_ShouldCallBackendAndRepo(t *testing.T) {
	t.Parallel()

	backend := &mockBackend{}
	repo := &mockRepo{}
	svc := NewServiceWithDeps(repo, backend)

	_, err := svc.CreateCollection(context.Background(), &CreateCollectionRequest{Name: "docs", Dimension: 768})
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	if backend.createCalls != 1 || repo.createCalls != 1 {
		t.Fatalf("unexpected calls: backend=%d repo=%d", backend.createCalls, repo.createCalls)
	}
}

func TestVectorDBService_CreateCollection_WhenRepoCreateFails_ShouldRollbackBackend(t *testing.T) {
	t.Parallel()

	backend := &mockBackend{}
	repo := &mockRepo{createErr: errors.New("db down")}
	svc := NewServiceWithDeps(repo, backend)

	_, err := svc.CreateCollection(context.Background(), &CreateCollectionRequest{Name: "docs", Dimension: 768})
	if err == nil {
		t.Fatal("CreateCollection() should fail")
	}
	if backend.deleteCalls != 1 {
		t.Fatalf("rollback not called, deleteCalls=%d", backend.deleteCalls)
	}
}

func TestVectorDBService_CreateCollection_WhenBackendAlreadyExists_ShouldReturnConflict(t *testing.T) {
	t.Parallel()

	backend := &mockBackend{createErr: errors.New("already exists")}
	repo := &mockRepo{}
	svc := NewServiceWithDeps(repo, backend)

	_, err := svc.CreateCollection(context.Background(), &CreateCollectionRequest{Name: "docs", Dimension: 768})
	if !errors.Is(err, ErrCollectionExists) {
		t.Fatalf("CreateCollection() err=%v, want ErrCollectionExists", err)
	}
}

func TestVectorDBService_DeleteCollection_WhenBackendNotFound_ShouldStillDeleteMetadata(t *testing.T) {
	t.Parallel()

	backend := &mockBackend{deleteErr: errors.New("not found")}
	repo := &mockRepo{}
	svc := NewServiceWithDeps(repo, backend)

	if err := svc.DeleteCollection(context.Background(), "docs"); err != nil {
		t.Fatalf("DeleteCollection() error = %v", err)
	}
}

func TestVectorDBService_GetCollectionStats_WhenBackendFails_ShouldFallbackToMetadata(t *testing.T) {
	t.Parallel()

	backend := &mockBackend{infoErr: errors.New("backend down")}
	repo := &mockRepo{getResp: &Collection{Name: "docs", VectorCount: 12, IndexedCount: 11, SizeBytes: 1024}}
	svc := NewServiceWithDeps(repo, backend)

	stats, err := svc.GetCollectionStats(context.Background(), "docs")
	if err != nil {
		t.Fatalf("GetCollectionStats() error = %v", err)
	}
	if stats.VectorCount != 12 || stats.IndexedCount != 11 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestSQLiteRepository_CreateAndGet_ShouldPersistCollection(t *testing.T) {
	t.Parallel()

	db := setupTestSQLite(t)
	repo, err := NewSQLiteRepository(db)
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}

	now := time.Now().UTC()
	meta := &Collection{
		ID:             "col_1",
		Name:           "docs",
		Description:    "documents",
		Dimension:      768,
		DistanceMetric: "cosine",
		IndexType:      "hnsw",
		StorageBackend: "qdrant",
		Tags:           []string{"prod", "shared"},
		Environment:    "prod",
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      "tester",
	}

	if createErr := repo.Create(context.Background(), meta); createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}

	got, err := repo.Get(context.Background(), "docs")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name != "docs" || got.Dimension != 768 {
		t.Fatalf("Get() got=%+v", got)
	}
}

func TestSQLiteRepository_ListUpdateDelete_ShouldSupportFiltersAndLifecycle(t *testing.T) {
	t.Parallel()

	db := setupTestSQLite(t)
	repo, err := NewSQLiteRepository(db)
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}

	now := time.Now().UTC()
	for _, item := range []*Collection{
		{
			ID:             "col_1",
			Name:           "docs-a",
			Description:    "A docs",
			Dimension:      768,
			DistanceMetric: "cosine",
			IndexType:      "hnsw",
			StorageBackend: "qdrant",
			Tags:           []string{"prod"},
			Environment:    "prod",
			Status:         "active",
			CreatedAt:      now,
			UpdatedAt:      now,
			CreatedBy:      "tester",
		},
		{
			ID:             "col_2",
			Name:           "docs-b",
			Description:    "B docs",
			Dimension:      768,
			DistanceMetric: "cosine",
			IndexType:      "hnsw",
			StorageBackend: "qdrant",
			Tags:           []string{"test"},
			Environment:    "test",
			Status:         "inactive",
			CreatedAt:      now,
			UpdatedAt:      now,
			CreatedBy:      "tester",
		},
	} {
		if createErr := repo.Create(context.Background(), item); createErr != nil {
			t.Fatalf("Create(%s) error = %v", item.Name, createErr)
		}
	}

	list, err := repo.List(context.Background(), &ListCollectionsQuery{Environment: "prod", Tag: "prod"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 || list[0].Name != "docs-a" {
		t.Fatalf("List() got=%+v", list)
	}

	updatedDesc := "updated docs"
	updatedStatus := "inactive"
	if updateErr := repo.Update(context.Background(), "docs-a", &UpdateCollectionRequest{Description: &updatedDesc, Status: &updatedStatus}); updateErr != nil {
		t.Fatalf("Update() error = %v", updateErr)
	}
	empty := "   "
	if updateErr := repo.Update(context.Background(), "docs-a", &UpdateCollectionRequest{Status: &empty}); updateErr != nil {
		t.Fatalf("Update() with empty status error = %v", updateErr)
	}

	updated, err := repo.Get(context.Background(), "docs-a")
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if updated.Description != updatedDesc || updated.Status != updatedStatus {
		t.Fatalf("Update() result=%+v", updated)
	}

	if err := repo.Delete(context.Background(), "docs-a"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.Get(context.Background(), "docs-a"); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("Get() after delete err=%v, want ErrCollectionNotFound", err)
	}
}

func setupTestSQLite(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("db.Close() error = %v", closeErr)
		}
	})
	return db
}

type mockRepo struct {
	createCalls     int
	createErr       error
	getResp         *Collection
	getErr          error
	importJobs      map[string]*ImportJob
	getImportJobErr error
	auditLogs       []AuditLog
	alertRules      map[int64]*AlertRule
	vectorAPIKeys   map[string]*VectorAPIKey
	backupTasks     map[int64]*BackupTask
}

func (m *mockRepo) Create(_ context.Context, _ *Collection) error {
	m.createCalls++
	return m.createErr
}

func (m *mockRepo) Get(_ context.Context, _ string) (*Collection, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.getResp != nil {
		copyValue := *m.getResp
		return &copyValue, nil
	}
	return &Collection{Name: "docs"}, nil
}

func (m *mockRepo) List(_ context.Context, _ *ListCollectionsQuery) ([]Collection, error) {
	return []Collection{}, nil
}

func (m *mockRepo) Update(_ context.Context, _ string, _ *UpdateCollectionRequest) error {
	return nil
}

func (m *mockRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockRepo) CreateImportJob(_ context.Context, job *ImportJob) error {
	if m.importJobs == nil {
		m.importJobs = make(map[string]*ImportJob)
	}
	copyValue := *job
	m.importJobs[job.ID] = &copyValue
	return nil
}

func (m *mockRepo) GetImportJob(_ context.Context, id string) (*ImportJob, error) {
	if m.getImportJobErr != nil {
		return nil, m.getImportJobErr
	}
	if m.importJobs == nil {
		return nil, ErrImportJobNotFound
	}
	job, ok := m.importJobs[id]
	if !ok {
		return nil, ErrImportJobNotFound
	}
	copyValue := *job
	return &copyValue, nil
}

func (m *mockRepo) ListImportJobs(_ context.Context, _ *ListImportJobsQuery) ([]ImportJob, error) {
	if m.importJobs == nil {
		return []ImportJob{}, nil
	}
	items := make([]ImportJob, 0, len(m.importJobs))
	for _, job := range m.importJobs {
		items = append(items, *job)
	}
	return items, nil
}

func (m *mockRepo) SummarizeImportJobs(_ context.Context, _ *ListImportJobsQuery) (*ImportJobSummary, error) {
	if m.importJobs == nil {
		return &ImportJobSummary{}, nil
	}
	summary := &ImportJobSummary{}
	for _, item := range m.importJobs {
		summary.Total++
		switch item.Status {
		case ImportJobStatusPending:
			summary.Pending++
		case ImportJobStatusRunning:
			summary.Running++
		case ImportJobStatusRetrying:
			summary.Retrying++
		case ImportJobStatusCompleted:
			summary.Completed++
		case ImportJobStatusFailed:
			summary.Failed++
		case ImportJobStatusCanceled:
			summary.Canceled++
		}
	}
	return summary, nil
}

func (m *mockRepo) UpdateImportJobStatus(_ context.Context, id string, req *UpdateImportJobStatusRequest) error {
	if m.importJobs == nil {
		return ErrImportJobNotFound
	}
	job, ok := m.importJobs[id]
	if !ok {
		return ErrImportJobNotFound
	}
	job.Status = req.Status
	if req.ProcessedRecords != nil {
		job.ProcessedRecords = *req.ProcessedRecords
	}
	if req.FailedRecords != nil {
		job.FailedRecords = *req.FailedRecords
	}
	if req.RetryCount != nil {
		job.RetryCount = int(*req.RetryCount)
	}
	if req.ErrorMessage != nil {
		job.ErrorMessage = *req.ErrorMessage
	}
	if req.StartedAt != nil {
		v := req.StartedAt.UTC()
		job.StartedAt = &v
	}
	if req.CompletedAt != nil {
		v := req.CompletedAt.UTC()
		job.CompletedAt = &v
	}
	job.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockRepo) CreateAuditLog(_ context.Context, log *AuditLog) error {
	if log == nil {
		return nil
	}
	copyValue := *log
	if copyValue.ID == 0 {
		copyValue.ID = int64(len(m.auditLogs) + 1)
	}
	m.auditLogs = append(m.auditLogs, copyValue)
	return nil
}

func (m *mockRepo) ListAuditLogs(_ context.Context, query *ListAuditLogsQuery) ([]AuditLog, error) {
	if query == nil {
		query = &ListAuditLogsQuery{}
	}
	items := make([]AuditLog, 0)
	for idx := range m.auditLogs {
		if query.ResourceType != "" && m.auditLogs[idx].ResourceType != query.ResourceType {
			continue
		}
		if query.ResourceID != "" && m.auditLogs[idx].ResourceID != query.ResourceID {
			continue
		}
		if query.Action != "" && m.auditLogs[idx].Action != query.Action {
			continue
		}
		items = append(items, m.auditLogs[idx])
	}
	limit := query.Limit
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []AuditLog{}, nil
	}
	items = items[offset:]
	if limit <= 0 || limit >= len(items) {
		return items, nil
	}
	return items[:limit], nil
}

func (m *mockRepo) CreateAlertRule(_ context.Context, rule *AlertRule) error {
	if rule == nil {
		return nil
	}
	if m.alertRules == nil {
		m.alertRules = make(map[int64]*AlertRule)
	}
	copyValue := *rule
	if copyValue.ID <= 0 {
		copyValue.ID = int64(len(m.alertRules) + 1)
	}
	m.alertRules[copyValue.ID] = &copyValue
	rule.ID = copyValue.ID
	return nil
}

func (m *mockRepo) ListAlertRules(_ context.Context) ([]AlertRule, error) {
	if m.alertRules == nil {
		return []AlertRule{}, nil
	}
	rules := make([]AlertRule, 0, len(m.alertRules))
	for _, rule := range m.alertRules {
		rules = append(rules, *rule)
	}
	return rules, nil
}

func (m *mockRepo) UpdateAlertRule(_ context.Context, id int64, req *UpdateAlertRuleRequest) error {
	if m.alertRules == nil {
		return ErrAlertRuleNotFound
	}
	rule, ok := m.alertRules[id]
	if !ok {
		return ErrAlertRuleNotFound
	}
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Metric != nil {
		rule.Metric = *req.Metric
	}
	if req.Operator != nil {
		rule.Operator = *req.Operator
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.Duration != nil {
		rule.Duration = *req.Duration
	}
	if req.Channels != nil {
		rule.Channels = copyTags(*req.Channels)
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	rule.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockRepo) DeleteAlertRule(_ context.Context, id int64) error {
	if m.alertRules == nil {
		return ErrAlertRuleNotFound
	}
	if _, ok := m.alertRules[id]; !ok {
		return ErrAlertRuleNotFound
	}
	delete(m.alertRules, id)
	return nil
}

func (m *mockRepo) CreateVectorAPIKey(_ context.Context, key *VectorAPIKey) error {
	if key == nil {
		return nil
	}
	if m.vectorAPIKeys == nil {
		m.vectorAPIKeys = make(map[string]*VectorAPIKey)
	}
	copyValue := *key
	if copyValue.ID <= 0 {
		copyValue.ID = int64(len(m.vectorAPIKeys) + 1)
	}
	m.vectorAPIKeys[copyValue.KeyHash] = &copyValue
	return nil
}

func (m *mockRepo) GetVectorAPIKeyByHash(_ context.Context, keyHash string) (*VectorAPIKey, error) {
	if m.vectorAPIKeys == nil {
		return nil, ErrVectorAPIKeyNotFound
	}
	item, ok := m.vectorAPIKeys[keyHash]
	if !ok {
		return nil, ErrVectorAPIKeyNotFound
	}
	copyValue := *item
	return &copyValue, nil
}

func (m *mockRepo) ListVectorAPIKeys(_ context.Context) ([]VectorAPIKey, error) {
	if m.vectorAPIKeys == nil {
		return []VectorAPIKey{}, nil
	}
	items := make([]VectorAPIKey, 0, len(m.vectorAPIKeys))
	for _, key := range m.vectorAPIKeys {
		items = append(items, *key)
	}
	return items, nil
}

func (m *mockRepo) DeleteVectorAPIKey(_ context.Context, id int64) error {
	if m.vectorAPIKeys == nil {
		return ErrVectorAPIKeyNotFound
	}
	for hash, key := range m.vectorAPIKeys {
		if key.ID == id {
			delete(m.vectorAPIKeys, hash)
			return nil
		}
	}
	return ErrVectorAPIKeyNotFound
}

func (m *mockRepo) CreateBackupTask(_ context.Context, task *BackupTask) error {
	if task == nil {
		return nil
	}
	if m.backupTasks == nil {
		m.backupTasks = make(map[int64]*BackupTask)
	}
	copyValue := *task
	if copyValue.ID <= 0 {
		copyValue.ID = int64(len(m.backupTasks) + 1)
	}
	m.backupTasks[copyValue.ID] = &copyValue
	task.ID = copyValue.ID
	return nil
}

func (m *mockRepo) GetBackupTask(_ context.Context, id int64) (*BackupTask, error) {
	if m.backupTasks == nil {
		return nil, ErrBackupTaskNotFound
	}
	item, ok := m.backupTasks[id]
	if !ok {
		return nil, ErrBackupTaskNotFound
	}
	copyValue := *item
	return &copyValue, nil
}

func (m *mockRepo) ListBackupTasks(_ context.Context, _ *ListBackupsQuery) ([]BackupTask, error) {
	if m.backupTasks == nil {
		return []BackupTask{}, nil
	}
	items := make([]BackupTask, 0, len(m.backupTasks))
	for _, task := range m.backupTasks {
		items = append(items, *task)
	}
	return items, nil
}

func (m *mockRepo) UpdateBackupTask(_ context.Context, id int64, req *UpdateBackupTaskRequest) error {
	if m.backupTasks == nil {
		return ErrBackupTaskNotFound
	}
	item, ok := m.backupTasks[id]
	if !ok {
		return ErrBackupTaskNotFound
	}
	item.Status = req.Status
	if req.ErrorMessage != nil {
		item.ErrorMessage = *req.ErrorMessage
	}
	if req.StartedAt != nil {
		v := req.StartedAt.UTC()
		item.StartedAt = &v
	}
	if req.CompletedAt != nil {
		v := req.CompletedAt.UTC()
		item.CompletedAt = &v
	}
	item.UpdatedAt = time.Now().UTC()
	return nil
}

type mockBackend struct {
	createCalls int
	deleteCalls int
	upsertCalls int
	searchCalls int
	createErr   error
	deleteErr   error
	upsertErr   error
	searchErr   error
	infoErr     error
	getByIDResp *SearchResult
	getByIDErr  error
}

func (m *mockBackend) CreateCollection(_ context.Context, _ string, _ int, _ string) error {
	m.createCalls++
	return m.createErr
}

func (m *mockBackend) DeleteCollection(_ context.Context, _ string) error {
	m.deleteCalls++
	return m.deleteErr
}

func (m *mockBackend) GetCollectionInfo(_ context.Context, _ string) (*internalqdrant.CollectionInfo, error) {
	if m.infoErr != nil {
		return nil, m.infoErr
	}
	return &internalqdrant.CollectionInfo{VectorCount: 99, IndexedCount: 88, SizeBytes: 77}, nil
}

func (m *mockBackend) UpsertPoints(_ context.Context, _ string, _ []internalqdrant.UpsertPoint) error {
	m.upsertCalls++
	return m.upsertErr
}

func (m *mockBackend) Search(_ context.Context, _ string, _ []float32, _ int, _ float32) ([]SearchResult, error) {
	m.searchCalls++
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return []SearchResult{}, nil
}

func (m *mockBackend) GetByID(_ context.Context, _, _ string) (*SearchResult, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if m.getByIDResp == nil {
		return nil, ErrCollectionNotFound
	}
	item := *m.getByIDResp
	return &item, nil
}
