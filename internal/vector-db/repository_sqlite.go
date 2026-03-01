package vectordb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SQLiteRepository struct {
	db      *sql.DB
	initErr error
}

const sqliteLimitOffsetClause = " LIMIT ? OFFSET ?"

func NewSQLiteRepository(db *sql.DB) (*SQLiteRepository, error) {
	repo := &SQLiteRepository{db: db}
	if db != nil {
		if err := repo.ensureSchema(context.Background()); err != nil {
			repo.initErr = err
			return repo, err
		}
	}
	return repo, nil
}

func (r *SQLiteRepository) ensureSchema(ctx context.Context) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS vector_collections (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			dimension INTEGER NOT NULL,
			distance_metric TEXT NOT NULL DEFAULT 'cosine',
			index_type TEXT NOT NULL DEFAULT 'hnsw',
			hnsw_m INTEGER NOT NULL DEFAULT 16,
			hnsw_ef_construct INTEGER NOT NULL DEFAULT 100,
			ivf_nlist INTEGER NOT NULL DEFAULT 1024,
			storage_backend TEXT NOT NULL DEFAULT 'qdrant',
			tags TEXT,
			environment TEXT NOT NULL DEFAULT 'default',
			status TEXT NOT NULL DEFAULT 'active',
			vector_count INTEGER NOT NULL DEFAULT 0,
			indexed_count INTEGER NOT NULL DEFAULT 0,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			created_by TEXT NOT NULL,
			is_public INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_collections_name ON vector_collections(name)`,
		`CREATE INDEX IF NOT EXISTS idx_collections_status ON vector_collections(status)`,
		`CREATE INDEX IF NOT EXISTS idx_collections_environment ON vector_collections(environment)`,
		`CREATE TABLE IF NOT EXISTS vector_import_jobs (
			id TEXT PRIMARY KEY,
			collection_id TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			total_records INTEGER NOT NULL,
			processed_records INTEGER NOT NULL DEFAULT 0,
			failed_records INTEGER NOT NULL DEFAULT 0,
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			status TEXT NOT NULL DEFAULT 'pending',
			error_message TEXT,
			started_at TEXT,
			completed_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			created_by TEXT NOT NULL,
			FOREIGN KEY (collection_id) REFERENCES vector_collections(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_import_jobs_collection ON vector_import_jobs(collection_id)`,
		`CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON vector_import_jobs(status)`,
		`CREATE TABLE IF NOT EXISTS vector_audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			details TEXT,
			ip_address TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON vector_audit_logs(resource_type)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON vector_audit_logs(resource_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON vector_audit_logs(action)`,
		`CREATE TABLE IF NOT EXISTS vector_alert_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			metric TEXT NOT NULL,
			operator TEXT NOT NULL,
			threshold REAL NOT NULL,
			duration TEXT NOT NULL,
			channels TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alert_rules_metric ON vector_alert_rules(metric)`,
		`CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON vector_alert_rules(enabled)`,
		`CREATE TABLE IF NOT EXISTS vector_api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key_hash TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_vector_api_keys_role ON vector_api_keys(role)`,
		`CREATE INDEX IF NOT EXISTS idx_vector_api_keys_enabled ON vector_api_keys(enabled)`,
		`CREATE TABLE IF NOT EXISTS vector_backup_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_name TEXT NOT NULL,
			snapshot_name TEXT NOT NULL,
			action TEXT NOT NULL,
			status TEXT NOT NULL,
			source_backup_id INTEGER NOT NULL DEFAULT 0,
			error_message TEXT,
			started_at TEXT,
			completed_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			created_by TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_tasks_collection ON vector_backup_tasks(collection_name)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_tasks_status ON vector_backup_tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_tasks_action ON vector_backup_tasks(action)`,
	}

	for _, query := range queries {
		if _, err := r.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("ensure vector_collections schema failed: %w", err)
		}
	}

	if err := r.ensureColumn(ctx, "vector_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := r.ensureColumn(ctx, "indexed_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := r.ensureColumn(ctx, "size_bytes", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := r.ensureColumn(ctx, "hnsw_m", "INTEGER NOT NULL DEFAULT 16"); err != nil {
		return err
	}
	if err := r.ensureColumn(ctx, "hnsw_ef_construct", "INTEGER NOT NULL DEFAULT 100"); err != nil {
		return err
	}
	if err := r.ensureColumn(ctx, "ivf_nlist", "INTEGER NOT NULL DEFAULT 1024"); err != nil {
		return err
	}
	if err := r.ensureImportJobColumn(ctx, "retry_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	return r.ensureImportJobColumn(ctx, "max_retries", "INTEGER NOT NULL DEFAULT 3")
}

//nolint:dupl // Same schema-probe workflow as ensureImportJobColumn for another table.
func (r *SQLiteRepository) ensureColumn(ctx context.Context, name, columnType string) error {
	rows, err := r.db.QueryContext(ctx, `PRAGMA table_info(vector_collections)`)
	if err != nil {
		return fmt.Errorf("query vector_collections columns failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			colName   string
			colType   string
			notNull   int
			defaultV  sql.NullString
			primaryID int
		)
		if scanErr := rows.Scan(&cid, &colName, &colType, &notNull, &defaultV, &primaryID); scanErr != nil {
			return fmt.Errorf("scan vector_collections columns failed: %w", scanErr)
		}
		if colName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate vector_collections columns failed: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE vector_collections ADD COLUMN %s %s`, name, columnType)); err != nil {
		return fmt.Errorf("add %s column failed: %w", name, err)
	}
	return nil
}

//nolint:dupl // Same schema-probe workflow as ensureColumn for another table.
func (r *SQLiteRepository) ensureImportJobColumn(ctx context.Context, name, columnType string) error {
	rows, err := r.db.QueryContext(ctx, `PRAGMA table_info(vector_import_jobs)`)
	if err != nil {
		return fmt.Errorf("query vector_import_jobs columns failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			colName   string
			colType   string
			notNull   int
			defaultV  sql.NullString
			primaryID int
		)
		if scanErr := rows.Scan(&cid, &colName, &colType, &notNull, &defaultV, &primaryID); scanErr != nil {
			return fmt.Errorf("scan vector_import_jobs columns failed: %w", scanErr)
		}
		if colName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate vector_import_jobs columns failed: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE vector_import_jobs ADD COLUMN %s %s`, name, columnType)); err != nil {
		return fmt.Errorf("add vector_import_jobs.%s failed: %w", name, err)
	}
	return nil
}

func (r *SQLiteRepository) Create(ctx context.Context, col *Collection) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if col == nil {
		return fmt.Errorf("collection is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	tagsJSON, err := marshalTags(col.Tags)
	if err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO vector_collections (
			id, name, description, dimension, distance_metric, index_type, hnsw_m, hnsw_ef_construct, ivf_nlist, storage_backend,
			tags, environment, status, vector_count, indexed_count, size_bytes,
			created_at, updated_at, created_by, is_public
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, col.ID, col.Name, col.Description, col.Dimension, col.DistanceMetric, col.IndexType, col.HNSWM, col.HNSWEFConstruct, col.IVFNList, col.StorageBackend,
		tagsJSON, col.Environment, col.Status, col.VectorCount, col.IndexedCount, col.SizeBytes,
		col.CreatedAt.Format(time.RFC3339), col.UpdatedAt.Format(time.RFC3339), col.CreatedBy, boolToInt(col.IsPublic)); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ErrCollectionExists
		}
		return fmt.Errorf("create collection metadata failed: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) Get(ctx context.Context, name string) (*Collection, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, dimension, distance_metric, index_type, hnsw_m, hnsw_ef_construct, ivf_nlist, storage_backend,
			tags, environment, status, vector_count, indexed_count, size_bytes,
			created_at, updated_at, created_by, is_public
		FROM vector_collections
		WHERE name = ?
	`, strings.TrimSpace(name))

	col, err := scanCollectionRow(row)
	if err != nil {
		if errorsIsNoRows(err) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("get collection metadata failed: %w", err)
	}
	return col, nil
}

//nolint:gocyclo // Explicit filter branches keep query construction readable.
func (r *SQLiteRepository) List(ctx context.Context, query *ListCollectionsQuery) ([]Collection, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	if query == nil {
		query = &ListCollectionsQuery{}
	}

	where := []string{"1=1"}
	args := make([]interface{}, 0, 10)

	if v := strings.TrimSpace(query.Name); v != "" {
		where = append(where, "name = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(query.Environment); v != "" {
		where = append(where, "environment = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(query.Status); v != "" {
		where = append(where, "status = ?")
		args = append(args, v)
	}
	if query.IsPublic != nil {
		where = append(where, "is_public = ?")
		args = append(args, boolToInt(*query.IsPublic))
	}
	if v := strings.TrimSpace(query.Tag); v != "" {
		where = append(where, "tags LIKE ? ESCAPE '\\'")
		args = append(args, `%"`+escapeLikePattern(v)+`"%`)
	}
	if v := strings.TrimSpace(query.Search); v != "" {
		where = append(where, "(LOWER(name) LIKE ? OR LOWER(description) LIKE ?)")
		like := "%" + strings.ToLower(v) + "%"
		args = append(args, like, like)
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	limitClause := ""
	if query.Limit > 0 {
		limitClause = sqliteLimitOffsetClause
		args = append(args, query.Limit, offset)
	}

	//nolint:gosec // Dynamic clauses are constrained to internal whitelisted filters.
	q := `
		SELECT id, name, description, dimension, distance_metric, index_type, hnsw_m, hnsw_ef_construct, ivf_nlist, storage_backend,
			tags, environment, status, vector_count, indexed_count, size_bytes,
			created_at, updated_at, created_by, is_public
		FROM vector_collections
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_at DESC` + limitClause

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list collection metadata failed: %w", err)
	}
	defer rows.Close()

	items := make([]Collection, 0)
	for rows.Next() {
		col, scanErr := scanCollectionRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *col)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate collection metadata failed: %w", err)
	}
	if query.Limit <= 0 && offset > 0 {
		if offset >= len(items) {
			return []Collection{}, nil
		}
		return items[offset:], nil
	}
	return items, nil
}

//nolint:gocyclo // Update fields are intentionally explicit.
func (r *SQLiteRepository) Update(ctx context.Context, name string, req *UpdateCollectionRequest) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	updates := make([]string, 0, 8)
	args := make([]interface{}, 0, 10)

	if req.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, strings.TrimSpace(*req.Description))
	}
	if req.DistanceMetric != nil {
		if metric := strings.TrimSpace(*req.DistanceMetric); metric != "" {
			updates = append(updates, "distance_metric = ?")
			args = append(args, metric)
		}
	}
	if req.IndexType != nil {
		if indexType := strings.TrimSpace(*req.IndexType); indexType != "" {
			updates = append(updates, "index_type = ?")
			args = append(args, indexType)
		}
	}
	if req.HNSWM != nil && *req.HNSWM > 0 {
		updates = append(updates, "hnsw_m = ?")
		args = append(args, *req.HNSWM)
	}
	if req.HNSWEFConstruct != nil && *req.HNSWEFConstruct > 0 {
		updates = append(updates, "hnsw_ef_construct = ?")
		args = append(args, *req.HNSWEFConstruct)
	}
	if req.IVFNList != nil && *req.IVFNList > 0 {
		updates = append(updates, "ivf_nlist = ?")
		args = append(args, *req.IVFNList)
	}
	if req.StorageBackend != nil {
		if backend := strings.TrimSpace(*req.StorageBackend); backend != "" {
			updates = append(updates, "storage_backend = ?")
			args = append(args, backend)
		}
	}
	if req.Tags != nil {
		tagsJSON, err := marshalTags(req.Tags)
		if err != nil {
			return err
		}
		updates = append(updates, "tags = ?")
		args = append(args, tagsJSON)
	}
	if req.Environment != nil {
		if environment := strings.TrimSpace(*req.Environment); environment != "" {
			updates = append(updates, "environment = ?")
			args = append(args, environment)
		}
	}
	if req.Status != nil {
		if status := strings.TrimSpace(*req.Status); status != "" {
			updates = append(updates, "status = ?")
			args = append(args, status)
		}
	}
	if req.IsPublic != nil {
		updates = append(updates, "is_public = ?")
		args = append(args, boolToInt(*req.IsPublic))
	}
	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(name))

	//nolint:gosec // Dynamic set clauses are generated from fixed field names.
	query := `UPDATE vector_collections SET ` + strings.Join(updates, ", ") + ` WHERE name = ?`
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update collection metadata failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read update affected rows failed: %w", err)
	}
	if affected == 0 {
		return ErrCollectionNotFound
	}
	return nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, name string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM vector_collections WHERE name = ?`, strings.TrimSpace(name))
	if err != nil {
		return fmt.Errorf("delete collection metadata failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete affected rows failed: %w", err)
	}
	if affected == 0 {
		return ErrCollectionNotFound
	}
	return nil
}

func (r *SQLiteRepository) UpdateCollectionStats(ctx context.Context, name string, vectorCount, indexedCount, sizeBytes int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE vector_collections
		SET vector_count = ?, indexed_count = ?, size_bytes = ?, updated_at = ?
		WHERE name = ?
	`, vectorCount, indexedCount, sizeBytes, time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(name))
	if err != nil {
		return fmt.Errorf("update collection stats failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read update stats affected rows failed: %w", err)
	}
	if affected == 0 {
		return ErrCollectionNotFound
	}
	return nil
}

