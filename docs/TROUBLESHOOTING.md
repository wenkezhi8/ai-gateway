# AI Gateway 故障排查指南

> 系统化的问题诊断和解决方案

---

## 目录

1. [诊断流程](#诊断流程)
2. [日志分析](#日志分析)
3. [常见问题诊断](#常见问题诊断)
4. [性能问题排查](#性能问题排查)
5. [网络问题排查](#网络问题排查)
6. [数据问题排查](#数据问题排查)
7. [应急处理](#应急处理)

---

## 诊断流程

### 总体排查思路

```
用户报告问题
    ↓
确认问题现象
    ↓
检查服务状态 → 服务是否运行？ → 否 → 启动服务
    ↓ 是
检查日志信息 → 有错误日志？ → 是 → 分析错误原因
    ↓ 否
检查配置文件 → 配置正确？ → 否 → 修正配置
    ↓ 是
检查网络连接 → 网络正常？ → 否 → 解决网络问题
    ↓ 是
检查上游服务 → 上游可用？ → 否 → 联系服务商
    ↓ 是
深入分析问题
```

### 快速诊断命令

```bash
# 1. 检查服务状态
docker-compose ps

# 2. 查看实时日志
docker-compose logs -f --tail=100

# 3. 检查端口占用
netstat -tlnp | grep 8080

# 4. 测试健康检查
curl http://localhost:8080/health

# 5. 检查资源使用
docker stats

# 6. 测试DNS解析
nslookup api.openai.com

# 7. 检查网络连通性
ping -c 4 api.openai.com
```

---

## 日志分析

### 日志位置

| 日志类型 | 位置 | 说明 |
|---------|------|------|
| 网关日志 | docker logs gateway | 主要服务日志 |
| Redis日志 | docker logs redis | 缓存服务日志 |
| 访问日志 | /var/log/gateway/access.log | HTTP请求日志 |
| 错误日志 | /var/log/gateway/error.log | 错误详情 |

### 查看日志

```bash
# 查看最近的日志
docker-compose logs --tail=100 gateway

# 实时查看日志
docker-compose logs -f gateway

# 搜索特定关键词
docker-compose logs gateway | grep "error"
docker-compose logs gateway | grep "500"

# 查看特定时间段的日志
docker-compose logs --since="2024-01-01T00:00:00" gateway
docker-compose logs --until="2024-01-01T23:59:59" gateway
```

### 日志级别说明

| 级别 | 含义 | 示例 |
|------|------|------|
| DEBUG | 调试信息 | 请求详情、变量值 |
| INFO | 正常信息 | 请求完成、服务启动 |
| WARN | 警告信息 | 配置缺失、性能下降 |
| ERROR | 错误信息 | 请求失败、服务异常 |
| FATAL | 致命错误 | 服务崩溃、无法启动 |

### 常见日志模式

#### 1. 正常请求日志
```
[INFO] 2024/01/01 10:00:00 Request completed: method=POST path=/api/v1/chat/completions status=200 duration=1234ms
```

#### 2. 认证失败日志
```
[ERROR] 2024/01/01 10:00:00 Authentication failed: invalid API key ip=192.168.1.100
```

#### 3. 速率限制日志
```
[WARN] 2024/01/01 10:00:00 Rate limit exceeded: user=user123 limit=60/min current=65
```

#### 4. 上游服务错误日志
```
[ERROR] 2024/01/01 10:00:00 Upstream error: provider=openai error="connection timeout"
```

---

## 常见问题诊断

### 问题1：服务无法启动

**诊断步骤**：

```bash
# 1. 检查配置文件语法
cat configs/config.json | python -m json.tool

# 2. 检查端口占用
lsof -i :8080

# 3. 检查依赖服务
docker-compose ps

# 4. 查看启动错误
docker-compose logs gateway | head -50
```

**常见原因与解决**：

| 原因 | 解决方案 |
|------|---------|
| 端口被占用 | 修改端口或结束占用进程 |
| 配置文件错误 | 检查JSON语法 |
| 依赖服务未启动 | 先启动Redis |
| 权限不足 | 使用sudo或调整权限 |

---

### 问题2：API请求返回500错误

**诊断步骤**：

```bash
# 1. 查看详细错误
docker-compose logs gateway | grep "500" -A 5

# 2. 检查上游服务
curl -v https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# 3. 检查Redis连接
docker exec -it redis redis-cli ping
```

**解决方案流程图**：

```
500错误
  ├── 配置问题
  │     └── 检查config.json中的providers配置
  ├── 上游服务问题
  │     ├── API Key无效 → 重新配置
  │     ├── 余额不足 → 充值
  │     └── 服务不可用 → 等待恢复
  ├── 网络问题
  │     └── 检查代理设置
  └── 内部错误
        └── 查看stack trace，提交issue
```

---

### 问题3：认证失败（401）

**诊断步骤**：

```bash
# 1. 测试无认证端点
curl http://localhost:8080/health

# 2. 测试带认证端点
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer YOUR_KEY"

# 3. 检查认证配置
cat configs/config.json | jq '.auth'
```

**常见原因**：

1. **Authorization header格式错误**
```bash
# 错误
-H "Authorization: YOUR_KEY"
-H "Token: Bearer YOUR_KEY"

# 正确
-H "Authorization: Bearer YOUR_KEY"
```

2. **API Key未配置或错误**
```bash
# 检查环境变量
echo $OPENAI_API_KEY

# 检查配置文件
grep "api_key" configs/config.json
```

---

### 问题4：请求超时

**诊断步骤**：

```bash
# 1. 测试网络延迟
ping api.openai.com

# 2. 测试DNS解析
nslookup api.openai.com

# 3. 检查代理设置
env | grep -i proxy

# 4. 测试直接连接
curl -w "@curl-format.txt" -o /dev/null -s https://api.openai.com/v1/models
```

**curl-format.txt内容**：
```
time_namelookup:  %{time_namelookup}\n
time_connect:  %{time_connect}\n
time_appconnect:  %{time_appconnect}\n
time_starttransfer:  %{time_starttransfer}\n
time_total:  %{time_total}\n
```

**解决方案**：

```json
{
  "client": {
    "timeout": 60,
    "dial_timeout": 10,
    "tls_handshake_timeout": 10,
    "response_header_timeout": 10
  }
}
```

---

## 性能问题排查

### 问题5：响应速度慢

**诊断命令**：

```bash
# 1. 检查系统资源
top
htop
docker stats

# 2. 检查网络延迟
traceroute api.openai.com

# 3. 检查Redis性能
docker exec redis redis-cli --latency

# 4. 分析请求耗时
curl -w "Total: %{time_total}s\n" http://localhost:8080/api/v1/chat/completions
```

**性能分析检查清单**：

```
□ CPU使用率是否过高？ (>80%)
□ 内存是否不足？ (>90%)
□ 磁盘IO是否过高？
□ 网络带宽是否饱和？
□ Redis响应是否正常？
□ 上游服务是否慢？
□ 是否有大量慢请求？
□ 缓存命中率如何？
```

**优化方案**：

1. **启用缓存**：
```json
{
  "cache": {
    "enabled": true,
    "ttl": 3600
  }
}
```

2. **增加Worker**：
```json
{
  "server": {
    "workers": 4
  }
}
```

3. **使用连接池**：
```json
{
  "pool": {
    "max_idle": 100,
    "max_active": 200
  }
}
```

---

### 问题6：内存占用过高

**诊断步骤**：

```bash
# 1. 查看进程内存
ps aux --sort=-%mem | head

# 2. 查看容器资源
docker stats --no-stream

# 3. 检查缓存大小
docker exec redis redis-cli info memory

# 4. 分析内存泄漏（需要pprof）
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**解决方案**：

```json
{
  "cache": {
    "max_size": 10000,  // 限制缓存条目数
    "max_memory_mb": 512  // 限制内存使用
  }
}
```

---

## 网络问题排查

### 问题7：无法连接上游服务

**诊断流程**：

```
无法连接
    ↓
检查DNS → 能解析？ → 否 → 配置DNS/hosts
    ↓ 是
检查网络 → 能ping通？ → 否 → 检查防火墙/代理
    ↓ 是
检查TLS → 证书有效？ → 否 → 更新证书
    ↓ 是
检查代理 → 代理正确？ → 否 → 配置代理
    ↓ 是
联系服务商
```

**诊断命令**：

```bash
# 1. DNS解析
dig api.openai.com
nslookup api.openai.com

# 2. 网络连通性
ping -c 4 api.openai.com
traceroute api.openai.com

# 3. 端口连通性
telnet api.openai.com 443
nc -zv api.openai.com 443

# 4. TLS检查
openssl s_client -connect api.openai.com:443

# 5. 完整请求测试
curl -v https://api.openai.com/v1/models
```

**代理配置**：

```bash
# 环境变量方式
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080

# 配置文件方式
{
  "proxy": {
    "enabled": true,
    "url": "http://proxy.example.com:8080"
  }
}
```

---

### 问题8：间歇性网络故障

**诊断方法**：

```bash
# 1. 持续ping测试
ping -i 1 api.openai.com | tee ping.log

# 2. 请求成功率统计
for i in {1..100}; do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health
done | sort | uniq -c

# 3. 网络抖动检测
mtr --report api.openai.com
```

**解决方案**：

```json
{
  "retry": {
    "enabled": true,
    "max_attempts": 3,
    "wait_time_ms": 1000,
    "backoff_multiplier": 2
  }
}
```

---

## 数据问题排查

### 问题9：Redis连接失败

**诊断步骤**：

```bash
# 1. 检查Redis状态
docker-compose ps redis

# 2. 测试连接
docker exec redis redis-cli ping

# 3. 检查网络
docker exec gateway ping redis

# 4. 查看Redis日志
docker-compose logs redis
```

**常见错误与解决**：

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| Connection refused | Redis未启动 | `docker-compose up -d redis` |
| NOAUTH | 需要密码 | 配置redis密码 |
| LOADING | 数据加载中 | 等待加载完成 |
| MAXCLIENTS | 连接数超限 | 调整maxclients |

---

### 问题10：缓存数据不一致

**诊断命令**：

```bash
# 1. 查看缓存内容
docker exec redis redis-cli KEYS "*"

# 2. 查看特定key
docker exec redis redis-cli GET "cache:some-key"

# 3. 查看TTL
docker exec redis redis-cli TTL "cache:some-key"

# 4. 清空缓存
docker exec redis redis-cli FLUSHDB
```

**解决方案**：

```json
{
  "cache": {
    "consistency_check": true,
    "ttl": 3600,
    "refresh_ahead": 300  // 提前5分钟刷新
  }
}
```

---

## 应急处理

### 紧急故障处理流程

```
发现故障
    ↓
[1] 确认影响范围
    ↓
[2] 保存现场（日志、截图）
    ↓
[3] 尝试快速恢复
    ├── 重启服务
    ├── 回滚版本
    └── 切换备用服务
    ↓
[4] 通知相关人员
    ↓
[5] 根因分析
    ↓
[6] 修复并验证
    ↓
[7] 编写故障报告
```

### 快速恢复命令

```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart gateway

# 回滚到上一版本
git checkout HEAD~1
docker-compose up -d --build

# 扩容服务
docker-compose up -d --scale gateway=3

# 切换到备用配置
cp configs/config.backup.json configs/config.json
docker-compose restart gateway
```

### 故障报告模板

```markdown
## 故障报告

### 基本信息
- 故障时间：YYYY-MM-DD HH:MM - HH:MM
- 故障等级：P1/P2/P3
- 影响范围：[描述影响的服务和用户]

### 故障现象
[描述用户看到的问题]

### 根本原因
[经过分析后的根本原因]

### 处理过程
1. XX:XX 发现问题
2. XX:XX 开始排查
3. XX:XX 定位原因
4. XX:XX 实施修复
5. XX:XX 验证恢复

### 后续改进
- [ ] 改进项1
- [ ] 改进项2

### 相关日志
```
[粘贴关键日志]
```
```

---

## 联系支持

如果按照本指南仍无法解决问题：

1. 收集诊断信息：
```bash
# 一键收集诊断信息
./scripts/collect-diagnostics.sh > diagnostic-report.txt
```

2. 提交Issue：
   - 描述问题现象
   - 附上诊断报告
   - 说明复现步骤

3. 紧急联系：
   - 技术支持邮箱
   - 在线客服

---

**最后更新**：2024年1月
