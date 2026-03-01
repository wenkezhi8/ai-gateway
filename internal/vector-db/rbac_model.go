package vectordb

import "time"

type VectorPermission string

const (
	VectorPermissionSearch    VectorPermission = "vector.search"
	VectorPermissionRecommend VectorPermission = "vector.recommend"
	VectorPermissionRead      VectorPermission = "vector.read"
	VectorPermissionManage    VectorPermission = "vector.manage"
)

type VectorAPIKey struct {
	ID        int64     `json:"id"`
	KeyHash   string    `json:"-"`
	Role      string    `json:"role"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateVectorPermissionRequest struct {
	APIKey string `json:"api_key"`
	Role   string `json:"role"`
}
