package cache

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStackExecutor interface {
	Do(ctx context.Context, args ...any) *redis.Cmd
}

// RedisStackVectorStore implements VectorCacheStore using RedisJSON + RediSearch.
type RedisStackVectorStore struct {
	exec redisStackExecutor
	cfg  RedisStackVectorConfig
}

// NewRedisStackVectorStoreFromRedisCache creates a vector store from existing Redis cache.
func NewRedisStackVectorStoreFromRedisCache(rc *RedisCache, cfg RedisStackVectorConfig) *RedisStackVectorStore {
	if rc == nil {
		return nil
	}
	return NewRedisStackVectorStoreWithExecutor(rc.GetClient(), cfg)
}

// NewRedisStackVectorStoreWithExecutor is used by tests and production.
func NewRedisStackVectorStoreWithExecutor(exec redisStackExecutor, cfg RedisStackVectorConfig) *RedisStackVectorStore {
	if cfg.IndexName == "" || cfg.KeyPrefix == "" || cfg.Dimension <= 0 || cfg.QueryTimeout <= 0 {
		def := DefaultRedisStackVectorConfig()
		if cfg.IndexName == "" {
			cfg.IndexName = def.IndexName
		}
		if cfg.KeyPrefix == "" {
			cfg.KeyPrefix = def.KeyPrefix
		}
		if cfg.Dimension <= 0 {
			cfg.Dimension = def.Dimension
		}
		if cfg.QueryTimeout <= 0 {
			cfg.QueryTimeout = def.QueryTimeout
		}
	}
	return &RedisStackVectorStore{exec: exec, cfg: cfg}
}

func (s *RedisStackVectorStore) EnsureIndex(ctx context.Context) error {
	if s == nil || s.exec == nil || !s.cfg.Enabled {
		return nil
	}

	err := s.exec.Do(ctx,
		"FT.CREATE", s.cfg.IndexName,
		"ON", "JSON",
		"PREFIX", "1", s.cfg.KeyPrefix,
		"SCHEMA",
		"$.cache_key", "AS", "cache_key", "TEXT", "SORTABLE",
		"$.intent", "AS", "intent", "TAG", "SORTABLE",
		"$.task_type", "AS", "task_type", "TAG", "SORTABLE",
		"$.create_ts", "AS", "create_ts", "NUMERIC", "SORTABLE",
		"$.last_hit_ts", "AS", "last_hit_ts", "NUMERIC", "SORTABLE",
		"$.expire_ts", "AS", "expire_ts", "NUMERIC", "SORTABLE",
		"$.vector", "AS", "vector", "VECTOR", "HNSW", "12",
		"TYPE", "FLOAT32",
		"DIM", strconv.Itoa(s.cfg.Dimension),
		"DISTANCE_METRIC", "COSINE",
		"M", "16",
		"EF_CONSTRUCTION", "200",
		"EF_RUNTIME", "64",
	).Err()
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "index already exists") || strings.Contains(msg, "index name is already in use") {
		return nil
	}
	return fmt.Errorf("ensure vector index: %w", err)
}

func (s *RedisStackVectorStore) RebuildIndex(ctx context.Context) error {
	if s == nil || s.exec == nil || !s.cfg.Enabled {
		return nil
	}
	dropErr := s.exec.Do(ctx, "FT.DROPINDEX", s.cfg.IndexName).Err()
	if dropErr != nil {
		msg := strings.ToLower(dropErr.Error())
		if !strings.Contains(msg, "unknown index name") {
			return fmt.Errorf("drop vector index: %w", dropErr)
		}
	}
	return s.EnsureIndex(ctx)
}

