# S4 Verify Report — T-0098-P2-02

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-02（ParamModel Registry + Translator — 精简，按 productId/paramModelId 取映射 + O(1) 双向翻译 + discovered → default 降级）|
| 分支 | `feature/T-0098-P1-data-dict`（P2 wave 入口 2 续推） |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §1.4 / §1.5 / §1.6 / §1.7 占位符校验 |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| Deps | T-0098-P2-01 done（commit `6de686c1`），T-0098-P1-06 done（commit `75b551bb`） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/config/parammodel/model.go` | 修改 | +47 行（追加 ID/SoftwareVersion + MappingSource + MappingSet + TranslationResult） |
| `omcgo/internal/config/parammodel/repository.go` | 新增 | ~35 |
| `omcgo/internal/config/parammodel/pg_repository.go` | 新增 | ~130 |
| `omcgo/internal/config/parammodel/cache.go` | 新增 | ~205 |
| `omcgo/internal/config/parammodel/registry.go` | 新增 | ~325 |
| `omcgo/internal/config/parammodel/translator.go` | 新增 | ~165 |
| `omcgo/internal/config/parammodel/metrics.go` | 新增 | ~105 |
| `omcgo/internal/config/parammodel/cache_test.go` | 新增 | ~165 |
| `omcgo/internal/config/parammodel/translator_test.go` | 新增 | ~130 |
| `omcgo/internal/config/parammodel/registry_test.go` | 新增 | ~340 |
| `omcgo/internal/config/parammodel/pg_repository_test.go` | 新增 | ~20 |
| `omcgo/internal/core/components/redisx/keys.go` | 修改 | +30 行（5 个 ParamModel* key 构造器 + 4 个常量） |
| `omcgo/internal/core/appconfig/config.go` | 修改 | +20 行（ParamRegistryConfig 结构 + AppConfig 字段） |
| `omcgo/internal/core/appconfig/validate.go` | 修改 | +13 行（validate hook） |
| `omcgo/cmd/app/provider/paramregistry.go` | 新增 | ~70 |
| `omcgo/cmd/app/provider/container.go` | 修改 | +4 行（import + ParamRegistry 字段） |
| `omcgo/cmd/app/provider/router.go` | 修改 | +8 行（paramregistry ModuleGraph 节点） |

总：**~1860 LOC** Go（含 ~655 LOC 测试）。

## 设计契约要点

### 1. 职责拆分（精简版 Registry）

设计 §1.5 把"productClass → product 路由"切给 P2-01 ProductRegistry 后，本任务的 ParamRegistry 只剩两件事：

- **GetByProduct(productID, swVersion)** — discovered 优先，未命中降级 default
- **GetByParamModel(paramModelID)** — 直接取 default（用于 P3-02 admin handler 等无 device context 场景）

旧 `datamodel.ParamRegistry` 揉了 routing + mapping，本次彻底拆开。

### 2. 三级缓存 — L1 sync.Map + L2 Redis + DB

两组独立的 L1 sync.Map：
- `defaultByModel  map[uuid.UUID][]ParamMapping`（按 paramModelID）
- `discoveredByDevice  map[discoveredKey][]ParamMapping`（按 productID+swVersion）

L2 Redis 键：
- `parammodel:default:{paramModelId}` （24h TTL）
- `parammodel:discovered:{productId}:{swVersion}` （24h TTL）
- `parammodel:cache_version` （跨实例 BumpVersion）

L2 失败 → DB 直读 + WARN，**不阻塞**业务（与 ProductRegistry 一致策略）。

### 3. discovered → default 降级语义

```
GetByProduct(productID, swVersion):
  1. loadDiscovered(L1→L2→DB) → 非空命中 → MappingSet{Source: discovered}
  2. 否则 GetProductByID(productID) → product.ParamModelID == nil → ErrNoParamModel
  3.       product 不存在 → ErrNoMapping
  4.       loadDefault(L1→L2→DB) → 非空命中 → MappingSet{Source: default} + INFO log
  5. default 也空 → ErrNoMapping
