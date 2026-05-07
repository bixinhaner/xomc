# S4 Verify Report — T-0098-P2-01

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-01（ProductRegistry — productClass 全局正则路由 + L1 sync.Map + L2 Redis 缓存 + 引用校验） |
| 分支 | `feature/T-0098-P1-data-dict`（P1 wave 收官分支续推 P2 第一条） |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §4.3 / §4.3.1 |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| Deps | T-0098-P1-06 done（commit `75b551bb`） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/product/model.go` | 修改 | +29 行（追加 `MatchResult` / `ValidationReport` / `ValidationIssue`） |
| `omcgo/internal/product/repository.go` | 新增 | ~30 |
| `omcgo/internal/product/pg_repository.go` | 新增 | ~165 |
| `omcgo/internal/product/cache.go` | 新增 | ~110 |
| `omcgo/internal/product/registry.go` | 新增 | ~265 |
| `omcgo/internal/product/metrics.go` | 新增 | ~70 |
| `omcgo/internal/product/cache_test.go` | 新增 | ~115 |
| `omcgo/internal/product/registry_test.go` | 新增 | ~365 |
| `omcgo/internal/product/pg_repository_test.go` | 新增 | ~35 |
| `omcgo/internal/core/components/redisx/keys.go` | 修改 | +24 行（Product 4 个键 + 3 个常量） |
| `omcgo/cmd/app/provider/productregistry.go` | 新增 | ~60 |
| `omcgo/cmd/app/provider/container.go` | 修改 | +4 行（import + ProductRegistry 字段） |
| `omcgo/cmd/app/provider/router.go` | 修改 | +6 行（productregistry 模块注册） |

总：**~1280 LOC** Go（含 ~515 LOC 测试）。

## 设计契约要点

### 1. 路由匹配 — atomic.Pointer 持有不可变 slice

`Registry.patterns` 用 `atomic.Pointer[[]compiledPattern]`：Refresh 时整个 slice 被替换（不可变快照），匹配热路径无锁直接遍历。比 sync.Map 更缓存友好，且天然可以按 sort_order 升序固定（设计 §4.3.1：全局序首命中即返回）。

### 2. 缓存层级 — L1 sync.Map + L2 Redis + DB

- `productByID` 用 sync.Map（按 ID 索引，单点取，并发安全）
- L2 通过 `Cache` 接口注入；`NopCache` 是退化形态，acs / 单进程 dev 可用
- `GetProductByID` 顺序：L1 → L2 → DB；DB 命中后回填 L1+L2（best-effort，L2 写失败仅 WARN 不返错）
- L2 失败（Redis 故障）→ 降级到 DB 直读，保留服务可用性

### 3. Refresh 失效协议

`Refresh()` 一次性：
1. `repo.ListActivePatterns` 拉全集（DB 已按 sort_order 升序）
2. 编译每条正则；编译失败 WARN 跳过（不让单条坏正则瘫痪 Registry）
3. `atomic.Store` 替换 patterns slice
4. drop L1 product 详情（避免读旧值）
5. `cache.BumpVersion`（best-effort，失败仅 WARN — 多实例最差表现是这次失效信号丢失）

### 4. ErrOrphan 语义

`MatchProductClass` 在「全部 pattern 不命中」时返回 `ErrOrphan`。设计 §4.3.1 明确：基站 productClass 路由失败应被视为孤儿待人工绑定（前端 P4-07 / API P3-01 提供绑定 UI），而不是错误丢弃。

### 5. 危险处理 — 悬挂 pattern 容错

数据不一致场景（pattern.product_id 指向不存在的 product）：当前 Registry 在该 pattern 命中时记 WARN 后继续遍历后续 pattern，让兜底正则有机会接住。这是对设计 §4.3.1「一次匹配」的健壮性扩展，避免脏数据导致整段路由瘫痪。

### 6. ValidateReferences — 软引用启动期校验（WARN-only）

| 引用 | 强度 | 处理 |
|------|------|------|
| `param_model_id` | 强（PG FK） | 由 PG 自动校验，本方法不再处理 |
| `indicator_platform` × `indicator_device_type` | 软（VARCHAR） | 查 `perf_indicators_{enb\|gsm\|gnb}` distinct platform；缺失记 `WarnedProducts` |
| `alarm_ne_type` | 软（VARCHAR） | 查 `alarm_definitions` distinct ne_type；缺失记 `WarnedProducts` |

- 同 `device_type` 的多个 product 共享一次 platform 查询（按 device_type 分桶 cache，省 N-1 次 DB 往返）
- 启动期 + handler 保存前两个调用点；handler 层（P3-01）可拒绝有 BlockedProducts 的保存（本任务保留接口口径，硬阻断逻辑由 handler 实现）

### 7. Provider 接线 — `productregistry` 模块依赖 `dictload`

router.go ModuleGraph 中 `productregistry` 显式 `Depends: ["dictload"]`，确保 P1-06 4 Loader 写完 DB 后 Registry 才 Refresh。Container.ProductRegistry 暴露给后续 P2-02..P2-10 + P3-01 消费。

### 8. Redis 键命名 — 全部走 redisx.Keys

`product:byID:{uuid}` / `product:cache_version` 全部通过 `redisx.Keys.ProductByID` / `redisx.Keys.ProductCacheVersion` 构造，命名空间集中在 `redisx/keys.go`（与 datamodel:* / acs:* 一致）。

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ 无输出 |
| go vet（变动包） | `go vet ./internal/product/... ./cmd/app/provider/... ./internal/core/components/redisx/...` | ✅ 无输出 |
| 单测（product） | `go test ./internal/product/... -race -cover` | ✅ ok / **registry 87-95% / cache 100% / metrics 100%** / 总 39%（pg_repo + 既有 P1-06 loader.go 0% 拉低，无 DB 集成测试） |
| 单测（redisx） | `go test ./internal/core/components/redisx/...` | ✅ ok |
| 全 internal/ 回归 | `go test ./...` | ⚠️ 1 项 `TestDownloadHandler` 失败 — **stash 验证（stash@{0} drop 后再跑）为 P2-01 前已存在**，与本任务无关，沿用 P1-06 verify L2 决策 |

### dev-pipeline §B3 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` — 与 P1-06 同一 flake（pre-existing），无新增失败
- [x] 无新增 TODO/FIXME/panic("not implemented")（grep 无命中）
- [x] 无新增 `if carrier == "cmcc|ctcc|cucc"` 硬编码（grep 无命中）
- [x] 公共接口无新增 `any` / `interface{}` — 唯一 `any` 命中是 `sync.Map.Range` 回调（stdlib 强制签名，非暴露 API）

