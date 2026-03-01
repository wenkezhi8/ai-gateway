package vectordb

type GetScatterDataRequest struct {
	CollectionName string `json:"collection_name"`
	SampleSize     int    `json:"sample_size"`
}

type ScatterPoint struct {
	ID    string  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Label string  `json:"label"`
	Score float32 `json:"score"`
}

type ScatterDataResponse struct {
	Points []ScatterPoint `json:"points"`
	Total  int            `json:"total"`
}
