# AI Gateway - Deployment Guide

## Quick Start (One Command)

### Linux/Mac
```bash
./scripts/start-gateway.sh
```

### Windows
```cmd
scripts\start-gateway.bat
```

---

## Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM minimum
- 10GB disk space

---

## Step-by-Step Deployment

### 1. Clone and Configure

```bash
# Clone the repository
git clone <repository-url>
cd ai-gateway

# Copy environment template
cp .env.example .env

# Edit .env and add your API keys
nano .env
```

### 2. Start Services

**Basic (Gateway + Redis + Web):**
```bash
./scripts/start-gateway.sh
```

**With Monitoring (Prometheus + Grafana):**
```bash
./scripts/start-gateway.sh --monitoring
```

### 3. Verify Deployment

- Gateway API: http://localhost:8000/health
- Web Dashboard: http://localhost:3000
- Prometheus: http://localhost:9090 (with --monitoring)
- Grafana: http://localhost:3001 (with --monitoring)

---

## Service Ports

| Service | Default Port | Environment Variable |
|---------|-------------|---------------------|
| Gateway API | 8000 | GATEWAY_PORT |
| Web Dashboard | 3000 | WEB_PORT |
| Redis | 6379 | REDIS_PORT |
| Prometheus | 9090 | PROMETHEUS_PORT |
| Grafana | 3001 | GRAFANA_PORT |

---

## Management Commands

### Start Services
```bash
./scripts/start-gateway.sh              # Start basic services
./scripts/start-gateway.sh --monitoring # Start with monitoring
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

## Upgrade

### Automatic Upgrade (Recommended)
```bash
./scripts/upgrade.sh
```

### Manual Upgrade
```bash
# 1. Create backup
./scripts/upgrade.sh --backup-only

# 2. Pull updates
git pull

# 3. Restart services
./scripts/start-gateway.sh --restart
```

### Rollback
```bash
./scripts/upgrade.sh --rollback ./backups/backup_YYYYMMDD_HHMMSS
```

---

## Data Persistence

The following data is persisted in Docker volumes:

| Volume | Purpose |
|--------|---------|
| gateway-data | SQLite database |
| redis-data | Redis AOF/RDB |
| prometheus-data | Metrics storage |
| grafana-data | Dashboards & settings |

---

## Troubleshooting

### Port Already in Use
```bash
# Check what's using the port
lsof -i :8000

# Change port in .env
GATEWAY_PORT=8001
```

### Docker Issues
```bash
# Reset Docker state
docker compose down -v
./scripts/start-gateway.sh
```

### View Container Logs
```bash
docker compose logs -f gateway
docker compose logs -f redis
```

---

## Production Checklist

- [ ] Configure API keys in .env
- [ ] Change Grafana admin password
- [ ] Set up SSL/TLS (use reverse proxy)
- [ ] Configure alert notifications
- [ ] Set up backup schedule
- [ ] Review resource limits

---

## Architecture

```
                    ┌─────────────┐
                    │   Browser   │
                    └──────┬──────┘
                           │
                           ▼
┌──────────────────────────────────────────────────┐
│                  Web (Nginx)                     │
│                   Port 3000                       │
└─────────────────────┬────────────────────────────┘
                      │
                      ▼
┌──────────────────────────────────────────────────┐
│              Gateway API (Go)                    │
│                   Port 8000                       │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐          │
│  │ Router  │  │ Limiter │  │  Cache  │          │
│  └────┬────┘  └────┬────┘  └────┬────┘          │
└───────┼────────────┼────────────┼───────────────┘
        │            │            │
        ▼            ▼            ▼
┌─────────────┐ ┌─────────┐ ┌──────────────┐
│   OpenAI    │ │  Redis  │ │   SQLite     │
│  Anthropic  │ │  Cache  │ │   Database   │
│    Azure    │ └─────────┘ └──────────────┘
└─────────────┘

With Monitoring (--monitoring flag):
┌─────────────┐     ┌─────────────┐
│ Prometheus  │────▶│   Grafana   │
│  Port 9090  │     │  Port 3001  │
└─────────────┘     └─────────────┘
```
