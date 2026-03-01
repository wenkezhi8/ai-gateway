package vectordb

import "time"

// Collection defines vector collection metadata.
type Collection struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Dimension       int       `json:"dimension"`
	DistanceMetric  string    `json:"distance_metric"`
	IndexType       string    `json:"index_type"`
	HNSWM           int       `json:"hnsw_m"`
	HNSWEFConstruct int       `json:"hnsw_ef_construct"`
	IVFNList        int       `json:"ivf_nlist"`
	StorageBackend  string    `json:"storage_backend"`
	Tags            []string  `json:"tags"`
	Environment     string    `json:"environment"`
	Status          string    `json:"status"`
	VectorCount     int64     `json:"vector_count"`
	IndexedCount    int64     `json:"indexed_count"`
	SizeBytes       int64     `json:"size_bytes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedBy       string    `json:"created_by"`
	IsPublic        bool      `json:"is_public"`
}

// CreateCollectionRequest defines create payload.
type CreateCollectionRequest struct {
	Name           string   `json:"name" binding:"required"`
	Description    string   `json:"description"`
	Dimension      int      `json:"dimension" binding:"required"`
	DistanceMetric string   `json:"distance_metric"`
	IndexType      string   `json:"index_type"`
	StorageBackend string   `json:"storage_backend"`
	Tags           []string `json:"tags"`
	Environment    string   `json:"environment"`
	Status         string   `json:"status"`
	CreatedBy      string   `json:"created_by"`
	IsPublic       bool     `json:"is_public"`
}

// UpdateCollectionRequest defines update payload.
type UpdateCollectionRequest struct {
	Description     *string  `json:"description"`
	DistanceMetric  *string  `json:"distance_metric"`
	IndexType       *string  `json:"index_type"`
	HNSWM           *int     `json:"hnsw_m"`
	HNSWEFConstruct *int     `json:"hnsw_ef_construct"`
	IVFNList        *int     `json:"ivf_nlist"`
	StorageBackend  *string  `json:"storage_backend"`
	Tags            []string `json:"tags"`
	Environment     *string  `json:"environment"`
	Status          *string  `json:"status"`
	IsPublic        *bool    `json:"is_public"`
}

// ListCollectionsQuery defines list filters.
type ListCollectionsQuery struct {
	Name        string
	Search      string
	Environment string
	Status      string
	Tag         string
	IsPublic    *bool
	Offset      int
	Limit       int
}

// DeleteCollectionRequest defines delete payload.
type DeleteCollectionRequest struct {
	Force bool `json:"force"`
}

// CollectionListResponse defines list response payload.
type CollectionListResponse struct {
	Collections []Collection `json:"collections"`
	Total       int          `json:"total"`
}

// CollectionDetailResponse defines detail response payload.
type CollectionDetailResponse struct {
	Collection *Collection `json:"collection"`
}

// CollectionStats defines collection statistics.
type CollectionStats struct {
	Name         string `json:"name"`
	VectorCount  int64  `json:"vector_count"`
	IndexedCount int64  `json:"indexed_count"`
	SizeBytes    int64  `json:"size_bytes"`
}
