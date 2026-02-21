# AI Gateway 监控仪表盘 API 文档

## 概述

本文档描述了监控仪表盘后端 API 接口，供前端开发人员对接使用。

## 基础信息

- **Base URL**: `/api/admin/metrics`
- **认证**: 需要管理员权限 (Bearer Token)
- **响应格式**: JSON

## 通用响应格式

```json
{
  "success": true,
  "data": { ... }
}
```

错误响应:
```json
{
  "success": false,
  "error": {
    "code": "error_code",
    "message": "Error description"
  }
}
```

---

## API 接口

### 1. 获取系统概览

获取仪表盘首页所需的概览统计数据。

**请求**
```
GET /api/admin/metrics/overview
```

**响应**
```json
{
  "success": true,
  "data": {
    "total_requests": 125847,
    "requests_today": 3421,
    "success_rate": 99.2,
    "avg_latency_ms": 245,
    "total_tokens": 8547123,
    "active_accounts": 5,
    "active_providers": 3,
    "cache_hit_rate": 34.5,
    "provider_stats": [
      {
        "name": "openai",
        "requests": 50000,
        "tokens": 3000000,
        "success_rate": 99.5,
        "avg_latency_ms": 220
      },
      {
        "name": "anthropic",
        "requests": 45000,
        "tokens": 3500000,
        "success_rate": 99.1,
        "avg_latency_ms": 280
      },
      {
        "name": "volcengine",
        "requests": 30847,
        "tokens": 2047123,
        "success_rate": 98.8,
        "avg_latency_ms": 235
      }
    ],
    "top_models": [
      {"name": "gpt-4", "requests": 30000, "tokens": 2500000},
      {"name": "claude-3-opus", "requests": 25000, "tokens": 2800000},
      {"name": "gpt-3.5-turbo", "requests": 20000, "tokens": 500000},
      {"name": "doubao-pro", "requests": 15000, "tokens": 1200000},
      {"name": "claude-3-sonnet", "requests": 10000, "tokens": 800000}
    ]
  }
}
```

**字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| total_requests | int64 | 历史总请求数 |
| requests_today | int64 | 今日请求数 |
| success_rate | float64 | 成功率 (0-100) |
| avg_latency_ms | int64 | 平均延迟 (毫秒) |
| total_tokens | int64 | 总 Token 消耗 |
| active_accounts | int | 活跃账号数 |
| active_providers | int | 活跃服务商数 |
| cache_hit_rate | float64 | 缓存命中率 (0-100) |
| provider_stats | array | 各服务商统计 |
| top_models | array | 热门模型 Top 5 |

---

### 2. 获取请求趋势

获取请求趋势数据，用于绘制折线图。

**请求**
```
GET /api/admin/metrics/requests?period=24h&interval=hour
```

**参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| period | string | 否 | 时间范围: `24h`, `7d`, `30d`，默认 `24h` |
| interval | string | 否 | 聚合间隔: `hour`, `day`，默认 `hour` |

**响应**
```json
{
  "success": true,
  "data": {
    "period": "24h",
    "interval": "hour",
    "data": [
      {
        "timestamp": "2024-01-15T00:00:00Z",
        "requests": 125,
        "success": 123,
        "failed": 2,
        "avg_latency_ms": 230
      },
      {
        "timestamp": "2024-01-15T01:00:00Z",
        "requests": 98,
        "success": 97,
        "failed": 1,
        "avg_latency_ms": 215
      }
      // ... 更多数据点
    ]
  }
}
```

**字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| timestamp | datetime | ISO 8601 时间戳 |
| requests | int64 | 该时段请求数 |
| success | int64 | 成功请求数 |
| failed | int64 | 失败请求数 |
| avg_latency_ms | int64 | 平均延迟 (毫秒) |

---

### 3. 获取服务商统计

获取各服务商的详细统计数据，用于饼图和表格展示。

**请求**
```
GET /api/admin/metrics/providers
```

**响应**
```json
{
  "success": true,
  "data": {
    "providers": [
      {
        "name": "openai",
        "models": ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"],
        "enabled": true,
        "requests": 50000,
        "tokens": 3000000,
        "success_rate": 99.5,
        "avg_latency_ms": 220,
        "last_used": "2024-01-15T10:30:00Z"
      },
      {
        "name": "anthropic",
        "models": ["claude-3-opus", "claude-3-sonnet", "claude-3-haiku"],
        "enabled": true,
        "requests": 45000,
        "tokens": 3500000,
        "success_rate": 99.1,
        "avg_latency_ms": 280,
        "last_used": "2024-01-15T10:28:00Z"
      }
    ],
    "distribution": {
      "openai": 39.7,
      "anthropic": 35.8,
      "volcengine": 24.5
    },
    "total": 125847
  }
}
```

**字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| providers | array | 服务商详细列表 |
| distribution | object | 请求占比分布 (百分比) |
| total | int64 | 总请求数 |

---

### 4. 获取缓存统计

获取缓存系统性能数据。

**请求**
```
GET /api/admin/metrics/cache
```

