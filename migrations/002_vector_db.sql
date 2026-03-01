-- Vector Database Schema
-- Collection metadata table
CREATE TABLE IF NOT EXISTS vector_collections (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT,
	dimension INTEGER NOT NULL,
	distance_metric TEXT NOT NULL DEFAULT 'cosine',
	index_type TEXT NOT NULL DEFAULT 'hnsw',
	storage_backend TEXT NOT NULL DEFAULT 'sqlite',
	tags TEXT,
	environment TEXT NOT NULL DEFAULT 'production',
	status TEXT NOT NULL DEFAULT 'active',
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
