# AI Gateway 测试指南

## 测试概述

本文档描述了AI智能网关项目的测试策略、测试用例模板和最佳实践。

## 测试范围

### 1. 单元测试
- 服务商适配器测试
- 路由策略测试
- 限额监控测试
- 缓存系统测试
- 中间件测试

### 2. 集成测试
- API接口测试
- 端到端流程测试
- 安全性测试

### 3. 性能测试
- 负载测试
- 压力测试
- 并发测试

---

## 单元测试模板

### Go测试模板

#### 基础单元测试模板

```go
package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCache 测试缓存初始化
func TestNewCache(t *testing.T) {
	// Arrange
	config := &CacheConfig{
		MaxSize: 1000,
		TTL:     5 * time.Minute,
	}

	// Act
	cache := NewCache(config)

	// Assert
	assert.NotNil(t, cache)
	assert.Equal(t, 1000, cache.MaxSize)
}

// TestCache_Set_Get 测试缓存存取
func TestCache_Set_Get(t *testing.T) {
	// Arrange
	cache := NewMemoryCache()
	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Act
	err := cache.Set(ctx, key, value, 5*time.Minute)
	require.NoError(t, err)

	// Assert
	result, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

// TestCache_Delete 测试缓存删除
func TestCache_Delete(t *testing.T) {
	// Arrange
	cache := NewMemoryCache()
	ctx := context.Background()
	key := "test-key"
	cache.Set(ctx, key, []byte("value"), 5*time.Minute)

	// Act
	err := cache.Delete(ctx, key)

	// Assert
	require.NoError(t, err)
	_, err = cache.Get(ctx, key)
	assert.Error(t, err)
}

// TestCache_ConcurrentAccess 测试并发访问
func TestCache_ConcurrentAccess(t *testing.T) {
	// Arrange
	cache := NewMemoryCache()
	ctx := context.Background()
	key := "concurrent-key"

	// Act & Assert
	for i := 0; i < 100; i++ {
		go func(i int) {
			err := cache.Set(ctx, key, []byte("value"), 5*time.Minute)
			assert.NoError(t, err)
		}(i)
	}
}
```

#### 表驱动测试模板

```go
func TestRouter_SelectProvider(t *testing.T) {
	tests := []struct {
		name           string
		config         RouterConfig
		request        ChatRequest
		expected       string
		expectedError  bool
	}{
		{
			name: "选择成本最低的提供商",
			config: RouterConfig{
				Strategy: "cost-based",
				Providers: []Provider{
					{Name: "openai", Cost: 0.002},
					{Name: "claude", Cost: 0.0015},
				},
			},
			request:      ChatRequest{Model: "gpt-4"},
			expected:     "claude",
			expectedError: false,
		},
		{
			name: "无可用提供商",
			config: RouterConfig{
				Strategy:  "cost-based",
				Providers: []Provider{},
			},
			request:       ChatRequest{Model: "gpt-4"},
			expected:      "",
			expectedError: true,
		},
		{
			name: "基于模型匹配",
			config: RouterConfig{
				Strategy: "model-based",
				Providers: []Provider{
					{Name: "openai", Models: []string{"gpt-4"}},
					{Name: "claude", Models: []string{"claude-3"}},
				},
			},
			request:       ChatRequest{Model: "gpt-4"},
			expected:      "openai",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.config)
			result, err := router.SelectProvider(tt.request)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
```

#### Mock测试模板

```go
package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProvider 是Provider接口的Mock实现
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatResponse), args.Error(1)
}

// TestProvider_SelectionWithMock 使用Mock进行测试
func TestProvider_SelectionWithMock(t *testing.T) {
	// Arrange
	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("mock-provider")
	mockProvider.On("Chat", mock.Anything, mock.Anything).Return(&ChatResponse{
		ID:      "test-id",
		Choices: []Choice{{Message: Message{Content: "Hello"}}},
	}, nil)

	// Act
	adapter := NewProviderAdapter([]Provider{mockProvider})
	result, err := adapter.Route(context.Background(), &ChatRequest{Model: "test"})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Hello", result.Choices[0].Message.Content)
	mockProvider.AssertExpectations(t)
}
```

---

## 集成测试模板

### API集成测试模板

```go
package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/config"
	"ai-gateway/internal/router"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPI_ChatCompletions 端到端测试 - 聊天补全
func TestAPI_ChatCompletions(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Server: config.ServerConfig{
			Mode: "test",
		},
		Providers: config.ProvidersConfig{
			OpenAI: config.ProviderConfig{
				APIKey:  "test-key",
				Enabled: true,
			},
		},
	}

	r := router.New(cfg)
	server := httptest.NewServer(r)
	defer server.Close()

	// Test case 1: 成功请求
	t.Run("成功请求", func(t *testing.T) {
		body := map[string]interface{}{
			"model": "gpt-4",
			"messages": []map[string]string{
				{"role": "user", "content": "Hello"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/chat/completions", "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test case 2: 无效请求
	t.Run("无效请求 - 缺少model字段", func(t *testing.T) {
		body := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": "Hello"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/chat/completions", "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test case 3: 认证失败
	t.Run("认证失败", func(t *testing.T) {
		body := map[string]interface{}{
			"model": "gpt-4",
			"messages": []map[string]string{
				{"role": "user", "content": "Hello"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", server.URL+"/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer invalid-key")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestAPI_HealthCheck 健康检查测试
func TestAPI_HealthCheck(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Mode: "test"}}
	r := router.New(cfg)
	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "healthy", result["status"])
}
```

---

## 性能测试模板

```go
package benchmark

import (
	"context"
	"testing"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/router"
)

// BenchmarkCache_Get 基准测试 - 缓存读取
func BenchmarkCache_Get(b *testing.B) {
	c := cache.NewMemoryCache()
	ctx := context.Background()
	c.Set(ctx, "test-key", []byte("test-value"), 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(ctx, "test-key")
	}
}

// BenchmarkCache_Set 基准测试 - 缓存写入
func BenchmarkCache_Set(b *testing.B) {
	c := cache.NewMemoryCache()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(ctx, "test-key", []byte("test-value"), 0)
	}
}

// BenchmarkRouter_SelectProvider 基准测试 - 路由选择
func BenchmarkRouter_SelectProvider(b *testing.B) {
	r := router.NewRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.SelectProvider("gpt-4")
	}
}
```

---

## 测试覆盖率要求

| 模块 | 最低覆盖率 |
|------|-----------|
| 核心业务逻辑 | 80% |
| 服务商适配器 | 75% |
| 缓存系统 | 70% |
| 路由策略 | 75% |
| 限额监控 | 70% |
| 中间件 | 65% |

---

## 运行测试

### 运行所有测试
```bash
make test
```

### 运行特定包的测试
```bash
go test ./internal/cache/...
```

### 运行带覆盖率的测试
```bash
go test -cover ./...
```

### 生成覆盖率报告
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 运行基准测试
```bash
go test -bench=. ./...
```

---

## 测试最佳实践

1. **使用表驱动测试**：对于多个测试场景，使用表驱动测试提高代码可读性
2. **Mock外部依赖**：使用mock隔离外部API调用
3. **测试边界条件**：包括空值、极限值、错误情况
4. **清晰的测试命名**：测试名称应描述测试场景和预期结果
5. **独立的测试**：每个测试应该独立运行，不依赖其他测试
6. **清理资源**：测试结束后清理临时资源和goroutines

---

## 测试工具

- **testify**：断言和mock库
- **httptest**：HTTP测试工具
- **mockery**：Mock生成工具
- **ginkgo**：BDD测试框架（可选）
