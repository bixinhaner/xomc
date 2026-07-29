# PM 数据完整性、聚合水位与周期可见性 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 根治 PM 制式错投、指标映射漂移、KPI 依赖缺失、窗口提前关闭和当前周期不可见问题，并先将小时窗口关闭宽限统一调整为 12 分钟。

**Architecture:** 采集入口先从 XML 内容识别制式，与设备元数据交叉校验，不一致文件只保留 MinIO 原件和隔离记录，不进入指标路由。指标启用服务负责计算并事务化写入 KPI 依赖闭包；流式聚合使用“最小宽限时间 + 队列屏障”关闭窗口，迟到事件通过修订版重建级联修正父窗口。Dashboard 查询返回活动任务版本的已发布或进行中快照，并携带周期覆盖和版本有效区间。

**Tech Stack:** Go 1.x、Gin、pgx、Squirrel、PostgreSQL/TimescaleDB、Redis、NATS JetStream、Prometheus、React、TypeScript、React Query、Ant Design。

## Global Constraints

- 所有 git 操作在仓库根目录执行；分支名使用 `codex/` 前缀。
- 后端遵循 `handler -> service -> repository/model`，SQL 使用 Squirrel + pgx。
- 先写失败测试并确认按预期失败，再写最小实现。
- 前端共享 API、Hook、Types、i18n 放在 `omcmb/frontend-core/`；页面交互放在 `omcmb/webcode/`。
- 所有用户可见文案进入中英文 i18n，不在组件中硬编码。
- 小时关闭宽限默认值统一为 `12m`；天、周、月继续为 `15m`、`30m`、`30m`。
- 12 分钟只是最小保护时间；只有对应队列屏障已被消费，窗口才允许超时关闭。
- XML 制式不一致文件不得写入 `pm_metrics`、KPI 或聚合 outbox。
- 未命中指标只计一次；`whitelist_miss`、`known_but_disabled`、`technology_mismatch` 必须分别统计。
- BLQ 映射采用显式 `reportKey` 修正，不启用全局大小写不敏感匹配。
- 启用 KPI 时自动启用递归 Counter 依赖闭包；禁用仍被已启用 KPI 引用的 Counter 时返回明确错误。
- 已发布窗口收到迟到事件必须进入可重试重建流程，不能永久忽略。
- Dashboard 的 `complete` 仅表示整个自然周期完整；任务版本片段完整使用独立字段表达。

---

## File Map

### 采集隔离与指标告警

- Modify: `omcgo/internal/pm/collector/parser.go` — 解析 XML 制式证据。
- Create: `omcgo/internal/pm/collector/technology_classifier.go` — 纯函数制式判定。
- Create: `omcgo/internal/pm/collector/technology_classifier_test.go` — LTE/NR/GSM/未知/冲突测试。
- Modify: `omcgo/internal/pm/collector/collector.go` — 白名单前执行制式校验，拆分过滤原因。
- Create: `omcgo/internal/pm/collector/quarantine_store.go` — 隔离记录接口及 PostgreSQL 实现。
- Modify: `omcgo/internal/pm/collector/result_normalization_test.go` — 验证错制式不入库、不进 outbox。
- Create: `omcgo/migrations/tsdb/000005_pm_file_quarantine.sql` — 隔离记录表。
- Modify: `omcgo/internal/pm/metrics.go`、`omcgo/internal/pm/metrics_test.go` — 三类独立计数器。
- Modify: `deployments/monitoring/alerts/omc-rules.yml` — 拆分业务告警及正确文案。

### 指标映射与 KPI 依赖闭包

- Modify: `omcgo/data/indicator-library/enb/BLQ.xml` — 修正两个 BLQ reportKey。
- Create: `omcgo/migrations/000003_blq_report_keys.sql` — 增加平台关系级 report_key，并仅修正 BLQ 映射。
- Create: `omcgo/internal/pm/indicator/dependency_resolver.go` — 解析并展开 KPI 依赖。
- Create: `omcgo/internal/pm/indicator/dependency_resolver_test.go` — 直接、递归、循环依赖测试。
- Modify: `omcgo/internal/pm/indicator/service.go`、`service_test.go` — 事务启用闭包和禁用保护。
- Modify: `omcgo/internal/pm/indicator/repository.go`、`pg_enabled_repository.go` — 事务查询启用集。

### 聚合关闭水位与迟到重建

- Modify: `omcgo/internal/pm/stream/config.go`、`config_test.go` — Go 默认 12 分钟。
- Modify: `deployments/docker/docker-compose.yml` — 开发/现场 Compose 默认 12 分钟。
- Modify: `deployments/release/bundle/deploy/docker-compose.app.yml` — 发布包默认 12 分钟。
- Modify: `deployments/docker/README.md` — 配置文档。
- Create: `omcgo/migrations/tsdb/000006_pm_aggregation_barrier_revision.sql` — 屏障、水位、修订和重建任务。
- Modify: `omcgo/internal/pm/stream/outbox_repository.go`、`outbox_relay.go` — 窗口字段和屏障发布。
- Create: `omcgo/internal/pm/stream/watermark.go`、`watermark_test.go` — 屏障状态机。
- Modify: `omcgo/internal/pm/stream/consumer.go`、`window_repository.go` — 双条件关闭和迟到标脏。
- Create: `omcgo/internal/pm/stream/rebuild.go`、`rebuild_test.go` — 重建小时及级联父窗口。
- Modify: `omcgo/internal/pm/stream/finalizer.go`、`rollup_outbox.go`、`recovery.go` — 修订版替换发布。
- Modify: `omcgo/internal/pm/stream/metrics.go` — 水位阻塞、重建和修订指标。

### Dashboard 当前周期与覆盖率

