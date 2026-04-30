# PRD F06-topology-auto-grouping — 拓扑自动分组规则引擎激活

**PRD ID**：F06-topology-auto-grouping
**功能域**：F06 OMC-R 核心 / topology 子模块
**作者**：Claude（PM 16.10 + 架构 16.1 + 电信 16.4 联合起草）
**创建日期**：2026-04-30
**最后更新**：2026-04-30
**状态**：S2 done（2026-04-30 — D1-D7 全部拍板 + S1 排期 sprint-09 + S2 设计备忘补完 §12；待 user 确认进 S3 编码）
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`
**关联 Sprint**：`docs/project/sprint/sprint-09.md`（2026-05-01 ~ 2026-05-14）
**关联 Risk**：`docs/project/risk-register.md#R-104`（拓扑自动分组规则引擎未激活，P1 / Open）
**关联 Backlog**：`docs/project/backlog.md` §4 Triaged T-0027（**真号**，2026-04-20 早登记，2026-04-30 ID 冲突修正后保号）

---

## 1. 业务背景（Why）

**一句话**：现有规则引擎骨架已经 80% 落地（rule_model / rule_service / rule_handler / matcher 五件套 + DI 装配 + 路由注册全在），但核心电缆未通——`getAllDevices()` 返空切片、无 EventBus 订阅、无 cron 重评——导致规则永远匹配不到设备，运维只能手工拉清单往组里塞。本 PRD 的任务是**接线**，不是重做。

**当前痛点**：

- **规模天花板**：10 万级基站起步规模下，分组依赖运维人员凭借设备名 / 站点名手工拉到对应组，单次操作上限 ~500 设备/小时（MOVE API 单条调用）。100 万规模时**直接撞死**。
- **维护成本爆炸**：现网新设备日均~200 台首次注册，每天需人工分配；外勤工程师反馈"分组工作量已占运维 30% 时间"（运维内部统计 2026-03，未公开）。
- **规则数据已存在但休眠**：`device_rules` 表 + 规则编辑前端页 (`omcmb/webcode/src/pages/device/DeviceRules/`) 已经支持 `name` / `lac` / `tac` 三种 matchingMode；用户能创建规则但**点"应用"按钮永远 0 命中**（因为后端 `getAllDevices()` 是 stub）。
- **不做会发生什么**：T-0025 RC 冻结已通过依赖累计阈值，但生产部署到 100 万规模时此处必然爆炸；与 T-0030 F10 互操作扩充用例（按 OUI / model 分组跑）形成阻塞链。

**为什么现在做**：

- 主链路任务（Wave 1/2/3）已完成 95%+；P0 风险关闭 1/5（剩余 4 项均 staging 外部依赖），本任务 P1 是**纯本地代码可推**的最后一颗高价值石头。
- 规则引擎骨架是过去 Sprint 已落的沉没投入，**不接通则全部浪费**；越早接通收益越早开始累积。
- DeviceMatcher 的 LAC / TAC / Name 三种 mode 已含完整 evaluator 逻辑，缺的是数据源 + 触发器 + 调度器三件——即"接线"工作。

---

## 2. 用户故事（Who / What）

> As a **运维人员**，I want **新设备首次上线后被自动分到正确的设备组**，So that **不必每天手工分配 200+ 台新设备，把分组工作量从 30% 降到 < 5%**。

> As a **运营规划人员**，I want **按 LAC / TAC / 设备名前缀创建的规则能一键评估并应用到全网存量设备**，So that **机房调整 / 区域重新划分时不必逐台 MOVE，单次规则应用覆盖 ≥ 1 万台设备**。

> As a **系统管理员**，I want **规则引擎能在每小时自动重评启用规则**，So that **设备搬迁、LAC 重配后分组自动跟上，无需我深夜守在控制台手动 ApplyRule**。

> As a **运维人员**，I want **手工指定的分组关系不被规则覆盖**，So that **特殊设备（如测试样机、VIP 客户基站）的人工标注永久保留**。