func (r *SQLiteRepository) CreateImportJob(ctx context.Context, job *ImportJob) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if job == nil {
		return fmt.Errorf("import job is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vector_import_jobs (
			id, collection_id, file_name, file_path, file_size,
			total_records, processed_records, failed_records, retry_count, max_retries, status,
			error_message, started_at, completed_at, created_at, updated_at, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		job.ID,
		job.CollectionID,
		job.FileName,
		job.FilePath,
		job.FileSize,
		job.TotalRecords,
		job.ProcessedRecords,
		job.FailedRecords,
		job.RetryCount,
		job.MaxRetries,
		string(job.Status),
		nullableString(job.ErrorMessage),
		nullableTime(job.StartedAt),
		nullableTime(job.CompletedAt),
		job.CreatedAt.Format(time.RFC3339),
		job.UpdatedAt.Format(time.RFC3339),
		job.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create import job failed: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) GetImportJob(ctx context.Context, id string) (*ImportJob, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT j.id, j.collection_id, c.name, j.file_name, j.file_path, j.file_size,
			j.total_records, j.processed_records, j.failed_records, j.retry_count, j.max_retries,
			j.status, j.error_message,
			j.started_at, j.completed_at, j.created_at, j.updated_at, j.created_by
		FROM vector_import_jobs j
		LEFT JOIN vector_collections c ON c.id = j.collection_id
		WHERE j.id = ?
	`, strings.TrimSpace(id))

	job, err := scanImportJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrImportJobNotFound
		}
		return nil, fmt.Errorf("get import job failed: %w", err)
	}

	return job, nil
}

//nolint:gocyclo // Branching is explicit for list filters and pagination behavior.
func (r *SQLiteRepository) ListImportJobs(ctx context.Context, query *ListImportJobsQuery) ([]ImportJob, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	if query == nil {
		query = &ListImportJobsQuery{}
	}

	where := []string{"1=1"}
	args := make([]interface{}, 0, 8)

	if name := strings.TrimSpace(query.CollectionName); name != "" {
		where = append(where, "c.name = ?")
		args = append(args, name)
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		where = append(where, "j.status = ?")
		args = append(args, status)
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	limitClause := ""
	if query.Limit > 0 {
		limitClause = sqliteLimitOffsetClause
		args = append(args, query.Limit, offset)
	}

	//nolint:gosec // Dynamic where/order clauses are internally controlled.
	stmt := `
		SELECT j.id, j.collection_id, c.name, j.file_name, j.file_path, j.file_size,
			j.total_records, j.processed_records, j.failed_records, j.retry_count, j.max_retries,
			j.status, j.error_message,
			j.started_at, j.completed_at, j.created_at, j.updated_at, j.created_by
		FROM vector_import_jobs j
		LEFT JOIN vector_collections c ON c.id = j.collection_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY j.created_at DESC` + limitClause

	rows, err := r.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("list import jobs failed: %w", err)
	}
	defer rows.Close()

	items := make([]ImportJob, 0)
	for rows.Next() {
		job, scanErr := scanImportJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import jobs failed: %w", err)
	}
	if query.Limit <= 0 && offset > 0 {
		if offset >= len(items) {
			return []ImportJob{}, nil
		}
		return items[offset:], nil
	}

	return items, nil
}

//nolint:gocyclo // Explicit summary branches keep status mapping simple.
func (r *SQLiteRepository) SummarizeImportJobs(ctx context.Context, query *ListImportJobsQuery) (*ImportJobSummary, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	if query == nil {
		query = &ListImportJobsQuery{}
	}

	where := []string{"1=1"}
	args := make([]interface{}, 0, 2)

	if name := strings.TrimSpace(query.CollectionName); name != "" {
		where = append(where, "c.name = ?")
		args = append(args, name)
	}

	//nolint:gosec // Dynamic where clauses are constrained to whitelisted fields.
	stmt := `
		SELECT j.status, COUNT(1)
		FROM vector_import_jobs j
		LEFT JOIN vector_collections c ON c.id = j.collection_id
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY j.status`

	rows, err := r.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("summarize import jobs failed: %w", err)
	}
	defer rows.Close()

	summary := &ImportJobSummary{}
	for rows.Next() {
		var (
			status string
			count  int
		)
		if scanErr := rows.Scan(&status, &count); scanErr != nil {
			return nil, fmt.Errorf("scan import jobs summary failed: %w", scanErr)
		}
		summary.Total += count
		switch strings.ToLower(strings.TrimSpace(status)) {
		case ImportJobStatusPending.String():
			summary.Pending += count
		case ImportJobStatusRunning.String():
			summary.Running += count
		case ImportJobStatusRetrying.String():
			summary.Retrying += count
		case ImportJobStatusCompleted.String():
			summary.Completed += count
		case ImportJobStatusFailed.String():
			summary.Failed += count
		case ImportJobStatusCanceled.String():
			summary.Canceled += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import jobs summary failed: %w", err)
	}

	return summary, nil
}

func (r *SQLiteRepository) UpdateImportJobStatus(ctx context.Context, id string, req *UpdateImportJobStatusRequest) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	updates := []string{"status = ?", "updated_at = ?"}
	args := []interface{}{string(req.Status), time.Now().UTC().Format(time.RFC3339)}

	if req.ProcessedRecords != nil {
		updates = append(updates, "processed_records = ?")
		args = append(args, *req.ProcessedRecords)
	}
	if req.FailedRecords != nil {
		updates = append(updates, "failed_records = ?")
		args = append(args, *req.FailedRecords)
	}
	if req.RetryCount != nil {
		updates = append(updates, "retry_count = ?")
		args = append(args, *req.RetryCount)
	}
	if req.ErrorMessage != nil {
		updates = append(updates, "error_message = ?")
		args = append(args, nullableString(*req.ErrorMessage))
	}
	if req.StartedAt != nil {
		updates = append(updates, "started_at = ?")
		args = append(args, req.StartedAt.UTC().Format(time.RFC3339))
	}
	if req.CompletedAt != nil {
		updates = append(updates, "completed_at = ?")
		args = append(args, req.CompletedAt.UTC().Format(time.RFC3339))
	}

	args = append(args, strings.TrimSpace(id))
	//nolint:gosec // Dynamic set clause is assembled from fixed status fields.
	stmt := `UPDATE vector_import_jobs SET ` + strings.Join(updates, ", ") + ` WHERE id = ?`
	result, err := r.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return fmt.Errorf("update import job failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read import job update rows failed: %w", err)
	}
	if affected == 0 {
		return ErrImportJobNotFound
	}

	return nil
}

func (r *SQLiteRepository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if log == nil {
		return fmt.Errorf("audit log is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	createdAt := log.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vector_audit_logs (user_id, action, resource_type, resource_id, details, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		defaultString(log.UserID, "system"),
		defaultString(log.Action, "unknown"),
		defaultString(log.ResourceType, "unknown"),
		defaultString(log.ResourceID, "unknown"),
		nullableString(log.Details),
		nullableString(log.IPAddress),
		createdAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create audit log failed: %w", err)
	}
	return nil
}

//nolint:gocyclo // Filter branching is straightforward and intentional.
func (r *SQLiteRepository) ListAuditLogs(ctx context.Context, query *ListAuditLogsQuery) ([]AuditLog, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	if query == nil {
		query = &ListAuditLogsQuery{}
	}

	where := []string{"1=1"}
	args := make([]interface{}, 0, 6)

	if v := strings.TrimSpace(query.ResourceType); v != "" {
		where = append(where, "resource_type = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(query.ResourceID); v != "" {
		where = append(where, "resource_id = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(query.Action); v != "" {
		where = append(where, "action = ?")
		args = append(args, v)
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	//nolint:gosec // Dynamic where clause uses controlled field list.
	stmt := `
		SELECT id, user_id, action, resource_type, resource_id, details, ip_address, created_at
		FROM vector_audit_logs
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY id DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs failed: %w", err)
	}
	defer rows.Close()

	items := make([]AuditLog, 0)
	for rows.Next() {
		var (
			item       AuditLog
			details    sql.NullString
			ipAddr     sql.NullString
			createdRaw string
		)
		if scanErr := rows.Scan(&item.ID, &item.UserID, &item.Action, &item.ResourceType, &item.ResourceID, &details, &ipAddr, &createdRaw); scanErr != nil {
			return nil, fmt.Errorf("scan audit log failed: %w", scanErr)
		}
		item.Details = details.String
		item.IPAddress = ipAddr.String
		if t, parseErr := time.Parse(time.RFC3339, createdRaw); parseErr == nil {
			item.CreatedAt = t
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs failed: %w", err)
	}

	return items, nil
}

func (r *SQLiteRepository) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if rule == nil {
		return fmt.Errorf("alert rule is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	channelsJSON, err := marshalTags(rule.Channels)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO vector_alert_rules (name, metric, operator, threshold, duration, channels, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rule.Name,
		rule.Metric,
		rule.Operator,
		rule.Threshold,
		rule.Duration,
		channelsJSON,
		boolToInt(rule.Enabled),
		rule.CreatedAt.Format(time.RFC3339),
		rule.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create alert rule failed: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("read alert rule id failed: %w", err)
	}
	rule.ID = id
	return nil
}

func (r *SQLiteRepository) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, metric, operator, threshold, duration, channels, enabled, created_at, updated_at
		FROM vector_alert_rules
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list alert rules failed: %w", err)
	}
	defer rows.Close()

	rules := make([]AlertRule, 0)
	for rows.Next() {
		var (
			rule       AlertRule
			channels   sql.NullString
			enabled    int
			createdRaw string
			updatedRaw string
		)
		if scanErr := rows.Scan(&rule.ID, &rule.Name, &rule.Metric, &rule.Operator, &rule.Threshold, &rule.Duration, &channels, &enabled, &createdRaw, &updatedRaw); scanErr != nil {
			return nil, fmt.Errorf("scan alert rule failed: %w", scanErr)
		}
		rule.Channels = unmarshalTags(channels.String)
		rule.Enabled = enabled == 1
		if parsed, parseErr := time.Parse(time.RFC3339, createdRaw); parseErr == nil {
			rule.CreatedAt = parsed
		}
		if parsed, parseErr := time.Parse(time.RFC3339, updatedRaw); parseErr == nil {
			rule.UpdatedAt = parsed
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alert rules failed: %w", err)
	}

	return rules, nil
}