- Create: `omcgo/internal/pm/stream/progress_service.go`、`progress_service_test.go` — 活动版本窗口快照。
- Modify: `omcgo/internal/pm/adhoc/handler.go`、`handler_test.go` — 任务结果返回周期状态。
- Modify: `omcgo/internal/pm/adhoc/repository.go`、`results_query_test.go` — 活动版本优先和版本片段语义。
- Modify: `omcmb/frontend-core/src/types/pmAdhoc.ts` — 周期覆盖类型。
- Modify: `omcmb/frontend-core/src/services/api/pmAdhocApi.ts` — 映射后端状态。
- Modify: `omcmb/frontend-core/src/hooks/api/usePmAdhoc.ts` — 5 分钟定时刷新，无事件即时刷新。
- Modify: `omcmb/webcode/src/pages/performance/PmDashboard/TaskDashboardPane.tsx` — 部分结果和覆盖率展示。
- Modify: `omcmb/webcode/src/pages/performance/PmDashboard/TaskDashboardPane.test.tsx` — 活动版本、部分周期、旧版本测试。
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`、`en-US/index.ts` — 状态文案。

---

### Task 1: 建立缺陷基线和可回放验收夹具

**Files:**
- Create: `omcgo/internal/pm/collector/testdata/lte_blq_valid.xml`
- Create: `omcgo/internal/pm/collector/testdata/gsm_payload_with_lte_identity.xml`
- Create: `omcgo/internal/pm/stream/testdata/late_hourly_events.json`
- Create: `docs/qa-report/pm-integrity-acceptance.md`

**Interfaces:**
- Produces: 可由后续测试重复读取的 LTE 正常文件、GSM 错投文件和四槽位迟到事件。
- Produces: 固定验收指标和 SQL/PromQL 检查清单。

- [ ] **Step 1: 固化最小 XML 夹具**

从已确认样本裁剪出：

```text
lte_blq_valid.xml:
  ERAB.EstabInitAttNbr.Sum
  ERAB.EstabInitSuccNbr.Sum
  OTHER.CellServiceTime

gsm_payload_with_lte_identity.xml:
  BTS.ActiveSeconds
  Call.TotalAttempts
  SDCCH.Attempts
  TCH.SeizureAttempts
```

保留真实 `managedElement`、`granPeriod`、`measType` 和 `measValue` 结构，不放入客户敏感标识。

- [ ] **Step 2: 编写验收文档**

文档必须固定以下通过条件：

```text
1. GSM XML + LTE payload identity => quarantine=1, pm_metrics=0, outbox=0
2. BLQ LTE XML => C000010070/C000010080 均命中
3. 启用 K900010006 => 递归依赖 Counter 全部启用
4. 20k 文件批次在 12m 内排空 => 小时窗口不因 timeout 提前发布
5. 发布后补入第 4 槽 => revision 增加，hour/day/week 结果修正
6. 当前 daily/weekly => active task version + partial + coverage
7. 旧 task version slice => period_complete=false
```

- [ ] **Step 3: 提交夹具**

```bash
git add omcgo/internal/pm/collector/testdata omcgo/internal/pm/stream/testdata docs/qa-report/pm-integrity-acceptance.md
git commit -m "test(pm): 固化数据完整性回归夹具"
```

---

### Task 2: 将小时关闭宽限统一调整为 12 分钟

**Files:**
- Modify: `omcgo/internal/pm/stream/config_test.go`
- Modify: `omcgo/internal/pm/stream/config.go`
- Modify: `deployments/docker/docker-compose.yml`
- Modify: `deployments/release/bundle/deploy/docker-compose.app.yml`
- Modify: `deployments/docker/README.md`

**Interfaces:**
- Produces: `DefaultConfig().CloseGrace == 12*time.Minute`。
- Preserves: 环境变量 `PM_AGGREGATION_CLOSE_GRACE` 可将宽限调大，但小于 `12m` 时强制钳制到 `12m`。

- [ ] **Step 1: 写默认值失败测试**

```go
func TestDefaultConfigUsesTwelveMinuteHourlyCloseGrace(t *testing.T) {
    require.Equal(t, 12*time.Minute, DefaultConfig().CloseGrace)
    require.Equal(t, 15*time.Minute, DefaultConfig().DailyCloseGrace)
}
```

- [ ] **Step 2: 确认测试因默认仍为 5 分钟而失败**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -run TestDefaultConfigUsesTwelveMinuteHourlyCloseGrace -count=1
```

Expected: `5m0s != 12m0s`。

- [ ] **Step 3: 修改全部默认值来源**

```go
CloseGrace: 12 * time.Minute,
```

Compose 两处统一为：

```yaml
PM_AGGREGATION_CLOSE_GRACE: "${PM_AGGREGATION_CLOSE_GRACE:-12m}"
```

- [ ] **Step 4: 更新部署文档并验证**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -run 'Test(DefaultConfig|ConfigFromEnv)' -count=1
docker compose -f deployments/docker/docker-compose.yml config
rg -n 'PM_AGGREGATION_CLOSE_GRACE.*12m|默认 `12m`' deployments
```

Expected: 测试通过，Compose 渲染为 `12m`，开发与发布包无 `:-5m` 残留。

- [ ] **Step 5: 提交保护配置**

```bash
git add omcgo/internal/pm/stream/config.go omcgo/internal/pm/stream/config_test.go deployments/docker/docker-compose.yml deployments/release/bundle/deploy/docker-compose.app.yml deployments/docker/README.md
git commit -m "fix(pm): 延长小时聚合关闭宽限至十二分钟"
```

---

### Task 3: XML 实际制式识别与错投隔离

**Files:**
- Create: `omcgo/internal/pm/collector/technology_classifier.go`
- Create: `omcgo/internal/pm/collector/technology_classifier_test.go`
- Modify: `omcgo/internal/pm/collector/parser.go`
- Create: `omcgo/internal/pm/collector/quarantine_store.go`
- Create: `omcgo/migrations/tsdb/000005_pm_file_quarantine.sql`
- Modify: `omcgo/internal/pm/collector/collector.go`
- Modify: `omcgo/internal/pm/collector/result_normalization_test.go`

**Interfaces:**
- Produces:

```go
type TechnologyEvidence struct {
    Technology string
    Confidence float64
    Signals    []string
}