> As a **客服 / 工单人员**，I want **看到设备所在分组的来源（手工 / 规则）以及规则名**，So that **回答"为什么这台设备在 X 组"时能快速定位是规则匹配还是人工指派**。

---

## 3. 验收标准（Given/When/Then）

### A1 — 手工触发规则应用对存量设备生效

```
Given: 数据库中已有 1000 台设备（device_info 字段就绪）
       已创建规则 R = {matching_mode: "lac", lac_list: [100, 101], target_group_id: G, enabled: true, priority: 50}
       其中 300 台设备的 device_info.lac ∈ {100, 101}
When:  调用 POST /device-rules/:R/apply
Then:  - device_rule_tasks 新增一行 status: success, matched_count: 300
       - device_group_members 中 group_id=G 多了 300 行 source_type='rule'
       - 响应 JSON 含 {applied_count: 300, error_count: 0}
       - EventBus 发布 topology.rule.applied 事件含 rule_id / matched_count
       - 任务执行延迟 ≤ 5s（1000 设备规模）
```

### A2 — 新设备首次上线触发自动评估

```
Given: 已有启用规则 R 监控 LAC=200 的设备入 G 组
       数据库中无现存匹配设备
When:  CPE 首次上线，bootstrap Inform 携带 LAC=200
       触发 device.inform.bootstrap 事件
Then:  - 30 秒内设备被加入 G（device_group_members 新增 source_type='rule'）
       - device_rule_tasks 不新增（自动评估走单设备路径，不入 task 表）
       - log 含 "rule.auto_eval matched device=... rule=R group=G"
```

### A3 — 多规则同时匹配按 priority 数字小者胜

```
Given: 启用规则 R1（priority=10，target_group=G1）matchingMode=name 含 "test-"
       启用规则 R2（priority=20，target_group=G2）matchingMode=lac=300
       新设备 D 名 "test-001" 且 LAC=300（同时符合 R1 和 R2）
When:  device.inform.bootstrap 触发
Then:  - D 被加入 G1（不加入 G2）
       - device_group_members 仅新增一行 group_id=G1 source_type='rule'
       - log 含 "rule.priority_winner R1 < R2 (10 < 20)"
```

### A4 — 手工分组在 cron 重评中保留

```
Given: 设备 D 通过 ApplyRule 被规则 R 加入 G 组（source_type='rule'）
       运维手工 MOVE D 到 H 组（POST /device-groups/H/devices，sourceType 改为 'manual'）
       规则 R 的 LAC 阈值改了，D 已不再符合 R predicate
When:  @hourly cron 触发 re-evaluation
Then:  - D 仍在 H 组（source_type='manual' 不被触动）
       - D 不被加入或移回 R.target_group_id
       - cron 任务日志记 "skipped manual: D"
```

### A5 — 规则评估失败不影响其他规则

```
Given: 启用规则 R1（其 target_group_id 指向已被 DELETE 的孤儿 group）
       启用规则 R2（正常）
When:  调 POST /device-rules/all/apply（批量应用所有启用规则）
Then:  - device_rule_tasks 中 R1 行 status='failed'，error_msg 含 "target group not found"
       - device_rule_tasks 中 R2 行 status='success'
       - HTTP 200 响应含 {total_rules: 2, success: 1, failed: 1}
       - metric `omc_topology_rule_apply_failures_total` +1
       - 不影响 R2 已分配设备
```

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 备注 |
|------|------|------|------|------|
| 规则匹配字段（LAC / TAC / Name / OUI） | 通用 LTE/5G + TR-069 概念 | 同 | 同 | LAC/TAC 由 3GPP 标准定义；name 由运营商自定义但 OMC 不强制规范 |
| 自动分组触发事件 | `device.inform.bootstrap` | 同 | 同 | TR-069 标准事件 1 BOOT / 0 BOOTSTRAP |
| 行政区域 / 网格分组 | CMCC 内部有"三级网格"概念 | — | — | **本 PRD 非目标，延后专项** |
| 规则优先级语义 | 数字小者胜 | 同 | 同 | OMC 平台层一致 |

