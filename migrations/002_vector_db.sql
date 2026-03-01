-- Vector Database Schema
-- Collection metadata table
CREATE TABLE IF NOT EXISTS vector_collections (
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
	is_public INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_collections_name ON vector_collections(name);
CREATE INDEX IF NOT EXISTS idx_collections_status ON vector_collections(status);
CREATE INDEX IF NOT EXISTS idx_collections_environment ON vector_collections(environment);
CREATE INDEX IF NOT EXISTS idx_collections_created_by ON vector_collections(created_by);

-- Import jobs table
CREATE TABLE IF NOT EXISTS vector_import_jobs (
	id TEXT PRIMARY KEY,
	collection_id TEXT NOT NULL,
	file_name TEXT NOT NULL,
	file_path TEXT NOT NULL,
	file_size INTEGER NOT NULL,
	total_records INTEGER NOT NULL,
	processed_records INTEGER DEFAULT 0,
	failed_records INTEGER DEFAULT 0,
	retry_count INTEGER DEFAULT 0,
	max_retries INTEGER DEFAULT 3,
	status TEXT NOT NULL DEFAULT 'pending',
	error_message TEXT,
	started_at TEXT,
	completed_at TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	created_by TEXT NOT NULL,
	FOREIGN KEY (collection_id) REFERENCES vector_collections(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_collection ON vector_import_jobs(collection_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON vector_import_jobs(status);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created_by ON vector_import_jobs(created_by);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created_at ON vector_import_jobs(created_at);

-- Audit logs table
CREATE TABLE IF NOT EXISTS vector_audit_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT NOT NULL,
	action TEXT NOT NULL,
	resource_type TEXT NOT NULL,
	resource_id TEXT NOT NULL,
	details TEXT,
	ip_address TEXT,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON vector_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON vector_audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON vector_audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON vector_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON vector_audit_logs(created_at);

-- Alert rules table
CREATE TABLE IF NOT EXISTS vector_alert_rules (
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
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_metric ON vector_alert_rules(metric);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON vector_alert_rules(enabled);

-- Vector API keys table
CREATE TABLE IF NOT EXISTS vector_api_keys (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	key_hash TEXT NOT NULL UNIQUE,
	role TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_vector_api_keys_role ON vector_api_keys(role);
CREATE INDEX IF NOT EXISTS idx_vector_api_keys_enabled ON vector_api_keys(enabled);

-- Backup tasks table
CREATE TABLE IF NOT EXISTS vector_backup_tasks (
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
);

CREATE INDEX IF NOT EXISTS idx_backup_tasks_collection ON vector_backup_tasks(collection_name);
CREATE INDEX IF NOT EXISTS idx_backup_tasks_status ON vector_backup_tasks(status);
CREATE INDEX IF NOT EXISTS idx_backup_tasks_action ON vector_backup_tasks(action);
