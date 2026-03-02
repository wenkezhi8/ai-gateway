package vectordb

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestVectorDBService_SearchVectors_WhenInvalidRequest_ShouldFail(t *testing.T) {
	t.Parallel()

	t.Run("nil request", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{})
		_, err := svc.SearchVectors(context.Background(), nil)
		if err == nil {
			t.Fatal("SearchVectors() should fail for nil request")
		}
	})

	t.Run("backend is nil", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(&mockRepo{}, nil)
		_, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{
			CollectionName: "docs",
			TopK:           3,
			Vector:         []float32{0.1, 0.2},
		})
		if !errors.Is(err, ErrBackendUnavailable) {
			t.Fatalf("SearchVectors() err=%v, want ErrBackendUnavailable", err)
		}
	})

	t.Run("repo is nil", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(nil, &mockSearchBackend{})
		_, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{
			CollectionName: "docs",
			TopK:           3,
			Vector:         []float32{0.1, 0.2},
		})
		if err == nil {
			t.Fatal("SearchVectors() should fail when repo is nil")
		}
	})

	t.Run("collection not found", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(&mockRepo{getErr: ErrCollectionNotFound}, &mockSearchBackend{})
		_, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{
			CollectionName: "missing",
			TopK:           3,
			Vector:         []float32{0.1, 0.2},
		})
		if !errors.Is(err, ErrCollectionNotFound) {
			t.Fatalf("SearchVectors() err=%v, want ErrCollectionNotFound", err)
		}
	})

	t.Run("missing collection name", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{})
		_, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{TopK: 1, Vector: []float32{0.1}})
		if err == nil {
			t.Fatal("SearchVectors() should fail for missing collection_name")
		}
	})

	t.Run("invalid topk", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{})
		_, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{CollectionName: "docs", TopK: 0, Vector: []float32{0.1}})
		if err == nil {
			t.Fatal("SearchVectors() should fail for top_k <= 0")
		}
	})

	t.Run("missing vector", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{})
		_, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{CollectionName: "docs", TopK: 1})
		if err == nil {
			t.Fatal("SearchVectors() should fail for missing vector")
		}
	})
}

func TestVectorDBService_SearchVectors_WhenBackendSuccess_ShouldReturnResults(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{
		searchResp: []SearchResult{
			{ID: "vec-1", Score: 0.91, Payload: map[string]any{"title": "doc 1"}},
			{ID: "vec-2", Score: 0.88, Payload: map[string]any{"title": "doc 2"}},
		},
	}
	svc := NewServiceWithDeps(&mockRepo{}, backend)

	resp, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{
		CollectionName: "docs",
		TopK:           2,
		MinScore:       0.8,
		Vector:         []float32{0.1, 0.2, 0.3},
	})
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}
	if resp.Total != 2 || len(resp.Results) != 2 {
		t.Fatalf("SearchVectors() resp=%+v", resp)
	}
	if backend.searchCalls != 1 {
		t.Fatalf("SearchVectors() backend call mismatch, got=%d", backend.searchCalls)
	}
	if resp.Results[0].ID != "vec-1" {
		t.Fatalf("SearchVectors() first result id=%s, want vec-1", resp.Results[0].ID)
	}
}

func TestVectorDBService_SearchVectors_WhenFiltersProvided_ShouldFilterByPayload(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{
		searchResp: []SearchResult{
			{ID: "vec-1", Score: 0.91, Payload: map[string]any{"doc_type": "faq", "lang": "zh"}},
			{ID: "vec-2", Score: 0.88, Payload: map[string]any{"doc_type": "guide", "lang": "en"}},
		},
	}
	svc := NewServiceWithDeps(&mockRepo{}, backend)

	resp, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{
		CollectionName: "docs",
		TopK:           5,
		Vector:         []float32{0.1, 0.2, 0.3},
		Filters:        map[string]any{"doc_type": "faq"},
	})
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("SearchVectors() total=%d, want 1", resp.Total)
	}
	if resp.Results[0].ID != "vec-1" {
		t.Fatalf("SearchVectors() first result id=%s, want vec-1", resp.Results[0].ID)
	}
}