**结论**：**无 — 三家一致**。理由：

1. 规则引擎仅基于 TR-069 标准设备属性（LAC/TAC/OUI/ProductClass/Name）评估，这些是 3GPP + Broadband Forum 标准字段，不存在运营商专有差异。
2. CMCC "三级网格"是行政区划分，需要额外 `admin_region` 字段（当前 schema 不支持，且非 P1 RC 需求）—— 显式列入 §5 非目标。
3. 任何后续运营商专有维度（如 CMCC 网格 / CTCC 大区）通过 `Carrier` 接口扩展点 `RuleAttributes(carrier)` 实现，本 PRD 不预先开洞。

---

## 5. 非目标（Non-Goals）

- ✂️ **地理坐标多边形分组**（按 `latitude` / `longitude` + GIS polygon predicate）：需 PostGIS 扩展 + 新增 polygon 列；ROI 当前不足，待 GA 后用户反馈再启动
- ✂️ **CMCC 三级网格 / 行政区域分组**：需要 `admin_region` 字段 + 字典表 + 行政区联动 UI；CMCC 规范侧也未在 RC 范围强求，延后专项
- ✂️ **拖拽式 / 表达式编辑器规则设计器**：现有 4 字段表单（matchingMode + name_rule_list + lac_list + tac_list）足够 MVP，复杂表达式延后
- ✂️ **device.inform.periodic / value_change 高频事件触发**：心跳 5min/次频率太高，cron @hourly 重评足够；防止规则反复触发分组震荡
- ✂️ **规则之间冲突的复杂解决（合并组 / 分裂组）**：本 PRD 仅 priority asc first-match-wins；多组归属语义留给后续如需要扩
- ✂️ **审计 audit log 独立详表**：device_rule_tasks 已记录 rule_id / status / matched_count / failed_count / error_msg / created_at，足够追溯；不另起 audit_log
- ✂️ **跨用户 RBAC 权限模型扩展**：沿用现有 `role_device_groups` 表（用户在哪些组有权见 → 规则 ApplyRule 不破坏此层级）；不新增"规则可见 / 不可见"维度
- ✂️ **规则 dry-run / 模拟预览 UI**：后端 `/device-rules/:id/evaluate` 端点（仅查不写）作为隐式 API 暴露，FE 不在 MVP 加按钮
- ✂️ **跨实例规则同步 / 高可用**：规则评估走 NATS JetStream 队列（W3.E.1 已就位），多实例自然分布消费；本 PRD 不另设强一致协议

---

## 6. 依赖

### 阻塞项（必须先解决）

- [x] **rule_engine 骨架**（rule_model / rule_service / rule_handler / matcher 五件套 + DI 装配 + 路由注册）— **已完成**（commit history 早期沉没投入）
- [x] **EventBus subjects 5 类迁移**（含 device.inform.bootstrap）— **W3.E.2 完成 2026-04-28** commit `e00c39d5`
- [x] **devices + device_info 表 schema**（LAC / TAC / OUI / manufacturer 字段全在）— migrations/000003 早期落地

### 被阻塞项（本功能不完成会影响什么）

- **T-0030 F10 互操作用例库扩充**：互操作场景常需要按"OUI=华为 + product_class=PicoRRU" 类分组跑用例；自动分组不通则手工 setup 用例数据
- **T-0035 前端多皮肤 v2 DeviceRules 页面**：webcode-v2 需共用同一规则 API；本 PRD 不变更 API 契约即不阻塞，但 source_type 字段需同步暴露
- **生产部署 100 万规模目标**：手工分组到 10 万就撞墙，本 PRD 是规模 PR 的前置条件

### 外部依赖

- **无**：不新增第三方库；不联调外部凭据；不依赖 staging 环境（本地 docker compose 即可验证）

---

## 7. 度量（如何证明上线成功）