func ClassifyTechnology(counters []model.PMCounter) TechnologyEvidence

type QuarantineRecord struct {
    SourceFileID       uuid.UUID
    DeviceSN           string
    DeclaredTechnology string
    DetectedTechnology string
    Reason             string
    MinIOPath          string
    Evidence           []string
}

type QuarantineStore interface {
    Save(ctx context.Context, record QuarantineRecord) (inserted bool, err error)
}
```

- Consumes: parser 已生成的原始 `PMCounter.CounterName` 和 `CellID`。

- [ ] **Step 1: 写分类器失败测试**

覆盖以下表驱动用例：

```go
{
  name: "gsm fingerprints",
  names: []string{"BTS.ActiveSeconds", "SDCCH.Attempts", "TCH.SeizureAttempts"},
  want: "gsm",
},
{
  name: "lte fingerprints",
  names: []string{"RRC.AttConnEstab", "ERAB.EstabInitAttNbr.Sum", "PDCP.UpPktDelayDl"},
  want: "lte",
},
{
  name: "ambiguous stays unknown",
  names: []string{"OTHER.CellServiceTime"},
  want: "",
},
```

分类规则要求至少两个同制式家族信号且领先第二名，单个通用指标不能判定。

- [ ] **Step 2: 运行分类器测试并确认失败**

Run:

```bash
cd omcgo && go test ./internal/pm/collector -run TestClassifyTechnology -count=1
```

Expected: `ClassifyTechnology` 尚不存在。

- [ ] **Step 3: 实现纯函数分类器**

使用固定低成本前缀表：

```go
var technologyPrefixes = map[string][]string{
    "gsm": {"BTS.", "SDCCH.", "TCH.", "Call."},
    "lte": {"RRC.", "ERAB.", "PDCP.", "S1SIG."},
    "nr":  {"NR.", "GNBCU.", "GNBDU.", "NRCELL."},
}
```

输出证据最多保留 20 项，日志和数据库中不得写入全量 counter 名。

- [ ] **Step 4: 写隔离迁移**

```sql
CREATE TABLE pm_file_quarantines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id uuid NOT NULL,
    device_sn text NOT NULL,
    declared_technology varchar(16) NOT NULL,
    detected_technology varchar(16) NOT NULL,
    reason varchar(64) NOT NULL,
    minio_path text NOT NULL,
    evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_file_id, reason)
);
CREATE INDEX idx_pm_file_quarantines_created_at
    ON pm_file_quarantines (created_at DESC);
```

Down 段删除索引和表。

- [ ] **Step 5: 写 collector 失败测试**

测试注入 LTE payload identity 和 GSM XML，断言：

```go
require.NoError(t, collector.handle(...))
require.Len(t, quarantine.records, 1)
require.Zero(t, copyIngest.calls)
require.Zero(t, outbox.events)
```

隔离持久化失败时必须返回错误以触发消息重试，不能静默 ack。

- [ ] **Step 6: 在白名单之前执行交叉校验**

逻辑固定为：

```go
evidence := ClassifyTechnology(content.Counters)
if evidence.Technology != "" &&
   !strings.EqualFold(evidence.Technology, payload.Technology) {
    if err := c.quarantine.Save(ctx, record); err != nil {
        return fmt.Errorf("quarantine PM technology mismatch: %w", err)
    }
    c.metrics.TechnologyMismatchFilesTotal.
        WithLabelValues(payload.Technology, evidence.Technology).Inc()
    return nil
}
```

未知制式继续原链路，不把低置信度判定当成错投。

- [ ] **Step 7: 验证迁移和采集测试**

Run:

```bash
cd omcgo && go test ./internal/pm/collector ./internal/pm -count=1
cd omcgo && ./scripts/check-migrations.sh
```

Expected: GSM 错投被隔离；正常 LTE 文件继续入库。

- [ ] **Step 8: 提交**

```bash
git add omcgo/internal/pm/collector omcgo/migrations/tsdb/000005_pm_file_quarantine.sql
git commit -m "fix(pm): 隔离设备身份与XML制式不一致文件"
```

---

### Task 4: 拆分 PM 指标过滤计数和告警语义

**Files:**
- Modify: `omcgo/internal/pm/metrics.go`
- Modify: `omcgo/internal/pm/metrics_test.go`
- Modify: `omcgo/internal/pm/collector/collector.go`
- Modify: `omcgo/internal/pm/collector/result_normalization_test.go`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`

**Interfaces:**
- Produces:

```text
omc_pm_whitelist_miss_values_total{carrier,technology}
omc_pm_known_disabled_values_total{carrier,technology}
omc_pm_technology_mismatch_files_total{declared_technology,detected_technology}
```

- Removes after one release: 错误复用同一 backing state 的 `omc_pm_dropped_counters_total` alias。

- [ ] **Step 1: 写互斥计数失败测试**

构造 1 个未知 counter、1 个库中已知但禁用 counter、1 个已启用 counter，断言：

```text
whitelist_miss = 1
known_disabled = 1
technology_mismatch = 0
```

未知 counter 在第二阶段不得再次计入 `known_disabled`。

- [ ] **Step 2: 确认旧实现发生重复计数**

Run:

```bash
cd omcgo && go test ./internal/pm/collector ./internal/pm -run 'Test.*(Whitelist|Disabled|Metrics)' -count=1
```

Expected: 旧 `DroppedCountersTotal` 与发现计数共用 backing state，断言失败。

- [ ] **Step 3: 用独立 CounterVec 替换 alias**

```go
WhitelistMissValuesTotal      *prometheus.CounterVec
KnownDisabledValuesTotal      *prometheus.CounterVec
TechnologyMismatchFilesTotal  *prometheus.CounterVec
```

