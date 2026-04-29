# PRD: 备份压缩集成（T-0074 / R-102 followup）

> **关联**: Backlog T-0074 / Sprint-07 / Domain=F06/backup / Type=feat
> **作者**: Claude（代 Owner=Go）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: **gzip + zstd 实施 + lz4/bzip2 显式 stub**；集成点 = `acs/upload/handler.go`（不是 `backup/executor.go`）；附带修复 file_type "2" → "3" 单字符 bug

---

## 1. 业务背景

T-0071 持久化 MVP 已落 19 字段 × 7 类 schema，其中"压缩"段（`enable_compression` / `compression_level` / `compression_format`）只**存**不**用** — UI 端打了"尚未生效"灰色 Tag，executor 端 zero touch。

R-102 闭环路径：T-0073 已完成 cleanup + alarm Phase 1；T-0075 加密留后；本任务把"压缩"段从持久化推到生效，让用户改的 `compression_format=zstd compression_level=6` 真的能让备份文件压缩存盘。

---

## 2. ULTRATHINK 决策块

### 2.1 关键发现：执行点不在 executor 而在 upload handler

任务字面叫"executor 压缩集成"，但 audit 后发现：

| 节点 | 角色 | 看不看到文件内容？ |
|------|------|-------------------|
| `backup/executor.go` | 监听 `backup.task.created`，逐设备 enqueue Upload device task | ❌ 不看 |
| `acs/rpc/upload.go` | 渲染 SOAP Upload RPC（含 ACS 上传 URL）发给 CPE | ❌ 不看 |
| **`acs/upload/handler.go:132`** | **CPE PUT/POST 文件到 `/smallcell/FileUploadService` → `minioClient.PutObject(... r.Body ...)`** | **✅ 看，且是唯一看得见的点** |
| `transfer/bridge.go:270` | 处理 `AutonomousTransferComplete`（CPE 主动 push），仅 PM/MR/datamodel/log 走这条 | ❌ 看不到 backup（backup 是 ACS-requested，不是 autonomous） |

**结论**：压缩必须 hook 到 upload handler 的 `r.Body → PutObject` 流上。Executor 是 producer，不参与；TransferBridge 是另一条流，也不参与。任务标题"executor 压缩集成"沿用 T-0071 PRD 的命名，但实际工作位于 `acs/upload/`。

### 2.2 关键发现：file_type "2" 路由 bug

`backup/executor.go:136` enqueue 时硬编码 `"file_type": "2"`，进入 `acs/upload/handler.go:170` `normalizeFileType("2")` → `tr069.FileTypePatch`（补丁包）→ `BucketAndCategory(FileTypePatch)` → `firmware bucket / patch/` 子目录。

但 TR-069 标准 + 项目 `pkg/tr069/filetype.go:11` 定义：
- `FileTypeConfig = "3"` // 厂商配置文件 — 双向（**这才是 backup 该用的**）
- `FileTypePatch = "2"`  // 补丁包（vendor 扩展，下行）

即 backup 上传文件**当前实际落到了固件/补丁桶**，不是 `config_backup` 桶。这是 latent bug。

**决策**：T-0074 内附带修复 `executor.go:136` → `"file_type": "3"`（单字符变更）。理由：
- 不修则压缩 hook（按 `ft == FileTypeConfig` 触发）永远不会触发，T-0074 价值为零
- 修复 1 字符，**专注于本 task 路径上的物件**，符合 CLAUDE.md "fix root cause"
- 不是漂移式重构（drive-by refactor），仅修必要的最小集

### 2.3 算法范围：gzip + zstd ✅ / lz4 + bzip2 stub

Schema CHECK 接受 4 个 format 值（gzip/bzip2/lz4/zstd）。如果四个都实现，需要新增 2 个 direct dep：
- `github.com/pierrec/lz4/v4`（lz4，pure Go，成熟）
- `github.com/dsnet/compress/bzip2`（bzip2 写，alpha-quality 标识但 bzip2 子包稳定多年）

权衡：

| 方案 | 优点 | 缺点 |
|------|------|------|
| A：4 个全做 | 用户改 format 即生效；契约 100% 兑现 | +2 direct deps；bzip2 慢且 2026 几乎无人用 |
| **B：gzip + zstd 全做，lz4/bzip2 注册但返回 `ErrCompressionFormatNotImplemented`** | 0 新 deps（zstd 在 klauspost/compress，已是 indirect dep）；覆盖 99% 真实用例 | 用户选 lz4/bzip2 时 PUT 会被 service 拒（policy_service.validatePolicy 已校验白名单，但需新增"已实现 vs 已注册"的细粒度区分）；需 followup |
| C：只做 gzip | 最小工作量 | 失去 zstd（2026 默认压缩选择）；不值 |

