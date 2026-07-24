# Release deployment observability

The default production installation starts the complete monitoring compose
profile. The app, ACS, and worker production configs enable OTLP tracing and
send spans to the bundled `otelcol:4317` collector.

`install.sh --skip-monitoring` also forces `OMCGO_TRACER_ENABLED=false`. This
keeps the existing no-op tracer behavior when no collector is deployed; do not
enable tracing unless an OTLP collector endpoint is configured and reachable.

The OpenTelemetry Collector image is distroless, so health validation must not
run a shell, `curl`, or `wget` inside that container. Its `health_check`
extension listens on port 13133, the release compose publishes that port only
on `127.0.0.1`, and `healthcheck.sh` probes it externally from the host.

## Configuration backup bucket compatibility

The physical S3/MinIO bucket is `config-backup`. The legacy `config_backup`
spelling is accepted only as a logical API/config input and is normalized
before any MinIO call. No physical legacy-bucket migration exists or is needed:
an underscore bucket cannot have been created through an S3-compatible API.
Startup continues to create/use the valid `config-backup` bucket from the
service production configuration.