//nolint:gocyclo // Update fields are intentionally explicit.
func (r *SQLiteRepository) UpdateAlertRule(ctx context.Context, id int64, req *UpdateAlertRuleRequest) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	updates := make([]string, 0, 8)
	args := make([]interface{}, 0, 10)

	if req.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, strings.TrimSpace(*req.Name))
	}
	if req.Metric != nil {
		updates = append(updates, "metric = ?")
		args = append(args, strings.TrimSpace(*req.Metric))
	}
	if req.Operator != nil {
		updates = append(updates, "operator = ?")
		args = append(args, strings.TrimSpace(*req.Operator))
	}
	if req.Threshold != nil {
		updates = append(updates, "threshold = ?")
		args = append(args, *req.Threshold)
	}
	if req.Duration != nil {
		updates = append(updates, "duration = ?")
		args = append(args, strings.TrimSpace(*req.Duration))
	}
	if req.Channels != nil {
		channelsJSON, err := marshalTags(*req.Channels)
		if err != nil {
			return err
		}
		updates = append(updates, "channels = ?")
		args = append(args, channelsJSON)
	}
	if req.Enabled != nil {
		updates = append(updates, "enabled = ?")
		args = append(args, boolToInt(*req.Enabled))
	}
	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339), id)
	//nolint:gosec // Dynamic set clause is assembled from fixed field names.
	query := `UPDATE vector_alert_rules SET ` + strings.Join(updates, ", ") + ` WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update alert rule failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read alert rule update rows failed: %w", err)
	}
	if affected == 0 {
		return ErrAlertRuleNotFound
	}
	return nil
}

//nolint:dupl // Delete workflow intentionally mirrors other delete helpers with distinct table/error mapping.
func (r *SQLiteRepository) DeleteAlertRule(ctx context.Context, id int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM vector_alert_rules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete alert rule failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read alert rule delete rows failed: %w", err)
	}
	if affected == 0 {
		return ErrAlertRuleNotFound
	}
	return nil
}

func (r *SQLiteRepository) CreateVectorAPIKey(ctx context.Context, key *VectorAPIKey) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if key == nil {
		return fmt.Errorf("vector api key is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vector_api_keys (key_hash, role, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(key_hash) DO UPDATE SET role = excluded.role, enabled = excluded.enabled, updated_at = excluded.updated_at
	`, key.KeyHash, strings.TrimSpace(key.Role), boolToInt(key.Enabled), key.CreatedAt.Format(time.RFC3339), key.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create vector api key failed: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) GetVectorAPIKeyByHash(ctx context.Context, keyHash string) (*VectorAPIKey, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT id, key_hash, role, enabled, created_at, updated_at
		FROM vector_api_keys WHERE key_hash = ?
	`, strings.TrimSpace(keyHash))

	var (
		item       VectorAPIKey
		enabled    int
		createdRaw string
		updatedRaw string
	)
	if err := row.Scan(&item.ID, &item.KeyHash, &item.Role, &enabled, &createdRaw, &updatedRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVectorAPIKeyNotFound
		}
		return nil, fmt.Errorf("get vector api key failed: %w", err)
	}
	item.Enabled = enabled == 1
	if parsed, parseErr := time.Parse(time.RFC3339, createdRaw); parseErr == nil {
		item.CreatedAt = parsed
	}
	if parsed, parseErr := time.Parse(time.RFC3339, updatedRaw); parseErr == nil {
		item.UpdatedAt = parsed
	}

	return &item, nil
}

func (r *SQLiteRepository) ListVectorAPIKeys(ctx context.Context) ([]VectorAPIKey, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, key_hash, role, enabled, created_at, updated_at
		FROM vector_api_keys
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list vector api keys failed: %w", err)
	}
	defer rows.Close()

	items := make([]VectorAPIKey, 0)
	for rows.Next() {
		var (
			item       VectorAPIKey
			enabled    int
			createdRaw string
			updatedRaw string
		)
		if scanErr := rows.Scan(&item.ID, &item.KeyHash, &item.Role, &enabled, &createdRaw, &updatedRaw); scanErr != nil {
			return nil, fmt.Errorf("scan vector api key failed: %w", scanErr)
		}
		item.Enabled = enabled == 1
		if parsed, parseErr := time.Parse(time.RFC3339, createdRaw); parseErr == nil {
			item.CreatedAt = parsed
		}
		if parsed, parseErr := time.Parse(time.RFC3339, updatedRaw); parseErr == nil {
			item.UpdatedAt = parsed
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vector api keys failed: %w", err)
	}

	return items, nil
}

//nolint:dupl // Delete workflow intentionally mirrors other delete helpers with distinct table/error mapping.
func (r *SQLiteRepository) DeleteVectorAPIKey(ctx context.Context, id int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM vector_api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete vector api key failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read vector api key delete rows failed: %w", err)
	}
	if affected == 0 {
		return ErrVectorAPIKeyNotFound
	}
	return nil
}

func (r *SQLiteRepository) CreateBackupTask(ctx context.Context, task *BackupTask) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if task == nil {
		return fmt.Errorf("backup task is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO vector_backup_tasks (
			collection_name, snapshot_name, action, status, source_backup_id, error_message,
			started_at, completed_at, created_at, updated_at, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		task.CollectionName,
		task.SnapshotName,
		task.Action,
		task.Status,
		task.SourceBackupID,
		nullableString(task.ErrorMessage),
		nullableTime(task.StartedAt),
		nullableTime(task.CompletedAt),
		task.CreatedAt.Format(time.RFC3339),
		task.UpdatedAt.Format(time.RFC3339),
		task.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create backup task failed: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("read backup task id failed: %w", err)
	}
	task.ID = id
	return nil
}

func (r *SQLiteRepository) GetBackupTask(ctx context.Context, id int64) (*BackupTask, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT id, collection_name, snapshot_name, action, status, source_backup_id, error_message,
			started_at, completed_at, created_at, updated_at, created_by
		FROM vector_backup_tasks WHERE id = ?
	`, id)
	task, err := scanBackupTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBackupTaskNotFound
		}
		return nil, fmt.Errorf("get backup task failed: %w", err)
	}
	return task, nil
}