**采纳方案 B**：跟随 T-0071 的 "MVP + followup" 模式。

**Followup（T-0074 完成后立**`docs/project/backlog.md`）：
- **T-0077（新登记）**: lz4 + bzip2 算法实现（含 deps + tests），Est=S，依赖 T-0074。

UI 不需变化（仍展示 4 选项），但 service 校验层在 EnableCompression=true + format ∈ {lz4, bzip2} 时返回明确 400 with localized message "lz4/bzip2 暂未支持，请选择 gzip 或 zstd"。

### 2.4 流式 vs 全量

CPE 上传的 `r.Body` 是 `io.Reader`（HTTP body）。MinIO `PutObject` 接受 `io.Reader`。中间插一层 `compressionReader`（包装 r.Body 通过 gzip/zstd writer → pipe → reader）即可流式压缩，**不缓存全量到内存**。

- 实现采用 `io.Pipe()` + 后台 goroutine 写压缩端
- gzip 压缩窗口 ~32 KB，zstd 默认窗口 ~128 KB；内存常数级别
- PutObject 第 4 个参数 `objectSize` 改为 `-1`（流式 unknown size，MinIO multipart 自动分块）

### 2.5 文件名扩展名

压缩后 objectPath 追加扩展名 `.gz` / `.zst`。Restore 流程（T-0072）需识别扩展名解压；T-0074 不实现 restore 侧，只确保扩展名一致并文档化在 PRD §6 的 "解压侧契约" 段，T-0072 实施时按此契约消费。

### 2.6 桶路由

修 file_type=3 后，`BucketAndCategory(FileTypeConfig) → buckets.ConfigBackup, "backup"`。意味着 backup 文件存到 `config_backup` 桶 `backup/{date}/{filename}.gz` 路径。需确认 `appconfig.BucketConfig.ConfigBackup` 已存在 — 已确认（`storage/router.go:28` 引用），无 schema 变更。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我把 `compression_format=zstd compression_level=6 enable_compression=true` 存到 policy 后，下次 backup task 落到 MinIO 的文件应该是 `.zst` 压缩文件，不是裸 XML |
| Go 开发者 | 我希望有一个 `backup.Compressor` 接口可在 upload handler 直接 wrap reader，写少行代码就能接入 |
| QA | 我希望能用 unit test（httptest + 压缩 policy）证明 upload handler 真的产出 `.gz` / `.zst`（魔术字节 + magic header 检查） |
| PM | 我希望前端"压缩"段的"尚未生效" Tag 在 T-0074 后改为绿色"已生效"（仅 gzip/zstd 选项；lz4/bzip2 维持灰）|

---

## 4. 验收标准（GWT）

### V1 — gzip 压缩生效（默认 policy）
- **Given** policy `enable_compression=true compression_format=gzip compression_level=6`
- **When** CPE POST `/smallcell/FileUploadService?fileType=3&filename=cfg.xml` 携带 1 KB 任意 payload
- **Then** MinIO 中对应对象路径以 `.gz` 结尾；下载后用 `gunzip` 可还原原 payload；魔术字节 `1f 8b`

### V2 — zstd 压缩生效（用户切到高压比）
- **Given** policy `enable_compression=true compression_format=zstd compression_level=9`
- **When** 同 V1 CPE 上传
- **Then** MinIO 对象 `.zst` 结尾；用 `zstd -d` 可还原；魔术字节 `28 b5 2f fd`

### V3 — 压缩关闭时透传
- **Given** policy `enable_compression=false`
- **When** 同 V1 上传
- **Then** MinIO 对象路径**不**带 `.gz/.zst` 扩展名；payload 与 r.Body 字节级一致

### V4 — fileType=3 路由到 config_backup 桶
- **Given** executor 已修复（`file_type: "3"`）
- **When** backup task 触发整条链路
- **Then** 文件落到 `${buckets.ConfigBackup}/backup/{YYYY}/{MM}/{DD}/cfg.xml.gz`（不是 firmware bucket）

