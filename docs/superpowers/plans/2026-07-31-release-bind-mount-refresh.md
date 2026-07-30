# Release Bind Mount Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure every release switch refreshes containers that bind-mount configuration through `/opt/omc/current`, so the running configuration always matches the deployed release.

**Architecture:** Keep the existing immutable release directory and `current` symlink. After the normal Compose reconciliation, explicitly force-recreate only the six stateless monitoring services that mount release-local configuration; keep named data volumes and all database, Redis, NATS, and MinIO containers untouched.

**Tech Stack:** Bash, Docker Compose, shell regression tests.

## Global Constraints

- Preserve all named and host data volumes.
- Do not force-recreate PostgreSQL, TimescaleDB, Redis, NATS, or MinIO.
- Continue to support `--skip-monitoring`.
- The post-switch health check remains the deployment success gate.

---

### Task 1: Lock the release-switch behavior with a regression test

**Files:**
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`

**Interfaces:**
- Consumes: `INSTALL`, the packaged `install.sh` path already defined by the test.
- Produces: a static assertion that the installer force-recreates the exact release-bound monitoring services with `--no-deps`.

- [x] **Step 1: Write the failing test**

```bash
contains "升级强制刷新版本目录 bind mount" \
  '"${DC[@]}" up -d --force-recreate --no-deps prometheus alertmanager grafana loki otelcol tempo' \
  "$INSTALL"
```

- [x] **Step 2: Run test to verify it fails**

Run: `bash deployments/release/bundle/deploy/storage-compose_test.sh`

Expected: FAIL at `升级强制刷新版本目录 bind mount`.

### Task 2: Refresh only release-bound stateless services

**Files:**
- Modify: `deployments/release/bundle/deploy/install.sh`
- Test: `deployments/release/bundle/deploy/storage-compose_test.sh`

**Interfaces:**
- Consumes: `DC` Compose command array and `SKIP_MONITORING`.
- Produces: a second, scoped Compose reconciliation after the normal full-stack `up -d`.

- [x] **Step 1: Add the minimal implementation**

```bash
if [ "$SKIP_MONITORING" = 0 ]; then
  log "刷新版本目录 bind mount（仅监控无状态容器，保留数据卷）..."
  "${DC[@]}" up -d --force-recreate --no-deps \
    prometheus alertmanager grafana loki otelcol tempo
fi
```

- [x] **Step 2: Run the regression test**

Run: `bash deployments/release/bundle/deploy/storage-compose_test.sh`

Expected: PASS.

- [x] **Step 3: Run all release shell tests**

Run: `for test in deployments/release/**/*_test.sh; do bash "$test"; done`

Expected: all tests PASS.

### Task 3: Verify, submit, deploy, and prove the running mount

**Files:**
- No source changes expected.

**Interfaces:**
- Consumes: the committed release package and target host `172.24.224.197`.
- Produces: an MR, merged release, deployed containers whose creation timestamps and mounted alert rules match the new release.

- [x] **Step 1: Run repository validation**

Run: `cd omcgo && go build ./... && go test ./...`

Run: `cd omcmb && npm run typecheck`

Expected: all commands pass.

- [ ] **Step 2: Commit and create the MR**

```bash
git add deployments/release/bundle/deploy/install.sh \
  deployments/release/bundle/deploy/storage-compose_test.sh \
  docs/superpowers/plans/2026-07-31-release-bind-mount-refresh.md
git commit -m "fix: 刷新发布版本配置挂载"
git push -u origin codex/release-bind-mount-refresh-root-fix
glab mr create --fill
```

- [ ] **Step 3: Merge and deploy**

Build the merged `main` commit into an immutable release package, verify package checksums, install it on `172.24.224.197`, and require the packaged health check to pass.

- [ ] **Step 4: Verify runtime identity**

Confirm the six monitoring containers were recreated after deployment, `/etc/prometheus/alerts/omc-rules.yml` no longer contains `PMKnownIndicatorsDisabled`, and all persistent volumes remain attached.

- [ ] **Step 5: Continue operational validation**

Observe business success/error rates, ACS and worker queues, KPI hour/day rollups, PostgreSQL/TimescaleDB activity, CPU, memory, disk throughput, and I/O wait until backlogs drain and no new actionable anomaly remains.
