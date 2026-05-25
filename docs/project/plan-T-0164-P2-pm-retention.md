# T-0164-P2 / G2 PM 保留策略实施 Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans.

**Goal:** 为 PM 时序数据建立统一的压缩 + 保留策略管理层，提供 sys_configs 5 键 + 默认值常量 + UI 入口字段，供 G3/G5 的 migration 内嵌挂 policy 时引用。

**Architecture:** 仅做"策略源 + 配置层"，不直接挂表 policy（挂 policy 动作下沉到 G3 pm_metrics 15min + G5 hourly/daily/weekly/monthly 各自 migration 内）。sys_configs 5 键变化时通过 EventBus 通知 G5 cron 清理任务 + admin handler 触发 hypertable policy alter_job。

**Tech Stack:** Go + sys_configs 表（已存在）+ EventBus + 前端字段映射。

**Deps:** 无（独立，G3/G5 实施时引用本任务定义的常量与键名）。

---

## 0. 文件结构

**新建**：
- `omcgo/internal/pm/retention/policies.go` — 5 个保留期常量 + key 命名规范 + 默认值（金字塔）
- `omcgo/internal/pm/retention/service.go` — 读 sys_configs → 解析 → 通知 G5 cron + admin alter_job

**修改**：
- `omcgo/migrations/000164_seed_pm_retention_sysconfigs.sql` — 5 个 sys_configs 键注入（INSERT ON CONFLICT DO NOTHING）
- `omcgo/internal/core/appconfig/sysconfigs/keys.go` — 加 5 个常量
- `omcgo/internal/admin/handler.go` — sys_configs PUT 接口在保留期键变化时触发 retention.Service.Reload

**前端**（业务层在 frontend-core）：
- `omcmb/frontend-core/src/services/api/sysConfigApi.ts` 已有；只在 i18n 加 5 键 label
- `omcmb/webcode/src/pages/admin/SystemConfig/PmRetentionSection.tsx` — 新组件，5 个 InputNumber + 单位下拉（天）+ 默认值 reset

## 1. 5 键命名与默认值（金字塔保留）

| 键 | 默认值（天） | 适用粒度 |
|----|------|--------|
| `pm.retention.raw_15min_days` | 30 | pm_metrics（15min hypertable） |
| `pm.retention.hourly_days` | 180 | pm_metrics_hourly（hypertable） |
| `pm.retention.daily_days` | 730 (2y) | pm_metrics_daily（普通表 + cron） |
| `pm.retention.weekly_days` | 730 (2y) | pm_metrics_weekly（普通表 + cron） |
| `pm.retention.monthly_days` | 1825 (5y) | pm_metrics_monthly（普通表 + cron） |

**压缩策略**（仅 hypertable，普通表无压缩）：
- pm_metrics（15min）：compress after 7d
- pm_metrics_hourly：compress after 14d

## 2. Tasks

### Task 1: 写 `policies.go` 常量与类型

**Files:**
- Create: `omcgo/internal/pm/retention/policies.go`
- Test: `omcgo/internal/pm/retention/policies_test.go`

```go
package retention

import "time"

type PolicyKey string

const (
    KeyRaw15MinDays PolicyKey = "pm.retention.raw_15min_days"
    KeyHourlyDays   PolicyKey = "pm.retention.hourly_days"
    KeyDailyDays    PolicyKey = "pm.retention.daily_days"
    KeyWeeklyDays   PolicyKey = "pm.retention.weekly_days"
    KeyMonthlyDays  PolicyKey = "pm.retention.monthly_days"
)

var DefaultDays = map[PolicyKey]int{
    KeyRaw15MinDays: 30,
    KeyHourlyDays:   180,
    KeyDailyDays:    730,
    KeyWeeklyDays:   730,
    KeyMonthlyDays:  1825,
}

type Granularity string

const (
    Granularity15Min   Granularity = "15min"
    GranularityHourly  Granularity = "hourly"
    GranularityDaily   Granularity = "daily"
    GranularityWeekly  Granularity = "weekly"
    GranularityMonthly Granularity = "monthly"
)

func KeyForGranularity(g Granularity) PolicyKey { /* ... */ }
func DefaultDuration(k PolicyKey) time.Duration { /* days * 24h */ }
```

测试：常量映射完整 + 默认值范围（>=1 day & <=10 years）。

- [ ] 写测试 → 跑 fail → 写实现 → 跑 pass

### Task 2: 写 migration 注入 5 个 sys_configs 键

**Files:**
- Create: `omcgo/migrations/000164_seed_pm_retention_sysconfigs.sql`

```sql
-- +goose Up
-- +goose StatementBegin
INSERT INTO sys_configs (key, value, description, value_type, updated_at)
VALUES
  ('pm.retention.raw_15min_days', '30',  'PM 15min 原始表保留天数（默认 30d）',   'int', NOW()),
  ('pm.retention.hourly_days',    '180', 'PM 小时聚合表保留天数（默认 180d）',     'int', NOW()),
  ('pm.retention.daily_days',     '730', 'PM 日聚合表保留天数（默认 2y）',         'int', NOW()),
  ('pm.retention.weekly_days',    '730', 'PM 周聚合表保留天数（默认 2y）',         'int', NOW()),
  ('pm.retention.monthly_days',   '1825','PM 月聚合表保留天数（默认 5y）',         'int', NOW())
ON CONFLICT (key) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM sys_configs
WHERE key IN (
  'pm.retention.raw_15min_days',
  'pm.retention.hourly_days',
  'pm.retention.daily_days',
  'pm.retention.weekly_days',
  'pm.retention.monthly_days'
);
-- +goose StatementEnd
```