### V5 — lz4 / bzip2 拒绝 PUT
- **Given** policy.enable_compression=true 且 format ∈ {lz4, bzip2}
- **When** PUT /api/v1/backup/policy
- **Then** 400 Bad Request；error_message 含中英两语 "lz4/bzip2 not supported in current build, use gzip or zstd"
- 注意：format ∈ {lz4, bzip2} 且 enable_compression=false 时**允许通过**（仅持久化，不会触发执行）

### V6 — Prometheus metrics
- **Given** 压缩生效场景（V1 / V2）执行 N 次后
- **When** scrape `/metrics`
- **Then** 见到：
  - `omc_backup_compression_bytes_in_total{format="gzip"}` ≥ N × payload_size
  - `omc_backup_compression_bytes_out_total{format="gzip"}` ≥ 1（非负）
  - `omc_backup_compression_duration_seconds_count{format="gzip"}` = N

### V7 — 大文件流式（不爆内存）
- **Given** 默认 gzip policy
- **When** CPE 上传 100 MB 文件（unit test 用 io.LimitReader 模拟）
- **Then** 压缩 + PutObject 成功；测试进程峰值内存增量 < 10 MB（远小于 100 MB，证明流式）

### V8 — Compressor 接口可独立测试
- **Given** `compression.go` 单测
- **When** 跑 `go test ./internal/backup -run TestCompressor`
- **Then** gzip + zstd 各 5 个 level 完整 round-trip 测试通过；lz4 / bzip2 各返回 `ErrCompressionFormatNotImplemented`

---

## 5. 运营商差异矩阵

| 字段 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 压缩格式默认 | 一致：gzip level=6（与 schema default 一致） | 同 | 同 |
| 是否强制压缩 | 无运营商特殊要求 | 同 | 同 |
| 文件命名规范 | 后缀沿用 OMC 标准 `.gz/.zst` | 同 | 同 |

**本条整体一致：备份压缩属于 OMC 内部存储优化，三家运营商均无规范约束。**

---

## 6. 非目标（明确不做）

| # | 非目标 | 后续承接 |
|---|--------|---------|
| N1 | lz4 + bzip2 算法实施 | T-0077（新登记）|
| N2 | restore 侧解压 / 扩展名识别 | T-0072（已 triaged）按本 PRD §2.5 解压契约消费 |
| N3 | 加密 + 压缩组合（先压再加 vs 先加再压）| T-0075（已 triaged，加密 task 自行决策）|
| N4 | TransferBridge 路径（PM/MR/datamodel）的压缩 | 不在 R-102 范围；PM/MR 文件多已是压缩格式 |
| N5 | compression Prometheus 仪表盘 PNG/JSON | T-0076 cleanup metrics dashboard 一并做 |
| N6 | 旧未压缩备份的批量回填压缩 | 不做；按 forward-only 策略 |
| N7 | upload handler 的 path traversal / size limit / auth 改造 | 已存在，不动 |

---

## 7. 依赖

| 依赖 | 状态 | 说明 |
|------|------|------|
| T-0071 backup_policies schema + service | ✅ done | 19 字段 schema 已落，含 compression_* 三字段 |
| `pkg/tr069.FileTypeConfig = "3"` | ✅ 已存在 | filetype.go:11 |
| `appconfig.BucketConfig.ConfigBackup` | ✅ 已存在 | storage/router.go:28 引用 |
| `klauspost/compress/zstd` | ✅ go.mod indirect | 本任务**提升为 direct**（写到 require 块）|
| stdlib `compress/gzip` | ✅ stdlib | 0 dep |
| `acs/upload/handler.go` | ✅ 已存在 | 加注 PolicyGetter / Compressor adapter |

无新外部依赖（gzip + zstd 全部可用）。

---

## 8. 度量

| 度量 | 类型 | 含义 |
|------|------|------|
| `omc_backup_compression_bytes_in_total{format}` | counter | 累计原始字节（compress 输入端） |
| `omc_backup_compression_bytes_out_total{format}` | counter | 累计压缩后字节（输出端） |
| `omc_backup_compression_duration_seconds{format}` | histogram | 单次压缩耗时分布 |
| `omc_backup_compression_errors_total{format,reason}` | counter | 压缩错误（reason=open\|copy\|close） |

派生（可在 dashboard 端做，metric 不必导出）：
- 平均压缩比 = `bytes_out / bytes_in`
- p99 压缩耗时（histogram quantile）

---

## 9. 设计备忘（S2）

### 9.1 接口契约

