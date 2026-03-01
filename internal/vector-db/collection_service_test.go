package vectordb

import (
	"context"
	"testing"
)

func TestVectorDBService_CRUD(t *testing.T) {
	t.Parallel()

	svc := NewService()
	ctx := context.Background()

	created, err := svc.CreateCollection(ctx, &CreateCollectionRequest{
		Name:           "docs",
		Description:    "documents",
		Dimension:      768,
		DistanceMetric: "cosine",
		IndexType:      "hnsw",
		StorageBackend: "memory",
		Tags:           []string{"prod", "shared"},
		Environment:    "prod",
		CreatedBy:      "tester",
		IsPublic:       true,
	})
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}
	if created.Name != "docs" {
		t.Fatalf("CreateCollection() name = %q, want %q", created.Name, "docs")
	}

	_, errDup := svc.CreateCollection(ctx, &CreateCollectionRequest{Name: "docs", Dimension: 768})
	if errDup == nil {
		t.Fatal("CreateCollection() duplicate should fail")
	}

	got, err := svc.GetCollection(ctx, "docs")
	if err != nil {
		t.Fatalf("GetCollection() error = %v", err)
	}
	if got.Dimension != 768 {
		t.Fatalf("GetCollection() dimension = %d, want %d", got.Dimension, 768)
	}

	updatedDesc := "updated documents"
	updatedStatus := "inactive"
	err = svc.UpdateCollection(ctx, "docs", &UpdateCollectionRequest{
		Description: &updatedDesc,
		Status:      &updatedStatus,
	})
	if err != nil {
		t.Fatalf("UpdateCollection() error = %v", err)
	}
	updated, err := svc.GetCollection(ctx, "docs")
	if err != nil {
		t.Fatalf("GetCollection() after update error = %v", err)
	}
	if updated.Description != updatedDesc {
		t.Fatalf("UpdateCollection() description = %q, want %q", updated.Description, updatedDesc)
	}
	if updated.Status != updatedStatus {
		t.Fatalf("UpdateCollection() status = %q, want %q", updated.Status, updatedStatus)
	}

	list, err := svc.ListCollections(ctx, &ListCollectionsQuery{Environment: "prod"})
	if err != nil {
		t.Fatalf("ListCollections() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListCollections() len = %d, want %d", len(list), 1)
	}

	stats, err := svc.GetCollectionStats(ctx, "docs")
	if err != nil {
		t.Fatalf("GetCollectionStats() error = %v", err)
	}
	if stats.Name != "docs" {
		t.Fatalf("GetCollectionStats() name = %q, want %q", stats.Name, "docs")
	}

	if err := svc.DeleteCollection(ctx, "docs"); err != nil {
		t.Fatalf("DeleteCollection() error = %v", err)
	}
	if _, err := svc.GetCollection(ctx, "docs"); err == nil {
		t.Fatal("GetCollection() should fail after delete")
	}
}
