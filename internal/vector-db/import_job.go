package vectordb

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	internalqdrant "ai-gateway/internal/qdrant"

	"github.com/sirupsen/logrus"
)

const importBatchSize = 100

func (s *Service) CreateImportJob(ctx context.Context, req *CreateImportJobRequest) (*ImportJob, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}

	collectionName := strings.TrimSpace(req.CollectionName)
	if collectionName == "" {
		return nil, fmt.Errorf("collection_name is required")
	}
	if strings.TrimSpace(req.FileName) == "" {
		return nil, fmt.Errorf("file_name is required")
	}
	if strings.TrimSpace(req.FilePath) == "" {
		return nil, fmt.Errorf("file_path is required")
	}
	if req.FileSize <= 0 {
		return nil, fmt.Errorf("file_size must be positive")
	}
	if req.TotalRecords <= 0 {
		return nil, fmt.Errorf("total_records must be positive")
	}

	collection, err := s.repo.Get(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	job := &ImportJob{
		ID:               fmt.Sprintf("job_%d", now.UnixNano()),
		CollectionID:     collection.ID,
		CollectionName:   collection.Name,
		FileName:         strings.TrimSpace(req.FileName),
		FilePath:         strings.TrimSpace(req.FilePath),
		FileSize:         req.FileSize,
		TotalRecords:     req.TotalRecords,
		ProcessedRecords: 0,
		FailedRecords:    0,
		RetryCount:       0,
		MaxRetries:       defaultMaxRetries(req.MaxRetries),
		Status:           ImportJobStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
		CreatedBy:        defaultString(req.CreatedBy, "system"),
	}

	if err := s.repo.CreateImportJob(ctx, job); err != nil {
		return nil, err
	}

	copyValue := *job
	return &copyValue, nil
}

func (s *Service) GetImportJob(ctx context.Context, id string) (*ImportJob, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	jobID := strings.TrimSpace(id)
	if jobID == "" {
		return nil, fmt.Errorf("id is required")
	}
	return s.repo.GetImportJob(ctx, jobID)
}

func (s *Service) ListImportJobs(ctx context.Context, query *ListImportJobsQuery) ([]ImportJob, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if query == nil {
		query = &ListImportJobsQuery{}
	}
	return s.repo.ListImportJobs(ctx, query)
}

func (s *Service) GetImportJobSummary(ctx context.Context, query *ListImportJobsQuery) (*ImportJobSummary, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if query == nil {
		query = &ListImportJobsQuery{}
	}
	return s.repo.SummarizeImportJobs(ctx, query)
}

func (s *Service) UpdateImportJobStatus(ctx context.Context, id string, req *UpdateImportJobStatusRequest) error {
	if s.repo == nil {
		return fmt.Errorf("repository is required")
	}
	if req == nil {
		return fmt.Errorf("request is required")
	}
	jobID := strings.TrimSpace(id)
	if jobID == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(string(req.Status)) == "" {
		return fmt.Errorf("status is required")
	}
	return s.repo.UpdateImportJobStatus(ctx, jobID, req)
}