**响应**
```json
{
  "success": true,
  "data": {
    "request_cache": {
      "hits": 12500,
      "misses": 25000,
      "hit_rate": 33.3,
      "size": 52428800,
      "max_size": 104857600,
      "evictions": 150
    },
    "context_cache": {
      "hits": 8000,
      "misses": 12000,
      "hit_rate": 40.0,
      "size": 20971520,
      "max_size": 52428800,
      "evictions": 50
    },
    "route_cache": {
      "hits": 45000,
      "misses": 5000,
      "hit_rate": 90.0,
      "size": 1048576,
      "max_size": 10485760,
      "evictions": 10
    },
    "usage_cache": {
      "hits": 30000,
      "misses": 10000,
      "hit_rate": 75.0,
      "size": 5242880,
      "max_size": 10485760,
      "evictions": 20
    },
    "response_cache": {
      "hits": 5000,
      "misses": 20000,
      "hit_rate": 20.0,
      "size": 104857600,
      "max_size": 209715200,
      "evictions": 200
    },
    "token_savings": 1500000
  }
}
```

**字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| hits | int64 | 缓存命中次数 |
| misses | int64 | 缓存未命中次数 |
| hit_rate | float64 | 命中率 (0-100) |
| size | int64 | 当前缓存大小 (bytes) |
| max_size | int64 | 最大缓存大小 (bytes) |
| evictions | int64 | 驱逐次数 |
| token_savings | int64 | 节省的 Token 数量 |

---

### 5. 获取用量统计

获取 Token 和请求的详细用量统计。

**请求**
```
GET /api/admin/metrics/usage?start=2024-01-01T00:00:00Z&end=2024-01-15T23:59:59Z
```

**参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | datetime | 否 | 开始时间 (ISO 8601)，默认 24 小时前 |
| end | datetime | 否 | 结束时间 (ISO 8601)，默认当前时间 |

**响应**
```json
{
  "success": true,
  "data": {
    "start_time": "2024-01-01T00:00:00Z",
    "end_time": "2024-01-15T23:59:59Z",
    "total_tokens": 8547123,
    "prompt_tokens": 5213890,
    "output_tokens": 3333233,
    "total_requests": 125847,
    "by_model": [
      {
        "model": "gpt-4",
        "requests": 30000,
        "tokens": 2500000,
        "prompt_tokens": 1500000,
        "output_tokens": 1000000,
        "percent_of_total": 29.3
      }
    ],
    "by_user": [
      {
        "user_id": "user_123",
        "requests": 5000,
        "tokens": 350000,
        "percent_of_total": 4.1
      }
    ],
    "daily_trend": [
      {
        "date": "2024-01-15T00:00:00Z",
        "requests": 3421,
        "tokens": 234567,
        "users": 45
      }
    ]
  }
}
```

---

### 6. 获取实时指标

获取实时监控数据，用于动态仪表盘。

**请求**
```
GET /api/admin/metrics/realtime
```

**响应**
```json
{
  "success": true,
  "data": {
    "timestamp": "2024-01-15T10:30:45Z",
    "active_connections": 25,
    "requests_per_minute": 45,
    "tokens_per_minute": 12500,
    "avg_latency_ms": 235,
    "error_rate": 0.5,
    "top_models": [
      {"name": "gpt-4", "requests": 30000, "tokens": 2500000}
    ],
    "recent_errors": [
      {
        "timestamp": "2024-01-15T10:28:30Z",
        "provider": "openai",
        "model": "gpt-4",
        "error": "rate_limit_exceeded",
        "count": 3
      }
    ]
  }
}
```

**刷新建议**: 每 5-10 秒轮询一次

---

## 前端集成建议

### 图表组件映射

| API | 图表类型 | 用途 |
|-----|---------|------|
| /overview | 数字卡片 | 关键指标展示 |
| /requests | 折线图 | 请求趋势 |
| /providers | 饼图 + 表格 | 服务商分布 |
| /cache | 进度条 + 环形图 | 缓存性能 |
| /usage | 柱状图 | 用量趋势 |
| /realtime | 实时刷新 | 动态监控 |

### ECharts 配置示例

**请求趋势折线图**
```javascript
{
  xAxis: {
    type: 'category',
    data: trends.map(t => t.timestamp)
  },
  yAxis: {
    type: 'value'
  },
  series: [
    {
      name: '请求数',
      type: 'line',
      data: trends.map(t => t.requests),
      smooth: true
    },
    {
      name: '成功',
      type: 'line',
      data: trends.map(t => t.success),
      smooth: true
    }
  ]
}
```

**服务商分布饼图**
```javascript
{
  series: [{
    type: 'pie',
    data: Object.entries(distribution).map(([name, value]) => ({
      name,
      value
    }))
  }]
}
```

---

## 注意事项

1. 所有时间戳使用 ISO 8601 格式 (UTC)
2. 百分比值范围 0-100，不是 0-1
3. 延迟单位统一为毫秒 (ms)
4. 大数值建议前端格式化显示 (如 1.2M, 15.3K)
5. 实时接口建议 5-10 秒轮询间隔

如有问题，请联系架构师。
