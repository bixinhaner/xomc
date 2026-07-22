# PM/KPI Recovery and Redeploy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复已成功入库并压缩改名的 PM 文件因事件重投而反复读取旧 MinIO 路径的问题，清理失效积压，重新发布 OMC 并形成升级后的 KPI 压测结论。

**Architecture:** collector 在设备解析后、读取 MinIO 前，通过最小 `FileMarkerLookup` 查询 `(device_sn, path.Base(minio_path))` 是否已有 `parsed=true` marker；命中时将事件作为 duplicate 成功 ACK，未命中时保持既有解析和 CopyIngest 流程。运行侧使用现有发布包、Compose 和资源规划脚本升级，清理范围严格限制为 NATS `PM` stream 与 PM 模块 DLQ。

**Tech Stack:** Go 1.x、pgx v5、MinIO、NATS JetStream、PostgreSQL/TimescaleDB、Docker Compose、Bash。

## Global Constraints

- 不删除 `pm_files`、`pm_metrics`、设备数据或已成功归档的 MinIO 对象。
- 不修改 KPI 白名单；`whitelist_miss` 按预期保留。
- 不使用 `run/scripts/restart-all.sh`；容器服务只通过交付包和 Docker Compose 管理。
- `OMC_PUBLIC_HOST` 固定为 `172.24.224.78`。
- AIDE 保留日常任务，但排除 Docker 和 OMC 动态数据目录。
- 目标机只有单块旋转逻辑盘；报告必须明确存储瓶颈，不能宣称资源限额能消除该瓶颈。

---

### Task 1: Add parsed-marker lookup and duplicate short circuit

**Files:**
- Create: `omcgo/internal/pm/collector/duplicate_test.go`
- Modify: `omcgo/internal/pm/collector/collector.go`
- Modify: `omcgo/internal/pm/pg_file_store.go`
- Modify: `omcgo/cmd/worker/main.go`

**Interfaces:**
- Produces: `FileMarkerLookup.IsFileParsed(context.Context, string, string) (bool, error)`.
- Produces: `PMCollector.SetFileMarkerLookup(FileMarkerLookup)`.
- Consumes: `PgPMFileStore`, already constructed in worker startup and backed by the TimescaleDB pool containing `pm_files`.

- [ ] **Step 1: Write the failing collector tests**

Create a fake lookup that records arguments and returns either `true`, `false`, or an error. Build a fat `FileReceivedPayload` with a valid device UUID and `MinIOPath: "pm/SN001/report.xml"`, while leaving `minioClient` nil. Assert:

```go
func TestHandleFileReceived_ParsedMarkerSkipsMinIO(t *testing.T) {
    lookup := &fakeFileMarkerLookup{parsed: true}
    c := &PMCollector{logger: zap.NewNop(), fileMarkerLookup: lookup}
    err := c.handleFileReceived(context.Background(), fileReceivedEvent(t))
    require.NoError(t, err)
    require.Equal(t, "SN001", lookup.deviceSN)
    require.Equal(t, "report.xml", lookup.fileName)
}
```

Add a second test with `lookup.err = errors.New("timescaledb unavailable")` and assert the returned error contains `lookup parsed pm file marker`. Because both cases return before MinIO, a missing short circuit would panic on the nil client and fail the test.

- [ ] **Step 2: Run the tests and confirm RED**

Run:

```bash
cd omcgo
go test ./internal/pm/collector -run 'TestHandleFileReceived_(ParsedMarkerSkipsMinIO|MarkerLookupError)' -count=1
```

Expected: compile failure because `FileMarkerLookup` and `fileMarkerLookup` do not exist.

- [ ] **Step 3: Implement the minimal lookup and collector behavior**

Add:

```go
type FileMarkerLookup interface {
    IsFileParsed(ctx context.Context, deviceSN, fileName string) (bool, error)
}

func (c *PMCollector) SetFileMarkerLookup(lookup FileMarkerLookup) {
    c.fileMarkerLookup = lookup
}
```