**验证**：`bash omcgo/scripts/check-migrations.sh` 通过；`migrate up` + `migrate down` + `migrate up` 双向幂等。

- [ ] 写 migration → `migrate up` 验证 5 行入库 → `migrate down` 验证清空

### Task 3: 写 `retention/service.go` reload + 通知

**Files:**
- Create: `omcgo/internal/pm/retention/service.go`
- Test: `omcgo/internal/pm/retention/service_test.go`

```go
type Service struct {
    sysCfg     sysconfigs.Reader  // 读 sys_configs
    eventBus   event.Bus           // 通知 G5 cron 用新值
    timescaleClient timescale.Client // 触发 alter_job 改 hypertable policy
}

func (s *Service) Reload(ctx context.Context) error {
    // 读 5 键 → 验证范围 → 对比当前 → 变化项发 EventBus pm.retention.changed
    // hypertable 项触发 timescaleClient.AlterRetentionPolicy(table, days)
    // 普通表项发 event 通知 G5 cron 用新值
}

func (s *Service) Get(ctx context.Context, key retention.PolicyKey) (int, error) {
    // sys_configs 读，缺失走默认
}
```

测试覆盖：① reload 成功 ② 值越界（<1 或 >3650）拒绝 + log warn ③ 部分键缺失走默认 ④ EventBus 发布次数 = 变化项数。

- [ ] TDD：4 个测试用例 → 实现 → 全过

### Task 4: 注入到 admin sys_configs PUT handler

**Files:**
- Modify: `omcgo/internal/admin/handler.go`（找到 UpdateSysConfigs 或类似 handler）

在 PUT /api/v1/admin/sys-configs 成功后，若 body 含 5 个保留期键之一，调用 `retention.Service.Reload(ctx)`。

```go
// in UpdateSysConfigs handler, after successful DB update:
if anyKeyMatches(req.Keys, retention.AllKeys()) {
    if err := h.retentionSvc.Reload(c.Request.Context()); err != nil {
        logger.L(ctx).Warn("retention reload failed", zap.Error(err))
        // 不阻塞响应
    }
}
```

测试：handler 单测覆盖"键命中 retention → 调 Reload" + "键不命中 → 不调 Reload"。

- [ ] 测试 → 实现 → 全过

### Task 5: 前端 PmRetentionSection 组件 + i18n

**Files:**
- Create: `omcmb/webcode/src/pages/admin/SystemConfig/PmRetentionSection.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/admin.ts` + `en-US/admin.ts`（5 keys + label + 单位描述）

```tsx
// PmRetentionSection.tsx 骨架
const PM_RETENTION_KEYS = [
  { key: 'pm.retention.raw_15min_days', defaultValue: 30, min: 1, max: 365 },
  { key: 'pm.retention.hourly_days',    defaultValue: 180, min: 7, max: 1825 },
  // ...
];

export const PmRetentionSection: React.FC = () => {
  const { data, isLoading } = useSysConfigs(PM_RETENTION_KEYS.map(k => k.key));
  const { mutateAsync: updateConfigs } = useUpdateSysConfigs();
  // Form with 5 InputNumber + 单位（天） + Reset to default
};
```

测试：vitest snapshot + 渲染 5 InputNumber + 提交触发 useUpdateSysConfigs。

- [ ] 写组件 + 测试 → typecheck + vitest 全过

### Task 6: 集成 + commit

挂载 PmRetentionSection 到 webcode/src/pages/admin/SystemConfig/index.tsx；wire 后端 retention.Service 到 admin handler（modules.go）。

跑：
```bash
cd omcgo && go build ./... && go test ./internal/pm/retention/... ./internal/admin/...
cd omcmb/webcode && npm run typecheck && npm run test
```

commit message：
```
feat(pm): 实施 G2 PM 保留策略管理层（sys_configs 5 键 + reload 通知 + UI）

What: 新建 internal/pm/retention 包（policies.go 常量 + service.go reload）+ migration 000164 注入 5 个 sys_configs 键（金字塔保留默认值：15min=30d / hour=180d / day=2y / week=2y / month=5y）+ admin handler 接 reload 触发器 + 前端 PmRetentionSection 组件挂在系统配置页。
Why: G2 设计文档 §4.2；为 G3/G5 即将创建的 5 级粒度表提供统一策略管理层；sys_configs 改动后 EventBus 通知 G5 cron + alter_job 联动 hypertable policy。
Impact: 仅新增 5 个 sys_configs 键；未挂表 policy（留 G3/G5 migration 内嵌挂，引用本任务常量）。前端系统配置页新增 PM 保留策略卡片。

PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: -
Backlog: T-0164-P2
Review: <审查报告路径>
```

- [ ] go test + npm test 全过 → 走 /commit skill 提交

## 3. 验收

- [ ] migration up + down + up 三轮幂等
- [ ] sys_configs 表实有 5 个键 + 默认值
- [ ] retention.Service.Get 读 sys_configs，缺失走默认
- [ ] sys_configs PUT 5 个键之一后，retention.Service.Reload 被调用
- [ ] 前端系统配置页 PM 保留策略卡片渲染 5 个 InputNumber + Reset 按钮
- [ ] webcode typecheck + lint + vitest 全过；webcode-v2/v3 typecheck 仍 pre-existing baseline

## 4. Out of scope

- 实际挂 compression / retention policy on 5 表 → G3 / G5 各自 migration 内嵌做
- 历史数据迁移 → 无（项目未上生产）
- alter_job 触发的端到端真机验证 → 留早上