func TestVectorDBService_SearchVectors_WhenTextSearchProvided_ShouldEmbedAndSearch(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{searchResp: []SearchResult{{ID: "vec-1", Score: 0.9}}}
	svc := NewServiceWithDeps(&mockRepo{}, backend)
	resp, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{
		CollectionName: "docs",
		TopK:           5,
		Text:           "hello",
	})
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("SearchVectors() total=%d, want 1", resp.Total)
	}
	if len(backend.lastVector) == 0 {
		t.Fatal("SearchVectors() should pass embedded vector to backend")
	}
}

func TestVectorDBService_SearchVectors_WhenBackendError_ShouldWrap(t *testing.T) {
	t.Parallel()

	backendErr := errors.New("backend unavailable")
	svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{searchErr: backendErr})
	_, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{
		CollectionName: "docs",
		TopK:           2,
		Vector:         []float32{0.1, 0.2},
	})
	if err == nil {
		t.Fatal("SearchVectors() should fail when backend fails")
	}
	if !strings.Contains(err.Error(), "search vectors failed") {
		t.Fatalf("SearchVectors() err=%v, want wrapped context", err)
	}
}

func TestVectorDBService_SearchVectors_WhenRepeatedRequest_ShouldHitCache(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{searchResp: []SearchResult{{ID: "vec-1", Score: 0.9}}}
	svc := NewServiceWithDeps(&mockRepo{}, backend)
	req := &SearchVectorsRequest{CollectionName: "docs", TopK: 3, Vector: []float32{0.1, 0.2}}

	if _, err := svc.SearchVectors(context.Background(), req); err != nil {
		t.Fatalf("first SearchVectors() error = %v", err)
	}
	if _, err := svc.SearchVectors(context.Background(), req); err != nil {
		t.Fatalf("second SearchVectors() error = %v", err)
	}
	if backend.searchCalls != 1 {
		t.Fatalf("backend.searchCalls=%d, want 1", backend.searchCalls)
	}
}

func TestVectorDBService_RecommendVectors_WhenValidRequest_ShouldDelegateSearch(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{searchResp: []SearchResult{{ID: "vec-1", Score: 0.9}}}
	svc := NewServiceWithDeps(&mockRepo{}, backend)

	resp, err := svc.RecommendVectors(context.Background(), &RecommendVectorsRequest{
		CollectionName: "docs",
		TopK:           1,
		Vector:         []float32{0.1, 0.2},
	})
	if err != nil {
		t.Fatalf("RecommendVectors() error = %v", err)
	}
	if resp.Total != 1 || backend.searchCalls != 1 {
		t.Fatalf("RecommendVectors() resp=%+v searchCalls=%d", resp, backend.searchCalls)
	}
}

func TestVectorDBService_RecommendVectors_WhenRequestIsNil_ShouldFail(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{})
	_, err := svc.RecommendVectors(context.Background(), nil)
	if err == nil {
		t.Fatal("RecommendVectors() should fail for nil request")
	}
}

func TestVectorDBService_RecommendVectors_WhenInvalidRequest_ShouldFail(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{}
	svc := NewServiceWithDeps(&mockRepo{}, backend)

	t.Run("top_k invalid", func(t *testing.T) {
		t.Parallel()
		_, err := svc.RecommendVectors(context.Background(), &RecommendVectorsRequest{CollectionName: "docs", TopK: 0, Vector: []float32{0.1}})
		if err == nil {
			t.Fatal("RecommendVectors() should fail when top_k is invalid")
		}
	})

	t.Run("text provided", func(t *testing.T) {
		t.Parallel()
		_, err := svc.RecommendVectors(context.Background(), &RecommendVectorsRequest{CollectionName: "docs", TopK: 1, Text: "hello"})
		if err != nil {
			t.Fatalf("RecommendVectors() error = %v", err)
		}
	})
}

func TestVectorDBService_RecommendVectors_WhenBackendError_ShouldWrap(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{searchErr: errors.New("down")})
	_, err := svc.RecommendVectors(context.Background(), &RecommendVectorsRequest{
		CollectionName: "docs",
		TopK:           1,
		Vector:         []float32{0.1, 0.2},
	})
	if err == nil {
		t.Fatal("RecommendVectors() should fail when backend fails")
	}
	if !strings.Contains(err.Error(), "search vectors failed") {
		t.Fatalf("RecommendVectors() err=%v, want wrapped backend context", err)
	}
}

