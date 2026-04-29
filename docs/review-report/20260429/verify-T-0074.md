# Verify Report — T-0074 backup 压缩集成 MVP

> **Task**: T-0074 / R-102 followup / Sprint-07
> **Branch**: main / pre-commit @ 60ec38ce
> **Date**: 2026-04-29
> **PRD**: `docs/project/prd/T-0074-backup-executor-compression.md`

---

## S4 出口门检查

| 门 | 结果 | 详情 |
|----|------|------|
| `go build ./...` | ✅ | 全项目编译通过（CGO_ENABLED=0） |
| `go test ./internal/backup/... ./internal/acs/upload/...` | ✅ | 0 fail（含 -race -count=1） |
| `go vet ./...` （touched packages）| ✅ | clean |
| `go mod tidy` | ✅ | klauspost/compress 由 indirect 提升 direct；零新增 direct dep |
| 全项目 `go test ./...` | ⚠️ | 2 个 task package 用例失败（`source_id NULL` scan bug），与 stash 前一致，**与 T-0074 无关**（pre-existing） |
| 新端点 vs E2E claim 比 | ✅ | 新路由数 R=0（仅扩展现有 PUT /backup/policy 校验）；新增 1 claim（W2D bk-8）；E/R = N/A（既有 endpoint 行为变化即覆盖） |
| Migration 双向演练 | ✅ N/A | 本任务无 schema 变更 |
| Metric 名可 grep | ✅ | 4/4 命中 — `omc_backup_compression_{bytes_in,bytes_out,duration_seconds,errors}_total` |
| 累计型依赖核销 | ✅ N/A | 无累计依赖 |

---

## 文件改动清单

新增：
- `omcgo/internal/backup/compression.go`（172 行）— Compressor 接口 + factory + gzip/zstd 完整实现 + lz4/bzip2 stub
- `omcgo/internal/backup/compression_test.go`（~190 行）— round-trip × 9 levels × 2 algos + level 边界 + invalid format + stub error + streaming-memory 50MB
- `omcgo/internal/acs/upload/handler_test.go`（~140 行）— maybeWrapForCompression × 8 case + countingReader

修改：
- `omcgo/internal/backup/policy_metrics.go`（+58 行）— 4 个 collectors 注册 + 3 个 Record 方法
- `omcgo/internal/backup/policy_service.go`（+10 行）— validatePolicy 增加 lz4/bzip2 + EnableCompression=true 拒绝
- `omcgo/internal/backup/policy_service_test.go`（+12 行）— 3 个 case
- `omcgo/internal/backup/executor.go`（+3 行 / -1）— file_type "2" → "3" + 注释解释 bug fix
- `omcgo/internal/backup/executor_test.go`（+1 / -1）— 断言 "2" → "3"
- `omcgo/internal/acs/upload/handler.go`（+105 行）— Handler 字段 + SetCompression + maybeWrapForCompression + countingReader + ServeHTTP 集成
- `omcgo/cmd/acs/main.go`（+10 行）— DI: PolicyRepository + PolicyService + PolicyMetrics → SetCompression
- `omcgo/scripts/e2e_verify.sh`（+8 行）— W2D bk-8 claim
- `omcgo/go.mod`（klauspost/compress indirect→direct）

---

## GWT 验收对照

| GWT | 验证手段 | 状态 |
|-----|---------|-----|
| V1 gzip 压缩生效（魔术字节 + round-trip） | TestGzipRoundTrip_allLevels（9 levels）| ✅ |
| V2 zstd 压缩生效（魔术字节 + round-trip） | TestZstdRoundTrip_allLevels（9 levels）| ✅ |
| V3 压缩关闭时透传 | TestMaybeWrapForCompression_disabled | ✅ |
| V4 fileType=3 路由到 config_backup | executor.go:138 fix + TestHandleTask_PushUploadCommand 断言 "3" | ✅ |
| V5 lz4 / bzip2 拒绝 PUT | TestValidatePolicy `lz4 + enable_compression=true rejected` + e2e bk-8 | ✅ |
| V6 Prometheus metrics | grep 4/4 命中 + RecordCompressionBytes/Duration/Error 调用点 | ✅ |
| V7 大文件流式（50MB heap < 20MB） | TestStreamingMemory | ✅ |
| V8 Compressor 接口可独立测试 | compression_test.go 全套 | ✅ |

