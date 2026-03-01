package vectordb

import (
	"context"
	"testing"
)

func TestVisualizationService_GetScatterData_ShouldReturnPoints(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{
		searchResp: []SearchResult{
			{ID: "p1", Score: 0.9, Payload: map[string]any{"title": "doc1", "x": 0.2, "y": 0.3}},
			{ID: "p2", Score: 0.8, Payload: map[string]any{"title": "doc2", "x": 0.5, "y": 0.7}},
		},
	}
	svc := NewServiceWithDeps(&mockRepo{getResp: &Collection{Name: "docs", Dimension: 3}}, backend)

	resp, err := svc.GetScatterData(context.Background(), &GetScatterDataRequest{CollectionName: "docs", SampleSize: 2})
	if err != nil {
		t.Fatalf("GetScatterData() error = %v", err)
	}
	if len(resp.Points) != 2 {
		t.Fatalf("GetScatterData() points=%d, want 2", len(resp.Points))
	}
	if resp.Points[0].ID == "" || resp.Points[0].Label == "" {
		t.Fatalf("GetScatterData() point=%+v", resp.Points[0])
	}
}
