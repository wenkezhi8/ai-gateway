# AI Gateway 开发规范

## 修改代码后的验证流程

### 1. 快速验证（推荐每次修改后运行）

```bash
./scripts/verify_all.sh
```

### 2. 完整验证流程

```bash
# 1. 编译后端
go build -o bin/gateway ./cmd/gateway

# 2. 构建前端（如果修改了前端）
cd web && npm run build && cd ..

# 3. 重启服务
pkill -f "bin/gateway"
./bin/gateway &

# 4. 运行验证脚本
./scripts/verify_all.sh

# 5. 运行单元测试
go test ./... -v
```

### 3. 修改代码前的检查清单

- [ ] 理解要修改的代码及其依赖关系
- [ ] 检查是否有其他地方调用了要修改的函数
- [ ] 考虑修改是否会影响其他功能

### 4. 修改代码后的检查清单

- [ ] 运行 `./scripts/verify_all.sh`
- [ ] 检查所有测试通过
- [ ] 手动测试受影响的功能

## 常见问题及解决方案

### 问题1: 修改后API返回404

检查路由配置文件: `internal/router/router.go`

### 问题2: 前端页面空白

1. 检查前端是否重新构建: `cd web && npm run build`
2. 检查静态文件服务配置

### 问题3: Dashboard数据不显示

1. 检查对应API返回格式: `curl localhost:8566/api/admin/dashboard/stats`
2. 确认返回的 `success: true`
3. 确认 `data` 字段存在

## 开发命令速查

```bash
# 启动服务
./bin/gateway

# 验证所有功能
./scripts/verify_all.sh

# 运行测试
go test ./... -v

# 构建前端
cd web && npm run build

# 检查服务状态
curl localhost:8566/health
```
