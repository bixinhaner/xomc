# T-0164-P1 / G1 KPI 路由按设备→产品→平台公式 + KPIEngine 重写 Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** 把 KPIEngine 从"代码硬编码 KPIDefinitions + 按 carrier+tech 过滤"重构为"按设备 productClass → ProductRegistry → indicator_platform → 平台公式集合路由 + 完全消费 DB 中的 perf_indicators_* / rela_platform_indicator_formula_*"。

**Architecture:** PMCollector 写入前先查 device → ProductRegistry.MatchProductClass → 拿到 indicator_platform → 用 indicator_platform 在 perf_indicators_{enb,gsm,gnb} + rela_platform_indicator_formula_{enb,gsm,gnb} 取公式子集 → 只算这台设备所属产品声明支持的指标。L1 内存缓存 + L2 Redis（带 cache_version 失效协议，顺手修 T-0106）。

**Tech Stack:** Go + pgx + Redis + Squirrel + T-0098 落库的 product / param_model / indicator 表族。

**Deps:** T-0164-P3 ✅（pm_metrics 表 + metric_type 已就位）+ T-0098 ✅。

---

## 0. 元数据路由链条

```
device.serial_number
    ↓ devices.product_class（CPE 上报的 productClass）
ProductRegistry.MatchProductClass(productClass)  ← product_class_patterns 表正则匹配
    ↓ Product { ID, IndicatorPlatform, IndicatorDeviceType, ... }
indicator_platform (eg. 'BLQ-LTE-V1')
    ↓ JOIN perf_indicators_{enb|gsm|gnb} ON platform_name
{ KPIList: [indicator_id, counter_path, statis_type, formula_ref] }
    ↓ JOIN rela_platform_indicator_formula_{enb|gsm|gnb} ON platform_name AND indicator_id
{ Formula: arithmetic_expression + counter_dependencies[] }
```

**indicator_device_type**：决定路由到 _enb / _gsm / _gnb 哪个表族（如 'enb', 'gsm', 'gnb'）。

**关键设计**：路由结果可缓存（L1 + L2），cache key 为 `kpi-route:product:{product_id}`，TTL 24h；通过 `parammodel:cache_version` 等版本协议失效（T-0106 修复后的版本协议）。

## 1. 文件结构

**新建**：
- `omcgo/internal/pm/kpi/router/router.go` — KPIRoute 主类型 + LookupByDevice + L1 缓存
- `omcgo/internal/pm/kpi/router/redis_cache.go` — L2 缓存 + cache_version 失效协议（顺手修 T-0106）
- `omcgo/internal/pm/kpi/router/router_test.go` + `redis_cache_test.go`

**修改**：
- `omcgo/internal/pm/kpi/engine.go` — 计算前调 router.LookupByDevice 拿 KPI 子集；删 carrier 适配器代码里的硬编码 KPIDefinitions 引用
- `omcgo/internal/product/registry.go` — 暴露 LookupKPIPlatform(productID) (string, error) 方法（如已存在则跳过）
- `omcgo/internal/core/dictloader/parammodel/cache.go` — 修 RedisCache.GetDefault/GetDiscovered 加 cache_version 校验（T-0106）
- `omcgo/internal/carrier/{cmcc,ctcc,cucc}/*.go` — 删除硬编码 KPIDefinitions（如存在）

**删除**：
- 任何文件中残留的 `func GetKPIDefinitions(carrier, tech string) []KPIDefinition` 类似函数

## 2. Tasks

### Task 1: 写 router.go 主类型 + 测试

**Files:**
- Create: `omcgo/internal/pm/kpi/router/router.go`
- Test: `omcgo/internal/pm/kpi/router/router_test.go`

