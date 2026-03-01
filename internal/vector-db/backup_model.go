package vectordb

import "time"

type BackupAction string

const (
	BackupActionBackup  BackupAction = "backup"
	BackupActionRestore BackupAction = "restore"
)

type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
)

type BackupTask struct {
	ID             int64        `json:"id"`
	CollectionName string       `json:"collection_name"`
	SnapshotName   string       `json:"snapshot_name"`
	Action         BackupAction `json:"action"`
	Status         BackupStatus `json:"status"`
	SourceBackupID int64        `json:"source_backup_id"`
	ErrorMessage   string       `json:"error_message"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	StartedAt      *time.Time   `json:"started_at,omitempty"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	CreatedBy      string       `json:"created_by"`
}

type CreateBackupRequest struct {
	CollectionName string `json:"collection_name"`
	SnapshotName   string `json:"snapshot_name"`
	CreatedBy      string `json:"created_by"`
}

type ListBackupsQuery struct {
	CollectionName string
	Action         string
	Status         string
	Offset         int
	Limit          int
}

type UpdateBackupTaskRequest struct {
	Status       BackupStatus
	ErrorMessage *string
	StartedAt    *time.Time
	CompletedAt  *time.Time
}
