package vectordb

import "time"

type IndexConfig struct {
	CollectionName  string    `json:"collection_name"`
	IndexType       string    `json:"index_type"`
	HNSWM           int       `json:"hnsw_m"`
	HNSWEFConstruct int       `json:"hnsw_ef_construct"`
	IVFNList        int       `json:"ivf_nlist"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UpdateIndexConfigRequest struct {
	IndexType       string `json:"index_type"`
	HNSWM           *int   `json:"hnsw_m"`
	HNSWEFConstruct *int   `json:"hnsw_ef_construct"`
	IVFNList        *int   `json:"ivf_nlist"`
}
