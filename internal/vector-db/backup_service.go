package vectordb

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) CreateBackup(ctx context.Context, req *CreateBackupRequest) (*BackupTask, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	collectionName := strings.TrimSpace(req.CollectionName)
	if collectionName == "" {
		return nil, fmt.Errorf("collection_name is required")
	}
	if _, err := s.repo.Get(ctx, collectionName); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	task := &BackupTask{
		CollectionName: collectionName,
		SnapshotName:   defaultString(req.SnapshotName, fmt.Sprintf("snapshot-%d", now.Unix())),
		Action:         BackupActionBackup,
		Status:         BackupStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      defaultString(req.CreatedBy, "system"),
	}
	if err := s.repo.CreateBackupTask(ctx, task); err != nil {
		return nil, err
	}
	if err := s.markBackupTaskCompleted(ctx, task.ID); err != nil {
		return nil, err
	}
	return s.repo.GetBackupTask(ctx, task.ID)
}

func (s *Service) ListBackups(ctx context.Context, query *ListBackupsQuery) ([]BackupTask, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	return s.repo.ListBackupTasks(ctx, query)
}

func (s *Service) TriggerRestore(ctx context.Context, sourceBackupID int64, createdBy string) (*BackupTask, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if sourceBackupID <= 0 {
		return nil, fmt.Errorf("source backup id must be positive")
	}
	source, err := s.repo.GetBackupTask(ctx, sourceBackupID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	task := &BackupTask{
		CollectionName: source.CollectionName,
		SnapshotName:   source.SnapshotName,
		Action:         BackupActionRestore,
		Status:         BackupStatusPending,
		SourceBackupID: source.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      defaultString(createdBy, "system"),
	}
	if err := s.repo.CreateBackupTask(ctx, task); err != nil {
		return nil, err
	}
	if err := s.markBackupTaskCompleted(ctx, task.ID); err != nil {
		return nil, err
	}
	return s.repo.GetBackupTask(ctx, task.ID)
}

func (s *Service) RetryBackupTask(ctx context.Context, id int64) (*BackupTask, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if id <= 0 {
		return nil, fmt.Errorf("id must be positive")
	}
	if _, err := s.repo.GetBackupTask(ctx, id); err != nil {
		return nil, err
	}
	if err := s.markBackupTaskCompleted(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.GetBackupTask(ctx, id)
}

func (s *Service) markBackupTaskCompleted(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	if err := s.repo.UpdateBackupTask(ctx, id, &UpdateBackupTaskRequest{Status: BackupStatusRunning, StartedAt: &now}); err != nil {
		return err
	}
	return s.repo.UpdateBackupTask(ctx, id, &UpdateBackupTaskRequest{Status: BackupStatusCompleted, CompletedAt: &now})
}

func (s *Service) RunBackupPolicy(ctx context.Context, req *RunBackupPolicyRequest) (*BackupPolicyResult, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	collectionName := strings.TrimSpace(req.CollectionName)
	if collectionName == "" {
		return nil, fmt.Errorf("collection_name is required")
	}
	retention := req.RetentionCount
	if retention <= 0 {
		retention = 30
	}

	created, err := s.CreateBackup(ctx, &CreateBackupRequest{
		CollectionName: collectionName,
		CreatedBy:      defaultString(req.CreatedBy, "system"),
	})
	if err != nil {
		return nil, err
	}

	deletedCount, err := s.repo.DeleteOldBackupTasks(ctx, collectionName, retention)
	if err != nil {
		return nil, err
	}

	return &BackupPolicyResult{CreatedTask: created, DeletedCount: deletedCount}, nil
}