func (s *RedisStackVectorStore) GetExact(ctx context.Context, cacheKey string) (*VectorCacheDocument, error) {
	if s == nil || s.exec == nil || !s.cfg.Enabled {
		return nil, nil
	}
	fullKey := s.fullKey(cacheKey)
	raw, err := s.exec.Do(ctx, "JSON.GET", fullKey, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("json get exact: %w", err)
	}

	payload, ok := raw.(string)
	if !ok || strings.TrimSpace(payload) == "" {
		return nil, nil
	}

	// Redis JSON.GET with path "$" returns array-wrapped payload.
	var wrapped []VectorCacheDocument
	if err := json.Unmarshal([]byte(payload), &wrapped); err == nil && len(wrapped) > 0 {
		return &wrapped[0], nil
	}

	var doc VectorCacheDocument
	if err := json.Unmarshal([]byte(payload), &doc); err != nil {
		return nil, fmt.Errorf("decode exact cache doc: %w", err)
	}
	return &doc, nil
}

func (s *RedisStackVectorStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]VectorSearchHit, error) {
	if s == nil || s.exec == nil || !s.cfg.Enabled || len(vector) == 0 {
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

	searchCtx := ctx
	cancel := func() {}
	if s.cfg.QueryTimeout > 0 {
		searchCtx, cancel = context.WithTimeout(ctx, s.cfg.QueryTimeout)
	}
	defer cancel()

	blob, err := vectorToFloat32Blob(vector)
	if err != nil {
		return nil, err
	}

	baseQuery := fmt.Sprintf("*=>[KNN %d @vector $BLOB AS vector_score]", topK)
	if strings.TrimSpace(intent) != "" {
		baseQuery = fmt.Sprintf("@intent:{%s}=>[KNN %d @vector $BLOB AS vector_score]", escapeTagValue(intent), topK)
	}

	raw, err := s.exec.Do(
		searchCtx,
		"FT.SEARCH", s.cfg.IndexName, baseQuery,
		"PARAMS", "2", "BLOB", string(blob),
		"SORTBY", "vector_score", "ASC",
		"RETURN", "4", "$.cache_key", "$.intent", "$.response", "vector_score",
		"DIALECT", "2",
	).Result()
	if err != nil {
		if err == redis.Nil {
			return []VectorSearchHit{}, nil
		}
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	items, ok := raw.([]any)
	if !ok || len(items) <= 1 {
		return []VectorSearchHit{}, nil
	}

	results := make([]VectorSearchHit, 0, topK)
	for i := 1; i+1 < len(items); i += 2 {
		redisKey, _ := items[i].(string)
		fields, _ := items[i+1].([]any)
		fieldMap := parseSearchFieldMap(fields)

		score := parseFloatValue(fieldMap["vector_score"])
		score = clampFloat64(score, 0, 1)
		similarity := 1 - score
		if similarity < minSimilarity {
			continue
		}

		cacheKey := normalizeFieldValue(fieldMap["$.cache_key"])
		hitIntent := normalizeFieldValue(fieldMap["$.intent"])
		respRaw := normalizeFieldValue(fieldMap["$.response"])

		results = append(results, VectorSearchHit{
			RedisKey:   redisKey,
			CacheKey:   cacheKey,
			Intent:     hitIntent,
			Score:      score,
			Similarity: similarity,
			Response:   json.RawMessage(respRaw),
		})
	}

	return results, nil
}

func (s *RedisStackVectorStore) Upsert(ctx context.Context, doc *VectorCacheDocument) error {
	if s == nil || s.exec == nil || !s.cfg.Enabled || doc == nil {
		return nil
	}
	if strings.TrimSpace(doc.CacheKey) == "" {
		return fmt.Errorf("upsert vector cache doc: empty cache_key")
	}

	now := time.Now().Unix()
	if doc.CreateTS <= 0 {
		doc.CreateTS = now
	}
	if doc.LastHitTS <= 0 {
		doc.LastHitTS = doc.CreateTS
	}
	if doc.TTLSec <= 0 {
		doc.TTLSec = int64((24 * time.Hour).Seconds())
	}
	if doc.ExpireTS <= 0 {
		doc.ExpireTS = doc.CreateTS + doc.TTLSec
	}
	if doc.Slots == nil {
		doc.Slots = map[string]string{}
	}
	if strings.TrimSpace(doc.Tier) == "" {
		doc.Tier = VectorTierHot
	}

	payload, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal vector cache doc: %w", err)
	}

	key := s.fullKey(doc.CacheKey)
	if err := s.exec.Do(ctx, "JSON.SET", key, "$", string(payload)).Err(); err != nil {
		return fmt.Errorf("json set vector cache doc: %w", err)
	}
	if err := s.exec.Do(ctx, "EXPIRE", key, strconv.FormatInt(doc.TTLSec, 10)).Err(); err != nil {
		return fmt.Errorf("expire vector cache doc: %w", err)
	}
	return nil
}

func (s *RedisStackVectorStore) Delete(ctx context.Context, cacheKey string) error {
	if s == nil || s.exec == nil || !s.cfg.Enabled {
		return nil
	}
	return s.exec.Do(ctx, "DEL", s.fullKey(cacheKey)).Err()
}

func (s *RedisStackVectorStore) TouchTTL(ctx context.Context, cacheKey string, ttlSec int64) error {
	if s == nil || s.exec == nil || !s.cfg.Enabled || ttlSec <= 0 {
		return nil
	}
	key := s.fullKey(cacheKey)
	if err := s.exec.Do(ctx, "EXPIRE", key, strconv.FormatInt(ttlSec, 10)).Err(); err != nil {
		return err
	}
	_ = s.exec.Do(ctx, "JSON.SET", key, "$.last_hit_ts", strconv.FormatInt(time.Now().Unix(), 10)).Err()
	return nil
}

func (s *RedisStackVectorStore) Stats(ctx context.Context) (VectorStoreStats, error) {
	stats := VectorStoreStats{
		Enabled:      s != nil && s.cfg.Enabled,
		IndexName:    s.cfg.IndexName,
		KeyPrefix:    s.cfg.KeyPrefix,
		Dimension:    s.cfg.Dimension,
		QueryTimeout: s.cfg.QueryTimeout.Milliseconds(),
	}
	if s == nil || s.exec == nil || !s.cfg.Enabled {
		return stats, nil
	}

	// Probe index availability without failing the caller on missing index.
	if err := s.exec.Do(ctx, "FT.INFO", s.cfg.IndexName).Err(); err != nil {
		return stats, nil
	}
	return stats, nil
}

func (s *RedisStackVectorStore) fullKey(cacheKey string) string {
	return strings.TrimRight(s.cfg.KeyPrefix, ":") + ":" + strings.TrimLeft(cacheKey, ":")
}

// MemoryUsagePercent returns Redis memory usage percentage by parsing INFO memory.
func (s *RedisStackVectorStore) MemoryUsagePercent(ctx context.Context) (float64, error) {
	if s == nil || s.exec == nil || !s.cfg.Enabled {
		return 0, nil
	}
	infoRaw, err := s.exec.Do(ctx, "INFO", "memory").Result()
	if err != nil {
		return 0, fmt.Errorf("redis info memory: %w", err)
	}
	info := normalizeFieldValue(infoRaw)
	var used, max float64
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "used_memory:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "used_memory:"))
			used, _ = strconv.ParseFloat(v, 64)
			continue
		}
		if strings.HasPrefix(line, "maxmemory:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "maxmemory:"))
			max, _ = strconv.ParseFloat(v, 64)
		}
	}
	if max <= 0 {
		return 0, nil
	}
	return (used / max) * 100, nil
}

