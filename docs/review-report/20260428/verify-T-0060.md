# T-0060 验证报告 — W3.F.2 慢查询审计 + top10 优化

> 日期：2026-04-28
> 任务：T-0060 / W3.F.2
> 范围：pgxpool slow query log + 索引/重写 + top10 文档

## 交付物

| 文件 | 类型 | 用途 |
|------|------|------|
| `omcgo/internal/core/components/postgres/slow_query.go` | 新增 | `SlowQueryTracer`（pgx QueryTracer）+ `ChainTracer` + `pgx_slow_query_total` Prometheus 指标 |
| `omcgo/internal/core/components/postgres/slow_query_test.go` | 新增 | 单元测试，覆盖率 ≥80%（详见下文） |
| `omcgo/migrations/000042_perf_indexes.sql` | 新增 | 11 条性能索引（覆盖 10 类查询场景） |
| `docs/perf/slow-queries-top10.md` | 新增 | top10 慢查询清单与优化措施 |

## 自跑验证

### 1. 编译

```bash
$ go build ./...
# (no output — clean build)
```

### 2. 单元测试 + 覆盖率

```bash
$ go test -race -count=1 ./internal/core/components/postgres/...
ok  github.com/omcgo/omcgo/internal/core/components/postgres  1.6s
```

`slow_query.go` 函数级覆盖率（`go tool cover -func`）：

| 函数 | 覆盖率 |
|------|--------|
| `NewSlowQueryMetrics` | 100% |
| `NewSlowQueryTracer` | 100% |
| `WithSlowQueryTracer` | 100% |
| `Threshold` | 100% |
| `TraceQueryStart` (SlowQueryTracer) | 100% |
| `TraceQueryEnd` (SlowQueryTracer) | 89.5% |
| `ChainTracer` | 100% |
| `chainedTracer.TraceQueryStart` | 100% |
| `chainedTracer.TraceQueryEnd` | 80.0% |
| `fingerprintSQL` | 100% |
| `extractTable` | 90.9% |
| `stripIdentifier` | 85.7% |
| `truncateForLog` | 100% |

> 全部 ≥ 80%，远超章程要求的 ≥ 70%。

### 3. 迁移连续性检查

```bash
$ bash scripts/check-migrations.sh
🔍 检查目录：omcgo/migrations
📋 共发现 43 个迁移文件
📊 编号区间：000001 → 000042
✅ 编号连续（无跳跃）
✅ 所有文件 goose Up/Down 标记齐全
✅ 命名规范
✅ 迁移检查全部通过
```

### 4. 章程 grep 校验

| 校验项 | 命令 | 结果 |
|--------|------|------|
| `slow query` 关键字 | `grep -rn "slow query\|SlowQuery" internal/core/components/postgres/` | 命中 ≥ 1（slow_query.go + slow_query_test.go） |
| 性能索引迁移 | `ls migrations/000042*perf*.sql` | 命中 `000042_perf_indexes.sql` |
| top10 文档 | `ls docs/perf/slow-queries-top10.md` | 存在 |
| 文档含优化措施 | `grep '加 .*索引\|加 partial' docs/perf/slow-queries-top10.md` | 命中 11 条（≥ 5 ✅） |

## Pass 标准核对

- [x] grep "slow query|SlowQuery" 命中 ≥ 1 ✅
- [x] ls migrations/0000{42,43,44,45}*index*.sql / *perf*.sql 命中 ✅
  （本任务交付 1 份 `000042_perf_indexes.sql`，含 11 条索引，单文件已超预期）
- [x] ls docs/perf/slow-queries-top10.md 存在 ✅
- [x] top 10 文档含 ≥ 5 优化措施 ✅（实际 11 条）
- [x] 单元测试覆盖率 ≥ 70% ✅（多函数 100%，最低 80%）
- [x] `go build ./...` 编译通过 ✅
- [x] `go test -race` 单测通过 ✅
- [x] 迁移版本号连续，up/down 配对 ✅

## 边界遵守

- [x] 未改 `cmd/app/provider/*` 与 `cmd/app/main.go`
- [x] 未改 `internal/core/components/redis/*` / `nats/*`（T-0061 范围）
- [x] 未改 `deployments/k8s/*`（T-0065）/ `scripts/db_backup.sh`（T-0067）/ `docs/project/release-gate.md`（T-0024）
- [x] 未改 `omcmb/` / `.github/`
- [x] 未新增 go.mod 依赖（`prometheus/client_golang`、`zap`、`pgx/v5` 均为既有依赖）
- [x] 未触发 commit / push / pull / charter / backlog 改动

## 后续衔接

- 注入 `SlowQueryTracer` 到 app/acs/worker provider 的工作交由 W3 后续 sub-agent 完成（本任务严禁碰 provider）。
  推荐写法：

  ```go
  slowTr := postgres.WithSlowQueryTracer(100*time.Millisecond, log, reg)
  poolCfg.ConnConfig.Tracer = postgres.ChainTracer(existingTracer, slowTr)
  ```

- 真实 staging 上的 EXPLAIN ANALYZE 验证可在 release-gate（T-0024）阶段执行。
