package vectordb

import (
	"context"
	"errors"
	"testing"
	"time"
)

//nolint:govet // Test flow intentionally reuses short err names in scoped assertions.
func TestSQLiteRepository_ImportAuditAlertAndAPIKeyFlows_ShouldWork(t *testing.T) {
	t.Parallel()

	db := setupTestSQLite(t)
	repo, err := NewSQLiteRepository(db)
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	col := &Collection{
		ID:             "col_1",
		Name:           "docs",
		Dimension:      4,
		DistanceMetric: "cosine",
		IndexType:      "hnsw",
		StorageBackend: "qdrant",
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      "tester",
	}
	if err := repo.Create(ctx, col); err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}
	if err := repo.UpdateCollectionStats(ctx, "docs", 10, 9, 1024); err != nil {
		t.Fatalf("repo.UpdateCollectionStats() error = %v", err)
	}

	job := &ImportJob{
		ID:             "job_1",
		CollectionID:   "col_1",
		CollectionName: "docs",
		FileName:       "docs.json",
		FilePath:       "/tmp/docs.json",
		FileSize:       100,
		TotalRecords:   10,
		Status:         ImportJobStatusPending,
		RetryCount:     0,
		MaxRetries:     3,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      "tester",
	}
	if err := repo.CreateImportJob(ctx, job); err != nil {
		t.Fatalf("repo.CreateImportJob() error = %v", err)
	}
	gotJob, err := repo.GetImportJob(ctx, "job_1")
	if err != nil {
		t.Fatalf("repo.GetImportJob() error = %v", err)
	}
	if gotJob.ID != "job_1" {
		t.Fatalf("repo.GetImportJob().ID=%s, want job_1", gotJob.ID)
	}
	jobs, err := repo.ListImportJobs(ctx, &ListImportJobsQuery{CollectionName: "docs", Limit: 10})
	if err != nil {
		t.Fatalf("repo.ListImportJobs() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("repo.ListImportJobs() len=%d, want 1", len(jobs))
	}
	proc := int64(8)
	fail := int64(2)
	retry := int64(1)
	if err := repo.UpdateImportJobStatus(ctx, "job_1", &UpdateImportJobStatusRequest{
		Status:           ImportJobStatusFailed,
		ProcessedRecords: &proc,
		FailedRecords:    &fail,
		RetryCount:       &retry,
	}); err != nil {
		t.Fatalf("repo.UpdateImportJobStatus() error = %v", err)
	}
	summary, err := repo.SummarizeImportJobs(ctx, &ListImportJobsQuery{CollectionName: "docs"})
	if err != nil {
		t.Fatalf("repo.SummarizeImportJobs() error = %v", err)
	}
	if summary.Total != 1 || summary.Failed != 1 {
		t.Fatalf("repo.SummarizeImportJobs() summary=%+v", summary)
	}

	if err := repo.CreateAuditLog(ctx, &AuditLog{UserID: "u1", Action: "import_run_failed", ResourceType: "import_job", ResourceID: "job_1", Details: "d1", CreatedAt: now}); err != nil {
		t.Fatalf("repo.CreateAuditLog() error = %v", err)
	}
	auditLogs, err := repo.ListAuditLogs(ctx, &ListAuditLogsQuery{ResourceType: "import_job", Limit: 10})
	if err != nil {
		t.Fatalf("repo.ListAuditLogs() error = %v", err)
	}
	if len(auditLogs) == 0 {
		t.Fatalf("repo.ListAuditLogs() should return items")
	}

	rule := &AlertRule{
		Name:      "r1",
		Metric:    "search_p95_ms",
		Operator:  "gt",
		Threshold: 500,
		Duration:  "5m",
		Channels:  []string{"webhook"},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateAlertRule(ctx, rule); err != nil {
		t.Fatalf("repo.CreateAlertRule() error = %v", err)
	}
	rules, err := repo.ListAlertRules(ctx)
	if err != nil {
		t.Fatalf("repo.ListAlertRules() error = %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("repo.ListAlertRules() len=%d, want 1", len(rules))
	}
	newName := "r2"
	if err := repo.UpdateAlertRule(ctx, rule.ID, &UpdateAlertRuleRequest{Name: &newName}); err != nil {
		t.Fatalf("repo.UpdateAlertRule() error = %v", err)
	}
	if err := repo.DeleteAlertRule(ctx, rule.ID); err != nil {
		t.Fatalf("repo.DeleteAlertRule() error = %v", err)
	}

	key := &VectorAPIKey{KeyHash: "hash-1", Role: "admin", Enabled: true, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateVectorAPIKey(ctx, key); err != nil {
		t.Fatalf("repo.CreateVectorAPIKey() error = %v", err)
	}
	if _, err := repo.GetVectorAPIKeyByHash(ctx, "hash-1"); err != nil {
		t.Fatalf("repo.GetVectorAPIKeyByHash() error = %v", err)
	}
	keys, err := repo.ListVectorAPIKeys(ctx)
	if err != nil {
		t.Fatalf("repo.ListVectorAPIKeys() error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("repo.ListVectorAPIKeys() len=%d, want 1", len(keys))
	}
	if err := repo.DeleteVectorAPIKey(ctx, keys[0].ID); err != nil {
		t.Fatalf("repo.DeleteVectorAPIKey() error = %v", err)
	}
}

//nolint:govet // Test flow intentionally reuses short err names in scoped assertions.
func TestSQLiteRepository_DeleteOldBackupTasks_ShouldKeepLatest(t *testing.T) {
	t.Parallel()

	db := setupTestSQLite(t)
	repo, err := NewSQLiteRepository(db)
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		task := &BackupTask{
			CollectionName: "docs",
			SnapshotName:   "snap",
			Action:         BackupActionBackup,
			Status:         BackupStatusCompleted,
			CreatedAt:      now.Add(time.Duration(i) * time.Minute),
			UpdatedAt:      now.Add(time.Duration(i) * time.Minute),
			CreatedBy:      "tester",
		}
		if err := repo.CreateBackupTask(ctx, task); err != nil {
			t.Fatalf("repo.CreateBackupTask() error = %v", err)
		}
	}

	deleted, err := repo.DeleteOldBackupTasks(ctx, "docs", 1)
	if err != nil {
		t.Fatalf("repo.DeleteOldBackupTasks() error = %v", err)
	}
	if deleted != 2 {
		t.Fatalf("repo.DeleteOldBackupTasks() deleted=%d, want 2", deleted)
	}
}

//nolint:govet // Test flow intentionally reuses short err names in scoped assertions.
func TestSQLiteRepository_ErrorBranches_ShouldReturnDomainErrors(t *testing.T) {
	t.Parallel()

	repo, err := NewSQLiteRepository(setupTestSQLite(t))
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}
	ctx := context.Background()

	if err := repo.Create(ctx, nil); err == nil {
		t.Fatal("Create(nil) should fail")
	}
	if _, err := repo.Get(ctx, "missing"); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("Get(missing) err=%v, want ErrCollectionNotFound", err)
	}
	if err := repo.Update(ctx, "missing", &UpdateCollectionRequest{}); err != nil {
		t.Fatalf("Update(empty req) should be no-op, err=%v", err)
	}
	desc := "updated"
	if err := repo.Update(ctx, "missing", &UpdateCollectionRequest{Description: &desc}); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("Update(missing) err=%v, want ErrCollectionNotFound", err)
	}
	if err := repo.Delete(ctx, "missing"); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("Delete(missing) err=%v, want ErrCollectionNotFound", err)
	}
	if err := repo.UpdateCollectionStats(ctx, "missing", 1, 1, 1); !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("UpdateCollectionStats(missing) err=%v, want ErrCollectionNotFound", err)
	}

	if err := repo.CreateImportJob(ctx, nil); err == nil {
		t.Fatal("CreateImportJob(nil) should fail")
	}
	if _, err := repo.GetImportJob(ctx, "missing"); !errors.Is(err, ErrImportJobNotFound) {
		t.Fatalf("GetImportJob(missing) err=%v, want ErrImportJobNotFound", err)
	}
	if err := repo.UpdateImportJobStatus(ctx, "missing", &UpdateImportJobStatusRequest{Status: ImportJobStatusRunning}); !errors.Is(err, ErrImportJobNotFound) {
		t.Fatalf("UpdateImportJobStatus(missing) err=%v, want ErrImportJobNotFound", err)
	}

	if err := repo.CreateAuditLog(ctx, nil); err == nil {
		t.Fatal("CreateAuditLog(nil) should fail")
	}
	logs, err := repo.ListAuditLogs(ctx, &ListAuditLogsQuery{Limit: -1, Offset: -3})
	if err != nil {
		t.Fatalf("ListAuditLogs(empty) error = %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("ListAuditLogs(empty) len=%d, want 0", len(logs))
	}

	if err := repo.CreateAlertRule(ctx, nil); err == nil {
		t.Fatal("CreateAlertRule(nil) should fail")
	}
	if err := repo.UpdateAlertRule(ctx, 1, &UpdateAlertRuleRequest{}); err != nil {
		t.Fatalf("UpdateAlertRule(empty req) should be no-op, err=%v", err)
	}
	name := "n"
	if err := repo.UpdateAlertRule(ctx, 1, &UpdateAlertRuleRequest{Name: &name}); !errors.Is(err, ErrAlertRuleNotFound) {
		t.Fatalf("UpdateAlertRule(missing) err=%v, want ErrAlertRuleNotFound", err)
	}
	if err := repo.DeleteAlertRule(ctx, 1); !errors.Is(err, ErrAlertRuleNotFound) {
		t.Fatalf("DeleteAlertRule(missing) err=%v, want ErrAlertRuleNotFound", err)
	}

	if err := repo.CreateVectorAPIKey(ctx, nil); err == nil {
		t.Fatal("CreateVectorAPIKey(nil) should fail")
	}
	if _, err := repo.GetVectorAPIKeyByHash(ctx, "missing"); !errors.Is(err, ErrVectorAPIKeyNotFound) {
		t.Fatalf("GetVectorAPIKeyByHash(missing) err=%v, want ErrVectorAPIKeyNotFound", err)
	}
	if err := repo.DeleteVectorAPIKey(ctx, 1); !errors.Is(err, ErrVectorAPIKeyNotFound) {
		t.Fatalf("DeleteVectorAPIKey(missing) err=%v, want ErrVectorAPIKeyNotFound", err)
	}

	if err := repo.CreateBackupTask(ctx, nil); err == nil {
		t.Fatal("CreateBackupTask(nil) should fail")
	}
	if _, err := repo.GetBackupTask(ctx, 1); !errors.Is(err, ErrBackupTaskNotFound) {
		t.Fatalf("GetBackupTask(missing) err=%v, want ErrBackupTaskNotFound", err)
	}
	if err := repo.UpdateBackupTask(ctx, 1, &UpdateBackupTaskRequest{Status: BackupStatusFailed}); !errors.Is(err, ErrBackupTaskNotFound) {
		t.Fatalf("UpdateBackupTask(missing) err=%v, want ErrBackupTaskNotFound", err)
	}
	if _, err := repo.DeleteOldBackupTasks(ctx, "", 1); err == nil {
		t.Fatal("DeleteOldBackupTasks(empty collection) should fail")
	}
}