```go
package router

type KPIRoute struct {
    ProductID         uuid.UUID
    IndicatorPlatform string
    IndicatorDeviceType string  // 'enb' / 'gsm' / 'gnb'
    Counters          []CounterDef   // perf_indicators_* 行子集
    KPIs              []KPIDef       // 同上 + arithmetic formula
}

type CounterDef struct {
    IndicatorID uuid.UUID
    Path        string       // standardPath
    StatisType  string       // sum/avg/max/pct
}

type KPIDef struct {
    IndicatorID  uuid.UUID
    Name         string
    StatisType   string
    Formula      string       // arithmetic expression
    Dependencies []string     // counter paths
}

type Router struct {
    productReg   product.Registry
    indicatorRepo indicator.Repository
    formulaRepo  formula.Repository
    l1Cache      *lru.Cache       // process-local
    l2Cache      *RedisCache       // distributed
}

func (r *Router) LookupByDevice(ctx context.Context, deviceSN string) (*KPIRoute, error) {
    // 1) productReg.MatchByDeviceSN(ctx, deviceSN) → Product
    // 2) Product.IndicatorPlatform + IndicatorDeviceType
    // 3) cache lookup（key=kpi-route:product:{product_id}）
    // 4) miss → indicatorRepo.ListByPlatform + formulaRepo.ListByPlatform → assemble KPIRoute
    // 5) populate L1 + L2 cache + emit metric
}
```

测试用例（用 mock repositories）：
1. 缓存命中 L1 → 1 次内存读
2. L1 miss / L2 hit → 1 次 Redis 读
3. L1 miss / L2 miss → DB 查 + 写两级缓存
4. productClass 未匹配 → ErrProductNotMatched（fallback 跳过本设备计算）
5. indicator_platform 在 perf_indicators_* 无记录 → 返回空 KPIRoute + log warn

- [ ] TDD 5 case → 全过

### Task 2: 写 redis_cache.go + cache_version 协议

**Files:**
- Create: `omcgo/internal/pm/kpi/router/redis_cache.go`
- Test: `omcgo/internal/pm/kpi/router/redis_cache_test.go`

```go
type RedisCache struct {
    rdb         *redis.Client
    schemaVersion string  // dictloader 启动期固定
}

func (c *RedisCache) Get(ctx context.Context, productID uuid.UUID) (*KPIRoute, error) {
    // 1) GET kpi-route:cache_version → currentVersion
    // 2) GET kpi-route:product:{product_id} → value (JSON)
    // 3) 反序列化检查 value.SchemaVersion == currentVersion，mismatch 视为 miss
    // 4) return KPIRoute or nil
}

func (c *RedisCache) Put(ctx context.Context, kr *KPIRoute) error {
    // SET kpi-route:product:{product_id} = {schema_version, kpi_route_data} EX 86400
}

func (c *RedisCache) BumpVersion(ctx context.Context) error {
    // INCR kpi-route:cache_version
    // 用于 indicator / platform 字典变更时手动失效（如 admin import）
}
```

测试：版本不匹配视为 miss / Put 后 Get 命中 / BumpVersion 后所有 Get miss。

- [ ] TDD 3 case → 全过

### Task 3: 顺手修 T-0106（parammodel cache_version 协议）

**Files:**
- Modify: `omcgo/internal/core/dictloader/parammodel/cache.go`

按 T-0106 建议：让 `RedisCache.GetDefault` / `GetDiscovered` 在 cache hit 时比对 `parammodel:cache_version` 与缓存值携带的 schema_version；mismatch 即视为 miss 击穿到 DB。

```go
type CacheEntry struct {
    SchemaVersion string
    Payload       []byte
}

func (c *RedisCache) GetDefault(...) (..., error) {
    raw, err := c.rdb.Get(...).Bytes()
    var entry CacheEntry
    if err := json.Unmarshal(raw, &entry); err != nil { return nil, ErrCacheMiss }
    if entry.SchemaVersion != c.schemaVersion { return nil, ErrCacheMiss }
    // continue with entry.Payload
}
```

**注意**：这是 T-0106 修复，应当 commit 时单独提一句"顺手修 T-0106"。

- [ ] 写测试覆盖 schema_version 漂移 → 全过；在 backlog.md §5 Proposed T-0106 行更新状态（不本任务直接闭，但提一句"由 T-0164-P1 顺手做了"）

### Task 4: KPIEngine 改造 + 删硬编码 KPIDefinitions

**Files:**
- Modify: `omcgo/internal/pm/kpi/engine.go`
- Modify: `omcgo/internal/carrier/{cmcc,ctcc,cucc}/*.go`（若存在 KPIDefinitions）

