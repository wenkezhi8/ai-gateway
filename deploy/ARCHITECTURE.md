# AI Gateway - Deployment Architecture

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          Internet/Users                          │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │  Reverse Proxy  │ (Optional - Nginx/Traefik)
                    │   SSL/TLS       │
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌──────────────┐   ┌──────────────────┐   ┌──────────────┐
│ Web Dashboard│   │  Gateway API     │   │  Monitoring  │
│   (Nginx)    │   │      (Go)        │   │  (Grafana)   │
│  Port 3000   │   │   Port 8000      │   │  Port 3001   │
└──────────────┘   └────────┬─────────┘   └──────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐   ┌─────────────────┐   ┌──────────────┐
│    Redis     │   │     SQLite      │   │  Prometheus  │
│    Cache     │   │    Database     │   │   Metrics    │
│  Port 6379   │   │  (Docker Vol)   │   │  Port 9090   │
└──────────────┘   └─────────────────┘   └──────────────┘
        │                                         │
        │                                         ▼
        │                                 ┌──────────────┐
        │                                 │Alertmanager  │
        │                                 │  Port 9093   │
        │                                 └──────────────┘
        │
        └─────────────────┬───────────────────────┐
                          │                       │
                          ▼                       ▼
                  ┌──────────────┐       ┌──────────────┐
                  │  OpenAI API  │       │ Anthropic API│
                  └──────────────┘       └──────────────┘
```

---

## Container Architecture

### Docker Network

All services run in a dedicated Docker network (`ai-gateway-network`):

```
Network: ai-gateway-network (172.28.0.0/16)
├── gateway         (172.28.0.10)
├── redis           (172.28.0.11)
├── web             (172.28.0.12)
├── prometheus      (172.28.0.20)
├── grafana         (172.28.0.21)
└── alertmanager    (172.28.0.22)
```

### Volume Mapping

```
Docker Volumes
├── gateway-data     → /app/data (SQLite database)
├── redis-data       → /data (Redis AOF/RDB)
├── prometheus-data  → /prometheus (Metrics storage)
├── grafana-data     → /var/lib/grafana (Dashboards)
└── alertmanager-data→ /alertmanager (Alert state)

Host Mounts
├── ./configs        → /app/configs (Configuration files)
├── ./monitoring     → /etc/prometheus, /etc/grafana (Monitoring config)
└── ./logs           → /app/logs (Log files - optional)
```

---

## Service Dependencies

```
┌─────────────┐
│     web     │
└──────┬──────┘
       │ depends_on
       ▼
┌─────────────┐     depends_on     ┌─────────────┐
│   gateway   │ ─────────────────▶ │    redis    │
└──────┬──────┘                    └─────────────┘
       │
       │ scrapes metrics
       ▼
┌─────────────┐     triggers      ┌──────────────┐
│ prometheus  │ ─────────────────▶│alertmanager  │
└──────┬──────┘                   └──────────────┘
       │
       │ data source
       ▼
