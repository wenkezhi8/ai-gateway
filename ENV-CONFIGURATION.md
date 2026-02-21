# AI Gateway - 环境变量配置指南

## 概述

本文档详细说明AI Gateway的所有环境变量配置选项。

---

## 快速开始

1. 复制环境变量模板：
```bash
cp .env.example .env
```

2. 编辑 `.env` 文件，配置你的API密钥：
```bash
nano .env
```

3. 启动服务：
```bash
./deploy/quick-start.sh
```

---

## 必需配置

### 服务端口

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `GATEWAY_PORT` | 8000 | API网关端口 |
| `WEB_PORT` | 3000 | Web控制台端口 |
| `REDIS_PORT` | 6379 | Redis端口 |

### API密钥

**至少配置一个API密钥才能正常使用**

#### OpenAI

| 变量名 | 必需 | 说明 |
|--------|------|------|
| `OPENAI_API_KEY` | 推荐 | OpenAI API密钥 |
| `OPENAI_BACKUP_API_KEY` | 可选 | OpenAI备份密钥（用于故障转移） |

获取地址: https://platform.openai.com/api-keys

格式: `sk-...`

#### Anthropic (Claude)

| 变量名 | 必需 | 说明 |
|--------|------|------|
| `ANTHROPIC_API_KEY` | 推荐 | Anthropic API密钥 |

获取地址: https://console.anthropic.com/settings/keys

格式: `sk-ant-...`

#### Azure OpenAI

| 变量名 | 必需 | 说明 |
|--------|------|------|
| `AZURE_OPENAI_API_KEY` | 可选 | Azure OpenAI密钥 |
| `AZURE_OPENAI_ENDPOINT` | 可选 | Azure OpenAI端点URL |

获取地址: https://portal.azure.com

#### 火山方舟 (Volcano Ark)

| 变量名 | 必需 | 说明 |
|--------|------|------|
| `VOLCANO_API_KEY` | 可选 | 火山方舟API密钥 |
| `VOLCANO_ENDPOINT` | 可选 | 火山方舟端点URL |

获取地址: https://console.volcengine.com/ark

默认端点: `https://ark.cn-beijing.volces.com/api/v3`

---

## 可选配置

### 日志配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `LOG_LEVEL` | info | 日志级别: debug, info, warn, error |

**推荐配置**:
- 开发环境: `debug`
- 测试环境: `info`
- 生产环境: `warn`

### Redis配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `REDIS_MAXMEMORY` | 256mb | Redis最大内存 |

**推荐配置**:
- 开发环境: `256mb`
- 生产环境: `512mb` 或更高

### 速率限制

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `RATE_LIMIT` | 100 | 每秒请求数限制 |
| `RATE_BURST` | 200 | 突发请求数 |

### 缓存配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `CACHE_ENABLED` | true | 启用请求缓存 |
| `CACHE_TTL` | 3600 | 缓存过期时间（秒） |
| `CACHE_MAX_SIZE` | 100 | 缓存最大大小（MB） |

### 高级配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `METRICS_ENABLED` | true | 启用Prometheus指标 |
| `REQUEST_TIMEOUT` | 60 | 请求超时时间（秒） |
| `MAX_REQUEST_SIZE` | 10 | 最大请求体大小（MB） |

---

## 监控配置（可选）

### Prometheus

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `PROMETHEUS_PORT` | 9090 | Prometheus端口 |

### Grafana

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `GRAFANA_PORT` | 3001 | Grafana端口 |
| `GRAFANA_ADMIN_USER` | admin | Grafana管理员用户名 |
| `GRAFANA_ADMIN_PASSWORD` | admin123 | Grafana管理员密码 |

**⚠️ 生产环境请务必修改默认密码！**

### Alertmanager

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `ALERTMANAGER_PORT` | 9093 | Alertmanager端口 |

---

## 告警通知（可选）

### 邮件配置

| 变量名 | 说明 |
|--------|------|
| `SMTP_HOST` | SMTP服务器地址 |
| `SMTP_PORT` | SMTP端口（通常是587） |
| `SMTP_FROM` | 发件人地址 |
| `SMTP_USER` | SMTP用户名 |
| `SMTP_PASSWORD` | SMTP密码 |
| `EMAIL_TO` | 接收告警的邮箱 |

---

## 配置示例

### 开发环境配置

```env
# Server Ports
GATEWAY_PORT=8000
WEB_PORT=3000
REDIS_PORT=6379

# API Keys
OPENAI_API_KEY=sk-your-dev-key
ANTHROPIC_API_KEY=sk-ant-your-dev-key

# Logging
LOG_LEVEL=debug

# Cache
CACHE_ENABLED=true
CACHE_TTL=3600

# Metrics
METRICS_ENABLED=true
```

### 生产环境配置

```env
# Server Ports
GATEWAY_PORT=8000
WEB_PORT=3000
REDIS_PORT=6379

# API Keys
OPENAI_API_KEY=sk-prod-key-1
OPENAI_BACKUP_API_KEY=sk-prod-key-2
ANTHROPIC_API_KEY=sk-ant-prod-key
VOLCANO_API_KEY=your-volcano-key
VOLCANO_ENDPOINT=https://ark.cn-beijing.volces.com/api/v3

# Logging
LOG_LEVEL=warn

# Performance
RATE_LIMIT=200
RATE_BURST=400
REDIS_MAXMEMORY=512mb

# Cache
CACHE_ENABLED=true
CACHE_TTL=7200
CACHE_MAX_SIZE=500

# Monitoring
METRICS_ENABLED=true
PROMETHEUS_PORT=9090
GRAFANA_PORT=3001
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=your-secure-password

# Alerts
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_FROM=noreply@example.com
SMTP_USER=your-smtp-user
SMTP_PASSWORD=your-smtp-password
EMAIL_TO=admin@example.com
```

---

## 配置最佳实践

### 安全建议

1. **不要提交 .env 文件到版本控制**
   - `.env` 已在 `.gitignore` 中
   - 只提交 `.env.example` 模板

2. **使用强密码**
   - Grafana密码至少16位
   - 包含大小写字母、数字、特殊字符

3. **定期轮换密钥**
   - 建议每90天更换API密钥
   - 使用不同的密钥用于不同环境

4. **限制访问**
   - 生产环境不要暴露所有端口
   - 使用防火墙限制访问

### 性能建议

1. **Redis内存**
   - 根据缓存需求设置
   - 监控内存使用情况

2. **速率限制**
   - 根据API配额设置
   - 避免超过服务商限制

3. **缓存策略**
   - 合理设置TTL
   - 监控缓存命中率

---

## 故障排查

### API密钥无效

**症状**: 401 Unauthorized 错误

**解决方案**:
1. 检查密钥格式是否正确
2. 确认密钥未过期
3. 验证密钥有足够的配额

### Redis连接失败

**症状**: "connection refused" 错误

**解决方案**:
1. 确认Redis服务已启动
2. 检查端口配置是否正确
3. 验证网络连接

### 端口被占用

**症状**: "address already in use" 错误

**解决方案**:
```bash
# 检查端口占用
lsof -i :8000

# 修改.env中的端口配置
GATEWAY_PORT=8001
```

---

## 获取帮助

- 查看日志: `./scripts/start-gateway.sh --logs`
- 检查配置: `./deploy/verify-config.sh`
- 查看文档: `/docs` 目录
- GitHub Issues: 报告问题和获取支持

---

## 更新日志

### v1.0.0 (2024-02-14)
- 添加火山方舟（Volcano Ark）支持
- 添加备份API密钥配置
- 添加高级缓存配置
- 添加请求限制配置
- 完善监控配置选项
