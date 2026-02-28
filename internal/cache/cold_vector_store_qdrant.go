package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// QdrantColdVectorStoreConfig controls qdrant cold tier settings.
type QdrantColdVectorStoreConfig struct {
	URL        string
	APIKey     string
	Collection string
	Timeout    time.Duration
	Dimension  int
}

// QdrantColdVectorStore implements cold vector persistence on qdrant.
type QdrantColdVectorStore struct {
	cfg        QdrantColdVectorStoreConfig
	httpClient *http.Client
}

// NewQdrantColdVectorStore creates qdrant cold store client.
func NewQdrantColdVectorStore(cfg QdrantColdVectorStoreConfig) *QdrantColdVectorStore {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 1500 * time.Millisecond
	}
	if strings.TrimSpace(cfg.Collection) == "" {
		cfg.Collection = "ai_gateway_cold_vectors"
	}
	return &QdrantColdVectorStore{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// EnsureSchema ensures qdrant collection exists.
func (s *QdrantColdVectorStore) EnsureSchema(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if strings.TrimSpace(s.cfg.URL) == "" {
		return fmt.Errorf("qdrant url is empty")
	}
	if s.cfg.Dimension <= 0 {
		return fmt.Errorf("qdrant dimension must be positive")
	}

	getPath := fmt.Sprintf("/collections/%s", url.PathEscape(s.cfg.Collection))
	status, _, err := s.doJSON(ctx, http.MethodGet, getPath, nil)
	if err == nil && status >= 200 && status < 300 {
		return nil
	}

	payload := map[string]any{
		"vectors": map[string]any{
			"size":     s.cfg.Dimension,
			"distance": "Cosine",
		},
	}
	status, body, err := s.doJSON(ctx, http.MethodPut, getPath, payload)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("ensure qdrant schema failed: status=%d body=%s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

// Upsert writes one cold vector point to qdrant.
func (s *QdrantColdVectorStore) Upsert(ctx context.Context, doc *VectorCacheDocument) error {
	if s == nil || doc == nil {
		return nil
	}
	if strings.TrimSpace(s.cfg.URL) == "" {
		return fmt.Errorf("qdrant url is empty")
	}

	payload := map[string]any{
		"points": []map[string]any{
			{
				"id":     doc.CacheKey,
				"vector": doc.Vector,
				"payload": map[string]any{
					"cache_key":        doc.CacheKey,
					"intent":           doc.Intent,
					"task_type":        doc.TaskType,
					"slots":            doc.Slots,
					"normalized_query": doc.NormalizedQuery,
					"response":         doc.Response,
					"provider":         doc.Provider,
					"model":            doc.Model,
					"quality_score":    doc.QualityScore,
					"create_ts":        doc.CreateTS,
					"last_hit_ts":      doc.LastHitTS,
					"expire_ts":        doc.ExpireTS,
					"ttl_sec":          doc.TTLSec,
					"tier":             doc.Tier,
					"migrate_ts":       doc.MigrateTS,
				},
			},
		},
	}

	path := fmt.Sprintf("/collections/%s/points?wait=false", url.PathEscape(s.cfg.Collection))
	status, body, err := s.doJSON(ctx, http.MethodPut, path, payload)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("qdrant upsert failed: status=%d body=%s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

// VectorSearch performs qdrant similarity search.
func (s *QdrantColdVectorStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]VectorSearchHit, error) {
	if s == nil || len(vector) == 0 {
		return []VectorSearchHit{}, nil
	}
	if strings.TrimSpace(s.cfg.URL) == "" {
		return nil, fmt.Errorf("qdrant url is empty")
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

	payload := map[string]any{
		"vector":          vector,
		"limit":           topK,
		"score_threshold": minSimilarity,
		"with_payload":    true,
		"with_vector":     false,
	}
	if strings.TrimSpace(intent) != "" {
		payload["filter"] = map[string]any{
			"must": []map[string]any{
				{
					"key": "intent",
					"match": map[string]any{
						"value": intent,
					},
				},
			},
		}
	}

	path := fmt.Sprintf("/collections/%s/points/search", url.PathEscape(s.cfg.Collection))
	status, body, err := s.doJSON(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("qdrant vector search failed: status=%d body=%s", status, strings.TrimSpace(string(body)))
	}

	var resp struct {
		Result []struct {
			ID      any            `json:"id"`
			Score   float64        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode qdrant vector search response: %w", err)
	}

	hits := make([]VectorSearchHit, 0, len(resp.Result))
	for _, item := range resp.Result {
		cacheKey := stringifyAny(item.ID)
		if payloadKey := stringifyAny(item.Payload["cache_key"]); payloadKey != "" {
			cacheKey = payloadKey
		}
		hitIntent := stringifyAny(item.Payload["intent"])
		responseRaw, _ := json.Marshal(item.Payload["response"])
		if len(responseRaw) == 0 {
			responseRaw = []byte("null")
		}
		score := 1 - item.Score
		if score < 0 {
			score = 0
		}
		hits = append(hits, VectorSearchHit{
			CacheKey:   cacheKey,
			Intent:     hitIntent,
			Score:      score,
			Similarity: item.Score,
			Response:   json.RawMessage(responseRaw),
		})
	}
	return hits, nil
}

// GetExact retrieves one point by cache key.
func (s *QdrantColdVectorStore) GetExact(ctx context.Context, cacheKey string) (*VectorCacheDocument, error) {
	if s == nil {
		return nil, nil
	}
	if strings.TrimSpace(s.cfg.URL) == "" {
		return nil, fmt.Errorf("qdrant url is empty")
	}

	path := fmt.Sprintf(
		"/collections/%s/points/%s?with_payload=true&with_vector=true",
		url.PathEscape(s.cfg.Collection),
		url.PathEscape(cacheKey),
	)
	status, body, err := s.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("qdrant get exact failed: status=%d body=%s", status, strings.TrimSpace(string(body)))
	}

	var resp struct {
		Result struct {
			ID      any            `json:"id"`
			Vector  []float64      `json:"vector"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode qdrant get exact response: %w", err)
	}
	if resp.Result.Payload == nil && len(resp.Result.Vector) == 0 {
		return nil, nil
	}

	doc := &VectorCacheDocument{
		CacheKey:        stringifyAny(resp.Result.Payload["cache_key"]),
		Intent:          stringifyAny(resp.Result.Payload["intent"]),
		TaskType:        stringifyAny(resp.Result.Payload["task_type"]),
		NormalizedQuery: stringifyAny(resp.Result.Payload["normalized_query"]),
		Vector:          resp.Result.Vector,
		Provider:        stringifyAny(resp.Result.Payload["provider"]),
		Model:           stringifyAny(resp.Result.Payload["model"]),
		QualityScore:    toFloat(resp.Result.Payload["quality_score"]),
		CreateTS:        int64(toFloat(resp.Result.Payload["create_ts"])),
		LastHitTS:       int64(toFloat(resp.Result.Payload["last_hit_ts"])),
		ExpireTS:        int64(toFloat(resp.Result.Payload["expire_ts"])),
		TTLSec:          int64(toFloat(resp.Result.Payload["ttl_sec"])),
		Tier:            stringifyAny(resp.Result.Payload["tier"]),
		MigrateTS:       int64(toFloat(resp.Result.Payload["migrate_ts"])),
	}
	if doc.CacheKey == "" {
		doc.CacheKey = stringifyAny(resp.Result.ID)
	}
	if slots, ok := resp.Result.Payload["slots"].(map[string]any); ok {
		doc.Slots = make(map[string]string, len(slots))
		for k, v := range slots {
			doc.Slots[k] = stringifyAny(v)
		}
	}
	doc.Response = resp.Result.Payload["response"]
	return doc, nil
}

// Delete removes one point from qdrant.
func (s *QdrantColdVectorStore) Delete(ctx context.Context, cacheKey string) error {
	if s == nil {
		return nil
	}
	if strings.TrimSpace(s.cfg.URL) == "" {
		return fmt.Errorf("qdrant url is empty")
	}

	path := fmt.Sprintf("/collections/%s/points/delete?wait=false", url.PathEscape(s.cfg.Collection))
	status, body, err := s.doJSON(ctx, http.MethodPost, path, map[string]any{
		"points": []string{cacheKey},
	})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("qdrant delete failed: status=%d body=%s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

// Stats returns qdrant collection stats.
func (s *QdrantColdVectorStore) Stats(ctx context.Context) (ColdVectorStoreStats, error) {
	stats := ColdVectorStoreStats{
		Backend:    ColdVectorBackendQdrant,
		Collection: s.cfg.Collection,
		Available:  s != nil && strings.TrimSpace(s.cfg.URL) != "",
	}
	if s == nil || strings.TrimSpace(s.cfg.URL) == "" {
		return stats, nil
	}

	path := fmt.Sprintf("/collections/%s", url.PathEscape(s.cfg.Collection))
	status, body, err := s.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return stats, err
	}
	if status < 200 || status >= 300 {
		return stats, fmt.Errorf("qdrant stats failed: status=%d body=%s", status, strings.TrimSpace(string(body)))
	}

	var resp struct {
		Result struct {
			PointsCount  int64 `json:"points_count"`
			VectorsCount int64 `json:"vectors_count"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return stats, err
	}
	if resp.Result.PointsCount > 0 {
		stats.Entries = resp.Result.PointsCount
	} else {
		stats.Entries = resp.Result.VectorsCount
	}
	return stats, nil
}

func (s *QdrantColdVectorStore) doJSON(ctx context.Context, method, path string, body any) (int, []byte, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(s.cfg.URL), "/") + path
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(s.cfg.APIKey); key != "" {
		req.Header.Set("api-key", key)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, payload, nil
}

func stringifyAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return fmt.Sprintf("%.0f", x)
	case float32:
		return fmt.Sprintf("%.0f", x)
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", x))
	}
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	default:
		return 0
	}
}