| 指标 | 基线 | 目标 | 度量方式 |
|------|------|------|---------|
| `omc_topology_rule_evaluations_total{result="matched"}` | 0（stub 永远不增）| ≥ 1000/h（生产规模）| Prometheus counter |
| `omc_topology_rule_evaluation_duration_seconds{rule_id}` p95 | N/A | < 5s（1 万设备 / 单规则）| Prometheus histogram |
| `omc_topology_rule_apply_failures_total` | N/A | < 1% 总评估数 | Prometheus counter ratio |
| `omc_topology_devices_in_rule_groups` / 总分组设备数 | 0%（全手工）| ≥ 80%（生产稳态）| Prometheus gauge ratio |
| 运维分组工作量 | ~30% 工时 | < 5% 工时 | 运维内部统计季度回顾 |
| device.inform.bootstrap → group 分配延迟 p95 | N/A（无触发器）| < 30s | tracing span |

**反例监控**（上线后**不应**发生）：

- ❌ 规则评估锁库导致 inform 处理变慢 > 100ms（应通过 NATS 异步消费隔离）
- ❌ cron @hourly 重评把生产 PG CPU 打 > 50%（应分批 LIMIT 1000）
- ❌ source_type='manual' 行被 cron 重评误覆盖（A4 用例硬守）
- ❌ 规则间相互触发死循环（应在事件路径加 idempotency_key 去重）

---

## 8. 实施要点（非规范性，供参考）

**预计涉及模块**：

- `omcgo/internal/topology/`（**主战场**）
  - `rule_service.go:507-509` — 实现 `getAllDevices()` 接 `DeviceRepository.ListAll(ctx, filter)`
  - `rule_service.go` — 新增 `subscribeDeviceEvents(eventBus)` 订阅 `device.inform.bootstrap`
  - `rule_service.go` — 新增 `Start(ctx)` 启动 cron @hourly 重评
  - `pg_repository.go` — 新增 `UpdateMemberSource(ctx, groupID, deviceID, source string)` 区分 manual/rule
  - `matcher.go` — 已有 evaluate 逻辑，新增 priority sort helper（ApplyAllRules 时用）
- `omcgo/cmd/app/provider/modules.go` — DI 注入 EventBus 给 DeviceRuleService（当前未注入）
- `omcgo/migrations/` — 1 条新迁移（编号续上当前最大 + 1）
- `omcmb/frontend-core/` — `useTopology.ts` 暴露 `sourceType` 字段读取（已有 mock 即接，0 改动）
- `omcmb/webcode/src/pages/device/DeviceGrouping/` — 表格列加 "来源" / "规则名"（M-S 改动）

**预计新端点**：

无（已有 14 个规则相关 REST 端点完整）。

**预计新增迁移**（1 条）：

```sql
-- +goose Up
ALTER TABLE device_group_members
  ADD COLUMN IF NOT EXISTS source_type VARCHAR(16) NOT NULL DEFAULT 'manual'
  CHECK (source_type IN ('manual', 'rule'));
ALTER TABLE device_group_members
  ADD COLUMN IF NOT EXISTS source_rule_id UUID;
CREATE INDEX IF NOT EXISTS idx_device_group_members_source
  ON device_group_members(source_type, source_rule_id);

-- +goose Down
DROP INDEX IF EXISTS idx_device_group_members_source;
ALTER TABLE device_group_members DROP COLUMN IF EXISTS source_rule_id;
ALTER TABLE device_group_members DROP COLUMN IF EXISTS source_type;
```

**预计工作量**：M = 1-3 人日

- 0.5 d：getAllDevices() 实现 + 单元测试
- 0.5 d：EventBus 订阅 + 单设备评估路径
- 0.5 d：cron @hourly 重评 + manual 保护
- 0.5 d：migration 000NNN + UpdateMemberSource + repo 测试
- 0.5 d：metric / log / 集成测试 / E2E claim 1-2 条
- 0.5 d：FE 表格列改 + 类型透传

**注**：实现细节以代码评审为准；§11 决策点未拍板前 S2 设计备忘需补完。

---

## 9. 审批

