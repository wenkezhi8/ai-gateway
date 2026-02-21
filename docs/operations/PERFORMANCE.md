# AI Gateway - 性能优化指南

## 概述

本文档提供AI Gateway的性能优化建议、监控方法和容量规划指南。

---

## 性能监控

### 关键性能指标

| 指标 | 目标值 | 告警阈值 | 说明 |
|------|--------|----------|------|
| 响应时间 P50 | < 100ms | > 200ms | 中位数响应时间 |
| 响应时间 P99 | < 500ms | > 1s | 99分位响应时间 |
| 吞吐量 | > 1000 req/s | < 500 req/s | 每秒请求数 |
| 错误率 | < 0.1% | > 1% | 失败请求比例 |
| 缓存命中率 | > 80% | < 60% | 缓存效率 |
| CPU使用率 | < 70% | > 85% | 处理器使用 |
| 内存使用率 | < 80% | > 90% | 内存使用 |
| 磁盘I/O | < 100 MB/s | > 500 MB/s | 磁盘读写 |

### Prometheus查询

#### 响应时间
```promql
# P50响应时间
histogram_quantile(0.50, rate(ai_gateway_api_response_time_seconds_bucket[5m]))

# P99响应时间
histogram_quantile(0.99, rate(ai_gateway_api_response_time_seconds_bucket[5m]))

# 平均响应时间
rate(ai_gateway_api_response_time_seconds_sum[5m]) / rate(ai_gateway_api_response_time_seconds_count[5m])
```

#### 吞吐量
```promql
# 每秒请求数
rate(ai_gateway_api_requests_total[1m])

# 成功请求率
rate(ai_gateway_api_requests_total{status="200"}[5m]) / rate(ai_gateway_api_requests_total[5m])
```

#### 错误率
```promql
# 错误请求率
rate(ai_gateway_api_requests_failed_total[5m]) / rate(ai_gateway_api_requests_total[5m])

# 按错误类型分组
sum by (error_type) (rate(ai_gateway_api_requests_failed_total[5m]))
```

#### 缓存性能
```promql
# 缓存命中率
rate(ai_gateway_cache_hits_total[5m]) / rate(ai_gateway_cache_requests_total[5m])

# 缓存大小
ai_gateway_cache_size_bytes
```

### 性能测试

#### 负载测试

```bash
# 使用Apache Bench
ab -n 10000 -c 100 http://localhost:8000/api/v1/chat/completions

# 使用wrk
wrk -t 12 -c 400 -d 30s http://localhost:8000/api/v1/models

# 使用hey
hey -n 10000 -c 100 -m POST -H "Content-Type: application/json" \
    -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}' \
    http://localhost:8000/api/v1/chat/completions
```

#### 基准测试

```bash
#!/bin/bash
# benchmark.sh

echo "Running performance benchmark..."

# 测试1: 健康检查
echo "Testing health endpoint..."
hey -n 10000 -c 100 http://localhost:8000/health

# 测试2: 模型列表
echo "Testing models endpoint..."
hey -n 1000 -c 50 http://localhost:8000/api/v1/models

# 测试3: 聊天完成
echo "Testing chat completions..."
hey -n 100 -c 10 -m POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $OPENAI_API_KEY" \
    -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}],"max_tokens":50}' \
    http://localhost:8000/api/v1/chat/completions

echo "Benchmark completed!"
```

---

## 应用层优化

### 缓存优化

#### 启用缓存
```bash
# .env配置
CACHE_ENABLED=true
CACHE_TTL=3600
CACHE_MAX_SIZE=500
```

#### 缓存策略

**请求缓存**
- 缓存相同请求的响应
- 适用于幂等操作
- TTL: 1-24小时

**语义缓存**
- 缓存语义相似的请求
- 使用embedding计算相似度
- 适用于问答场景

#### 缓存预热
```bash
# 预热常用模型
curl -X POST http://localhost:8000/api/v1/cache/warm \
  -H "Content-Type: application/json" \
  -d '{
    "models": ["gpt-3.5-turbo", "gpt-4"],
    "prompts": ["Hello", "Hi", "你好"]
  }'
```

#### Redis优化