┌─────────────┐
│   grafana   │
└─────────────┘
```

---

## Data Flow

### Request Flow

```
1. User Request
   └─▶ Web Dashboard (http://localhost:3000)
        │
        ├─▶ Static Assets (React App)
        │
        └─▶ API Requests → Gateway API (http://localhost:8000)
             │
             ├─▶ Rate Limiting Check
             │    └─▶ Redis Cache
             │
             ├─▶ Route to Provider
             │    ├─▶ OpenAI API
             │    ├─▶ Anthropic API
             │    └─▶ Azure OpenAI API
             │
             ├─▶ Response Caching
             │    └─▶ Redis Cache
             │
             └─▶ Return Response
```

### Monitoring Data Flow

```
1. Metrics Collection
   Gateway API
   └─▶ /metrics endpoint (Prometheus format)
        │
        └─▶ Prometheus scrapes every 15s
             │
             ├─▶ Store in TSDB
             │
             └─▶ Evaluate alert rules
                  │
                  └─▶ Fire alerts → Alertmanager
                       │
                       ├─▶ Email notifications
                       ├─▶ Slack webhook
                       └─▶ PagerDuty (optional)

2. Visualization
   Grafana
   └─▶ Query Prometheus
        │
        └─▶ Render dashboards
             │
             └─▶ User views in browser
```

---

## Resource Allocation

### Development (Default)

| Service | CPU Limit | Memory Limit | CPU Reserve | Memory Reserve |
|---------|-----------|--------------|-------------|----------------|
| Gateway | - | - | - | - |
| Redis | - | 256MB | - | - |
| Web | - | - | - | - |
| Prometheus | - | - | - | - |
| Grafana | - | - | - | - |

### Production

| Service | CPU Limit | Memory Limit | CPU Reserve | Memory Reserve |
|---------|-----------|--------------|-------------|----------------|
| Gateway | 2 cores | 2GB | 0.5 cores | 512MB |
| Redis | 1 core | 1GB | 0.25 cores | 256MB |
| Web | 1 core | 512MB | 0.1 cores | 128MB |
| Prometheus | 1 core | 2GB | 0.25 cores | 512MB |
| Grafana | 1 core | 1GB | 0.1 cores | 256MB |
| Alertmanager | 0.5 cores | 512MB | 0.1 cores | 128MB |

---

## High Availability Setup (Optional)

### Load Balanced Gateway

```
                    ┌─────────────┐
                    │Load Balancer│
                    └──────┬──────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
            ▼              ▼              ▼
      ┌──────────┐  ┌──────────┐  ┌──────────┐
      │ Gateway 1│  │ Gateway 2│  │ Gateway 3│
      └────┬─────┘  └────┬─────┘  └────┬─────┘
           │             │             │
           └─────────────┼─────────────┘
                         │
                         ▼
                  ┌─────────────┐
                  │Redis Cluster│
                  └─────────────┘
```

### Redis Sentinel (For HA)

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Redis Master │ ──▶ │ Redis Slave 1│ ──▶ │ Redis Slave 2│
└──────┬───────┘     └──────────────┘     └──────────────┘
       │
       ├──────────┬──────────┬──────────┐
       ▼          ▼          ▼          ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│Sentinel 1│ │Sentinel 2│ │Sentinel 3│
└──────────┘ └──────────┘ └──────────┘
```

---

## Security Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     External Zone                        │
│  ┌─────────────┐                                        │
│  │   Users     │                                        │
│  └──────┬──────┘                                        │
└─────────┼───────────────────────────────────────────────┘
          │ HTTPS (443)
          ▼
┌─────────────────────────────────────────────────────────┐
│                     DMZ Zone                             │
│  ┌─────────────────┐                                    │
│  │  Reverse Proxy  │ SSL Termination                    │
│  │    (Nginx)      │ Rate Limiting                      │
│  └────────┬────────┘                                    │
└───────────┼─────────────────────────────────────────────┘
            │ HTTP (Internal)
            ▼
┌─────────────────────────────────────────────────────────┐
│                   Internal Zone                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │            Docker Network (172.28.0.0/16)        │  │
│  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐ │  │
│  │  │Gateway │  │ Redis  │  │  Web   │  │Monitor │ │  │
│  │  └────────┘  └────────┘  └────────┘  └────────┘ │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Security Best Practices

1. **Network Isolation**
   - All services in private Docker network
   - Only necessary ports exposed to host
   - Monitoring endpoints not publicly accessible

2. **API Key Security**
   - Stored in .env file (not in code)
   - .env excluded from git
   - Can use Docker secrets in production

3. **Container Security**
   - Gateway runs as non-root user
   - Read-only config mounts
   - Resource limits enforced

4. **Monitoring Access**
   - Grafana requires authentication
   - Prometheus not exposed externally
   - Alertmanager not exposed externally

---

## Scaling Considerations

### Vertical Scaling (Increase Resources)

- Increase CPU/memory limits in docker-compose.prod.yml
- Adjust Redis maxmemory
- Increase Prometheus retention

### Horizontal Scaling (Add Instances)

1. **Gateway Scaling**
   - Deploy multiple gateway instances
   - Use load balancer (Nginx/HAProxy)
   - Ensure sticky sessions if needed

2. **Redis Scaling**
   - Use Redis Sentinel for HA
   - Or Redis Cluster for sharding
   - Consider external Redis service

3. **Database Scaling**
   - Move from SQLite to PostgreSQL
   - Set up read replicas
   - Consider managed database service

---

## Disaster Recovery

### Backup Strategy

```
┌─────────────────┐
│  Production     │
│  Environment    │
└────────┬────────┘
         │
         │ Automated Backup (Daily)
         ▼
┌─────────────────┐
│  Backup Storage │
│  - SQLite DB    │
│  - Redis dump   │
│  - Configs      │
└────────┬────────┘
         │
         │ Replication (Optional)
         ▼
┌─────────────────┐
│  Off-site/Cloud │
│  Storage (S3)   │
└─────────────────┘
```

### Recovery Procedure

1. Stop all services
2. Restore volumes from backup
3. Restart services
4. Verify data integrity
5. Check all services are healthy

---

## Monitoring Architecture

### Prometheus Metrics

```
Gateway exposes:
├── HTTP request metrics
│   ├── Request count by endpoint
│   ├── Request duration histogram
│   └── Response size
├── Provider metrics
│   ├── Requests per provider
│   ├── Error rate by provider
│   └── Token usage
├── Cache metrics
│   ├── Hit rate
│   ├── Miss rate
│   └── Cache size
└── System metrics
    ├── CPU usage
    ├── Memory usage
    └── Goroutine count
```

### Alert Flow

```
Prometheus
│
├─▶ Alert: HighErrorRate
│    └─▶ Alertmanager
│         └─▶ Slack/Email
│
├─▶ Alert: HighLatency
│    └─▶ Alertmanager
│         └─▶ Slack/Email
│
└─▶ Alert: ProviderDown
     └─▶ Alertmanager
          └─▶ PagerDuty (critical)
```

---

## Next Steps

1. Review [Production Checklist](./PRODUCTION-CHECKLIST.md)
2. Configure [Environment Variables](./README.md#configuration)
3. Set up [Monitoring](../monitoring/)
4. Configure [Alerts](../monitoring/alertmanager.yml)
5. Test [Backup & Recovery](../scripts/upgrade.sh)
