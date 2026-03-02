package vectordb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// SearchVectors searches vectors by input vector.
func (s *Service) SearchVectors(ctx context.Context, req *SearchVectorsRequest) (*SearchVectorsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if s.backend == nil {
		return nil, ErrBackendUnavailable
	}

	collectionName := strings.TrimSpace(req.CollectionName)
	if collectionName == "" {
		return nil, fmt.Errorf("collection_name is required")
	}
	if req.TopK <= 0 {
		return nil, fmt.Errorf("top_k must be positive")
	}
	if len(req.Vector) == 0 && strings.TrimSpace(req.Text) != "" {
		if s.embedder == nil {
			return nil, fmt.Errorf("text embedder is required")
		}
		vector, err := s.embedder.Embed(ctx, req.Text)
		if err != nil {
			return nil, fmt.Errorf("embed text failed: %w", err)
		}
		req.Vector = vector
	}
	if len(req.Vector) == 0 {
		return nil, fmt.Errorf("vector is required")
	}

	if _, err := s.repo.Get(ctx, collectionName); err != nil {
		return nil, fmt.Errorf("get collection %s failed: %w", collectionName, err)
	}

	cacheKey := s.buildSearchCacheKey(req)
	if cached, ok := s.getSearchCache(cacheKey); ok {
		return cached, nil
	}

	items, err := s.backend.Search(ctx, collectionName, req.Vector, req.TopK, req.MinScore)
	if err != nil {
		return nil, fmt.Errorf("search vectors failed: %w", err)
	}
	if len(req.Filters) > 0 {
		items = applyPayloadFilters(items, req.Filters)
	}

	resp := &SearchVectorsResponse{Results: items, Total: len(items)}
	s.setSearchCache(cacheKey, resp)
	return resp, nil
}

func (s *Service) buildSearchCacheKey(req *SearchVectorsRequest) string {
	if req == nil {
		return ""
	}
	filterRaw, err := json.Marshal(req.Filters)
	if err != nil {
		filterRaw = []byte("{}")
	}
	return fmt.Sprintf("%s|%d|%f|%v|%s|%s", strings.TrimSpace(req.CollectionName), req.TopK, req.MinScore, req.Vector, strings.TrimSpace(req.Text), string(filterRaw))
}

func (s *Service) getSearchCache(key string) (*SearchVectorsResponse, bool) {
	if strings.TrimSpace(key) == "" {
		return nil, false
	}
	s.searchMu.RLock()
	defer s.searchMu.RUnlock()
	cached, ok := s.searchCache[key]
	if !ok {
		return nil, false
	}
	copyResp := SearchVectorsResponse{Total: cached.Total}
	if len(cached.Results) > 0 {
		copyResp.Results = append([]SearchResult{}, cached.Results...)
	}
	return &copyResp, true
}

func (s *Service) setSearchCache(key string, resp *SearchVectorsResponse) {
	if strings.TrimSpace(key) == "" || resp == nil {
		return
	}
	copyResp := SearchVectorsResponse{Total: resp.Total}
	if len(resp.Results) > 0 {
		copyResp.Results = append([]SearchResult{}, resp.Results...)
	}
	s.searchMu.Lock()
	s.searchCache[key] = copyResp
	s.searchMu.Unlock()
}

func applyPayloadFilters(items []SearchResult, filters map[string]any) []SearchResult {
	if len(items) == 0 || len(filters) == 0 {
		return items
	}
	filtered := make([]SearchResult, 0, len(items))
	for idx := range items {
		if payloadMatchesFilters(items[idx].Payload, filters) {
			filtered = append(filtered, items[idx])
		}
	}
	return filtered
}

func payloadMatchesFilters(payload, filters map[string]any) bool {
	if len(filters) == 0 {
		return true
	}
	if payload == nil {
		return false
	}
	for key, expected := range filters {
		actual, ok := payload[key]
		if !ok {
			return false
		}
		if normalizeFilterValue(actual) != normalizeFilterValue(expected) {
			return false
		}
	}
	return true
}

func normalizeFilterValue(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(strings.ToLower(value))
	case []byte:
		return strings.TrimSpace(strings.ToLower(string(value)))
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case bool:
		if value {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(strings.ToLower(fmt.Sprint(v)))
	}
}

// RecommendVectors recommends vectors based on input vector.
func (s *Service) RecommendVectors(ctx context.Context, req *RecommendVectorsRequest) (*SearchVectorsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	searchReq := (*SearchVectorsRequest)(req)
	return s.SearchVectors(ctx, searchReq)
}

// GetVectorByID gets one vector record by id.
func (s *Service) GetVectorByID(ctx context.Context, collectionName, id string) (*SearchResult, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if s.backend == nil {
		return nil, ErrBackendUnavailable
	}

	name := strings.TrimSpace(collectionName)
	if name == "" {
		return nil, fmt.Errorf("collection_name is required")
	}
	key := strings.TrimSpace(id)
	if key == "" {
		return nil, fmt.Errorf("id is required")
	}

	if _, err := s.repo.Get(ctx, name); err != nil {
		return nil, fmt.Errorf("get collection %s failed: %w", name, err)
	}

	item, err := s.backend.GetByID(ctx, name, key)
	if err != nil {
		return nil, fmt.Errorf("get vector by id failed: %w", err)
	}
	return item, nil
}
