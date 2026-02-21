# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Enterprise optimization: golangci-lint configuration
- Enterprise optimization: ESLint 9 + Prettier configuration for frontend
- Enterprise optimization: .editorconfig for unified editor settings
- Enterprise optimization: pre-commit hooks configuration
- Enterprise optimization: Enhanced Makefile with CI commands

## [1.0.0] - 2024-01-01

### Added
- Multi-provider support: OpenAI, Anthropic, Zhipu, DeepSeek, Qwen, etc.
- Intelligent rate limiting with per-user and global quotas
- Response caching to reduce API costs
- Flexible routing strategies (cost-based, round-robin, failover)
- OpenAI-compatible RESTful API
- Web dashboard for monitoring and configuration
- Docker and Docker Compose support
- Prometheus + Grafana monitoring stack
- JWT authentication
- Audit logging
- Swagger API documentation

### Security
- Request body size limit (10MB) to prevent DoS attacks
- API key masking in logs
- CORS middleware

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2024-01-01 | Initial release |

---

## How to Update

```bash
# Pull latest changes
git pull origin main

# Update dependencies
make deps

# Rebuild
make build

# Restart service
./bin/ai-gateway
```