engine.go：
```go
func (e *Engine) Compute(ctx context.Context, pmFile *parser.PMFile, deviceSN string) ([]metrics.PMMetric, error) {
    route, err := e.router.LookupByDevice(ctx, deviceSN)
    if err != nil {
        if errors.Is(err, router.ErrProductNotMatched) {
            log.L(ctx).Warn("device not matched to product, skip KPI", zap.String("device_sn", deviceSN))
            return nil, nil  // 跳过本设备计算
        }
        return nil, err
    }
    
    // counter 写入：metric_type='counter'，statis_type=route.Counters[i].StatisType
    // kpi 写入：metric_type='kpi'，arithmetic 公式按 route.KPIs[i].Formula 计算
    // ...
}
```

删 carrier 包内硬编码 KPIDefinitions 列表（如 `internal/carrier/cmcc/kpi.go`），如不存在跳过。

测试：
- TestEngine_Compute_RoutesToProductKPIs（mock router 返指定 KPIList，断言 Engine 只算这些）
- TestEngine_Compute_SkipsWhenProductNotMatched
- TestEngine_Compute_WritesCorrectMetricType（counter 'counter' / kpi 'kpi'）

- [ ] TDD 3 case → 全过

### Task 5: 集成 + commit

跑：
```bash
cd omcgo && go build ./... && go test ./internal/pm/kpi/... ./internal/product/... ./internal/core/dictloader/...
# 集成测试：用 BLQ 设备 fixture（productClass 'FAPService.BLQ_xxx'）
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
SELECT product_id, indicator_platform, indicator_device_type 
FROM products 
WHERE id IN (SELECT product_id FROM product_class_patterns WHERE pattern ~ 'BaiBLQ');
"
# 跑 KPIEngine 计算一份 mock PM 文件 → 查 pm_metrics → 断言 metric_path 在 BLQ 产品声明的 KPI 子集内
```

commit message：
```
feat(pm): 实施 G1 KPI 路由按设备→产品→平台公式 + KPIEngine 重写（顺手修 T-0106）

What: 新建 internal/pm/kpi/router 包（KPIRoute 类型 + LookupByDevice + L1 LRU + L2 Redis cache_version 协议）；KPIEngine.Compute 改为先调 router.LookupByDevice 拿"设备所属产品声明支持的 KPI 子集"再算（删 carrier 适配器代码里所有硬编码 KPIDefinitions 列表 + 按 carrier+tech 过滤的旧路由逻辑）；顺手修 T-0106 — parammodel RedisCache.GetDefault/GetDiscovered 加 schema_version 校验（mismatch 视为 miss 击穿到 DB）。
Why: G1 设计文档 §4.1；T-0098 落库的 perf_indicators_* + rela_platform_indicator_formula_* + product_class_patterns 真正派上用场；让"基站只算它该算的指标"成立；多 worker 一致缓存失效。
Impact: KPIEngine 仅消费 DB 元数据，不再依赖代码硬编码；T-0106 风险闭环（schema 演进后旧缓存自动失效）；如 device.product_class 未匹配 product_class_patterns，KPI 计算跳过该设备（log warn）。

PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: T-0106 闭环
Backlog: T-0164-P1
Review: <审查报告路径>
```

- [ ] go test 全过 → /commit skill

## 3. 验收

- [ ] router 测试 5 case 全过
- [ ] redis_cache 测试 3 case 全过
- [ ] parammodel cache_version 测试覆盖 schema_version 漂移
- [ ] grep `KPIDefinitions` 全仓 → 0 命中（确认硬编码删除）
- [ ] BLQ 设备 fixture 计算路径：device.serial_number=fake-blq-001 / product_class=FAPService.BaiBLQ → router 返 BLQ 产品 KPI 子集 → engine 只算这些
- [ ] product 未匹配设备（如 product_class=gNB-100 不在 product_class_patterns）→ KPI 计算 skip + log warn
- [ ] 集成测试：跑 cpe_simulator.py 一次完整 Inform + PM upload → pm_metrics 表得到正确 metric_type 行

## 4. Out of scope

- product_class_patterns 表补齐缺失 productClass（gNB-100 / FAP-XXX）→ T-0142
- 真机 BLQ 推送 PM 文件 → T-0121 阻塞
- KPI 名称 i18n → 现状已通过 perf_indicators_*.name_i18n 字段
- 设备组维度 KPI → G5（设备组聚合）
