# Release deployment observability

The default production installation starts the complete monitoring compose
profile. The app, ACS, and worker production configs enable OTLP tracing and
send spans to the bundled `otelcol:4317` collector.

`install.sh --skip-monitoring` also forces `OMCGO_TRACER_ENABLED=false`. This
keeps the existing no-op tracer behavior when no collector is deployed; do not
enable tracing unless an OTLP collector endpoint is configured and reachable.
The selected profile is persisted as `OMCGO_SKIP_MONITORING=0|1` in the
release `deploy/.env`. Later standalone `svc.sh` and `healthcheck.sh` runs read
that state, so a monitoring-free install does not accidentally require
monitoring containers or OTEL health while a normal production install still
checks the complete monitoring stack.

The OpenTelemetry Collector image is distroless, so health validation must not
run a shell, `curl`, or `wget` inside that container. Its `health_check`
extension listens on port 13133, the release compose publishes that port only
on `127.0.0.1`, and `healthcheck.sh` probes it externally from the host.

## Resource-plan contract

`resources.env` is a required, complete deployment contract rather than a
best-effort override file. Generate it with `plan-resources.sh`; it records
schema version `3` and the probed host CPU and memory. The planner writes and
validates a temporary file in the destination directory, then atomically
replaces `resources.env`; generation or validation failure preserves the
previous last-good file.

`install.sh --check-only` validates the candidate that a real install would
use: the new package's `deploy/resources.env`, otherwise the current release,
otherwise `etc/resources.env.saved`. A normal install validates that candidate
before switching `current` or restarting anything, then validates the copied
file again. `install.sh` and `svc.sh` reject missing, partial, malformed, or
internally inconsistent files, including legacy files that only contain Redis
values. Re-run `plan-resources.sh`; deleting the file no longer falls back to
Compose defaults.

After deployment, `healthcheck.sh` validates a present contract against the
rendered Compose limits, Docker `NanoCpus`/`Memory`, Go GOMAXPROCS metrics,
Redis runtime settings, and PostgreSQL/TimescaleDB runtime settings. Any
mismatch names the service with its expected and actual values.

## Physical Redis isolation

Production runs `redis-core` and `redis-pm` as separate containers, persistence
paths, memory budgets, and AOF rewrite domains. `redis-core` keeps host port
6379 and the compatibility DNS alias `redis`; `redis-pm` is reachable only on
the Compose network. App and Worker use both endpoints, while ACS and other
core services use only `redis-core`.

Before upgrading, re-run `plan-resources.sh`; schema-v2 single-Redis plans are
rejected intentionally. Core Redis reserves at least 1 GiB and PM Redis at
least 2 GiB above `maxmemory` for AOF copy-on-write. During migration and the
rollback acceptance window, retain legacy `pmagg:*` keys in core Redis until
their TTL expires; do not delete them merely because PM traffic has switched.

## Redis aggregation v2 rollout gate

`PM_AGGREGATION_REDIS_V2_WRITE_ENABLED` defaults to `true`. This release reads
both v1 and v2 state and migrates an old window when its next event arrives.
Rollback is supported only to a dual-reader release; set the gate to `false`
before resuming traffic. Never roll back to a pre-dual-reader Worker because it
cannot finalize v2 state.

## Configuration backup bucket compatibility

The physical S3/MinIO bucket is `config-backup`. The legacy `config_backup`
spelling is accepted only as a logical API/config input and is normalized
before any MinIO call. No physical legacy-bucket migration exists or is needed:
an underscore bucket cannot have been created through an S3-compatible API.
Startup continues to create/use the valid `config-backup` bucket from the
service production configuration.

`backup-reencrypt` also defaults to `config-backup`. Its `--bucket` flag and
`OMC_MINIO_BUCKET` environment variable continue accepting `config_backup` as
legacy input, but List/Get/Put operations always receive the physical name.
