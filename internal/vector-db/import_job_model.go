package vectordb

import "time"

type ImportJobStatus string

const (
	ImportJobStatusPending   ImportJobStatus = "pending"
	ImportJobStatusRunning   ImportJobStatus = "running"
	ImportJobStatusCompleted ImportJobStatus = "completed"
	ImportJobStatusFailed    ImportJobStatus = "failed"
	ImportJobStatusCanceled  ImportJobStatus = "cancelled" //nolint:misspell // API keeps cancelled spelling.
	ImportJobStatusRetrying  ImportJobStatus = "retrying"
)

type ImportJob struct {
	ID               string          `json:"id"`
	CollectionID     string          `json:"collection_id"`
	CollectionName   string          `json:"collection_name,omitempty"`
	FileName         string          `json:"file_name"`
	FilePath         string          `json:"file_path"`
	FileSize         int64           `json:"file_size"`
	TotalRecords     int64           `json:"total_records"`
	ProcessedRecords int64           `json:"processed_records"`
	FailedRecords    int64           `json:"failed_records"`
	RetryCount       int             `json:"retry_count"`
	MaxRetries       int             `json:"max_retries"`
	Status           ImportJobStatus `json:"status"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	CreatedBy        string          `json:"created_by"`
}

type CreateImportJobRequest struct {
	CollectionName string `json:"collection_name" binding:"required"`
	FileName       string `json:"file_name" binding:"required"`
	FilePath       string `json:"file_path" binding:"required"`
	FileSize       int64  `json:"file_size" binding:"required"`
	TotalRecords   int64  `json:"total_records" binding:"required"`
	MaxRetries     int    `json:"max_retries"`
	CreatedBy      string `json:"created_by"`
}

type UpdateImportJobStatusRequest struct {
	Status           ImportJobStatus `json:"status" binding:"required"`
	ProcessedRecords *int64          `json:"processed_records"`
	FailedRecords    *int64          `json:"failed_records"`
	RetryCount       *int64          `json:"retry_count"`
	ErrorMessage     *string         `json:"error_message"`
	StartedAt        *time.Time      `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at"`
}

type ListImportJobsQuery struct {
	CollectionName string
	Status         string
	Offset         int
	Limit          int
}

type ImportJobSummary struct {
	Pending   int `json:"pending"`
	Running   int `json:"running"`
	Retrying  int `json:"retrying"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Canceled  int `json:"cancelled"` //nolint:misspell // API keeps cancelled spelling for compatibility.
	Total     int `json:"total"`
}

type AuditLog struct {
	ID           int64     `json:"id"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Details      string    `json:"details"`
	IPAddress    string    `json:"ip_address,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type ListAuditLogsQuery struct {
	ResourceType string
	ResourceID   string
	Action       string
	Limit        int
	Offset       int
}