//nolint:govet // Test flow intentionally reuses short err names in scoped assertions.
func TestSQLiteRepository_ListQueries_ShouldHandlePaginationFallbackBranches(t *testing.T) {
	t.Parallel()

	repo, err := NewSQLiteRepository(setupTestSQLite(t))
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}
	ctx := context.Background()
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		col := &Collection{
			ID:             "col_p_" + time.Now().Add(time.Duration(i)*time.Nanosecond).Format("150405.000000000"),
			Name:           "docs-p-" + time.Now().Add(time.Duration(i)*time.Nanosecond).Format("150405.000000000"),
			Description:    "desc",
			Dimension:      3,
			DistanceMetric: "cosine",
			IndexType:      "hnsw",
			StorageBackend: "qdrant",
			Tags:           []string{"prod"},
			Environment:    "prod",
			Status:         "active",
			CreatedAt:      now.Add(time.Duration(i) * time.Second),
			UpdatedAt:      now.Add(time.Duration(i) * time.Second),
			CreatedBy:      "tester",
		}
		if err := repo.Create(ctx, col); err != nil {
			t.Fatalf("repo.Create() error = %v", err)
		}
	}

	items, err := repo.List(ctx, &ListCollectionsQuery{Search: "docs-p", Offset: 1})
	if err != nil {
		t.Fatalf("repo.List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("repo.List() len=%d, want 2", len(items))
	}
	items, err = repo.List(ctx, &ListCollectionsQuery{Search: "docs-p", Offset: 20})
	if err != nil {
		t.Fatalf("repo.List(offset overflow) error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("repo.List(offset overflow) len=%d, want 0", len(items))
	}

	job := &ImportJob{
		ID:             "job_pag_1",
		CollectionID:   "col_p_1",
		CollectionName: "docs-p-1",
		FileName:       "a.json",
		FilePath:       "/tmp/a.json",
		FileSize:       10,
		TotalRecords:   1,
		Status:         ImportJobStatusPending,
		MaxRetries:     1,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      "tester",
	}
	if err := repo.CreateImportJob(ctx, job); err != nil {
		t.Fatalf("repo.CreateImportJob() error = %v", err)
	}
	importItems, err := repo.ListImportJobs(ctx, &ListImportJobsQuery{Offset: 3})
	if err != nil {
		t.Fatalf("repo.ListImportJobs() error = %v", err)
	}
	if len(importItems) != 0 {
		t.Fatalf("repo.ListImportJobs(offset overflow) len=%d, want 0", len(importItems))
	}

	backup := &BackupTask{
		CollectionName: "docs",
		SnapshotName:   "snap-a",
		Action:         BackupActionBackup,
		Status:         BackupStatusCompleted,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      "tester",
	}
	if err := repo.CreateBackupTask(ctx, backup); err != nil {
		t.Fatalf("repo.CreateBackupTask() error = %v", err)
	}
	backupItems, err := repo.ListBackupTasks(ctx, &ListBackupsQuery{Offset: 3})
	if err != nil {
		t.Fatalf("repo.ListBackupTasks() error = %v", err)
	}
	if len(backupItems) != 0 {
		t.Fatalf("repo.ListBackupTasks(offset overflow) len=%d, want 0", len(backupItems))
	}
}