### dev-pipeline §B4 硬门

- [x] 新端点 E/R 比 — N/A（本任务零新增 HTTP 端点；CRUD 在 P3-01 接力）
- [x] 迁移双向演练 — N/A（本任务无新迁移）
- [x] metric / log 名 grep — 全部能找到：
  - `product_registry_match_total` / `product_registry_match_duration_seconds` / `product_registry_cache_hit_total` / `product_registry_refresh_total` ✅
  - "ProductRegistry refreshed" / "ProductRegistry loaded" / "ProductRegistry reference validation" ✅
- [x] 累计型依赖核销 — N/A（Deps=T-0098-P1-06 已 done，非累计型）

## 单测覆盖矩阵（成功 + 失败两条路径）

| 测试 | 覆盖 |
|------|------|
| `TestRegistry_Refresh_LoadsAndCompiles` | 成功路径：2 patterns 编译入快照 |
| `TestRegistry_Refresh_SkipsBadRegex` | 失败路径：1 条坏正则不阻塞 Registry |
| `TestRegistry_Refresh_PropagatesRepoError` | 失败路径：DB 读失败传播 |
| `TestRegistry_Match_FirstHitInGlobalOrder` | 成功路径：specific 命中 + fallback 命中 |
| `TestRegistry_Match_FAPGlobalOrderDominates` | 设计契约：FAP 兜底必须排在末尾（sort_order 决定，不依赖插入顺序） |
| `TestRegistry_Match_OrphanWhenNoPattern` | 失败路径：空 pattern 集 → ErrOrphan |
| `TestRegistry_Match_OrphanWhenNoMatch` | 失败路径：有 pattern 但全不命中 → ErrOrphan |
| `TestRegistry_Match_DanglingPatternFallsThrough` | 健壮性：悬挂 pattern 不阻塞后续匹配 |
| `TestRegistry_GetProductByID_L1HitAfterFirstLoad` | 成功路径：L1 二次命中不再访问 DB |
| `TestRegistry_GetProductByID_L2HitFillsL1` | 成功路径：L2 命中后回填 L1 |
| `TestRegistry_GetProductByID_L2FailureFallsBackToDB` | 失败路径：Redis 故障降级 DB |
| `TestRegistry_GetProductByID_NotFound` | 失败路径：DB 不存在 → (nil, nil) |
| `TestRegistry_Refresh_ClearsL1AndBumpsVersion` | 跨实例失效：BumpVersion 触发 |
| `TestRegistry_ValidateReferences_HappyPath` | 成功路径：全引用 OK |
| `TestRegistry_ValidateReferences_DetectsMissingPlatform` | 失败路径：indicator_platform 缺失 → WARN |
| `TestRegistry_ValidateReferences_DetectsMissingAlarmNeType` | 失败路径：alarm_ne_type 缺失 → WARN |
| `TestRegistry_ValidateReferences_BatchesPlatformQueries` | 性能契约：同 device_type 共享一次 fetch |
| `TestRedisCache_RoundTrip` | Set/Get/Invalidate 三段路径 |
| `TestRedisCache_Version` | BumpVersion 原子递增 |
| `TestRedisCache_SetNilProductRejected` | 失败路径：nil product 拒绝 |
| `TestNopCache` | 退化形态：所有方法 no-op、版本号永远 0 |
| `TestIndicatorTableByDeviceType` | 表名静态路由 + 大小写敏感 + 未识别 deviceType 返空 |
| `TestNewPgRepository` | 构造器不 panic |

