package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteColdVectorStoreConfig configures sqlite cold vector storage.
type SQLiteColdVectorStoreConfig struct {
	Path string
}

// SQLiteColdVectorStore persists cold vectors in sqlite.
type SQLiteColdVectorStore struct {
	db   *sql.DB
	path string
}

// NewSQLiteColdVectorStore creates a sqlite cold vector store.
func NewSQLiteColdVectorStore(cfg SQLiteColdVectorStoreConfig) (*SQLiteColdVectorStore, error) {
	path := strings.TrimSpace(cfg.Path)
	if path == "" {
		path = "data/ai-gateway-cold-vectors.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite cold vector dir: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite cold vector store: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	store := &SQLiteColdVectorStore{db: db, path: path}
	if err := store.EnsureSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// EnsureSchema ensures sqlite schema exists.
func (s *SQLiteColdVectorStore) EnsureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	queries := []string{
		`CREATE TABLE IF NOT EXISTS cold_vectors (
			cache_key TEXT PRIMARY KEY,
			intent TEXT NOT NULL,
			task_type TEXT,
			slots_json TEXT,
			normalized_query TEXT,
			vector_json TEXT NOT NULL,
			response_json TEXT,
			provider TEXT,
			model TEXT,
			quality_score REAL DEFAULT 0,
			create_ts INTEGER DEFAULT 0,
			last_hit_ts INTEGER DEFAULT 0,
			expire_ts INTEGER DEFAULT 0,
			ttl_sec INTEGER DEFAULT 0,
			tier TEXT DEFAULT 'cold',
			migrate_ts INTEGER DEFAULT 0,
			updated_ts INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cold_vectors_intent ON cold_vectors(intent)`,
		`CREATE INDEX IF NOT EXISTS idx_cold_vectors_last_hit_ts ON cold_vectors(last_hit_ts, create_ts)`,
	}

	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("ensure sqlite cold vector schema: %w", err)
		}
	}
	return nil
}

