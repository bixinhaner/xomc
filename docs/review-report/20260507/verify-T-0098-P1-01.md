# S4 Verify Report — T-0098-P1-01

| 字段 | 值 |
|------|-----|
| Task | T-0098-P1-01（创建 `internal/core/dictloader/` + `DictLoaderConfig`） |
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/methodology/AI承诺对峙清单.md` W3 / `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 1 |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §5 共享基础设施 |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/core/dictloader/scanner.go` | 新增 | ~110 |
| `omcgo/internal/core/dictloader/lifecycle.go` | 新增 | ~170 |
| `omcgo/internal/core/dictloader/cache_version.go` | 新增 | ~140 |
| `omcgo/internal/core/dictloader/report.go` | 新增 | ~95 |
| `omcgo/internal/core/dictloader/scanner_test.go` | 新增 | ~115 |
| `omcgo/internal/core/dictloader/lifecycle_test.go` | 新增 | ~165 |
| `omcgo/internal/core/dictloader/cache_version_test.go` | 新增 | ~130 |
| `omcgo/internal/core/dictloader/report_test.go` | 新增 | ~85 |
| `omcgo/internal/core/appconfig/config.go` | 修改 | +21 行（DictLoaderConfig + AppConfig 字段） |
| `omcgo/internal/core/appconfig/validate.go` | 修改 | +18 行（DictLoaderConfig.validate + 调用） |
| `omcgo/internal/core/appconfig/validate_test.go` | 修改 | +16 行（3 用例） |

## 出口门核销

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ 通过（无输出） |
| go test (新包) | `go test ./internal/core/dictloader/... -count=1 -race` | ✅ 通过（覆盖率 **98.4%**） |
| go test (改动包) | `go test ./internal/core/appconfig/... -count=1` | ✅ 通过 |
| go test (邻域回归) | `go test ./internal/core/... -count=1` | ✅ 22 个子包全绿，无新增失败 |
| go vet | `go vet ./internal/core/dictloader/... ./internal/core/appconfig/...` | ✅ 无输出 |
| 静态检查 | `golangci-lint run` | ⚠️ **本地缺二进制**（`which golangci-lint` 未命中）；`go vet` 替代 |
| 新端点 E/R | — | N/A（P1-01 仅交付框架，无路由变更） |
| 迁移双向演练 | — | N/A（无迁移文件） |
| metric/log grep | — | N/A（仅 zap 结构化日志 scope 内，无新 Prometheus 指标） |
| 累计型 deps | — | N/A（Deps=空） |

## 测试覆盖明细（dictloader package）

```
ok  	github.com/omcgo/omcgo/internal/core/dictloader	2.083s	coverage: 98.4% of statements
```

- `scanner.go` — 5 用例（含 whitelist / 大小写扩展 / 隐藏文件跳过 / 子目录跳过 / 缺失目录报错 / Diff 4 种场景）
- `lifecycle.go` — 9 用例（Register 重复/空名/nil；LoadAll 成功/部分失败/并发上限/真并发；ReloadOne 成功/失败/未知；Snapshot 是 copy；Concurrent Register）
- `cache_version.go` — 8 用例（Increment / Watch 远端 bump / Watch 自增不触发自身 OnBump / Context 退出 / NilRedis / Key 命名 / 默认轮询周期 / 远端值乱码不 panic）
- `report.go` — 7 用例（Finish 时长；AddError 累积/Nil noop；FirstError；Combined errors.Join；String 含 errors 字段；ReportError Error+Unwrap）

## 行为契约要点

1. **CacheVersion 的写者 vs 读者语义**：`Increment` 同步 `cv.local`，写者自己的 `Watch` 不会再触发 OnBump（避免自我失效）；远端 INCR 才会触发本实例 OnBump。设计 §5.4 一致。
2. **Registry 错误隔离**：`LoadAll` 中某 Loader 失败不影响其他；所有 Report（含失败）写入 Snapshot；首个错误用 `fmt.Errorf("loader %s: %w", ...)` 包装返回。
3. **Scanner 不递归**：`*.xml` 大小写不敏感、隐藏文件（`.*`）与子目录均跳过；whitelist 为 nil 时返回所有（基础 contract）。
4. **Report 既能拆又能合**：`FirstError` / `Combined`（基于 `errors.Join`）二选一供调用方使用；`HasErrors` 不影响 LoadOnce 的返回值（loader 自决何时 fatal）。
5. **DictLoaderConfig 验证**：仅约束 `LoadConcurrency ∈ [0, 64]` 与 `CacheVersionPollInterval ≥ 0`；零值由 `dictloader` 包构造期 fallback 默认（concurrency=4，poll=30s）。

## 不在本任务交付范围（P1-06 接力）

- 4 个具体 Loader 实现（product / paramModel / indicator / alarm-definition）
- DI 接线（`cmd/app/provider/{config,container,modules,router}.go`）
- 启动期 LoadAll 触发与 Watch goroutine 启动
- 业务表的 cache_version 触发点（DB 写完成后调 `cv.Increment`）

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | golangci-lint 未安装；自动化流水线接入时需补 | 运维侧 / Wave 3 G.1 已 cover CI 安全扫描，可一并补 lint stage |
| L2 | `cv.Watch` 没有显式 backoff，若 Redis 长期不可达每 30s（默认）会持续 WARN 日志 | 实战观察后再决定加抑制（log dedup 或 exponential backoff） |
| L3 | `Registry.LoadAll` 未做 ctx-aware 取消单个 loader（errgroup 取消整个 group） | 由 Loader 内部决定如何响应 ctx；Reload 同理 |

## 结论

**S4 出口门通过**。可进入 S5。
