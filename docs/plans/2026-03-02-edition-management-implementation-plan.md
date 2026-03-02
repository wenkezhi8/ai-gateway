# AI Gateway 版本化 UI 控制系统实施计划

**目标**：实现三版本（基础版/标准版/企业版）UI 动态控制系统，用户可在系统设置中切换版本，菜单、路由、页面内容根据版本自动适配，默认为基础版。

**架构**：
- **后端**：配置文件定义版本类型 + REST API 提供版本信息 + 依赖检查 + 配置热更新
- **前端**：Pinia Store 管理版本状态 + 动态菜单生成 + 路由守卫 + 页面内容适配
- **切换流程**：用户选择版本 → 后端校验依赖 → 保存配置 → 前端刷新菜单

**技术栈**：
- 后端：Go 1.24, Gin, SQLite, Redis client, HTTP client
- 前端：Vue 3, TypeScript, Pinia, Element Plus
- 测试：Go testing, Vitest

**开发规范**：
- 所有开发在 `main` 分支进行
- 禁止多个 AI 同时开发
- 遵循 TDD 流程（Red → Green → Refactor）
- 每次任务完成后必须提交并保持工作区干净

---

## Task 1: 后端版本数据模型与配置结构

**Files:**
- Create: `internal/config/edition.go`
- Modify: `internal/config/config.go`
- Test: `internal/config/edition_test.go`

**Step 1: Write failing test**

```go
func TestEditionConfig_DefaultShouldBeBasic(t *testing.T) {
    t.Parallel()
    
    cfg := &Config{}
    edition := cfg.GetEditionConfig()
    
    if edition.Type != EditionBasic {
        t.Fatalf("default edition = %v, want %v", edition.Type, EditionBasic)
    }
}

func TestEditionConfig_ShouldValidateDependencies(t *testing.T) {
    t.Parallel()
    
    definition := EditionDefinitions[EditionEnterprise]
    if len(definition.Dependencies) == 0 {
        t.Fatal("enterprise edition should have dependencies")
    }
    
    expected := []string{"redis", "ollama", "qdrant"}
    if !reflect.DeepEqual(definition.Dependencies, expected) {
        t.Fatalf("dependencies = %v, want %v", definition.Dependencies, expected)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestEditionConfig -v`
Expected: FAIL（`EditionConfig` 类型未定义）

**Step 3: Write minimal implementation**

```go
// internal/config/edition.go
package config

type EditionType string

const (
    EditionBasic      EditionType = "basic"
    EditionStandard   EditionType = "standard"
    EditionEnterprise EditionType = "enterprise"
)

type EditionFeatures struct {
    VectorCache        bool `json:"vector_cache"`
    VectorDBManagement bool `json:"vector_db_management"`
    KnowledgeBase      bool `json:"knowledge_base"`
    ColdHotTiering     bool `json:"cold_hot_tiering"`
}

type EditionConfig struct {
    Type          EditionType     `json:"type"`
    Features      EditionFeatures `json:"features"`
    DisplayName   string          `json:"display_name"`
    Description   string          `json:"description"`
    Dependencies  []string        `json:"dependencies"`
}

var EditionDefinitions = map[EditionType]EditionConfig{
    EditionBasic: {
        Type:        EditionBasic,
        DisplayName: "基础版",
        Description: "纯AI网关功能，轻量级部署",
        Features: EditionFeatures{
            VectorCache:        false,
            VectorDBManagement: false,
            KnowledgeBase:      false,
            ColdHotTiering:     false,
        },
        Dependencies: []string{"redis"},
    },
    EditionStandard: {
        Type:        EditionStandard,
        DisplayName: "标准版",
        Description: "网关 + 语义缓存，中大规模场景",
        Features: EditionFeatures{
            VectorCache:        true,
            VectorDBManagement: false,
            KnowledgeBase:      false,
            ColdHotTiering:     false,
        },
        Dependencies: []string{"redis", "ollama"},
    },
    EditionEnterprise: {
        Type:        EditionEnterprise,
        DisplayName: "企业版",
        Description: "完整功能，企业级生产环境",
        Features: EditionFeatures{
            VectorCache:        true,
            VectorDBManagement: true,
            KnowledgeBase:      true,
            ColdHotTiering:     true,
        },
        Dependencies: []string{"redis", "ollama", "qdrant"},
    },
}

func (c *Config) GetEditionConfig() EditionConfig {
    editionType := EditionType(c.Edition.Type)
    if editionType == "" {
        editionType = EditionBasic
    }
    return EditionDefinitions[editionType]
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestEditionConfig -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/edition.go internal/config/edition_test.go
git commit -m "feat(config): add edition type definitions and default config"
```

---

## Task 2: 后端依赖检查逻辑

**Files:**
- Create: `internal/handler/admin/edition_deps.go`
- Test: `internal/handler/admin/edition_deps_test.go`

**Step 1: Write failing test**

```go
func TestCheckDependencies_AllHealthy(t *testing.T) {
    t.Parallel()
    
    // Mock Redis, Ollama, Qdrant
    status := checkAllDependencies()
    
    if status["redis"].Healthy == false {
        t.Fatal("redis should be healthy in test env")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run TestCheckDependencies -v`
Expected: FAIL（`checkAllDependencies` 未定义）

**Step 3: Write minimal implementation**