// Upsert writes one cold vector document.
func (s *SQLiteColdVectorStore) Upsert(ctx context.Context, doc *VectorCacheDocument) error {
	if s == nil || s.db == nil || doc == nil {
		return nil
	}
	if strings.TrimSpace(doc.CacheKey) == "" {
		return fmt.Errorf("sqlite cold upsert: empty cache_key")
	}

	now := time.Now().Unix()
	if doc.CreateTS <= 0 {
		doc.CreateTS = now
	}
	if doc.LastHitTS <= 0 {
		doc.LastHitTS = doc.CreateTS
	}
	if strings.TrimSpace(doc.Tier) == "" {
		doc.Tier = VectorTierCold
	}

	slotsRaw, err := json.Marshal(doc.Slots)
	if err != nil {
		return fmt.Errorf("sqlite cold upsert marshal slots: %w", err)
	}
	vectorRaw, err := json.Marshal(doc.Vector)
	if err != nil {
		return fmt.Errorf("sqlite cold upsert marshal vector: %w", err)
	}
	responseRaw, err := json.Marshal(doc.Response)
	if err != nil {
		return fmt.Errorf("sqlite cold upsert marshal response: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO cold_vectors (
			cache_key, intent, task_type, slots_json, normalized_query, vector_json, response_json, provider, model,
			quality_score, create_ts, last_hit_ts, expire_ts, ttl_sec, tier, migrate_ts, updated_ts
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cache_key) DO UPDATE SET
			intent=excluded.intent,
			task_type=excluded.task_type,
			slots_json=excluded.slots_json,
			normalized_query=excluded.normalized_query,
			vector_json=excluded.vector_json,
			response_json=excluded.response_json,
			provider=excluded.provider,
			model=excluded.model,
			quality_score=excluded.quality_score,
			create_ts=excluded.create_ts,
			last_hit_ts=excluded.last_hit_ts,
			expire_ts=excluded.expire_ts,
			ttl_sec=excluded.ttl_sec,
			tier=excluded.tier,
			migrate_ts=excluded.migrate_ts,
			updated_ts=excluded.updated_ts`,
		doc.CacheKey, doc.Intent, doc.TaskType, string(slotsRaw), doc.NormalizedQuery, string(vectorRaw),
		string(responseRaw), doc.Provider, doc.Model, doc.QualityScore, doc.CreateTS, doc.LastHitTS, doc.ExpireTS,
		doc.TTLSec, doc.Tier, doc.MigrateTS, now,
	)
	if err != nil {
		return fmt.Errorf("sqlite cold upsert exec: %w", err)
	}
	return nil
}

// VectorSearch performs brute-force cosine similarity search over cold vectors.
func (s *SQLiteColdVectorStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]VectorSearchHit, error) {
	if s == nil || s.db == nil || len(vector) == 0 {
		return []VectorSearchHit{}, nil
	}
	if topK <= 0 {
		topK = 1
	}
	if minSimilarity < 0 {
		minSimilarity = 0
	}
	if minSimilarity > 1 {
		minSimilarity = 1
	}

	query := `SELECT cache_key, intent, vector_json, response_json FROM cold_vectors`
	args := []any{}
	if strings.TrimSpace(intent) != "" {
		query += ` WHERE intent = ?`
		args = append(args, intent)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite cold vector search query: %w", err)
	}
	defer rows.Close()

	hits := make([]VectorSearchHit, 0, topK)
	for rows.Next() {
		var cacheKey, hitIntent, vectorJSON, responseJSON string
		if err := rows.Scan(&cacheKey, &hitIntent, &vectorJSON, &responseJSON); err != nil {
			return nil, fmt.Errorf("sqlite cold vector search scan: %w", err)
		}

		var candidate []float64
		if err := json.Unmarshal([]byte(vectorJSON), &candidate); err != nil {
			continue
		}
		similarity := cosineSimilarity(vector, candidate)
		if similarity < minSimilarity {
			continue
		}
		score := 1 - similarity
		if score < 0 {
			score = 0
		}
		if score > 1 {
			score = 1
		}
		hits = append(hits, VectorSearchHit{
			CacheKey:   cacheKey,
			Intent:     hitIntent,
			Score:      score,
			Similarity: similarity,
			Response:   json.RawMessage(responseJSON),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite cold vector search rows: %w", err)
	}

	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Similarity > hits[j].Similarity
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

// GetExact returns one cold vector document by cache key.
func (s *SQLiteColdVectorStore) GetExact(ctx context.Context, cacheKey string) (*VectorCacheDocument, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	row := s.db.QueryRowContext(
		ctx,
		`SELECT cache_key, intent, task_type, slots_json, normalized_query, vector_json, response_json, provider, model,
			quality_score, create_ts, last_hit_ts, expire_ts, ttl_sec, tier, migrate_ts
		FROM cold_vectors WHERE cache_key = ?`,
		cacheKey,
	)
	var (
		doc          VectorCacheDocument
		slotsJSON    string
		vectorJSON   string
		responseJSON string
	)
	err := row.Scan(
		&doc.CacheKey, &doc.Intent, &doc.TaskType, &slotsJSON, &doc.NormalizedQuery, &vectorJSON, &responseJSON,
		&doc.Provider, &doc.Model, &doc.QualityScore, &doc.CreateTS, &doc.LastHitTS, &doc.ExpireTS, &doc.TTLSec,
		&doc.Tier, &doc.MigrateTS,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("sqlite cold get exact: %w", err)
	}

	if strings.TrimSpace(slotsJSON) != "" {
		_ = json.Unmarshal([]byte(slotsJSON), &doc.Slots)
	}
	if strings.TrimSpace(vectorJSON) != "" {
		_ = json.Unmarshal([]byte(vectorJSON), &doc.Vector)
	}
	if strings.TrimSpace(responseJSON) != "" {
		var anyResp any
		if err := json.Unmarshal([]byte(responseJSON), &anyResp); err == nil {
			doc.Response = anyResp
		} else {
			doc.Response = json.RawMessage(responseJSON)
		}
	}
	return &doc, nil
}

// Delete removes one cold vector document.
func (s *SQLiteColdVectorStore) Delete(ctx context.Context, cacheKey string) error {
	if s == nil || s.db == nil {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM cold_vectors WHERE cache_key = ?`, cacheKey); err != nil {
		return fmt.Errorf("sqlite cold delete: %w", err)
	}
	return nil
}

// Stats returns sqlite cold vector stats.
func (s *SQLiteColdVectorStore) Stats(ctx context.Context) (ColdVectorStoreStats, error) {
	stats := ColdVectorStoreStats{
		Backend:    ColdVectorBackendSQLite,
		Available:  s != nil && s.db != nil,
		Collection: s.path,
	}
	if s == nil || s.db == nil {
		return stats, nil
	}
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM cold_vectors`).Scan(&count); err != nil {
		return stats, err
	}
	stats.Entries = count
	return stats, nil
}

func cosineSimilarity(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	if sim < 0 {
		return 0
	}
	if sim > 1 {
		return 1
	}
	return sim
}