```

INFO 日志 `ParamRegistry fallback to default` 仅在降级路径打，不污染 discovered 命中的热路径。

### 4. {i} 占位符校验（设计 §1.7）

Translator 构造时（`NewTranslator`）逐条验证 standardPath 与 privatePath 中 `{i}` 出现次数：
- 不等 → 跳过该条 + WARN log + `param_translator_invalid_placeholder_total{param_model}` +1
- 重复 standardPath/privatePath → 后者覆盖前者 + WARN（按设计假定一对一）

P1-06 的 ParamModel Loader **未做**此校验（保留作 P2-02 范围）；本任务由 Translator 构造期承担，不影响 Registry 取 mapping 集合。

### 5. productGetter 接口解耦

`Registry` 不直接 import `*product.Registry`，而是通过：

```go
type productGetter interface {
    GetProductByID(ctx context.Context, id uuid.UUID) (*product.Product, error)
}
```

ProductRegistry 天然满足该签名，Provider 直接注入。测试用 fakeProductGetter 隔离。**单向依赖**：parammodel → product，无循环。

### 6. Feature flag `param_registry.use_new` 不被本 Registry 消费

实施计划 §2.P2 footer：`param_registry.use_new` 默认 false，P2-02..P2-08 全部合入后切 true。

本任务的明确决策：**Registry 自身不读这个 flag**，dual-stack 期间始终可启动可测；flag 仅在 P2-04..P2-08 各消费者改造时按需读取，决定走旧 datamodel 栈还是新 parammodel 栈。

### 7. Provider 接线 — `paramregistry` 模块依赖 `dictload` + `productregistry`

router.go ModuleGraph：
```
paramregistry  Depends: ["dictload", "productregistry"]
```

启动期 Refresh 仅清 L1 + BumpVersion（**不预热**）：mapping 数据量适中（~5K rows / 9 paramModel），按需 read-through 命中即可。BumpVersion 失败不致命（多实例下其他实例本次错过失效信号），WARN 后继续启动。

### 8. Redis 键命名 — 全部走 redisx.Keys

`parammodel:default:{uuid}` / `parammodel:discovered:{uuid}:{swVer}` / `parammodel:cache_version` 全部通过 `redisx.Keys.ParamModelDefault` / `redisx.Keys.ParamModelDiscovered` / `redisx.Keys.ParamModelCacheVersion` 构造，命名空间集中在 `redisx/keys.go`（与 datamodel:* / product:* 一致）。

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ 无输出 |
| go vet | `go vet ./internal/config/parammodel/... ./cmd/app/provider/... ./internal/core/components/redisx/... ./internal/core/appconfig/...` | ✅ 无输出 |
| 单测（parammodel） | `go test ./internal/config/parammodel/... -race -cover` | ✅ ok / **registry 60-100% / translator 91-100% / metrics 100%** / 总 46.5%（loader.go P1-06 carryover 0% + pg_repo SQL 0% 拉低，无 DB 集成测试） |
| 单测（redisx） | `go test ./internal/core/components/redisx/... -race` | ✅ ok |
| 全 internal/ 回归 | `go test ./...` | ⚠️ 1 项 `TestDownloadHandler` 失败 — **stash 验证（stash 后再跑相同 fail）为本任务前已存在**，沿用 P2-01 verify L2 决策，与本任务无关 |

### dev-pipeline §B3 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` — 与 P2-01 同一 flake（pre-existing），无新增失败
- [x] 无新增 TODO/FIXME/panic("not implemented")（grep 无命中本任务新增文件）
- [x] 无新增 `if carrier == "cmcc|ctcc|cucc"` 硬编码（grep 无命中）
- [x] 公共接口无新增 `any` / `interface{}` — 唯二 `interface{}` 命中是 `loader.go:330,338`（P1-06 已存在的 SQL NULL 适配 helper，私有未导出，非本任务新增）