//nolint:gocyclo // Explicit filters for readability.
func (r *SQLiteRepository) ListBackupTasks(ctx context.Context, query *ListBackupsQuery) ([]BackupTask, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}
	if query == nil {
		query = &ListBackupsQuery{}
	}

	where := []string{"1=1"}
	args := make([]interface{}, 0, 6)
	if name := strings.TrimSpace(query.CollectionName); name != "" {
		where = append(where, "collection_name = ?")
		args = append(args, name)
	}
	if action := strings.TrimSpace(query.Action); action != "" {
		where = append(where, "action = ?")
		args = append(args, action)
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	limitClause := ""
	if query.Limit > 0 {
		limitClause = sqliteLimitOffsetClause
		args = append(args, query.Limit, offset)
	}

	//nolint:gosec // Dynamic filters are from fixed fields.
	stmt := `
		SELECT id, collection_name, snapshot_name, action, status, source_backup_id, error_message,
			started_at, completed_at, created_at, updated_at, created_by
		FROM vector_backup_tasks
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY id DESC` + limitClause

	rows, err := r.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("list backup tasks failed: %w", err)
	}
	defer rows.Close()

	items := make([]BackupTask, 0)
	for rows.Next() {
		task, scanErr := scanBackupTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup tasks failed: %w", err)
	}
	if query.Limit <= 0 && offset > 0 {
		if offset >= len(items) {
			return []BackupTask{}, nil
		}
		return items[offset:], nil
	}
	return items, nil
}