Immediately after `resolveDevice`, call the lookup with `path.Base(payload.MinIOPath)`. On lookup error return `fmt.Errorf("lookup parsed pm file marker: %w", err)`. When parsed is true, increment `FilesProcessedTotal{status="duplicate"}`, observe processing duration, add a `pm.duplicate=true` span attribute, log the skip, and return nil before UUID parsing or MinIO access.

Implement the store method with one indexed lookup:

```go
func (s *PgPMFileStore) IsFileParsed(ctx context.Context, deviceSN, fileName string) (bool, error) {
    var parsed bool
    err := s.pool.QueryRow(ctx,
        `SELECT parsed FROM pm_files WHERE device_sn = $1 AND file_name = $2`,
        deviceSN, fileName,
    ).Scan(&parsed)
    if err == pgx.ErrNoRows {
        return false, nil
    }
    if err != nil {
        return false, fmt.Errorf("query parsed pm_file marker: %w", err)
    }
    return parsed, nil
}
```

Wire the existing `pmFileStore` into the collector:

```go
pmCollector.SetFileMarkerLookup(pmFileStore)
```

- [ ] **Step 4: Run focused and package tests**

Run:

```bash
cd omcgo
gofmt -w internal/pm/collector/duplicate_test.go internal/pm/collector/collector.go internal/pm/pg_file_store.go cmd/worker/main.go
go test ./internal/pm/collector ./internal/pm ./cmd/worker -count=1
```

Expected: all packages report `ok`.

- [ ] **Step 5: Commit**

```bash
git add omcgo/internal/pm/collector/duplicate_test.go omcgo/internal/pm/collector/collector.go omcgo/internal/pm/pg_file_store.go omcgo/cmd/worker/main.go
git commit -m "fix: 跳过已入库的PM重投事件"
```

### Task 2: Verify the source tree and build a release

**Files:**
- Generated, not committed: `deployments/release/archive/project/<version>/omc-release-<version>-amd64.tar.xz`

**Interfaces:**
- Consumes: the code committed in Task 1.
- Produces: one amd64 release archive plus its SHA-256 digest.

- [ ] **Step 1: Run backend verification**

```bash
cd omcgo
go build ./...
go test ./...
```

Expected: both commands exit 0.

- [ ] **Step 2: Run deployment-script checks**

```bash
bash -n deployments/release/build-release.sh
bash -n deployments/release/bundle/deploy/install.sh
bash -n deployments/release/bundle/deploy/plan-resources.sh
```

Expected: all commands exit 0 without output.

- [ ] **Step 3: Build the release package**

```bash
cd deployments/release
./build-release.sh --channel release --arch amd64
```

Expected: a new uniquely tagged `omc-release-*-amd64.tar.xz` and checksum are created under `archive/project/`; app, ACS, worker and web images are rebuilt from this checkout.

### Task 3: Prepare and clean the target host

**Files:**
- Create on host: `/etc/aide/aide.conf.d/99_z_omc_runtime_excludes`
- Create on host: `/etc/logrotate.d/omcgo`
- Modify on host: `/opt/omc/current/deploy/.env`

**Interfaces:**
- Consumes: SSH root access to `172.24.224.78`.
- Produces: quiet AIDE scope, bounded OMC logs, correct public host, empty historical PM stream and PM-only DLQ.

- [ ] **Step 1: Record the pre-change state**

Collect container state, PM stream/consumer counters, PM DLQ count, `pm_files` and `pm_metrics` counts, current AIDE PID, disk statistics, and SHA-256 of the current deployment `.env` without printing its contents.

- [ ] **Step 2: Stop PM writers**

From `/opt/omc/current/deploy`, stop only app, ACS and worker through the Compose wrapper. Verify NATS, PostgreSQL, TimescaleDB and MinIO remain healthy for scoped cleanup.