```go
// internal/handler/admin/edition_deps.go
package admin

import (
    "context"
    "fmt"
    "net/http"
    "time"
    
    "github.com/redis/go-redis/v9"
    "ai-gateway/internal/config"
)

type DependencyStatus struct {
    Name    string `json:"name"`
    Address string `json:"address"`
    Healthy bool   `json:"healthy"`
    Message string `json:"message"`
}

func checkAllDependencies() map[string]DependencyStatus {
    return map[string]DependencyStatus{
        "redis":  checkRedis(),
        "ollama": checkOllama(),
        "qdrant": checkQdrant(),
    }
}

func checkRedis() DependencyStatus {
    cfg := config.Get().Redis
    client := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Password: cfg.Password,
        DB:       cfg.DB,
    })
    
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    err := client.Ping(ctx).Err()
    
    return DependencyStatus{
        Name:    "Redis",
        Address: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Healthy: err == nil,
        Message: getStatusMessage(err),
    }
}

func checkOllama() DependencyStatus {
    cfg := config.Get().VectorCache
    baseURL := cfg.OllamaBaseURL
    
    if baseURL == "" {
        baseURL = "http://127.0.0.1:11434"
    }
    
    client := &http.Client{Timeout: 2 * time.Second}
    resp, err := client.Get(baseURL + "/api/tags")
    
    healthy := err == nil && resp.StatusCode == 200
    if resp != nil {
        resp.Body.Close()
    }
    
    return DependencyStatus{
        Name:    "Ollama",
        Address: baseURL,
        Healthy: healthy,
        Message: getStatusMessage(err),
    }
}

func checkQdrant() DependencyStatus {
    cfg := config.Get().VectorCache
    qdrantURL := cfg.ColdVectorQdrantURL
    
    if qdrantURL == "" {
        return DependencyStatus{
            Name:    "Qdrant",
            Address: "未配置",
            Healthy: false,
            Message: "Qdrant URL 未配置",
        }
    }
    
    client := &http.Client{Timeout: 2 * time.Second}
    resp, err := client.Get(qdrantURL + "/collections")
    
    healthy := err == nil && resp.StatusCode == 200
    if resp != nil {
        resp.Body.Close()
    }
    
    return DependencyStatus{
        Name:    "Qdrant",
        Address: qdrantURL,
        Healthy: healthy,
        Message: getStatusMessage(err),
    }
}

func getStatusMessage(err error) string {
    if err == nil {
        return "正常"
    }
    return err.Error()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run TestCheckDependencies -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/admin/edition_deps.go internal/handler/admin/edition_deps_test.go
git commit -m "feat(admin): add dependency check logic for edition switching"
```

---

## Task 3: 后端版本管理 API

**Files:**
- Create: `internal/handler/admin/edition.go`
- Create: `internal/config/edition_manager.go`
- Test: `internal/handler/admin/edition_test.go`

**Step 1: Write failing test**

```go
func TestGetEdition_ShouldReturnCurrentConfig(t *testing.T) {
    t.Parallel()
    
    router := setupTestRouter()
    req := httptest.NewRequest("GET", "/api/admin/edition", nil)
    w := httptest.NewRecorder()
    
    router.ServeHTTP(w, req)
    
    if w.Code != http.StatusOK {
        t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
    }
    
    var resp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &resp)
    
    if !resp["success"].(bool) {
        t.Fatal("response should be successful")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run TestGetEdition -v`
Expected: FAIL（路由未注册）

**Step 3: Write minimal implementation**

```go
// internal/handler/admin/edition.go
package admin

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "ai-gateway/internal/config"
)

// GetEdition 获取当前版本配置
func GetEdition(c *gin.Context) {
    cfg := config.Get().GetEditionConfig()
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    cfg,
    })
}

// UpdateEdition 更新版本配置
func UpdateEdition(c *gin.Context) {
    var req struct {
        Type config.EditionType `json:"type" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "invalid_request",
            "message": "无效的请求参数",
        })
        return
    }
    
    // 验证版本类型
    if _, exists := config.EditionDefinitions[req.Type]; !exists {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "invalid_edition",
            "message": "无效的版本类型",
        })
        return
    }
    
    // 检查依赖
    missingDeps := checkDependencies(req.Type)
    if len(missingDeps) > 0 {
        c.JSON(http.StatusPreconditionFailed, gin.H{
            "success": false,
            "error":   "missing_dependencies",
            "message": "缺少必需的依赖服务",
            "data": gin.H{
                "missing": missingDeps,
            },
        })
        return
    }
    
    // 更新配置
    if err := config.UpdateEdition(req.Type); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   "update_failed",
            "message": "更新配置失败",
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "版本配置已更新，部分功能需要重启服务生效",
        "data": gin.H{
            "restart_required": true,
            "edition":          config.Get().GetEditionConfig(),
        },
    })
}

// GetEditionDefinitions 获取所有版本定义
func GetEditionDefinitions(c *gin.Context) {
    definitions := make([]config.EditionConfig, 0)
    for _, edition := range config.EditionDefinitions {
        definitions = append(definitions, edition)
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    definitions,
    })
}

// CheckDependencies 检查依赖服务状态
func CheckDependencies(c *gin.Context) {
    status := checkAllDependencies()
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    status,
    })
}

func checkDependencies(editionType config.EditionType) []string {
    definition := config.EditionDefinitions[editionType]
    missing := []string{}
    
    allStatus := checkAllDependencies()
    for _, dep := range definition.Dependencies {
        if status, exists := allStatus[dep]; exists && !status.Healthy {
            missing = append(missing, dep)
        }
    }
    
    return missing
}
```

```go
// internal/config/edition_manager.go
package config

