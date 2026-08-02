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

### PM Redis migration runbook

Run from `/opt/omc/current/deploy`. The migration service is in the optional
`operations` profile and defaults to `--dry-run`, so a normal install never
starts it.

1. Wait until the preceding hourly window is published, stop both App and
   Worker with `bash svc.sh stop app worker`, and confirm no PM Redis writer or
   consumer advances. App and Worker form one Redis routing compatibility unit:
   never run versions/configurations that point their PM clients at different
   endpoints.
2. Start and check the target with `bash svc.sh start redis-pm` and
   `docker compose -p omcgo --env-file .env --env-file resources.env -f docker-compose.infra.yml exec -T redis-pm redis-cli ping`.
3. Run the safe preview:

   ```bash
   docker compose -p omcgo --env-file .env --env-file resources.env \
     -f docker-compose.infra.yml -f docker-compose.app.yml \
     --profile operations run --rm pm-redis-migrate
   ```

4. If the preview reports no unexplained conflict, run the copy by overriding
   the default command. The command copies only `pmagg:*`; `kpi-route:*` is
   intentionally rebuilt on cache miss:

   ```bash
   docker compose -p omcgo --env-file .env --env-file resources.env \
     -f docker-compose.infra.yml -f docker-compose.app.yml \
     --profile operations run --rm pm-redis-migrate \
     --pattern 'pmagg:*' --scan-count 500 --pipeline-size 100 --output json
   ```

   A differing destination key fails closed. Use `--replace` only after
   confirming Worker remains stopped and the destination value is stale.
5. Require `failed=0`, `conflicts=0`, equal source/target key counts, and every
   key verified by DUMP payload plus TTL tolerance. Start App and Worker from
   the same release/config only after verification. Keep the old core keys
   until their TTL expires.
6. If PM health fails, stop both App and Worker. Because PM may contain events
   newer than the retained core copy, reverse-copy authoritative `pmagg:*`
   state before changing endpoints (dry-run first, then explicit replace):

   ```bash
   docker compose -p omcgo --env-file .env --env-file resources.env \
     -f docker-compose.infra.yml -f docker-compose.app.yml \
     --profile operations run --rm pm-redis-migrate \
     --source redis-pm:6379 --target redis-core:6379 \
     --pattern 'pmagg:*' --scan-count 500 --pipeline-size 100 --dry-run --output json

   docker compose -p omcgo --env-file .env --env-file resources.env \
     -f docker-compose.infra.yml -f docker-compose.app.yml \
     --profile operations run --rm pm-redis-migrate \
     --source redis-pm:6379 --target redis-core:6379 \
     --pattern 'pmagg:*' --scan-count 500 --pipeline-size 100 --replace --output json
   ```

   Require full DUMP/TTL verification and zero failures, then start the prior
   dual-reader App and Worker together against core Redis. Do not point both
   explicit production endpoints at the same address: the configuration guard
   rejects that unsafe topology.

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