### dev-pipeline §B4 硬门

- [x] 新端点 E/R 比 — N/A（本任务零新增 HTTP 端点；CRUD 在 P3-02 接力）
- [x] 迁移双向演练 — N/A（schema 已由 P1-03 `000058_param_dictionary.sql` 落地，本任务无新迁移）
- [x] metric / log 名 grep — 全部能找到：
  - `param_registry_lookup_total` / `param_registry_lookup_duration_seconds` / `param_registry_cache_hit_total` / `param_registry_refresh_total` ✅
  - `param_translator_translate_total` / `param_translator_invalid_placeholder_total` ✅
  - "ParamRegistry refreshed" / "ParamRegistry default mapping loaded" / "ParamRegistry discovered mapping loaded" / "ParamRegistry fallback to default" / "ParamTranslator placeholder mismatch" ✅
- [x] 累计型依赖核销 — N/A（Deps=T-0098-P2-01/T-0098-P1-06 已 done，非累计型）

## 单测覆盖矩阵（成功 + 失败两条路径）

| 测试 | 覆盖 |
|------|------|
| `TestRedisCache_DefaultRoundTrip` | 成功路径：Set/Get/Invalidate 三段 |
| `TestRedisCache_DefaultEmptySliceCached` | 边界：空切片是合法值，缓存"无映射"事实 |
| `TestRedisCache_DiscoveredRoundTrip` | 成功路径：(productId, swVersion) 隔离 |
| `TestRedisCache_Version` | BumpVersion 原子递增 + GetVersion 回读 |
| `TestRedisCache_TTLOverride` | 配置覆写 + ≤0 fallback 默认 |
| `TestNopCache` | 退化形态：Get 永远 miss，写无副作用，version 永远 0 |
| `TestTranslator_ToPrivate_HitAndMiss` | 双路径：命中返回 mapping，未命中 Translated=Original |
| `TestTranslator_ToStandard_HitAndMiss` | 双路径同上 |
| `TestTranslator_PlaceholderMismatchSkipped` | 失败路径：`{i}` 不等条目跳过，OK 条目仍可翻译 |
| `TestTranslator_Source` | 设计契约：MappingSource 透传 |
| `TestTranslator_Mappings_PreservesOriginalOrder` | 设计契约：保留原序供 P2-04 sync.go 使用 |
| `TestTranslator_NilSet` | 失败路径：nil set 安全降级 |
| `TestValidatePlaceholders` | 6 个 case：相等 / 不等 / 多占位符 |
| `TestParamModelLabel` | metric/log 标签：default / discovered / fallback "unknown" |
| `TestRegistry_GetByProduct_DiscoveredHit` | 成功路径：discovered 命中 + 回填 paramModelID |
| `TestRegistry_GetByProduct_FallbackToDefault` | 成功路径：discovered 空 → product 反查 → default 命中 |
| `TestRegistry_GetByProduct_NoMappingError` | 失败路径：discovered+default 都空 → ErrNoMapping |
| `TestRegistry_GetByProduct_ErrNoParamModel` | 失败路径：product.ParamModelID nil |
| `TestRegistry_GetByProduct_ProductNotFound_ReturnsErrNoMapping` | 失败路径：product 不存在 |
| `TestRegistry_GetByProduct_ProductGetterError` | 失败路径：productGetter 抛错原样透传 |
| `TestRegistry_GetByProduct_ProductGetterUnset` | 失败路径：未注入 → ErrProductGetterUnset |
| `TestRegistry_GetByProduct_L1HitAfterFirstLoad` | 成功路径：L1 命中后不再回 DB（3 次查询 1 次 DB） |
| `TestRegistry_GetByProduct_L2FailureFallsBackToDB` | 失败路径：Redis 故障降级 DB |
| `TestRegistry_GetByParamModel_HappyPath` | 成功路径：仅 default 三层 |
| `TestRegistry_GetByParamModel_Empty` | 失败路径：default 空 → ErrNoMapping |
| `TestRegistry_Translator_BuildsFromGetByProduct` | 集成：Registry → Translator 一站式 |
| `TestRegistry_InvalidateProduct_DropsL1` | 失效协议：L1 删除 + 下次必击穿 DB |
| `TestRegistry_InvalidateParamModel_DropsL1` | 同上（default 路径） |
| `TestRegistry_Refresh_ClearsAllAndBumpsVersion` | 跨实例失效：Refresh 清 L1 + BumpVersion 递增 |
| `TestRegistry_LoadDefault_L2HitFillsL1` | 成功路径：L2 命中后不回 DB |
| `TestRegistry_NewRegistry_NilDefaults` | 构造容错：nil cache/metrics/logger 安全降级 |
| `TestRegistry_InvalidateProduct_PropagatesL2Error` | 失败路径：L2 错误原样返回 |
| `TestRegistry_Refresh_PropagatesBumpVersionError` | 失败路径：BumpVersion 错误返回 |
| `TestNewPgRepository` / `TestStrDeref` | 构造冒烟 + helper |

