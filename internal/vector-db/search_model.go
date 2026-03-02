package vectordb

import "errors"

var (
	// ErrTextSearchNotSupported indicates text query is not available yet.
	ErrTextSearchNotSupported = errors.New("text search is not supported")
)

// SearchVectorsRequest defines vector search input.
type SearchVectorsRequest struct {
	CollectionName string         `json:"collection_name"`
	TopK           int            `json:"top_k"`
	MinScore       float32        `json:"min_score"`
	Vector         []float32      `json:"vector"`
	Text           string         `json:"text"`
	Filters        map[string]any `json:"filters"`
}

// RecommendVectorsRequest defines recommendation input.
type RecommendVectorsRequest SearchVectorsRequest

// SearchResult defines one matched vector record.
type SearchResult struct {
	ID      string         `json:"id"`
	Score   float32        `json:"score"`
	Payload map[string]any `json:"payload"`
}

// SearchVectorsResponse defines search output.
type SearchVectorsResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}
