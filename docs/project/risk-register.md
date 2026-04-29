# Risk Register（风险登记册）

> **性质**：活文档。每 Sprint 回顾时更新；每季度深度复盘。  
> **守护人**：项目经理（`CLAUDE.md §16.11`）  
> **初始化依据**：`docs/implementation-completeness-report-20260420.md` 五大 P0 + 次级短板

---

## 字段说明

| 字段 | 含义 |
|------|------|
| ID | R-NNN，永不复用 |
| 等级 | P0（阻塞发布）/ P1（影响质量）/ P2（优化项） |
| 概率 | 高 / 中 / 低 |
| 影响 | 若发生的最坏结果 |
| Owner | 负责跟踪的角色/人 |
| 状态 | Open / Mitigating / Closed |
| 下次复盘 | 日期 |

---

## P0 风险（发布阻塞）

### R-001 告警通知链路缺失
- **描述**：F04 告警模块规则支持 action="notify"，但无任何下游通知渠道实现（邮件/短信/Webhook/移动推送全无）
- **等级**：P0
- **概率**：高（已发生 — 当前状态即是）
- **影响**：告警无法送达运维人员，运营场景下致命
- **Owner**：PM（写 PRD）+ 电信业务专家 + Go 工程专家
- **状态**：Open
- **关联 Task**：T-0007（邮件）/ T-0009（短信凭据）/ T-0011（Webhook）/ T-0014（短信）
- **缓解**：PRD `docs/project/prd/F04-alarm-notification.md` 已产出；进 Sprint 1-2 实现邮件+Webhook，Sprint 3 补短信
- **下次复盘**：Sprint-01 回顾（~2026-05-04）

### R-002 E2E 用例实现为 0
- **描述**：`scripts/e2e_verify.sh` 框架 4459 行但实际用例数为 0（grep `claim` = 0），声称的 452 用例完全未落地
- **等级**：P0
- **概率**：高（已发生）
- **影响**：无法回归；任何合入都可能引入未发现的破坏
- **Owner**：QA/发布经理
- **状态**：Open
- **关联 Task**：T-0006（累计型，Sprint-01..07，目标 ≥200）
- **缓解**：CI 中加入 E2E 用例数阈值（初始 20，每 Sprint +20 到 200+）；Sprint 1 先补核心流程（登录/设备/告警/PM/Inform）
- **下次复盘**：每 Sprint 回顾

### R-003 F08 北向 OSS 协议栈缺失
- **描述**：仅实现 REST JSON 导出，无 SNMP/CORBA/MTOSI/TMF 标准协议适配
- **等级**：P0（对接运营商 OSS 时）
- **概率**：中（取决于首个商用运营商的接入要求）
- **影响**：不满足运营商 OSS 接入规范，可能阻塞商用
- **Owner**：PM（需求矩阵）+ 架构师
- **状态**：Open
- **关联 Task**：T-0013（SNMP 骨架）/ T-0017（联调）/ T-0020（推送可靠性）
- **缓解**：PM 产出运营商需求矩阵 → 优先实现刚需协议（通常 SNMP Trap）→ 6-10 周独立 track
- **下次复盘**：Sprint-02 回顾

### R-004 事件总线仅 SSE，无分布式协调
- **描述**：`internal/events/` 仅实现 SSE Hub + 持久化，没有 NATS JetStream 主题订阅，单进程事件总线限制横向扩展
- **等级**：P0（扩展性）
- **概率**：中
- **影响**：10 万→100 万扩展时，模块间事件解耦无法实现
- **Owner**：架构师 + 运维
- **状态**：Open
- **关联 Task**：T-0010（NATS 主改造，XL）+ T-0012（worker 重试，依赖 T-0010）
- **缓解**：基础设施层引入 NATS JetStream 主题订阅；影响 F04 通知/F08 推送/transfer 三模块，需并排改造
- **下次复盘**：Sprint-02

### R-005 数据库迁移版本号跳跃
- **描述**：`migrations/` 缺 000010、000015-000018，goose 严格模式可能歧义
- **等级**：P0（可快速修复）
- **概率**：高（已存在）
- **影响**：新环境首次迁移可能失败；团队认知混乱
- **Owner**：数据与存储专家
- **状态**：**Closed（2026-04-20）**
- **关闭依据**：采用方案 (a) 补占位——创建 5 个 `000010/000015-000018_reserved_placeholder.sql`
  （Up/Down 均为 `SELECT 1;` noop + 注释说明占位用途）。`scripts/check-migrations.sh`
  运行确认编号 000001→000023 连续无跳跃。同时修正脚本 Down 空判定阈值 bug
  （`<=1` → `<1`）——原阈值会把只有一条 SQL 的 Down 段误判为空。后续接入 CI 由 R-107 跟进。