func (r *SQLiteRepository) UpdateBackupTask(ctx context.Context, id int64, req *UpdateBackupTaskRequest) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if err := r.ensureSchema(ctx); err != nil {
		return err
	}

	updates := []string{"status = ?", "updated_at = ?"}
	args := []interface{}{string(req.Status), time.Now().UTC().Format(time.RFC3339)}
	if req.ErrorMessage != nil {
		updates = append(updates, "error_message = ?")
		args = append(args, nullableString(*req.ErrorMessage))
	}
	if req.StartedAt != nil {
		updates = append(updates, "started_at = ?")
		args = append(args, req.StartedAt.UTC().Format(time.RFC3339))
	}
	if req.CompletedAt != nil {
		updates = append(updates, "completed_at = ?")
		args = append(args, req.CompletedAt.UTC().Format(time.RFC3339))
	}
	args = append(args, id)

	//nolint:gosec // Dynamic set clause is assembled from fixed fields.
	stmt := `UPDATE vector_backup_tasks SET ` + strings.Join(updates, ", ") + ` WHERE id = ?`
	result, err := r.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return fmt.Errorf("update backup task failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read backup task update rows failed: %w", err)
	}
	if affected == 0 {
		return ErrBackupTaskNotFound
	}
	return nil
}

func (r *SQLiteRepository) DeleteOldBackupTasks(ctx context.Context, collectionName string, keepLatest int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("sqlite database is required")
	}
	if r.initErr != nil {
		return 0, r.initErr
	}
	if err := r.ensureSchema(ctx); err != nil {
		return 0, err
	}
	if strings.TrimSpace(collectionName) == "" {
		return 0, fmt.Errorf("collection_name is required")
	}
	if keepLatest < 1 {
		keepLatest = 1
	}

	result, err := r.db.ExecContext(ctx, `
		DELETE FROM vector_backup_tasks
		WHERE id IN (
			SELECT id FROM vector_backup_tasks
			WHERE collection_name = ? AND action = ?
			ORDER BY created_at DESC
			LIMIT -1 OFFSET ?
		)
	`, strings.TrimSpace(collectionName), string(BackupActionBackup), keepLatest)
	if err != nil {
		return 0, fmt.Errorf("delete old backup tasks failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read delete old backup tasks affected rows failed: %w", err)
	}
	return affected, nil
}

