# Vector DB Phase 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `internal/vector-db` 从内存实现升级为 `SQLite 元数据 + Qdrant 集合生命周期` 的可用后端，并保持现有 admin API 契约不变。

**Architecture:** 采用 service 编排层 + SQLite repository + Qdrant backend client 的最小拆分。写路径先调用 Qdrant，再落 SQLite，SQLite 失败时执行补偿删除；读路径以 SQLite 为主，统计优先 Qdrant 并支持降级。Handler 只做参数/响应编排，不下沉业务逻辑。

**Tech Stack:** Go, Gin, SQLite (`database/sql`), Qdrant Go client, Go testing (`testing`), testify/mock（若仓库已使用）

---

### Task 1: 定义可替换接口与构造注入

**Files:**
- Modify: `internal/vector-db/collection.go`
- Test: `internal/vector-db/collection_service_test.go`

**Step 1: Write the failing test**

```go
func TestVectorDBService_CreateCollection_ShouldCallBackendAndRepo(t *testing.T) {
	t.Parallel()

	backend := &mockBackend{}
	repo := &mockRepo{}
	svc := NewServiceWithDeps(repo, backend)

	_, err := svc.CreateCollection(context.Background(), &CreateCollectionRequest{Name: "docs", Dimension: 768})
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	if backend.createCalls != 1 || repo.createCalls != 1 {
		t.Fatalf("unexpected calls: backend=%d repo=%d", backend.createCalls, repo.createCalls)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/vector-db -run TestVectorDBService_CreateCollection_ShouldCallBackendAndRepo -v`
Expected: FAIL（`NewServiceWithDeps` 未定义或调用次数不符合预期）

**Step 3: Write minimal implementation**

```go
type CollectionRepository interface { /* create/get/list/update/delete */ }
type CollectionBackend interface { /* create/delete/info */ }

func NewServiceWithDeps(repo CollectionRepository, backend CollectionBackend) *Service {
	return &Service{repo: repo, backend: backend}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/vector-db -run TestVectorDBService_CreateCollection_ShouldCallBackendAndRepo -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/vector-db/collection.go internal/vector-db/collection_service_test.go
git commit -m "refactor(vector-db): inject backend and repository interfaces"
```

### Task 2: 实现 SQLite repository（Create/Get）

**Files:**
- Create: `internal/vector-db/repository_sqlite.go`
- Create: `internal/vector-db/repository_sqlite_test.go`
- Modify: `internal/vector-db/collection_model.go`

**Step 1: Write the failing test**

```go
func TestSQLiteRepository_CreateAndGet_ShouldPersistCollection(t *testing.T) {
	t.Parallel()

	db := setupTestSQLite(t)
	repo := NewSQLiteRepository(db)

	meta := &Collection{Name: "docs", Dimension: 768, DistanceMetric: "cosine"}
	if err := repo.Create(context.Background(), meta); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.Get(context.Background(), "docs")
	if err != nil || got.Name != "docs" {
		t.Fatalf("Get() got=%v err=%v", got, err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/vector-db -run TestSQLiteRepository_CreateAndGet_ShouldPersistCollection -v`
Expected: FAIL（`NewSQLiteRepository` 未定义或 SQL 行为缺失）

**Step 3: Write minimal implementation**

```go
func (r *SQLiteRepository) Create(ctx context.Context, col *Collection) error { /* INSERT vector_collections */ }
func (r *SQLiteRepository) Get(ctx context.Context, name string) (*Collection, error) { /* SELECT by name */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/vector-db -run TestSQLiteRepository_CreateAndGet_ShouldPersistCollection -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/vector-db/repository_sqlite.go internal/vector-db/repository_sqlite_test.go internal/vector-db/collection_model.go
git commit -m "feat(vector-db): add sqlite repository create/get"
```

### Task 3: 扩展 SQLite repository（List/Update/Delete）

**Files:**
- Modify: `internal/vector-db/repository_sqlite.go`
- Modify: `internal/vector-db/repository_sqlite_test.go`

