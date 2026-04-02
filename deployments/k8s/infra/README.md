# Infrastructure Production Configuration

This directory documents the recommended production infrastructure setup for omcgo.
Each component should be deployed as a managed service or HA cluster.

## PostgreSQL (Primary DB)

- **Recommended**: Patroni HA cluster (3 nodes) or cloud-managed (RDS, Cloud SQL)
- **Version**: PostgreSQL 16+
- **Config**: `max_connections=500`, `shared_buffers=4GB`, `effective_cache_size=12GB`
- **Storage**: SSD, IOPS >= 3000
- **Replication**: Synchronous streaming replication with automatic failover
- **Backup**: pg_basebackup daily + WAL archiving to S3/MinIO

## TimescaleDB (Time-series)

- **Recommended**: TimescaleDB extension on a dedicated PostgreSQL instance
- **Version**: TimescaleDB 2.x on PostgreSQL 16+
- **Config**: `timescaledb.max_background_workers=8`
- **Retention**: Configure data retention policies per hypertable
  - PM counters: 90 days raw, 2 years aggregated
  - Alarm history: 1 year
- **Compression**: Enable native compression on chunks older than 7 days

## Redis Cluster

- **Recommended**: 6-node Redis Cluster (3 master + 3 replica) or cloud-managed
- **Version**: Redis 7+
- **Config**: `maxmemory=8gb`, `maxmemory-policy=allkeys-lru`
- **Persistence**: AOF with `appendfsync everysec`
- **Purpose**: Session state, command queues, data model cache, rate limiting

## NATS JetStream

- **Recommended**: 3-node NATS cluster with JetStream enabled
- **Version**: NATS 2.10+
- **Config**: `jetstream { max_memory_store: 4GB, max_file_store: 50GB }`
- **Streams**: Single `omcgo` stream with subjects `omcgo.>`
- **Consumers**: Durable pull consumers per worker group

## MinIO / S3

- **Recommended**: MinIO distributed mode (4+ nodes) or AWS S3
- **Config**: Erasure coding EC:4 for data durability
- **Buckets**: pm-files, mr-files, firmware, config-backup, logs
- **Lifecycle**: Transition to cold storage after 30 days, delete after 365 days
- **Versioning**: Enable on firmware bucket for rollback support

## Monitoring Stack

- **Prometheus**: Scrape all omcgo pods via annotations, 15s interval
- **Grafana**: Import dashboard from `deployments/monitoring/grafana-dashboard.json`
- **Alertmanager**: Configure alerts for:
  - ACS active sessions > 80% capacity
  - Error rate > 1%
  - Pod restart count > 3 in 5 minutes
  - NATS consumer lag > 10000
