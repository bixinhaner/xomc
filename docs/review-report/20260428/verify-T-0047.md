# T-0047 验证报告 — `internal/core/` 测试覆盖率提升

| 项 | 内容 |
|------|------|
| 任务 | T-0047 / 章程 W2.B.3 |
| 目标 | `internal/core/` 包测试覆盖率 ≥ 50% |
| 基线 | 43.2% (任务卡描述的 16% 已不准确，实测基线 43.2%) |
| 最终 | **55.7%** |
| 提升 | +12.5 pp |
| Pass | YES |

## 基线测量

```
$ go test -coverprofile=cov_core.out ./internal/core/...
... (有一处既存失败：core/model pagination_test 与生产代码上限 1000 不一致，期望 100)
total: 43.2%
```

`internal/core/model/pagination_test.go` 的 `Test_ListRequest_Limit` 与 `Test_ListRequest_Limit_MutatesPageSize`
期望 PageSize 上限为 100，但生产代码 `pagination.go` 已改为 1000（见 commit `7440e52d`），
属于既存的"生产改了测试没跟"的破损。修复测试为新上限 1000 后基线计算成立。

## 改动清单（仅测试文件 / 修复既存失败）

| 文件 | 类型 | 说明 |
|------|------|------|
| `internal/core/model/pagination_test.go` | 修复 | 把 100 上限改为 1000，对齐生产代码 |
| `internal/core/model/time_test.go` | 新增 | `Time` 类型 JSON / Scan / Value / IsZero / NowTime 14 个用例 |
| `internal/core/model/constants_test.go` | 新增 | `ValidCarriers` + 7 个枚举完备性用例 |
| `internal/core/tracing/tracing_test.go` | 新增 | `StartSpan` / `RecordError` / `SetOK` / 名称常量唯一性 8 用例 |
| `internal/core/storage/path_test.go` | 新增 | `ObjectPath` / `ObjectPathAt` / `FirmwarePath` / `ExchangePath` / `DataModelPath` / `DatePrefix` 7 用例 |
| `internal/core/carrier/cmcc/adapter_test.go` | 新增 | CMCC adapter 接口全覆盖（Code/Name/SupportedTechnologies/DefaultDataModelVersions/参数映射/告警映射/PLMNID 校验/SupportsDirectConnection/GetInfoParamMapping）13 用例 |
| `internal/core/carrier/ctcc/adapter_test.go` | 新增 | CTCC adapter 接口覆盖 11 用例 |
| `internal/core/carrier/cucc/adapter_test.go` | 新增 | CUCC adapter 接口覆盖（含 NR-only / SST 校验等专属差异）12 用例 |

**未改动任何生产代码**，仅 `_test.go` 文件 + 一个既存失败测试修复。

## 各子包覆盖明细

```
internal/core/appconfig         60.6%  (无变化)
internal/core/carrier           100.0% (无变化)
internal/core/carrier/cmcc      96.6%  (从 0.0% → 96.6%)
internal/core/carrier/ctcc      93.0%  (从 0.0% → 93.0%)
internal/core/carrier/cucc      91.1%  (从 0.0% → 91.1%)
internal/core/components        40.2%  (无变化)
internal/core/components/redisx 66.0%  (无变化)
internal/core/errors            96.0%  (无变化)
internal/core/event             57.0%  (无变化)
internal/core/health            100.0% (无变化)
internal/core/middleware        89.4%  (无变化)
internal/core/model             97.8%  (从 26.7% → 97.8%; 修复既存失败 + 新增 time/constants 测试)
internal/core/reliability       93.2%  (无变化)
internal/core/storage           39.4%  (从 0.0% → 39.4%; 仅纯函数 path 部分，DB 路径 query.go 因需 pgxpool 不在单测范围)
internal/core/tracing           100.0% (从 0.0% → 100.0%)
─────────────────────────────────────────
TOTAL                           55.7%  (从 43.2% → 55.7%)
```

未覆盖到的 0% 子包（DB/Redis/NATS/MinIO/Logger 集成层）被故意跳过，原因：
- `components/{logger,minio,monitor,nats,postgres,redis}` 都是基础设施适配器，需要真实连接，单元测试需要 mock 服务，超出本任务范围（属于集成测）
- `core/context` 是空骨架包，无可测内容
- `core/utils` 仅有包注释，无函数

## 验证步骤

```bash
cd omcgo
go build ./...                                  # PASS
go test -race -count=1 ./internal/core/...      # 全部 ok
go test -coverprofile=cov_core.out ./internal/core/...
go tool cover -func=cov_core.out | tail -1      # total: 55.7%
```

## 结论

| 章程 W2.B.3 Pass 标准 | 状态 |
|---------------------|------|
| 覆盖率 ≥ 50% | PASS（55.7%） |
| 既有测试不破坏 | PASS（修复一处与生产代码不一致的既存失败） |
| 仅 `_test.go` 改动 | PASS（外加 1 个测试夹具修正，无生产代码改动） |