type backupTaskScanner interface {
	Scan(dest ...interface{}) error
}

func scanBackupTask(scanner backupTaskScanner) (*BackupTask, error) {
	var (
		task         BackupTask
		errorMessage sql.NullString
		startedAt    sql.NullString
		completedAt  sql.NullString
		createdAt    string
		updatedAt    string
	)
	if err := scanner.Scan(
		&task.ID,
		&task.CollectionName,
		&task.SnapshotName,
		&task.Action,
		&task.Status,
		&task.SourceBackupID,
		&errorMessage,
		&startedAt,
		&completedAt,
		&createdAt,
		&updatedAt,
		&task.CreatedBy,
	); err != nil {
		return nil, err
	}
	task.ErrorMessage = errorMessage.String
	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		task.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		task.UpdatedAt = parsed
	}
	if startedAt.Valid {
		if parsed, err := time.Parse(time.RFC3339, startedAt.String); err == nil {
			t := parsed
			task.StartedAt = &t
		}
	}
	if completedAt.Valid {
		if parsed, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
			t := parsed
			task.CompletedAt = &t
		}
	}
	return &task, nil
}

type collectionScanner interface {
	Scan(dest ...interface{}) error
}

func scanCollectionRow(row collectionScanner) (*Collection, error) {
	return scanCollection(row)
}

