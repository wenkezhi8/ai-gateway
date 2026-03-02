package vectordb

import (
	"context"
	"testing"
)

func TestSQLiteRepository_WhenDBNil_ShouldReturnErrorsAcrossMethods(t *testing.T) {
	t.Parallel()

	repo := &SQLiteRepository{}
	ctx := context.Background()

	if err := repo.ensureSchema(ctx); err == nil {
		t.Fatal("ensureSchema() should fail when db is nil")
	}
	if _, err := NewSQLiteRepository(nil); err != nil {
		t.Fatalf("NewSQLiteRepository(nil) error = %v", err)
	}

	if err := repo.Create(ctx, &Collection{}); err == nil {
		t.Fatal("Create() should fail when db is nil")
	}
	if _, err := repo.Get(ctx, "docs"); err == nil {
		t.Fatal("Get() should fail when db is nil")
	}
	if _, err := repo.List(ctx, &ListCollectionsQuery{}); err == nil {
		t.Fatal("List() should fail when db is nil")
	}
	if err := repo.Update(ctx, "docs", &UpdateCollectionRequest{}); err == nil {
		t.Fatal("Update() should fail when db is nil")
	}
	if err := repo.Delete(ctx, "docs"); err == nil {
		t.Fatal("Delete() should fail when db is nil")
	}
	if err := repo.UpdateCollectionStats(ctx, "docs", 1, 1, 1); err == nil {
		t.Fatal("UpdateCollectionStats() should fail when db is nil")
	}

	if err := repo.CreateImportJob(ctx, &ImportJob{}); err == nil {
		t.Fatal("CreateImportJob() should fail when db is nil")
	}
	if _, err := repo.GetImportJob(ctx, "job"); err == nil {
		t.Fatal("GetImportJob() should fail when db is nil")
	}
	if _, err := repo.ListImportJobs(ctx, &ListImportJobsQuery{}); err == nil {
		t.Fatal("ListImportJobs() should fail when db is nil")
	}
	if _, err := repo.SummarizeImportJobs(ctx, &ListImportJobsQuery{}); err == nil {
		t.Fatal("SummarizeImportJobs() should fail when db is nil")
	}
	if err := repo.UpdateImportJobStatus(ctx, "job", &UpdateImportJobStatusRequest{Status: ImportJobStatusRunning}); err == nil {
		t.Fatal("UpdateImportJobStatus() should fail when db is nil")
	}

	if err := repo.CreateAuditLog(ctx, &AuditLog{}); err == nil {
		t.Fatal("CreateAuditLog() should fail when db is nil")
	}
	if _, err := repo.ListAuditLogs(ctx, &ListAuditLogsQuery{}); err == nil {
		t.Fatal("ListAuditLogs() should fail when db is nil")
	}

	if err := repo.CreateAlertRule(ctx, &AlertRule{}); err == nil {
		t.Fatal("CreateAlertRule() should fail when db is nil")
	}
	if _, err := repo.ListAlertRules(ctx); err == nil {
		t.Fatal("ListAlertRules() should fail when db is nil")
	}
	if err := repo.UpdateAlertRule(ctx, 1, &UpdateAlertRuleRequest{}); err == nil {
		t.Fatal("UpdateAlertRule() should fail when db is nil")
	}
	if err := repo.DeleteAlertRule(ctx, 1); err == nil {
		t.Fatal("DeleteAlertRule() should fail when db is nil")
	}

	if err := repo.CreateVectorAPIKey(ctx, &VectorAPIKey{}); err == nil {
		t.Fatal("CreateVectorAPIKey() should fail when db is nil")
	}
	if _, err := repo.GetVectorAPIKeyByHash(ctx, "hash"); err == nil {
		t.Fatal("GetVectorAPIKeyByHash() should fail when db is nil")
	}
	if _, err := repo.ListVectorAPIKeys(ctx); err == nil {
		t.Fatal("ListVectorAPIKeys() should fail when db is nil")
	}
	if err := repo.DeleteVectorAPIKey(ctx, 1); err == nil {
		t.Fatal("DeleteVectorAPIKey() should fail when db is nil")
	}

	if err := repo.CreateBackupTask(ctx, &BackupTask{}); err == nil {
		t.Fatal("CreateBackupTask() should fail when db is nil")
	}
	if _, err := repo.GetBackupTask(ctx, 1); err == nil {
		t.Fatal("GetBackupTask() should fail when db is nil")
	}
	if _, err := repo.ListBackupTasks(ctx, &ListBackupsQuery{}); err == nil {
		t.Fatal("ListBackupTasks() should fail when db is nil")
	}
	if err := repo.UpdateBackupTask(ctx, 1, &UpdateBackupTaskRequest{Status: BackupStatusFailed}); err == nil {
		t.Fatal("UpdateBackupTask() should fail when db is nil")
	}
	if _, err := repo.DeleteOldBackupTasks(ctx, "docs", 1); err == nil {
		t.Fatal("DeleteOldBackupTasks() should fail when db is nil")
	}
}
