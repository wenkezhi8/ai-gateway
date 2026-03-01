package vectordb

import (
	"context"
	"fmt"
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
	if strings.TrimSpace(req.Text) != "" {
		return nil, ErrTextSearchNotSupported
	}
	if len(req.Vector) == 0 {
		return nil, fmt.Errorf("vector is required")
	}

	if _, err := s.repo.Get(ctx, collectionName); err != nil {
		return nil, fmt.Errorf("get collection %s failed: %w", collectionName, err)
	}

	items, err := s.backend.Search(ctx, collectionName, req.Vector, req.TopK, req.MinScore)
	if err != nil {
		return nil, fmt.Errorf("search vectors failed: %w", err)
	}

	resp := &SearchVectorsResponse{Results: items, Total: len(items)}
	return resp, nil
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