白名单函数只返回已映射 Counter；启用过滤只接收内部 indicator ID，因此两个阶段集合天然互斥。

- [ ] **Step 4: 拆分告警**

新增三条规则：

```yaml
- alert: PMReportKeysMissingFromLibrary
  expr: sum by (carrier, technology) (increase(omc_pm_whitelist_miss_values_total[15m])) > 0

- alert: PMKnownIndicatorsDisabled
  expr: sum by (carrier, technology) (increase(omc_pm_known_disabled_values_total[15m])) > 0

- alert: PMTechnologyMismatch
  expr: sum by (declared_technology, detected_technology) (increase(omc_pm_technology_mismatch_files_total[15m])) > 0
```

文案必须分别说明“未入库”“按配置过滤”“原件已隔离”，不得再声称未知值已动态登记。

- [ ] **Step 5: 验证规则和单测**

Run:

```bash
cd omcgo && go test ./internal/pm/... -count=1
docker run --rm --entrypoint promtool -v "$PWD/deployments/monitoring:/monitoring:ro" prom/prometheus check rules /monitoring/alerts/omc-rules.yml
```

Expected: 单测通过，Prometheus 规则校验成功。

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/pm deployments/monitoring/alerts/omc-rules.yml
git commit -m "fix(pm): 拆分指标漂移与禁用告警"
```

---

### Task 5: 修正 BLQ 指标映射并覆盖已部署数据

**Files:**
- Modify: `omcgo/data/indicator-library/enb/BLQ.xml`
- Create: `omcgo/migrations/000003_blq_report_keys.sql`
- Modify: `omcgo/internal/pm/indicator/loader_test.go`
- Create: `omcgo/internal/pm/indicator/blq_mapping_test.go`

**Interfaces:**
- Produces:

```text
C000010070.reportKey = ERAB.EstabInitAttNbr.Sum
C000010080.reportKey = ERAB.EstabInitSuccNbr.Sum
```

- [ ] **Step 1: 写 XML 映射失败测试**

```go
require.Equal(t, "ERAB.EstabInitAttNbr.Sum", byID["C000010070"].ReportKey)
require.Equal(t, "ERAB.EstabInitSuccNbr.Sum", byID["C000010080"].ReportKey)
```

- [ ] **Step 2: 确认当前大小写和拼写导致失败**

Run:

```bash
cd omcgo && go test ./internal/pm/indicator -run TestBLQCriticalReportKeys -count=1
```

Expected: 实际得到 `.sum` 和 `EstEstab`。

- [ ] **Step 3: 修正 BLQ.xml**

只修改 `reportKey` 和与其同义的 `enName`；内部 ID、arithmetic、formula 不变。禁止改成全局大小写不敏感匹配。

- [ ] **Step 4: 编写幂等主库迁移**

迁移通过 `rela_platform_indicator_formula_enb` 限定 `platform_name='BLQ'` 对应指标，仅更新 BLQ：

```sql
UPDATE perf_indicators_enb
SET report_key = CASE id
    WHEN 'C000010070' THEN 'ERAB.EstabInitAttNbr.Sum'
    WHEN 'C000010080' THEN 'ERAB.EstabInitSuccNbr.Sum'
END,
updated_at = now()
WHERE id IN ('C000010070', 'C000010080')
  AND EXISTS (
      SELECT 1
      FROM rela_platform_indicator_formula_enb r
      WHERE r.indicator_id = perf_indicators_enb.id
        AND r.platform_name = 'BLQ'
  );
```

`perf_indicators_enb` 的主键列是 `id`，平台关系表通过 `indicator_id` 引用；迁移测试断言更新行数为 2。

- [ ] **Step 5: 验证**

Run:

```bash
cd omcgo && go test ./internal/pm/indicator -count=1
cd omcgo && ./scripts/check-migrations.sh
```

Expected: 两个 XML 映射正确，迁移可重复执行。

- [ ] **Step 6: 提交**

```bash
git add omcgo/data/indicator-library/enb/BLQ.xml omcgo/migrations/000003_blq_report_keys.sql omcgo/internal/pm/indicator
git commit -m "fix(pm): 修正BLQ关键ERAB上报名映射"
```

---

### Task 6: KPI 启用依赖闭包与禁用保护

**Files:**
- Create: `omcgo/internal/pm/indicator/dependency_resolver.go`
- Create: `omcgo/internal/pm/indicator/dependency_resolver_test.go`
- Modify: `omcgo/internal/pm/indicator/repository.go`
- Modify: `omcgo/internal/pm/indicator/pg_enabled_repository.go`
- Modify: `omcgo/internal/pm/indicator/service.go`
- Modify: `omcgo/internal/pm/indicator/service_test.go`

**Interfaces:**
- Produces:

```go
type DependencyClosure struct {
    Requested  []string
    KPIs       []string
    Counters   []string
}

func ResolveDependencyClosure(
    requested []string,
    arithmeticByID map[string]string,
    isCounterByID map[string]bool,
) (DependencyClosure, error)
```

- [ ] **Step 1: 写依赖解析失败测试**

覆盖：

```text
K900010076 -> C000060216, C000060273
K900010006 -> C000000012, C000000005, C000120002,
              C000120001, C000010080, C000010070