- **复盘**：初版脚本误报 000001 / 000021 "Down 空"，诊断后确认两者 Down 段完整
  （分别 DROP FUNCTION / DROP TABLE），误报源于脚本阈值 bug，已修正（见 R-108 Closed）

---

## 整改期风险（Wave 1-3 专属）

> 来源：`docs/project/整改路线图-2026Q2.md` 第 5 章。
> 这些风险在整改启动后立刻生效；不达 Wave 退出条件不进下一波。

### R-INT-01 整改期间冻结令破功
- **描述**：客户/老板压力或团队惯性导致"冻结新功能"在 Wave 1-2 期间被打破，in-progress 涨到 5+
- **等级**：P0
- **概率**：高（首次执行冻结）
- **影响**：整改方案破产，重回 74% 完成度僵尸状态
- **Owner**：项目经理
- **状态**：Open
- **关联**：Wave 1.1 ~ W8 全部任务；`docs/project/backlog.md` FREEZE 块
- **缓解**：
  - 周一规划会硬性"finishing only"
  - 周报公布完成数（W1 末按 `docs/methodology/AI承诺对峙清单.md` 8 项硬承诺核验）
  - 任何插队需求转入 Wave 4 后队列
- **下次复盘**：每周五 finishing day（首次 2026-05-01）

### R-INT-02 CI 红灯被绕过
- **描述**：CI 建好后频繁失败，团队加 `// nolint:all` 或 `--no-verify` 绕过
- **等级**：P0
- **概率**：中
- **影响**：DoD 形同虚设，所有铁律失效
- **Owner**：后端 lead
- **状态**：Open
- **关联**：W1.1（CI 工作流）、W1.2（PR 模板）
- **缓解**：
  - CI 红灯 24h 内必须修
  - 连续 3 次跳过 = 升级到团队复盘
  - PR review 拒绝 `// nolint:all`、`--no-verify`
- **下次复盘**：每两周

### R-INT-03 E2E 用例补齐速度跟不上
- **描述**：W1 末 ≥ 20 / W8 末 ≥ 100 / W14 末 ≥ 200 的速度无法维持
- **等级**：P0
- **概率**：中
- **影响**：W1/W8 退出条件不达，整改路线图无法推进
- **Owner**：QA/发布经理
- **状态**：Open
- **关联**：W1.6、W8.3、长承诺 7
- **缓解**：
  - 最低速度 10 用例/周；不达则削减新切片
  - 每个新功能 PR 必须新增至少 1 条 E2E 断言
  - CI 中加用例数阈值检查
- **下次复盘**：每周

### R-INT-04 NATS JetStream 改造引入回归
- **描述**：Wave 3 Block E（W9-W10）NATS 切换破坏 F04/F08/transfer 联动
- **等级**：P0
- **概率**：中（跨域基础设施变更高风险）
- **影响**：Wave 3 阻塞 4-6 周，可能退回 ChannelBus
- **Owner**：架构专家
- **状态**：Open
- **关联**：Wave 3 Block E、R-004（事件总线）
- **缓解**：
  - 强制 `/ultrareview` 多 agent 审查
  - 配置驱动可切换实现，灰度发布
  - 故障演练（NATS 挂掉自动重连不丢消息）
- **下次复盘**：Block E 末

### R-INT-05 整改完成后惯性回潮
- **描述**：14 周整改完成后流程纪律松懈，三铁五铁形同虚设
- **等级**：P1
- **概率**：高（无持续问责的制度都会衰减）
- **影响**：6 个月内重回僵尸状态
- **Owner**：项目经理
- **状态**：Open
- **关联**：永久制度（流程刚性升级第 3 章）
- **缓解**：
  - DoD 三个不可绕过开关写进 `.github/CODEOWNERS` 与 CI 配置
  - 季度审计：`docs/methodology/AI承诺对峙清单.md` 通过率单调
  - 工程文化文档新人必读
- **下次复盘**：GA 后每季度

---

## P1 风险（质量影响）