func TestVectorDBService_GetVectorByID_WhenFound_ShouldReturnResult(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{getByIDResp: &SearchResult{ID: "vec-1", Payload: map[string]any{"title": "doc"}}}
	svc := NewServiceWithDeps(&mockRepo{}, backend)

	result, err := svc.GetVectorByID(context.Background(), "docs", "vec-1")
	if err != nil {
		t.Fatalf("GetVectorByID() error = %v", err)
	}
	if result.ID != "vec-1" {
		t.Fatalf("GetVectorByID() id=%s, want vec-1", result.ID)
	}
}

func TestVectorDBService_GetVectorByID_WhenMissingCollection_ShouldWrap(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithDeps(&mockRepo{getErr: ErrCollectionNotFound}, &mockSearchBackend{})
	_, err := svc.GetVectorByID(context.Background(), "docs", "vec-1")
	if err == nil {
		t.Fatal("GetVectorByID() should fail for missing collection")
	}
	if !strings.Contains(err.Error(), "get collection docs failed") {
		t.Fatalf("GetVectorByID() err=%v, want wrapped context", err)
	}
}

func TestVectorDBService_GetVectorByID_WhenInvalidRequest_ShouldFail(t *testing.T) {
	t.Parallel()

	t.Run("repo is nil", func(t *testing.T) {
		t.Parallel()
		svc := NewServiceWithDeps(nil, &mockSearchBackend{})
		_, err := svc.GetVectorByID(context.Background(), "docs", "vec-1")
		if err == nil {
			t.Fatal("GetVectorByID() should fail when repo is nil")
		}
	})

	t.Run("backend is nil", func(t *testing.T) {
		t.Parallel()
		svc := NewServiceWithDeps(&mockRepo{}, nil)
		_, err := svc.GetVectorByID(context.Background(), "docs", "vec-1")
		if !errors.Is(err, ErrBackendUnavailable) {
			t.Fatalf("GetVectorByID() err=%v, want ErrBackendUnavailable", err)
		}
	})

	t.Run("empty collection name", func(t *testing.T) {
		t.Parallel()
		svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{})
		_, err := svc.GetVectorByID(context.Background(), "", "vec-1")
		if err == nil {
			t.Fatal("GetVectorByID() should fail when collection_name is empty")
		}
	})

	t.Run("empty id", func(t *testing.T) {
		t.Parallel()
		svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{})
		_, err := svc.GetVectorByID(context.Background(), "docs", "")
		if err == nil {
			t.Fatal("GetVectorByID() should fail when id is empty")
		}
	})

	t.Run("backend get by id failed", func(t *testing.T) {
		t.Parallel()
		svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{getByIDErr: errors.New("backend down")})
		_, err := svc.GetVectorByID(context.Background(), "docs", "vec-1")
		if err == nil {
			t.Fatal("GetVectorByID() should fail when backend fails")
		}
		if !strings.Contains(err.Error(), "get vector by id failed") {
			t.Fatalf("GetVectorByID() err=%v, want wrapped context", err)
		}
	})

	t.Run("backend not found propagates", func(t *testing.T) {
		t.Parallel()
		svc := NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{getByIDErr: ErrCollectionNotFound})
		_, err := svc.GetVectorByID(context.Background(), "docs", "vec-1")
		if !errors.Is(err, ErrCollectionNotFound) {
			t.Fatalf("GetVectorByID() err=%v, want ErrCollectionNotFound", err)
		}
	})
}

type mockSearchBackend struct {
	mockBackend
	searchCalls int
	searchResp  []SearchResult
	searchErr   error
	lastVector  []float32
	getByIDResp *SearchResult
	getByIDErr  error
}

func (m *mockSearchBackend) Search(_ context.Context, _ string, vector []float32, _ int, _ float32) ([]SearchResult, error) {
	m.searchCalls++
	m.lastVector = append([]float32{}, vector...)
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	items := make([]SearchResult, len(m.searchResp))
	copy(items, m.searchResp)
	return items, nil
}

func (m *mockSearchBackend) GetByID(_ context.Context, _, _ string) (*SearchResult, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if m.getByIDResp == nil {
		return nil, ErrCollectionNotFound
	}
	item := *m.getByIDResp
	return &item, nil
}