import (
    "encoding/json"
    "fmt"
    "os"
    "sync"
)

var (
    editionMutex sync.RWMutex
)

func UpdateEdition(newEdition EditionType) error {
    editionMutex.Lock()
    defer editionMutex.Unlock()
    
    // 验证版本
    if _, exists := EditionDefinitions[newEdition]; !exists {
        return fmt.Errorf("invalid edition: %s", newEdition)
    }
    
    // 读取当前配置
    cfg := Get()
    
    // 更新版本
    if cfg.Edition == nil {
        cfg.Edition = &EditionConfigWrapper{}
    }
    cfg.Edition.Type = string(newEdition)
    
    // 根据版本自动设置功能开关
    definition := EditionDefinitions[newEdition]
    cfg.VectorCache.Enabled = definition.Features.VectorCache
    
    // 写入配置文件
    if err := saveConfig(cfg); err != nil {
        return err
    }
    
    // 重新加载配置
    return Reload()
}

func saveConfig(cfg *Config) error {
    configPath := os.Getenv("CONFIG_PATH")
    if configPath == "" {
        configPath = "./configs/config.json"
    }
    
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(configPath, data, 0644)
}

type EditionConfigWrapper struct {
    Type string `json:"type"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run TestGetEdition -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/admin/edition.go internal/config/edition_manager.go internal/handler/admin/edition_test.go
git commit -m "feat(admin): add edition management API endpoints"
```

---

## Task 4: 后端路由注册

**Files:**
- Modify: `internal/handler/admin/admin.go`

**Step 1: Write minimal implementation**

```go
// 在 RegisterRoutes 函数中添加
admin.GET("/edition", GetEdition)
admin.PUT("/edition", UpdateEdition)
admin.GET("/edition/definitions", GetEditionDefinitions)
admin.GET("/edition/dependencies", CheckDependencies)
```

**Step 2: Verify**

Run: `go build ./cmd/gateway`
Expected: 编译成功

**Step 3: Commit**

```bash
git add internal/handler/admin/admin.go
git commit -m "feat(admin): register edition API routes"
```

---

## Task 5: 前端版本 Store

**Files:**
- Create: `web/src/store/domain/edition.ts`
- Test: `web/src/store/domain/edition.test.ts`

**Step 1: Write failing test**

```typescript
describe('edition store', () => {
  it('should fetch edition config', async () => {
    const store = useEditionStore()
    
    await store.fetchEditionConfig()
    
    expect(store.config).toBeDefined()
    expect(store.config?.type).toBe('basic')
  })
  
  it('should compute hasVectorCache correctly', () => {
    const store = useEditionStore()
    
    store.config = {
      type: 'standard',
      features: {
        vector_cache: true,
        vector_db_management: false,
        knowledge_base: false,
        cold_hot_tiering: false
      },
      display_name: '标准版',
      description: '',
      dependencies: []
    }
    
    expect(store.hasVectorCache).toBe(true)
    expect(store.hasVectorDBManagement).toBe(false)
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit edition.test.ts`
Expected: FAIL（文件不存在）

**Step 3: Write minimal implementation**

```typescript
// web/src/store/domain/edition.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export type EditionType = 'basic' | 'standard' | 'enterprise'

export interface EditionConfig {
  type: EditionType
  features: {
    vector_cache: boolean
    vector_db_management: boolean
    knowledge_base: boolean
    cold_hot_tiering: boolean
  }
  display_name: string
  description: string
  dependencies: string[]
}

export interface DependencyStatus {
  name: string
  address: string
  healthy: boolean
  message: string
}

export const useEditionStore = defineStore('edition', () => {
  const config = ref<EditionConfig | null>(null)
  const definitions = ref<EditionConfig[]>([])
  const dependencies = ref<Record<string, DependencyStatus>>({})
  const loading = ref(false)
  const updating = ref(false)

  const isBasic = computed(() => config.value?.type === 'basic')
  const isStandard = computed(() => config.value?.type === 'standard')
  const isEnterprise = computed(() => config.value?.type === 'enterprise')
  
  const hasVectorCache = computed(() => config.value?.features.vector_cache ?? false)
  const hasVectorDBManagement = computed(() => config.value?.features.vector_db_management ?? false)
  const hasKnowledgeBase = computed(() => config.value?.features.knowledge_base ?? false)
  const hasColdHotTiering = computed(() => config.value?.features.cold_hot_tiering ?? false)

  async function fetchEditionConfig() {
    loading.value = true
    try {
      const response = await api.get('/api/admin/edition')
      config.value = response.data.data
    } catch (error) {
      config.value = {
        type: 'basic',
        features: {
          vector_cache: false,
          vector_db_management: false,
          knowledge_base: false,
          cold_hot_tiering: false
        },
        display_name: '基础版',
        description: '纯AI网关功能',
        dependencies: ['redis']
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchDefinitions() {
    try {
      const response = await api.get('/api/admin/edition/definitions')
      definitions.value = response.data.data
    } catch (error) {
      console.error('Failed to fetch edition definitions:', error)
    }
  }

  async function checkDependencies() {
    try {
      const response = await api.get('/api/admin/edition/dependencies')
      dependencies.value = response.data.data
    } catch (error) {
      console.error('Failed to check dependencies:', error)
    }
  }

  async function updateEdition(newEdition: EditionType) {
    updating.value = true
    try {
      const response = await api.put('/api/admin/edition', { type: newEdition })
      config.value = response.data.data.edition
      
      return {
        success: true,
        restartRequired: response.data.data.restart_required,
        message: response.data.message
      }
    } catch (error: any) {
      const errorData = error.response?.data
      
      if (errorData?.error === 'missing_dependencies') {
        return {
          success: false,
          missingDependencies: errorData.data.missing,
          message: errorData.message
        }
      }
      
      return {
        success: false,
        message: errorData?.message || '更新失败'
      }
    } finally {
      updating.value = false
    }
  }

  return {
    config,
    definitions,
    dependencies,
    loading,
    updating,
    isBasic,
    isStandard,
    isEnterprise,
    hasVectorCache,
    hasVectorDBManagement,
    hasKnowledgeBase,
    hasColdHotTiering,
    fetchEditionConfig,
    fetchDefinitions,
    checkDependencies,
    updateEdition
  }
})
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit edition.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/store/domain/edition.ts web/src/store/domain/edition.test.ts
git commit -m "feat(frontend): add edition store with pinia"
```

---

## Task 6: 前端版本选择组件

**Files:**
- Create: `web/src/views/settings/components/EditionSelector.vue`
- Test: `web/src/views/settings/components/EditionSelector.test.ts`

**Step 1: Write failing test**

```typescript
describe('EditionSelector', () => {
  it('should render edition cards', () => {
    const wrapper = mount(EditionSelector, {
      global: {
        plugins: [createTestingPinia()]
      }
    })
    
    expect(wrapper.findAll('.edition-card')).toHaveLength(3)
  })
  
  it('should disable card when dependencies missing', async () => {
    const wrapper = mount(EditionSelector, {
      global: {
        plugins: [createTestingPinia({
          initialState: {
            edition: {
              dependencies: {
                qdrant: { healthy: false }
              }
            }
          }
        })]
      }
    })
    
    const enterpriseCard = wrapper.findAll('.edition-card')[2]
    expect(enterpriseCard.classes()).toContain('disabled')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit EditionSelector.test.ts`
Expected: FAIL（组件不存在）

**Step 3: Write minimal implementation**

```vue
<template>
  <div class="edition-selector">
    <div class="section-title">版本管理</div>
    
    <el-alert
      v-if="restartRequired"
      type="warning"
      :closable="false"
      class="restart-alert"
    >
      <template #title>
        <el-icon><WarningFilled /></el-icon>
        版本配置已更新，部分功能需要重启服务生效
      </template>
      <el-button type="primary" size="small" @click="handleRestart">
        重启服务
      </el-button>
    </el-alert>

    <div class="edition-cards">
      <div
        v-for="edition in editionStore.definitions"
        :key="edition.type"
        class="edition-card"
        :class="{ 
          active: selectedEdition === edition.type,
          disabled: !canSelectEdition(edition.type)
        }"
        @click="handleSelectEdition(edition.type)"
      >
        <div class="edition-header">
          <el-radio 
            :model-value="selectedEdition" 
            :label="edition.type"
            :disabled="!canSelectEdition(edition.type)"
          >
            {{ edition.display_name }}
          </el-radio>
          <el-tag 
            v-if="edition.type === 'enterprise'" 
            type="danger" 
            size="small"
          >
            推荐
          </el-tag>
        </div>
        
        <div class="edition-description">{{ edition.description }}</div>
        
        <div class="edition-features">
          <div 
            v-for="feature in getFeatureLabels(edition.features)" 
            :key="feature"
            class="feature-item"
          >
            <el-icon color="#67c23a"><CircleCheckFilled /></el-icon>
            <span>{{ feature }}</span>
          </div>
        </div>

        <div class="edition-dependencies">
          <div class="deps-title">依赖服务：</div>
          <div class="deps-list">
            <el-tag
              v-for="dep in edition.dependencies"
              :key="dep"
              :type="getDependencyStatus(dep) ? 'success' : 'info'"
              size="small"
            >
              {{ dep.toUpperCase() }}
            </el-tag>
          </div>
        </div>
      </div>
    </div>

    <div class="dependency-status">
      <div class="status-title">依赖服务状态</div>
      <div class="status-grid">
        <div
          v-for="(status, key) in editionStore.dependencies"
          :key="key"
          class="status-item"
        >
          <el-icon :color="status.healthy ? '#67c23a' : '#f56c6c'">
            <CircleCheckFilled v-if="status.healthy" />
            <CircleCloseFilled v-else />
          </el-icon>
          <div class="status-info">
            <div class="status-name">{{ status.name }}</div>
            <div class="status-address">{{ status.address }}</div>
          </div>
        </div>
      </div>
    </div>

    <div class="actions">
      <el-button 
        type="primary" 
        :loading="editionStore.updating"
        :disabled="selectedEdition === editionStore.config?.type"
        @click="handleSave"
      >
        保存配置
      </el-button>
      <el-button @click="handleReset">重置为默认</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useEditionStore } from '@/store/domain/edition'
import type { EditionType, EditionFeatures } from '@/store/domain/edition'
import { WarningFilled, CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue'

const editionStore = useEditionStore()

const selectedEdition = ref<EditionType>('basic')
const restartRequired = ref(false)

onMounted(async () => {
  await Promise.all([
    editionStore.fetchEditionConfig(),
    editionStore.fetchDefinitions(),
    editionStore.checkDependencies()
  ])
  
  selectedEdition.value = editionStore.config?.type || 'basic'
})

const featureLabels: Record<keyof EditionFeatures, string> = {
  vector_cache: '语义级缓存（向量检索）',
  vector_db_management: '向量数据库管理界面',
  knowledge_base: '知识库管理',
  cold_hot_tiering: '冷热分层架构'
}

function getFeatureLabels(features: EditionFeatures): string[] {
  const labels: string[] = ['多服务商接入（7家）', '智能路由、限额管控']
  
  if (!features.vector_cache) {
    labels.push('基础缓存（哈希去重）')
  }
  
  Object.entries(features).forEach(([key, enabled]) => {
    if (enabled) {
      labels.push(featureLabels[key as keyof EditionFeatures])
    }
  })
  
  return labels
}

function getDependencyStatus(dep: string): boolean {
  return editionStore.dependencies[dep]?.healthy ?? false
}

function canSelectEdition(edition: EditionType): boolean {
  const editionDeps = editionStore.definitions.find(e => e.type === edition)?.dependencies || []
  return editionDeps.every(dep => getDependencyStatus(dep))
}

async function handleSelectEdition(edition: EditionType) {
  if (!canSelectEdition(edition)) {
    const editionConfig = editionStore.definitions.find(e => e.type === edition)
    const missingDeps = editionConfig?.dependencies.filter(dep => !getDependencyStatus(dep)) || []
    
    ElMessage.warning(`缺少依赖服务：${missingDeps.join(', ').toUpperCase()}`)
    return
  }
  
  selectedEdition.value = edition
}

async function handleSave() {
  const result = await editionStore.updateEdition(selectedEdition.value)
  
  if (result.success) {
    ElMessage.success(result.message)
    
    if (result.restartRequired) {
      restartRequired.value = true
    }
    
    setTimeout(() => {
      window.location.reload()
    }, 1500)
  } else {
    if (result.missingDependencies) {
      ElMessage.error(`缺少依赖服务：${result.missingDependencies.join(', ').toUpperCase()}`)
    } else {
      ElMessage.error(result.message)
    }
  }
}

function handleReset() {
  selectedEdition.value = 'basic'
}

async function handleRestart() {
  try {
    await ElMessageBox.confirm(
      '确定要重启服务吗？重启期间服务将暂时不可用。',
      '重启服务',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    ElMessage.success('服务重启中，请稍候...')
    
    setTimeout(() => {
      window.location.reload()
    }, 5000)
  } catch (error) {
    // 用户取消
  }
}
</script>

<style scoped lang="scss">
.edition-selector {
  padding: 24px;
  
  .section-title {
    font-size: 18px;
    font-weight: 600;
    margin-bottom: 20px;
  }
  
  .restart-alert {
    margin-bottom: 20px;
  }
  
  .edition-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 16px;
    margin-bottom: 24px;
  }
  
  .edition-card {
    border: 2px solid #e4e7ed;
    border-radius: 8px;
    padding: 20px;
    cursor: pointer;
    transition: all 0.3s;
    
    &:hover:not(.disabled) {
      border-color: #409eff;
      box-shadow: 0 2px 12px rgba(64, 158, 255, 0.2);
    }
    
    &.active {
      border-color: #409eff;
      background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
    }
    
    &.disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }
  
  .edition-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }
  
  .edition-description {
    color: #606266;
    font-size: 14px;
    margin-bottom: 16px;
  }
  
  .edition-features {
    margin-bottom: 16px;
    
    .feature-item {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;
      font-size: 13px;
    }
  }
  
  .edition-dependencies {
    .deps-title {
      font-size: 12px;
      color: #909399;
      margin-bottom: 8px;
    }
    
    .deps-list {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
    }
  }
  
  .dependency-status {
    margin-bottom: 24px;
    
    .status-title {
      font-size: 14px;
      font-weight: 600;
      margin-bottom: 12px;
    }
    
    .status-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 12px;
    }
    
    .status-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px;
      background: #f5f7fa;
      border-radius: 6px;
    }
    
    .status-info {
      .status-name {
        font-weight: 500;
        font-size: 14px;
      }
      
      .status-address {
        font-size: 12px;
        color: #909399;
      }
    }
  }
  
  .actions {
    display: flex;
    gap: 12px;
  }
}
</style>
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit EditionSelector.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/settings/components/EditionSelector.vue web/src/views/settings/components/EditionSelector.test.ts
git commit -m "feat(settings): add edition selector component"
```

---

## Task 7: 设置页面集成

**Files:**
- Modify: `web/src/views/settings/index.vue`

**Step 1: Write minimal implementation**

```vue
<template>
  <div class="settings-page">
    <el-tabs v-model="activeTab" class="settings-tabs">
      <el-tab-pane label="基础配置" name="basic">
        <BasicSettings />
      </el-tab-pane>
      
      <el-tab-pane label="认证设置" name="auth">
        <AuthSettings />
      </el-tab-pane>
      
      <el-tab-pane label="版本管理" name="edition">
        <EditionSelector />
      </el-tab-pane>
      
      <el-tab-pane label="高级配置" name="advanced">
        <AdvancedSettings />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import BasicSettings from './components/BasicSettings.vue'
import AuthSettings from './components/AuthSettings.vue'
import EditionSelector from './components/EditionSelector.vue'
import AdvancedSettings from './components/AdvancedSettings.vue'

const activeTab = ref('basic')
</script>
```

**Step 2: Verify**

Run: `cd web && npm run typecheck`
Expected: 类型检查通过

**Step 3: Commit**

```bash
git add web/src/views/settings/index.vue
git commit -m "feat(settings): integrate edition selector into settings page"
```

---

## Task 8: 动态菜单配置

**Files:**
- Create: `web/src/components/Layout/menu-config.ts`
- Test: `web/src/components/Layout/menu-config.test.ts`

**Step 1: Write failing test**

```typescript
describe('getMenuItems', () => {
  it('should return all basic menus', () => {
    const edition: EditionConfig = {
      type: 'basic',
      features: {
        vector_cache: false,
        vector_db_management: false,
        knowledge_base: false,
        cold_hot_tiering: false
      },
      display_name: '基础版',
      description: '',
      dependencies: []
    }
    
    const menus = getMenuItems(edition)
    
    expect(menus).toHaveLength(12)
    expect(menus.find(m => m.path === '/dashboard')).toBeDefined()
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit menu-config.test.ts`
Expected: FAIL（文件不存在）

**Step 3: Write minimal implementation**

```typescript
// web/src/components/Layout/menu-config.ts
import type { EditionConfig } from '@/store/domain/edition'

export interface MenuItem {
  path: string
  title: string
  icon: string
  requiredFeature?: keyof EditionConfig['features']
}

const ALL_MENUS: MenuItem[] = [
  { path: '/dashboard', title: '监控仪表盘', icon: 'Monitor' },
  { path: '/ops', title: '运维监控', icon: 'Operation' },
  { path: '/chat', title: 'AI 对话', icon: 'ChatDotRound' },
  { path: '/api-management', title: 'API 管理', icon: 'Connection' },
  { path: '/model-management', title: '模型管理', icon: 'Collection' },
  { path: '/providers-accounts', title: '账号与限额', icon: 'Key' },
  { path: '/usage', title: 'API 使用统计', icon: 'DataLine' },
  { path: '/trace', title: '请求链路追踪', icon: 'Share' },
  { path: '/routing', title: '路由策略', icon: 'Guide' },
  { path: '/cache', title: '缓存管理', icon: 'Box' },
  { path: '/alerts', title: '告警管理', icon: 'Bell' },
  { path: '/settings', title: '系统设置', icon: 'Setting' }
]

export function getMenuItems(edition: EditionConfig | null): MenuItem[] {
  if (!edition) {
    return ALL_MENUS
  }
  
  return ALL_MENUS.filter(menu => {
    if (menu.requiredFeature) {
      return edition.features[menu.requiredFeature]
    }
    return true
  })
}
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit menu-config.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/components/Layout/menu-config.ts web/src/components/Layout/menu-config.test.ts
git commit -m "feat(layout): add dynamic menu configuration"
```

---

## Task 9: Layout 组件改造

**Files:**
- Modify: `web/src/components/Layout/index.vue`

**Step 1: Modify implementation**

```vue
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useEditionStore } from '@/store/domain/edition'
import { getMenuItems } from './menu-config'

const editionStore = useEditionStore()

const menuItems = computed(() => getMenuItems(editionStore.config))

const showVectorDBEntry = computed(() => editionStore.hasVectorDBManagement)
const showKnowledgeEntry = computed(() => editionStore.hasKnowledgeBase)

const editionBadge = computed(() => {
  if (!editionStore.config) return null
  const badges = {
    basic: { text: '基础版', color: '#909399' },
    standard: { text: '标准版', color: '#67c23a' },
    enterprise: { text: '企业版', color: '#409eff' }
  }
  return badges[editionStore.config.type]
})

onMounted(async () => {
  await editionStore.fetchEditionConfig()
})
</script>

<template>
  <el-container class="layout-container">
    <el-aside class="sidebar" :class="{ 'is-collapsed': isCollapse }">
      <div class="logo">
        <div class="logo-icon">
          <el-icon :size="24"><Platform /></el-icon>
        </div>
        <transition name="fade">
          <span v-show="!isCollapse" class="logo-text">
            AI Gateway
            <el-tag 
              v-if="editionBadge" 
              :color="editionBadge.color" 
              size="small" 
              class="edition-tag"
            >
              {{ editionBadge.text }}
            </el-tag>
          </span>
        </transition>
      </div>

      <nav class="sidebar-nav">
        <el-tooltip
          v-for="item in menuItems"
          :key="item.path"
          :content="item.title"
          placement="right"
          :disabled="!isCollapse"
        >
          <router-link
            :to="item.path"
            class="nav-item"
            :class="{ active: isActive(item.path) }"
          >
            <el-icon :size="20"><component :is="item.icon" /></el-icon>
            <transition name="fade">
              <span v-show="!isCollapse" class="nav-text">{{ item.title }}</span>
            </transition>
          </router-link>
        </el-tooltip>
      </nav>
    </el-aside>

    <el-container class="main-container">
      <el-header class="header glass-header">
        <div class="header-right">
          <!-- 向量管理入口 - 仅企业版 -->
          <el-tooltip 
            v-if="showVectorDBEntry"
            content="向量数据独立界面 (新窗口)" 
            placement="bottom"
          >
            <a :href="vectorDBConsoleURL" target="_blank" class="vector-db-btn">
              <el-icon :size="18"><DataAnalysis /></el-icon>
              <span class="vector-db-text">向量管理</span>
              <span class="external-badge">↗</span>
            </a>
          </el-tooltip>

          <!-- 知识库入口 - 仅企业版 -->
          <el-tooltip 
            v-if="showKnowledgeEntry"
            content="知识库独立界面 (新窗口)" 
            placement="bottom"
          >
            <a :href="knowledgeConsoleURL" target="_blank" class="knowledge-btn">
              <el-icon :size="18"><Document /></el-icon>
              <span class="knowledge-text">知识库</span>
              <span class="external-badge">↗</span>
            </a>
          </el-tooltip>

          <!-- 其他 header 内容保持不变 -->
        </div>
      </el-header>
      
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped lang="scss">
.edition-tag {
  margin-left: 8px;
  font-size: 11px;
  padding: 2px 6px;
  border: none;
  color: white;
}
</style>
```

**Step 2: Verify**

Run: `cd web && npm run typecheck`
Expected: 类型检查通过

**Step 3: Commit**

```bash
git add web/src/components/Layout/index.vue
git commit -m "feat(layout): add edition-aware menu and header buttons"
```

---

## Task 10: 路由守卫

**Files:**
- Create: `web/src/router/guards/edition-guard.ts`
- Modify: `web/src/router/index.ts`
- Test: `web/src/router/guards/edition-guard.test.ts`

**Step 1: Write failing test**

```typescript
describe('edition guard', () => {
  it('should block vector-db route for basic edition', async () => {
    const router = createRouter()
    
    setupEditionGuard(router)
    
    const result = await router.push('/vector-db/collections')
    
    expect(result).toBeUndefined()
    expect(router.currentRoute.value.path).toBe('/dashboard')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit edition-guard.test.ts`
Expected: FAIL（守卫未实现）

**Step 3: Write minimal implementation**

```typescript
// web/src/router/guards/edition-guard.ts
import type { Router } from 'vue-router'
import { useEditionStore } from '@/store/domain/edition'

const VERSION_REQUIRED_ROUTES: Record<string, string[]> = {
  '/vector-db': ['enterprise'],
  '/knowledge': ['enterprise']
}

export function setupEditionGuard(router: Router) {
  router.beforeEach(async (to, from, next) => {
    const editionStore = useEditionStore()
    
    if (!editionStore.config) {
      await editionStore.fetchEditionConfig()
    }
    
    for (const [pathPrefix, allowedEditions] of Object.entries(VERSION_REQUIRED_ROUTES)) {
      if (to.path.startsWith(pathPrefix)) {
        if (!allowedEditions.includes(editionStore.config?.type || 'basic')) {
          next({
            path: '/dashboard',
            query: { error: 'edition_required' }
          })
          return
        }
      }
    }
    
    next()
  })
}
```

```typescript
// web/src/router/index.ts
import { setupEditionGuard } from './guards/edition-guard'

// 在路由创建后添加
setupEditionGuard(router)

export default router
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit edition-guard.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/router/guards/edition-guard.ts web/src/router/guards/edition-guard.test.ts web/src/router/index.ts
git commit -m "feat(router): add edition-based route guard"
```

---

## Task 11: 缓存页面适配

**Files:**
- Modify: `web/src/views/cache/index.vue`

**Step 1: Write minimal implementation**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useEditionStore } from '@/store/domain/edition'