| 角色 | 姓名 / 占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理（PM） | @chenbo01 / Claude(代起草) | 2026-04-30 | 需 §11 决策点拍板 |
| 架构师 | TBD | | rule_engine 设计 + EventBus 集成方案审 |
| 领域专家（电信） | TBD | | LAC/TAC 语义 + 运营商差异确认 |
| QA / 发布经理 | TBD | | DoD 清单 + E2E 用例规划 |

---

## 10. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-04-30 | v1.0 | 初稿（基于三路 ULTRATHINK 调研：rule_engine 代码现状 + R-104 + 业务文档摸底 + Carrier 差异 + 前后端协作面）| Claude（dev-pipeline §B0） |

---

## 11. 待 PM 拍板的决策点（S0 → S1 之前必须回答 ≥ 大头）

> **守护**：S2 设计入口要求待定点 < 3。下面 7 条 PM 需明确回答 ≥ 5 条；剩余 ≤ 2 条可在 S2 设计备忘中由架构师补论证。

### D1 — 自动评估的触发事件范围（推荐方案 ✅）

- ✅ **D1.A（推荐 default）**：仅订阅 `device.inform.bootstrap`（首次注册）
- ⬜ D1.B：D1.A + `device.inform.value_change`（参数变更也重评）
- ⬜ D1.C：D1.A + 一切 `device.inform.*`

**推荐 D1.A 理由**：bootstrap 是低频事件（生产 ~200/d）；value_change 高频（5min/设备）；如要支持 LAC/TAC 变更触发分组迁移，让 cron @hourly 兜底足够实时（场景容忍度 1 小时）。

### D2 — Cron 重评频率

- ✅ **D2.A（推荐 default）**：`@hourly`（每小时）
- ⬜ D2.B：`@daily`（每天 02:00）
- ⬜ D2.C：可配置（admin_settings 表）

**推荐 D2.A 理由**：与 backup PolicyMonitor `@hourly` 模式一致；100 万规模下 1 小时一轮全表扫单规则匹配 < 5min（按 backup orphan reaper 实测扩外推）。

### D3 — Manual override 是否被规则反向拆解

- ✅ **D3.A（推荐 default）**：永久保留，cron / 自动评估都不触动 `source_type='manual'` 行
- ⬜ D3.B：用户可在 UI 设"重置为规则托管"开关
- ⬜ D3.C：规则更新时强制重评（覆盖 manual）

**推荐 D3.A 理由**：A4 用例硬守；用户语义明确——手工就是手工，要切回规则托管必须显式 DELETE 后重新 ApplyRule。

### D4 — 单设备多规则匹配

- ✅ **D4.A（推荐 default）**：priority asc 数字小者胜，仅最高 priority 规则的 target_group 生效
- ⬜ D4.B：所有匹配规则全部生效（设备同时属于多个组）
- ⬜ D4.C：报错拒绝（规则间冲突视为配置错误）

**推荐 D4.A 理由**：device_group_members PK = (group_id, device_id)，单设备多组在 schema 层支持；但运维心智成本高，单 group 最简；priority 已是数据模型字段，复用即可。

### D5 — Default group 行为 ✅ 拍板 2026-04-30：D5.B

- ⬜ D5.A：规则引擎激活后，default group 的设备保留不动（本 PRD 不动 default 组）
- ✅ **D5.B（拍板）**：default group 自动随规则重评迁出（凡符合规则的设备从 default 迁到目标组）

**理由**（PM 拍板时 ULTRATHINK）：default 是"未分类临时容器"语义，不是 manual 永久归属；首次 inform 自动入 default 是**未经决策的安置**，不应受 A4 manual override 保护（manual 是用户**显式**操作语义）。D5.A 让 default 变成"永久未分类垃圾桶"，规则引擎价值砍半。**风险缓解**：source_type='rule' 标记后运维 manual 改一下即升级 'manual'（A4 守护），首次激活前可在 admin UI 加 banner 提示。

### D6 — 规则评估的事件发布