```bash
# 查看Redis配置
docker exec ai-gateway-redis redis-cli config get "*"

# 优化内存策略
docker exec ai-gateway-redis redis-cli config set maxmemory-policy allkeys-lru

# 优化持久化
docker exec ai-gateway-redis redis-cli config set save "900 1 300 10"

# 禁用持久化（仅缓存场景）
docker exec ai-gateway-redis redis-cli config set save ""
```

### 连接池优化

```go
// configs/config.json
{
  "http_client": {
    "max_idle_conns": 100,
    "max_idle_conns_per_host": 10,
    "idle_conn_timeout": 90
  },
  "redis": {
    "pool_size": 100,
    "min_idle_conns": 10,
    "pool_timeout": 30
  }
}
```

### 并发控制

```bash
# 调整速率限制
RATE_LIMIT=200
RATE_BURST=400

# 调整worker数量
MAX_WORKERS=100
```

---

## 系统层优化

### Docker优化

#### 资源限制
```yaml
# docker-compose.yml
services:
  gateway:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 4G
        reservations:
          cpus: '2'
          memory: 2G
```

#### 日志优化
```yaml
services:
  gateway:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### Linux系统优化

#### 内核参数
```bash
# /etc/sysctl.conf

# 网络优化
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_tw_reuse = 1

# 内存优化
vm.swappiness = 10
vm.dirty_ratio = 60
vm.dirty_background_ratio = 30

# 文件描述符
fs.file-max = 2097152
fs.nr_open = 2097152

# 应用配置
sudo sysctl -p
```

#### 文件描述符限制
```bash
# /etc/security/limits.conf
* soft nofile 65535
* hard nofile 65535
root soft nofile 65535
root hard nofile 65535
```

#### 磁盘I/O优化
```bash
# 使用deadline或noop调度器
echo deadline > /sys/block/sda/queue/scheduler

# 或在/etc/default/grub中添加
GRUB_CMDLINE_LINUX="elevator=deadline"
```

### 网络优化

#### TCP优化
```bash
# 增加TCP缓冲区
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
```

#### 连接优化
```bash
# 减少TCP超时
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 1200
net.ipv4.tcp_keepalive_probes = 5
net.ipv4.tcp_keepalive_intvl = 30
```

---

## 数据库优化

### SQLite优化

#### 性能配置
```sql
-- 启用WAL模式
PRAGMA journal_mode = WAL;

-- 增加缓存大小
PRAGMA cache_size = -64000;  -- 64MB

-- 同步模式
PRAGMA synchronous = NORMAL;

-- 临时存储
PRAGMA temp_store = MEMORY;
```

#### 索引优化
```sql
-- 为常用查询创建索引
CREATE INDEX idx_requests_created_at ON usage_logs(created_at);
CREATE INDEX idx_requests_user_id ON usage_logs(user_id);
CREATE INDEX idx_requests_model ON usage_logs(model);

-- 分析查询计划
EXPLAIN QUERY PLAN SELECT * FROM usage_logs WHERE user_id = ?;
```

#### 定期维护
```bash
# 每周执行
sqlite3 /path/to/ai-gateway.db "VACUUM;"
sqlite3 /path/to/ai-gateway.db "ANALYZE;"
sqlite3 /path/to/ai-gateway.db "PRAGMA optimize;"
```

### 迁移到PostgreSQL

当数据量超过10GB或并发很高时，考虑迁移到PostgreSQL：

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: ai_gateway
      POSTGRES_USER: gateway
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    command:
      - "postgres"
      - "-c"
      - "max_connections=200"
      - "-c"
      - "shared_buffers=2GB"
      - "-c"
      - "effective_cache_size=6GB"
```

---

## 网络优化

### Nginx优化

```nginx
# nginx.conf
worker_processes auto;
worker_rlimit_nofile 65535;

events {
    worker_connections 65535;
    use epoll;
    multi_accept on;
}

http {
    # 连接优化
    keepalive_timeout 65;
    keepalive_requests 100;

    # 缓冲优化
    client_body_buffer_size 16k;
    client_header_buffer_size 1k;
    client_max_body_size 10m;
    large_client_header_buffers 4 8k;

    # 输出优化
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;

    # 压缩
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain application/json;

    # 上游优化
    upstream gateway {
        least_conn;
        server gateway:8080 weight=5;
        server gateway:8080 weight=5;
        keepalive 32;
    }
}
```