### R-101 Software 模块缺灰度升级策略 + 回退审计/触发
- **描述**：`internal/software/` 批量升级仅支持全量并发控制，无分组/延时/百分比策略；回退仅"机械触发"，无审计/原因/灰度联动
- **等级**：P1
- **概率**：中
- **影响**：大规模升级无法分批验证；回退缺审计影响事故追溯
- **Owner**：电信业务专家
- **状态**：✅ **完整闭环 Closed** (2026-04-29) — T-0018 灰度 + T-0021 回退增强双链落地
- **关联 Task**：T-0018（灰度升级策略）✅ done / T-0021（回退增强）✅ done
- **缓解（灰度，T-0018）**：`internal/software/canary.go` 类型 + DefaultCanaryStages [1,10,50,100] + ValidateStages / DevicesForStage / FailureRate；`canary_monitor.go` cron 每 1 min 失败率阈值检查 → 自动暂停 + 4 Prometheus 指标；4 Admin endpoints (advance/pause/resume/abort)；migration 000045 +7 字段；BatchUpgradeRequest 加 strategy 向后兼容
- **缓解（回退，T-0021）**：`RollbackRequest` +3 audit 字段（reason/source/target_firmware_id）；4 source 枚举 + CHECK 约束 + IsValidRollbackSource；TargetFirmwareID 非空时 sub_tasks.DestVersion 取该固件版本；`CanaryStrategy +RollbackOnFailure`（默认 false 维持"不擅自回滚"D4 invariant）；canary monitor 阈值超 + opt-in → pause 后追加 auto-rollback；3 新 Counter（`software_rollback_total{source}` / `software_rollback_devices_total{source}` / `software_rollback_with_target_total`）；migration 000046 +4 列 + 1 部分索引；e2e_verify.sh +1 claim
- **关闭依据**：`docs/project/prd/T-0018-software-canary-upgrade.md` + `docs/review-report/20260429/verify-T-0018.md` + `docs/project/prd/T-0021-software-rollback-enhanced.md` + `docs/review-report/20260429/verify-T-0021.md` + commit `5baaf633`
- **下次复盘**：N/A（已闭环）

### R-102 前端 Backup/Software 页面仅骨架
- **描述**：Backup 48%、Software 52% 完成度，业务逻辑缺失（任务创建/升级进度/回滚）
- **等级**：P1
- **概率**：高
- **影响**：运维场景不可用
- **Owner**：前端专家
- **状态**：⚠️ Partially Closed (2026-04-29) — Software/Topology/Report 全闭环；Backup 仅 Tasks/FTP 子集闭环；Schedule/Policy/Restore 留 3 followup
- **关联 Task**：T-0019 ✅（Software Canary 消费）/ T-0022 ✅（Topology/Report）/ T-0016 ✅（Backup Tasks + FTP）；followups T-0070 (Schedule UI 重设计) / T-0071 (Policy 后端+前端) / T-0072 (Restore 流程设计)
- **关闭依据（已闭环部分）**：T-0019 commit `4ebdc91c` + T-0022 commit `c5e134c0` + T-0016 commit `c939ff12`
- **未闭环部分**：审计发现 Backup 子模块原 48% 完成度估值偏乐观，实际 ≈20%（BackupTasks `void data` 不消费 hook、BackupSchedule 名实不符、BackupPolicy 后端无 endpoint、RestoreData 后端无 restore）；本任务务实闭环 Tasks+FTP，剩余三个子模块各需独立 design 工作
- **下次复盘**：T-0070/0071/0072 完成时（预计 sprint-06+）

### R-103 License 容量/过期未拦截
- **描述**：`MaxDevices`/`ExpiryDate` 字段有，但无超限拦截与自动禁用
- **等级**：P1
- **概率**：中
- **影响**：违反商业约束，不符合授权要求
- **Owner**：电信业务专家
- **状态**：✅ Closed (2026-04-28)
- **关联 Task**：T-0015（License 容量/过期拦截）✅ done
- **缓解**：T-0015 完成 — Enforcer + Monitor cron + 6 Prometheus metrics + device.CreateDevice 拦截闸 + GET /licenses/quota 端点；6 项决策（过期软告警限写 D1 / 三档容量阈值 D2 / max active license D3 / perpetual 跳过 D4 / grace_period_days 可配 D5 / 第一版仅 device D6）全部实施
- **关闭依据**：`docs/project/prd/F06-license-enforcement.md` + `docs/review-report/20260428/verify-T-0015.md` + commit (post-S6)
- **下次复盘**：N/A（已关闭）

### R-104 拓扑自动分组规则引擎未激活
- **描述**：`rule_service` 注释为 TODO，rule_matcher 存在但未接入
- **等级**：P1
- **概率**：低
- **影响**：分组依赖手工维护，规模上不去
- **Owner**：架构师
- **状态**：Open
- **关联 Task**：T-0027（triaged，等 Sprint Planning 拉起）
- **缓解**：Sprint 5 激活
- **下次复盘**：Sprint-04

### R-105 syslog 远程转发缺失
- **描述**：仅查询，无 UDP/TCP syslog 转发
- **等级**：P1
- **概率**：低
- **影响**：与统一日志平台集成困难
- **Owner**：运维与可观测性专家
- **状态**：Open
- **关联 Task**：T-0028（triaged，按需拉起）
- **缓解**：按需实现
- **下次复盘**：Sprint-05

