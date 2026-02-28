<template>
  <div class="cache-page">
    <!-- 改动点: 顶部 Hero 与全局操作 -->
    <div class="cache-hero">
      <div class="hero-main">
        <div class="hero-title">缓存管理</div>
        <div class="hero-subtitle">查看缓存命中、质量与任务类型策略，全链路观察缓存收益</div>
        <!-- UX: 常驻后端徽标，降低状态判断成本 -->
        <div class="backend-badge" :class="`backend-${cacheBackend.backend}`">
          <span class="badge-dot" />
          当前缓存后端：{{ cacheBackend.backend.toUpperCase() }}
        </div>
      </div>
      <div class="hero-actions">
        <el-button class="ghost-btn" @click="showWarmupDialog">
          <el-icon><Plus /></el-icon>
          预热缓存
        </el-button>
        <el-button class="ghost-btn" @click="exportCacheData">
          <el-icon><Download /></el-icon>
          导出
        </el-button>
        <el-button type="primary" @click="refreshAllCache">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 改动点: 统计概览卡片 -->
    <div class="stats-grid">
      <div v-for="stat in summaryStats" :key="stat.title" class="stat-card">
        <div class="stat-icon" :style="{ background: stat.color + '1A', color: stat.color }">
          <el-icon :size="22"><component :is="stat.icon" /></el-icon>
        </div>
        <div class="stat-body">
          <div class="stat-label">{{ stat.title }}</div>
          <div class="stat-value">{{ stat.value }}</div>
          <div class="stat-sub">{{ stat.subtitle }}</div>
        </div>
      </div>
    </div>

    <div v-if="cacheBackend.degraded || !cacheBackend.persistent" class="backend-alert">
      <div>
        <div class="backend-title">当前缓存后端为 Memory（非持久化），服务重启后缓存内容会丢失。</div>
        <div v-if="cacheBackend.reason" class="backend-sub">降级原因：{{ cacheBackend.reason }}</div>
      </div>
      <el-button
        v-if="!cacheBackend.persistent"
        type="warning"
        size="small"
        plain
        @click="showRedisRecoveryGuide"
      >
        <el-icon><Warning /></el-icon>
        恢复 Redis 指引
      </el-button>
    </div>

    <div class="panel types-section">
      <div class="panel-header">
        <div>
          <div class="panel-title">缓存类型</div>
          <div class="panel-subtitle">按类型管理内容、请求、路由与语义缓存</div>
        </div>
        <div class="panel-hint">指标展示：命中率 / 条目数</div>
      </div>

      <div class="type-grid">
        <div v-for="cacheType in cacheTypes" :key="cacheType.id" class="type-card" :data-tone="cacheType.tone">
          <div class="type-card-head">
            <div class="type-title">
              <div class="type-icon">
                <el-icon><component :is="cacheType.icon" /></el-icon>
              </div>
              <div>
                <div class="type-name">{{ cacheType.name }}</div>
                <div class="type-alias">{{ cacheType.alias }}</div>
              </div>
            </div>
            <div class="type-right">
              <el-tag size="small" :type="cacheType.enabled ? 'success' : 'info'">
                {{ cacheType.enabled ? '已启用' : '已禁用' }}
              </el-tag>
              <el-switch v-model="cacheType.enabled" size="small" @change="handleTypeChange(cacheType)" />
            </div>
          </div>
          <div class="type-desc">{{ cacheType.description }}</div>
          <div class="type-prefix">
            <span class="prefix-label">Key</span>
            <span class="prefix-value">{{ cacheType.prefix }}</span>
          </div>
          <div class="type-metrics">
            <div class="metric">
              <div class="metric-label">命中率</div>
              <div class="metric-value">{{ cacheType.hitRate }}%</div>
            </div>
            <div class="metric">
              <div class="metric-label">条目数</div>
              <div class="metric-value">{{ cacheType.entries }}</div>
              <div class="metric-sub">大小 {{ cacheType.size }}</div>
            </div>
          </div>
          <div class="type-progress">
            <div class="progress-meta">
              <span>命中率趋势</span>
              <span>{{ cacheType.hitRate }}%</span>
            </div>
            <el-progress
              :percentage="cacheType.hitRate"
              :color="getProgressColor(cacheType.hitRate)"
              :stroke-width="6"
              :show-text="false"
            />
          </div>
          <div class="type-actions">
            <el-button type="primary" link size="small" @click="clearCacheType(cacheType)">
              <el-icon><Delete /></el-icon> 清空
            </el-button>
            <el-button type="primary" link size="small" @click="viewCacheDetail(cacheType)">
              <el-icon><View /></el-icon> 详情
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <div class="panel signature-section">
      <div class="panel-header">
        <div>
          <div class="panel-title">语义签名命中观察</div>
          <div class="panel-subtitle">展示 0.5B 生成的高频语义签名，用于审计缓存命中原因</div>
        </div>
        <el-button link type="primary" @click="loadSemanticSignatures">刷新</el-button>
      </div>
      <el-table :data="semanticSignatures" size="small" max-height="240" empty-text="暂无语义签名数据">
        <el-table-column prop="signature" label="语义签名" min-width="220" show-overflow-tooltip />
        <el-table-column prop="task_type" label="任务类型" width="100" />
        <el-table-column prop="hit_count" label="命中" width="80" />
        <el-table-column prop="quality_score" label="质量" width="90">
          <template #default="{ row }">{{ Number(row.quality_score || 0).toFixed(1) }}</template>
        </el-table-column>
        <el-table-column prop="model" label="模型" min-width="130" show-overflow-tooltip />
      </el-table>
    </div>

    <div class="panel signature-section">
      <div class="panel-header">
        <div>
          <div class="panel-title">向量索引状态</div>
          <div class="panel-subtitle">Redis Stack 向量索引健康、维度与检索超时参数</div>
        </div>
        <div style="display: flex; gap: 8px">
          <el-button link type="primary" @click="loadVectorStats">刷新</el-button>
          <el-button type="warning" size="small" plain :loading="vectorRebuilding" @click="rebuildVectorCacheIndex">
            重建索引
          </el-button>
        </div>
      </div>
      <el-descriptions :column="2" border size="small">
        <el-descriptions-item label="向量启用">
          <el-tag :type="vectorStats.enabled ? 'success' : 'info'">{{ vectorStats.enabled ? '已启用' : '未启用' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="索引名">{{ vectorStats.index_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="键前缀">{{ vectorStats.key_prefix || '-' }}</el-descriptions-item>
        <el-descriptions-item label="向量维度">{{ vectorStats.dimension || '-' }}</el-descriptions-item>
        <el-descriptions-item label="查询超时">{{ vectorStats.query_timeout_ms || 0 }} ms</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="vectorStats.enabled ? 'success' : 'warning'">{{ vectorStats.message || (vectorStats.enabled ? 'ready' : 'disabled') }}</el-tag>
        </el-descriptions-item>
      </el-descriptions>
    </div>

    <div class="panel config-panel">
      <div class="panel-header">
        <div>
          <div class="panel-title">缓存策略面板</div>
          <div class="panel-subtitle">缓存内容优先，集中管理策略与规则</div>
        </div>
      </div>

      <div class="summary-grid">
        <div v-for="item in strategySummary" :key="item.title" class="summary-card">
          <div class="summary-title">{{ item.title }}</div>
          <div class="summary-value">{{ item.value }}</div>
          <div class="summary-sub">{{ item.subtitle }}</div>
        </div>
      </div>

      <el-tabs v-model="activeTab" class="cache-tabs">
            <el-tab-pane label="缓存内容" name="entries">
              <div class="entries-toolbar">
                <div class="entries-toolbar-left">
                  <!-- FIX: 筛选条件变更时重置分页，避免空页 -->
                  <el-select v-model="entriesFilter.type" placeholder="任务类型" clearable style="width: 150px" @change="handleEntriesTypeChange">
                    <el-option label="全部" value="" />
                    <el-option
                      v-for="taskType in CACHE_TASK_TYPE_OPTIONS"
                      :key="taskType.value"
                      :label="taskType.label"
                      :value="taskType.value"
                    />
                  </el-select>
                  <!-- FIX: 搜索条件变更时重置分页，避免空页 -->
                  <el-input v-model="entriesFilter.search" placeholder="搜索键名..." style="width: 250px" clearable @input="handleEntriesSearchInput">
                    <template #prefix><el-icon><Search /></el-icon></template>
                  </el-input>
                  <el-switch
                    v-model="entriesFilter.aggregate"
                    inline-prompt
                    active-text="聚合"
                    inactive-text="明细"
                    @change="handleEntriesModeChange"
                  />
                  <el-switch
                    v-model="entriesFilter.readableOnly"
                    inline-prompt
                    active-text="可读"
                    inactive-text="全部"
                    @change="handleEntriesModeChange"
                  />
                </div>
                <div class="entries-toolbar-right">
                  <el-button type="warning" plain @click="cleanupInvalidEntries">清理异常条目</el-button>
                  <el-button
                    type="danger"
                    plain
                    :disabled="selectedEntryKeys.length === 0"
                    @click="batchDeleteEntries"
                  >
                    批量删除（{{ selectedEntryKeys.length }}）
                  </el-button>
                </div>
              </div>

              <div v-if="entriesLoading" class="entries-skeleton">
                <el-skeleton v-for="row in 5" :key="row" animated>
                  <template #template>
                    <div class="skeleton-row">
                      <el-skeleton-item variant="text" style="width: 10%" />
                      <el-skeleton-item variant="text" style="width: 26%" />
                      <el-skeleton-item variant="text" style="width: 26%" />
                      <el-skeleton-item variant="text" style="width: 12%" />
                      <el-skeleton-item variant="text" style="width: 8%" />
                      <el-skeleton-item variant="text" style="width: 8%" />
                    </div>
                  </template>
                </el-skeleton>
              </div>

              <el-empty v-else-if="cacheEntries.length === 0" description="暂无缓存数据，发送 AI 请求后将自动生成缓存">
                <el-button type="primary" size="small" @click="showWarmupDialog">
                  <el-icon><Plus /></el-icon>
                  去预热缓存
                </el-button>
              </el-empty>

              <el-table v-else :data="cacheEntries" stripe class="entries-table" row-key="key" @selection-change="handleEntrySelectionChange">
                <el-table-column type="selection" width="46" />
                <el-table-column label="任务类型" width="100">
                  <template #default="{ row }">
                    <el-tag size="small" :type="getTaskTypeTag(row.task_type)">{{ getTaskTypeName(row.task_type) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="任务类型来源" width="110">
                  <template #default="{ row }">
                    <el-tag size="small" :type="getTaskTypeSourceTag(row.task_type_source)">{{ getTaskTypeSourceName(row.task_type_source) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="ID" width="120" show-overflow-tooltip>
                  <template #default="{ row }">
                    <code class="cache-id" :title="row.key">{{ getCacheId(row.key) }}</code>
                  </template>
                </el-table-column>
                <el-table-column label="用户消息" min-width="200">
                  <template #default="{ row }">
                    <div class="message-preview">{{ getUserMessage(row) }}</div>
                  </template>
                </el-table-column>
                <el-table-column label="AI 回复" min-width="200">
                  <template #default="{ row }">
                    <div class="message-preview">{{ getAIResponse(row) }}</div>
                  </template>
                </el-table-column>
                <el-table-column label="模型" width="120">
                  <template #default="{ row }">
                    <el-tag size="small">{{ row.model || '-' }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="hits" label="命中记录" width="90">
                  <template #default="{ row }">
                    <span class="hits-count">{{ row.hit_recorded ? (row.hits || 0) : '-' }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="条目数" width="88">
                  <template #default="{ row }">
                    <span>{{ row.group_count || 1 }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="TTL" width="80">
                  <template #default="{ row }">
                    <span>{{ formatTTL(row.ttl) }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="created_at" label="创建时间" width="160">
                  <template #default="{ row }">
                    {{ formatTime(row.created_at) }}
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="140" fixed="right">
                  <template #default="{ row }">
                    <el-button type="primary" link size="small" @click="viewEntryDetail(row)">详情</el-button>
                    <el-button type="danger" link size="small" @click="deleteEntry(row)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="pagination">
                <el-pagination
                  v-model:current-page="entriesFilter.page"
                  v-model:page-size="entriesFilter.pageSize"
                  :total="entriesTotal"
                  :page-sizes="[10, 20, 50, 100]"
                  layout="total, sizes, prev, pager, next"
                  @change="loadCacheEntries"
                />
              </div>
            </el-tab-pane>

            <el-tab-pane label="策略配置" name="general">
              <el-form :model="cacheConfig" label-width="140px" class="config-form">
                <el-form-item label="启用缓存">
                  <el-switch v-model="cacheConfig.enabled" @change="saveConfig" />
                </el-form-item>

                <el-form-item label="缓存策略">
                  <el-select v-model="cacheConfig.strategy" style="width: 100%" @change="saveConfig">
                    <el-option label="语义相似度" value="semantic" />
                    <el-option label="精确匹配" value="exact" />
                    <el-option label="前缀匹配" value="prefix" />
                  </el-select>
                </el-form-item>

                <el-form-item label="相似度阈值">
                  <el-row style="width: 100%" :gutter="16">
                    <el-col :span="18">
                      <el-slider v-model="cacheConfig.similarityThreshold" :min="0" :max="100" @change="saveConfig" />
                    </el-col>
                    <el-col :span="6">
                      <el-tag>{{ cacheConfig.similarityThreshold }}%</el-tag>
                    </el-col>
                  </el-row>
                </el-form-item>

                <el-form-item label="默认TTL">
                  <el-input-number v-model="cacheConfig.defaultTTLSeconds" :min="60" :max="86400" @change="saveConfig" />
                  <span class="unit-label">秒</span>
                </el-form-item>

                <el-form-item label="最大条目数">
                  <el-input-number v-model="cacheConfig.maxEntries" :min="100" :max="100000" :step="500" @change="saveConfig" />
                  <span class="unit-label">条</span>
                </el-form-item>

                <el-form-item label="淘汰策略">
                  <el-select v-model="cacheConfig.evictionPolicy" style="width: 100%" @change="saveConfig">
                    <el-option label="LRU (最近最少使用)" value="lru" />
                    <el-option label="LFU (最不经常使用)" value="lfu" />
                    <el-option label="FIFO (先进先出)" value="fifo" />
                  </el-select>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane label="任务类型 TTL" name="task-ttl">
              <div class="task-ttl-panel">
                <div class="task-ttl-header">
                  <div class="task-ttl-title">缓存策略（按任务类型 TTL）</div>
                  <el-button type="primary" size="small" @click="saveTTLConfig" :loading="ttlSaving">
                    <el-icon><Check /></el-icon>
                    保存
                  </el-button>
                </div>

                <el-alert type="info" :closable="false" show-icon style="margin-bottom: 12px">
                  <template #title>
                    按任务类型设置缓存过期时间（小时），将同步到全局缓存 TTL 策略。
                  </template>
                </el-alert>

                <div class="ttl-list">
                  <div v-for="item in ttlTaskTypeList" :key="item.key" class="ttl-item">
                    <div class="ttl-info">
                      <div class="ttl-name">{{ item.name }}</div>
                      <div class="ttl-desc">{{ item.description }}</div>
                    </div>
                    <el-input-number v-model="item.ttl" :min="0" :max="2160" :step="24" size="small" />
                  </div>
                </div>

                <el-button style="width: 100%; margin-top: 12px" @click="resetTTLConfig">
                  <el-icon><Refresh /></el-icon>
                  恢复默认
                </el-button>
              </div>
            </el-tab-pane>

            <el-tab-pane label="规则管理" name="rules">
              <div class="rules-header">
                <el-button type="primary" size="small" @click="showAddRuleDialog">
                  <el-icon><Plus /></el-icon>
                  添加规则
                </el-button>
              </div>

              <el-table :data="cacheRules" stripe class="rules-table">
                <el-table-column prop="pattern" label="匹配模式" min-width="200">
                  <template #default="{ row }">
                    <code class="pattern-code">{{ row.pattern }}</code>
                  </template>
                </el-table-column>
                <el-table-column prop="modelFilter" label="模型过滤" width="140">
                  <template #default="{ row }">
                    <el-tag size="small" v-if="row.modelFilter">{{ row.modelFilter }}</el-tag>
                    <span v-else class="text-muted">全部</span>
                  </template>
                </el-table-column>
                <el-table-column prop="ttl" label="TTL" width="100">
                  <template #default="{ row }">
                    {{ formatTTL(row.ttl) }}
                  </template>
                </el-table-column>
                <el-table-column prop="priority" label="优先级" width="100">
                  <template #default="{ row }">
                    <el-tag size="small" :type="getPriorityType(row.priority)">{{ getPriorityText(row.priority) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="enabled" label="状态" width="80">
                  <template #default="{ row }">
                    <el-switch v-model="row.enabled" size="small" @change="handleRuleChange(row)" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100" fixed="right">
                  <template #default="{ row }">
                    <el-button type="primary" link size="small" @click="editRule(row)">编辑</el-button>
                    <el-button type="danger" link size="small" @click="deleteRule(row)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>

            <el-tab-pane label="热门缓存" name="hot">
              <el-table :data="hotCaches" stripe class="hot-cache-table">
                <el-table-column prop="query" label="查询哈希" min-width="200">
                  <template #default="{ row }">
                    <div class="query-cell">
                      <el-icon><Key /></el-icon>
                      <code class="hash-code">{{ row.hash }}</code>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="model" label="模型" width="140">
                  <template #default="{ row }">
                    <el-tag size="small">{{ row.model }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="hits" label="命中次数" width="100" sortable>
                  <template #default="{ row }">
                    <span class="hits-count">{{ row.hits.toLocaleString() }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="size" label="大小" width="100">
                  <template #default="{ row }">
                    {{ row.size }}
                  </template>
                </el-table-column>
                <el-table-column prop="lastHit" label="最后命中" width="120">
                  <template #default="{ row }">
                    <span class="time-text">{{ row.lastHit }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="createdAt" label="创建时间" width="120">
                  <template #default="{ row }">
                    <span class="time-text">{{ row.createdAt }}</span>
                  </template>
                </el-table-column>
              </el-table>

              <div class="pagination">
                <el-pagination
                  v-model:current-page="hotCachePage"
                  v-model:page-size="hotCachePageSize"
                  :total="hotCacheTotal"
                  :page-sizes="[10, 20, 50]"
                  layout="total, sizes, prev, pager, next"
                />
              </div>
            </el-tab-pane>

            <el-tab-pane label="高级功能" name="advanced">
              <el-row :gutter="24">
                <!-- Redis 状态 -->
                <el-col :span="12">
                  <div class="advanced-section">
                    <h4><el-icon><Coin /></el-icon> Redis 缓存状态</h4>
                    <div class="status-card">
                      <div class="status-item">
                        <span class="label">连接状态</span>
                        <el-tag :type="redisHealth.status === 'healthy' ? 'success' : 'danger'">
                          {{ redisHealth.status === 'healthy' ? '已连接' : '未连接' }}
                        </el-tag>
                      </div>
                      <div class="status-item">
                        <span class="label">后端类型</span>
                        <span class="value">{{ cacheBackend.backend.toUpperCase() }}</span>
                      </div>
                      <div class="status-item">
                        <span class="label">持久化</span>
                        <el-tag :type="cacheBackend.persistent ? 'success' : 'danger'">
                          {{ cacheBackend.persistent ? '是' : '否（重启丢失）' }}
                        </el-tag>
                      </div>
                      <div class="status-item">
                        <span class="label">延迟</span>
                        <span class="value">{{ redisHealth.latency_ms || 0 }} ms</span>
                      </div>
                    </div>
                    <el-button type="primary" size="small" @click="runHealthCheck" style="margin-top: 12px">
                      <el-icon><Refresh /></el-icon> 健康检查
                    </el-button>
                    <el-alert :type="cacheBackend.persistent ? 'info' : 'error'" :closable="false" show-icon style="margin-top: 12px">
                      <template #title>
                        {{ cacheBackend.persistent ? '当前使用 Redis 持久化缓存' : '当前为 Memory 缓存，重启后将丢失' }}
                      </template>
                    </el-alert>
                  </div>
                </el-col>

                <!-- 请求去重 -->
                <el-col :span="12">
                  <div class="advanced-section">
                    <h4><el-icon><Connection /></el-icon> 请求去重配置</h4>
                    <el-form label-width="100px" size="small">
                      <el-form-item label="启用去重">
                        <el-switch v-model="dedupConfig.enabled" @change="saveDedupConfig" />
                      </el-form-item>
                      <el-form-item label="最大等待数">
                        <el-input-number v-model="dedupConfig.maxPending" :min="1" :max="100" @change="saveDedupConfig" />
                      </el-form-item>
                      <el-form-item label="请求超时">
                        <el-input-number v-model="dedupConfig.requestTimeout" :min="1" :max="300" @change="saveDedupConfig" />
                        <span class="unit-label">秒</span>
                      </el-form-item>
                    </el-form>
                    <el-alert type="success" :closable="false" show-icon style="margin-top: 12px">
                      <template #title>相同请求自动合并，减少重复调用</template>
                    </el-alert>
                  </div>
                </el-col>
              </el-row>

              <!-- 语义缓存 -->
              <div class="advanced-section" style="margin-top: 20px">
                <h4><el-icon><MagicStick /></el-icon> 语义缓存说明</h4>
                <el-descriptions :column="2" border>
                  <el-descriptions-item label="工作原理">
                    基于向量相似度匹配，相似请求可复用缓存结果
                  </el-descriptions-item>
                  <el-descriptions-item label="相似度阈值">
                    {{ cacheConfig.similarityThreshold }}% - 高于此值的请求将被视为相同
                  </el-descriptions-item>
                  <el-descriptions-item label="缓存策略">
                    {{ cacheConfig.strategy === 'semantic' ? '语义相似度' : cacheConfig.strategy === 'exact' ? '精确匹配' : '前缀匹配' }}
                  </el-descriptions-item>
                  <el-descriptions-item label="自动降级">
                    Redis 不可用时自动降级到内存缓存
                  </el-descriptions-item>
                </el-descriptions>
              </div>
            </el-tab-pane>

      </el-tabs>
    </div>

    <!-- 缓存内容详情对话框 -->
    <el-dialog v-model="entryDetailVisible" title="缓存内容详情" width="700px">
      <div v-if="entryDetail" class="entry-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="键名" :span="2">
            <code class="detail-key">{{ entryDetail.key }}</code>
          </el-descriptions-item>
          <el-descriptions-item label="类型">{{ entryDetail.type }}</el-descriptions-item>
          <el-descriptions-item label="任务类型">{{ getTaskTypeName(entryDetail.task_type) }}</el-descriptions-item>
          <el-descriptions-item label="任务类型来源">{{ getTaskTypeSourceName(entryDetail.task_type_source) }}</el-descriptions-item>
          <el-descriptions-item label="大小">{{ formatSize(entryDetail.size) }}</el-descriptions-item>
          <el-descriptions-item label="命中次数">{{ entryDetail.hits || 0 }}</el-descriptions-item>
          <el-descriptions-item label="TTL">{{ entryDetail.ttl === 0 ? '永不过期' : formatTTL(entryDetail.ttl) }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(entryDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="过期时间">{{ entryDetail.expires_at ? formatTime(entryDetail.expires_at) : '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="detail-value">
          <h4>用户消息</h4>
          <el-input
            type="textarea"
            :model-value="getUserMessageFull(entryDetail)"
            :rows="6"
            readonly
          />
        </div>

        <div class="detail-value">
          <h4>AI 回复</h4>
          <el-input
            type="textarea"
            :model-value="getAIResponseFull(entryDetail)"
            :rows="12"
            readonly
          />
        </div>

        <div v-if="entryDetail.model_stats && Object.keys(entryDetail.model_stats).length > 0" class="detail-value">
          <h4>实际命中模型</h4>
          <el-table :data="toModelStatsRows(entryDetail.model_stats)" size="small" border>
            <el-table-column prop="model" label="模型" min-width="220" />
            <el-table-column prop="count" label="条目数" width="100" align="right" />
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button @click="entryDetailVisible = false">关闭</el-button>
        <el-button type="danger" @click="deleteEntryAndClose">删除此缓存</el-button>
      </template>
    </el-dialog>

    <!-- 缓存预热对话框 -->
    <el-dialog v-model="warmupDialogVisible" title="缓存预热 - 添加测试缓存" width="600px">
      <el-form :model="warmupForm" :rules="warmupRules" ref="warmupFormRef" label-width="100px">
        <el-form-item label="任务类型" prop="task_type">
          <el-select v-model="warmupForm.task_type" placeholder="选择任务类型" style="width: 100%">
            <el-option
              v-for="taskType in CACHE_TASK_TYPE_OPTIONS"
              :key="taskType.value"
              :label="taskType.label"
              :value="taskType.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="用户消息" prop="user_message">
          <el-input v-model="warmupForm.user_message" type="textarea" :rows="3" placeholder="输入测试用户消息..." />
        </el-form-item>
        <el-form-item label="AI 回复" prop="ai_response">
          <el-input v-model="warmupForm.ai_response" type="textarea" :rows="4" placeholder="输入测试 AI 回复..." />
        </el-form-item>
        <el-form-item label="模型">
          <el-input v-model="warmupForm.model" placeholder="例如：gpt-4o" />
        </el-form-item>
        <el-form-item label="服务商">
          <el-input v-model="warmupForm.provider" placeholder="例如：openai" />
        </el-form-item>
        <el-form-item label="TTL (小时)">
          <el-input-number v-model="warmupForm.ttl" :min="0" :max="720" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="warmupDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitWarmup" :loading="warmupLoading">添加缓存</el-button>
      </template>
    </el-dialog>

    <!-- 添加/编辑规则对话框 -->
    <el-dialog v-model="ruleDialogVisible" :title="isEditRule ? '编辑缓存规则' : '添加缓存规则'" width="550px">
      <el-form :model="ruleForm" :rules="ruleFormRules" ref="ruleFormRef" label-width="120px">
        <el-form-item label="匹配模式" prop="pattern">
          <el-input v-model="ruleForm.pattern" placeholder="例如：chat:* 或 gpt-4:*" />
        </el-form-item>
        <el-form-item label="模型过滤">
          <el-select v-model="ruleForm.modelFilter" placeholder="选择模型（可选）" clearable style="width: 100%">
            <el-option label="所有模型" value="" />
            <el-option-group
              v-for="group in CACHE_RULE_MODEL_OPTIONS"
              :key="group.label"
              :label="group.label"
            >
              <el-option
                v-for="option in group.options"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="TTL">
          <el-row :gutter="10">
            <el-col :span="12">
              <el-input-number v-model="ruleForm.ttlValue" :min="1" style="width: 100%" />
            </el-col>
            <el-col :span="12">
              <el-select v-model="ruleForm.ttlUnit" style="width: 100%">
                <el-option label="秒" value="seconds" />
                <el-option label="分钟" value="minutes" />
                <el-option label="小时" value="hours" />
                <el-option label="天" value="days" />
              </el-select>
            </el-col>
          </el-row>
        </el-form-item>
        <el-form-item label="优先级">
          <el-select v-model="ruleForm.priority" style="width: 100%">
            <el-option label="高" value="high" />
            <el-option label="中" value="medium" />
            <el-option label="低" value="low" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRule">保存</el-button>
      </template>
    </el-dialog>

    <!-- 缓存详情对话框 -->
    <el-dialog v-model="detailDialogVisible" :title="cacheDetail?.name + ' 统计'" width="600px">
      <div v-if="cacheDetail" class="cache-detail">
        <el-row :gutter="20">
          <el-col :span="12">
            <div class="detail-item">
              <span class="label">命中率</span>
              <el-progress :percentage="cacheDetail.hitRate" :color="getProgressColor(cacheDetail.hitRate)" :stroke-width="10" />
            </div>
          </el-col>
          <el-col :span="12">
            <div class="detail-item">
              <span class="label">内存使用</span>
              <el-progress :percentage="cacheDetail.memoryUsage" color="#409eff" :stroke-width="10" />
            </div>
          </el-col>
        </el-row>

        <el-descriptions :column="2" border class="detail-desc">
          <el-descriptions-item label="总条目数">{{ cacheDetail.entries }}</el-descriptions-item>
          <el-descriptions-item label="总大小">{{ cacheDetail.size }}</el-descriptions-item>
          <el-descriptions-item label="总命中">{{ cacheDetail.totalHits?.toLocaleString() }}</el-descriptions-item>
          <el-descriptions-item label="总未命中">{{ cacheDetail.totalMisses?.toLocaleString() }}</el-descriptions-item>
          <el-descriptions-item label="平均响应">{{ cacheDetail.avgResponse }}</el-descriptions-item>
          <el-descriptions-item label="上次清理">{{ cacheDetail.lastCleared }}</el-descriptions-item>
        </el-descriptions>

        <div class="detail-chart">
          <h4>命中率趋势（最近24小时）</h4>
          <div ref="detailChartRef" class="chart-container"></div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  addTestCacheEntry,
  cleanupInvalidEntries as cleanupInvalidEntriesApi,
  clearCacheByType,
  createCacheRule,
  deleteCacheEntry,
  deleteCacheEntryGroup,
  deleteCacheRule,
  getCacheConfig,
  getCacheEntries,
  getCacheEntryDetail,
  getCacheHealth,
  getCacheRules,
  getCacheStats,
  getVectorStats,
  getSemanticSignatures,
  rebuildVectorIndex,
  getTtlConfig,
  updateCacheConfig,
  updateCacheRule,
  updateTtlConfig
} from '@/api/cache-domain'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'
import { API } from '@/constants/api'
import {
  CACHE_DEFAULT_TASK_TTL,
  CACHE_RULE_MODEL_OPTIONS,
  CACHE_TASK_TYPE_OPTIONS,
  CACHE_TASK_TTL_ITEMS,
  CACHE_WARMUP_DEFAULTS,
  type CacheTaskTTLItem
} from '@/constants/pages/cache'
import { buildCacheTypeCards, type CacheTypeCard, type CacheTypeState } from '@/utils/cache-type-meta'
import { extractAIResponseFull, extractUserMessageFull } from './cache-content-parser'
import * as echarts from 'echarts'

interface CacheTypeDetail extends CacheTypeCard {
  memoryUsage?: number
  totalHits?: number
  totalMisses?: number
  avgResponse?: string
  lastCleared?: string
}

interface CacheRule {
  id: number
  pattern: string
  modelFilter: string
  ttl: number
  priority: string
  enabled: boolean
}

interface HotCache {
  hash: string
  model: string
  hits: number
  size: string
  lastHit: string
  createdAt: string
}

const loading = ref(false)
const activeTab = ref('entries')
const ttlSaving = ref(false)
const ruleDialogVisible = ref(false)
const detailDialogVisible = ref(false)
const isEditRule = ref(false)
const ruleFormRef = ref<FormInstance>()
const detailChartRef = ref<HTMLElement>()
const hotCachePage = ref(1)
const hotCachePageSize = ref(10)
const hotCacheTotal = ref(100)
const cacheDetail = ref<CacheTypeDetail | null>(null)
let detailChart: echarts.ECharts | null = null
const vectorRebuilding = ref(false)

const overallStats = reactive({
  hitRate: 0,
  totalSize: '0 MB',
  totalEntries: 0,
  avgResponse: '0ms',
  tokenSavings: 0
})

const cacheTypes = ref<CacheTypeCard[]>(buildCacheTypeCards())

const cacheConfig = reactive({
  enabled: true,
  strategy: 'semantic',
  similarityThreshold: 92,
  defaultTTLSeconds: 1800,
  maxEntries: 10000,
  evictionPolicy: 'lru',
  vectorEnabled: false,
  vectorDimension: 1024,
  vectorQueryTimeoutMs: 1200,
  vectorThresholds: {
    calc: 0.97,
    translate: 0.96,
    weather: 0.95,
    qa: 0.93,
    chat: 0.92
  } as Record<string, number>
})

const ttlTaskTypeList = ref<CacheTaskTTLItem[]>(
  CACHE_TASK_TTL_ITEMS.map(item => ({ ...item }))
)

const redisHealth = reactive({
  status: 'unknown',
  backend: '',
  latency_ms: 0
})

const cacheBackend = reactive({
  backend: 'memory',
  persistent: false,
  degraded: false,
  reason: ''
})

const vectorStats = reactive({
  enabled: false,
  index_name: '',
  key_prefix: '',
  dimension: 0,
  query_timeout_ms: 0,
  message: ''
})

const dedupConfig = reactive({
  enabled: true,
  maxPending: 100,
  requestTimeout: 30
})

// 改动点: 首页概览与策略摘要
const summaryStats = computed(() => [
  {
    title: '整体命中率',
    value: `${overallStats.hitRate}%`,
    subtitle: '全量缓存',
    icon: 'CircleCheckFilled',
    color: '#2563eb'
  },
  {
    title: '缓存条目',
    value: formatCompactNumber(overallStats.totalEntries),
    subtitle: '当前存量',
    icon: 'Files',
    color: '#0ea5e9'
  },
  {
    title: '缓存体积',
    value: overallStats.totalSize,
    subtitle: cacheBackend.persistent ? 'Redis' : 'Memory',
    icon: 'Coin',
    color: '#f97316'
  },
  {
    title: '节省 Token',
    value: formatCompactNumber(overallStats.tokenSavings),
    subtitle: '请求/上下文',
    icon: 'TrendCharts',
    color: '#16a34a'
  },
  {
    title: '平均命中耗时',
    value: overallStats.avgResponse,
    subtitle: '缓存读写',
    icon: 'Timer',
    color: '#8b5cf6'
  }
])

const strategySummary = computed(() => [
  {
    title: '默认策略',
    value: strategyLabel(cacheConfig.strategy),
    subtitle: `阈值 ${cacheConfig.similarityThreshold}%`
  },
  {
    title: '默认 TTL',
    value: formatDuration(cacheConfig.defaultTTLSeconds),
    subtitle: `最大条目 ${formatCompactNumber(cacheConfig.maxEntries)}`
  },
  {
    title: '淘汰策略',
    value: cacheConfig.evictionPolicy.toUpperCase(),
    subtitle: cacheBackend.persistent ? '持久化缓存' : '非持久化缓存'
  },
  {
    title: '请求去重',
    value: dedupConfig.enabled ? '已开启' : '未开启',
    subtitle: `最大等待 ${dedupConfig.maxPending}`
  }
])

const cacheRules = ref<CacheRule[]>([])

const hotCaches = ref<HotCache[]>([])
const semanticSignatures = ref<Array<{ signature: string; task_type: string; hit_count: number; quality_score: number; model: string }>>([])

// 缓存内容管理相关
const cacheEntries = ref<any[]>([])
const selectedEntryKeys = ref<string[]>([])
const entriesLoading = ref(false)
const entriesTotal = ref(0)
const entryDetailVisible = ref(false)
const entryDetail = ref<any>(null)
const entriesFilter = reactive({
  type: '',
  search: '',
  page: 1,
  pageSize: 20,
  aggregate: true,
  readableOnly: true
})
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

// 缓存预热相关
const warmupDialogVisible = ref(false)
const warmupLoading = ref(false)
const warmupFormRef = ref()
const warmupForm = reactive({
  task_type: 'fact',
  user_message: '',
  ai_response: '',
  model: CACHE_WARMUP_DEFAULTS.model,
  provider: CACHE_WARMUP_DEFAULTS.provider,
  ttl: CACHE_DEFAULT_TASK_TTL.fact
})
const warmupRules = {
  task_type: [{ required: true, message: '请选择任务类型', trigger: 'change' }],
  user_message: [{ required: true, message: '请输入用户消息', trigger: 'blur' }],
  ai_response: [{ required: true, message: '请输入AI回复', trigger: 'blur' }]
}

const ruleForm = reactive({
  id: 0,
  pattern: '',
  modelFilter: '',
  ttlValue: 1,
  ttlUnit: 'hours',
  priority: 'medium',
  enabled: true
})

const ruleFormRules: FormRules = {
  pattern: [{ required: true, message: 'Please enter pattern', trigger: 'blur' }]
}

// 改动点: 概览与策略摘要的格式化辅助函数
const formatCompactNumber = (num: number): string => {
  if (!num) return '0'
  if (num >= 1_000_000_000) return `${(num / 1_000_000_000).toFixed(2)}B`
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(2)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`
  return num.toString()
}

const formatDuration = (seconds: number): string => {
  if (!seconds) return '0'
  if (seconds >= 86400) return `${Math.round(seconds / 86400)}天`
  if (seconds >= 3600) return `${Math.round(seconds / 3600)}小时`
  if (seconds >= 60) return `${Math.round(seconds / 60)}分钟`
  return `${seconds}秒`
}

const strategyLabel = (strategy: string): string => {
  if (strategy === 'semantic') return '语义相似度'
  if (strategy === 'exact') return '精确匹配'
  if (strategy === 'prefix') return '前缀匹配'
  return strategy
}

const getProgressColor = (percentage: number) => {
  if (percentage >= 80) return '#34C759'
  if (percentage >= 60) return '#007AFF'
  if (percentage >= 40) return '#FF9500'
  return '#FF3B30'
}

const getPriorityType = (priority: string) => {
  const types: Record<string, string> = {
    high: 'danger',
    medium: 'warning',
    low: 'info'
  }
  return types[priority] || 'info'
}

const getPriorityText = (priority: string) => {
  const texts: Record<string, string> = {
    high: '高',
    medium: '中',
    low: '低'
  }
  return texts[priority] || priority
}

const handleTypeChange = async (type: CacheTypeCard) => {
  try {
    await updateCacheConfig({
      [type.id]: { enabled: type.enabled }
    })
    ElMessage.success(`${type.name} 已${type.enabled ? '启用' : '禁用'}`)
  } catch (e) {
    type.enabled = !type.enabled
    handleApiError(e, '操作失败')
  }
}

const clearCacheType = async (type: CacheTypeCard) => {
  try {
    await ElMessageBox.confirm(`确定清空 ${type.name} 的所有缓存吗？`, '警告', { type: 'warning' })
    await clearCacheByType(type.id)
    type.entries = 0
    type.size = '0 MB'
    type.hitRate = 0
    ElMessage.success(`${type.name} 已清空`)
  } catch (e: any) {
    if (e !== 'cancel') {
      handleApiError(e, '清空失败')
    }
  }
}

const viewCacheDetail = (type: CacheTypeCard) => {
  cacheDetail.value = {
    ...type,
    memoryUsage: 0,
    totalHits: 0,
    totalMisses: 0,
    avgResponse: '0ms',
    lastCleared: '-'
  }
  detailDialogVisible.value = true

  nextTick(() => {
    initDetailChart()
  })
}

const initDetailChart = () => {
  if (!detailChartRef.value) return

  if (detailChart) {
    detailChart.dispose()
  }
  detailChart = echarts.init(detailChartRef.value)
  const hours = Array.from({ length: 24 }, (_, i) => `${23 - i}h ago`).reverse()
  const hitRate = cacheDetail.value?.hitRate || 0
  const data = Array.from({ length: 24 }, () => hitRate)

  detailChart.setOption({
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: hours,
      axisLabel: { fontSize: 10 }
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { formatter: '{value}%' }
    },
    series: [{
      data: data,
      type: 'line',
      smooth: true,
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(0, 122, 255, 0.3)' },
          { offset: 1, color: 'rgba(0, 122, 255, 0.05)' }
        ])
      },
      lineStyle: { color: '#007AFF' },
      itemStyle: { color: '#007AFF' }
    }]
  })
}

const refreshAllCache = async () => {
  await Promise.all([
    loadCacheStats(),
    loadCacheConfig(),
    loadTTLConfig(),
    loadCacheHealth(),
    loadCacheRules(),
    loadCacheEntries(),
    loadVectorStats()
  ])
  handleSuccess('缓存数据已刷新')
}

const saveConfig = async () => {
  try {
    await updateCacheConfig({
      enabled: cacheConfig.enabled,
      strategy: cacheConfig.strategy,
      similarity_threshold: cacheConfig.similarityThreshold / 100,
      default_ttl_seconds: cacheConfig.defaultTTLSeconds,
      max_entries: cacheConfig.maxEntries,
      eviction_policy: cacheConfig.evictionPolicy,
      vector_enabled: cacheConfig.vectorEnabled,
      vector_dimension: cacheConfig.vectorDimension,
      vector_query_timeout_ms: cacheConfig.vectorQueryTimeoutMs,
      vector_thresholds: cacheConfig.vectorThresholds
    })
    handleSuccess('配置已保存')
  } catch (e) {
    handleApiError(e, '保存失败')
  }
}

const showAddRuleDialog = () => {
  isEditRule.value = false
  Object.assign(ruleForm, {
    id: 0,
    pattern: '',
    modelFilter: '',
    ttlValue: 1,
    ttlUnit: 'hours',
    priority: 'medium',
    enabled: true
  })
  ruleDialogVisible.value = true
}

const editRule = (rule: CacheRule) => {
  isEditRule.value = true
  let ttlValue = rule.ttl
  let ttlUnit = 'seconds'

  if (rule.ttl >= 86400) {
    ttlValue = rule.ttl / 86400
    ttlUnit = 'days'
  } else if (rule.ttl >= 3600) {
    ttlValue = rule.ttl / 3600
    ttlUnit = 'hours'
  } else if (rule.ttl >= 60) {
    ttlValue = rule.ttl / 60
    ttlUnit = 'minutes'
  }

  Object.assign(ruleForm, {
    id: rule.id,
    pattern: rule.pattern,
    modelFilter: rule.modelFilter,
    ttlValue,
    ttlUnit,
    priority: rule.priority,
    enabled: rule.enabled
  })
  ruleDialogVisible.value = true
}

const deleteRule = async (rule: CacheRule) => {
  try {
    await ElMessageBox.confirm(`确定删除规则 "${rule.pattern}" 吗？`, '警告', { type: 'warning' })
    await deleteCacheRule(rule.id)
    handleSuccess('规则已删除')
    loadCacheRules()
  } catch (e: any) {
    if (e !== 'cancel') {
      handleApiError(e, '删除失败')
    }
  }
}

const handleRuleChange = async (rule: CacheRule) => {
  try {
    await updateCacheRule(rule.id, {
      enabled: rule.enabled
    })
    handleSuccess(`规则已${rule.enabled ? '启用' : '禁用'}`)
  } catch (e) {
    rule.enabled = !rule.enabled
    handleApiError(e, '操作失败')
  }
}

const submitRule = async () => {
  if (!ruleFormRef.value) return
  try {
    const valid = await ruleFormRef.value.validate()
    if (valid) {
      let ttl = ruleForm.ttlValue
      switch (ruleForm.ttlUnit) {
        case 'days': ttl *= 86400; break
        case 'hours': ttl *= 3600; break
        case 'minutes': ttl *= 60; break
      }

      if (isEditRule.value) {
        await updateCacheRule(ruleForm.id, {
          pattern: ruleForm.pattern,
          model_filter: ruleForm.modelFilter,
          ttl: ttl,
          priority: ruleForm.priority,
          enabled: ruleForm.enabled
        })
        handleSuccess('规则已更新')
      } else {
        await createCacheRule({
          pattern: ruleForm.pattern,
          model_filter: ruleForm.modelFilter,
          ttl: ttl,
          priority: ruleForm.priority,
          enabled: ruleForm.enabled
        })
        handleSuccess('规则已添加')
      }
      ruleDialogVisible.value = false
      loadCacheRules()
    }
  } catch (error) {
    handleApiError(error, '操作失败')
  }
}

async function loadCacheRules() {
  try {
    const data: any = await getCacheRules()
    if (Array.isArray(data)) {
      cacheRules.value = data.map((r: any) => ({
        id: r.id,
        pattern: r.pattern,
        modelFilter: r.model_filter || '',
        ttl: r.ttl,
        priority: r.priority,
        enabled: r.enabled
      }))
    }
  } catch (e) {
    console.warn('Failed to load cache rules:', e)
  }
}

async function loadCacheStats() {
  loading.value = true
  try {
    const stats: any = await getCacheStats()
    if (stats) {
      
      let totalHits = 0
      let totalOps = 0
      let totalEntries = 0
      let totalSizeBytes = 0
      let totalLatencyMs = 0
      
      const typeStats: Record<string, any> = {}
      
      for (const [key, value] of Object.entries(stats)) {
        if (key.endsWith('_cache') || key === 'request_cache' || key === 'context_cache' || key === 'route_cache' || key === 'usage_cache' || key === 'response_cache') {
          const typeName = key.replace('_cache', '')
          const stat = value as any
          typeStats[typeName] = stat
          totalHits += stat.hits || 0
          totalOps += (stat.hits || 0) + (stat.misses || 0)
          totalEntries += stat.entries || 0
          totalSizeBytes += stat.size_bytes || 0
          totalLatencyMs += (stat.avg_latency_ms || 0) * (stat.hits + stat.misses)
        }
      }

      const redisHitRate = stats.redis_hit_rate ?? stats.redisHitRate
      if (typeof redisHitRate === 'number') {
        overallStats.hitRate = Math.round(redisHitRate * 100)
      } else {
        overallStats.hitRate = totalOps > 0 ? Math.round((totalHits / totalOps) * 100) : 0
      }
      overallStats.totalEntries = totalEntries
      overallStats.totalSize = formatSize(totalSizeBytes)
      overallStats.avgResponse = totalOps > 0 ? `${Math.round(totalLatencyMs / totalOps)}ms` : '0ms'
      overallStats.tokenSavings = stats.token_savings ?? stats.tokenSavings ?? 0
      
      const nextStates: CacheTypeState[] = cacheTypes.value.map(type => {
        const stat = typeStats[type.id]
        if (stat) {
          const hits = stat.hits || 0
          const misses = stat.misses || 0
          const ops = hits + misses
          return {
            id: type.id,
            enabled: type.enabled,
            hitRate: ops > 0 ? Math.round((hits / ops) * 100) : 0,
            entries: stat.entries || 0,
            size: formatSize(stat.size_bytes || 0)
          }
        }
        return {
          id: type.id,
          enabled: type.enabled,
          hitRate: type.hitRate ?? 0,
          entries: type.entries ?? 0,
          size: type.size ?? '0 MB'
        }
      })
      cacheTypes.value = buildCacheTypeCards(nextStates)
    }
  } catch (e) {
    console.warn('Failed to load cache stats:', e)
  } finally {
    loading.value = false
  }
}

async function loadSemanticSignatures() {
  try {
    const data: any = await getSemanticSignatures(12)
    semanticSignatures.value = Array.isArray(data) ? data : []
  } catch (e) {
    console.warn('Failed to load semantic signatures:', e)
  }
}

async function loadVectorStats() {
  try {
    const stats: any = await getVectorStats()
    vectorStats.enabled = Boolean(stats?.enabled)
    vectorStats.index_name = stats?.index_name || ''
    vectorStats.key_prefix = stats?.key_prefix || ''
    vectorStats.dimension = Number(stats?.dimension || 0)
    vectorStats.query_timeout_ms = Number(stats?.query_timeout_ms || 0)
    vectorStats.message = stats?.message || ''
  } catch (e) {
    vectorStats.enabled = false
    vectorStats.message = '获取失败'
    console.warn('Failed to load vector stats:', e)
  }
}

async function rebuildVectorCacheIndex() {
  vectorRebuilding.value = true
  try {
    await rebuildVectorIndex()
    handleSuccess('向量索引重建成功')
    await loadVectorStats()
  } catch (e) {
    handleApiError(e, '向量索引重建失败')
  } finally {
    vectorRebuilding.value = false
  }
}

// 改动点: 兼容后端 snake_case 字段并换算相似度
async function loadCacheConfig() {
  try {
    const cfg: any = await getCacheConfig()
    if (cfg) {
      cacheConfig.enabled = cfg.enabled ?? true
      cacheConfig.strategy = cfg.strategy || 'semantic'
      const similarity = cfg.similarity_threshold ?? cfg.similarityThreshold ?? 0.92
      cacheConfig.similarityThreshold = Math.round(similarity * 100)
      cacheConfig.defaultTTLSeconds = cfg.default_ttl_seconds || cfg.defaultTTLSeconds || 1800
      cacheConfig.maxEntries = cfg.max_entries || cfg.maxEntries || 10000
      cacheConfig.evictionPolicy = cfg.eviction_policy || cfg.evictionPolicy || 'lru'
      cacheConfig.vectorEnabled = cfg.vector_enabled ?? cfg.vectorEnabled ?? false
      cacheConfig.vectorDimension = cfg.vector_dimension || cfg.vectorDimension || 1024
      cacheConfig.vectorQueryTimeoutMs = cfg.vector_query_timeout_ms || cfg.vectorQueryTimeoutMs || 1200
      cacheConfig.vectorThresholds = cfg.vector_thresholds || cfg.vectorThresholds || cacheConfig.vectorThresholds

      if (cfg.dedup) {
        dedupConfig.enabled = cfg.dedup.enabled ?? dedupConfig.enabled
        dedupConfig.maxPending = cfg.dedup.max_pending ?? dedupConfig.maxPending
        dedupConfig.requestTimeout = cfg.dedup.request_timeout_seconds ?? dedupConfig.requestTimeout
      }
    }
  } catch (e) {
    console.warn('Failed to load cache config:', e)
  }
}

async function loadTTLConfig() {
  try {
    const data: any = await getTtlConfig()
    if (data?.task_type_defaults) {
      const defaults = data.task_type_defaults as Record<string, number>
      ttlTaskTypeList.value = ttlTaskTypeList.value.map(item => ({
        ...item,
        ttl: defaults[item.key] ?? CACHE_DEFAULT_TASK_TTL[item.key] ?? 24
      }))
    }
  } catch (e) {
    console.warn('Failed to load TTL config:', e)
  }
}

async function saveTTLConfig() {
  ttlSaving.value = true
  try {
    const taskTypeDefaults: Record<string, number> = {}
    for (const item of ttlTaskTypeList.value) {
      taskTypeDefaults[item.key] = item.ttl
    }
    await updateTtlConfig({
      task_type_defaults: taskTypeDefaults
    })
    handleSuccess('任务类型 TTL 配置已保存')
  } catch (e) {
    handleApiError(e, '保存失败')
  } finally {
    ttlSaving.value = false
  }
}

function resetTTLConfig() {
  ttlTaskTypeList.value = ttlTaskTypeList.value.map(item => ({
    ...item,
    ttl: CACHE_DEFAULT_TASK_TTL[item.key] ?? 24
  }))
  handleSuccess('已恢复默认配置')
}

// 改动点: 读取 health 返回的 backend/latency 字段
async function loadCacheHealth() {
  try {
    const health: any = await getCacheHealth()
    if (health) {
      redisHealth.status = health.status || 'unknown'
      redisHealth.backend = health.backend || 'memory'
      redisHealth.latency_ms = health.latency_ms || 0

      cacheBackend.backend = (health.backend || 'memory').toLowerCase()
      cacheBackend.persistent = Boolean(health.persistent)
      cacheBackend.degraded = Boolean(health.degraded)
      cacheBackend.reason = health.reason || ''
    }
  } catch (e) {
    redisHealth.status = 'unhealthy'
    redisHealth.backend = 'memory'
    cacheBackend.backend = 'memory'
    cacheBackend.persistent = false
    cacheBackend.degraded = true
    cacheBackend.reason = '无法获取后端状态'
    console.warn('Failed to load cache health:', e)
  }
}

async function runHealthCheck() {
  try {
    await getCacheHealth()
    await loadCacheHealth()
    await loadCacheStats()
    handleSuccess('健康检查完成')
  } catch (e) {
    handleApiError(e, '健康检查失败')
  }
}

function showRedisRecoveryGuide() {
  ElMessageBox.alert(
    [
      '1) 安装并启动 Redis（默认端口 6379）',
      '2) 检查 configs/config.json 中 redis.host/port 与实际一致',
      '3) 若使用远程 Redis，设置 REDIS_HOST/REDIS_PORT/REDIS_PASSWORD',
      '4) 执行 ./scripts/dev-restart.sh 重启网关',
      '5) 在本页点击"健康检查"，确认后端变为 REDIS 且持久化=是'
    ].join('<br/>'),
    '恢复 Redis 指引',
    {
      dangerouslyUseHTMLString: true,
      confirmButtonText: '我知道了',
      type: 'warning'
    }
  )
}

async function saveDedupConfig() {
  try {
    await updateCacheConfig({
      dedup: {
        enabled: dedupConfig.enabled,
        max_pending: dedupConfig.maxPending,
        request_timeout_seconds: dedupConfig.requestTimeout
      }
    })
    handleSuccess('请求去重配置已保存')
  } catch (e) {
    handleApiError(e, '保存失败')
  }
}

// 缓存内容管理函数
// FIX: 任务类型筛选变更时重置分页，避免空页
function handleEntriesTypeChange() {
  entriesFilter.page = 1
  loadCacheEntries()
}

function handleEntriesModeChange() {
  entriesFilter.page = 1
  loadCacheEntries()
}

// FIX: 搜索输入增加 300ms 防抖，避免高频请求
function handleEntriesSearchInput() {
  entriesFilter.page = 1
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    loadCacheEntries()
  }, 300)
}

function handleEntrySelectionChange(rows: any[]) {
  selectedEntryKeys.value = (rows || []).map(row => row.key).filter(Boolean)
}

async function loadCacheEntries() {
  entriesLoading.value = true
  try {
    const params = new URLSearchParams()
    params.append('type', 'response')
    // PERF: 维持服务端分页/筛选，避免在前端对大数据量二次遍历
    if (entriesFilter.type) params.append('task_type', entriesFilter.type)
    if (entriesFilter.search) params.append('search', entriesFilter.search)
    params.append('aggregate', entriesFilter.aggregate ? '1' : '0')
    params.append('readable_only', entriesFilter.readableOnly ? '1' : '0')
    params.append('page', entriesFilter.page.toString())
    params.append('page_size', entriesFilter.pageSize.toString())
    
    const data: any = await getCacheEntries(params.toString())
    if (data) {
      cacheEntries.value = data.entries || []
      entriesTotal.value = data.total || 0
      selectedEntryKeys.value = []
    }
  } catch (e) {
    console.warn('Failed to load cache entries:', e)
  } finally {
    entriesLoading.value = false
  }
}

async function cleanupInvalidEntries() {
  try {
    await ElMessageBox.confirm('将清理无任务类型、无消息、无回复、无创建时间的异常缓存条目，是否继续？', '清理异常条目', { type: 'warning' })
    const data: any = await cleanupInvalidEntriesApi()
    const deleted = data?.deleted || 0
    const failed = data?.failed || 0
    if (failed > 0) {
      ElMessage.warning(`已清理 ${deleted} 条，${failed} 条清理失败`)
    } else {
      handleSuccess(`已清理 ${deleted} 条异常缓存`)
    }
    await Promise.all([loadCacheEntries(), loadCacheStats()])
  } catch (e: any) {
    if (e !== 'cancel') {
      handleApiError(e, '清理失败')
    }
  }
}

async function batchDeleteEntries() {
  if (selectedEntryKeys.value.length === 0) return

  try {
    await ElMessageBox.confirm(`确定批量删除已选中的 ${selectedEntryKeys.value.length} 条缓存吗？`, '确认删除', { type: 'warning' })

    const keys = [...selectedEntryKeys.value]
    const selectedRows = cacheEntries.value.filter(row => keys.includes(row.key))
    let successCount = 0
    for (const row of selectedRows) {
      try {
        if (entriesFilter.aggregate && (row.group_count || 1) > 1) {
          const data: any = await deleteCacheEntryGroup({
            task_type: row.task_type || '',
            user_message: row.user_message || '',
            ai_response: row.ai_response || '',
            model: '',
            provider: row.provider || ''
          })
          successCount += Math.max(1, data?.deleted || 0)
        } else {
          await deleteCacheEntry(row.key)
          successCount++
        }
      } catch {
        // continue deleting other keys
      }
    }

    if (successCount > 0) {
      handleSuccess(`已删除 ${successCount} 条缓存`)
    }
    if (successCount < selectedRows.length) {
      ElMessage.warning(`有 ${selectedRows.length - successCount} 条删除失败，请重试`)
    }

    selectedEntryKeys.value = []
    await Promise.all([loadCacheEntries(), loadCacheStats()])
  } catch (e: any) {
    if (e !== 'cancel') {
      handleApiError(e, '批量删除失败')
    }
  }
}

async function viewEntryDetail(row: any) {
  try {
    const data: any = await getCacheEntryDetail(row.key)
    if (data) {
      entryDetail.value = {
        ...data,
        model_stats: row.model_stats || data.model_stats,
        task_type: row.task_type || data.task_type,
        task_type_source: row.task_type_source || data.task_type_source
      }
      entryDetailVisible.value = true
    }
  } catch (e) {
    handleApiError(e, '获取详情失败')
  }
}

async function deleteEntry(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除缓存 "${truncateKey(row.key, 30)}" 吗？`, '确认删除', { type: 'warning' })
    if (entriesFilter.aggregate && (row.group_count || 1) > 1) {
      await deleteCacheEntryGroup({
        task_type: row.task_type || '',
        user_message: row.user_message || '',
        ai_response: row.ai_response || '',
        model: '',
        provider: row.provider || ''
      })
    } else {
      await deleteCacheEntry(row.key)
    }
    handleSuccess('缓存已删除')
    loadCacheEntries()
    loadCacheStats()
  } catch (e: any) {
    if (e !== 'cancel') {
      handleApiError(e, '删除失败')
    }
  }
}

async function deleteEntryAndClose() {
  if (!entryDetail.value) return
  try {
    if (entriesFilter.aggregate && (entryDetail.value.group_count || 1) > 1) {
      await deleteCacheEntryGroup({
        task_type: entryDetail.value.task_type || '',
        user_message: entryDetail.value.user_message || '',
        ai_response: entryDetail.value.ai_response || '',
        model: '',
        provider: entryDetail.value.provider || ''
      })
    } else {
      await deleteCacheEntry(entryDetail.value.key)
    }
    handleSuccess('缓存已删除')
    entryDetailVisible.value = false
    loadCacheEntries()
    loadCacheStats()
  } catch (e) {
    handleApiError(e, '删除失败')
  }
}

function toModelStatsRows(modelStats: Record<string, number>): Array<{ model: string; count: number }> {
  return Object.entries(modelStats || {})
    .map(([model, count]) => ({ model, count }))
    .sort((a, b) => b.count - a.count)
}

function truncateKey(key: string, maxLen: number): string {
  if (key.length <= maxLen) return key
  return key.substring(0, maxLen) + '...'
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1024 / 1024).toFixed(1) + ' MB'
}

function formatTTL(seconds: number): string {
  if (seconds < 60) return seconds + 's'
  if (seconds < 3600) return Math.floor(seconds / 60) + 'm'
  if (seconds < 86400) return Math.floor(seconds / 3600) + 'h'
  return Math.floor(seconds / 86400) + 'd'
}

function formatTime(timestamp: string): string {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN')
}

function getCacheId(key: string): string {
  if (!key) return '-'
  const parts = key.split(':')
  const tail = parts[parts.length - 1] || key
  if (tail.length <= 12) return tail
  return tail.slice(0, 12)
}

function getUserMessageFull(row: any): string {
  return extractUserMessageFull(row)
}

function getAIResponseFull(row: any): string {
  return extractAIResponseFull(row)
}

// 获取任务类型标签颜色
function getTaskTypeTag(taskType: string): string {
  const types: Record<string, string> = {
    fact: 'primary',
    code: 'success',
    math: 'warning',
    chat: 'info',
    creative: 'danger',
    reasoning: 'success',
    translate: 'primary',
    long_text: 'warning',
    unknown: 'info',
    other: 'info'
  }
  return types[taskType] || 'info'
}

// 获取任务类型名称
function getTaskTypeName(taskType: string): string {
  const names: Record<string, string> = {
    fact: '事实',
    code: '代码',
    math: '数学',
    chat: '对话',
    creative: '创意',
    reasoning: '推理',
    translate: '翻译',
    long_text: '长文本',
    unknown: '其他',
    other: '其他'
  }
  return names[taskType] || taskType || '其他'
}

function getTaskTypeSourceTag(source: string): string {
  const types: Record<string, string> = {
    ollama: 'success',
    heuristic: 'warning',
    fallback: 'danger',
    manual: 'info',
    mixed: 'warning',
    legacy: 'info',
    unknown: 'info'
  }
  return types[source] || 'info'
}

function getTaskTypeSourceName(source: string): string {
  const names: Record<string, string> = {
    ollama: 'LLM',
    heuristic: '启发式',
    fallback: '回退',
    manual: '手动',
    mixed: '混合',
    legacy: '旧数据',
    unknown: '未知'
  }
  return names[source] || source || '-'
}

// 从缓存内容中提取用户消息
function getUserMessage(row: any): string {
  const full = getUserMessageFull(row)
  if (full === '-') return full
  return full.length > 100 ? full.slice(0, 100) + '...' : full
}

// 从缓存内容中提取AI回复
function getAIResponse(row: any): string {
  if (row.ai_response) return row.ai_response
  if (!row.value) return '-'
  try {
    const value = typeof row.value === 'string' ? JSON.parse(row.value) : row.value
    if (value.choices && Array.isArray(value.choices) && value.choices[0]) {
      const content = value.choices[0].message?.content
      if (content) {
        return content.length > 100 ? content.slice(0, 100) + '...' : content
      }
    }
  } catch (_error) {
    // CHANGE: invalid JSON fallback
    return '-'
  }
  return '-'
}

// 缓存预热相关函数
function showWarmupDialog() {
  warmupDialogVisible.value = true
}

async function submitWarmup() {
  if (!warmupFormRef.value) return
  
  try {
    await warmupFormRef.value.validate()
  } catch {
    return
  }
  
  warmupLoading.value = true
  try {
    await addTestCacheEntry(warmupForm)
    handleSuccess('测试缓存添加成功')
    warmupDialogVisible.value = false
    warmupForm.user_message = ''
    warmupForm.ai_response = ''
    loadCacheEntries()
    loadCacheStats()
  } catch (e) {
    handleApiError(e, '添加失败')
  } finally {
    warmupLoading.value = false
  }
}

// 导出缓存数据
async function exportCacheData() {
  try {
    const params = new URLSearchParams()
    if (entriesFilter.type) params.append('task_type', entriesFilter.type)
    
    const response = await fetch(`${API.CACHE.EXPORT}?${params.toString()}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    
    if (!response.ok) throw new Error('Export failed')
    
    const data = await response.json()
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `cache-export-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    
    handleSuccess('缓存数据已导出')
  } catch (e) {
    handleApiError(e, '导出失败')
  }
}

onMounted(() => {
  loadCacheStats()
  loadCacheConfig()
  loadTTLConfig()
  loadCacheHealth()
  loadCacheRules()
  loadCacheEntries()
  loadSemanticSignatures()
  loadVectorStats()
})

onUnmounted(() => {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
    searchDebounceTimer = null
  }
  if (detailChart) {
    detailChart.dispose()
    detailChart = null
  }
})
</script>

<style scoped lang="scss">
/* 改动点: 新版缓存管理视觉布局 */
.cache-page {
  --cache-ink: #0f172a;
  --cache-muted: #64748b;
  --cache-border: rgba(148, 163, 184, 0.2);
  --cache-panel: #ffffff;
  --cache-shadow: 0 16px 40px rgba(15, 23, 42, 0.08);
  --cache-accent: #0ea5e9;
  --cache-accent-strong: #2563eb;
  --cache-mono: 'SF Mono', 'JetBrains Mono', 'Menlo', 'Consolas', monospace;
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  font-family: 'PingFang SC', 'SF Pro Text', 'Segoe UI', 'Microsoft YaHei', sans-serif;
  color: var(--cache-ink);
  background:
    radial-gradient(circle at 15% 0%, rgba(14, 165, 233, 0.14), transparent 45%),
    radial-gradient(circle at 85% 10%, rgba(59, 130, 246, 0.12), transparent 40%);
}

.cache-hero {
  background: linear-gradient(130deg, #0b1220 0%, #1e293b 55%, #0f172a 100%);
  color: #fff;
  border-radius: 22px;
  padding: 22px 26px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  box-shadow: 0 18px 46px rgba(15, 23, 42, 0.2);

  .hero-main {
    display: flex;
    flex-direction: column;
  }

  .hero-title {
    font-size: 24px;
    font-weight: 600;
    letter-spacing: 0.6px;
  }

  .hero-subtitle {
    margin-top: 6px;
    font-size: 13px;
    color: rgba(255, 255, 255, 0.7);
  }

  .hero-actions {
    display: flex;
    gap: 10px;

    .ghost-btn {
      background: rgba(255, 255, 255, 0.16);
      border: 1px solid rgba(255, 255, 255, 0.2);
      color: #fff;
    }
  }

  .backend-badge {
    margin-top: 12px;
    width: fit-content;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 500;
    border: 1px solid rgba(255, 255, 255, 0.28);
    background: rgba(255, 255, 255, 0.12);
  }

  .backend-badge.backend-redis {
    background: rgba(22, 163, 74, 0.2);
    border-color: rgba(22, 163, 74, 0.4);
  }

  .backend-badge.backend-memory {
    background: rgba(245, 158, 11, 0.2);
    border-color: rgba(245, 158, 11, 0.4);
  }

  .badge-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #22c55e;
    box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.2);
  }

  .backend-badge.backend-memory .badge-dot {
    background: #f59e0b;
    box-shadow: 0 0 0 4px rgba(245, 158, 11, 0.2);
  }
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 14px;
}

.stat-card {
  background: var(--cache-panel);
  border-radius: 16px;
  padding: 14px;
  box-shadow: var(--cache-shadow);
  border: 1px solid var(--cache-border);
  display: flex;
  gap: 12px;

  .stat-icon {
    width: 42px;
    height: 42px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .stat-body {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .stat-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .stat-value {
    font-size: 20px;
    font-weight: 600;
    font-family: var(--cache-mono);
  }

  .stat-sub {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}

.backend-alert {
  background: linear-gradient(120deg, rgba(239, 68, 68, 0.1), rgba(251, 191, 36, 0.1));
  border: 1px solid rgba(239, 68, 68, 0.3);
  padding: 14px 18px;
  border-radius: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  font-size: 13px;
  color: #b91c1c;

  .backend-title {
    font-weight: 600;
  }

  .backend-sub {
    margin-top: 4px;
  }
}

.types-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel {
  background: var(--cache-panel);
  border-radius: 18px;
  padding: 18px;
  border: 1px solid var(--cache-border);
  box-shadow: var(--cache-shadow);
  animation: fadeUp 0.4s ease both;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;

  .panel-title {
    font-size: 16px;
    font-weight: 600;
  }

  .panel-subtitle {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}

.panel-hint {
  font-size: 12px;
  color: var(--cache-muted);
  background: rgba(148, 163, 184, 0.18);
  padding: 6px 10px;
  border-radius: 999px;
}

.type-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.type-card {
  position: relative;
  border-radius: 16px;
  padding: 16px;
  border: 1px solid var(--cache-border);
  background: linear-gradient(160deg, rgba(248, 250, 252, 0.9), rgba(255, 255, 255, 0.98));
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-shadow: 0 14px 32px rgba(15, 23, 42, 0.06);
  transition: transform 0.2s ease, box-shadow 0.2s ease;

  &[data-tone='ocean'] {
    --tone-color: #38bdf8;
    --tone-soft: rgba(56, 189, 248, 0.16);
  }

  &[data-tone='sunset'] {
    --tone-color: #fb923c;
    --tone-soft: rgba(251, 146, 60, 0.18);
  }

  &[data-tone='violet'] {
    --tone-color: #a78bfa;
    --tone-soft: rgba(167, 139, 250, 0.18);
  }

  &[data-tone='forest'] {
    --tone-color: #22c55e;
    --tone-soft: rgba(34, 197, 94, 0.16);
  }

  &[data-tone='ember'] {
    --tone-color: #f97316;
    --tone-soft: rgba(249, 115, 22, 0.18);
  }

  &[data-tone='neon'] {
    --tone-color: #22d3ee;
    --tone-soft: rgba(34, 211, 238, 0.18);
  }

  &::before {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: 16px;
    border: 1px solid transparent;
    background: linear-gradient(135deg, var(--tone-soft), transparent);
    pointer-events: none;
  }

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 18px 36px rgba(15, 23, 42, 0.12);
  }

  .type-card-head {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
  }

  .type-title {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .type-icon {
    width: 38px;
    height: 38px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--tone-soft, rgba(148, 163, 184, 0.18));
    color: var(--tone-color, var(--cache-accent));
  }

  .type-name {
    font-weight: 600;
    font-size: 15px;
  }

  .type-alias {
    font-size: 12px;
    color: var(--cache-muted);
  }

  .type-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .type-desc {
    font-size: 12px;
    color: var(--cache-muted);
    line-height: 1.5;
  }

  .type-prefix {
    display: flex;
    align-items: center;
    gap: 8px;
    font-family: var(--cache-mono);
    font-size: 11px;
    color: #0f172a;
    background: rgba(148, 163, 184, 0.16);
    padding: 6px 8px;
    border-radius: 10px;
    word-break: break-all;

    .prefix-label {
      color: var(--cache-muted);
      text-transform: uppercase;
      letter-spacing: 0.6px;
    }
  }

  .type-metrics {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;

    .metric {
      background: rgba(148, 163, 184, 0.12);
      border-radius: 12px;
      padding: 10px;
    }

    .metric-label {
      font-size: 12px;
      color: var(--cache-muted);
    }

    .metric-value {
      font-size: 18px;
      font-weight: 600;
      margin-top: 4px;
    }

    .metric-sub {
      font-size: 11px;
      color: var(--cache-muted);
      margin-top: 4px;
    }
  }

  .type-progress {
    .progress-meta {
      display: flex;
      justify-content: space-between;
      font-size: 12px;
      color: var(--cache-muted);
      margin-bottom: 6px;
    }
  }

  .type-actions {
    display: flex;
    gap: 12px;
  }
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.signature-section {
  background: linear-gradient(120deg, rgba(14, 165, 233, 0.08), rgba(248, 250, 252, 0.95));

  :deep(.el-table) {
    background: transparent;
  }

  :deep(.el-table th.el-table__cell) {
    background: rgba(148, 163, 184, 0.14);
    color: var(--cache-muted);
  }

  :deep(.el-table__row) {
    background: transparent;
  }
}

.summary-card {
  border-radius: 12px;
  padding: 12px;
  border: 1px dashed rgba(148, 163, 184, 0.45);
  background: linear-gradient(140deg, rgba(248, 250, 252, 0.9), rgba(255, 255, 255, 0.98));
  display: flex;
  flex-direction: column;
  gap: 4px;

  .summary-title {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .summary-value {
    font-size: 16px;
    font-weight: 600;
  }

  .summary-sub {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}

.cache-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 12px;
  }

  :deep(.el-tabs__nav-wrap::after) {
    display: none;
  }

  :deep(.el-tabs__nav) {
    background: #f1f5f9;
    border-radius: 999px;
    padding: 4px;
    border: none;
  }

  :deep(.el-tabs__item) {
    border: none;
    border-radius: 999px;
    font-size: 12px;
    padding: 6px 16px;
    color: var(--el-text-color-secondary);
  }

  :deep(.el-tabs__item.is-active) {
    background: var(--cache-accent-strong);
    color: #fff;
  }

  :deep(.el-tabs__active-bar) {
    display: none;
  }
}

.config-form {
  max-width: 720px;

  .unit-label {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
  }
}

.task-ttl-panel {
  max-width: 820px;
}

.task-ttl-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.task-ttl-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.ttl-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ttl-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.ttl-info {
  min-width: 0;
}

.ttl-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.ttl-desc {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.rules-header {
  margin-bottom: 12px;
}

.rules-table {
  .pattern-code {
    background: #f1f5f9;
    padding: 2px 8px;
    border-radius: 6px;
    font-family: var(--font-family-mono);
    font-size: 12px;
  }

  .text-muted {
    color: var(--el-text-color-secondary);
  }
}

.hot-cache-table {
  .query-cell {
    display: flex;
    align-items: center;
    gap: 6px;

    .hash-code {
      font-family: var(--font-family-mono);
      font-size: 12px;
      background: #f1f5f9;
      padding: 2px 6px;
      border-radius: 6px;
    }
  }

  .hits-count {
    font-weight: 600;
    color: var(--color-primary);
  }

  .time-text {
    font-family: var(--font-family-mono);
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}

.entries-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  align-items: center;
  flex-wrap: wrap;
}

.entries-toolbar-left {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.entries-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.entries-table {
  .key-cell {
    display: flex;
    align-items: center;
    gap: 8px;

    .key-text {
      font-family: var(--font-family-mono, monospace);
      font-size: 12px;
      background: rgba(148, 163, 184, 0.18);
      padding: 2px 6px;
      border-radius: 4px;
      word-break: break-all;
    }
  }

  .hits-count {
    font-weight: 600;
    color: var(--color-primary, #409eff);
  }
}

.entries-skeleton {
  border: 1px solid var(--cache-border);
  border-radius: 10px;
  padding: 12px;
  background: var(--cache-panel);

  .skeleton-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 10px;
  }
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.cache-detail {
  .detail-item {
    margin-bottom: 16px;

    .label {
      display: block;
      margin-bottom: 6px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .detail-desc {
    margin: 16px 0;
  }

  .detail-chart {
    h4 {
      margin-bottom: 12px;
      font-size: 14px;
      color: var(--el-text-color-primary);
    }

    .chart-container {
      height: 200px;
    }
  }
}

.advanced-section {
  h4 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .status-card {
    background: var(--el-fill-color-light);
    border-radius: 8px;
    padding: 16px;

    .status-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child {
        border-bottom: none;
      }

      .label {
        color: var(--el-text-color-secondary);
        font-size: 14px;
      }

      .value {
        font-weight: 500;
        color: var(--el-text-color-primary);
      }
    }
  }

  .unit-label {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

.entry-detail {
  .detail-key {
    font-family: var(--font-family-mono, monospace);
    font-size: 12px;
    background: #f1f5f9;
    padding: 4px 8px;
    border-radius: 4px;
    word-break: break-all;
  }

  .detail-value {
    margin-top: 16px;

    h4 {
      margin-bottom: 8px;
      font-size: 14px;
      font-weight: 600;
    }
  }
}

.message-preview {
  max-height: 60px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  font-size: 13px;
  color: var(--el-text-color-regular);
  line-height: 1.4;
}

.cache-id {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

@keyframes fadeUp {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .type-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .cache-hero {
    flex-direction: column;
    align-items: flex-start;
  }

  .type-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
