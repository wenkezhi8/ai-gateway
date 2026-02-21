# AI Gateway - Quick Deployment Guide

## One-Click Deployment

### Windows Users

Double-click `quick-start.bat` or run in Command Prompt:

```cmd
deploy\quick-start.bat
```

### Mac/Linux Users

Run in Terminal:

```bash
./deploy/quick-start.sh
```

---

## What Happens When You Run the Script?

1. **Checks Docker** - Ensures Docker is installed and running
2. **Creates .env file** - Automatically creates configuration file if missing
3. **Validates API Keys** - Checks if API keys are configured
4. **Pulls Images** - Downloads necessary Docker images
5. **Starts Services** - Launches all containers automatically

---

## After Startup

### Access Points

| Service | URL | Description |
|---------|-----|-------------|
| Web Dashboard | http://localhost:3000 | Main user interface |
| Gateway API | http://localhost:8000 | API endpoint |
| Health Check | http://localhost:8000/health | Service status |

### First-Time Setup

1. Open http://localhost:3000 in your browser
2. Go to **Settings** and configure your API keys:
   - OpenAI API Key
   - Anthropic API Key
   - Azure OpenAI (optional)
3. Start making API requests!

---

## Management Commands

### Basic Operations

```bash
# Stop services
./scripts/start-gateway.sh --stop

# Restart services
./scripts/start-gateway.sh --restart

# View logs
./scripts/start-gateway.sh --logs
```

### With Monitoring Stack

```bash
# Start with Prometheus + Grafana
./scripts/start-gateway.sh --monitoring

# Access monitoring
# Prometheus: http://localhost:9090
# Grafana:    http://localhost:3001
```

---

## Configuration

### Environment Variables

Edit `.env` file in project root:

```env
# Server Ports (customize if needed)
GATEWAY_PORT=8000
WEB_PORT=3000
REDIS_PORT=6379

# API Keys (required)
OPENAI_API_KEY=sk-your-key-here
ANTHROPIC_API_KEY=sk-ant-your-key-here

# Optional
AZURE_OPENAI_API_KEY=
AZURE_OPENAI_ENDPOINT=
```

### Getting API Keys

- **OpenAI**: https://platform.openai.com/api-keys
- **Anthropic**: https://console.anthropic.com/settings/keys
- **Azure OpenAI**: https://portal.azure.com

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
# Reset everything
docker compose down -v
./scripts/start-gateway.sh
```

### View Container Logs

```bash
docker compose logs -f gateway
docker compose logs -f redis
```

---

## System Requirements

- **Docker**: 20.10 or higher
- **Docker Compose**: 2.0 or higher
- **RAM**: 4GB minimum
- **Disk**: 10GB free space
- **OS**: Windows 10+, macOS 10.15+, or Linux

---

## Data Persistence

All data is stored in Docker volumes:

- `gateway-data` - SQLite database
- `redis-data` - Cache data
- `prometheus-data` - Metrics (with monitoring)
- `grafana-data` - Dashboards (with monitoring)

### Backup Data

```bash
# Create backup
./scripts/upgrade.sh --backup-only

# Restore from backup
./scripts/upgrade.sh --rollback ./backups/backup_YYYYMMDD_HHMMSS
```

---

## Next Steps

- Read the [Full Documentation](../README.md)
- Check the [Deployment Guide](../DEPLOYMENT.md)
- Configure [Monitoring](../monitoring/)
- Set up [Alerts](../monitoring/alertmanager.yml)

---

## Need Help?

- **Documentation**: Check the `docs/` folder
- **Issues**: Report bugs on GitHub Issues
- **Logs**: Check logs with `./scripts/start-gateway.sh --logs`