```go
// internal/backup/compression.go
package backup

import (
    "errors"
    "io"
)

var (
    ErrCompressionFormatNotImplemented = errors.New("compression format not implemented in current build")
    ErrCompressionLevelOutOfRange      = errors.New("compression level out of range [1,9]")
    ErrCompressionFormatInvalid        = errors.New("compression format must be one of gzip|bzip2|lz4|zstd")
)

// Compressor 包装一个 io.Reader，返回压缩流的 io.ReadCloser。
// Reader 模式选择（不是 Writer）：upload handler 拿到的是 r.Body（io.ReadCloser），
// MinIO PutObject 期待 io.Reader；中间插一个流式压缩 Reader 最自然。
type Compressor interface {
    // Wrap 读取 src 的明文，返回压缩后的 ReadCloser。
    // 关闭返回的 ReadCloser 会同时关闭内部的压缩 writer 与底层 src。
    Wrap(src io.Reader) (io.ReadCloser, error)
    Format() string    // "gzip" | "zstd" | "lz4" | "bzip2"
    Extension() string // ".gz"  | ".zst" | ".lz4" | ".bz2"
}

// NewCompressor 工厂。format/level 校验失败返回对应 sentinel error。
// 当 format ∈ {lz4, bzip2} 时返回 ErrCompressionFormatNotImplemented（B 方案）。
func NewCompressor(format string, level int) (Compressor, error)
```

### 9.2 流式实现要点

```go
// gzipCompressor.Wrap 用 io.Pipe + goroutine：
//
//   pr, pw := io.Pipe()
//   gzw, _ := gzip.NewWriterLevel(pw, level)
//   go func() {
//       _, copyErr := io.Copy(gzw, src)
//       closeErr := gzw.Close()
//       pw.CloseWithError(firstNonNil(copyErr, closeErr))
//   }()
//   return pr, nil
```

zstd 同模式（`zstd.NewWriter(pw, zstd.WithEncoderLevel(...))`）。

### 9.3 Level 映射

| OMC level | gzip (1..9) | zstd (SpeedFastest..SpeedBestCompression) |
|-----------|-------------|-------------------------------------------|
| 1         | 1 BestSpeed | SpeedFastest |
| 2         | 2           | SpeedFastest |
| 3         | 3           | SpeedDefault |
| 4         | 4           | SpeedDefault |
| 5         | 5           | SpeedDefault |
| 6 默认    | 6 Default   | SpeedDefault |
| 7         | 7           | SpeedBetterCompression |
| 8         | 8           | SpeedBetterCompression |
| 9         | 9 Best      | SpeedBestCompression |

### 9.4 Upload handler DI 改动

`acs/upload/handler.go` 加两个可选字段：

```go
type Handler struct {
    // ... existing fields
    policyGetter   backup.PolicyGetter   // optional, nil-safe = compression off
    metricsHook    *backup.PolicyMetrics // optional, nil-safe
}

// NewHandler 签名维持向后兼容；新增 setter（与 BackupExecutor.SetPolicyEnforcement 同模式）：
func (h *Handler) SetCompression(getter backup.PolicyGetter, metrics *backup.PolicyMetrics)
```

router 端 DI（`cmd/app/router/router.go`）：

```go
uploadHandler := upload.NewHandler(...)
uploadHandler.SetCompression(backupModule.PolicyService, backupModule.PolicyMetrics)
```

### 9.5 ServeHTTP 压缩判定流程

```
判 ft == FileTypeConfig
  | 否 → 透传走老路径
  | 是 → 取 policy（policyGetter.Get(ctx)）
        | err 或 nil → 透传 + warn log（best-effort，不阻断备份）
        | 拿到 → 判 EnableCompression
              | false → 透传
              | true  → format ∈ {lz4, bzip2} → 透传 + warn "未实现，跳过压缩"（service 层应已拦在 PUT，但二次保险）
                       format ∈ {gzip, zstd} → NewCompressor → Wrap → objectPath += ext → PutObject
                          失败回退：err → log + 透传（best-effort）
```

**最关键设计**：压缩失败 fall back 到原始流，不让备份本身失败。理由：备份是高可用功能，宁可存大文件，不能让压缩问题导致备份失败。

### 9.6 PolicyMetrics 扩展

`policy_metrics.go` 已在 T-0073 创建。本任务**新增** 4 个 collector：

```go
CompressionBytesIn  *prometheus.CounterVec   // labels: format
CompressionBytesOut *prometheus.CounterVec   // labels: format
CompressionDuration *prometheus.HistogramVec // labels: format; buckets default
CompressionErrors   *prometheus.CounterVec   // labels: format, reason
```