const editionStore = useEditionStore()

const tabs = computed(() => {
  const baseTabs = [
    { name: 'overview', label: '概览' },
    { name: 'rules', label: '缓存规则' },
    { name: 'stats', label: '统计' }
  ]
  
  if (editionStore.hasVectorCache) {
    baseTabs.push({ name: 'semantic', label: '语义检索' })
  }
  
  if (editionStore.hasColdHotTiering) {
    baseTabs.push({ name: 'tiering', label: '冷热分层' })
  }
  
  return baseTabs
})
</script>
```

**Step 2: Verify**

Run: `cd web && npm run typecheck`
Expected: 类型检查通过

**Step 3: Commit**

```bash
git add web/src/views/cache/index.vue
git commit -m "feat(cache): add edition-aware tabs"
```

---

## Task 12: 集成测试与验证

**Files:**
- Test: `tests/integration/edition_test.go`

**Step 1: Write integration test**

```go
func TestEditionSwitch_FullFlow(t *testing.T) {
    // 1. 获取当前版本（基础版）
    // 2. 检查依赖
    // 3. 切换到标准版（需要 Ollama）
    // 4. 验证配置已更新
    // 5. 验证功能开关已生效
}
```

**Step 2: Run all tests**

```bash
# 后端
make test
make lint
make build

