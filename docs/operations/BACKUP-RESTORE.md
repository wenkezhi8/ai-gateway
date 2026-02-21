# AI Gateway - 备份与恢复手册

## 概述

本文档详细说明AI Gateway的备份策略、备份操作和恢复流程。

---

## 备份策略

### 备份内容

| 组件 | 内容 | 位置 | 重要性 |
|------|------|------|--------|
| SQLite数据库 | 账号、配置、用量数据 | `/app/data/ai-gateway.db` | 🔴 关键 |
| Redis数据 | 缓存、会话、限流计数 | `/data/dump.rdb` | 🟡 重要 |
| 配置文件 | 应用配置 | `configs/` | 🔴 关键 |
| 环境变量 | 密钥和配置 | `.env` | 🔴 关键 |
| 监控数据 | Prometheus指标 | Docker卷 | 🟢 可选 |

### 备份频率

| 备份类型 | 频率 | 保留期限 | 存储位置 |
|----------|------|----------|----------|
| 完整备份 | 每日 | 30天 | 本地+远程 |
| 增量备份 | 每小时 | 7天 | 本地 |
| 配置备份 | 每次变更 | 90天 | Git仓库 |

---

## 自动备份

### 配置定时备份

#### 使用Cron（推荐）

```bash
# 1. 编辑crontab
crontab -e

# 2. 添加备份任务
# 每天凌晨2点执行完整备份
0 2 * * * cd /path/to/ai-gateway && ./scripts/upgrade.sh --backup-only >> /var/log/ai-gateway-backup.log 2>&1

# 每小时执行增量备份
0 * * * * cd /path/to/ai-gateway && ./scripts/backup-incremental.sh >> /var/log/ai-gateway-backup.log 2>&1
```

#### 使用Systemd Timer

```bash
# 1. 创建服务文件
sudo nano /etc/systemd/system/ai-gateway-backup.service
```

```ini
[Unit]
Description=AI Gateway Backup
After=docker.service

[Service]
Type=oneshot
ExecStart=/path/to/ai-gateway/scripts/upgrade.sh --backup-only
User=root

[Install]
WantedBy=multi-user.target
```

```bash
# 2. 创建定时器
sudo nano /etc/systemd/system/ai-gateway-backup.timer
```

```ini
[Unit]
Description=AI Gateway Backup Timer

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
# 3. 启用定时器
sudo systemctl enable ai-gateway-backup.timer
sudo systemctl start ai-gateway-backup.timer
```

---

## 手动备份

### 完整备份

#### 使用备份脚本（推荐）

```bash
# 执行备份
./scripts/upgrade.sh --backup-only

# 查看备份
ls -lh backups/
```

#### 手动完整备份

```bash
#!/bin/bash
# 手动完整备份脚本

BACKUP_DIR="./backups/backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "Starting backup to $BACKUP_DIR..."

# 1. 备份数据库
echo "Backing up database..."
docker cp ai-gateway:/app/data "$BACKUP_DIR/data"

# 2. 备份Redis
echo "Backing up Redis..."
docker exec ai-gateway-redis redis-cli BGSAVE
sleep 5
docker cp ai-gateway-redis:/data/dump.rdb "$BACKUP_DIR/redis-dump.rdb"

# 3. 备份配置
echo "Backing up configuration..."
cp -r configs "$BACKUP_DIR/"
cp .env "$BACKUP_DIR/"
cp docker-compose.yml "$BACKUP_DIR/"

# 4. 备份监控数据（可选）
echo "Backing up monitoring data..."
docker cp ai-gateway-prometheus:/prometheus "$BACKUP_DIR/prometheus-data" 2>/dev/null || true

# 5. 创建元数据
cat > "$BACKUP_DIR/metadata.json" <<EOF
{
  "timestamp": "$(date -Iseconds)",
  "version": "$(cat VERSION 2>/dev/null || echo 'unknown')",
  "git_commit": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
}
EOF

# 6. 压缩备份
echo "Compressing backup..."
tar czf "$BACKUP_DIR.tar.gz" -C "$(dirname $BACKUP_DIR)" "$(basename $BACKUP_DIR)"
rm -rf "$BACKUP_DIR"

echo "Backup completed: $BACKUP_DIR.tar.gz"
echo "Size: $(du -h "$BACKUP_DIR.tar.gz" | cut -f1)"
```

### 组件单独备份

#### 数据库备份

```bash
# 在线备份
docker exec ai-gateway sqlite3 /app/data/ai-gateway.db ".backup /tmp/backup.db"
docker cp ai-gateway:/tmp/backup.db ./backup-db-$(date +%Y%m%d).db

# 离线备份
docker-compose stop gateway
docker cp ai-gateway:/app/data/ai-gateway.db ./backup-db-$(date +%Y%m%d).db
docker-compose start gateway
```

#### Redis备份

```bash
# 触发RDB快照
docker exec ai-gateway-redis redis-cli BGSAVE

# 等待完成
docker exec ai-gateway-redis redis-cli LASTSAVE

# 复制备份文件
docker cp ai-gateway-redis:/data/dump.rdb ./backup-redis-$(date +%Y%m%d).rdb
```