KPI 引用 KPI -> 递归展开
循环依赖 -> 返回 ErrCircularIndicatorDependency
未知 token -> 返回 ErrUnknownIndicatorDependency
```

- [ ] **Step 2: 确认测试失败**

Run:

```bash
cd omcgo && go test ./internal/pm/indicator -run TestResolveDependencyClosure -count=1
```

- [ ] **Step 3: 实现依赖解析器**

复用公式 tokenizer，但新增导出的纯函数获取指标 token，禁止用字符串 `Contains` 判断 ID。输出排序后去重，保证审计和测试稳定。

- [ ] **Step 4: 写事务启用失败测试**

调用：

```go
EnableIndicators(ctx, &EnableIndicatorsRequest{
    DeviceType: "enb",
    OperatorCode: "default",
    IndicatorIDs: []string{"K900010076"},
})
```

断言同一事务写入：

```text
K900010076
C000060216
C000060273
```

任何依赖解析或写入失败时三条都不落库。

- [ ] **Step 5: 在 service 中事务写闭包**

固定流程：

```go
closure, err := s.resolveEnableClosure(ctx, dt, req.IndicatorIDs)
tx, err := s.pool.Begin(ctx)
err = s.enabledRepo.BatchCreate(
    ctx, dt, req.OperatorCode,
    append(closure.KPIs, closure.Counters...), tx,
)
err = tx.Commit(ctx)
```

返回/日志记录自动补齐的 Counter ID，便于审计。

- [ ] **Step 6: 写禁用保护测试并实现**

当禁用 `C000060216` 而 K900010076 仍启用时：

```go
require.ErrorIs(t, err, ErrEnabledKPIDependency)
require.Contains(t, err.Error(), "K900010076")
```

如请求同时禁用 KPI 和依赖 Counter，则允许同事务删除。

- [ ] **Step 7: 验证**

Run:

```bash
cd omcgo && go test ./internal/pm/indicator ./internal/pm/kpi/... -count=1
```

Expected: K006/K076 依赖闭包完整，循环/未知依赖 fail-closed。

- [ ] **Step 8: 提交**

```bash
git add omcgo/internal/pm/indicator
git commit -m "fix(kpi): 事务化维护启用指标依赖闭包"
```

---

### Task 7: 建立“12 分钟 + 队列屏障”关闭水位

**Files:**
- Create: `omcgo/migrations/tsdb/000006_pm_aggregation_barrier_revision.sql`
- Modify: `omcgo/internal/pm/stream/outbox_repository.go`
- Modify: `omcgo/internal/pm/stream/outbox_relay.go`
- Create: `omcgo/internal/pm/stream/watermark.go`
- Create: `omcgo/internal/pm/stream/watermark_test.go`
- Modify: `omcgo/internal/pm/stream/consumer.go`
- Modify: `omcgo/internal/pm/stream/window_repository.go`
- Modify: `omcgo/internal/core/event/types.go`
- Modify: `omcgo/internal/core/event/subjects.go`

**Interfaces:**
- Produces:

```go
type PMAggregationBarrierPayload struct {
    Granularity string    `json:"granularity"`
    WindowEnd   time.Time `json:"window_end"`
    BarrierID   uuid.UUID `json:"barrier_id"`
}

type WindowReadiness interface {
    RequestBarrier(ctx context.Context, windowEnd time.Time) error
    IsBarrierConsumed(ctx context.Context, windowEnd time.Time) (bool, error)
}
```

- [ ] **Step 1: 写迁移**

为 raw outbox 增加可索引窗口列：

```sql
ALTER TABLE pm_aggregation_outbox
    ADD COLUMN window_start timestamptz,
    ADD COLUMN window_end timestamptz;
CREATE INDEX idx_pm_aggregation_outbox_pending_window
    ON pm_aggregation_outbox (window_end, created_at)
    WHERE published_at IS NULL;
```

新增 `pm_aggregation_barriers(window_end, barrier_id, status, requested_at, consumed_at)`，`window_end` 唯一。

- [ ] **Step 2: 写水位失败测试**

测试状态机：

```text
now < end+12m                    => not due
now >= end+12m, pending outbox   => request barrier, not ready
barrier published, not consumed  => not ready
barrier consumed                 => ready
```

- [ ] **Step 3: 确认测试失败**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -run TestWindowWatermarkReadiness -count=1
```

- [ ] **Step 4: 原事务写 outbox 窗口列**

`InsertOutbox` 从 payload 同步写 `window_start/window_end`，不使用 JSON 表达式索引。

- [ ] **Step 5: 实现屏障发布**

只有当 `window_end <= now-12m` 且不存在 `published_at IS NULL AND window_end <= target` 的 raw outbox 时，relay 才发布屏障事件。屏障和普通事件使用同一有序 stream/consumer，保证消费屏障时，此前已发布事件均已走过 consumer。

- [ ] **Step 6: consumer 持久化已消费屏障**

屏障 handler 只更新：

```sql
UPDATE pm_aggregation_barriers
SET status='consumed', consumed_at=now()
WHERE barrier_id=$1;
```

处理成功后才 ack。

- [ ] **Step 7: TimeoutScanner 使用双条件**

`ListDueByGranularityAfter` 仍负责时间候选；小时候选必须额外 `IsBarrierConsumed(window_end)`。天/周/月继续使用父 rollup 完整性和各自 grace，不共享 raw 屏障。

- [ ] **Step 8: 验证**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -run 'Test(WindowWatermark|TimeoutScanner|Outbox)' -count=1
cd omcgo && ./scripts/check-migrations.sh
```

Expected: 队列未过屏障时，即使到达 12 分钟也不会 timeout 发布。

- [ ] **Step 9: 提交**

```bash
git add omcgo/internal/pm/stream omcgo/internal/core/event omcgo/migrations/tsdb/000006_pm_aggregation_barrier_revision.sql
git commit -m "fix(pm): 以队列屏障控制聚合窗口关闭"
```

---

### Task 8: 迟到事件修订重建与父窗口级联

**Files:**
- Modify: `omcgo/migrations/tsdb/000006_pm_aggregation_barrier_revision.sql`
- Create: `omcgo/internal/pm/stream/rebuild.go`
- Create: `omcgo/internal/pm/stream/rebuild_test.go`
- Modify: `omcgo/internal/pm/stream/consumer.go`
- Modify: `omcgo/internal/pm/stream/window_repository.go`
- Modify: `omcgo/internal/pm/stream/finalizer.go`
- Modify: `omcgo/internal/pm/stream/rollup_outbox.go`
- Modify: `omcgo/internal/pm/stream/recovery.go`
- Modify: `omcgo/internal/pm/stream/metrics.go`

**Interfaces:**
- Produces:

```go
type RebuildRequest struct {
    Key       WindowKey
    Revision  int64
    Cause     string
}

