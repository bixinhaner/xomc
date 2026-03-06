# OMCGo Production Deployment Guide

## Prerequisites

- Kubernetes 1.28+ cluster
- kubectl configured with cluster access
- Helm 3.x (for infrastructure components)
- Container registry with omcgo images built

### Infrastructure Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| PostgreSQL | 16+ single node | Patroni HA (3 nodes) |
| TimescaleDB | 2.x single node | Dedicated instance |
| Redis | 7+ single node | 6-node cluster |
| NATS | 2.10+ single node | 3-node cluster |
| MinIO | Single node | Distributed (4+ nodes) |

## Build Images

```bash
# Build all three binaries
docker build -t omcgo/acs:latest -f deployments/docker/Dockerfile.acs .
docker build -t omcgo/app:latest -f deployments/docker/Dockerfile.app .
docker build -t omcgo/worker:latest -f deployments/docker/Dockerfile.worker .

# Push to your registry
docker tag omcgo/acs:latest your-registry/omcgo/acs:v1.0.0
docker push your-registry/omcgo/acs:v1.0.0
# ... repeat for app and worker
```

## Deployment Steps

### 1. Create Namespace and Secrets

```bash
kubectl apply -f deployments/k8s/namespace.yaml

# Edit secrets with real values
cp deployments/k8s/secret.yaml /tmp/secret.yaml
# Replace all CHANGE_ME values in /tmp/secret.yaml
kubectl apply -f /tmp/secret.yaml
rm /tmp/secret.yaml
```

### 2. Deploy ConfigMap

```bash
kubectl apply -f deployments/k8s/configmap.yaml
```

### 3. Run Database Migrations

```bash
kubectl run --rm -it migrate \
  --namespace=omcgo \
  --image=omcgo/app:latest \
  --restart=Never \
  --env="DB_DSN=$(kubectl get secret omcgo-secrets -n omcgo -o jsonpath='{.data.DB_DSN}' | base64 -d)" \
  -- /app/migrate -dsn "$DB_DSN" up
```

### 4. Deploy Application Components

```bash
# ACS Engine
kubectl apply -f deployments/k8s/acs/

# App Server
kubectl apply -f deployments/k8s/app/

# Worker
kubectl apply -f deployments/k8s/worker/
```

### 5. Verify Deployment

```bash
kubectl get pods -n omcgo
kubectl get svc -n omcgo
kubectl get hpa -n omcgo

# Check health
kubectl port-forward svc/omcgo-app 8080:8080 -n omcgo
curl http://localhost:8080/healthz
```

## TLS Configuration

### ACS TLS (device-facing)

1. Generate or obtain TLS certificate for the ACS endpoint
2. Create Kubernetes TLS secret:
   ```bash
   kubectl create secret tls acs-tls \
     --cert=acs.crt --key=acs.key -n omcgo
   ```
3. Update ACS deployment to mount the TLS secret and set `tls.enabled: true` in config

### API TLS (Ingress)

1. Install cert-manager for automatic certificate management
2. The Ingress resource references `omcgo-tls` secret
3. Configure cert-manager ClusterIssuer for Let's Encrypt

## Scaling Strategy

### ACS Engine (Horizontal)

- Scales based on `acs_active_sessions` custom metric
- Target: 2000 sessions per instance
- Range: 2-20 replicas
- Each instance handles ~333 Informs/second

### App Server (Horizontal)

- Scales based on CPU utilization (70% target)
- Range: 2-5 replicas

### Worker (Event-driven via KEDA)

- Scales based on NATS JetStream consumer lag
- Range: 2-10 replicas
- Triggers when queue depth > 1000 messages

### Capacity Planning

| Scale | ACS Pods | App Pods | Worker Pods |
|-------|----------|----------|-------------|
| 10K devices | 2 | 2 | 2 |
| 50K devices | 4 | 2 | 3 |
| 100K devices | 8 | 3 | 5 |
| 500K devices | 20+ | 5 | 10 |

## Monitoring

### Prometheus

All pods expose metrics on port 9091 via annotations:
```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "9091"
prometheus.io/path: "/metrics"
```

### Grafana Dashboard

Import `deployments/monitoring/grafana-dashboard.json` into Grafana.

Key panels:
- ACS Active Sessions
- Inform Rate (req/s)
- RPC Latency (p50/p95/p99)
- NATS Queue Depth
- Pod CPU/Memory
- Go Runtime (goroutines, GC)

### Key Alerts

| Alert | Condition | Severity |
|-------|-----------|----------|
| ACS Overloaded | active_sessions > 80% max | Critical |
| High Error Rate | error_rate > 1% for 5min | Warning |
| Pod CrashLoop | restart_count > 3 in 5min | Critical |
| NATS Lag High | consumer_lag > 10000 | Warning |
| DB Connection Pool | pool_usage > 90% | Warning |

## Backup & Recovery

### Database Backup

```bash
# Daily logical backup
pg_dump -h postgres-primary -U omcgo -Fc omcgo > backup_$(date +%Y%m%d).dump

# Restore
pg_restore -h postgres-primary -U omcgo -d omcgo backup_20260306.dump
```

### MinIO Backup

Use `mc mirror` to replicate buckets to a secondary MinIO or S3 bucket.

## Troubleshooting

### ACS Not Receiving Informs

1. Check ACS LoadBalancer external IP: `kubectl get svc omcgo-acs -n omcgo`
2. Verify device can reach the ACS URL
3. Check ACS logs: `kubectl logs -l app=omcgo-acs -n omcgo --tail=100`
4. Verify Redis connectivity for session storage

### High Latency

1. Check database connection pool usage
2. Monitor NATS queue depth for backpressure
3. Review Redis memory usage and eviction rate
4. Scale ACS pods if session count is high

### Worker Not Processing

1. Check NATS consumer status: `nats consumer info omcgo worker`
2. Verify worker logs: `kubectl logs -l app=omcgo-worker -n omcgo`
3. Check for NATS connection issues