`reason` 取值：`open`（NewCompressor 失败）/ `copy`（io.Copy 中断）/ `close`（Close 失败）。

### 9.7 文件清单

新增：
- `omcgo/internal/backup/compression.go`（接口 + factory + gzip/zstd 实现 + lz4/bzip2 stub）≈ 200 行
- `omcgo/internal/backup/compression_test.go` ≈ 250 行（gzip/zstd round-trip × 5 levels + 大文件流式 + lz4/bzip2 stub error + level 边界 + invalid format）

修改：
- `omcgo/internal/backup/policy_metrics.go`：+ 4 collectors（≈ 30 行）
- `omcgo/internal/backup/policy_service.go`：validatePolicy 增加 "lz4/bzip2 + EnableCompression=true 时 reject" 校验（≈ 10 行）
- `omcgo/internal/backup/policy_service_test.go`：+ 2 case
- `omcgo/internal/backup/executor.go`：1 字符 `"file_type": "2"` → `"3"`
- `omcgo/internal/backup/executor_test.go`（如有）：相应断言更新
- `omcgo/internal/acs/upload/handler.go`：+ SetCompression + ServeHTTP 压缩分支（≈ 60 行）
- `omcgo/internal/acs/upload/handler_test.go`（如不存在则建）：+ 4 case（gzip 生效 / zstd 生效 / 压缩关 / 文件类型非 Config 透传）
- `omcgo/cmd/app/router/router.go`：DI 接线（≈ 3 行）
- `omcgo/scripts/e2e_verify.sh`：+ 1 claim 验证 PUT policy 含 lz4/bzip2 时 400（V5）

### 9.8 测试矩阵

| Layer | Test | 目标 GWT |
|-------|------|---------|
| compression.go | TestNewCompressor_validFormats | V8 |
| compression.go | TestNewCompressor_invalidFormat | V8 |
| compression.go | TestNewCompressor_levelOutOfRange | V8 |
| compression.go | TestGzipRoundTrip(level=1..9) | V1, V8 |
| compression.go | TestZstdRoundTrip(level=1..9) | V2, V8 |
| compression.go | TestLZ4StubError / TestBzip2StubError | V5, V8 |
| compression.go | TestStreamingMemory(100MB) | V7 |
| policy_service | TestValidatePolicy_lz4WithEnableCompressionRejected | V5 |
| policy_service | TestValidatePolicy_bzip2WithEnableCompressionRejected | V5 |
| upload/handler | TestServeHTTP_gzipCompressionApplied | V1, V4 |
| upload/handler | TestServeHTTP_zstdCompressionApplied | V2, V4 |
| upload/handler | TestServeHTTP_compressionDisabledPassThrough | V3 |
| upload/handler | TestServeHTTP_nonConfigFileTypePassThrough | V4（反向） |
| metrics | TestCompressionMetricsRegistered（grep）| V6 |

### 9.9 Carrier 差异适配

无差异点（PRD §5 已说明）。本任务不触 `internal/carrier/`。

### 9.10 待定点

| 待定 | 决策 |
|------|------|
| MinIO Content-Encoding HTTP header 是否要写？ | 写。`PutObjectOptions.ContentEncoding = "gzip"` 或 `"zstd"`，MinIO 客户端自动 round-trip；下游恢复方读 metadata 即可知格式（与扩展名互为冗余备份） |
| level 映射的 gzip 选用 stdlib 还是 klauspost 加速版？ | 用 stdlib `compress/gzip`：稳定、无新 dep；klauspost gzip 性能优势在批量场景，单流不显著 |
| zstd encoder 是否复用？ | 是。 `zstd.NewWriter` 创建 encoder 的开销不可忽略；每个 Compressor 实例持有一个 encoder pool（`sync.Pool`），但 MVP 阶段直接每次 new 也可（GC 压力低）— **MVP 简单优先，每次 new** |

---

## 10. 解压契约（给 T-0072 restore 用）

T-0074 落到 MinIO 的 backup 对象命名规则：

```
{config_backup_bucket}/backup/{YYYY/MM/DD}/{originalFileName}{ext}
```

其中 `{ext}`:
- `enable_compression=false` → 空（如 `cfg.xml`）
- `enable_compression=true` 且 format=gzip → `.gz`
- `enable_compression=true` 且 format=zstd → `.zst`

T-0072 restore 流程按 `objectPath` 末尾扩展名走分支解压；不存在历史压缩文件 mismatch 风险（T-0074 是 forward-only，旧文件不带扩展名）。