type Rebuilder interface {
    MarkDirty(ctx context.Context, key WindowKey, cause string) error
    Rebuild(ctx context.Context, request RebuildRequest) error
}
```

- [ ] **Step 1: 扩展迁移**

```sql
ALTER TABLE pm_aggregation_windows
    ADD COLUMN revision bigint NOT NULL DEFAULT 1,
    ADD COLUMN period_complete boolean NOT NULL DEFAULT false,
    ADD COLUMN version_slice_complete boolean NOT NULL DEFAULT false;

ALTER TABLE pm_aggregation_counter_rollups
    ADD COLUMN revision bigint NOT NULL DEFAULT 1;

CREATE TABLE pm_aggregation_rebuild_queue (
    task_version_id uuid NOT NULL,
    entity_key text NOT NULL,
    granularity varchar(16) NOT NULL,
    window_start timestamptz NOT NULL,
    requested_revision bigint NOT NULL,
    cause text NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    last_error text,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (task_version_id, entity_key, granularity, window_start)
);
```

- [ ] **Step 2: 写迟到修订失败测试**

场景：

```text
hourly revision=1 已发布，received_slots=3
第 4 个 15min 事件到达
=> rebuild queue pending revision=2
=> 不增加 permanent ignored counter
```

- [ ] **Step 3: 确认旧代码直接 ignore**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -run TestLateEventQueuesWindowRebuild -count=1
```

Expected: 旧代码进入 `ignore late PM aggregation event`。

- [ ] **Step 4: published 分支改为幂等标脏**

同一窗口的多个迟到事件合并为一个 pending rebuild；只在 `requested_revision` 增加时计数：

```text
omc_pm_aggregation_rebuild_requested_total{granularity,cause}
```

- [ ] **Step 5: 从持久化源重建小时**

小时重建删除该 key 的 Redis 临时状态，从 JetStream 保留的 raw 15min 事件完整 replay；禁止只把迟到增量加到旧结果。

- [ ] **Step 6: 修订版原子替换**

Finalizer 在事务中：

```sql
DELETE FROM pm_aggregation_results
WHERE task_version_id=$1 AND granularity=$2
  AND window_start=$3 AND dimension_key=$4;
```

随后插入 revision=2 的完整结果、counter rollup 和窗口元数据。事务提交前旧结果仍可读，提交后只读新修订。

- [ ] **Step 7: 父窗口按最新子修订级联**

小时 revision 增加后，把对应 daily 标脏；daily 重建从 `pm_aggregation_counter_rollups` 每个子窗口的最大 revision 读取。daily 完成后同样标脏 weekly/monthly。禁止让父窗口把旧、新两个 child revision 同时累加。

- [ ] **Step 8: 写端到端重建测试**

断言：

```text
hourly revision 1 -> 2
received_slots 3 -> 4
missing_slots 1 -> 0
daily revision 1 -> 2
KPI 由四槽 Counter 重新计算
重复投递同一迟到事件不再增加 revision
```

- [ ] **Step 9: 验证**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -count=1
cd omcgo && go test ./internal/pm/aggregator -count=1
```

Expected: 不再存在“发布后永久忽略”的代码路径；重建失败留在队列重试并产生告警指标。

- [ ] **Step 10: 提交**

```bash
git add omcgo/internal/pm/stream omcgo/migrations/tsdb/000006_pm_aggregation_barrier_revision.sql
git commit -m "fix(pm): 以修订重建处理聚合迟到事件"
```

---

### Task 9: 明确自然周期与任务版本片段完整性

**Files:**
- Modify: `omcgo/internal/pm/stream/rollup_event.go`
- Modify: `omcgo/internal/pm/stream/rollup_event_test.go`
- Modify: `omcgo/internal/pm/stream/finalizer.go`
- Modify: `omcgo/internal/pm/stream/window_repository.go`
- Modify: `omcgo/internal/pm/aggregator/query.go`
- Modify: `omcgo/internal/pm/aggregator/query_test.go`

**Interfaces:**
- Produces result metadata:

```json
{
  "task_version_id": "...",
  "effective_from": "...",
  "effective_to": "...",
  "received_slots": 4,
  "expected_slots": 24,
  "coverage_ratio": 0.166667,
  "version_slice_complete": true,
  "period_complete": false,
  "state": "partial"
}
```

- [ ] **Step 1: 写版本切换失败测试**

版本仅在 12:00–13:00 有效、收到该小时后：

```go
require.True(t, result.VersionSliceComplete)
require.False(t, result.PeriodComplete)
require.Equal(t, int64(1), result.ReceivedSlots)
require.Equal(t, int64(24), result.ExpectedSlots)
```

- [ ] **Step 2: 确认当前代码把 expected slots 裁成 1**

Run:

```bash
cd omcgo && go test ./internal/pm/stream -run Test.*VersionSlice.*NaturalPeriod -count=1
```

- [ ] **Step 3: 分离两个分母**

`expected_slots` 始终表示自然周期应有槽位（daily=24、weekly=7×24 或按父粒度、monthly=自然日数）；新增 `version_expected_slots` 表示版本有效片段应有槽位。

计算：

```go
versionSliceComplete := received >= versionExpected
periodComplete := received >= naturalExpected &&
    sourceReceived >= sourceExpected &&
    sourceIncomplete == 0