#### 配置备份

```bash
# 备份所有配置
tar czf backup-config-$(date +%Y%m%d).tar.gz \
  configs/ \
  .env \
  docker-compose.yml \
  deploy/
```

---

## 增量备份

### 增量备份脚本

```bash
#!/bin/bash
# incremental-backup.sh

BACKUP_DIR="./backups/incremental"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LAST_BACKUP=$(ls -td $BACKUP_DIR/backup_* 2>/dev/null | head -1)

mkdir -p "$BACKUP_DIR/backup_$TIMESTAMP"

if [ -z "$LAST_BACKUP" ]; then
    echo "No previous backup found, creating full backup"
    # 执行完整备份
    docker cp ai-gateway:/app/data "$BACKUP_DIR/backup_$TIMESTAMP/"
else
    echo "Creating incremental backup since $LAST_BACKUP"

    # 只备份变更的文件
    rsync -av --compare-dest="$LAST_BACKUP" \
        /var/lib/docker/volumes/ai-gateway_gateway-data/_data/ \
        "$BACKUP_DIR/backup_$TIMESTAMP/"
fi

# 清理旧备份（保留最近24个）
cd $BACKUP_DIR
ls -t | tail -n +25 | xargs rm -rf

echo "Incremental backup completed"
```

---

## 远程备份

### 同步到云存储

#### AWS S3

```bash
# 安装AWS CLI
pip install awscli

# 配置AWS凭证
aws configure

# 同步备份
aws s3 sync ./backups/ s3://your-bucket/ai-gateway-backups/ \
  --storage-class STANDARD_IA \
  --delete
```

#### 阿里云OSS

```bash
# 安装ossutil
wget http://gosspublic.alicdn.com/ossutil/1.7.0/ossutil64
chmod 755 ossutil64

# 配置
./ossutil64 config

# 上传备份
./ossutil64 cp -r ./backups/ oss://your-bucket/ai-gateway-backups/
```

#### 腾讯云COS

```bash
# 安装coscmd
pip install coscmd

# 配置
coscmd config -a <secret_id> -s <secret_key> -b <bucket> -r <region>

# 上传
coscmd upload -r ./backups/ /ai-gateway-backups/
```

---

## 数据恢复

### 完整恢复

#### 使用恢复脚本（推荐）

```bash
# 1. 停止服务
docker-compose down

# 2. 恢复备份
./scripts/upgrade.sh --rollback ./backups/backup_20240114_020000

# 3. 启动服务
docker-compose up -d

# 4. 验证
curl http://localhost:8000/health
curl http://localhost:8000/api/v1/accounts
```

#### 手动完整恢复

```bash
#!/bin/bash
# 手动恢复脚本

BACKUP_FILE="./backups/backup_20240114_020000.tar.gz"

echo "Warning: This will overwrite all current data!"
read -p "Continue? (yes/no) " confirm

if [ "$confirm" != "yes" ]; then
    echo "Restore cancelled"
    exit 1
fi

# 1. 停止服务
echo "Stopping services..."
docker-compose down

# 2. 解压备份
echo "Extracting backup..."
tar xzf "$BACKUP_FILE" -C /tmp/

BACKUP_DIR=$(tar tzf "$BACKUP_FILE" | head -1 | cut -f1 -d"/")

# 3. 恢复数据库
echo "Restoring database..."
docker volume create ai-gateway_gateway-data
docker run --rm -v ai-gateway_gateway-data:/data \
    -v /tmp/$BACKUP_DIR/data:/backup \
    alpine cp -a /backup/. /data/

# 4. 恢复Redis
echo "Restoring Redis..."
docker volume create ai-gateway_redis-data
docker run --rm -v ai-gateway_redis-data:/data \
    -v /tmp/$BACKUP_DIR:/backup \
    alpine cp /backup/redis-dump.rdb /data/dump.rdb

# 5. 恢复配置
echo "Restoring configuration..."
cp -r /tmp/$BACKUP_DIR/configs ./
cp /tmp/$BACKUP_DIR/.env ./
cp /tmp/$BACKUP_DIR/docker-compose.yml ./

# 6. 清理临时文件
rm -rf /tmp/$BACKUP_DIR

# 7. 启动服务
echo "Starting services..."
docker-compose up -d

# 8. 验证
echo "Verifying..."
sleep 10
curl -f http://localhost:8000/health || echo "Health check failed!"

echo "Restore completed!"
```

### 部分恢复

#### 仅恢复数据库

```bash
# 1. 停止服务
docker-compose stop gateway

# 2. 恢复数据库文件
docker cp ./backup-db-20240114.db ai-gateway:/app/data/ai-gateway.db

# 3. 启动服务
docker-compose start gateway

# 4. 验证
curl http://localhost:8000/api/v1/accounts
```

#### 仅恢复Redis

```bash
# 1. 停止Redis
docker-compose stop redis

# 2. 恢复Redis数据
docker cp ./backup-redis-20240114.rdb ai-gateway-redis:/data/dump.rdb

# 3. 启动Redis
docker-compose start redis

# 4. 验证
docker exec ai-gateway-redis redis-cli ping
```

