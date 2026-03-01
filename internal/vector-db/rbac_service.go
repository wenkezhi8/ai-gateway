package vectordb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type RBACService struct {
	repo CollectionRepository
}

func NewRBACService(repo CollectionRepository) *RBACService {
	return &RBACService{repo: repo}
}

func (s *RBACService) CreateAPIKey(ctx context.Context, rawKey, role string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("repository is required")
	}
	key := strings.TrimSpace(rawKey)
	if key == "" {
		return fmt.Errorf("api key is required")
	}
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	if normalizedRole == "" {
		normalizedRole = "reader"
	}
	if _, ok := permissionSetByRole[normalizedRole]; !ok {
		return fmt.Errorf("role is invalid")
	}

	now := time.Now().UTC()
	return s.repo.CreateVectorAPIKey(ctx, &VectorAPIKey{
		KeyHash:   hashAPIKey(key),
		Role:      normalizedRole,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *RBACService) CheckPermission(ctx context.Context, rawKey string, permission VectorPermission) (bool, error) {
	if s == nil || s.repo == nil {
		return false, fmt.Errorf("repository is required")
	}
	key := strings.TrimSpace(rawKey)
	if key == "" {
		return false, nil
	}

	apiKey, err := s.repo.GetVectorAPIKeyByHash(ctx, hashAPIKey(key))
	if err != nil {
		if err == ErrVectorAPIKeyNotFound {
			return false, nil
		}
		return false, err
	}
	if !apiKey.Enabled {
		return false, nil
	}

	permissions, ok := permissionSetByRole[strings.ToLower(strings.TrimSpace(apiKey.Role))]
	if !ok {
		return false, nil
	}
	_, allowed := permissions[permission]
	return allowed, nil
}

func (s *RBACService) ListAPIKeys(ctx context.Context) ([]VectorAPIKey, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	return s.repo.ListVectorAPIKeys(ctx)
}

func (s *RBACService) DeleteAPIKey(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("repository is required")
	}
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	return s.repo.DeleteVectorAPIKey(ctx, id)
}

var permissionSetByRole = map[string]map[VectorPermission]struct{}{
	"admin": {
		VectorPermissionSearch:    {},
		VectorPermissionRecommend: {},
		VectorPermissionRead:      {},
		VectorPermissionManage:    {},
		VectorPermissionImport:    {},
		VectorPermissionMonitor:   {},
	},
	"editor": {
		VectorPermissionSearch:    {},
		VectorPermissionRecommend: {},
		VectorPermissionRead:      {},
		VectorPermissionImport:    {},
		VectorPermissionMonitor:   {},
	},
	"viewer": {
		VectorPermissionSearch:    {},
		VectorPermissionRecommend: {},
		VectorPermissionRead:      {},
		VectorPermissionMonitor:   {},
	},
	"reader": {
		VectorPermissionSearch:    {},
		VectorPermissionRecommend: {},
		VectorPermissionRead:      {},
		VectorPermissionMonitor:   {},
	},
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}