```

- [ ] **Step 4: Query extra 暴露元数据**

保持旧 `complete` 字段兼容，但其值改为 `period_complete`；新增字段全部放入 `extra`，避免破坏已有 Row API。

- [ ] **Step 5: 验证**

Run:

```bash
cd omcgo && go test ./internal/pm/stream ./internal/pm/aggregator -count=1
```

Expected: 版本片段结束不再伪装成完整自然日/周/月。

- [ ] **Step 6: 提交**

```bash
git add omcgo/internal/pm/stream omcgo/internal/pm/aggregator
git commit -m "fix(pm): 区分版本片段与自然周期完整性"
```

---

### Task 10: Dashboard 返回当前天/周进行中结果和覆盖率

**Files:**
- Create: `omcgo/internal/pm/stream/progress_service.go`
- Create: `omcgo/internal/pm/stream/progress_service_test.go`
- Modify: `omcgo/internal/pm/adhoc/handler.go`
- Modify: `omcgo/internal/pm/adhoc/handler_test.go`
- Modify: `omcgo/internal/pm/adhoc/repository.go`
- Modify: `omcgo/internal/pm/adhoc/results_query_test.go`
- Modify: `omcmb/frontend-core/src/types/pmAdhoc.ts`
- Modify: `omcmb/frontend-core/src/services/api/pmAdhocApi.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/usePmAdhoc.ts`

**Interfaces:**
- Produces:

```go
type PeriodProgress struct {
    TaskVersionID       uuid.UUID `json:"task_version_id"`
    EffectiveFrom       time.Time `json:"effective_from"`
    EffectiveTo         *time.Time `json:"effective_to,omitempty"`
    Granularity         string `json:"granularity"`
    WindowStart         time.Time `json:"window_start"`
    WindowEnd           time.Time `json:"window_end"`
    ReceivedSlots       int64 `json:"received_slots"`
    ExpectedSlots       int64 `json:"expected_slots"`
    CoverageRatio       float64 `json:"coverage_ratio"`
    VersionSliceComplete bool `json:"version_slice_complete"`
    PeriodComplete      bool `json:"period_complete"`
    State               string `json:"state"`
}
```

- [ ] **Step 1: 写活动版本选择失败测试**

同一自然日存在旧版本 published 和新版本 open 时，结果必须选择查询时刻有效的新版本；旧版本只能作为历史片段返回，不能覆盖当前状态。

- [ ] **Step 2: 写进行中快照失败测试**

从 `pm_aggregation_windows` 与 Redis `WindowState` 读取当前 daily/weekly accumulator，复用 `buildFinalizedMetrics` 的纯计算部分生成只读 preview，不改变窗口状态、不写 result/outbox。

断言：

```text
state=partial
received_slots=4
expected_slots=24
coverage_ratio=0.166667
metric values 使用当前 accumulator
```

- [ ] **Step 3: 提取无副作用计算函数**

从 Finalizer 中提取：

```go
func BuildWindowSnapshot(
    version *TaskVersionSnapshot,
    key WindowKey,
    state WindowState,
) (WindowSnapshot, error)
```

Finalizer 和 ProgressService 共用，防止 preview 与最终结果公式漂移。

- [ ] **Step 4: 扩展 results API**

`GET /pm/adhoc/tasks/:id/results` 增加：

```text
include_partial=true
```

仅对 `daily`、`weekly` 当前未关闭窗口追加 preview；小时继续只读已发布结果。响应顶层增加 `period_progress`，不把 partial 写进持久结果表。

- [ ] **Step 5: 保持查询保护**

ProgressService 必须遵守现有 Dashboard 查询超时和并发限制；单请求最多读取当前任务、当前 daily/weekly 各一个窗口。Redis/DB 超时时返回已发布结果并带：

```json
{"state":"unavailable","period_complete":false}
```

不得把超时转换成假完整或假零值。

- [ ] **Step 6: 前端 API 和 Hook 每 5 分钟刷新**

React Query 固定：

```ts
refetchInterval: 5 * 60 * 1000,
refetchOnWindowFocus: false,
refetchOnReconnect: false,
```

不订阅 WebSocket、NATS 或浏览器事件，不做即时刷新。

- [ ] **Step 7: 验证后端和 frontend-core**

Run:

```bash
cd omcgo && go test ./internal/pm/stream ./internal/pm/adhoc -count=1
cd omcmb && npm run typecheck
```

Expected: 当前版本 partial 可见，旧版本不会伪装成最新完整结果。

- [ ] **Step 8: 提交**

```bash
git add omcgo/internal/pm/stream omcgo/internal/pm/adhoc omcmb/frontend-core
git commit -m "feat(pm): 提供当前周期聚合结果与覆盖率"
```

---

### Task 11: Dashboard 展示部分周期和覆盖信息

**Files:**
- Modify: `omcmb/webcode/src/pages/performance/PmDashboard/TaskDashboardPane.tsx`
- Modify: `omcmb/webcode/src/pages/performance/PmDashboard/TaskDashboardPane.test.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Consumes: Task 10 的 `PeriodProgress` 和 partial rows。
- Produces: 当前周期状态条、覆盖百分比和版本有效区间。

- [ ] **Step 1: 写 UI 失败测试**

覆盖：

```text
partial => 显示“进行中 4/24（16.7%）”
period_complete => 显示“自然周期完整”
version_slice_complete && !period_complete => 显示“版本片段完整，当前周期不完整”
unavailable => 显示“进行中状态暂不可用”，保留已发布曲线
```

- [ ] **Step 2: 运行测试并确认失败**

Run:

```bash
cd omcmb && npm test -- TaskDashboardPane.test.tsx --runInBand
```

- [ ] **Step 3: 实现状态展示**

在粒度选择器下方使用 Ant Design `Alert`/`Progress`：

```tsx
<Alert
  type={progress.periodComplete ? 'success' : 'info'}
  message={formatPeriodState(progress, intl)}
/>
```

曲线允许 partial 点，但 tooltip 明确标记“进行中”，不与最终点使用同一状态标签。

- [ ] **Step 4: 补齐中英文语料**

语料至少包含：

```text
perf.dashboard.period.partial
perf.dashboard.period.complete
perf.dashboard.period.versionSliceComplete
perf.dashboard.period.unavailable
perf.dashboard.period.effectiveRange
```