# 前端
cd web && npm run typecheck
cd web && npm run build
cd web && npm run test:unit
```

**Step 3: Manual verification**

- [ ] 基础版：12项菜单，无向量/知识库入口
- [ ] 标准版：缓存页面有"语义检索"tab
- [ ] 企业版：Header 显示向量管理/知识库按钮
- [ ] 版本切换：设置页面切换成功
- [ ] 依赖检查：缺少依赖时阻止切换

**Step 4: Commit**

```bash
git add tests/integration/edition_test.go
git commit -m "test: add edition integration tests"
```

---

## Task 13: 文档更新

**Files:**
- Create: `docs/EDITION-GUIDE.md`
- Update: `README.md`

**Step 1: Write documentation**

```markdown
# AI Gateway 版本管理指南

## 版本概述

AI Gateway 提供三个版本，满足不同规模场景的需求：

### 基础版
纯AI网关功能，轻量级部署。

**功能特性：**
- 多服务商接入（7家）
- 智能路由、限额管控
- 基础缓存（哈希去重）

**依赖服务：** Redis

### 标准版
网关 + 语义缓存，中大规模场景。

**功能特性：**
- 包含基础版全部功能
- 语义级缓存（向量检索）
- 需要本地 Ollama 生成向量

**依赖服务：** Redis + Ollama

