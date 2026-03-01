package vectordb

import (
	"context"
	"testing"
	"time"
)

func TestBackupService_CreateListRestoreRetry_ShouldWork(t *testing.T) {
	t.Parallel()

	repo, err := NewSQLiteRepository(setupTestSQLite(t))
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}

	now := time.Now().UTC()
	if createErr := repo.Create(context.Background(), &Collection{
		ID:              "col_docs",
		Name:            "docs",
		Description:     "docs collection",
		Dimension:       768,
		DistanceMetric:  "cosine",
		IndexType:       "hnsw",
		HNSWM:           16,
		HNSWEFConstruct: 100,
		IVFNList:        1024,
		StorageBackend:  "qdrant",
		Environment:     "prod",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedBy:       "tester",
	}); createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}

	svc := NewServiceWithDeps(repo, &mockBackend{})
	created, err := svc.CreateBackup(context.Background(), &CreateBackupRequest{CollectionName: "docs", SnapshotName: "snapshot-001", CreatedBy: "tester"})
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}
	if created.ID <= 0 || created.Action != BackupActionBackup {
		t.Fatalf("CreateBackup() result = %+v", created)
	}

	items, err := svc.ListBackups(context.Background(), &ListBackupsQuery{})
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("ListBackups() expected at least one task")
	}

	restored, err := svc.TriggerRestore(context.Background(), created.ID, "tester")
	if err != nil {
		t.Fatalf("TriggerRestore() error = %v", err)
	}
	if restored.Action != BackupActionRestore || restored.SourceBackupID != created.ID {
		t.Fatalf("TriggerRestore() result = %+v", restored)
	}

	if _, err := svc.RetryBackupTask(context.Background(), restored.ID); err != nil {
		t.Fatalf("RetryBackupTask() error = %v", err)
	}
}
