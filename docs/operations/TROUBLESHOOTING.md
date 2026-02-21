# AI Gateway - 故障排查指南

## 目录

1. [常见问题](#常见问题)
2. [服务启动问题](#服务启动问题)
3. [API请求问题](#api请求问题)
4. [性能问题](#性能问题)
5. [网络问题](#网络问题)
6. [数据问题](#数据问题)
7. [监控问题](#监控问题)
8. [错误代码参考](#错误代码参考)

---

## 常见问题

### Q1: 服务无法启动

**症状:**
```
Error: port is already allocated
```

**原因:** 端口被占用

**解决方案:**
```bash
# 1. 查找占用端口的进程
lsof -i :8000

# 2. 停止占用进程
kill -9 <PID>

# 3. 或修改端口
# 编辑 .env 文件
GATEWAY_PORT=8001
```

### Q2: Docker容器频繁重启

**症状:**
```
docker-compose ps
# 显示 Restarting 状态
```

**诊断步骤:**
```bash
# 1. 查看日志
docker-compose logs gateway

# 2. 检查健康检查
docker inspect ai-gateway | grep -A 10 Health

# 3. 检查资源限制
docker stats --no-stream
```

**常见原因及解决:**
- 内存不足: 增加内存限制或释放内存
- 配置错误: 检查 `configs/config.json`
- API密钥无效: 验证 `.env` 中的密钥

### Q3: API请求返回401

**症状:**
```json
{
  "error": "Unauthorized",
  "message": "Invalid API key"
}
```

**检查清单:**
- [ ] API密钥是否正确配置
- [ ] API密钥是否过期
- [ ] API密钥是否有足够权限
- [ ] 请求头是否正确

**解决方案:**
```bash
# 1. 检查配置
grep API_KEY .env

# 2. 测试API密钥
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/models

# 3. 更新密钥
nano .env
docker-compose restart gateway
```

### Q4: 响应时间过慢

**症状:** P99 > 1秒

**诊断:**
```bash
# 1. 检查网络延迟
ping api.openai.com

# 2. 检查Redis性能
docker exec ai-gateway-redis redis-cli --latency

# 3. 查看慢查询日志
docker-compose logs gateway | grep "took.*[0-9]{4,}ms"

# 4. 检查资源使用
docker stats --no-stream
```

**优化建议:**
- 启用缓存
- 增加并发限制
- 优化网络配置
- 扩容资源

---

## 服务启动问题

### 问题: Docker守护进程未运行

**错误信息:**
```
Cannot connect to the Docker daemon
```

**解决方案:**

**macOS:**
```bash
open -a Docker
```

**Linux:**
```bash
sudo systemctl start docker
sudo systemctl enable docker
```

**Windows:**
```bash
# 启动Docker Desktop
# 或通过服务管理器启动
```

### 问题: 镜像拉取失败

**错误信息:**
```
Error: pull access denied
```

**解决方案:**
```bash
# 1. 检查网络连接
ping hub.docker.com

# 2. 使用国内镜像源（中国用户）
# 编辑 /etc/docker/daemon.json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn"
  ]
}

# 3. 重启Docker
sudo systemctl restart docker

# 4. 重新拉取
docker-compose pull
```

### 问题: 卷挂载失败

**错误信息:**
```
Error: mount denied
```

**解决方案:**

**macOS:**
```bash
# 在Docker Desktop中添加文件共享路径
# Preferences → Resources → File Sharing
```

**Linux:**
```bash
# 检查SELinux配置
sudo setenforce 0  # 临时关闭

# 或添加SELinux策略
chcon -Rt svirt_sandbox_file_t /path/to/volume
```

### 问题: 环境变量未加载

**症状:** 配置未生效

**检查:**
```bash
# 1. 验证.env文件存在
ls -la .env

# 2. 检查文件格式
file .env

# 3. 查看变量值
docker-compose config | grep API_KEY
```

**解决方案:**
```bash
# 重新创建.env
cp .env.example .env
nano .env

# 重启服务
docker-compose down
docker-compose up -d
```

---

## API请求问题

### 问题: 请求超时

**错误信息:**
```
Error: context deadline exceeded
```

**诊断:**
```bash
# 1. 测试API提供商连接
curl -w "@curl-format.txt" -o /dev/null -s https://api.openai.com/v1/models

# 2. 检查DNS解析
nslookup api.openai.com

# 3. 查看超时配置
grep TIMEOUT .env
```

**解决方案:**
```bash
# 1. 增加超时时间
# .env
REQUEST_TIMEOUT=120

# 2. 检查网络代理设置
unset http_proxy
unset https_proxy

# 3. 使用更快的DNS
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
```

### 问题: 速率限制

**错误信息:**
```json
{
  "error": {
    "type": "rate_limit_exceeded",
    "message": "You exceeded your current quota"
  }
}
```

**诊断:**
```bash
# 1. 查看当前限流配置
grep RATE .env

# 2. 检查Redis限流计数
docker exec ai-gateway-redis redis-cli keys "*rate*"

# 3. 查看API配额
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/usage
```

**解决方案:**
```bash
# 1. 调整限流配置
# .env
RATE_LIMIT=50  # 降低请求速率

# 2. 使用备份账号
# configs/config.json 中启用 backup account

# 3. 清除限流缓存
docker exec ai-gateway-redis redis-cli FLUSHDB

# 4. 等待配额重置（通常每小时/每天）
```

### 问题: 模型不可用

**错误信息:**
```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "The model `gpt-5` does not exist"
  }
}
```

**解决方案:**
```bash
# 1. 查看可用模型
curl http://localhost:8000/api/v1/models

# 2. 检查配置
grep -r "gpt-5" configs/

# 3. 更新模型配置
# configs/config.json
# 将 gpt-5 改为 gpt-4 或 gpt-3.5-turbo

# 4. 重启服务
docker-compose restart gateway
```

### 问题: 请求体过大

**错误信息:**
```
Error: request body too large
```

**解决方案:**
```bash
# 1. 检查当前限制
grep MAX_REQUEST_SIZE .env

# 2. 增加限制
# .env
MAX_REQUEST_SIZE=50  # MB

# 3. 或分批处理请求
# 减少单次请求的tokens数量
```

---

## 性能问题

### 问题: 内存使用过高

**症状:** 容器内存接近限制

**诊断:**
```bash
# 1. 查看内存使用
docker stats --no-stream | grep ai-gateway

# 2. 查看进程内存
docker exec ai-gateway top -b -n 1

# 3. 查看Redis内存
docker exec ai-gateway-redis redis-cli info memory
```

**解决方案:**
```bash
# 1. 清理缓存
docker exec ai-gateway-redis redis-cli FLUSHDB

# 2. 调整Redis内存限制
# .env
REDIS_MAXMEMORY=128mb

# 3. 增加容器内存
# docker-compose.yml
deploy:
  resources:
    limits:
      memory: 4G

# 4. 重启服务
docker-compose up -d
```

### 问题: CPU使用过高

**症状:** CPU持续100%

**诊断:**
```bash
# 1. 查看CPU使用
docker stats --no-stream | grep ai-gateway

# 2. 查看进程CPU
docker exec ai-gateway top -b -n 1

# 3. 查看goroutine数量
curl http://localhost:8000/debug/pprof/goroutine?debug=1
```

**解决方案:**
```bash
# 1. 限制并发
# .env
RATE_LIMIT=50
RATE_BURST=100

# 2. 优化代码
# 检查是否有死循环或goroutine泄漏

# 3. 增加CPU核心
# docker-compose.yml
deploy:
  resources:
    limits:
      cpus: '4'

# 4. 重启服务
docker-compose restart gateway
```

### 问题: 磁盘空间不足

**错误信息:**
```
Error: no space left on device
```

**诊断:**
```bash
# 1. 查看磁盘使用
df -h

# 2. 查看Docker占用
docker system df

# 3. 查看卷大小
docker volume ls
docker volume inspect ai-gateway_gateway-data
```

**解决方案:**
```bash
# 1. 清理Docker资源
docker system prune -a --volumes

# 2. 清理旧日志
sudo rm -rf /var/lib/docker/containers/*/*-json.log

# 3. 清理旧备份
rm -rf backups/backup_2023*

# 4. 扩容磁盘
# 根据云服务商文档操作
```

### 问题: 数据库性能差

**症状:** SQLite查询慢

**诊断:**
```bash
# 1. 查看数据库大小
ls -lh /var/lib/docker/volumes/ai-gateway_gateway-data/_data/

# 2. 查看表大小
docker exec ai-gateway sqlite3 /app/data/ai-gateway.db \
  "SELECT name, SUM(pgsize) FROM dbstat GROUP BY name;"
```

**解决方案:**
```bash
# 1. 优化数据库
docker exec ai-gateway sqlite3 /app/data/ai-gateway.db "VACUUM;"
docker exec ai-gateway sqlite3 /app/data/ai-gateway.db "ANALYZE;"

# 2. 清理旧数据
docker exec ai-gateway sqlite3 /app/data/ai-gateway.db \
  "DELETE FROM usage_logs WHERE created_at < datetime('now', '-30 days');"

# 3. 考虑迁移到PostgreSQL（大量数据时）
```

---

## 网络问题

### 问题: DNS解析失败

**错误信息:**
```
Error: no such host
```

**诊断:**
```bash
# 1. 测试DNS
nslookup api.openai.com

# 2. 测试连接
ping api.openai.com

# 3. 查看DNS配置
cat /etc/resolv.conf
```

**解决方案:**
```bash
# 1. 使用公共DNS
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
echo "nameserver 1.1.1.1" | sudo tee -a /etc/resolv.conf

# 2. 或在Docker中配置
# docker-compose.yml
dns:
  - 8.8.8.8
  - 1.1.1.1

# 3. 检查防火墙
sudo iptables -L -n | grep DROP
```

### 问题: 连接被拒绝

**错误信息:**
```
Error: connection refused
```

**诊断:**
```bash
# 1. 检查端口
netstat -tuln | grep 8000

# 2. 检查防火墙
sudo iptables -L -n

# 3. 测试本地连接
curl http://localhost:8000/health
```

**解决方案:**
```bash
# 1. 检查服务状态
docker-compose ps

# 2. 重启服务
docker-compose restart gateway

# 3. 检查端口绑定
docker-compose down
docker-compose up -d
```

### 问题: SSL证书错误

**错误信息:**
```
Error: certificate verify failed
```

**解决方案:**
```bash
# 1. 更新CA证书
# macOS
brew install ca-certificates

# Ubuntu/Debian
sudo apt-get update && sudo apt-get install ca-certificates

# 2. 或跳过证书验证（仅测试环境）
# .env
NODE_TLS_REJECT_UNAUTHORIZED=0

# 3. 或添加自定义证书
# 将证书放到指定位置并配置
```

---

## 数据问题

### 问题: Redis连接失败

**错误信息:**
```
Error: redis connection refused
```

**诊断:**
```bash
# 1. 检查Redis状态
docker-compose ps redis

# 2. 测试连接
docker exec ai-gateway-redis redis-cli ping

# 3. 查看日志
docker-compose logs redis
```

**解决方案:**
```bash
# 1. 重启Redis
docker-compose restart redis

# 2. 检查配置
grep REDIS .env

# 3. 检查网络
docker network inspect ai-gateway-network

# 4. 清理Redis数据（最后手段）
docker-compose stop redis
docker volume rm ai-gateway_redis-data
docker-compose up -d
```

### 问题: 数据丢失

**症状:** 数据意外消失

**诊断:**
```bash
# 1. 检查卷状态
docker volume ls | grep ai-gateway

# 2. 查看数据目录
docker exec ai-gateway ls -la /app/data/

# 3. 检查备份
ls -lh backups/
```

**解决方案:**
```bash
# 1. 停止服务
docker-compose down

# 2. 恢复备份
./scripts/upgrade.sh --rollback ./backups/backup_YYYYMMDD_HHMMSS

# 3. 或手动恢复
docker cp ./backup-db.db ai-gateway:/app/data/ai-gateway.db

# 4. 重启并验证
docker-compose up -d
curl http://localhost:8000/api/v1/accounts
```

### 问题: 缓存失效

**症状:** 缓存命中率低

**诊断:**
```bash
# 1. 查看缓存统计
docker exec ai-gateway-redis redis-cli info stats | grep keyspace

# 2. 查看缓存键
docker exec ai-gateway-redis redis-cli keys "*"

# 3. 查看缓存配置
grep CACHE .env
```

**解决方案:**
```bash
# 1. 检查缓存TTL
docker exec ai-gateway-redis redis-cli ttl <key>

# 2. 调整缓存配置
# .env
CACHE_ENABLED=true
CACHE_TTL=7200  # 增加TTL
CACHE_MAX_SIZE=500  # 增加大小

# 3. 预热缓存
curl -X POST http://localhost:8000/api/v1/cache/warm

# 4. 重启服务
docker-compose restart gateway
```

---

## 监控问题

### 问题: Prometheus无法抓取指标

**错误信息:**
```
Error: no data in Prometheus
```

**诊断:**
```bash
# 1. 测试metrics端点
curl http://localhost:8000/metrics

# 2. 检查Prometheus配置
docker exec ai-gateway-prometheus cat /etc/prometheus/prometheus.yml

# 3. 查看Prometheus日志
docker-compose logs prometheus

# 4. 检查目标状态
curl http://localhost:9090/api/v1/targets
```

**解决方案:**
```bash
# 1. 验证网络
docker network inspect ai-gateway-network

# 2. 检查服务发现
# monitoring/prometheus.yml
static_configs:
  - targets: ['gateway:8080']  # 确保服务名正确

# 3. 重启Prometheus
docker-compose restart prometheus

# 4. 检查防火墙
# 确保9090端口可访问
```

### 问题: Grafana无法连接数据源

**错误信息:**
```
Error: data source connection failed
```

**解决方案:**
```bash
# 1. 检查数据源配置
# monitoring/grafana/provisioning/datasources/datasources.yml

# 2. 测试Prometheus连接
docker exec ai-gateway-grafana curl http://prometheus:9090/api/v1/query?query=up

# 3. 重启Grafana
docker-compose restart grafana

# 4. 手动配置数据源
# 访问 http://localhost:3001 → Configuration → Data Sources
```

### 问题: 告警未触发

**症状:** 问题发生但未收到告警

**诊断:**
```bash
# 1. 检查告警规则
curl http://localhost:9090/api/v1/rules

# 2. 查看Alertmanager日志
docker-compose logs alertmanager

# 3. 测试告警
curl -X POST http://localhost:9093/api/v1/alerts \
  -d '[{"labels":{"alertname":"TestAlert"}}]'
```

**解决方案:**
```bash
# 1. 检查告警配置
# monitoring/alertmanager.yml

# 2. 验证通知配置
# 检查邮件/Slack配置

# 3. 测试通知
# 在Alertmanager UI中发送测试告警

# 4. 重启Alertmanager
docker-compose restart alertmanager
```

---

## 错误代码参考

### HTTP状态码

| 状态码 | 含义 | 常见原因 |
|--------|------|----------|
| 200 | 成功 | 请求正常处理 |
| 400 | 请求错误 | 参数错误、格式错误 |
| 401 | 未授权 | API密钥无效或缺失 |
| 403 | 禁止访问 | 权限不足 |
| 404 | 未找到 | 路由或资源不存在 |
| 429 | 请求过多 | 超过速率限制 |
| 500 | 服务器错误 | 内部错误 |
| 502 | 网关错误 | 上游服务不可用 |
| 503 | 服务不可用 | 服务过载或维护 |
| 504 | 网关超时 | 上游服务响应超时 |

### 自定义错误码

| 错误码 | 含义 | 解决方案 |
|--------|------|----------|
| E001 | 配置文件错误 | 检查configs/config.json |
| E002 | Redis连接失败 | 检查Redis服务状态 |
| E003 | 数据库错误 | 检查SQLite文件权限 |
| E004 | API密钥无效 | 更新.env中的API密钥 |
| E005 | 限流触发 | 降低请求频率 |
| E006 | 缓存错误 | 重启Redis服务 |
| E007 | 路由错误 | 检查路由配置 |
| E008 | 提供商不可用 | 检查API提供商状态 |

### API提供商错误

#### OpenAI错误

| 错误类型 | 说明 | 解决方案 |
|----------|------|----------|
| invalid_api_key | API密钥无效 | 检查密钥格式 |
| insufficient_quota | 配额不足 | 充值或等待重置 |
| rate_limit_exceeded | 超过限制 | 降低请求频率 |
| model_not_found | 模型不存在 | 使用正确的模型名 |
| context_length_exceeded | Token超限 | 减少输入长度 |

#### Anthropic错误

| 错误类型 | 说明 | 解决方案 |
|----------|------|----------|
| authentication_error | 认证失败 | 检查API密钥 |
| permission_error | 权限不足 | 检查账户权限 |
| not_found_error | 资源不存在 | 检查请求路径 |
| rate_limit_error | 限流 | 降低请求频率 |
| api_error | API错误 | 稍后重试 |

---

## 故障排查流程

### 标准排查流程

```
1. 收集信息
   ├─ 查看错误信息
   ├─ 查看日志
   └─ 查看监控数据

2. 定位问题
   ├─ 确定问题类型
   ├─ 确定影响范围
   └─ 确定紧急程度

3. 快速恢复
   ├─ 尝试重启服务
   ├─ 尝试回滚版本
   └─ 尝试恢复备份

4. 根本分析
   ├─ 分析日志
   ├─ 分析配置
   └─ 分析代码

5. 彻底修复
   ├─ 修复配置
   ├─ 修复代码
   └─ 更新文档

6. 预防措施
   ├─ 添加监控
   ├─ 添加告警
   └─ 更新流程
```

### 紧急联系

**无法解决时：**
1. 保存现场日志
2. 记录复现步骤
3. 联系技术支持
4. 等待专家协助

---

**文档版本**: v1.0.0
**最后更新**: 2024-02-14
**维护者**: DevOps Team
