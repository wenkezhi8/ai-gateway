# AI Gateway - 运维手册

## 目录

1. [日常运维](#日常运维)
2. [服务管理](#服务管理)
3. [监控告警](#监控告警)
4. [日志管理](#日志管理)
5. [性能维护](#性能维护)
6. [安全管理](#安全管理)
7. [备份恢复](#备份恢复)
8. [应急响应](#应急响应)

---

## 日常运维

### 每日检查清单

#### 健康检查（5分钟）
```bash
# 1. 检查服务状态
docker-compose ps

# 2. 检查服务健康
curl -f http://localhost:8000/health || echo "Gateway unhealthy"
curl -f http://localhost:3000 || echo "Web unhealthy"

# 3. 检查Redis连接
docker exec ai-gateway-redis redis-cli ping

# 4. 查看最近日志
docker-compose logs --tail=50 gateway
```

#### 资源使用检查（5分钟）
```bash
# 1. 检查容器资源使用
docker stats --no-stream

# 2. 检查磁盘使用
df -h

# 3. 检查内存使用
free -h

# 4. 检查Docker卷使用
docker system df -v
```

### 每周检查清单

#### 日志审查（30分钟）
```bash
# 1. 查看错误日志
docker-compose logs gateway | grep -i error | tail -100

# 2. 查看告警日志
docker-compose logs gateway | grep -i warn | tail -100

# 3. 统计API调用
docker-compose logs gateway | grep "POST /api" | wc -l

# 4. 检查慢请求
docker-compose logs gateway | grep "took.*[0-9]{4,}ms"
```

#### 性能分析（30分钟）
```bash
# 1. 查看Prometheus指标
curl http://localhost:9090/api/v1/query?query=up

# 2. 检查响应时间
curl http://localhost:9090/api/v1/query?query=histogram_quantile(0.99,rate(ai_gateway_api_response_time_seconds_bucket[5m]))

# 3. 查看错误率
curl http://localhost:9090/api/v1/query?query=rate(ai_gateway_api_requests_failed_total[5m])

# 4. 检查缓存命中率
docker exec ai-gateway-redis redis-cli info stats | grep keyspace
```

### 每月检查清单

#### 容量规划（1小时）
- [ ] 检查月度API调用量趋势
- [ ] 审查资源使用增长
- [ ] 评估是否需要扩容
- [ ] 检查API配额使用情况
- [ ] 审查成本和优化机会

#### 安全审计（1小时）
- [ ] 检查API密钥有效期
- [ ] 审查访问日志异常
- [ ] 更新安全补丁
- [ ] 检查SSL证书有效期
- [ ] 审查用户权限

---

## 服务管理

### 启动服务

#### 标准启动
```bash
# 开发环境
docker-compose up -d

# 生产环境
docker-compose -f deploy/docker-compose.prod.yml up -d
```

#### 快速启动（一键）
```bash
# Mac/Linux
./deploy/quick-start.sh

# Windows
deploy\quick-start.bat
```

#### 带监控启动
```bash
./scripts/start-gateway.sh --monitoring
```

### 停止服务

#### 优雅停止
```bash
# 停止所有服务
docker-compose stop

# 停止特定服务
docker-compose stop gateway
```

#### 完全停止并清理
```bash
# 停止并删除容器
docker-compose down

# 停止并删除容器+卷（危险！）
docker-compose down -v
```

### 重启服务

#### 滚动重启
```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart gateway

# 零停机重启（生产环境）
docker-compose up -d --no-deps --build gateway
```

### 更新服务

#### 更新镜像
```bash
# 1. 拉取最新镜像
docker-compose pull

# 2. 重新创建容器
docker-compose up -d

# 3. 清理旧镜像
docker image prune -f
```

#### 更新代码
```bash
# 1. 备份当前版本
./scripts/upgrade.sh --backup-only

# 2. 拉取代码更新
git pull

# 3. 重新构建并启动
docker-compose up -d --build

# 4. 验证更新
curl http://localhost:8000/health
```

---

## 监控告警

### Prometheus监控

#### 访问Prometheus
```bash
# Web界面
open http://localhost:9090

# API查询
curl http://localhost:9090/api/v1/query?query=up
```

#### 关键指标

**服务可用性**
```promql
up{job="ai-gateway"}
```

**请求速率**
```promql
rate(ai_gateway_api_requests_total[5m])
```

**错误率**
```promql
rate(ai_gateway_api_requests_failed_total[5m]) / rate(ai_gateway_api_requests_total[5m])
```

**响应时间P99**
```promql
histogram_quantile(0.99, rate(ai_gateway_api_response_time_seconds_bucket[5m]))
```

**缓存命中率**
```promql
rate(ai_gateway_cache_hits_total[5m]) / rate(ai_gateway_cache_requests_total[5m])
```

### Grafana仪表盘

#### 访问Grafana
```bash
# Web界面
open http://localhost:3001

# 默认凭据
用户名: admin
密码: admin123 (生产环境请修改)
```

#### 导入仪表盘
1. 登录Grafana
2. 点击 "+" → "Import"
3. 上传JSON文件或输入ID
4. 选择Prometheus数据源
5. 点击"Import"

### 告警管理

#### 查看告警
```bash
# 访问Alertmanager
open http://localhost:9093

# 查看活跃告警
curl http://localhost:9093/api/v1/alerts
```

#### 告警规则

**服务宕机**
- 触发条件: `up{job="ai-gateway"} == 0` 持续1分钟
- 严重程度: Critical
- 通知: 立即

**高错误率**
- 触发条件: 错误率 > 5% 持续5分钟
- 严重程度: Warning
- 通知: 邮件+Slack

**响应时间过长**
- 触发条件: P99 > 500ms 持续3分钟
- 严重程度: Warning
- 通知: 邮件

#### 配置通知

**邮件通知**
```yaml
# monitoring/alertmanager.yml
receivers:
  - name: 'team-email'
    email_configs:
      - to: 'admin@example.com'
        from: 'alerts@ai-gateway.com'
        smarthost: 'smtp.example.com:587'
```

**Slack通知**
```yaml
receivers:
  - name: 'team-slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
        channel: '#alerts'
```

---

## 日志管理

### 查看日志

#### 实时日志
```bash
# 所有服务
docker-compose logs -f

# 特定服务
docker-compose logs -f gateway

# 最近N行
docker-compose logs --tail=100 gateway
```

#### 过滤日志
```bash
# 错误日志
docker-compose logs gateway | grep -i error

# 特定时间段
docker-compose logs --since=2024-01-01T00:00:00 gateway

# 特定关键词
docker-compose logs gateway | grep "api_key_expired"
```

### 日志分析

#### 统计错误类型
```bash
docker-compose logs gateway | grep -i error | awk '{print $NF}' | sort | uniq -c | sort -rn
```

#### 统计API调用
```bash
docker-compose logs gateway | grep "POST /api" | awk '{print $7}' | sort | uniq -c
```

#### 分析响应时间
```bash
docker-compose logs gateway | grep "took" | awk -F'took ' '{print $2}' | sort -n
```

### 日志轮转

#### 配置日志轮转
```yaml
# docker-compose.yml
services:
  gateway:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 手动清理日志
```bash
# 清理Docker日志
docker-compose down
sudo rm -rf /var/lib/docker/containers/*/*-json.log
docker-compose up -d
```

---

## 性能维护

### 性能监控

#### 关键指标
```bash
# CPU使用率
docker stats --no-stream | grep ai-gateway

# 内存使用
docker exec ai-gateway cat /proc/meminfo | grep Mem

# 网络流量
docker exec ai-gateway cat /proc/net/dev

# 磁盘I/O
iostat -x 1 10
```

### 性能优化

#### Redis优化
```bash
# 查看Redis内存使用
docker exec ai-gateway-redis redis-cli info memory

# 查看慢查询
docker exec ai-gateway-redis redis-cli slowlog get 10

# 优化配置
docker exec ai-gateway-redis redis-cli config set maxmemory-policy allkeys-lru
```

#### 数据库优化
```bash
# 查看数据库大小
ls -lh /var/lib/docker/volumes/ai-gateway_gateway-data/_data/

# 优化SQLite
sqlite3 /path/to/ai-gateway.db "VACUUM;"
sqlite3 /path/to/ai-gateway.db "ANALYZE;"
```

#### 缓存优化
```bash
# 查看缓存命中率
docker exec ai-gateway-redis redis-cli info stats | grep keyspace

# 清理缓存
docker exec ai-gateway-redis redis-cli FLUSHDB

# 预热缓存
curl -X POST http://localhost:8000/api/v1/cache/warm
```

### 容量规划

#### 评估当前使用
```bash
# API调用量
curl http://localhost:9090/api/v1/query?query=increase(ai_gateway_api_requests_total[30d])

# 数据增长
docker exec ai-gateway-redis redis-cli info memory | grep used_memory

# 资源使用趋势
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
```

#### 扩容建议

**水平扩容**
- 增加Gateway实例
- 配置负载均衡
- 使用Redis集群

**垂直扩容**
- 增加CPU核心
- 扩大内存容量
- 使用SSD存储

---

## 安全管理

### 访问控制

#### 修改默认密码
```bash
# 修改Grafana密码
docker exec -it ai-gateway-grafana grafana-cli admin reset-admin-password new-password

# 修改环境变量
nano .env
# 更新 GRAFANA_ADMIN_PASSWORD
```

#### API密钥管理
```bash
# 查看当前密钥配置
grep API_KEY .env

# 轮换密钥
# 1. 生成新密钥
# 2. 更新.env文件
# 3. 重启服务
docker-compose restart gateway
```

### SSL/TLS配置

#### 生成自签名证书
```bash
# 生成私钥和证书
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout nginx.key -out nginx.crt

# 放置证书
mkdir -p deploy/nginx/ssl
cp nginx.key nginx.crt deploy/nginx/ssl/
```

#### 配置Nginx
```nginx
server {
    listen 443 ssl;
    ssl_certificate /etc/nginx/ssl/nginx.crt;
    ssl_certificate_key /etc/nginx/ssl/nginx.key;
    # ...
}
```

### 安全审计

#### 定期检查
```bash
# 检查开放的端口
netstat -tuln

# 检查运行的服务
docker-compose ps

# 查看访问日志中的异常
docker-compose logs gateway | grep -i "401\|403\|500"

# 扫描镜像漏洞
docker scan ai-gateway:latest
```

---

## 备份恢复

### 自动备份

#### 配置定时备份
```bash
# 编辑crontab
crontab -e

# 添加每日备份（凌晨2点）
0 2 * * * /path/to/ai-gateway/scripts/upgrade.sh --backup-only
```

### 手动备份

#### 备份所有数据
```bash
# 使用备份脚本
./scripts/upgrade.sh --backup-only

# 手动备份
docker-compose exec gateway tar czf /tmp/backup.tar.gz /app/data
docker cp ai-gateway:/tmp/backup.tar.gz ./backup-$(date +%Y%m%d).tar.gz
```

#### 备份特定组件
```bash
# 备份SQLite数据库
docker cp ai-gateway:/app/data/ai-gateway.db ./backup-db-$(date +%Y%m%d).db

# 备份Redis数据
docker exec ai-gateway-redis redis-cli BGSAVE
docker cp ai-gateway-redis:/data/dump.rdb ./backup-redis-$(date +%Y%m%d).rdb

# 备份配置文件
tar czf backup-config-$(date +%Y%m%d).tar.gz configs/ .env
```

### 数据恢复

#### 完整恢复
```bash
# 1. 停止服务
docker-compose down

# 2. 恢复数据
./scripts/upgrade.sh --rollback ./backups/backup_20240101_020000

# 3. 启动服务
docker-compose up -d

# 4. 验证
curl http://localhost:8000/health
```

#### 部分恢复
```bash
# 恢复数据库
docker-compose down
docker cp ./backup-db-20240101.db ai-gateway:/app/data/ai-gateway.db
docker-compose up -d

# 恢复Redis
docker-compose stop redis
docker cp ./backup-redis-20240101.rdb ai-gateway-redis:/data/dump.rdb
docker-compose start redis
```

---

## 应急响应

### 服务故障

#### 诊断步骤
```bash
# 1. 检查服务状态
docker-compose ps

# 2. 查看日志
docker-compose logs --tail=100 gateway

# 3. 检查资源
docker stats --no-stream

# 4. 测试连接
curl -v http://localhost:8000/health
```

#### 快速恢复
```bash
# 方案1: 重启服务
docker-compose restart gateway

# 方案2: 回滚版本
docker-compose down
docker tag ai-gateway:latest ai-gateway:backup
docker tag ai-gateway:previous ai-gateway:latest
docker-compose up -d

# 方案3: 恢复备份
./scripts/upgrade.sh --rollback ./backups/latest
```

### 数据丢失

#### 紧急恢复
```bash
# 1. 停止所有服务
docker-compose down

# 2. 检查备份
ls -lh backups/

# 3. 恢复最新备份
./scripts/upgrade.sh --rollback ./backups/backup_YYYYMMDD_HHMMSS

# 4. 验证数据
docker-compose up -d
curl http://localhost:8000/api/v1/accounts
```

### 安全事件

#### 应急处理
```bash
# 1. 立即隔离
docker-compose down

# 2. 保存证据
docker-compose logs > incident-$(date +%Y%m%d).log
docker commit ai-gateway incident-snapshot

# 3. 更换密钥
# 编辑.env文件，更新所有API密钥

# 4. 安全重启
docker-compose up -d
```

### 性能问题

#### 快速诊断
```bash
# 1. 检查资源使用
docker stats --no-stream

# 2. 查看慢查询
docker-compose logs gateway | grep "took.*[0-9]{4,}ms"

# 3. 检查Redis
docker exec ai-gateway-redis redis-cli --latency

# 4. 查看连接数
docker exec ai-gateway netstat -an | grep ESTABLISHED | wc -l
```

#### 临时缓解
```bash
# 1. 清理缓存
docker exec ai-gateway-redis redis-cli FLUSHDB

# 2. 限制流量
# 修改 .env: RATE_LIMIT=50

# 3. 扩容
docker-compose up -d --scale gateway=2

# 4. 重启服务
docker-compose restart gateway
```

---

## 运维工具

### 常用命令速查

```bash
# 服务管理
docker-compose up -d              # 启动
docker-compose down               # 停止
docker-compose restart            # 重启
docker-compose logs -f            # 日志
docker-compose ps                 # 状态

# 健康检查
curl http://localhost:8000/health # 网关健康
curl http://localhost:3000        # Web健康
docker exec ai-gateway-redis redis-cli ping  # Redis

# 监控
open http://localhost:9090        # Prometheus
open http://localhost:3001        # Grafana
open http://localhost:9093        # Alertmanager

# 备份
./scripts/upgrade.sh --backup-only    # 创建备份
./scripts/upgrade.sh --rollback PATH  # 恢复备份

# 快速启动
./deploy/quick-start.sh           # 一键启动
./deploy/verify-config.sh         # 验证配置
```

### 监控脚本

#### 健康检查脚本
```bash
#!/bin/bash
# health-check.sh

services=("gateway:8000" "web:3000" "redis:6379")

for service in "${services[@]}"; do
    name=$(echo $service | cut -d: -f1)
    port=$(echo $service | cut -d: -f2)

    if curl -f http://localhost:$port/health > /dev/null 2>&1; then
        echo "✓ $name is healthy"
    else
        echo "✗ $name is unhealthy"
    fi
done
```

#### 资源监控脚本
```bash
#!/bin/bash
# resource-monitor.sh

echo "=== Resource Usage ==="
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}"

echo ""
echo "=== Disk Usage ==="
df -h | grep -E "Filesystem|/dev/"

echo ""
echo "=== Docker Volumes ==="
docker system df -v | grep -E "VOLUME NAME|ai-gateway"
```

---

## 联系支持

### 获取帮助
- 📖 查看文档: `/docs` 目录
- 📝 查看日志: `docker-compose logs`
- 🔍 运行诊断: `./deploy/verify-config.sh`
- 💬 GitHub Issues: 报告问题

### 紧急联系
- **运维团队**: ops@example.com
- **技术支持**: support@example.com
- **紧急热线**: +86-xxx-xxxx-xxxx

---

**文档版本**: v1.0.0
**最后更新**: 2024-02-14
**维护者**: DevOps Team
