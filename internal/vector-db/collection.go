package vectordb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	internalqdrant "ai-gateway/internal/qdrant"

	"github.com/sirupsen/logrus"
)

var (
	// ErrCollectionExists means collection name already exists.
	ErrCollectionExists = errors.New("collection already exists")
	// ErrCollectionNotFound means collection is missing.
	ErrCollectionNotFound = errors.New("collection not found")
)

// Service handles collection CRUD.
type Service struct {
	qdrantClient *internalqdrant.Client
	db           *sql.DB

	mu          sync.RWMutex
	collections map[string]*Collection
}

// NewService creates a mock vector db service.
func NewService() *Service {
	return &Service{
		qdrantClient: nil,
		db:           nil,
		collections:  make(map[string]*Collection),
	}
}

// CreateCollection creates a collection using in-memory storage.
func (s *Service) CreateCollection(_ context.Context, req *CreateCollectionRequest) (*Collection, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.collections[name]; exists {
		return nil, ErrCollectionExists
	}

	now := time.Now().UTC()
	collection := &Collection{
		ID:             fmt.Sprintf("col_%d", now.UnixNano()),
		Name:           name,
		Description:    strings.TrimSpace(req.Description),
		Dimension:      req.Dimension,
		DistanceMetric: defaultString(req.DistanceMetric, "cosine"),
		IndexType:      defaultString(req.IndexType, "hnsw"),
		StorageBackend: defaultString(req.StorageBackend, "memory"),
		Tags:           copyTags(req.Tags),
		Environment:    defaultString(req.Environment, "default"),
		Status:         defaultString(req.Status, "active"),
		VectorCount:    0,
		IndexedCount:   0,
		SizeBytes:      0,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      defaultString(req.CreatedBy, "system"),
		IsPublic:       req.IsPublic,
	}

	s.collections[name] = collection
	logrus.WithFields(logrus.Fields{"name": name, "env": collection.Environment}).Info("vector db collection created")

	result := *collection
	return &result, nil
}

// GetCollection gets one collection by name.
func (s *Service) GetCollection(_ context.Context, name string) (*Collection, error) {
	key := strings.TrimSpace(name)
	if key == "" {
		return nil, fmt.Errorf("name is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	col, ok := s.collections[key]
	if !ok {
		return nil, ErrCollectionNotFound
	}

	result := *col
	return &result, nil
}

// ListCollections lists collections with filters.
func (s *Service) ListCollections(_ context.Context, query *ListCollectionsQuery) ([]Collection, error) {
	if query == nil {
		query = &ListCollectionsQuery{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]Collection, 0, len(s.collections))
	for _, col := range s.collections {
		if !matchCollection(col, query) {
			continue
		}
		results = append(results, *col)
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(results) {
		return []Collection{}, nil
	}

	limit := query.Limit
	if limit <= 0 {
		limit = len(results)
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	page := make([]Collection, 0, end-offset)
	for idx := range results[offset:end] {
		copyCol := results[offset:end][idx]
		copyCol.Tags = copyTags(copyCol.Tags)
		page = append(page, copyCol)
	}

	return page, nil
}

// UpdateCollection updates mutable collection fields.
func (s *Service) UpdateCollection(_ context.Context, name string, req *UpdateCollectionRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	key := strings.TrimSpace(name)
	if key == "" {
		return fmt.Errorf("name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	col, ok := s.collections[key]
	if !ok {
		return ErrCollectionNotFound
	}

	if req.Description != nil {
		col.Description = strings.TrimSpace(*req.Description)
	}
	if req.DistanceMetric != nil {
		col.DistanceMetric = defaultString(*req.DistanceMetric, col.DistanceMetric)
	}
	if req.IndexType != nil {
		col.IndexType = defaultString(*req.IndexType, col.IndexType)
	}
	if req.StorageBackend != nil {
		col.StorageBackend = defaultString(*req.StorageBackend, col.StorageBackend)
	}
	if req.Tags != nil {
		col.Tags = copyTags(req.Tags)
	}
	if req.Environment != nil {
		col.Environment = defaultString(*req.Environment, col.Environment)
	}
	if req.Status != nil {
		col.Status = defaultString(*req.Status, col.Status)
	}
	if req.IsPublic != nil {
		col.IsPublic = *req.IsPublic
	}
	col.UpdatedAt = time.Now().UTC()

	logrus.WithField("name", key).Info("vector db collection updated")
	return nil
}

// DeleteCollection deletes one collection by name.
func (s *Service) DeleteCollection(_ context.Context, name string) error {
	key := strings.TrimSpace(name)
	if key == "" {
		return fmt.Errorf("name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.collections[key]; !ok {
		return ErrCollectionNotFound
	}

	delete(s.collections, key)
	logrus.WithField("name", key).Info("vector db collection deleted")
	return nil
}

// GetCollectionStats gets simple collection statistics.
func (s *Service) GetCollectionStats(ctx context.Context, name string) (*CollectionStats, error) {
	col, err := s.GetCollection(ctx, name)
	if err != nil {
		return nil, err
	}

	stats := &CollectionStats{
		Name:         col.Name,
		VectorCount:  col.VectorCount,
		IndexedCount: col.IndexedCount,
		SizeBytes:    col.SizeBytes,
	}
	return stats, nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func copyTags(tags []string) []string {
	if tags == nil {
		return nil
	}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		result = append(result, tag)
	}
	return result
}

//nolint:gocyclo // Filter combinations are intentionally explicit for readability.
func matchCollection(col *Collection, query *ListCollectionsQuery) bool {
	if col == nil || query == nil {
		return false
	}

	if query.Name != "" && col.Name != query.Name {
		return false
	}
	if query.Environment != "" && col.Environment != query.Environment {
		return false
	}
	if query.Status != "" && col.Status != query.Status {
		return false
	}
	if query.IsPublic != nil && col.IsPublic != *query.IsPublic {
		return false
	}
	if query.Tag != "" {
		matched := false
		for _, tag := range col.Tags {
			if tag == query.Tag {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if query.Search != "" {
		target := strings.ToLower(col.Name + " " + col.Description)
		if !strings.Contains(target, strings.ToLower(query.Search)) {
			return false
		}
	}
	return true
}
