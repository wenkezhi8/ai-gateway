# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 1.x.x   | ✅        |
| < 1.0   | ❌        |

## Reporting a Vulnerability

**请勿在公开的 GitHub Issue 中报告安全漏洞。**

如果您发现安全漏洞，请通过以下方式报告：

1. **Email**: security@your-domain.com
2. **GitHub Security Advisory**: 使用 GitHub 的 [Private Security Advisory](https://github.com/wenkezhi8/ai-gateway/security/advisories) 功能

我们承诺：
- 在 48 小时内确认收到报告
- 在 7 天内评估并提供初步反馈
- 及时修复确认的漏洞
- 在修复后公开致谢（经您同意）

## Security Best Practices

### 部署安全

1. **JWT Secret**
   ```bash
   # 生产环境必须设置强密码
   export JWT_SECRET="your-strong-random-secret-at-least-32-characters"
   ```

2. **API Keys**
   ```bash
   # 使用环境变量，不要硬编码
   export OPENAI_API_KEY="sk-..."
   export ANTHROPIC_API_KEY="sk-ant-..."
   ```

3. **Redis 密码**
   ```bash
   export REDIS_PASSWORD="your-redis-password"
   ```

4. **HTTPS**
   - 生产环境必须使用 HTTPS
   - 建议使用反向代理（Nginx, Caddy）

### 配置安全

1. **文件权限**
   ```bash
   # 配置文件权限
   chmod 600 configs/config.json
   chmod 600 data/api_keys.json
   ```

2. **网络安全**
   - 限制管理端口访问
   - 使用防火墙规则
   - 启用 rate limiting

3. **日志安全**
   - API keys 在日志中自动脱敏
   - 审计日志记录所有管理操作

### Docker 安全

```yaml
# docker-compose.yml 安全配置
services:
  gateway:
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
```

## Security Features

### 已实现

- ✅ Request body size limit (10MB)
- ✅ Rate limiting
- ✅ JWT authentication
- ✅ API key masking in logs
- ✅ CORS protection
- ✅ Audit logging
- ✅ Input validation

### 计划中

- 🔲 TLS mutual authentication
- 🔲 IP whitelist/blacklist
- 🔲 Request signing
- 🔲 Secrets encryption at rest

## Security Checklist

部署前检查：

- [ ] JWT_SECRET 已设置为强密码
- [ ] API keys 通过环境变量配置
- [ ] 启用 HTTPS
- [ ] 配置文件权限正确
- [ ] Redis 密码已设置
- [ ] Rate limiting 已启用
- [ ] 管理端口已限制访问
- [ ] 审计日志已启用

## Third-party Dependencies

我们定期扫描依赖漏洞：

```bash
# Go 依赖扫描
make security-scan

# 前端依赖扫描
cd web && npm audit
```

## Security Updates

安全更新将通过：
- GitHub Security Advisories
- Release notes
- Email 通知（订阅用户）