- [ ] **Step 5: 验证**

Run:

```bash
cd omcmb && npm test -- TaskDashboardPane.test.tsx --runInBand
cd omcmb && npm run typecheck
```

Expected: 文案走 i18n，5 分钟刷新，页面不注册事件即时刷新。

- [ ] **Step 6: 提交**

```bash
git add omcmb/frontend-core/src/i18n omcmb/webcode/src/pages/performance/PmDashboard
git commit -m "feat(pm): 展示周期覆盖率和版本完整性"
```

---

### Task 12: 全量回归、现场灰度和验收

**Files:**
- Modify: `docs/qa-report/pm-integrity-acceptance.md`
- Modify: `docs/operations/告警处置Runbook.md`

**Interfaces:**
- Consumes: Tasks 1–11 的所有产物。
- Produces: 可复核的自动化测试、现场 20k 批次数据和上线回滚记录。

- [ ] **Step 1: 运行后端完整验证**

Run:

```bash
cd omcgo && go build ./...
cd omcgo && go test ./...
```

Expected: 全部成功；若 sandbox 限制本地监听，按仓库规则提权复跑。

- [ ] **Step 2: 运行前端完整验证**

Run:

```bash
cd omcmb && npm run typecheck
```

Expected: 成功。

- [ ] **Step 3: 校验迁移、Compose 和告警**

Run:

```bash
cd omcgo && ./scripts/check-migrations.sh
docker compose -f deployments/docker/docker-compose.yml config
docker run --rm --entrypoint promtool -v "$PWD/deployments/monitoring:/monitoring:ro" prom/prometheus check rules /monitoring/alerts/omc-rules.yml
```

Expected: 三项全部成功；渲染配置中小时 close grace 为 `12m`。

- [ ] **Step 4: 在测试环境先执行数据库备份和迁移**

记录主库、时序库迁移前版本和备份位置；按部署 README 使用 Compose 重建 worker/app。不得用裸进程重启脚本。

- [ ] **Step 5: 验证运行时配置**

Run on target:

```bash
docker compose -f deployments/docker/docker-compose.yml exec worker printenv PM_AGGREGATION_CLOSE_GRACE
```

Expected: `12m`。

- [ ] **Step 6: 执行错制式和指标映射验收**

分别上传夹具，查询：

```text
GSM错投：quarantine +1，pm_metrics/outbox +0，technology_mismatch +1
正常BLQ：C000010070/C000010080 均入库
K900010006/K900010076：15min 和 hourly 均有值
```

- [ ] **Step 7: 执行 20k 文件批次验收**

记录：

```text
文件完成时间
raw outbox 清空时间
NATS pending 清空时间
barrier consumed 时间
小时 published 时间
received/expected slots
late/rebuild/revision 指标
CPU、内存、磁盘 await、TSDB locks/deadlocks
```

通过条件：

```text
窗口不得在 barrier consumed 前发布
有四个原始槽位的设备小时结果 received_slots=4
无新增 permanent ignored late events
```

- [ ] **Step 8: 执行迟到修订验收**

在小时 revision=1 发布后补传一个缺失槽位，确认：

```text
rebuild queue pending -> running -> completed
hourly revision=2
父 daily/weekly revision 增加
KPI 和 Counter 同步修正
重复补传保持幂等
```

- [ ] **Step 9: 执行 Dashboard 验收**

浏览器实际页面确认：

```text
当前 daily/weekly 显示活动版本 partial
覆盖率与数据库 received/expected 一致
旧版本片段不显示为自然周期完整
网络请求仅每 5 分钟定时刷新
查询超时/并发限制仍有效
```

- [ ] **Step 10: 更新 Runbook**

写明三类新告警的含义、隔离文件查询方式、重建队列检查和失败重试步骤，以及 `12m` 只是保护值、不得用继续调大代替水位排查。

- [ ] **Step 11: 最终提交**

```bash
git add docs/qa-report/pm-integrity-acceptance.md docs/operations/告警处置Runbook.md
git commit -m "docs(pm): 补充完整性与聚合修订验收记录"
```

- [ ] **Step 12: 创建 MR 前检查**

```bash
git status --short
git log --oneline origin/main..HEAD
git diff --check origin/main...HEAD
```

Expected: 无未提交文件、无空白错误，提交按任务分层。

---

## Delivery Order and Rollback Boundaries

1. **MR 1：保护与采集正确性** — Tasks 1–6。先上线 12 分钟、错制式隔离、告警拆分、BLQ 映射和 KPI 闭包。
2. **MR 2：聚合一致性** — Tasks 7–9。上线屏障水位、迟到修订重建和周期完整性语义。
3. **MR 3：当前周期可见性** — Tasks 10–11。上线 API 和 Dashboard。
4. **现场验收提交** — Task 12，仅补 Runbook 和实测记录。

每个 MR 可独立回滚。MR 2 的数据库新增列和表采用向后兼容默认值，回滚应用时不立即删除；确认旧版本稳定后再执行 down migration。MR 3 回滚后只失去 partial 展示，不影响最终聚合结果。

## Final Acceptance Gates

- 错制式文件隔离率 100%，错误链路入库率 0%。
- `whitelist_miss` 与 `known_but_disabled` 无重复计数。
- K900010006、K900010076 的依赖 Counter 在启用集内完整。
- 20k 批次小时窗口不在事件屏障之前关闭。
- 迟到事件最终重建成功率 100%，无永久忽略路径。
- 小时完整性等于真实原始槽位完整性，不再因队列延迟额外损失。
- 当前日/周显示活动版本、进行中结果、覆盖率和有效区间。
- `complete=true` 只代表完整自然周期。
- Dashboard 保持每 5 分钟定时刷新，不做事件即时刷新。
- Dashboard 查询超时、并发限制、慢查询监控和资源告警全部继续有效。