func (s *Service) RunImportJob(ctx context.Context, id string) (*ImportJob, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	jobID := strings.TrimSpace(id)
	if jobID == "" {
		return nil, fmt.Errorf("id is required")
	}

	now := time.Now().UTC()
	started := now
	runningStatus := UpdateImportJobStatusRequest{
		Status:    ImportJobStatusRunning,
		StartedAt: &started,
	}
	if err := s.repo.UpdateImportJobStatus(ctx, jobID, &runningStatus); err != nil {
		return nil, err
	}

	job, err := s.repo.GetImportJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	points, parsedProcessed, parsedFailed, runErr := executeImportFile(job.FilePath)
	completed := time.Now().UTC()
	if runErr != nil {
		totalFailed := job.TotalRecords
		errText := runErr.Error()
		failedStatus := UpdateImportJobStatusRequest{
			Status:        ImportJobStatusFailed,
			FailedRecords: &totalFailed,
			ErrorMessage:  &errText,
			CompletedAt:   &completed,
		}
		if err := s.repo.UpdateImportJobStatus(ctx, jobID, &failedStatus); err != nil {
			return nil, err
		}
		s.logImportJobError(ctx, job, "import_run_failed", errText)
		return s.repo.GetImportJob(ctx, jobID)
	}

	processed := parsedProcessed
	failed := parsedFailed
	if s.backend != nil && len(points) > 0 {
		upserted := int64(0)
		for start := 0; start < len(points); start += importBatchSize {
			end := start + importBatchSize
			if end > len(points) {
				end = len(points)
			}
			if err := s.backend.UpsertPoints(ctx, job.CollectionName, points[start:end]); err != nil {
				remaining := int64(len(points) - start)
				failed = parsedFailed + remaining
				processed = upserted
				errText := fmt.Sprintf("upsert batch failed: %v", err)
				failedStatus := UpdateImportJobStatusRequest{
					Status:           ImportJobStatusFailed,
					ProcessedRecords: &processed,
					FailedRecords:    &failed,
					ErrorMessage:     &errText,
					CompletedAt:      &completed,
				}
				if updateErr := s.repo.UpdateImportJobStatus(ctx, jobID, &failedStatus); updateErr != nil {
					return nil, updateErr
				}
				s.logImportJobError(ctx, job, "import_upsert_failed", errText)
				return s.repo.GetImportJob(ctx, jobID)
			}
			upserted += int64(end - start)
		}
		processed = upserted
		failed = parsedFailed
	}

	completedStatus := UpdateImportJobStatusRequest{
		Status:           ImportJobStatusCompleted,
		ProcessedRecords: &processed,
		FailedRecords:    &failed,
		CompletedAt:      &completed,
	}
	if err := s.repo.UpdateImportJobStatus(ctx, jobID, &completedStatus); err != nil {
		return nil, err
	}

	return s.repo.GetImportJob(ctx, jobID)
}

func (s *Service) RetryImportJob(ctx context.Context, id string) (*ImportJob, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	jobID := strings.TrimSpace(id)
	if jobID == "" {
		return nil, fmt.Errorf("id is required")
	}

	job, err := s.repo.GetImportJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != ImportJobStatusFailed && job.Status != ImportJobStatusCanceled {
		return nil, fmt.Errorf("retry is only allowed for failed or canceled jobs")
	}
	if job.RetryCount >= job.MaxRetries {
		s.logImportJobError(ctx, job, "import_retry_exceeded", ErrImportJobRetryExceeded.Error())
		return nil, ErrImportJobRetryExceeded
	}

	resetProcessed := int64(0)
	resetFailed := int64(0)
	nextRetryCount := int64(job.RetryCount + 1)
	retryingStatus := UpdateImportJobStatusRequest{
		Status:           ImportJobStatusRetrying,
		ProcessedRecords: &resetProcessed,
		FailedRecords:    &resetFailed,
		RetryCount:       &nextRetryCount,
	}
	if err := s.repo.UpdateImportJobStatus(ctx, jobID, &retryingStatus); err != nil {
		return nil, err
	}

	return s.RunImportJob(ctx, jobID)
}

func (s *Service) CancelImportJob(ctx context.Context, id string) (*ImportJob, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	jobID := strings.TrimSpace(id)
	if jobID == "" {
		return nil, fmt.Errorf("id is required")
	}

	job, err := s.repo.GetImportJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if job.Status != ImportJobStatusPending && job.Status != ImportJobStatusRunning && job.Status != ImportJobStatusRetrying {
		return nil, fmt.Errorf("cancel is only allowed for pending, running or retrying jobs")
	}

	now := time.Now().UTC()
	status := UpdateImportJobStatusRequest{Status: ImportJobStatusCanceled, CompletedAt: &now}
	if err := s.repo.UpdateImportJobStatus(ctx, jobID, &status); err != nil {
		return nil, err
	}

	return s.repo.GetImportJob(ctx, jobID)
}

func (s *Service) RetryFailedImportJobs(ctx context.Context, limit int) ([]ImportJob, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	query := &ListImportJobsQuery{
		Status: ImportJobStatusFailed.String(),
		Limit:  limit,
	}
	jobs, err := s.repo.ListImportJobs(ctx, query)
	if err != nil {
		return nil, err
	}

	retried := make([]ImportJob, 0, len(jobs))
	for idx := range jobs {
		if jobs[idx].RetryCount >= jobs[idx].MaxRetries {
			continue
		}
		updated, retryErr := s.RetryImportJob(ctx, jobs[idx].ID)
		if retryErr != nil {
			continue
		}
		retried = append(retried, *updated)
	}

	return retried, nil
}