**Step 1: Write the failing test**

```go
func TestSQLiteRepository_ListUpdateDelete_ShouldSupportFiltersAndLifecycle(t *testing.T) {
	t.Parallel()
	// 预置多条数据，断言 environment/status/tag/limit/offset
	// 更新 description/status
	// 删除后查询应返回 ErrCollectionNotFound
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/vector-db -run TestSQLiteRepository_ListUpdateDelete_ShouldSupportFiltersAndLifecycle -v`
Expected: FAIL（过滤或更新删除逻辑未实现）

**Step 3: Write minimal implementation**

```go
func (r *SQLiteRepository) List(ctx context.Context, q *ListCollectionsQuery) ([]Collection, error) { /* dynamic query */ }
func (r *SQLiteRepository) Update(ctx context.Context, name string, req *UpdateCollectionRequest) error { /* UPDATE */ }
func (r *SQLiteRepository) Delete(ctx context.Context, name string, force bool) error { /* DELETE or soft delete */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/vector-db -run TestSQLiteRepository_ListUpdateDelete_ShouldSupportFiltersAndLifecycle -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/vector-db/repository_sqlite.go internal/vector-db/repository_sqlite_test.go
git commit -m "feat(vector-db): add sqlite repository list update delete"
```

### Task 4: 扩展 Qdrant client 的集合操作接口

**Files:**
- Modify: `internal/qdrant/client.go`
- Modify: `internal/qdrant/client_test.go`

**Step 1: Write the failing test**

```go
func TestClient_CreateDeleteGetCollectionInfo_ShouldHandleInvalidInput(t *testing.T) {
	t.Parallel()
	client := &Client{}
	if err := client.CreateCollection(context.Background(), "", 768, "cosine"); err == nil {
		t.Fatal("CreateCollection() should fail for empty name")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/qdrant -run TestClient_CreateDeleteGetCollectionInfo_ShouldHandleInvalidInput -v`
Expected: FAIL（方法未实现）

**Step 3: Write minimal implementation**

```go
func (c *Client) CreateCollection(ctx context.Context, name string, dimension int, metric string) error { /* qdrant create */ }
func (c *Client) DeleteCollection(ctx context.Context, name string) error { /* qdrant delete */ }
func (c *Client) GetCollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) { /* qdrant info */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/qdrant -run TestClient_CreateDeleteGetCollectionInfo_ShouldHandleInvalidInput -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/qdrant/client.go internal/qdrant/client_test.go
git commit -m "feat(qdrant): add collection lifecycle operations"
```

### Task 5: 在 Service 中实现 Create 的补偿逻辑

**Files:**
- Modify: `internal/vector-db/collection.go`
- Modify: `internal/vector-db/collection_service_test.go`

**Step 1: Write the failing test**

```go
func TestVectorDBService_CreateCollection_WhenRepoCreateFails_ShouldRollbackBackend(t *testing.T) {
	t.Parallel()

	backend := &mockBackend{}
	repo := &mockRepo{createErr: errors.New("db down")}
	svc := NewServiceWithDeps(repo, backend)

	_, err := svc.CreateCollection(context.Background(), &CreateCollectionRequest{Name: "docs", Dimension: 768})
	if err == nil {
		t.Fatal("CreateCollection() should fail")
	}
	if backend.deleteCalls != 1 {
		t.Fatalf("rollback not called, deleteCalls=%d", backend.deleteCalls)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/vector-db -run TestVectorDBService_CreateCollection_WhenRepoCreateFails_ShouldRollbackBackend -v`
Expected: FAIL（未执行回滚）

**Step 3: Write minimal implementation**

```go
if err := s.backend.CreateCollection(...); err != nil { return nil, ErrBackendUnavailable }
if err := s.repo.Create(ctx, col); err != nil {
	_ = s.backend.DeleteCollection(ctx, col.Name)
	return nil, fmt.Errorf("create metadata failed: %w", err)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/vector-db -run TestVectorDBService_CreateCollection_WhenRepoCreateFails_ShouldRollbackBackend -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/vector-db/collection.go internal/vector-db/collection_service_test.go
git commit -m "feat(vector-db): add create compensation rollback"
```

