package vectordb

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestQdrantBackend_WhenClientMissing_ShouldReturnErrBackendUnavailable(t *testing.T) {
	t.Parallel()

	backend := &qdrantBackend{}
	ctx := context.Background()

	if err := backend.CreateCollection(ctx, "docs", 3, "cosine"); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("CreateCollection() err=%v, want ErrBackendUnavailable", err)
	}
	if err := backend.DeleteCollection(ctx, "docs"); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("DeleteCollection() err=%v, want ErrBackendUnavailable", err)
	}
	if _, err := backend.GetCollectionInfo(ctx, "docs"); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("GetCollectionInfo() err=%v, want ErrBackendUnavailable", err)
	}
	if err := backend.UpsertPoints(ctx, "docs", nil); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("UpsertPoints() err=%v, want ErrBackendUnavailable", err)
	}
	if _, err := backend.Search(ctx, "docs", []float32{0.1}, 1, 0); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("Search() err=%v, want ErrBackendUnavailable", err)
	}
	if _, err := backend.GetByID(ctx, "docs", "id-1"); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("GetByID() err=%v, want ErrBackendUnavailable", err)
	}
}

func TestService_GetRepository_WhenServiceNil_ShouldReturnNil(t *testing.T) {
	t.Parallel()

	var svc *Service
	if got := svc.GetRepository(); got != nil {
		t.Fatalf("GetRepository() = %v, want nil", got)
	}
}

func TestService_NewServiceWithConfig_WhenQdrantAddrInvalid_ShouldStillReturnService(t *testing.T) {
	t.Parallel()

	db := setupTestSQLite(t)
	svc := NewServiceWithConfig(ServiceConfig{DB: db, QdrantHTTPAddr: "://bad-url"})
	if svc == nil {
		t.Fatal("NewServiceWithConfig() returned nil")
	}
	if svc.GetRepository() == nil {
		t.Fatal("service repository should not be nil")
	}
}

func TestService_NewService_ShouldReturnNonNil(t *testing.T) {
	t.Setenv("AI_GATEWAY_SQLITE_PATH", t.TempDir()+"/vector-db-test.db")
	t.Setenv("AI_GATEWAY_QDRANT_URL", "://bad-url")
	svc := NewService()
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestCollectionHandler_RBACServiceAccessor_ShouldReturnRBACService(t *testing.T) {
	t.Parallel()

	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{}))
	if h.RBACService() == nil {
		t.Fatal("RBACService() should not return nil")
	}
}

func TestRateLimiter_UtilityPaths_ShouldExerciseFallbackBranches(t *testing.T) {
	limiter := NewVectorSearchRateLimiter(1, time.Second)
	if limiter == nil {
		t.Fatal("NewVectorSearchRateLimiter() returned nil")
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request = httptest.NewRequest("GET", "/", http.NoBody)
	ctx.Request.Header.Set("Authorization", "Bearer token-1")
	ctx.Params = []gin.Param{{Key: "name", Value: "docs"}}
	if key := limiter.requestKey(ctx); key != "token-1:docs" {
		t.Fatalf("requestKey()=%s, want token-1:docs", key)
	}

	limiter.redisClient = redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 10 * time.Millisecond, ReadTimeout: 10 * time.Millisecond, WriteTimeout: 10 * time.Millisecond})
	defer limiter.redisClient.Close()

	if _, _, err := limiter.allowNowWithRedis("k1"); err == nil {
		t.Fatal("allowNowWithRedis() should fail with unreachable redis")
	}
	allowed, _ := limiter.allowNow("k2")
	if !allowed {
		t.Fatal("allowNow() should fallback to in-memory limiter")
	}

	t.Setenv("AI_GATEWAY_REDIS_ADDR", "")
	if client := newVectorRedisClient(); client != nil {
		t.Fatal("newVectorRedisClient() with empty env should return nil")
	}
	t.Setenv("AI_GATEWAY_REDIS_ADDR", "127.0.0.1:1")
	if client := newVectorRedisClient(); client != nil {
		t.Fatal("newVectorRedisClient() with unavailable redis should return nil")
	}
}

func TestVisualizationUtilityFunctions_ShouldHandleFallbacks(t *testing.T) {
	t.Parallel()

	point := mapSearchResultToScatterPoint(SearchResult{ID: "doc-id-1", Score: 0.7, Payload: map[string]any{"title": ""}})
	if point.Label != "doc-id-1" {
		t.Fatalf("Label=%s, want doc-id-1", point.Label)
	}
	if point.X < -1 || point.X > 1 || point.Y < -1 || point.Y > 1 {
		t.Fatalf("fallback coords out of range: x=%f y=%f", point.X, point.Y)
	}

	x1, y1 := hashToXY("seed-1")
	x2, y2 := hashToXY("seed-1")
	if x1 != x2 || y1 != y2 {
		t.Fatalf("hashToXY() should be deterministic: (%f,%f) vs (%f,%f)", x1, y1, x2, y2)
	}
	if math.Abs(x1) > 1 || math.Abs(y1) > 1 {
		t.Fatalf("hashToXY() out of range: x=%f y=%f", x1, y1)
	}

	tests := []struct {
		name  string
		input any
		ok    bool
	}{
		{name: "float64", input: float64(1.2), ok: true},
		{name: "float32", input: float32(1.2), ok: true},
		{name: "int", input: int(3), ok: true},
		{name: "int32", input: int32(3), ok: true},
		{name: "int64", input: int64(3), ok: true},
		{name: "invalid", input: "3", ok: false},
	}
	for _, tc := range tests {
		_, ok := toFloat(tc.input)
		if ok != tc.ok {
			t.Fatalf("toFloat(%s) ok=%v, want %v", tc.name, ok, tc.ok)
		}
	}
}

func TestImportUtilityFunctions_ShouldParseVectorsAndNumbers(t *testing.T) {
	t.Parallel()

	if vec, ok := parseCSVVector("[0.1,0.2,0.3]"); !ok || len(vec) != 3 {
		t.Fatalf("parseCSVVector(json) = (%v,%v), want ok and len=3", vec, ok)
	}
	if vec, ok := parseCSVVector("0.4, 0.5, 0.6"); !ok || len(vec) != 3 {
		t.Fatalf("parseCSVVector(csv) = (%v,%v), want ok and len=3", vec, ok)
	}
	if _, ok := parseCSVVector(" "); ok {
		t.Fatal("parseCSVVector(empty) should fail")
	}
	if _, ok := parseCSVVector("bad,1"); ok {
		t.Fatal("parseCSVVector(invalid) should fail")
	}

	if _, ok := toFloat32(float64(1)); !ok {
		t.Fatal("toFloat32(float64) should succeed")
	}
	if _, ok := toFloat32(float32(1)); !ok {
		t.Fatal("toFloat32(float32) should succeed")
	}
	if _, ok := toFloat32(int(1)); !ok {
		t.Fatal("toFloat32(int) should succeed")
	}
	if _, ok := toFloat32(int64(1)); !ok {
		t.Fatal("toFloat32(int64) should succeed")
	}
	if _, ok := toFloat32("1"); ok {
		t.Fatal("toFloat32(string) should fail")
	}
}