## 不在本任务交付范围（接力）

| 项 | 接力任务 |
|---|---------|
| products / patterns CRUD handler + REST API | P3-01 |
| ParamRegistry + Translator | P2-02 |
| ParamModel Intersect | P2-03 |
| AlarmDefinition Registry + 接收路径 fallback | P2-10 |
| KPI loader enabled OR 合并 + operator_code default 桶刷新 | P2-09 |
| ProductRegistry handler-level 校验拒绝 BlockedProducts（硬阻断） | P3-01 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | pg_repository.go 0 单元测试 — Pg 路径需要集成测试基础设施 | P3-01 起加 dockertest 或 testcontainers 集成测试桩 |
| L2 | `TestDownloadHandler` 在 `internal/acs/rpc/` 失败 — git stash 验证为 **P2-01 前已存在**，与 P1-06 同一 flake | 沿用 P1-06 verify L2 决策；单独 backlog 任务跟进 |
| L3 | golangci-lint 本机未装 — 用 go vet 替代（无输出） | CI 上跑 `make lint`；本机后续装 |
| L4 | Refresh 当前对 patterns 全表重读；patterns 表预期 < 100 行长期可接受 | 若未来涨到 1000+ 行可考虑 Scanner-style 增量；目前 over-engineering |
| L5 | ValidateReferences 现仅产 WARN；P3-01 handler 保存 product 时应拒绝 BlockedProducts（硬阻断） | P3-01 在 handler 层加判断 |

## 结论

**S4 出口门通过**。Registry 核心逻辑 87-95% 单测覆盖；缓存与 metrics 100%；4 个 metric + 3 类 log 全部 grep 命中。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据，可直接进入 S6 commit。
