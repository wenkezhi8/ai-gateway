package vectordb

import (
	"context"
	"testing"
	"time"
)

func TestIndexConfigService_UpdateAndGet_ShouldPersist(t *testing.T) {
	t.Parallel()

	repo, err := NewSQLiteRepository(setupTestSQLite(t))
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}

	now := time.Now().UTC()
	createErr := repo.Create(context.Background(), &Collection{
		ID:              "col_docs",
		Name:            "docs",
		Description:     "docs collection",
		Dimension:       768,
		DistanceMetric:  "cosine",
		IndexType:       "hnsw",
		HNSWM:           16,
		HNSWEFConstruct: 100,
		IVFNList:        1024,
		StorageBackend:  "qdrant",
		Environment:     "prod",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedBy:       "tester",
	})
	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}

	svc := NewServiceWithDeps(repo, &mockBackend{})
	hnswM := 32
	hnswEFConstruct := 256
	updated, err := svc.UpdateIndexConfig(context.Background(), "docs", &UpdateIndexConfigRequest{
		IndexType:       "hnsw",
		HNSWM:           &hnswM,
		HNSWEFConstruct: &hnswEFConstruct,
	})
	if err != nil {
		t.Fatalf("UpdateIndexConfig() error = %v", err)
	}
	if updated.IndexType != "hnsw" || updated.HNSWM != 32 || updated.HNSWEFConstruct != 256 {
		t.Fatalf("UpdateIndexConfig() result = %+v", updated)
	}

	got, err := svc.GetIndexConfig(context.Background(), "docs")
	if err != nil {
		t.Fatalf("GetIndexConfig() error = %v", err)
	}
	if got.IndexType != "hnsw" || got.HNSWM != 32 || got.HNSWEFConstruct != 256 || got.IVFNList != 1024 {
		t.Fatalf("GetIndexConfig() result = %+v", got)
	}
}

func TestIndexConfigService_Update_WhenInvalidRequest_ShouldFail(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithDeps(&mockRepo{}, &mockBackend{})
	_, err := svc.UpdateIndexConfig(context.Background(), "docs", &UpdateIndexConfigRequest{IndexType: "invalid"})
	if err == nil {
		t.Fatal("UpdateIndexConfig() should fail")
	}
}