- ✅ **D6.A（推荐 default）**：每次 ApplyRule 完成 → 发 `topology.rule.applied { rule_id, matched_count, applied_count, failed_count }`；单设备评估也发 `topology.rule.matched { device_id, rule_id, group_id }`
- ⬜ D6.B：仅 ApplyRule 发，单设备静默
- ⬜ D6.C：不发任何事件

**推荐 D6.A 理由**：北向 OSS（F08）后续可订阅；前端 SSE 实时刷新分组也用得上；与 W3.E.2 5 类 subject 设计一致。

### D7 — 规则的范围维度（运营商专有维度纳入与否）✅ 拍板 2026-04-30：D7.A

- ✅ **D7.A（拍板）**：本 PRD 仅做 LAC/TAC/Name 三维度（与现有 matchingMode 一致）
- ⬜ D7.B：本 PRD 加 OUI / manufacturer / product_class 三维度（同步扩 matcher）
- ⬜ D7.C：本 PRD 仅接线，不扩维度（D7.A 即接线现状）

**理由**（PM 拍板时 ULTRATHINK）：T-0027 焦点是"激活骨架"非"扩展维度"，scope 收紧让任务更稳过门；LAC/TAC/Name 三维度已能解决 80% 场景，无紧迫业务驱动加 OUI/manufacturer/product_class；维度扩展可作 followup（一旦有"按 OUI 分组"需求即开 T-0098 子任务），ROI 实际驱动。维持 Est=M（1-3 天），与 §D2 cron @hourly + S2 4 接线断点工作量协同。T-0030 F10 互操作如未来需要 OUI 维度可一并扩展。

---

## 12. 设计备忘（S2 — 2026-04-30 完成）

> 激活 §16.1 架构 + §16.4 电信 + §16.2 Go + §16.5 数据 + §16.6 前端 + §16.9 运维 6 专家。
> S2 出口门：接口契约 ✅ / 迁移草案 ✅ / Carrier 扩展点 ✅ / 埋点名字 ✅ / 待定点 < 3 ✅。

### 12.1 接口签名（关键设计）

```go
// internal/topology/rule_service.go (新增依赖)

// DeviceLister — 替换现有 stub `getAllDevices()` 的最小接口
// 实现位于 internal/device/repository.go（新加 ListAllForRuleEval 方法）
type DeviceLister interface {
    ListAllForRuleEval(ctx context.Context) ([]*model.Device, error)
}

// DeviceRuleService 注入 3 个新依赖
type DeviceRuleService struct {
    repo         DeviceRuleRepository
    matcher      *DeviceMatcher
    deviceLister DeviceLister              // ← 新（D5.B 实现核心）
    eventBus     event.EventBus            // ← 新（订阅 device.inform.bootstrap）
    cron         *cron.Cron                // ← 新（@hourly 重评）
    workerCount  int
    taskQueue    chan *RuleTask
}

// 新方法
func (s *DeviceRuleService) Start(ctx context.Context) error
    // 启动 cron @hourly + EventBus 订阅；DI 装配阶段被调用
func (s *DeviceRuleService) handleDeviceBootstrap(ctx context.Context, evt *event.DeviceInformEvent) error
    // 单设备评估路径（A2 GWT）；按 priority 排序找首个 match 规则
func (s *DeviceRuleService) reEvaluateAll(ctx context.Context) error
    // cron 入口；分批 LIMIT 1000 + source_type 过滤（保留 manual 行 A4）
```

### 12.2 路由 — 0 新增

现有 14 个 rule REST 端点（`/device-rules/*`）已 `router.go:296` 注册。本任务**不新增端点**；仅完善 ApplyRule 行为（D5.B：对 default 组的 device_group_members 行也评估）。

### 12.3 迁移草案 `migrations/000051_device_group_member_source.sql`

