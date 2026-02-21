# AI Gateway - Deployment Package Summary

## Quick Start

### Windows Users
```cmd
deploy\quick-start.bat
```

### Mac/Linux Users
```bash
./deploy/quick-start.sh
```

That's it! The script will:
1. Check Docker installation
2. Create configuration files
3. Pull Docker images
4. Start all services
5. Open the dashboard in your browser

---

## Deployment Options

### 1. Quick Start (Recommended for Beginners)
**Best for**: First-time users, development, testing

```bash
# Windows
deploy\quick-start.bat

# Mac/Linux
./deploy/quick-start.sh
```

### 2. Standard Deployment
**Best for**: Regular development, testing new features

```bash
# Basic services (Gateway + Redis + Web)
./scripts/start-gateway.sh

# With monitoring (Prometheus + Grafana)
./scripts/start-gateway.sh --monitoring
```

### 3. Production Deployment
**Best for**: Production environments

```bash
# Use production compose file
docker-compose -f deploy/docker-compose.prod.yml up -d
```

See [PRODUCTION-CHECKLIST.md](./PRODUCTION-CHECKLIST.md) for production setup guide.

---

## Access Points

After successful deployment:

| Service | URL | Purpose |
|---------|-----|---------|
| Web Dashboard | http://localhost:3000 | Main interface |
| Gateway API | http://localhost:8000 | API endpoint |
| API Docs | http://localhost:8000/docs | API documentation |
| Health Check | http://localhost:8000/health | Service status |
| Prometheus | http://localhost:9090 | Metrics (with --monitoring) |
| Grafana | http://localhost:3001 | Dashboards (with --monitoring) |

---

## Configuration

### Required: API Keys

Edit `.env` file in project root:

```env
OPENAI_API_KEY=sk-your-openai-key
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
```

Get your keys from:
- OpenAI: https://platform.openai.com/api-keys
- Anthropic: https://console.anthropic.com/settings/keys

### Optional: Custom Ports

```env
GATEWAY_PORT=8000
WEB_PORT=3000
REDIS_PORT=6379
```

---

## Management Commands

### Start Services
```bash
./scripts/start-gateway.sh              # Basic
./scripts/start-gateway.sh --monitoring # With monitoring
```

### Stop Services
```bash
./scripts/start-gateway.sh --stop
```

### Restart Services
```bash
./scripts/start-gateway.sh --restart
```

### View Logs
```bash
./scripts/start-gateway.sh --logs
```

---

## File Structure

```
deploy/
├── quick-start.sh              # Mac/Linux quick start
├── quick-start.bat             # Windows quick start
├── docker-compose.prod.yml     # Production configuration
├── README.md                   # This file
├── PRODUCTION-CHECKLIST.md     # Production setup guide
├── ARCHITECTURE.md             # Architecture documentation
├── docker/
│   ├── Dockerfile             # Container image
│   └── docker-compose.yml     # Docker Compose config
└── nginx/
    └── nginx.conf             # Reverse proxy config
```

---

## System Requirements

### Minimum
- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM
- 10GB disk space

### Recommended (Production)
- Docker 24.0+
- Docker Compose 2.20+
- 8GB+ RAM
- 50GB+ SSD
- Multi-core CPU

---

## What Gets Installed

### Core Services (Always)
- **Gateway** - Go-based API gateway
- **Redis** - Caching layer
- **Web** - React dashboard

### Monitoring Stack (Optional)
- **Prometheus** - Metrics collection
- **Grafana** - Visualization dashboards
- **Alertmanager** - Alert management

---

## Data Persistence

All data is stored in Docker volumes:

```bash
# List volumes
docker volume ls | grep ai-gateway

# Inspect volume
docker volume inspect ai-gateway_gateway-data
```

| Volume | Content |
|--------|---------|
| gateway-data | SQLite database, logs |
| redis-data | Cache data |
| prometheus-data | Metrics (15-30 days) |
| grafana-data | Dashboards, settings |
| alertmanager-data | Alert state |

---

## Backup & Restore

### Create Backup
```bash
./scripts/upgrade.sh --backup-only
```

### Restore from Backup
```bash
./scripts/upgrade.sh --rollback ./backups/backup_YYYYMMDD_HHMMSS
```

---

## Troubleshooting

### Docker Issues

```bash
# Check Docker status
docker info

# View container logs
docker-compose logs -f gateway

# Restart a service
docker-compose restart gateway

# Reset everything
docker-compose down -v
./scripts/start-gateway.sh
```

### Port Conflicts

```bash
# Check what's using a port
lsof -i :8000

# Change port in .env
GATEWAY_PORT=8001
```

### API Key Issues

```bash
# Check if keys are set
grep API_KEY .env

# Edit configuration
nano .env
```

---

## Documentation

- [README.md](../README.md) - Project overview
- [DEPLOYMENT.md](../DEPLOYMENT.md) - Detailed deployment guide
- [PRODUCTION-CHECKLIST.md](./PRODUCTION-CHECKLIST.md) - Production setup
- [ARCHITECTURE.md](./ARCHITECTURE.md) - Architecture details

---

## Support

- Check logs: `./scripts/start-gateway.sh --logs`
- Review docs: `docs/` folder
- GitHub Issues: Report bugs and feature requests

---

## License

See [LICENSE](../LICENSE) file in the root directory.