func (s *Service) GetImportJobErrors(ctx context.Context, id, action string, limit, offset int) ([]AuditLog, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	jobID := strings.TrimSpace(id)
	if jobID == "" {
		return nil, fmt.Errorf("id is required")
	}
	if _, err := s.repo.GetImportJob(ctx, jobID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	logs, err := s.repo.ListAuditLogs(ctx, &ListAuditLogsQuery{
		ResourceType: "import_job",
		ResourceID:   jobID,
		Action:       strings.TrimSpace(action),
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *Service) ListAuditLogs(ctx context.Context, query *ListAuditLogsQuery) ([]AuditLog, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if query == nil {
		query = &ListAuditLogsQuery{}
	}
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return s.repo.ListAuditLogs(ctx, query)
}

func defaultMaxRetries(v int) int {
	if v <= 0 {
		return 3
	}
	return v
}

func (s ImportJobStatus) String() string {
	return string(s)
}

func (s *Service) logImportJobError(ctx context.Context, job *ImportJob, action, details string) {
	if s == nil || s.repo == nil || job == nil {
		return
	}
	if err := s.repo.CreateAuditLog(ctx, &AuditLog{
		UserID:       defaultString(job.CreatedBy, "system"),
		Action:       action,
		ResourceType: "import_job",
		ResourceID:   job.ID,
		Details:      strings.TrimSpace(details),
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		logrus.WithError(err).WithField("job_id", job.ID).Warn("create import job audit log failed")
	}
}

func executeImportFile(filePath string) (points []internalqdrant.UpsertPoint, processed, failed int64, err error) {
	path := strings.TrimSpace(filePath)
	if path == "" {
		return nil, 0, 0, fmt.Errorf("file_path is required")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("read import file failed: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return processJSONImport(content)
	case ".csv":
		return processCSVImport(content)
	case ".pdf":
		return processPDFImport(content)
	default:
		return nil, 0, 0, fmt.Errorf("unsupported import file type: %s", ext)
	}
}

//nolint:gocritic // Return shape intentionally mirrors CSV processor.
func processJSONImport(content []byte) ([]internalqdrant.UpsertPoint, int64, int64, error) {
	var records []map[string]any
	if err := json.Unmarshal(content, &records); err != nil {
		return nil, 0, 0, fmt.Errorf("parse json import file failed: %w", err)
	}
	if len(records) == 0 {
		return nil, 0, 0, fmt.Errorf("json import file has no records")
	}

	points := make([]internalqdrant.UpsertPoint, 0, len(records))
	for idx := range records {
		vector, ok := extractVector(records[idx]["vector"])
		if !ok {
			continue
		}
		id := extractRecordID(records[idx]["id"], idx)
		payload := make(map[string]any, len(records[idx]))
		for key, value := range records[idx] {
			if key == "vector" || key == "id" {
				continue
			}
			payload[key] = value
		}
		points = append(points, internalqdrant.UpsertPoint{ID: id, Vector: vector, Payload: payload})
	}

	processed := int64(len(records))
	failed := int64(len(records) - len(points))
	return points, processed, failed, nil
}

//nolint:gocyclo,gocritic // CSV parsing has explicit branch handling for clarity and keeps return shape aligned with JSON parser.
func processCSVImport(content []byte) ([]internalqdrant.UpsertPoint, int64, int64, error) {
	reader := csv.NewReader(strings.NewReader(string(content)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("parse csv import file failed: %w", err)
	}
	if len(records) <= 1 {
		return nil, 0, 0, fmt.Errorf("csv import file has no data rows")
	}

	headers := records[0]
	idxID := findCSVHeaderIndex(headers, "id")
	idxVector := findCSVHeaderIndex(headers, "vector")
	idxPayload := findCSVHeaderIndex(headers, "payload")

	points := make([]internalqdrant.UpsertPoint, 0, len(records)-1)
	var processed int64
	var failed int64
	for rowIdx, row := range records[1:] {
		hasData := false
		for _, col := range row {
			if strings.TrimSpace(col) != "" {
				hasData = true
				break
			}
		}
		if hasData {
			processed++
			if idxVector >= 0 && idxVector < len(row) {
				vector, ok := parseCSVVector(row[idxVector])
				if ok {
					id := fmt.Sprintf("row_%d", rowIdx+1)
					if idxID >= 0 && idxID < len(row) && strings.TrimSpace(row[idxID]) != "" {
						id = strings.TrimSpace(row[idxID])
					}
					payload := map[string]any{}
					if idxPayload >= 0 && idxPayload < len(row) && strings.TrimSpace(row[idxPayload]) != "" {
						payload["payload"] = strings.TrimSpace(row[idxPayload])
					}
					points = append(points, internalqdrant.UpsertPoint{ID: id, Vector: vector, Payload: payload})
				}
			}
		} else {
			failed++
		}
	}

	if idxVector >= 0 {
		failed += processed - int64(len(points))
	}
	return points, processed, failed, nil
}

func findCSVHeaderIndex(headers []string, key string) int {
	target := strings.ToLower(strings.TrimSpace(key))
	for idx := range headers {
		if strings.ToLower(strings.TrimSpace(headers[idx])) == target {
			return idx
		}
	}
	return -1
}

func parseCSVVector(raw string) ([]float32, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}
	var values []float64
	if err := json.Unmarshal([]byte(trimmed), &values); err == nil {
		vector := make([]float32, 0, len(values))
		for _, value := range values {
			vector = append(vector, float32(value))
		}
		if len(vector) == 0 {
			return nil, false
		}
		return vector, true
	}

	parts := strings.Split(trimmed, ",")
	vector := make([]float32, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return nil, false
		}
		vector = append(vector, float32(value))
	}
	if len(vector) == 0 {
		return nil, false
	}
	return vector, true
}

func extractVector(value any) ([]float32, bool) {
	if value == nil {
		return nil, false
	}
	switch typed := value.(type) {
	case []any:
		vector := make([]float32, 0, len(typed))
		for _, item := range typed {
			number, ok := toFloat32(item)
			if !ok {
				return nil, false
			}
			vector = append(vector, number)
		}
		if len(vector) == 0 {
			return nil, false
		}
		return vector, true
	case []float64:
		vector := make([]float32, 0, len(typed))
		for _, item := range typed {
			vector = append(vector, float32(item))
		}
		if len(vector) == 0 {
			return nil, false
		}
		return vector, true
	default:
		return nil, false
	}
}

func toFloat32(value any) (float32, bool) {
	switch typed := value.(type) {
	case float64:
		return float32(typed), true
	case float32:
		return typed, true
	case int:
		return float32(typed), true
	case int64:
		return float32(typed), true
	default:
		return 0, false
	}
}

func extractRecordID(value any, idx int) string {
	if value == nil {
		return fmt.Sprintf("row_%d", idx+1)
	}
	if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	return fmt.Sprintf("row_%d", idx+1)
}

func processPDFImport(content []byte) (points []internalqdrant.UpsertPoint, processed, failed int64, err error) {
	text := strings.TrimSpace(string(content))
	if text == "" {
		return nil, 0, 0, fmt.Errorf("pdf import file has no text")
	}

	lines := strings.Split(text, "\n")
	points = make([]internalqdrant.UpsertPoint, 0, len(lines))
	for idx := range lines {
		line := strings.TrimSpace(lines[idx])
		if line == "" {
			continue
		}
		processed++
		vector := embedTextToVector(line)
		if len(vector) == 0 {
			failed++
			continue
		}
		points = append(points, internalqdrant.UpsertPoint{
			ID:      fmt.Sprintf("pdf_line_%d", idx+1),
			Vector:  vector,
			Payload: map[string]any{"text": line, "source_type": "pdf"},
		})
	}
	if processed == 0 {
		return nil, 0, 0, fmt.Errorf("pdf import file has no valid lines")
	}
	failed += processed - int64(len(points))
	return points, processed, failed, nil
}

func embedTextToVector(text string) []float32 {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return nil
	}
	hash := sha256.Sum256([]byte(normalized))
	vector := make([]float32, 8)
	for i := 0; i < 8; i++ {
		chunk := binary.BigEndian.Uint32(hash[i*4 : (i+1)*4])
		vector[i] = float32(chunk%10000) / 10000
	}
	return vector
}