---

## ULTRATHINK 决策回顾

| 决策 | 兑现 |
|------|------|
| 集成点 = `acs/upload/handler.go`（不是 executor）| ✅ |
| gzip + zstd 实施，lz4/bzip2 stub（B 方案）| ✅ 0 新 direct dep |
| 修复 file_type "2"→"3" bug 一并提交 | ✅ executor.go:138 + 注释 |
| 流式 io.Pipe + goroutine（不缓存全量）| ✅ TestStreamingMemory 50MB heap < 20MB |
| 压缩失败 fall-back 到原始流 | ✅ maybeWrapForCompression 所有 err 路径返回 applied=false |
| Content-Encoding HTTP header 不写（防 S3 客户端自动解压）| ✅ 仅 PutObjectOptions{ContentType: "application/octet-stream"} |

---

## Followup 待登记（S7 阶段做）

- **T-0077（新）**: lz4 + bzip2 算法实施（含 deps `pierrec/lz4/v4` + `dsnet/compress/bzip2`）— Est=S，依赖 T-0074 ✅

---

## 备注

- ACS 进程现需 `inf.PgPool` 才能启用压缩（已于 cmd/acs/main.go:69 强制要求 PgPool 非空，与 task service 共用同一约束，无新增配置门槛）
- `app` 进程的 backup PolicyMetrics（modules.go:189）独立于 ACS 注册，二者各自暴露相同指标名；Prometheus scrape 端按 `instance` label 区分
- 双进程 metric 名一致（`omc_backup_compression_*`）符合"功能视图统一"原则；ACS 端是真实生产路径（CPE 上传到这里）

---

## S5 Code Review 复核（review-agent verdict: APPROVE-WITH-FIXES → APPROVE）

| 严重度 | 问题 | 修复 | 状态 |
|--------|------|------|------|
| C1 | pumpAndClose goroutine 在 PutObject 早退时可能阻塞在 src.Read | `Wrap(ctx, src)` 签名 + `ctxReader` 包装 src，每次 Read 前检查 ctx.Err() | ✅ Fixed in compression.go:33-51 + 130-144 |
| H1 | `countingReader.n` 跨 goroutine 读写（pump 写 / handler 读） | 改 `atomic.Int64` + `Add` / `Load` | ✅ Fixed in handler.go:339,355 |
| H2 | 错误消息英文单语；frontend dropdown 仍展示 lz4/bzip2 | service 错误改中英双语；**frontend 下拉禁用 defer T-0077** | ✅ 后端 fixed (policy_service.go:107-109)；FE 进 T-0077 |
| H3 | PutObject 失败时误记 `RecordCompressionError("copy")`，污染压缩错误指标 | 删除该 metric 调用，注释解释为何不记 | ✅ Fixed in handler.go:166-172 |
| M3 | `PutObjectOptions.ContentEncoding` 未设置（PRD §9.10 承诺） | `uploadOpts.ContentEncoding = cmp.format` | ✅ Fixed in handler.go:153 |
| M4 | zstd encoder 未 pool | 注释明确指向 T-0077 | ⏸ defer to T-0077 |
| M5 | 测试 zeroReader 死代码 | 删除（已被 countingZeroReader 替代） | ✅ Fixed in compression_test.go |
| M6 | ServeHTTP-level 集成测试缺失 | 需要 minio mock；记入测试债务（PRD §9.10）；e2e bk-8 部分覆盖 PUT 校验 | ⏸ test debt |
| M7 | r.Body close ownership 文档 | handler.go:138-142 加注释明确 net/http 拥有 close | ✅ Fixed |

**最终复核**：
- `go build ./...` ✅
- `go test -race ./internal/backup/... ./internal/acs/upload/... -count=1` ✅ all green
- 无 P0/P1 阻塞项
- frontend dropdown disable 外部依赖（lz4/bzip2 i18n）拆 T-0077 不延后本任务
