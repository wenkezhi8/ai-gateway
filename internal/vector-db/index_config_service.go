package vectordb

import (
	"context"
	"fmt"
	"strings"
)

const (
	indexTypeHNSW = "hnsw"
	indexTypeIVF  = "ivf"
)

func (s *Service) GetIndexConfig(ctx context.Context, name string) (*IndexConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	collectionName := strings.TrimSpace(name)
	if collectionName == "" {
		return nil, fmt.Errorf("name is required")
	}

	collection, err := s.repo.Get(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	return mapCollectionToIndexConfig(collection), nil
}

func (s *Service) UpdateIndexConfig(ctx context.Context, name string, req *UpdateIndexConfigRequest) (*IndexConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	collectionName := strings.TrimSpace(name)
	if collectionName == "" {
		return nil, fmt.Errorf("name is required")
	}

	updateReq, err := buildIndexConfigUpdateRequest(req)
	if err != nil {
		return nil, err
	}
	updateErr := s.repo.Update(ctx, collectionName, updateReq)
	if updateErr != nil {
		return nil, updateErr
	}

	collection, err := s.repo.Get(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	return mapCollectionToIndexConfig(collection), nil
}

func buildIndexConfigUpdateRequest(req *UpdateIndexConfigRequest) (*UpdateCollectionRequest, error) {
	indexType := strings.ToLower(strings.TrimSpace(req.IndexType))
	if indexType != "" && indexType != indexTypeHNSW && indexType != indexTypeIVF {
		return nil, fmt.Errorf("index_type only allowed: hnsw, ivf")
	}
	if req.HNSWM != nil && *req.HNSWM <= 0 {
		return nil, fmt.Errorf("hnsw_m must be positive")
	}
	if req.HNSWEFConstruct != nil && *req.HNSWEFConstruct <= 0 {
		return nil, fmt.Errorf("hnsw_ef_construct must be positive")
	}
	if req.IVFNList != nil && *req.IVFNList <= 0 {
		return nil, fmt.Errorf("ivf_nlist must be positive")
	}

	updateReq := &UpdateCollectionRequest{
		HNSWM:           req.HNSWM,
		HNSWEFConstruct: req.HNSWEFConstruct,
		IVFNList:        req.IVFNList,
	}
	if indexType != "" {
		updateReq.IndexType = &indexType
	}
	return updateReq, nil
}

func mapCollectionToIndexConfig(collection *Collection) *IndexConfig {
	if collection == nil {
		return nil
	}
	return &IndexConfig{
		CollectionName:  collection.Name,
		IndexType:       collection.IndexType,
		HNSWM:           collection.HNSWM,
		HNSWEFConstruct: collection.HNSWEFConstruct,
		IVFNList:        collection.IVFNList,
		UpdatedAt:       collection.UpdatedAt,
	}
}
