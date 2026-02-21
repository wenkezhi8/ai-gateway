# AI Gateway - Production Deployment Checklist

## Pre-Deployment Checklist

### 1. Environment Configuration

- [ ] Copy `.env.example` to `.env`
- [ ] Configure all required API keys:
  - [ ] `OPENAI_API_KEY`
  - [ ] `ANTHROPIC_API_KEY`
  - [ ] `AZURE_OPENAI_API_KEY` (if using Azure)
  - [ ] `AZURE_OPENAI_ENDPOINT` (if using Azure)
- [ ] Change default ports if needed:
  - [ ] `GATEWAY_PORT` (default: 8000)
  - [ ] `WEB_PORT` (default: 3000)
  - [ ] `REDIS_PORT` (default: 6379)
- [ ] Configure monitoring ports:
  - [ ] `PROMETHEUS_PORT` (default: 9090)
  - [ ] `GRAFANA_PORT` (default: 3001)
  - [ ] `ALERTMANAGER_PORT` (default: 9093)

### 2. Security Settings

- [ ] Change Grafana admin password:
  ```bash
  # In .env file
  GRAFANA_ADMIN_PASSWORD=your-secure-password-here
  ```
- [ ] Configure alert notifications (edit `monitoring/alertmanager.yml`):
  - [ ] Email notifications
  - [ ] Slack webhook (optional)
  - [ ] PagerDuty (optional)
- [ ] Review Redis memory limit:
  ```bash
  REDIS_MAXMEMORY=512mb
  ```
- [ ] Set up SSL/TLS:
  - [ ] Configure reverse proxy (Nginx/Traefik)
  - [ ] Install SSL certificates
  - [ ] Enable HTTPS redirect

### 3. Resource Planning

- [ ] Minimum system requirements met:
  - [ ] CPU: 2+ cores
  - [ ] RAM: 8GB minimum (16GB recommended)
  - [ ] Disk: 50GB+ SSD
- [ ] Adjust resource limits in `docker-compose.prod.yml`:
  ```yaml
  deploy:
    resources:
      limits:
        cpus: '2'
        memory: 2G
  ```

### 4. Backup Strategy

- [ ] Set up backup script:
  ```bash
  ./scripts/upgrade.sh --backup-only
  ```
- [ ] Configure automated backups (cron job):
  ```bash
  # Daily backup at 2 AM
  0 2 * * * /path/to/ai-gateway/scripts/upgrade.sh --backup-only
  ```
- [ ] Test backup restoration:
  ```bash
  ./scripts/upgrade.sh --rollback ./backups/backup_YYYYMMDD_HHMMSS
  ```

### 5. Monitoring & Alerts

- [ ] Review alert rules in `monitoring/alert_rules.yml`
- [ ] Configure alert thresholds:
  - [ ] Error rate threshold
  - [ ] Response time threshold
  - [ ] Resource usage threshold
- [ ] Test alert notifications:
  ```bash
  # Trigger test alert
  curl -X POST http://localhost:9093/api/v1/alerts -d '[{"labels":{"alertname":"TestAlert"}}]'
  ```
- [ ] Import Grafana dashboards:
  - [ ] Gateway metrics dashboard
  - [ ] System resources dashboard
  - [ ] Redis monitoring dashboard

### 6. Network Configuration

- [ ] Configure firewall rules:
  - [ ] Allow required ports only
  - [ ] Restrict monitoring endpoints to internal network
- [ ] Set up reverse proxy (if needed):
  ```nginx
  # Example Nginx config
  server {
      listen 80;
      server_name gateway.yourdomain.com;

      location / {
          proxy_pass http://localhost:8000;
          proxy_set_header Host $host;
          proxy_set_header X-Real-IP $remote_addr;
      }
  }
  ```
- [ ] Configure CORS if needed (in gateway config)

### 7. Logging

- [ ] Configure log rotation (already set in docker-compose.prod.yml)
- [ ] Set up centralized logging (optional):
  - [ ] ELK Stack
  - [ ] Loki
  - [ ] Cloud logging service
- [ ] Define log retention policy

### 8. High Availability (Optional)

- [ ] Set up load balancer:
  - [ ] Multiple gateway instances
  - [ ] Health checks configured
- [ ] Configure Redis replication:
  - [ ] Redis Sentinel
  - [ ] Redis Cluster
- [ ] Database replication (if using external DB)

---

## Deployment Steps

### 1. Prepare Environment

```bash
# Clone repository
git clone <repository-url>
cd ai-gateway

# Create .env from template
cp .env.example .env

# Edit configuration
nano .env
```

### 2. Build Images

```bash
# Build Docker images
docker compose -f deploy/docker-compose.prod.yml build

# Or pull from registry
docker compose -f deploy/docker-compose.prod.yml pull
```

### 3. Start Services

```bash
# Start all services
docker compose -f deploy/docker-compose.prod.yml up -d

# Check status
docker compose -f deploy/docker-compose.prod.yml ps

# View logs
docker compose -f deploy/docker-compose.prod.yml logs -f
```

### 4. Verify Deployment

```bash
# Check gateway health
curl http://localhost:8000/health

# Check web interface
curl http://localhost:3000

# Check Prometheus
curl http://localhost:9090/-/healthy

# Check Grafana
curl http://localhost:3001/api/health
```

### 5. Post-Deployment

- [ ] Import Grafana dashboards
- [ ] Configure alert channels
- [ ] Test API endpoints
- [ ] Verify monitoring data
- [ ] Document deployment details

---

## Post-Deployment Checklist

- [ ] All services are running and healthy
- [ ] API endpoints are accessible
- [ ] Monitoring dashboards show data
- [ ] Alerts are configured and working
- [ ] Backups are scheduled
- [ ] SSL/TLS is working (if configured)
- [ ] Firewall rules are in place
- [ ] Documentation is updated
- [ ] Team is notified of deployment

---

## Maintenance Commands

### View Logs

```bash
# All services
docker compose -f deploy/docker-compose.prod.yml logs -f

# Specific service
docker compose -f deploy/docker-compose.prod.yml logs -f gateway
```

### Restart Services

```bash
# Restart all
docker compose -f deploy/docker-compose.prod.yml restart

# Restart specific service
docker compose -f deploy/docker-compose.prod.yml restart gateway
```

### Update Services

```bash
# Pull latest images
docker compose -f deploy/docker-compose.prod.yml pull

# Recreate containers
docker compose -f deploy/docker-compose.prod.yml up -d
```

### Backup & Restore

```bash
# Create backup
./scripts/upgrade.sh --backup-only

# Restore from backup
./scripts/upgrade.sh --rollback ./backups/backup_YYYYMMDD_HHMMSS
```

---

## Emergency Procedures

### Service Down

1. Check logs: `docker compose logs -f <service>`
2. Restart service: `docker compose restart <service>`
3. If restart fails, check resource usage
4. Consider rolling back to previous version

### High Error Rate

1. Check Prometheus alerts
2. Review gateway logs
3. Check API provider status
4. Review rate limiting settings

### Database Issues

1. Check Redis logs
2. Verify Redis connectivity
3. Check memory usage
4. Consider clearing cache if needed

---

## Contacts & Resources

- **Documentation**: `/docs` folder
- **Runbook**: Link to operations runbook
- **On-call**: Contact information
- **Status Page**: Link to status page