```sql
-- +goose Up
ALTER TABLE device_group_members
  ADD COLUMN IF NOT EXISTS source_type VARCHAR(16) NOT NULL DEFAULT 'manual'
  CHECK (source_type IN ('manual', 'rule'));
ALTER TABLE device_group_members
  ADD COLUMN IF NOT EXISTS source_rule_id UUID;
CREATE INDEX IF NOT EXISTS idx_device_group_members_source
  ON device_group_members(source_type, source_rule_id);

-- +goose Down
DROP INDEX IF EXISTS idx_device_group_members_source;
ALTER TABLE device_group_members DROP COLUMN IF EXISTS source_rule_id;
ALTER TABLE device_group_members DROP COLUMN IF EXISTS source_type;
```

**历史数据策略**：DEFAULT 'manual' 让所有现存 device_group_members 行受 A4 保护；新规则触发的 INSERT/UPDATE 显式置 source_type='rule' + source_rule_id。

### 12.4 Carrier 扩展点 — 无

§4 已说明三家一致（LAC/TAC/Name 是 3GPP+TR-069 标准字段）。本任务**不**触及 `internal/carrier/`；不引入 `if carrier == ...` 硬编码。

### 12.5 观测埋点完整名字清单

#### Prometheus metrics（6 个）

| 名 | 类型 | 标签 | 说明 |
|----|------|------|------|
| `omc_topology_rule_evaluations_total` | Counter | `result=matched\|skipped\|failed`，`source=manual\|cron\|inform` | 评估总数（§7 度量第 1） |
| `omc_topology_rule_evaluation_duration_seconds` | Histogram | `rule_id` | 单规则评估耗时（§7 度量第 2） |
| `omc_topology_rule_apply_failures_total` | Counter | `reason=db_error\|target_group_missing\|other` | 失败计数（§7 度量第 3） |
| `omc_topology_devices_in_rule_groups` | Gauge | — | 当前归属规则组的设备数（§7 度量第 4） |
| `omc_topology_active_rules` | Gauge | — | 当前启用规则数（新增） |
| `omc_topology_default_group_migrations_total` | Counter | `result` | D5.B 专用：从 default 迁出计数 |

#### Zap structured log keys（7 类）

| Key | Level | 触发点 | 用例守护 |
|-----|-------|-------|---------|
| `topology.rule.evaluating` | info | ApplyRule 入口 | A1 |
| `topology.rule.matched` | info | 单设备 match 命中 | A2 |
| `topology.rule.priority_winner` | debug | 多规则 match 时 priority 比较 | A3 |
| `topology.rule.manual_skipped` | debug | source_type='manual' 行被跳过 | A4 |
| `topology.rule.apply_failed` | warn | rule task 失败 | A5 |
| `topology.cron.tick` | info | cron @hourly 触发 | — |
| `topology.event.bootstrap_received` | debug | EventBus device.inform.bootstrap 收到 | A2 |

### 12.6 跨模块联合变更

涉及 7 个变更点（全在单 PR 内，不拆分）：

- `omcgo/internal/topology/rule_service.go` — 接通 4 接线断点（getAllDevices / EventBus / cron / source_type）
- `omcgo/internal/topology/matcher.go` — 沿用现有 LAC/TAC/Name 3 mode（D7.A：不动）
- `omcgo/internal/topology/pg_repository.go` — INSERT/UPDATE device_group_members 增 source_type / source_rule_id 列
- `omcgo/internal/topology/model.go` — DeviceGroupMember struct 加 SourceType + SourceRuleID 字段
- `omcgo/cmd/app/provider/modules.go` — DI 装配 EventBus + cron 给 DeviceRuleService + 调 Start(ctx)
- `omcgo/migrations/000051_device_group_member_source.sql` — 见 §12.3
- `omcgo/internal/device/repository.go` — 新加 `ListAllForRuleEval(ctx)` 方法
- `omcmb/frontend-core/src/types/topology.ts` — DeviceGroupMember 加 sourceType 字段（snake_case → camelCase 自动转）
- `omcmb/webcode/src/pages/device/DeviceGrouping/` — 表格列加"来源（手工 / 规则名）"显示

**联合变更策略**：单 PR 包含 backend + FE，因 sourceType 端到端贯通；S6 commit 一并；E2E 加 5 条 claim（A1-A5 各 1）。

### 12.7 NATS 订阅设计

