package vectordb

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
)

func (s *Service) GetScatterData(ctx context.Context, req *GetScatterDataRequest) (*ScatterDataResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if s.backend == nil {
		return nil, ErrBackendUnavailable
	}
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	collectionName := strings.TrimSpace(req.CollectionName)
	if collectionName == "" {
		return nil, fmt.Errorf("collection_name is required")
	}
	if req.SampleSize <= 0 {
		return nil, fmt.Errorf("sample_size must be positive")
	}

	collection, err := s.repo.Get(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("get collection %s failed: %w", collectionName, err)
	}

	probeVector := make([]float32, collection.Dimension)
	results, err := s.backend.Search(ctx, collectionName, probeVector, req.SampleSize, -1)
	if err != nil {
		return nil, fmt.Errorf("search vectors for visualization failed: %w", err)
	}

	points := make([]ScatterPoint, 0, len(results))
	for idx := range results {
		points = append(points, mapSearchResultToScatterPoint(results[idx]))
	}

	return &ScatterDataResponse{Points: points, Total: len(points)}, nil
}

func mapSearchResultToScatterPoint(item SearchResult) ScatterPoint {
	label := strings.TrimSpace(item.ID)
	if title, ok := item.Payload["title"].(string); ok && strings.TrimSpace(title) != "" {
		label = strings.TrimSpace(title)
	}

	x, okX := toFloat(item.Payload["x"])
	y, okY := toFloat(item.Payload["y"])
	if !okX || !okY {
		x, y = hashToXY(item.ID)
	}

	return ScatterPoint{
		ID:    item.ID,
		X:     x,
		Y:     y,
		Label: label,
		Score: item.Score,
	}
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func hashToXY(seed string) (x, y float64) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	v := h.Sum64()
	x = float64(v%1000)/500 - 1
	y = float64((v/1000)%1000)/500 - 1
	return math.Round(x*1000) / 1000, math.Round(y*1000) / 1000
}
