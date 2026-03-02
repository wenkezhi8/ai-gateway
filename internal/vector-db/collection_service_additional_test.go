package vectordb

import (
	"context"
	"errors"
	"testing"

	internalqdrant "ai-gateway/internal/qdrant"
)

func TestVectorDBService_CreateCollection_ValidationAndDependencyErrors(t *testing.T) {
	t.Parallel()

	if _, err := NewServiceWithDeps(&mockRepo{}, &mockBackend{}).CreateCollection(context.Background(), nil); err == nil {
		t.Fatal("CreateCollection(nil) should fail")
	}

	if _, err := NewServiceWithDeps(nil, &mockBackend{}).CreateCollection(context.Background(), &CreateCollectionRequest{Name: "docs", Dimension: 3}); err == nil {
		t.Fatal("CreateCollection() should fail when repo is nil")
	}

	if _, err := NewServiceWithDeps(&mockRepo{}, nil).CreateCollection(context.Background(), &CreateCollectionRequest{Name: "docs", Dimension: 3}); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("CreateCollection() err=%v, want ErrBackendUnavailable", err)
	}

	svc := NewServiceWithDeps(&mockRepo{}, &mockBackend{})
	if _, err := svc.CreateCollection(context.Background(), &CreateCollectionRequest{Name: " ", Dimension: 3}); err == nil {
		t.Fatal("CreateCollection(empty name) should fail")
	}
	if _, err := svc.CreateCollection(context.Background(), &CreateCollectionRequest{Name: "docs", Dimension: 0}); err == nil {
		t.Fatal("CreateCollection(non-positive dimension) should fail")
	}
}

func TestVectorDBService_DeleteCollection_ErrorBranches(t *testing.T) {
	t.Parallel()

	backend := &mockBackend{deleteErr: errors.New("backend exploded")}
	repo := &mockRepo{}
	svc := NewServiceWithDeps(repo, backend)

	deleteErr := svc.DeleteCollection(context.Background(), "docs")
	if !errors.Is(deleteErr, ErrBackendUnavailable) {
		t.Fatalf("DeleteCollection() err=%v, want ErrBackendUnavailable", deleteErr)
	}

	if err := svc.DeleteCollection(context.Background(), " "); err == nil {
		t.Fatal("DeleteCollection(empty name) should fail")
	}

	svc = NewServiceWithDeps(&mockRepo{getErr: ErrCollectionNotFound}, &mockBackend{})
	deleteErr = svc.DeleteCollection(context.Background(), "missing")
	if !errors.Is(deleteErr, ErrCollectionNotFound) {
		t.Fatalf("DeleteCollection(missing) err=%v, want ErrCollectionNotFound", deleteErr)
	}
}

func TestVectorDBService_EmptyCollection_ErrorBranches(t *testing.T) {
	t.Parallel()

	baseRepo := &mockRepo{getResp: &Collection{Name: "docs", Dimension: 3, DistanceMetric: "cosine"}}

	svc := NewServiceWithDeps(baseRepo, &scriptedBackend{deleteErrs: []error{errors.New("fail")}})
	emptyErr := svc.EmptyCollection(context.Background(), "docs")
	if !errors.Is(emptyErr, ErrBackendUnavailable) {
		t.Fatalf("EmptyCollection(delete fail) err=%v, want ErrBackendUnavailable", emptyErr)
	}

	svc = NewServiceWithDeps(baseRepo, &scriptedBackend{createErrs: []error{errors.New("create fail")}})
	emptyErr = svc.EmptyCollection(context.Background(), "docs")
	if !errors.Is(emptyErr, ErrBackendUnavailable) {
		t.Fatalf("EmptyCollection(create fail) err=%v, want ErrBackendUnavailable", emptyErr)
	}

	svc = NewServiceWithDeps(baseRepo, &scriptedBackend{createErrs: []error{errors.New("already exists")}, deleteErrs: []error{nil, errors.New("retry delete fail")}})
	emptyErr = svc.EmptyCollection(context.Background(), "docs")
	if !errors.Is(emptyErr, ErrBackendUnavailable) {
		t.Fatalf("EmptyCollection(retry delete fail) err=%v, want ErrBackendUnavailable", emptyErr)
	}

	svc = NewServiceWithDeps(baseRepo, &scriptedBackend{createErrs: []error{errors.New("already exists"), errors.New("retry create fail")}})
	emptyErr = svc.EmptyCollection(context.Background(), "docs")
	if !errors.Is(emptyErr, ErrBackendUnavailable) {
		t.Fatalf("EmptyCollection(retry create fail) err=%v, want ErrBackendUnavailable", emptyErr)
	}

	svc = NewServiceWithDeps(&mockRepo{getResp: &Collection{Name: "docs", Dimension: 3, DistanceMetric: "cosine"}, updateStatsErr: errors.New("stats fail")}, &scriptedBackend{})
	emptyErr = svc.EmptyCollection(context.Background(), "docs")
	if emptyErr == nil {
		t.Fatal("EmptyCollection(update stats fail) should fail")
	}

	svc = NewServiceWithDeps(&mockRepo{}, &scriptedBackend{})
	if err := svc.EmptyCollection(context.Background(), " "); err == nil {
		t.Fatal("EmptyCollection(empty name) should fail")
	}
}

type scriptedBackend struct {
	deleteErrs []error
	createErrs []error

	deleteCalls int
	createCalls int
}

func (b *scriptedBackend) CreateCollection(_ context.Context, _ string, _ int, _ string) error {
	if b.createCalls < len(b.createErrs) {
		err := b.createErrs[b.createCalls]
		b.createCalls++
		return err
	}
	b.createCalls++
	return nil
}

func (b *scriptedBackend) DeleteCollection(_ context.Context, _ string) error {
	if b.deleteCalls < len(b.deleteErrs) {
		err := b.deleteErrs[b.deleteCalls]
		b.deleteCalls++
		return err
	}
	b.deleteCalls++
	return nil
}

func (b *scriptedBackend) GetCollectionInfo(_ context.Context, _ string) (*internalqdrant.CollectionInfo, error) {
	return &internalqdrant.CollectionInfo{VectorCount: 1, IndexedCount: 1, SizeBytes: 1}, nil
}

func (b *scriptedBackend) UpsertPoints(_ context.Context, _ string, _ []internalqdrant.UpsertPoint) error {
	return nil
}

func (b *scriptedBackend) Search(_ context.Context, _ string, _ []float32, _ int, _ float32) ([]SearchResult, error) {
	return nil, nil
}

func (b *scriptedBackend) GetByID(_ context.Context, _, _ string) (*SearchResult, error) {
	return nil, nil
}