- [ ] **Step 3: Purge only the PM queue and PM DLQ**

Build `kpiperf` for linux/amd64 locally, copy it to the host, then run:

```bash
/tmp/kpiperf -mode purge -nats nats://127.0.0.1:4222
```

Delete only:

```sql
DELETE FROM dead_letters WHERE source_module = 'pm';
```

Verify PM stream message count and PM DLQ count are zero while `pm_files` and `pm_metrics` counts are unchanged.

- [ ] **Step 4: Install host safeguards**

Install AIDE recursive exclusions for `/home/docker-data` and `/opt/omc/{run,current,releases,packages}`. Stop only the currently running AIDE scan, if present, while leaving its daily schedule enabled. Install logrotate with `daily`, `maxsize 100M`, `rotate 7`, `compress`, `delaycompress`, `missingok`, `notifempty`, and `copytruncate`; validate with `aide --config-check` and `logrotate -d`.

- [ ] **Step 5: Set the public host**

Replace or append exactly:

```text
OMC_PUBLIC_HOST=172.24.224.78
```

in the current deployment `.env`, preserve mode/ownership, and verify only the key’s value.

### Task 4: Copy, plan resources, install, and validate

**Files:**
- Copy to host: `/opt/omc/packages/<release archive>`
- Generate in extracted release: `deploy/resources.env`

**Interfaces:**
- Consumes: release archive from Task 2 and host preparation from Task 3.
- Produces: running Compose services on the new image tag.

- [ ] **Step 1: Transfer and verify**

Copy the archive and checksum to `/opt/omc/packages`, compare local and remote SHA-256, and extract into its own release directory.

- [ ] **Step 2: Re-plan resources**

With application writers stopped, run:

```bash
bash deploy/plan-resources.sh --assume-dedicated
```

Review generated CPU/memory values for 32 CPUs and 32 GiB RAM. Do not invent a separate storage device.

- [ ] **Step 3: Install and restart**

Run:

```bash
bash deploy/install.sh
```

Expected: images load, migrations complete, `current` switches to the new release, Compose recreates services, and built-in health checks pass.

- [ ] **Step 4: Verify deployment**

Run the packaged healthcheck and Compose status. Confirm every required container is healthy, app/ACS/worker use the new image tag, worker sees `OMC_PUBLIC_HOST=172.24.224.78`, old PM pending does not reappear, and no new PM DLQ growth occurs during the initial observation.

### Task 5: Measure the post-change KPI load

**Files:**
- No source changes.

**Interfaces:**
- Consumes: the existing CPE workload against the upgraded deployment.
- Produces: before/after performance findings and the limiting subsystem.

- [ ] **Step 1: Establish a stable observation window**

Wait until services have warmed up and collect at least five minutes of access logs and metrics. If CPE traffic is temporarily absent, extend observation while continuing health and queue checks.

- [ ] **Step 2: Calculate endpoint and pipeline performance**

Report FileUpload requests/second, HTTP success rate, average/P50/P95/P99/max latency, ACS session latency, worker success/failure rates and P95, NATS pending/ack-pending/redelivery, database waits, MinIO activity, container CPU/memory/block I/O, host load/iowait, and disk utilization/queue/latency.

- [ ] **Step 3: Validate data correctness**

Sample recent uploaded files and confirm matching successful rows exist in both `pm_files` and `pm_metrics`. Confirm duplicate events are ACKed without increasing failures or PM DLQ.

- [ ] **Step 4: State the result**

Compare with the pre-change baseline: approximately 59.2 FileUpload/s, 99.985% HTTP success, 10.24s average, 8.7s P50, 22.3s P95, 31.4s P99, 43.1s max, historical PM pending around 2.27M, and disk saturation around 87% with elevated iowait. Separate upload capacity, ingestion capacity, and the single-rotational-disk hardware ceiling.