### R-106 worker 进程无重试/死信队列
- **描述**：EventBus 订阅失败无重试，长运行稳定性未验证
- **等级**：P1
- **概率**：中
- **影响**：PM/MR 文件处理失败会直接丢弃
- **Owner**：架构师 + 运维
- **状态**：✅ Closed (2026-04-28)
- **关联 Task**：T-0012（worker 重试/死信） ✅ done
- **缓解**：T-0012 完成 — `internal/core/reliability/dlq` 通用 DLQ 子包 + `runner` retry 装饰器（内层 3 次 + NATSEventBus 外层 5 次双重保险）+ `/api/v1/admin/dead-letters` 4 端点（List/Get/Delete/Replay，admin RBAC）+ migration 000044 dead_letters 表 + 4 Prometheus metric + PM Collector 接入示范；其他 11 subscriber 后续 PR 扩展
- **关闭依据**：`docs/project/prd/T-0012-worker-retry-dlq.md` + `docs/review-report/20260428/verify-T-0012.md` + commit (post-S6)
- **下次复盘**：N/A（已关闭）

### R-107 Prometheus/Grafana/AlertManager 容器编排缺失
- **描述**：metrics/tracing 已集成但 docker-compose 无监控栈
- **等级**：P1
- **概率**：高
- **影响**：Release Gate 中"可观测性就绪"无法自动化验证
- **Owner**：运维与可观测性专家
- **状态**：Open
- **关联 Task**：T-0008（Prometheus/Grafana/AlertManager 容器编排 + 基础 dashboard）
- **缓解**：Sprint 1 补 compose 文件 + 基础 dashboard
- **下次复盘**：Sprint-01

### R-108 个别迁移 Down 段缺失（误报，已撤销）
- **描述**（初版）：`000001_extensions_functions.sql` 与 `000021_notifications.sql`
  的 Down 段被脚本标记为空
- **等级**：P1 → **撤销**
- **状态**：**Closed（2026-04-20，误报）**
- **复盘**：验证原文件后确认两者 Down 段实际完整：
  · 000001：`DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;`
    （extensions `uuid-ossp`/`timescaledb` 为项目级共享基础设施，按 `omcgo/CLAUDE.md
    §5.5.8 "仅限迁移专属扩展"` 规则不应在 Down 中 DROP，刻意保留）
  · 000021：`DROP TABLE IF EXISTS notifications;`
- **根因**：`check-migrations.sh` 阈值 bug——`grep -v '^\s*--'` 已剥掉 `-- +goose Down`
  标记行，但判定仍用 `<=1`，导致只含一条 SQL 的 Down 段被误判为空。已修正为 `<1`
- **教训**：新工具投产前需用 golden sample（含代表性正确/错误样本）校准阈值

---

## P2 风险（优化项）

### R-201 RF 控制硬编码 LTE
- **描述**：device 模块 RF 开关未走 Carrier 适配
- **等级**：P2
- **Owner**：电信业务专家
- **状态**：Open

### R-202 F10 互操作用例库稀疏
- **描述**：仅 RPC/Protocol/DataModel 三类
- **等级**：P2
- **Owner**：测试专家

### R-203 CAPTCHA 图形生成待补
- **描述**：admin 模块仅端点，无图形生成
- **等级**：P2
- **Owner**：安全合规专家

### R-204 FTP 连接测试占位符
- **描述**：backup 模块 `POST /ftp-configs/test` 返回 "not implemented"
- **等级**：P2
- **Owner**：Go 工程专家

### R-205 多级聚合与分析能力缺失
- **描述**：PM 仅设备级，MR 无地理聚合，无趋势/根因分析
- **等级**：P2（功能扩展）
- **Owner**：电信业务专家 + 数据与存储专家

---

## 关闭的风险

| ID | 关闭日期 | 关闭摘要 |
|----|---------|---------|
| R-005 | 2026-04-20 | 数据库迁移版本号跳跃。补 5 个 `reserved_placeholder` 占位迁移，`check-migrations.sh` 检查通过。衍生风险 R-108（个别 Down 段缺失）已登记 |

---

## 复盘节奏

- **每 Sprint 回顾**：更新 Open/Mitigating 状态，检查 Owner 有无变更
- **每月第一个 Sprint**：审视 P0 列表，确保 <14 天已关闭或有明确进展
- **每季度**：深度复盘 P1/P2，决定是否升级或关闭

**当前版本**：v1.1（2026-04-20，全部 Open 风险补"关联 Task"字段，双向链通 Backlog）