### Task 6: 在 Service 中实现 Delete/Stats 的后端联动与降级

**Files:**
- Modify: `internal/vector-db/collection.go`
- Modify: `internal/vector-db/collection_service_test.go`

**Step 1: Write the failing test**

```go
func TestVectorDBService_GetCollectionStats_WhenBackendFails_ShouldFallbackToMetadata(t *testing.T) {
	t.Parallel()
	backend := &mockBackend{infoErr: errors.New("backend down")}
	repo := &mockRepo{getResp: &Collection{Name: "docs", VectorCount: 12, IndexedCount: 11, SizeBytes: 1024}}
	svc := NewServiceWithDeps(repo, backend)

	stats, err := svc.GetCollectionStats(context.Background(), "docs")
	if err != nil || stats.VectorCount != 12 {
		t.Fatalf("stats=%v err=%v", stats, err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/vector-db -run TestVectorDBService_GetCollectionStats_WhenBackendFails_ShouldFallbackToMetadata -v`
Expected: FAIL（未降级）

**Step 3: Write minimal implementation**

```go
info, err := s.backend.GetCollectionInfo(ctx, name)
if err != nil {
	col, getErr := s.repo.Get(ctx, name)
	if getErr != nil { return nil, getErr }
	return &CollectionStats{...from metadata...}, nil
}
return &CollectionStats{...from backend...}, nil
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/vector-db -run TestVectorDBService_GetCollectionStats_WhenBackendFails_ShouldFallbackToMetadata -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/vector-db/collection.go internal/vector-db/collection_service_test.go
git commit -m "feat(vector-db): add stats fallback and delete backend coordination"
```

### Task 7: 连接 Admin 初始化与 Handler 错误语义

**Files:**
- Modify: `internal/handler/admin/admin.go`
- Modify: `internal/vector-db/collection_handler.go`
- Modify: `internal/vector-db/collection_handler_test.go`

**Step 1: Write the failing test**

```go
func TestCollectionHandler_Create_WhenBackendUnavailable_ShouldReturn503(t *testing.T) {
	t.Parallel()
	// mock service 返回 ErrBackendUnavailable，断言 HTTP 503
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/vector-db -run TestCollectionHandler_Create_WhenBackendUnavailable_ShouldReturn503 -v`
Expected: FAIL（错误映射未覆盖）

**Step 3: Write minimal implementation**

```go
switch {
case errors.Is(err, ErrBackendUnavailable): c.JSON(http.StatusServiceUnavailable, ...)
case errors.Is(err, ErrCollectionExists): c.JSON(http.StatusConflict, ...)
default: c.JSON(http.StatusInternalServerError, ...)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/vector-db -run TestCollectionHandler_Create_WhenBackendUnavailable_ShouldReturn503 -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/admin/admin.go internal/vector-db/collection_handler.go internal/vector-db/collection_handler_test.go
git commit -m "refactor(vector-db): wire sqlite qdrant service into admin handler"
```

### Task 8: 全量回归验证与交付检查

**Files:**
- Modify: `docs/plans/2026-03-02-vector-db-phase1-implementation-plan.md`（记录实际执行备注，可选）

**Step 1: Run package-level tests first**

Run: `go test ./internal/vector-db ./internal/qdrant -v`
Expected: PASS

**Step 2: Run backend required checks**

Run: `make lint && go test ./... && go build ./cmd/gateway`
Expected: PASS

**Step 3: Run frontend type check if contracts changed**

Run: `cd web && npm run typecheck`
Expected: PASS

**Step 4: Verify git scope and cleanliness**

Run: `git status --short`
Expected: 仅包含本任务目标文件改动（并行改动文件不得混入提交）

**Step 5: Commit**

```bash
git add <仅本任务文件>
git commit -m "feat(vector-db): complete phase1 sqlite-qdrant collection workflow"
```