// ListMigrationCandidates scans hot keys and returns least-active docs first.
func (s *RedisStackVectorStore) ListMigrationCandidates(ctx context.Context, batchSize int) ([]*VectorCacheDocument, error) {
	if s == nil || s.exec == nil || !s.cfg.Enabled {
		return nil, nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	pattern := strings.TrimRight(s.cfg.KeyPrefix, ":") + ":*"
	var (
		cursor = "0"
		keys   []string
	)
	for {
		raw, err := s.exec.Do(ctx, "SCAN", cursor, "MATCH", pattern, "COUNT", strconv.Itoa(batchSize*2)).Result()
		if err != nil {
			return nil, fmt.Errorf("scan vector keys: %w", err)
		}
		items, ok := raw.([]any)
		if !ok || len(items) != 2 {
			break
		}
		cursor = normalizeFieldValue(items[0])
		keyItems, _ := items[1].([]any)
		for _, item := range keyItems {
			key := normalizeFieldValue(item)
			if key != "" {
				keys = append(keys, key)
			}
		}
		if cursor == "0" || len(keys) >= batchSize*4 {
			break
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}

	docs := make([]*VectorCacheDocument, 0, len(keys))
	for _, key := range keys {
		raw, err := s.exec.Do(ctx, "JSON.GET", key, "$").Result()
		if err != nil {
			continue
		}
		payload := normalizeFieldValue(raw)
		if strings.TrimSpace(payload) == "" {
			continue
		}
		var wrapped []VectorCacheDocument
		if err := json.Unmarshal([]byte(payload), &wrapped); err == nil && len(wrapped) > 0 {
			d := wrapped[0]
			if strings.TrimSpace(d.CacheKey) != "" && len(d.Vector) > 0 {
				docs = append(docs, &d)
			}
			continue
		}

		var doc VectorCacheDocument
		if err := json.Unmarshal([]byte(payload), &doc); err != nil {
			continue
		}
		if strings.TrimSpace(doc.CacheKey) == "" || len(doc.Vector) == 0 {
			continue
		}
		docs = append(docs, &doc)
	}
	if len(docs) == 0 {
		return nil, nil
	}

	sort.Slice(docs, func(i, j int) bool {
		left := docs[i].LastHitTS
		right := docs[j].LastHitTS
		if left <= 0 {
			left = docs[i].CreateTS
		}
		if right <= 0 {
			right = docs[j].CreateTS
		}
		return left < right
	})
	if len(docs) > batchSize {
		docs = docs[:batchSize]
	}
	return docs, nil
}

func vectorToFloat32Blob(vec []float64) ([]byte, error) {
	if len(vec) == 0 {
		return nil, fmt.Errorf("empty vector")
	}
	blob := make([]byte, len(vec)*4)
	for i, v := range vec {
		f32 := float32(v)
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid vector value at %d", i)
		}
		binary.LittleEndian.PutUint32(blob[i*4:], math.Float32bits(f32))
	}
	return blob, nil
}

func parseSearchFieldMap(fields []any) map[string]any {
	result := make(map[string]any, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		key, _ := fields[i].(string)
		result[key] = fields[i+1]
	}
	return result
}

func normalizeFieldValue(v any) string {
	switch x := v.(type) {
	case string:
		return unwrapJSONPathValue(x)
	case []byte:
		return unwrapJSONPathValue(string(x))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", x))
	}
}

func unwrapJSONPathValue(s string) string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return raw
	}
	// RediSearch + JSON may return ["value"] for JSONPath fields.
	if strings.HasPrefix(raw, "[") {
		var arr []json.RawMessage
		if err := json.Unmarshal([]byte(raw), &arr); err == nil && len(arr) > 0 {
			var str string
			if err := json.Unmarshal(arr[0], &str); err == nil {
				return str
			}
			return strings.TrimSpace(string(arr[0]))
		}
	}
	return raw
}

func parseFloatValue(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f
	default:
		f, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprintf("%v", x)), 64)
		return f
	}
}

func clampFloat64(v float64, min float64, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func escapeTagValue(v string) string {
	replacer := strings.NewReplacer(
		"-", `\-`,
		"{", `\{`,
		"}", `\}`,
		"|", `\|`,
		" ", `\ `,
	)
	return replacer.Replace(strings.TrimSpace(v))
}
