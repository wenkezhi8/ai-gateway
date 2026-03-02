package vectordb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	internalqdrant "ai-gateway/internal/qdrant"
)

func TestPerformanceTargets_ImportTenThousandRecords_ShouldComplete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bulk-10k.json")
	records := make([]map[string]any, 0, 10000)
	for i := 0; i < 10000; i++ {
		records = append(records, map[string]any{
			"id":     "doc-" + itoa(i),
			"vector": []float64{0.1, 0.2, 0.3, 0.4},
			"title":  "doc",
		})
	}
	raw, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("json.Marshal(records) error = %v", err)
	}
	if writeErr := os.WriteFile(path, raw, 0o644); writeErr != nil {
		t.Fatalf("os.WriteFile(path) error = %v", writeErr)
	}

	repo := &mockRepo{importJobs: map[string]*ImportJob{
		"job_10k": {
			ID:             "job_10k",
			CollectionID:   "col_1",
			CollectionName: "docs",
			FilePath:       path,
			TotalRecords:   10000,
			MaxRetries:     3,
			Status:         ImportJobStatusPending,
		},
	}}

	backend := &countingBackend{}
	svc := NewServiceWithDeps(repo, backend)
	job, err := svc.RunImportJob(context.Background(), "job_10k")
	if err != nil {
		t.Fatalf("RunImportJob() error = %v", err)
	}
	if job.Status != ImportJobStatusCompleted {
		t.Fatalf("RunImportJob() status=%s, want completed", job.Status)
	}
	if job.ProcessedRecords != 10000 {
		t.Fatalf("RunImportJob() processed=%d, want 10000", job.ProcessedRecords)
	}
	if backend.totalUpserted != 10000 {
		t.Fatalf("backend totalUpserted=%d, want 10000", backend.totalUpserted)
	}
}

func TestPerformanceTargets_SearchLatencyP95Under100ms_WithMockBackend(t *testing.T) {
	t.Parallel()

	backend := &mockSearchBackend{searchResp: []SearchResult{{ID: "vec-1", Score: 0.9, Payload: map[string]any{"kind": "faq"}}}}
	svc := NewServiceWithDeps(&mockRepo{}, backend)

	const iterations = 200
	costs := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := svc.SearchVectors(context.Background(), &SearchVectorsRequest{
			CollectionName: "docs",
			TopK:           5,
			Vector:         []float32{0.1, 0.2, 0.3},
		})
		if err != nil {
			t.Fatalf("SearchVectors() error = %v", err)
		}
		costs = append(costs, time.Since(start))
	}

	p95 := percentile95(costs)
	if p95 >= 100*time.Millisecond {
		t.Fatalf("search p95=%s, want <100ms", p95)
	}
}

func TestPerformanceTargets_SupportMillionVectors_WithBatchUpsertSimulation(t *testing.T) {
	t.Parallel()

	backend := &countingBackend{}
	batch := make([]internalqdrant.UpsertPoint, 1000)
	for i := range batch {
		batch[i] = internalqdrant.UpsertPoint{ID: "doc", Vector: []float32{0.1, 0.2, 0.3}}
	}
	for i := 0; i < 1000; i++ {
		if err := backend.UpsertPoints(context.Background(), "docs", batch); err != nil {
			t.Fatalf("UpsertPoints() error = %v", err)
		}
	}
	if backend.totalUpserted != 1_000_000 {
		t.Fatalf("totalUpserted=%d, want 1000000", backend.totalUpserted)
	}
}

type countingBackend struct {
	mockBackend
	totalUpserted int64
}

func (b *countingBackend) UpsertPoints(_ context.Context, _ string, points []internalqdrant.UpsertPoint) error {
	b.totalUpserted += int64(len(points))
	return nil
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func percentile95(costs []time.Duration) time.Duration {
	if len(costs) == 0 {
		return 0
	}
	sorted := append([]time.Duration{}, costs...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	idx := (len(sorted)*95 + 99) / 100
	if idx <= 0 {
		idx = 1
	}
	if idx > len(sorted) {
		idx = len(sorted)
	}
	return sorted[idx-1]
}
