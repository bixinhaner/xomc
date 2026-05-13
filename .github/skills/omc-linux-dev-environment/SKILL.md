---
name: omc-linux-dev-environment
description: 'Use when setting up, repairing, or explaining the OMC Linux development environment. Covers user-space installation of Go, protoc, protoc-gen-go, protoc-gen-go-grpc, sqlc, swag, golangci-lint, Docker Compose, Node/npm; Docker vs mixed local startup; root-owned log files; and package-lock noise from npm workspaces.'
argument-hint: 'Describe whether you need fresh setup, startup help, or troubleshooting.'
user-invocable: true
---

# OMC Linux Dev Environment

This skill summarizes the Linux development environment that has been validated in this workspace and the operational choices behind it.

## When to Use

- Set up this repository on a Linux host without relying on system-wide package installs.
- Recreate the already-verified toolchain on another machine.
- Explain why local development mode and Docker mode expose different ports.
- Troubleshoot Docker build, migration, logging permission, or npm lockfile issues.
- Decide whether to run in Docker mode, mixed local mode, or Kubernetes mode.

## Validated Baseline

Validated on Linux with user-space tools installed under the current user rather than `/usr/local`.

### Installed Tools

| Tool | Version | Path |
|------|---------|------|
| Go | `go1.25.9` | `~/.opencode/bin/go` |
| protoc | `34.1` | `~/.opencode/bin/protoc` |
| protoc-gen-go | `v1.36.6` | `~/.opencode/bin/protoc-gen-go` |
| protoc-gen-go-grpc | `1.5.1` | `~/.opencode/bin/protoc-gen-go-grpc` |
| sqlc | `v1.30.0` | `~/.opencode/bin/sqlc` |
| swag | `v1.16.4` | `~/.opencode/bin/swag` |
| golangci-lint | `v1.64.8` | `~/.opencode/bin/golangci-lint` |
| Node.js | `v20.19.5` | user environment |
| npm | `10.8.2` | user environment |
| Docker | `29.1.3` | system package |
| Docker Compose plugin | `v5.1.3` | Docker CLI plugin |

### Registry and Proxy Settings

- npm registry: `https://registry.npmmirror.com`
- Go proxy: `https://goproxy.cn,direct`

These settings reduce installation failures on mainland China networks.

## Installation Layout

Use a user-space binary directory and make sure it is on `PATH`.

```bash
export PATH="$HOME/.opencode/bin:$PATH"
```

This repository has been proven to work with binaries installed in that location.

## Recommended Startup Modes

### 1. Mixed Local Development Mode

Use this for day-to-day coding and debugging.

- Dependencies in Docker: PostgreSQL, Redis, NATS, MinIO, Prometheus, Alertmanager, Grafana.
- Host processes: `omcgo-app`, `omcgo-acs`, `omcgo-worker`, and the Vite frontend.

Expected ports in this mode:

- Frontend page: `3000`
- App API: `8081`
- ACS: `8080`
- Worker metrics: `9092`

Key scripts:

- [run/scripts/start-all.sh](./../../run/scripts/start-all.sh)
- [run/scripts/start-backend.sh](./../../run/scripts/start-backend.sh)
- [run/scripts/start-frontend.sh](./../../run/scripts/start-frontend.sh)
- [run/scripts/stop-all.sh](./../../run/scripts/stop-all.sh)

Important behavior:

- `3000` is the Vite development server.
- `8081` is the Go App API directly, so visiting `/` on that port returns API 404 JSON rather than a page.
- `8080` is the ACS entry and should be used for TR-069 device access.

### 2. Docker Compose Mode

Use this when you want a deployment-like single-host environment.

Compose file:

- [deployments/docker/docker-compose.yml](./../../deployments/docker/docker-compose.yml)

Expected behavior:

- The `web` container serves built frontend assets via nginx.
- The `web` container also proxies API and ACS traffic.
- Because of that gateway layer, page access differs from local dev mode.

Useful build invocation:

```bash
DOCKER_BUILDKIT=1 \
COMPOSE_DOCKER_CLI_BUILD=1 \
BUILDKIT_PROGRESS=plain \
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

### 3. Kubernetes Mode

Use this only when a real Kubernetes cluster exists locally or remotely.

Do not confuse this with mixed local mode. K8s mode requires:

- Kubernetes 1.28+
- `kubectl`
- Helm
- Built container images in a registry

Reference:

- [omcgo/docs/operations/deployment-guide.md](./../../omcgo/docs/operations/deployment-guide.md)

## Frontend Build Behavior

The production frontend image is built from [deployments/docker/Dockerfile.web](./../../deployments/docker/Dockerfile.web).

Important choices already encoded there:

- `NODE_ENV=development` is forced in build stages to keep build tooling available.
- `npm ci` is used instead of `npm install` for deterministic dependency resolution.
- The image expects the existing `webcode/package-lock.json` to remain authoritative.

## Known Issues and Fixes

### sqlc Version

Use `sqlc v1.30.0`.

Reason:

- `sqlc v1.31.1` showed compatibility issues with Go 1.25 in this environment.

### Docker Build Requires BuildKit

Symptom:

- Docker build fails on `RUN --mount=type=cache` stages.

Fix:

- Always enable BuildKit when building compose services in this repository.

### Goose Duplicate Migration Version

Symptom:

- `migrate-schema` panics with duplicate migration version `38`.

Verified fix in this workspace:

- The nullable firmware migration now lives at [omcgo/migrations/000048_upgrade_tasks_firmware_id_nullable.sql](./../../omcgo/migrations/000048_upgrade_tasks_firmware_id_nullable.sql).

### Root-Owned Log Files After Docker Runs

Symptom:

- Local host startup fails with `Permission denied` writing under `run/logs/...`.

Cause:

- Docker containers bind-mount host log directories and write as root by default.

Fix:

- Stop application containers.
- Remove or re-own the root-created files under `run/logs/app`, `run/logs/acs`, and `run/logs/worker`.
- Restart host processes.

### npm Workspace Lockfile Noise

Symptom:

- `omcmb/package-lock.json` changes even though no real dependency was intentionally updated.

Cause:

- Running `npm install` in a workspace can rewrite the root lockfile metadata.
- npm version differences can also reserialize optional package metadata.

Fix:

- Prefer existing `node_modules` when possible.
- Prefer `npm ci` for deterministic installs.
- If the lockfile diff is only metadata noise, revert it before committing.

## Verified Access Patterns

In mixed local mode:

- Frontend page: `http://<host>:3000/`
- App API login: `http://<host>:8081/api/v1/auth/login`
- ACS health: `http://<host>:8080/healthz`
- ACS device URL: `http://<host>:8080/smallcell/AcsService`

In Docker Compose mode:

- The `web` container exposes the publish-style page/API gateway.
- The externally visible page port is not the same as Vite dev mode.

## Practical Decision Rules

- Use mixed local mode when writing code or debugging.
- Use Docker Compose mode when validating a single-host deployment flow.
- Use Kubernetes mode only when you actually have a cluster and want deployment-shape verification.

## Minimal Verification Commands

### Toolchain

```bash
go version
protoc --version
sqlc version
swag --version
golangci-lint version | head -1
docker --version
docker compose version
node -v
npm -v
```

### Mixed Local Mode Health Check

```bash
curl -sf http://127.0.0.1:3000/ >/dev/null
curl -sf http://127.0.0.1:8080/healthz >/dev/null
curl -sf http://127.0.0.1:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' >/dev/null
```

## Scope Notes

This skill documents the validated Linux development environment for this repository.

It is not a production runbook and it is not a replacement for the Kubernetes deployment guide.