### 企业版
完整功能，企业级生产环境。

**功能特性：**
- 包含标准版全部功能
- 向量数据库管理界面
- 冷热分层架构

**依赖服务：** Redis + Ollama + Qdrant

## 版本切换

1. 进入系统设置 → 版本管理
2. 选择目标版本
3. 检查依赖服务状态
4. 点击"保存配置"
5. 重启服务生效

## 功能对比

| 功能 | 基础版 | 标准版 | 企业版 |
|-----|-------|-------|-------|
| 多服务商接入 | ✅ | ✅ | ✅ |
| 智能路由 | ✅ | ✅ | ✅ |
| 基础缓存 | ✅ | ✅ | ✅ |
| 语义缓存 | ❌ | ✅ | ✅ |
| 向量数据库管理 | ❌ | ❌ | ✅ |
| 冷热分层 | ❌ | ❌ | ✅ |
| 知识库 | ❌ | ❌ | ✅ |
```

**Step 2: Commit**

```bash
git add docs/EDITION-GUIDE.md README.md
git commit -m "docs: add edition management guide"
```

---

## 验证清单

### 后端验证

- [ ] `GET /api/admin/edition` 返回正确配置
- [ ] `PUT /api/admin/edition` 更新成功
- [ ] `GET /api/admin/edition/definitions` 返回所有版本定义
- [ ] `GET /api/admin/edition/dependencies` 检查依赖状态
- [ ] 缺少依赖时返回 412 错误
- [ ] 配置文件正确更新
- [ ] 单元测试覆盖率 ≥ 80%
- [ ] `make lint` 通过
- [ ] `make test` 通过
- [ ] `make build` 成功

### 前端验证

- [ ] 版本 Store 正确获取配置
- [ ] 版本选择组件显示正确
- [ ] 依赖状态实时显示
- [ ] 保存后菜单动态刷新
- [ ] Logo 旁显示版本徽章
- [ ] 缓存页面 tab 根据版本显示
- [ ] Header 快捷入口根据版本显示
- [ ] 路由守卫正确拦截
- [ ] 单元测试覆盖率 ≥ 80%
- [ ] `npm run typecheck` 通过
- [ ] `npm run build` 成功
- [ ] `npm run test:unit` 通过

### 版本功能验证

**基础版**：
- [ ] 12项基础菜单
- [ ] 无向量管理入口
- [ ] 无知识库入口
- [ ] 缓存管理仅3个tab（概览/规则/统计）
- [ ] Logo 显示"基础版"徽章（灰色）

**标准版**：
- [ ] 12项基础菜单
- [ ] 缓存管理4个tab（增加"语义检索"）
- [ ] 无独立向量管理入口
- [ ] 无知识库入口
- [ ] Logo 显示"标准版"徽章（绿色）

**企业版**：
- [ ] 12项基础菜单
- [ ] 缓存管理5个tab（增加"冷热分层"）
- [ ] Header 显示"向量管理"快捷入口
- [ ] Header 显示"知识库"快捷入口
- [ ] 可访问 `/vector-db` 路由
- [ ] 可访问 `/knowledge` 路由
- [ ] Logo 显示"企业版"徽章（蓝色）

### 版本切换验证

- [ ] 基础版 → 标准版（需 Ollama）
- [ ] 标准版 → 企业版（需 Qdrant）
- [ ] 企业版 → 标准版（降级成功）
- [ ] 缺少依赖时阻止切换
- [ ] 切换后配置持久化
- [ ] 切换后菜单立即刷新

---

## 风险与回滚

### 风险点

1. **配置文件损坏**：版本切换时写入失败
   - 缓解：写入前备份，失败时回滚
   
2. **依赖检查误判**：网络抖动导致误判
   - 缓解：增加重试机制，设置超时
   
3. **前端状态不一致**：切换后未刷新
   - 缓解：强制刷新页面

### 回滚方案

```bash
# 1. 回滚配置文件
cp configs/config.json.backup configs/config.json

# 2. 重启服务
./scripts/dev-restart.sh

# 3. 清除浏览器缓存
# Chrome: Cmd+Shift+Delete

# 4. Git 回滚
git revert <commit-hash>
```

---

## 完成标准

仅当以下条件**全部满足**时，才能声明"已完成交付"：

1. ✅ 所有 Task 的测试通过
2. ✅ 后端 `make lint`、`make test`、`make build` 全部通过
3. ✅ 前端 `typecheck`、`build`、`test:unit` 全部通过
4. ✅ 基础版、标准版、企业版功能验证全部通过
5. ✅ 版本切换流程验证通过
6. ✅ 文档已更新
7. ✅ 工作区干净（`git status --short` 结果为空）

---

## 预估时间

- **后端开发**：3-4 小时（Task 1-4）
- **前端开发**：4-5 小时（Task 5-11）
- **集成测试**：1-2 小时（Task 12）
- **文档更新**：0.5 小时（Task 13）
- **总计**：8.5-11.5 小时（约 1.5-2 个工作日）

---

**创建日期**：2026-03-02  
**负责人**：OpenCode  
**状态**：待执行