func scanCollectionRows(rows collectionScanner) (*Collection, error) {
	return scanCollection(rows)
}

func scanCollection(scanner collectionScanner) (*Collection, error) {
	var (
		col        Collection
		tagsRaw    sql.NullString
		createdRaw string
		updatedRaw string
		isPublic   int
	)
	if err := scanner.Scan(
		&col.ID,
		&col.Name,
		&col.Description,
		&col.Dimension,
		&col.DistanceMetric,
		&col.IndexType,
		&col.HNSWM,
		&col.HNSWEFConstruct,
		&col.IVFNList,
		&col.StorageBackend,
		&tagsRaw,
		&col.Environment,
		&col.Status,
		&col.VectorCount,
		&col.IndexedCount,
		&col.SizeBytes,
		&createdRaw,
		&updatedRaw,
		&col.CreatedBy,
		&isPublic,
	); err != nil {
		return nil, err
	}
	col.IsPublic = isPublic == 1
	col.Tags = unmarshalTags(tagsRaw.String)

	if t, err := time.Parse(time.RFC3339, createdRaw); err == nil {
		col.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedRaw); err == nil {
		col.UpdatedAt = t
	}

	return &col, nil
}

func marshalTags(tags []string) (string, error) {
	if tags == nil {
		return "[]", nil
	}
	clean := copyTags(tags)
	bytes, err := json.Marshal(clean)
	if err != nil {
		return "", fmt.Errorf("marshal tags failed: %w", err)
	}
	return string(bytes), nil
}

func unmarshalTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	tags := make([]string, 0)
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	return copyTags(tags)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func nullableTime(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

type importJobScanner interface {
	Scan(dest ...interface{}) error
}

func scanImportJob(scanner importJobScanner) (*ImportJob, error) {
	var (
		job          ImportJob
		collection   sql.NullString
		errorMessage sql.NullString
		startedAt    sql.NullString
		completedAt  sql.NullString
		createdAt    string
		updatedAt    string
	)

	if err := scanner.Scan(
		&job.ID,
		&job.CollectionID,
		&collection,
		&job.FileName,
		&job.FilePath,
		&job.FileSize,
		&job.TotalRecords,
		&job.ProcessedRecords,
		&job.FailedRecords,
		&job.RetryCount,
		&job.MaxRetries,
		&job.Status,
		&errorMessage,
		&startedAt,
		&completedAt,
		&createdAt,
		&updatedAt,
		&job.CreatedBy,
	); err != nil {
		return nil, err
	}

	job.CollectionName = collection.String
	job.ErrorMessage = errorMessage.String
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		job.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		job.UpdatedAt = t
	}
	if startedAt.Valid {
		if t, err := time.Parse(time.RFC3339, startedAt.String); err == nil {
			v := t
			job.StartedAt = &v
		}
	}
	if completedAt.Valid {
		if t, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
			v := t
			job.CompletedAt = &v
		}
	}

	return &job, nil
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