#### 仅恢复配置

```bash
# 1. 备份当前配置
cp .env .env.backup
cp -r configs configs.backup

# 2. 恢复配置
tar xzf backup-config-20240114.tar.gz

# 3. 重启服务
docker-compose restart gateway

# 4. 验证
docker-compose logs gateway | tail -20
```

---

## 灾难恢复

### 灾难恢复计划 (DRP)

#### 场景1: 数据中心故障

**恢复步骤:**
1. 在备用位置准备新服务器
2. 安装Docker和必要工具
3. 从远程备份下载最新备份
4. 执行完整恢复
5. 更新DNS指向新服务器
6. 验证所有功能

**RTO (恢复时间目标):** 2小时
**RPO (恢复点目标):** 24小时

#### 场景2: 数据损坏

**恢复步骤:**
1. 立即停止服务
2. 评估损坏范围
3. 选择合适的备份点
4. 执行部分或完整恢复
5. 验证数据完整性
6. 恢复服务

**RTO:** 30分钟
**RPO:** 1小时

#### 场景3: 勒索软件攻击

**恢复步骤:**
1. 隔离受感染系统
2. 保留证据（不要删除文件）
3. 从离线备份恢复
4. 更换所有密钥和密码
5. 安全审计
6. 逐步恢复服务

**RTO:** 4小时
**RPO:** 取决于备份策略

### 异地备份策略

```
本地备份 (每小时) → 本地存储
    ↓
本地备份 (每日) → NAS/备份服务器
    ↓
远程备份 (每日) → 云存储 (S3/OSS/COS)
    ↓
归档备份 (每月) → 冷存储 (Glacier/Archive)
```

---

## 备份验证

### 自动验证

```bash
#!/bin/bash
# verify-backup.sh

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file>"
    exit 1
fi

echo "Verifying backup: $BACKUP_FILE"

# 1. 检查文件完整性
echo "Checking file integrity..."
if ! tar tzf "$BACKUP_FILE" > /dev/null 2>&1; then
    echo "ERROR: Backup file is corrupted!"
    exit 1
fi

# 2. 检查必需文件
echo "Checking required files..."
REQUIRED_FILES=("data/ai-gateway.db" "configs/config.json" ".env")
for file in "${REQUIRED_FILES[@]}"; do
    if ! tar tzf "$BACKUP_FILE" | grep -q "$file"; then
        echo "ERROR: Missing required file: $file"
        exit 1
    fi
done

# 3. 测试数据库完整性
echo "Testing database integrity..."
TEMP_DIR=$(mktemp -d)
tar xzf "$BACKUP_FILE" -C "$TEMP_DIR"
if ! sqlite3 "$TEMP_DIR/*/data/ai-gateway.db" "PRAGMA integrity_check;" | grep -q "ok"; then
    echo "ERROR: Database integrity check failed!"
    rm -rf "$TEMP_DIR"
    exit 1
fi
rm -rf "$TEMP_DIR"

echo "✓ Backup verification passed!"
exit 0
```

### 定期验证

```bash
# 每周验证最新备份
0 3 * * 0 /path/to/ai-gateway/scripts/verify-backup.sh $(ls -t backups/*.tar.gz | head -1)
```

---

## 备份监控

### 监控脚本

```bash
#!/bin/bash
# monitor-backups.sh

BACKUP_DIR="./backups"
MAX_AGE_HOURS=26  # 每日备份，允许26小时内的

# 查找最新备份
LATEST=$(find $BACKUP_DIR -name "*.tar.gz" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2)

if [ -z "$LATEST" ]; then
    echo "CRITICAL: No backups found!"
    exit 2
fi

# 检查备份年龄
BACKUP_AGE=$(( ($(date +%s) - $(stat -c %Y "$LATEST")) / 3600 ))

if [ $BACKUP_AGE -gt $MAX_AGE_HOURS ]; then
    echo "WARNING: Latest backup is $BACKUP_AGE_HOURS hours old (threshold: $MAX_AGE_HOURS)"
    exit 1
else
    echo "OK: Latest backup is $BACKUP_AGE_HOURS hours old"
    exit 0
fi
```

---

## 最佳实践

### 备份原则

1. **3-2-1原则**
   - 3份备份副本
   - 2种不同存储介质
   - 1份异地备份

2. **定期测试**
   - 每月测试恢复流程
   - 验证备份完整性
   - 记录恢复时间

3. **安全性**
   - 加密敏感备份
   - 限制访问权限
   - 记录备份操作日志

4. **自动化**
   - 自动化备份流程
   - 自动化验证
   - 自动化告警

### 检查清单

#### 每日检查
- [ ] 确认备份任务完成
- [ ] 检查备份日志无错误
- [ ] 验证备份文件大小正常

#### 每周检查
- [ ] 验证最新备份可用
- [ ] 清理过期备份
- [ ] 检查存储空间

#### 每月检查
- [ ] 执行恢复测试
- [ ] 审查备份策略
- [ ] 更新DRP文档

---

**文档版本**: v1.0.0
**最后更新**: 2024-02-14
**维护者**: DevOps Team