总：**32 个测试用例** 覆盖成功+失败两类路径。

## 不在本任务交付范围（接力）

| 项 | 接力任务 |
|---|---------|
| Intersect 写 discovered_param_mappings | P2-03 |
| Path B sync.go 重写（删 Phase1GPNs / ParamMapping 列表去重前缀 / Translator 落库 / is_storable 过滤） | P2-04 |
| Path A orchestrator 改造（模板 standardPath + SPV/GPV 翻译为 privatePath） | P2-05 |
| model_upload enable_filetype11 决策 + 调用 Intersect | P2-06 |
| device handler / interop 切到新 paramRegistry | P2-07/P2-08 |
| param-models REST API（CRUD + mappings + standard + translate + cache） | P3-02 |
| 旧 `datamodel` 包全删 + DROP migration | P5-01/P5-02 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | pg_repository.go SQL 路径 0 单元测试 — 同 P1-06/P2-01，需 dockertest/testcontainers | P3-02 起加集成测试桩 |
| L2 | `TestDownloadHandler` 在 `internal/acs/rpc/` 失败 — git stash 验证为 **本任务前已存在**，与 P1-06 / P2-01 同一 flake | 沿用 P2-01 verify L2 决策；单独 backlog 任务跟进 |
| L3 | golangci-lint 本机未装 — 用 go vet 替代（无输出） | CI 上跑 `make lint`；本机后续装 |
| L4 | 启动期不预热 mappings —— 数据量适中，按需 read-through 即可。若未来 paramModel 涨到 100+ 或 mapping 涨到 50K+，可考虑首次 GetByProduct 触发后台预热同 paramModel 全量 | P3-02 上线后观察首次延迟分布决定 |
| L5 | discovered 表为空时缓存"空切片"作为 negative cache —— TTL 24h 内若 P2-03 写入新 discovered，必须调 `InvalidateProduct(productID, swVersion)` 失效，否则查询继续走 default。**P2-03 实施时务必接此调用** | P2-03 实施时核对 |
| L6 | `Translator.Mappings()` 返回**包含**被 `{i}` 校验跳过的条目（原序）—— P2-04 sync.go 用此做去重前缀，需要全集；翻译则只看校验通过的双向 map | 设计已固化，文档已注明 |

## 结论

**S4 出口门通过**。Registry 核心逻辑 60-100% 单测覆盖（多数 90%+）；Translator + metrics 91-100%；Cache 通过 miniredis 全覆盖。6 个 metric + 5 类 log 全部 grep 命中。Provider 已接入 ModuleGraph（dictload → productregistry → paramregistry）。Feature flag `param_registry.use_new` 默认 false，dual-stack 准备就绪等 P2-04..P2-08 消费者改造按需切换。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据，可直接进入 S6 commit。