- 主题：`device.inform.bootstrap`（W3.E.2 已上线 5 类 subject 之一）
- Queue group：`topology-rule-engine`（多实例分布式负载，单一事件只消费一次）
- Idempotency：source_rule_id + device_id 复合 key 去重；重复消费同事件不致多次 INSERT

### 12.8 待定点（< 3 标准 ✅ 满足）

**W1**（最关键）：device 首次注册时是否在 device_group_members 表写入"default 组成员"行？
- 选项 W1.A：写入（每个 device 必有 group_member 行 → ApplyRule 自然能 evaluate）
- 选项 W1.B：不写入（device 隐式属 default → ApplyRule 需先 list 所有 device 再判属 default）
- **S3 第一动作**：grep `internal/device/service.go` Create 路径确认；按真实状态决定 D5.B 实施细节
- 影响：W1.A 简单（直接 UPDATE 现有行 source_type='rule'）；W1.B 需先 INSERT 再 UPDATE
- 风险等级：M（不阻塞 S3 启动，第一日内可消除）

**W2**：cron 重评对 source_type='manual' 行的处理路径（依赖 W1）
- 历史行 source_type='manual'（DEFAULT 值）→ A4 用例硬守不触动
- 但 D5.B 决策要求规则可触动 default 组中"未决策"行 — 取决于 W1 答案
- **S3 实施**：W1 确认后设计 reEvaluateAll 的 SQL WHERE 子句（`WHERE source_type='rule' OR source_type IS NULL` vs `WHERE source_type != 'manual'`）
- 风险等级：L（W1 解决后自然消除）

**已收敛 2 项 < 3 阈值 ✅**

**W3**（S3 Day 3 暴露 — pre-existing 缺口）：matcher.go MatchRequest 期望 `LAC *int` / `TAC *int`，但 devices 与 device_info 表均**无 LAC/TAC 列**（grep 全部 migrations/*.sql 确认；只有 device_groups.lac_list 和 device_rules.lac_list 是匹配条件方），现网设备级 LAC/TAC 数据无来源 ⇒ LAC / TAC 匹配 mode 在生产环境全部 false。
- 范围影响：D7.A 选定的三维（Name/LAC/TAC）中实际只有 Name 可工作，LAC/TAC 是哑路径
- **Day 3 决策**：T-0027 收紧到 Name 匹配可用 + LAC/TAC nil 安全降级（matcher 已支持 nil → false）；LAC/TAC 数据源（候选 device_parameters TR-069 path / sites 关联 / device_info 加列）**carve out 为 T-0027 followup**（候选 T-0098 待开），不阻塞 T-0027 主体过门
- 影响验收：A1/A2 GWT 用例若按 LAC 匹配验证需先 mock 数据；e2e 用例临时改成 Name 模式
- 风险等级：M（不阻塞 R-104 关闭，仅限缩 T-0027 实际 active 维度从 3 → 1）

### 12.9 S3 实施清单（按时序）

1. (1h) S3 第一动作：grep device.Create 确认 W1 答案 + 选定 reEvaluateAll SQL 路径
2. (2h) 写测试 — A1-A5 GWT 5 条单元测试 + 各 1 条 e2e claim
3. (3h) 实施 getAllDevices() — 接 DeviceRepository.ListAllForRuleEval
4. (3h) 实施 EventBus 订阅 + handleDeviceBootstrap
5. (2h) 实施 cron @hourly + reEvaluateAll
6. (2h) migration 000051 + repo 改 source_type/source_rule_id 列写入
7. (2h) modules.go DI 装配 + Start(ctx)
8. (2h) 6 metric + 7 log key 埋点
9. (2h) FE — types + DeviceGrouping 表格列改 + i18n
10. (1h) bash scripts/check-migrations.sh + 完整 build/test/race
11. (1h) E2E claim 5 条 + verify-T-0027.md 写
12. (0.5h) S5 self-review 14 项 + DoD 勾选

总计 ~21h ≈ **3 人日**（M 上限），与 PRD §8 估算一致。