### 负载均衡

```yaml
# 水平扩展
docker-compose up -d --scale gateway=3

# 配合Nginx负载均衡
upstream gateway_cluster {
    least_conn;
    server gateway1:8080;
    server gateway2:8080;
    server gateway3:8080;
}
```

---

## 容量规划

### 资源评估

#### 基准性能
- 单实例: 1000 req/s
- CPU: 2核 @ 70%
- 内存: 2GB @ 80%
- 网络: 100 Mbps

#### 扩展公式

**CPU需求**
```
所需CPU核心 = (目标QPS / 单核QPS) * 冗余系数
           = (10000 / 500) * 1.5
           = 30核
```

**内存需求**
```
所需内存 = (并发连接数 * 单连接内存) + 缓存大小 + 系统预留
        = (10000 * 10KB) + 1GB + 512MB
        = ~1.6GB
```

**存储需求**
```
每日增长 = (请求数 * 平均日志大小)
        = (1000000 * 500B)
        = 500MB/天
        = 15GB/月
```

### 扩展策略

#### 垂直扩展（Scale Up）
```yaml
# 增加单实例资源
deploy:
  resources:
    limits:
      cpus: '8'
      memory: 16G
```

**优点**: 简单快速
**缺点**: 单点故障，成本高

#### 水平扩展（Scale Out）
```yaml
# 增加实例数量
docker-compose up -d --scale gateway=5
```

**优点**: 高可用，线性扩展
**缺点**: 需要负载均衡

#### 自动扩展

使用Kubernetes HPA:
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ai-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ai-gateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

---

## 性能调优检查清单

### 应用层
- [ ] 启用缓存并配置合理的TTL
- [ ] 优化数据库查询和索引
- [ ] 调整连接池大小
- [ ] 配置合适的并发限制
- [ ] 启用响应压缩

### 系统层
- [ ] 优化内核参数
- [ ] 增加文件描述符限制
- [ ] 配置Docker资源限制
- [ ] 优化磁盘I/O调度
- [ ] 启用网络优化

### 监控层
- [ ] 配置性能监控
- [ ] 设置性能告警
- [ ] 定期性能测试
- [ ] 分析性能趋势
- [ ] 制定容量计划

### 优化验证
- [ ] 运行基准测试
- [ ] 对比优化前后性能
- [ ] 记录优化效果
- [ ] 持续监控改进

---

## 性能问题排查

### 响应慢

**症状**: P99 > 1秒

**排查步骤:**
```bash
# 1. 查看慢请求日志
docker-compose logs gateway | grep "took.*[0-9]{4,}ms"

# 2. 检查API提供商延迟
curl -w "@curl-format.txt" https://api.openai.com/v1/models

# 3. 检查Redis延迟
docker exec ai-gateway-redis redis-cli --latency

# 4. 查看CPU使用
docker stats --no-stream | grep gateway

# 5. 检查网络
ping api.openai.com
traceroute api.openai.com
```

**解决方案:**
- 启用缓存
- 增加并发
- 优化网络
- 扩容资源

### 吞吐量低

**症状**: QPS < 500

**排查步骤:**
```bash
# 1. 检查限流配置
grep RATE .env

# 2. 查看连接数
netstat -an | grep ESTABLISHED | wc -l

# 3. 检查worker数量
ps aux | grep ai-gateway

# 4. 查看队列积压
curl http://localhost:8000/debug/vars | jq .queue_length
```

**解决方案:**
- 增加速率限制
- 增加worker
- 启用连接复用
- 水平扩展

### 内存泄漏

**症状**: 内存持续增长

**排查步骤:**
```bash
# 1. 监控内存使用
watch -n 1 'docker stats --no-stream | grep gateway'

# 2. 查看goroutine数量
curl http://localhost:8000/debug/pprof/goroutine?debug=1

# 3. 内存profile
curl http://localhost:8000/debug/pprof/heap > heap.out
go tool pprof heap.out

# 4. 查看对象数量
curl http://localhost:8000/debug/pprof/allocs?debug=1
```

**解决方案:**
- 修复代码中的泄漏
- 定期重启服务
- 增加内存限制
- 优化缓存策略

---

**文档版本**: v1.0.0
**最后更新**: 2024-02-14
**维护者**: DevOps Team
